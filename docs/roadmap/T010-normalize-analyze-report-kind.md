---
estado: Pending
tipo: task
---
# T010: Normalize the analyze report `kind` to the `rootline/<name>` convention

**Contribuye a**: JSON contract consistency — every command emits `kind: "rootline/<name>"` (`rootline/query`, `rootline/stats`, `rootline/tree`, …) except `analyze`, which emits `kind: "analyze"` (`internal/infer/report.go:47`).

## Context

Found by the docs-northstar audit (2026-07-22). This is a **breaking contract change**: `cmd/rootline/schema.go:223` auto-detects analyze reports by probing for the exact string `"analyze"`, and external consumers may depend on it. The fix must:

- Emit `rootline/analyze` (or agreed name) from the report writer.
- Keep `schema apply` auto-detection accepting BOTH the old and new kind during a deprecation window, or gate the change behind the report `version` field bump.
- Update any docs/skills that document the report kind.

## Acceptance criteria

- `rootline analyze -o json | jq .kind` returns the normalized kind.
- `rootline schema apply --report <analyze.json>` accepts reports produced before and after the change (or the version field disambiguates).
- Conventional commit marks the breaking surface appropriately (`feat!`/`fix!`) if the old kind is dropped.
