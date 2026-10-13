#!/usr/bin/env bash
# Shared path, ownership, and immutable evidence rules for the local Dagu pilot.
set -euo pipefail
# Normalize every script and its children, including Git push hooks.
umask 022
fail() { printf 'rootline-dagu: %s\n' "$*" >&2; exit 1; }
valid_id() { [[ "$1" =~ ^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$ ]] || fail "invalid identifier: $1"; }
atomic_write() {
    local target=$1 value=$2 temporary
    temporary=$(mktemp "${target}.tmp.XXXXXX")
    printf '%s\n' "$value" > "$temporary"
    mv -f "$temporary" "$target"
}
# Hooks and nested Git callers must not redirect operations into another repository.
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_COMMON_DIR GIT_PREFIX GIT_NAMESPACE GIT_CEILING_DIRECTORIES GIT_DISCOVERY_ACROSS_FILESYSTEM
init() {
    [[ $# == 2 ]] || fail 'expected SPACE_ID RUN_ID'
    SPACE=$1 RUN=$2
    valid_id "$SPACE"; valid_id "$RUN"
    REPO=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd -P)
    HOME_DIR="$REPO/.workspace/dagu"
    export DAGU_HOME="$HOME_DIR"
    COMMON=$(git -C "$REPO" rev-parse --git-common-dir)
    [[ "$COMMON" == /* ]] || COMMON="$REPO/$COMMON"
    COMMON=$(cd "$COMMON" && pwd -P)
    LOCK="$COMMON/rootline-dagu.lock"
    OWNER=$(printf '%s\n%s' "$SPACE" "$RUN")
    ROOT=${ROOTLINE_WORKTREE_ROOT:-${HOME:?HOME must be set}/.local/share/rootline/worktrees}
    ROOT=${ROOT%/}
    [[ "$ROOT" == /* && "$ROOT" != / ]] || fail 'worktree root must be absolute and non-root'
    # Resolve every existing ancestor, refusing symlink aliases and traversal.
    local probe=$ROOT suffix='' resolved
    while [[ ! -e "$probe" ]]; do
        suffix="/$(basename "$probe")$suffix"
        probe=$(dirname "$probe")
    done
    resolved=$(cd "$probe" && pwd -P)
    [[ "$resolved" != / ]] || resolved=
    [[ "$resolved$suffix" == "$ROOT" ]] || fail 'worktree root must be canonical (no symlinks or traversal)'
    WORKTREE="$ROOT/$SPACE/$RUN"
    BRANCH="dagu/$SPACE/$RUN"
    STATE="$HOME_DIR/data/rootline-runs/$SPACE/$RUN"
    for probe in "$REPO/.workspace" "$HOME_DIR" "$HOME_DIR/data" "$HOME_DIR/data/rootline-runs" "$HOME_DIR/data/rootline-runs/$SPACE" "$STATE" "$ROOT/$SPACE" "$WORKTREE" "$LOCK"; do
        [[ ! -L "$probe" ]] || fail "refusing symlink: $probe"
    done
    RECORD=$(printf '%s\n%s\n%s\n%s' "$OWNER" "$REPO" "$WORKTREE" "$BRANCH")
}
owned_lock() {
    [[ -f "$LOCK" && ! -L "$LOCK" && "$(cat "$LOCK")" == "$OWNER" ]]
}
acquire_lock() {
    [[ ! -e "$LOCK" || -f "$LOCK" ]] || fail 'lock path is not a regular owner file'
    local temporary
    temporary=$(mktemp "$COMMON/.rootline-dagu-owner.XXXXXX")
    printf '%s\n' "$OWNER" > "$temporary"
    # Hard-link publication is atomic and exclusive on macOS and Linux. There
    # is no crash window containing a lock whose owner has not been written.
    if ln "$temporary" "$LOCK" 2>/dev/null; then
        rm "$temporary"
    else
        rm "$temporary"
        owned_lock || fail 'repository lock belongs to another run; recovery must name its owner'
    fi
}
load_state() {
    [[ -f "$STATE/owner" && ! -L "$STATE/owner" && "$(cat "$STATE/owner")" == "$RECORD" ]] || fail 'missing or mismatched run ownership record'
    local entry
    for entry in "$STATE/phases" "$STATE/base.sha" "$STATE/tested.sha" "$STATE/evidence" "$STATE/rootline" "$STATE/rootline.tmp" "$STATE/pr" "$STATE/issue" "$STATE/merged" "$STATE/cleaned" "$STATE/ci.owner" "$STATE/ci.sha" "$STATE/ci.checks" "$STATE/phases/"*; do
        [[ ! -L "$entry" ]] || fail "refusing symlinked run evidence: $entry"
    done
}
read_sha() {
    [[ -f "$STATE/tested.sha" && ! -L "$STATE/tested.sha" ]] || fail 'missing tested SHA'
    SHA=$(cat "$STATE/tested.sha")
    [[ "$SHA" =~ ^[0-9a-f]{40}$ ]] || fail 'invalid tested SHA'
}
clean_at_sha() {
    [[ "$(git -C "$WORKTREE" symbolic-ref --quiet --short HEAD)" == "$BRANCH" ]] || fail 'worktree branch changed'
    [[ "$(git -C "$WORKTREE" rev-parse HEAD)" == "$SHA" ]] || fail 'worktree SHA changed after testing began'
    [[ -z "$(git -C "$WORKTREE" status --porcelain --untracked-files=all)" ]] || fail 'worktree is not clean'
}
ensure_worktree() {
    load_state
    acquire_lock
    if [[ ! -d "$WORKTREE" ]]; then
        # A terminal failure handler may have cleaned resources. A targeted retry
        # recreates precisely the recorded commit, never the latest master.
        read_sha
        mkdir -p "$ROOT/$SPACE"
        [[ ! -e "$WORKTREE" ]] || fail 'worktree destination is occupied'
        git -C "$REPO" show-ref --verify --quiet "refs/heads/$BRANCH" && fail 'unowned branch blocks retry'
        git -C "$REPO" worktree add -b "$BRANCH" "$WORKTREE" "$SHA"
    fi
    [[ "$(git -C "$WORKTREE" rev-parse --show-toplevel)" == "$WORKTREE" ]] || fail 'worktree path mismatch'
    local work_common
    work_common=$(git -C "$WORKTREE" rev-parse --git-common-dir)
    [[ "$work_common" == /* ]] || work_common="$WORKTREE/$work_common"
    [[ "$(cd "$work_common" && pwd -P)" == "$COMMON" ]] || fail 'worktree belongs to another repository'
}
PHASES=(tidy-clean check test coverage-check test-install validate govulncheck gitleaks)
completed_evidence() {
    load_state; read_sha
    [[ -f "$STATE/evidence" && "$(cat "$STATE/evidence")" == "$SHA" ]] || fail 'gate evidence is incomplete or stale'
    local phase
    for phase in "${PHASES[@]}"; do
        [[ -f "$STATE/phases/$phase" && "$(cat "$STATE/phases/$phase")" == "$SHA" ]] || fail "missing exact-SHA phase: $phase"
    done
    # A surviving local worktree must still be clean at the tested commit.
    if [[ -d "$WORKTREE" ]]; then
        ensure_worktree
        clean_at_sha
    elif git -C "$REPO" show-ref --verify --quiet "refs/heads/$BRANCH"; then
        [[ "$(git -C "$REPO" rev-parse "$BRANCH")" == "$SHA" ]] || fail 'local branch moved beyond tested SHA'
    fi
}
