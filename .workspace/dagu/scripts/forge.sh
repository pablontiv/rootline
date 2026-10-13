#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/lib.sh"
[[ $# == 3 ]] || fail 'expected SPACE_ID RUN_ID pr|status|ci|merge'
ACTION=$3
case "$ACTION" in pr|status|ci|merge) ;; *) fail 'unknown forge action' ;; esac
init "$1" "$2"
completed_evidence
acquire_lock
ISSUE=${ISSUE_NUMBER:-}
[[ "$ISSUE" =~ ^[1-9][0-9]*$ ]] || fail 'forge mode requires a positive ISSUE_NUMBER'
if [[ -f "$STATE/issue" ]]; then
    [[ "$(cat "$STATE/issue")" == "$ISSUE" ]] || fail 'run issue number changed'
else
    atomic_write "$STATE/issue" "$ISSUE"
fi
cd "$REPO"
FORGE_REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
[[ "$FORGE_REPO" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || fail 'invalid forge repository'
FORGE_OWNER=${FORGE_REPO%%/*}
lookup_pr() {
    local row
    row=$(gh pr list --repo "$FORGE_REPO" --state all --head "$BRANCH" --base master \
        --json number,headRefOid,state,url --jq '.[] | [.number, .headRefOid, .state, .url] | @tsv')
    [[ "$row" != *$'\n'* ]] || fail 'multiple PRs match the run branch'
    PR='' HEAD_SHA='' PR_STATE='' PR_URL=''
    if [[ -n "$row" ]]; then
        IFS=$'\t' read -r PR HEAD_SHA PR_STATE PR_URL <<< "$row"
        [[ "$PR" =~ ^[1-9][0-9]*$ && "$HEAD_SHA" =~ ^[0-9a-f]{40}$ ]] || fail 'invalid PR response'
    fi
}
ci_evidence_valid() {
    [[ -f "$STATE/ci.owner" && -f "$STATE/ci.sha" && -s "$STATE/ci.checks" ]] &&
        [[ "$(cat "$STATE/ci.owner")" == "$RECORD" ]] &&
        [[ "$(cat "$STATE/ci.sha")" == "$SHA" ]]
}
local_status() {
    gh api "repos/$FORGE_REPO/commits/$SHA/status" \
        --jq '[.statuses[] | select(.context == "dagu/rootline-local")][0].state // ""'
}
verify_local_status() {
    [[ "$(local_status)" == success ]] ||
        fail 'merge blocked: local gate status is missing or not successful'
}
verify_hosted_ci() {
    [[ -n "$PR" && "$HEAD_SHA" == "$SHA" && ( "$PR_STATE" == OPEN || "$PR_STATE" == MERGED ) ]] ||
        fail 'hosted CI check needs the PR at the tested SHA'
    gh pr checks "$PR" --repo "$FORGE_REPO" --watch --fail-fast
    local rows name bucket workflow
    rows=$(gh pr checks "$PR" --repo "$FORGE_REPO" --json name,bucket,workflow \
        --jq '.[] | select(.workflow != "") | [.name, .bucket, .workflow] | @tsv')
    [[ -n "$rows" ]] || fail 'no hosted CI checks were reported for the PR'
    while IFS=$'\t' read -r name bucket workflow; do
        [[ -n "$name" && -n "$workflow" && ( "$bucket" == pass || "$bucket" == skipping ) ]] ||
            fail "hosted CI is not successful: ${name:-unknown}"
    done <<< "$rows"
    lookup_pr
    [[ -n "$PR" && "$HEAD_SHA" == "$SHA" && ( "$PR_STATE" == OPEN || "$PR_STATE" == MERGED ) ]] ||
        fail 'PR head changed while waiting for hosted CI'
    atomic_write "$STATE/ci.owner" "$RECORD"
    atomic_write "$STATE/ci.sha" "$SHA"
    atomic_write "$STATE/ci.checks" "$rows"
    ci_evidence_valid || fail 'hosted CI evidence was not persisted'
}
lookup_pr
case "$ACTION" in
    pr)
        if [[ -n "$PR" ]]; then
            [[ "$PR_STATE" != CLOSED ]] || fail 'run PR was closed without merge'
            if [[ "$HEAD_SHA" != "$SHA" ]]; then
                git merge-base --is-ancestor "$HEAD_SHA" "$SHA" ||
                    fail 'existing PR head diverges from tested SHA'
                completed_evidence
                git push origin "$SHA:refs/heads/$BRANCH"
                lookup_pr
                [[ -n "$PR" && "$HEAD_SHA" == "$SHA" ]] ||
                    fail 'updated PR does not match tested SHA'
            fi
        else
            # Push the immutable SHA, not whichever commit a branch now names.
            completed_evidence
            git push origin "$SHA:refs/heads/$BRANCH"
            # A lost API response is safe: the next attempt looks up before create.
            TITLE=$(git show -s --format=%s "$SHA")
            [[ -n "$TITLE" && "$TITLE" != *$'\n'* ]] || fail 'tested commit has no usable PR title'
            gh pr create --repo "$FORGE_REPO" --base master --head "$FORGE_OWNER:$BRANCH" \
                --title "$TITLE" \
                --body "Local gate passed at commit $SHA. Run: $SPACE/$RUN. Hosted CI remains required. Closes #$ISSUE."
            lookup_pr
            [[ -n "$PR" && "$HEAD_SHA" == "$SHA" ]] || fail 'created PR does not match tested SHA'
        fi
        atomic_write "$STATE/pr" "$PR"
        printf '%s tested=%s\n' "$PR_URL" "$SHA"
        ;;
    status)
        [[ -n "$PR" && "$HEAD_SHA" == "$SHA" ]] || fail 'PR missing or stale'
        existing=$(local_status)
        if [[ "$existing" != success ]]; then
            gh api --method POST "repos/$FORGE_REPO/statuses/$SHA" \
                -f state=success -f context=dagu/rootline-local \
                -f "description=Complete local gate: $SPACE/$RUN" -f "target_url=$PR_URL" >/dev/null
        fi
        printf '%s tested=%s context=dagu/rootline-local\n' "$PR_URL" "$SHA"
        ;;
    ci)
        verify_hosted_ci
        printf '%s tested=%s hosted-ci=success\n' "$PR_URL" "$SHA"
        ;;
    merge)
        [[ -n "$PR" ]] || fail 'run PR is missing'
        # Re-query the PR before evaluating the local merge preconditions.
        row=$(gh pr view "$PR" --repo "$FORGE_REPO" --json headRefOid,state \
            --jq '[.headRefOid, .state] | @tsv')
        IFS=$'\t' read -r HEAD_SHA PR_STATE <<< "$row"
        completed_evidence
        [[ "$HEAD_SHA" == "$SHA" ]] || fail 'merge blocked: PR head differs from tested SHA'
        [[ -f "$STATE/base.sha" ]] || fail 'missing primary base SHA'
        BASE_SHA=$(cat "$STATE/base.sha")
        [[ "$BASE_SHA" =~ ^[0-9a-f]{40}$ ]] || fail 'invalid primary base SHA'
        if [[ "$PR_STATE" == MERGED && -f "$STATE/merged" && "$(cat "$STATE/merged")" == "$SHA" ]]; then
            printf 'merged PR %s tested=%s\n' "$PR" "$SHA"
            exit 0
        fi
        [[ "$(git symbolic-ref --quiet --short HEAD)" == master ]] || fail 'primary checkout moved off master'
        [[ -z "$(git status --porcelain --untracked-files=all)" ]] || fail 'primary checkout is dirty'
        git fetch --prune origin
        CURRENT_SHA=$(git rev-parse HEAD)
        case "$PR_STATE" in
            OPEN)
                [[ "$CURRENT_SHA" == "$BASE_SHA" ]] || fail 'primary master changed after the run started'
                lookup_pr
                [[ -n "$PR" && "$HEAD_SHA" == "$SHA" && "$PR_STATE" == OPEN ]] ||
                    fail 'merge blocked: PR is not open at the tested SHA'
                verify_hosted_ci
                verify_local_status
                [[ "$HEAD_SHA" == "$SHA" && "$PR_STATE" == OPEN ]] ||
                    fail 'merge blocked: PR changed during final checks'
                [[ "$(git symbolic-ref --quiet --short HEAD)" == master ]] ||
                    fail 'primary checkout moved off master'
                [[ -z "$(git status --porcelain --untracked-files=all)" ]] ||
                    fail 'primary checkout became dirty before merge'
                [[ "$(git rev-parse HEAD)" == "$BASE_SHA" ]] ||
                    fail 'primary master changed before merge'
                gh pr merge "$PR" --repo "$FORGE_REPO" --squash --match-head-commit "$SHA"
                row=$(gh pr view "$PR" --repo "$FORGE_REPO" --json headRefOid,state \
                    --jq '[.headRefOid, .state] | @tsv')
                IFS=$'\t' read -r HEAD_SHA PR_STATE <<< "$row"
                [[ "$HEAD_SHA" == "$SHA" && "$PR_STATE" == MERGED ]] || fail 'GitHub has not completed the exact-SHA merge'
                git fetch --prune origin
                ;;
            MERGED)
                [[ "$CURRENT_SHA" == "$BASE_SHA" || "$CURRENT_SHA" == "$(git rev-parse origin/master)" ]] ||
                    fail 'primary master changed before merge recovery'
                ;;
            *) fail 'PR is not open or merged' ;;
        esac
        # Squash rewrites the tested commit, so a clean local master may diverge.
        # The exact head is merged; realign the owned primary branch to the remote result.
        [[ -z "$(git status --porcelain --untracked-files=all)" ]] ||
            fail 'primary checkout became dirty during merge; realign master manually'
        CURRENT_SHA=$(git rev-parse HEAD)
        [[ "$CURRENT_SHA" == "$BASE_SHA" || "$CURRENT_SHA" == "$(git rev-parse origin/master)" ]] ||
            fail 'primary master changed during merge; realign master manually'
        git merge --ff-only origin/master >/dev/null 2>&1 || git reset --hard origin/master >/dev/null
        [[ "$(git rev-parse HEAD)" == "$(git rev-parse origin/master)" ]] || fail 'primary master did not realign to origin/master'
        atomic_write "$STATE/merged" "$SHA"
        printf 'merged PR %s tested=%s\n' "$PR" "$SHA"
        ;;
esac
