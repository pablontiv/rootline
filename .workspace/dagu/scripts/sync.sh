#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/lib.sh"
init "$@"
acquire_lock
mkdir -p "$STATE"
if [[ -e "$STATE/owner" ]]; then
    load_state
else
    [[ ! -e "$WORKTREE" ]] || fail 'worktree destination already exists'
    git -C "$REPO" show-ref --verify --quiet "refs/heads/$BRANCH" && fail 'run branch already exists'
    atomic_write "$STATE/owner" "$RECORD"
fi
[[ "$(git -C "$REPO" symbolic-ref --quiet --short HEAD)" == master ]] || fail 'primary checkout must be on master'
[[ -z "$(git -C "$REPO" status --porcelain --untracked-files=all)" ]] || fail 'primary checkout must be clean'
# Always fetch before the fast-forward, even when repeating a sync checkpoint.
git -C "$REPO" fetch --prune origin
git -C "$REPO" merge --ff-only origin/master
if [[ -d "$WORKTREE" ]]; then
    [[ "$(git -C "$WORKTREE" symbolic-ref --quiet --short HEAD)" == "$BRANCH" ]] || fail 'run branch mismatch'
    [[ -z "$(git -C "$WORKTREE" status --porcelain --untracked-files=all)" ]] || fail 'existing worktree is dirty'
else
    [[ ! -e "$STATE/tested.sha" ]] || fail 'run already tested; retry the failed phase, not sync'
    git -C "$REPO" show-ref --verify --quiet "refs/heads/$BRANCH" && fail 'run branch exists without owned worktree'
    mkdir -p "$ROOT/$SPACE"
    git -C "$REPO" worktree add -b "$BRANCH" "$WORKTREE" master
fi
atomic_write "$STATE/base.sha" "$(git -C "$WORKTREE" rev-parse HEAD)"
printf 'worktree=%s\nbranch=%s\n' "$WORKTREE" "$BRANCH"
