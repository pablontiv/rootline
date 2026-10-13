#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/lib.sh"
init "$@"
if [[ ! -e "$LOCK" ]]; then
    if [[ -e "$WORKTREE" ]] || git -C "$REPO" show-ref --verify --quiet "refs/heads/$BRANCH"; then
        fail 'resources remain without an owned lock; refusing unowned cleanup'
    fi
    exit 0
fi
owned_lock || fail 'cleanup cannot release another run lock'
if [[ -e "$STATE/owner" ]]; then
    load_state
    registration=$(git -C "$REPO" worktree list --porcelain | grep -F -x "worktree $WORKTREE" || true)
    if [[ -e "$WORKTREE" || -n "$registration" ]]; then
        if [[ -e "$WORKTREE" ]]; then
            [[ -d "$WORKTREE" && "$(git -C "$WORKTREE" rev-parse --show-toplevel)" == "$WORKTREE" ]] || fail 'not the owned worktree'
            [[ "$(git -C "$WORKTREE" symbolic-ref --quiet --short HEAD)" == "$BRANCH" ]] || fail 'refusing cleanup of changed worktree branch'
            common=$(git -C "$WORKTREE" rev-parse --git-common-dir)
            [[ "$common" == /* ]] || common="$WORKTREE/$common"
            [[ "$(cd "$common" && pwd -P)" == "$COMMON" ]] || fail 'refusing cleanup of another repository'
        fi
        # `worktree remove --force` also clears this exact path's stale registration.
        git -C "$REPO" worktree remove --force "$WORKTREE"
    fi
    if git -C "$REPO" show-ref --verify --quiet "refs/heads/$BRANCH"; then
        # Git refuses branch deletion if another worktree has checked it out.
        git -C "$REPO" branch -D "$BRANCH"
    fi
    atomic_write "$STATE/cleaned" "$OWNER"
fi
# Remove only the known owner file, never recursively delete a lock directory.
owned_lock || fail 'lock ownership changed during cleanup'
rm "$LOCK"
printf 'cleaned %s/%s\n' "$SPACE" "$RUN"
