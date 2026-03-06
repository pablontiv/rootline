---
tipo: field-log
linea: inference-engine-architecture
ciclos_registrados: 3
---
# Field Log: inference-engine-architecture

> "If it's not in the field log, it didn't happen."

---

## Cycle 1 — 2026-03-03

### Cycle intention

Resolve Q5 (body content scope) and Q2 (agent architecture) by investigating:
1. How much body content processing already exists in the Go engine
2. Whether categories 9/11/12/13 (assigned to "agents" in the research) are actually semantic or mostly deterministic

### What I did

- Full codebase audit of body content touchpoints across all packages
- Detailed analysis of categories 9, 11, 12, 13 against actual code capabilities
- Cross-referenced research document claims against implementation reality

### Observations

**Q5 — Body content is NOT outside the engine. The boundary is already porous:**

| Package | Body processing | Nature |
|---------|----------------|--------|
| `internal/extract/` | Wiki-link parsing (`ParseLinks()`), bold-colon metadata extraction (`ScanBodyFields()`), full body stored in `Record.Body` | Structural extraction |
| `internal/graph/` | Builds directed graph from extracted links | Graph computation |
| `internal/query/` | `body contains 'text'` operator, body exposed in expr environment | Full-text search |
| `internal/proposal/` | `detectExtractBody()` scans body for `**Field**: value` patterns, proposes migration to frontmatter | Body-to-frontmatter migration |
| `internal/derive/` | `InjectLinkedFields()` uses links (from body) to resolve cross-document field values | Cross-doc derivation |
| `internal/rules/` | `validateLinks()` validates wiki-links extracted from body against schema rules | Link validation |

The engine already treats body as a **queryable, indexable, and migratable resource**. The research document's characterization of "rootline currently doesn't validate body content" is factually incorrect — it validates link structure from body content.

**Q2 — Agent-layer categories are mostly engine work, not agent work:**

The research assigns categories 9/11/12/13 entirely to the "agent layer." In reality, most of the work in each category is deterministic computation (Go code), not semantic reasoning (LLM). The split is engine vs agent — there is no intermediate "skill layer." Skills are how the agent organizes its work, not a separate computational actor.

| Category | Research says | Engine (Go code) | Agent (LLM) |
|----------|-------------|-------------------|-------------|
| 9 (Dependencies) | Agent needed | ~80%: link extraction (`ParseLinks()`), graph construction, typed link grouping — all existing Go code | ~20%: free-text dependency disambiguation ("T001-T004 (all running)" as dep vs mention) |
| 11 (Traceability) | Semantic matching | ~70%: parent traversal (filepath), heading extraction (regex), list item parsing (regex) | ~30%: semantic matching of contribution text against acceptance criteria |
| 12 (Invariants) | Reasoning for redundancy | ~90%: `INV\d+:` extraction (regex), section detection (regex), grouping | ~10%: redundancy analysis between invariants |
| 13 (Sub-schema by type) | "Program synthesis" | ~95%: YAML block extraction (regex), yaml.v3 parsing, field co-occurrence stats, `requires` rule generation (already implemented) | ~5%: edge cases |

**Key finding**: `internal/infer/hierarchy.go` already has `fieldsCompatible()` which does per-level type specialization with enum value overlap detection. Category 13 is ~80% implemented.

**Correction note**: An earlier version of this analysis used "skill" as an intermediate category between engine and agent. This was incorrect — a skill is a codified procedure that an LLM follows, not independent computation. If a procedure consists of regex, parsing, and counting, it should be Go code (engine), not an LLM following instructions. The proportions above reflect the corrected binary split.

### Reflection

- **What did I learn?**
  The three-layer model from the research is wrong. There are two layers, not three: engine (Go code) and agent (LLM). Skills are not a layer — they are how the agent works. The research assigns ~4 full categories to the agent layer, but ~80% of that work is deterministic computation that belongs in the engine.

- **What does this mean?**
  The boundary is binary: can it be expressed as Go code, or does it need an LLM? If a procedure is regex + parsing + counting, it's engine work regardless of whether it could also be described as a "skill." The agent handles only what genuinely requires language understanding: semantic matching, disambiguation, redundancy detection.

- **What patterns do I see?**
  **Pattern: Computation-then-understanding** — Every inference category decomposes into engine computation (Go) and optional agent reasoning (LLM). There is no middle ground. The engine produces structured evidence; the agent interprets ambiguous cases. Most categories need no agent at all.

