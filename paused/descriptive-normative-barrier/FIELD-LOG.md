---
tipo: field-log
linea: descriptive-normative-barrier
ciclos_registrados: 1
---
# Field Log: descriptive-normative-barrier

> "If it's not in the field log, it didn't happen."

---

## Cycle 1 — 2026-03-03

### Cycle intention

Classify all 13 existing normative inference cases by signal type. Map the gradient to understand where the real boundary is.

### What I did

Took the 13 normative cases found during the inference-engine-architecture investigation and classified each by: what signal it uses, what it claims, and whether that signal is in data form, schema structure, or content meaning.

### Observations

**Classification by signal type:**

| # | Case | Signal type | Signal | Normative claim | File |
|---|------|-------------|--------|----------------|------|
| 1 | DetectDrift | Unanimity | All children same value ≠ parent | Parent SHOULD match children | `drift.go:19-75` |
| 2 | extend_enum | Frequency | 2+ records have same "invalid" value | Schema SHOULD include this value | `proposal.go:203-250` |
| 3 | correct_value | Distance | Levenshtein to closest valid enum value | Data SHOULD have the closest match | `proposal.go:307-338` |
| 4 | migrate_value | Pattern | Value contains `(...)` metadata | Metadata SHOULD be extracted to separate field | `proposal.go:252-305` |
| 5 | InferEstado | Rule | All children same → same; mixed → In Progress | Parent estado SHOULD derive from children | `proposal/infer.go:22-43` |
| 6 | detectInferFromSiblings | Majority | 60%+ siblings share value | Missing field SHOULD match majority | `sibling_infer.go:52-126` |
| 7 | detectOutlierValue | Majority | 75%+ siblings share value | Outlier SHOULD match majority | `sibling_infer.go:128-214` |
| 8 | AggregateAll | Computation | Declared expr evaluated bottom-up | Parent frontmatter SHOULD match computed value | `derive/aggregate.go:24-169` |
| 9 | Required inference | Frequency | >80% presence across records | Field SHOULD be required | `infer.go:112-115` |
| 10 | Enum inference | Frequency + Cardinality | ≤20 unique values, >50% presence | Field SHOULD be enum type | `infer.go:100-105` |
| 11 | distributeFields | Structural | Field at all levels with compatible type | Field SHOULD go to root schema | `hierarchy.go:266-309` |
| 12 | Stem health (8 checks) | Structural | Schema invariants (type consistency, scope match, etc.) | Schema SHOULD follow structural rules | `stemhealth.go:24-275` |
| 13 | DetectRemoveStemField | Structural | Child redefines parent field | Child SHOULD NOT override inherited field | `proposal/stem_health.go:15-51` |

**Grouping by signal source:**

| Signal source | Cases | What it detects | Engine-computable? |
|---------------|-------|-----------------|-------------------|
| **Data patterns** (frequency, majority, unanimity) | 1, 2, 6, 7, 9, 10 | Statistical regularities in record values | Yes — counting |
| **Data relationships** (distance, pattern, rule) | 3, 4, 5 | Structural properties of individual values | Yes — string ops |
| **Schema structure** (structural) | 11, 12, 13 | Invariants about schema itself | Yes — schema analysis |
| **Declared computation** | 8 | Expression evaluation | Yes — expr engine |

**What's NOT here — what the engine can't do:**

| Category | Example | Why not computable |
|----------|---------|-------------------|
| Semantic similarity | "Contribuye a: Restore exitoso" matches "Restore should succeed" | Requires understanding meaning equivalence |
| Contextual disambiguation | "T001-T004 (all running)" as dependency vs casual mention | Requires understanding intent |
| Conceptual relationship | "Memory < 512MB" overlaps with "No leaks during 24h" | Requires understanding domain concepts |
| Domain knowledge | tipo=k8s-workload implies {namespace, image} | Requires knowing what k8s workloads are |

### Reflection

- **What did I learn?**

  The 13 cases group cleanly into 4 signal types, but all 4 are engine-computable. The signal types are: data patterns (counting), data relationships (string operations), schema structure (schema analysis), and declared computation (expression evaluation). None require LLM.

  The cases where LLM IS needed share one characteristic: **the signal is in the meaning of the text, not in its form**. The engine can detect that a value appears in 80% of records (form). It cannot detect that "Restore exitoso" and "Restore should succeed" mean the same thing (meaning).

- **What does this mean?**

  The barrier is not descriptive/normative. It's not even statistical/semantic. It's **form vs meaning**.

  - **Form**: anything detectable by pattern matching on the text's structure — regex, frequency, distance, cardinality, schema topology. All engine-computable.
  - **Meaning**: anything requiring understanding what the text refers to in the world. Requires LLM.

  This is sharper than my initial hypothesis (statistical/semantic). "Statistical" undersells the engine — it also does structural and relational inference, not just counting. "Semantic" is too broad — not all meaning-related tasks are equally hard.

- **What patterns do I see?**

  Hypothesis B (structural/semantic, 3 zones) is partially right — structural is a distinct signal type — but the architectural boundary is still binary: form (engine) vs meaning (LLM). The 3 zones are within the engine side, not spanning the boundary.

  Hypothesis C (no barrier, confidence levels) is wrong — there IS a hard boundary. The engine CANNOT determine that two sentences mean the same thing regardless of confidence level. This isn't a threshold problem; it's a capability boundary.

  The research's "descriptive/normative" framing was wrong because it conflated the **type of claim** (descriptive vs normative) with the **type of signal** (form vs meaning). The engine makes normative claims all the time — "this field SHOULD be required" — it just does it using formal signals, not semantic ones.

### Emerging questions

1. Is the form/meaning boundary actually as hard as it seems? Could embedding-based similarity (computed in Go, no LLM) handle some "meaning" cases? (e.g., cosine similarity on pre-computed embeddings)
2. Does this boundary map cleanly onto the 13 inference categories from the research?
3. Should "computation-then-understanding" theory be renamed to reflect form/meaning rather than computation/understanding?

### Next cycle

The boundary is clearer now. Recommend `/discover reflect` to evaluate:
- Is the form/meaning distinction sharp enough to be a theory?
- Does it replace or extend "computation-then-understanding"?
- Is the line ready to close?

---

*Field log — R&D Framework*
