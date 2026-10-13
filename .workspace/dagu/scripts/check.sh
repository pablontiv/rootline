#!/usr/bin/env bash
set -euo pipefail
# Remove forge credentials before loading shared code or launching any check.
unset GH_TOKEN GITHUB_TOKEN
source "$(dirname "$0")/lib.sh"
[[ $# == 3 ]] || fail 'expected SPACE_ID RUN_ID PHASE'
PHASE=$3
init "$1" "$2"
found=false
for item in "${PHASES[@]}"; do [[ "$item" != "$PHASE" ]] || found=true; done
$found || fail "unknown phase: $PHASE"
ensure_worktree
mkdir -p "$STATE/phases"
if [[ ! -e "$STATE/tested.sha" ]]; then
    [[ "$PHASE" == tidy-clean ]] || fail 'first phase must be tidy-clean'
    atomic_write "$STATE/tested.sha" "$(git -C "$WORKTREE" rev-parse HEAD)"
fi
read_sha
clean_at_sha
seal_evidence() {
    local item
    for item in "${PHASES[@]}"; do
        [[ -f "$STATE/phases/$item" && "$(cat "$STATE/phases/$item")" == "$SHA" ]] || fail 'incomplete gate'
    done
    atomic_write "$STATE/evidence" "$SHA"
}
# Checkpoint reuse is allowed only after rechecking SHA and worktree cleanliness.
if [[ -f "$STATE/phases/$PHASE" ]]; then
    [[ "$(cat "$STATE/phases/$PHASE")" == "$SHA" ]] || fail 'phase checkpoint SHA mismatch'
    # Recover a crash between the final phase marker and evidence publication.
    [[ "$PHASE" != gitleaks ]] || seal_evidence
    printf 'checkpoint %s %s\n' "$PHASE" "$SHA"
    exit 0
fi
for item in "${PHASES[@]}"; do
    [[ "$item" != "$PHASE" ]] || break
    [[ -f "$STATE/phases/$item" && "$(cat "$STATE/phases/$item")" == "$SHA" ]] || fail "required earlier phase: $item"
done
cd "$WORKTREE"
case "$PHASE" in
    tidy-clean) go mod tidy ;;
    check) just check ;;
    test) just test ;;
    coverage-check) just coverage-check ;;
    test-install) just test-install ;;
    validate)
        # Build this worktree's source, never use an installed rootline binary.
        go build -o "$STATE/rootline.tmp" ./cmd/rootline
        mv -f "$STATE/rootline.tmp" "$STATE/rootline"
        "$STATE/rootline" validate --all docs/roadmap/
        ;;
    govulncheck) GOTOOLCHAIN=go1.27.2 go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./... ;;
    gitleaks) gitleaks git --redact ;;
esac
clean_at_sha
atomic_write "$STATE/phases/$PHASE" "$SHA"
if [[ "$PHASE" == gitleaks ]]; then
    seal_evidence
fi
printf 'passed %s %s\n' "$PHASE" "$SHA"
