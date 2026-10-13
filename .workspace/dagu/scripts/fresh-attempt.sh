#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/lib.sh"
[[ $# == 5 ]] || fail 'expected RUN_ID EXECUTION_ID MODE SPACE_ID ISSUE_NUMBER'
LOGICAL_RUN=$1 EXECUTION=$2 MODE=$3 ISSUE=$5
valid_id "$EXECUTION"
[[ "$EXECUTION" != "$LOGICAL_RUN" ]] || fail 'execution ID must differ from logical RUN_ID'
case "$MODE" in shadow|pr|merge) ;; *) fail 'unknown fresh-attempt mode' ;; esac
[[ "$ISSUE" =~ ^[1-9][0-9]*$ ]] || fail 'fresh attempt requires a positive issue number'
init "$4" "$LOGICAL_RUN"
ARCHIVES="$HOME_DIR/data/rootline-attempts/$SPACE/$RUN"
ARCHIVE="$ARCHIVES/$EXECUTION"
for entry in "$HOME_DIR/data/rootline-attempts" "$HOME_DIR/data/rootline-attempts/$SPACE" "$ARCHIVES" "$ARCHIVE"; do
    [[ ! -L "$entry" ]] || fail "refusing symlinked attempt archive: $entry"
done
[[ ! -e "$ARCHIVE" ]] || fail 'execution ID already archived; choose a unique execution ID'
prior_attempt() {
    load_state
    [[ -f "$STATE/cleaned" && "$(cat "$STATE/cleaned")" == "$OWNER" ]] || fail 'prior attempt needs terminal cleanup first'
    [[ ! -e "$WORKTREE" ]] || fail 'prior worktree remains after cleanup'
    git -C "$REPO" show-ref --verify --quiet "refs/heads/$BRANCH" && fail 'prior branch remains after cleanup'
    [[ -z "$(git -C "$REPO" worktree list --porcelain | grep -F -x "worktree $WORKTREE" || true)" ]] || fail 'prior worktree registration remains'
    [[ -f "$STATE/issue" && "$(cat "$STATE/issue")" == "$ISSUE" ]] || fail 'prior forge issue differs from requested issue'
    [[ -f "$STATE/pr" ]] || fail 'prior attempt is not PR-bound'
    PR=$(cat "$STATE/pr")
    [[ "$PR" =~ ^[1-9][0-9]*$ ]] || fail 'invalid prior PR number'
    [[ ! -e "$STATE/merged" ]] || fail 'prior attempt already merged'
    if [[ -e "$STATE/tested.sha" ]]; then
        read_sha
    else
        # A previous launcher may have failed before sync or the first gate.
        [[ -f "$STATE/attempt" && -f "$STATE/previous" ]] || fail 'prior attempt has no tested SHA or fresh-launch linkage'
    fi
    if [[ -f "$STATE/attempt" ]]; then
        [[ "$(cat "$STATE/attempt")" != "$EXECUTION" ]] || fail 'execution ID already used by the live attempt'
    fi
}
prior_attempt
# Unlike checkpoint retries, a fresh launch must acquire an unoccupied lock,
# even for the same logical owner. Retain it through archival and DAG cleanup.
[[ ! -e "$LOCK" ]] || fail 'repository is locked; recover the named owner first'
lock_file=$(mktemp "$COMMON/.rootline-dagu-owner.XXXXXX")
printf '%s\n' "$OWNER" > "$lock_file"
if ! ln "$lock_file" "$LOCK" 2>/dev/null; then
    rm "$lock_file"
    fail 'repository lock was acquired by another launch'
fi
rm "$lock_file"
prior_attempt
# Reserve a unique container exclusively; its state destination cannot exist.
# The rename is on the same filesystem, never a copy followed by deletion.
mkdir -p "$ARCHIVES"
mkdir "$ARCHIVE" || fail 'execution archive already exists'
mkdir "$ARCHIVE/live"
atomic_write "$ARCHIVE/live/owner" "$RECORD"
atomic_write "$ARCHIVE/live/issue" "$ISSUE"
atomic_write "$ARCHIVE/live/pr" "$PR"
atomic_write "$ARCHIVE/live/attempt" "$EXECUTION"
atomic_write "$ARCHIVE/live/previous" "rootline-attempts/$SPACE/$RUN/$EXECUTION/state"
mv "$STATE" "$ARCHIVE/state"
mv "$ARCHIVE/live" "$STATE"
printf 'archived=%s\nlogical_owner=%s/%s execution=%s\n' "$ARCHIVE/state" "$SPACE" "$RUN" "$EXECUTION"
# A launcher failure before the exit handler must not release the owner lock or
# erase evidence. Owner-bound cleanup is the explicit recovery path.
if "$HOME_DIR/bin/dagu" start --run-id "$EXECUTION" "$HOME_DIR/dags/rootline-feature.yaml" -- \
    "SPACE_ID=$SPACE" "RUN_ID=$RUN" "MODE=$MODE" "ISSUE_NUMBER=$ISSUE"; then
    exit 0
else
    result=$?
    printf 'rootline-dagu: launch failed (%s); archive retained at %s; inspect execution %s and recover owner %s/%s before another fresh attempt\n' \
        "$result" "$ARCHIVE/state" "$EXECUTION" "$SPACE" "$RUN" >&2
    exit "$result"
fi
