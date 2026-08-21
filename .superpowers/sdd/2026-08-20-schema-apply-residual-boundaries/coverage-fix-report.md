# Coverage fix report

## Scope
- Test-only fix in `internal/fsx/atomic_target_test.go`
- No production code changed

## Change made
Added lexical boundary assertions for `ResolveAtomicTarget`:
- missing allowed root must include `opening root <logical-root>`
- missing parent must include `stat <logical-target>`

## Validation
- `go test ./internal/fsx -race -count=1` ✅
- `go test ./internal/fsx -coverprofile=/tmp/fsx-cover.out -count=1` ✅
- `go tool cover -func=/tmp/fsx-cover.out` ✅
  - package total: `85.0%`
- `git diff --check` ✅

## Notes
- Tests exercise real `ResolveAtomicTarget` calls, not mocks.
- Coverage reached the strict floor without altering production behavior.