### Emerging questions

1. If the engine already processes body content extensively, should categories 6-8 (body structure, back-references, constants) move into the engine?
2. With ~80% of "agent" categories being engine work, does Q2 reduce to: one agent for the ~20% semantic residue?
3. Should the decomposition be by **phase** (compute → understand) rather than by **category**?

### Next cycle

**Hypothesis A (three-layer) is wrong.** Evidence supports **Hypothesis B (two layers)**: engine handles all deterministic computation (including body content), and one agent handles the semantic residue (~20% of categories 9/11/12/13).

Before deciding, need to evaluate:
- Whether categories 4/6/7/8 can move into the engine — this would confirm Q5
- The "computation-then-understanding" pattern as a candidate theory

Recommend: one more cycle focused on categories 4/6/7/8, then `/discover reflect` to decide continue/close/fork.

---

## Cycle 2 — 2026-03-03

### Cycle intention

Evaluate whether categories 4 (structural), 6 (body structure), 7 (back-references), 8 (constants) can move into the Go engine. This resolves Q5 definitively and narrows Q2.

### What I did

- Codebase audit of each category against existing infrastructure in `internal/rules/`, `internal/infer/`, `internal/extract/`, `internal/graph/`
- Checked implementation status, estimated effort, and deterministic nature for each

### Observations

| Category | Status | Effort | Deterministic |
|----------|--------|--------|---------------|
| **4: Structural** (require_index, min/max_children) | **Already implemented** in `internal/rules/structural.go` with full test coverage | Zero | Yes |
| **6: Body structure** (required headings) | Partial — `Record.Body` available, needs heading extraction regex + frequency analysis in `infer.go` | Low (~2-4h) | Yes |
| **7: Back-references** (links to `../README.md`) | Partial — target resolution exists in `graph.go:resolveTarget()`, needs parent-pattern detection | Low (~1-2h) | Yes |
| **8: Constants** (single value in 100% of records) | Data already collected in `FieldStats.UniqueValues`, needs `len(uniq) == 1` check in `infer.go:96-105` | Minimal (~30min) | Yes |

Key details:
- **Cat 4**: `ValidateDirectory()` in `structural.go:12-83` already validates `RequireIndex`, `MinChildren`, `MaxChildren`. Tests in `structural_test.go`. Nothing to build.
- **Cat 6**: `MarkdownExtractor.Extract()` populates `Record.Body`. Missing: `parseHeadings()` function + heading frequency analysis in `Analyze()`.
- **Cat 7**: `graph.go:resolveTarget()` handles `../` paths with `filepath.Clean()`. Missing: detection that a link targets `../README.md` specifically.
- **Cat 8**: `infer.go:92-93` already computes `uniq` slice and populates `stats.UniqueValues`. Missing: one conditional branch for `len(uniq) == 1 && stats.Count == total`.

### Reflection

- **What did I learn?**
  All 4 categories are deterministic and belong in the engine. Category 4 is already shipped. Category 8 is a one-line addition. The research document underestimated how much infrastructure already exists.

- **What does this mean?**
  **Q5 resolved**: Body content is engine territory. The engine already processes body in 6 packages; adding categories 6/7 is incremental, not architectural. The presupposition "body is outside the engine" is definitively false.

  **Q2 resolved**: The question was 1 generalist agent vs N specialized agents. With categories 1-8 in the engine and ~80% of categories 9-13 also being engine-computable (Go code), the semantic residue is ~20% of 4 categories. One agent for that residue.

- **What patterns do I see?**
  The "computation-then-understanding" pattern from Cycle 1 is reinforced. Every category decomposes into engine computation (Go) + optional agent reasoning (LLM). No intermediate layer.

### Emerging questions

1. Q1 (report versioning), Q3 (threshold configurability), Q4 (v3 connection) remain open but have lower roadmap-shape impact
2. Should the "extraction-then-reasoning" pattern be formalized as a theory?
3. Is the line ready for `/discover reflect` → close → `/roadmap`?

### Next cycle

Two questions resolved (Q2, Q5). Recommend `/discover reflect` to evaluate whether to close this line and proceed to roadmap decomposition, or continue exploring Q1/Q3/Q4.

---

## Cycle 3 — 2026-03-03

