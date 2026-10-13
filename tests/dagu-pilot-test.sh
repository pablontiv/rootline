#!/usr/bin/env bash
# Offline pilot fault tests: real local Git topology, stubbed gate/forge commands.
set -euo pipefail
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_COMMON_DIR GIT_PREFIX GIT_NAMESPACE GIT_CEILING_DIRECTORIES GIT_DISCOVERY_ACROSS_FILESYSTEM
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null
SOURCE=$(cd "$(dirname "$0")/.." && pwd -P)
TMP=$(mktemp -d)
TMP=$(cd "$TMP" && pwd -P)
trap 'rm -rf "$TMP"' EXIT
trap 'exit 1' HUP INT TERM
fail() { printf 'dagu-pilot-test: %s\n' "$*" >&2; exit 1; }
expect_failure() { if "$@" >"$TMP/expected-failure.log" 2>&1; then fail "unexpected success: $*"; fi; }
assert_absent() { [[ ! -e "$1" ]] || fail "unexpected resource: $1"; }
line_count() { wc -l < "$1" | tr -d ' '; }
mkdir -p "$TMP/repo/.workspace/dagu/scripts" "$TMP/stubs" "$TMP/stub-state"
cp "$SOURCE"/.workspace/dagu/scripts/*.sh "$TMP/repo/.workspace/dagu/scripts/"
printf '.workspace/dagu/data/\n.workspace/dagu/logs/\n' > "$TMP/repo/.gitignore"
printf 'fixture\n' > "$TMP/repo/record"
git init --bare --initial-branch=master "$TMP/remote.git" >/dev/null
git -C "$TMP/repo" init --initial-branch=master >/dev/null
git -C "$TMP/repo" config user.email pilot@example.invalid
git -C "$TMP/repo" config user.name 'Dagu pilot test'
git -C "$TMP/repo" add .
git -C "$TMP/repo" commit -m 'chore: fixture' >/dev/null
git -C "$TMP/repo" remote add origin "$TMP/remote.git"
git -C "$TMP/repo" push -u origin master >/dev/null
# Advance only the remote, proving sync fetches and fast-forwards before add.
git clone "$TMP/remote.git" "$TMP/peer" >/dev/null
git -C "$TMP/peer" config user.email pilot@example.invalid
git -C "$TMP/peer" config user.name 'Dagu pilot test'
printf 'remote advance\n' >> "$TMP/peer/record"
git -C "$TMP/peer" commit -am 'chore: advance' >/dev/null
git -C "$TMP/peer" push >/dev/null
REMOTE_SHA=$(git -C "$TMP/peer" rev-parse HEAD)
git clone "$TMP/remote.git" "$TMP/stub-state/squash" >/dev/null
git -C "$TMP/stub-state/squash" config user.email pilot@example.invalid
git -C "$TMP/stub-state/squash" config user.name 'Dagu pilot test'
cat > "$TMP/stubs/gate" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
[[ ! ${GH_TOKEN+x} && ! ${GITHUB_TOKEN+x} ]] || { echo 'forge credential leaked into gate' >&2; exit 90; }
[[ "$(umask)" == 0022 ]] || { echo 'gate did not normalize umask' >&2; exit 93; }
name=$(basename "$0")
printf '%s %s\n' "$name" "$*" >> "$STUB_STATE/gates"
if [[ "$name" == go && "$1" == run && "$2" == golang.org/x/vuln/cmd/govulncheck@v1.9.0 ]]; then
    [[ "${GOTOOLCHAIN:-}" == go1.27.2 ]] || { echo 'govulncheck did not use the patched toolchain' >&2; exit 94; }
fi
if [[ "$name" == just && "$1" == test && -f "$STUB_STATE/fail-test" ]]; then
    rm "$STUB_STATE/fail-test"
    exit 71
fi
if [[ "$name" == go && "$1" == build ]]; then
    [[ "$2" == -o && "$4" == ./cmd/rootline ]] || exit 91
    cat > "$3" <<'BUILT'
#!/usr/bin/env bash
set -euo pipefail
[[ ! ${GH_TOKEN+x} && ! ${GITHUB_TOKEN+x} ]] || exit 90
[[ "$*" == 'validate --all docs/roadmap/' ]] || exit 92
printf 'rootline %s\n' "$*" >> "$STUB_STATE/gates"
BUILT
    chmod +x "$3"
fi
STUB
chmod +x "$TMP/stubs/gate"
for command in go just govulncheck gitleaks; do ln -s gate "$TMP/stubs/$command"; done
cat > "$TMP/stubs/gh" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
[[ ${GH_TOKEN:-} == forge-secret && ${GITHUB_TOKEN:-} == github-secret ]] || exit 93
printf '%s\n' "$*" >> "$STUB_STATE/forge"
case "$1 $2" in
    'repo view') echo fixture/rootline ;;
    'pr list')
        [[ "$*" == *'--head dagu/pilot/one'* && "$*" != *'--head fixture:'* ]] || exit 97
        if [[ -f "$STUB_STATE/pr" ]]; then
            head=$(cat "$STUB_STATE/sha")
            if [[ -f "$STUB_STATE/pr-head-override" ]]; then
                remote_head=$(git -C "$PRIMARY_REPO" ls-remote origin refs/heads/dagu/pilot/one | cut -f1)
                if [[ "$remote_head" == "$head" ]]; then
                    rm "$STUB_STATE/pr-head-override"
                else
                    head=$(cat "$STUB_STATE/pr-head-override")
                fi
            fi
            printf '17\t%s\t%s\thttps://example.invalid/pull/17\n' "$head" "$(cat "$STUB_STATE/pr-state")"
        fi
        ;;
    'pr create')
        [[ ! -e "$STUB_STATE/pr" ]] || exit 94
        [[ "$*" == *'--head fixture:dagu/pilot/one'* ]] || exit 97
        [[ "$*" == *'--title feat: pilot fixture change'* && "$*" == *'Closes #261.'* ]] || exit 98
        touch "$STUB_STATE/pr"
        echo OPEN > "$STUB_STATE/pr-state"
        echo created >> "$STUB_STATE/creates"
        if [[ -f "$STUB_STATE/fail-create-response" ]]; then rm "$STUB_STATE/fail-create-response"; exit 75; fi
        echo https://example.invalid/pull/17
        ;;
    'pr view') printf '%s\t%s\n' "$(cat "$STUB_STATE/sha")" "$(cat "$STUB_STATE/pr-state")" ;;
    'pr checks')
        echo checked >> "$STUB_STATE/ci-calls"
        if [[ "$*" == *'--watch'* ]]; then
            if [[ -f "$STUB_STATE/fail-ci" ]]; then rm "$STUB_STATE/fail-ci"; exit 8; fi
        elif [[ ! -f "$STUB_STATE/no-hosted-ci" ]]; then
            printf 'CI / test\tpass\tCI\nSecurity / scan\tpass\tCI\nrelease\tskipping\tCI\n'
        fi
        ;;
    'pr merge')
        [[ "$*" == *'--squash --match-head-commit '*"$(cat "$STUB_STATE/sha")" ]] || exit 95
        git -C "$STUB_STATE/squash" fetch -q origin
        base=$(git -C "$STUB_STATE/squash" rev-parse origin/master)
        tree=$(git -C "$STUB_STATE/squash" rev-parse "$(cat "$STUB_STATE/sha")^{tree}")
        merged=$(printf 'chore: squash pilot run\n' | git -C "$STUB_STATE/squash" commit-tree "$tree" -p "$base")
        git -C "$STUB_STATE/squash" push -q origin "$merged:master"
        echo MERGED > "$STUB_STATE/pr-state"
        echo merged >> "$STUB_STATE/merges"
        if [[ -f "$STUB_STATE/dirty-primary-on-merge" ]]; then
            rm "$STUB_STATE/dirty-primary-on-merge"
            printf 'unsaved during merge\n' >> "$PRIMARY_REPO/record"
        fi
        if [[ -f "$STUB_STATE/commit-primary-on-merge" ]]; then
            rm "$STUB_STATE/commit-primary-on-merge"
            printf 'committed during merge\n' >> "$PRIMARY_REPO/record"
            git -C "$PRIMARY_REPO" add record
            git -C "$PRIMARY_REPO" commit -m 'chore: concurrent primary commit' >/dev/null
        fi
        ;;
    *)
        if [[ "$1" == api && "$2" == --method ]]; then
            echo success > "$STUB_STATE/status"
            echo posted >> "$STUB_STATE/status-posts"
        elif [[ "$1" == api ]]; then
            [[ "$2" == "repos/fixture/rootline/commits/$(cat "$STUB_STATE/sha")/status" ]] || exit 99
            [[ ! -f "$STUB_STATE/status" ]] || cat "$STUB_STATE/status"
        else exit 96; fi
        ;;
esac
STUB
chmod +x "$TMP/stubs/gh"
export PATH="$TMP/stubs:$PATH" STUB_STATE="$TMP/stub-state" PRIMARY_REPO="$TMP/repo"
export ROOTLINE_WORKTREE_ROOT="$TMP/worktrees" GH_TOKEN=forge-secret GITHUB_TOKEN=github-secret ISSUE_NUMBER=261
SCRIPTS="$TMP/repo/.workspace/dagu/scripts"
STATE="$TMP/repo/.workspace/dagu/data/rootline-runs/pilot/one"
WORKTREE="$TMP/worktrees/pilot/one"
LOCK="$TMP/repo/.git/rootline-dagu.lock"
FETCH_WORKTREE="$TMP/worktrees/pilot/fetch"
"$SCRIPTS/sync.sh" pilot fetch >/dev/null
[[ "$(git -C "$FETCH_WORKTREE" rev-parse HEAD)" == "$REMOTE_SHA" ]] || fail 'sync did not fast-forward before worktree'
"$SCRIPTS/cleanup.sh" pilot fetch >/dev/null
printf 'feature change\n' >> "$TMP/repo/record"
git -C "$TMP/repo" commit -am 'feat: pilot fixture change' >/dev/null
FEATURE_SHA=$(git -C "$TMP/repo" rev-parse HEAD)
"$SCRIPTS/sync.sh" pilot one >/dev/null
[[ "$(git -C "$WORKTREE" rev-parse HEAD)" == "$FEATURE_SHA" ]] || fail 'run did not branch from committed local tip'
[[ "$(cat "$LOCK")" == $'pilot\none' ]] || fail 'lock lacks exact owner'
expect_failure "$SCRIPTS/sync.sh" pilot two
expect_failure "$SCRIPTS/cleanup.sh" pilot two
[[ -d "$WORKTREE" && -f "$LOCK" ]] || fail 'foreign cleanup changed owned resources'
expect_failure "$SCRIPTS/cleanup.sh" ../pilot one
expect_failure "$SCRIPTS/check.sh" pilot one test
"$SCRIPTS/check.sh" pilot one tidy-clean >/dev/null
"$SCRIPTS/check.sh" pilot one check >/dev/null
touch "$STUB_STATE/fail-test"
expect_failure "$SCRIPTS/check.sh" pilot one test
assert_absent "$STATE/phases/test"
assert_absent "$STATE/evidence"
expect_failure "$SCRIPTS/forge.sh" pilot one pr
# Terminal cleanup preserves evidence/checkpoints; a failed-step retry rehydrates
# the exact original SHA and does not rerun successful phases.
"$SCRIPTS/cleanup.sh" pilot one >/dev/null
assert_absent "$WORKTREE"; assert_absent "$LOCK"
"$SCRIPTS/check.sh" pilot one test >/dev/null
[[ -f "$STATE/phases/check" && -f "$STATE/phases/test" ]] || fail 'retry lost checkpoints'
lines_before=$(line_count "$STUB_STATE/gates")
"$SCRIPTS/check.sh" pilot one check >/dev/null
[[ "$(line_count "$STUB_STATE/gates")" == "$lines_before" ]] || fail 'successful phase ran again'
for phase in coverage-check test-install validate govulncheck gitleaks; do "$SCRIPTS/check.sh" pilot one "$phase" >/dev/null; done
SHA=$(git -C "$WORKTREE" rev-parse HEAD)
[[ "$(cat "$STATE/evidence")" == "$SHA" ]] || fail 'final evidence is not SHA-bound'
lines_before=$(line_count "$STUB_STATE/gates")
rm "$STATE/evidence"
"$SCRIPTS/check.sh" pilot one gitleaks >/dev/null
[[ "$(cat "$STATE/evidence")" == "$SHA" && "$(line_count "$STUB_STATE/gates")" == "$lines_before" ]] || fail 'final evidence publication did not recover from a crash'
[[ $(grep -c '^go mod tidy$' "$STUB_STATE/gates") == 1 ]] || fail 'tidy checkpoint reran'
[[ $(grep -c '^just test$' "$STUB_STATE/gates") == 2 ]] || fail 'failed phase was not retried exactly once'
grep -q '^rootline validate --all docs/roadmap/$' "$STUB_STATE/gates" || fail 'did not use worktree-built validator'
grep -q '^go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...$' "$STUB_STATE/gates" || fail 'govulncheck version is not pinned'
printf '%s\n' "$SHA" > "$STUB_STATE/sha"
expect_failure env -u ISSUE_NUMBER "$SCRIPTS/forge.sh" pilot one pr
# An existing PR may advance only when its old head is an ancestor of the newly
# tested SHA; the push and subsequent lookup must bind it to current evidence.
BASE_SHA=$(git -C "$WORKTREE" rev-parse "$SHA^")
git -C "$TMP/repo" push origin "$BASE_SHA:refs/heads/dagu/pilot/one" >/dev/null
touch "$STUB_STATE/pr"
printf 'OPEN\n' > "$STUB_STATE/pr-state"
printf '%s\n' "$BASE_SHA" > "$STUB_STATE/pr-head-override"
"$SCRIPTS/forge.sh" pilot one pr >/dev/null
[[ "$(git -C "$TMP/repo" ls-remote origin refs/heads/dagu/pilot/one | cut -f1)" == "$SHA" ]] || fail 'existing PR branch did not fast-forward to tested SHA'
rm "$STUB_STATE/pr" "$STUB_STATE/pr-state"
git -C "$TMP/repo" push origin :refs/heads/dagu/pilot/one >/dev/null
# Failure after server-side creation must look up and reuse the PR next time.
touch "$STUB_STATE/fail-create-response"
expect_failure "$SCRIPTS/forge.sh" pilot one pr
"$SCRIPTS/forge.sh" pilot one pr >/dev/null
expect_failure env ISSUE_NUMBER=262 "$SCRIPTS/forge.sh" pilot one status
[[ "$(line_count "$STUB_STATE/creates")" == 1 ]] || fail 'PR duplicated after lost response'
"$SCRIPTS/forge.sh" pilot one status >/dev/null
"$SCRIPTS/forge.sh" pilot one status >/dev/null
[[ "$(line_count "$STUB_STATE/status-posts")" == 1 ]] || fail 'successful status was republished'
touch "$STUB_STATE/fail-ci"
expect_failure "$SCRIPTS/forge.sh" pilot one ci
assert_absent "$STATE/ci.sha"
touch "$STUB_STATE/no-hosted-ci"
expect_failure "$SCRIPTS/forge.sh" pilot one ci
rm "$STUB_STATE/no-hosted-ci"
"$SCRIPTS/forge.sh" pilot one ci >/dev/null
[[ "$(cat "$STATE/ci.sha")" == "$SHA" ]] || fail 'hosted CI evidence is not SHA-bound'
grep -q $'^CI / test\tpass\tCI$' "$STATE/ci.checks" || fail 'hosted CI check evidence is incomplete'
printf 'failure\n' > "$STUB_STATE/status"
expect_failure "$SCRIPTS/forge.sh" pilot one merge
assert_absent "$STUB_STATE/merges"
rm "$STUB_STATE/status"
expect_failure "$SCRIPTS/forge.sh" pilot one merge
assert_absent "$STUB_STATE/merges"
"$SCRIPTS/forge.sh" pilot one status >/dev/null
# Mutating local HEAD after the gate invalidates every checkpoint and forge action.
printf 'late commit\n' >> "$WORKTREE/record"
git -C "$WORKTREE" commit -am 'chore: late commit' >/dev/null
expect_failure "$SCRIPTS/check.sh" pilot one test
expect_failure "$SCRIPTS/forge.sh" pilot one merge
assert_absent "$STUB_STATE/merges"
"$SCRIPTS/cleanup.sh" pilot one >/dev/null
"$SCRIPTS/cleanup.sh" pilot one >/dev/null
assert_absent "$WORKTREE"; assert_absent "$LOCK"
if git -C "$TMP/repo" show-ref --verify --quiet refs/heads/dagu/pilot/one; then fail 'owned branch leaked'; fi
# A stale remote PR head still blocks merge when local resources are gone.
printf '%040d\n' 1 > "$STUB_STATE/sha"
expect_failure "$SCRIPTS/forge.sh" pilot one merge
assert_absent "$STUB_STATE/merges"
"$SCRIPTS/cleanup.sh" pilot one >/dev/null
printf '%s\n' "$SHA" > "$STUB_STATE/sha"
touch "$STUB_STATE/fail-ci"
expect_failure "$SCRIPTS/forge.sh" pilot one merge
assert_absent "$STUB_STATE/merges"
touch "$STUB_STATE/commit-primary-on-merge"
expect_failure "$SCRIPTS/forge.sh" pilot one merge
grep -q 'committed during merge' "$TMP/repo/record" || fail 'merge discarded a concurrent primary commit'
[[ "$(git -C "$TMP/repo" log -1 --format=%s)" == 'chore: concurrent primary commit' ]] ||
    fail 'merge rewrote a concurrent primary commit'
git -C "$TMP/repo" reset --hard "$(cat "$STATE/base.sha")" >/dev/null
printf 'unsaved during merge recovery\n' >> "$TMP/repo/record"
expect_failure "$SCRIPTS/forge.sh" pilot one merge
grep -q 'unsaved during merge recovery' "$TMP/repo/record" || fail 'recovery discarded a concurrent primary edit'
git -C "$TMP/repo" checkout -- record
"$SCRIPTS/forge.sh" pilot one merge >/dev/null
"$SCRIPTS/forge.sh" pilot one merge >/dev/null
[[ "$(line_count "$STUB_STATE/merges")" == 1 ]] || fail 'merge was not idempotent'
[[ "$(git -C "$TMP/repo" rev-parse HEAD)" == "$(git --git-dir="$TMP/remote.git" rev-parse master)" ]] || fail 'primary master did not realign after squash'
"$SCRIPTS/cleanup.sh" pilot one >/dev/null
"$SCRIPTS/cleanup.sh" pilot one >/dev/null
assert_absent "$LOCK"; assert_absent "$WORKTREE"
STALE_WORKTREE="$TMP/worktrees/pilot/stale"
"$SCRIPTS/sync.sh" pilot stale >/dev/null
rm -rf "$STALE_WORKTREE"
"$SCRIPTS/cleanup.sh" pilot stale >/dev/null
assert_absent "$LOCK"; assert_absent "$STALE_WORKTREE"
if git -C "$TMP/repo" show-ref --verify --quiet refs/heads/dagu/pilot/stale; then fail 'stale registered branch leaked'; fi
printf 'dagu-pilot-test: PASS (lock ownership, checkpoint retry, exact-SHA evidence, forge idempotency, cleanup, credential isolation)\n'