### Cycle intention

Resolve the 3 remaining open questions: Q1 (report versioning), Q3 (threshold configurability), Q4 (v3 entity model connection).

### What I did

- Audited all 18+ versioned JSON contracts across CLI and MCP to understand rootline's existing versioning pattern
- Located all 8 hardcoded inference thresholds with exact file/line references
- Read the full intrinsic-hierarchy-principle.md research document to assess v3 readiness

### Observations

**Q1 — Report versioning is already answered by existing convention:**

Rootline has 18+ JSON contract types, all using `version: 1` + `kind` field (e.g., `"rootline/validate"`, `"rootline/tree"`). Pattern is additive-only: new fields use `omitempty`, no version negotiation logic exists, no deprecation markers. This is consistent across:
- `internal/query/query.go` (QueryResult, CountResult)
- `internal/rules/result.go` (ValidationResult, BatchValidationResult)
- `internal/proposal/proposal.go` (Proposal.Report)
- `internal/mcp/tools.go` (tree, stats, graph results)
- `cmd/rootline/fix.go` (BatchFixResult)
- `internal/migrate/` (DiffResult, ToV2Result, RenameResult)

**Conclusion**: Not an open question. `analyze` follows the same pattern: `version: 1`, `kind: "rootline/analyze"`, additive-only.

**Q3 — Thresholds: hardcoded is the right default:**

8 thresholds found:

| Threshold | Value | Location |
|-----------|-------|----------|
| Enum max unique | ≤20 | `infer.go:100` |
| Enum presence | >50% | `infer.go:100` |
| Required field | >80% | `infer.go:113` |
| Sibling inference | 60% | `proposal/sibling_infer.go:13` |
| Outlier detection | 75% | `proposal/sibling_infer.go:14` |
| Min siblings infer | 2 | `proposal/sibling_infer.go:15` |
| Min siblings outlier | 3 | `proposal/sibling_infer.go:16` |
| Mixed content warning | >20% | `cmd/rootline/init.go:86` |

No CLI flags, no `.stem` config section, no env vars for tuning. The `analyze` report design already addresses this: it exposes porcentual evidence (`presence: 0.86`), not opinions (`confidence: high`). The consumer (human, skill, agent) applies their own threshold.

**Conclusion**: Keep hardcoded. The report separates observation from judgment. Configurability can be added later as CLI flags without architectural change.

**Q4 — v3 is not ready, `analyze` targets v2 only:**

The intrinsic-hierarchy-principle research (estado: In Progress) has its own unresolved questions:
- Is `children:` hierarchy re-declaration or type constraint? (philosophical tension unresolved)
- Should `index:` be convention or declaration? (Hugo convention vs explicit)
- Can aggregate formulas be inherited across entity types?
- Migration path v2→v3 undefined (3 options listed, none chosen)
- The "circular problem" (Part 5): every attempt to define entity types re-declares the hierarchy

**Conclusion**: v3 is a separate line of inquiry. `analyze` targets v2 `.stem` format only. The additive-only versioning (Q1) means extending to v3 later requires no breaking changes — just new inference categories.

### Reflection

- **What did I learn?**
  Q1 was not actually an open question — the convention already existed. Q3 is a design decision (hardcoded for now), not an architectural question. Q4 is correctly deferred because v3 has its own unresolved research.

- **What does this mean?**
  All 5 questions are resolved. The line of inquiry has reached saturation — no new architectural decisions are needed before roadmap decomposition.

  | Q | Decision | Basis |
  |---|----------|-------|
  | Q1 | Additive-only, `version: 1` + `kind` | 18+ existing contracts follow this pattern |
  | Q2 | One agent | Semantic residue is ~20% of 4 categories |
  | Q3 | Hardcoded, report exposes evidence | 8 thresholds, porcentual evidence in report |
  | Q4 | v2 only, v3 is separate research | v3 has own open questions |
  | Q5 | Body content belongs in engine | Already processed in 6 packages |

- **What patterns do I see?**
  "Computation-then-understanding" pattern confirmed across 3 cycles. Candidate for `/discover theory`.

### Emerging questions

None related to this line. The line is ready for `/discover reflect`.

### Next cycle

No next cycle needed. Recommend `/discover reflect` → CLOSE → formalize "computation-then-understanding" as theory → proceed to `/roadmap`.

---

*Field log — R&D Framework*
