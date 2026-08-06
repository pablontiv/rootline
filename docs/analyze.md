---
estado: Completed
---
# Analyze

Run all inference detectors over a directory and produce a structured report
of schema and content patterns. The report feeds `schema apply` (schema
proposals) and `repair apply` (data-only repairs).

## Usage

```bash
rootline analyze [directory]           # defaults to .
rootline analyze docs/ -o json
rootline analyze docs/ --incremental
rootline analyze docs/ --threshold 0.75
```

## Flags

| Flag | Description |
|------|-------------|
| `--incremental` | Report only inferences not covered by existing `.stem` files |
| `--threshold <0.0-1.0>` | Section-pattern detection threshold (default `0.60`) |

Global flags `--output json|table` and `--field <path>` also apply.

## Detectors

Fourteen detectors run per invocation — twelve data detectors and two
governance detectors.

Markdown is parsed into an AST for every record before the detectors run. This
is required by the section-pattern, invariant, and formal-dependency detectors;
their output is therefore part of the normal command contract rather than an
optional parsing mode.

**Data:** field types, required fields, enum values, constant fields, link
types, back references, cross references, section patterns, invariants,
formal dependencies, traceability links, structural rules.

**Governance:** schema coverage (directories without a `.stem`), validation
gaps (enum without values, untyped fields, incomplete sequences, required
understatement).

### Data Detectors

| Detector | Description |
|----------|-------------|
| Field Type Inference | Infer the data type of each field from record values |
| Required Field Detection | Identify fields present in >80% of records |
| Enum Value Detection | Extract discrete value sets for enum fields |
| Constant Field Detection | Find fields with only one value across all records |
| Link Type Validation | Validate wiki-links against link schema rules |
| Back Reference Detection | Identify missing reciprocal link declarations |
| Cross Reference Detection | Detect broken document-internal cross-references |
| Body Section Patterns | Identify section headings in markdown bodies |
| Invariant Extraction | Extract `INV\d+` identifiers from bodies |
| Formal Dependency Extraction | Extract wiki-link dependencies from bodies |
| Traceability Link Extraction | Identify traceability field claims (`Contribuye a`, `Cubre`, `Satisface`) |
| Structural Rule Detection | Infer directory naming patterns and hierarchy rules |

### Governance Detectors

| Detector | Description |
|----------|-------------|
| Schema Coverage | Directories without an owning `.stem` file |
| Validation Gaps | Missing enum values, untyped fields, sequence incompleteness, required field understatement |

## JSON Output

```json
{
  "version": 1,
  "kind": "rootline/analyze",
  "path": "docs/roadmap/O16-autoupdate-integration",
  "categories": [
    {
      "id": "field_types",
      "name": "Field Type Inference",
      "inference_count": 2,
      "inferences": [
        {
          "type": "field_type",
          "field": "tipo",
          "value": "enum",
          "message": "field \"tipo\" inferred as enum (4/4 records)",
          "requires_agent": false
        },
        {
          "type": "field_type",
          "field": "estado",
          "value": "enum",
          "message": "field \"estado\" inferred as enum (3/4 records)",
          "requires_agent": false
        }
      ]
    },
    {
      "id": "required_fields",
      "name": "Required Field Detection",
      "inference_count": 1,
      "inferences": [
        {
          "type": "required_field",
          "field": "tipo",
          "message": "field \"tipo\" appears in >80% of records — required",
          "requires_agent": false
        }
      ]
    },
    {
      "id": "enum_values",
      "name": "Enum Value Detection",
      "inference_count": 2,
      "inferences": [
        {
          "type": "enum_values",
          "field": "tipo",
          "value": "[outcome task]",
          "message": "field \"tipo\" has enum values: [outcome task]",
          "requires_agent": false
        },
        {
          "type": "enum_values",
          "field": "estado",
          "value": "[Completed]",
          "message": "field \"estado\" has enum values: [Completed]",
          "requires_agent": false
        }
      ]
    }
  ],
  "summary": {
    "total_inferences": 24,
    "agent_required": 3,
    "engine_resolved": 21
  }
}
```

### Output Fields

- `version` — contract version.
- `kind` — `rootline/analyze` (distinguishes from other report kinds).
- `path` — the scanned directory (as specified on the command line).
- `categories[]` — one per detector: `id`, `name`, `inference_count`, `inferences[]`.
  - Each inference carries: `type`, `field`, `value`, `message`, `requires_agent`.
  - `requires_agent: true` marks inferences needing human or agent disambiguation (not automatically applied by fix/schema commands).
- `summary`:
  - `total_inferences` — count of all inferences across all detectors.
  - `agent_required` — count of inferences with `requires_agent: true`.
  - `engine_resolved` — count of inferences ready for automatic application.

## Consuming the Report

Analyze generates two kinds of inferences:

- **Schema proposals** — field types, enums, required flags, structural rules. Feed these to `schema apply` to update `.stem` files.
- **Data repairs** — enum corrections, field additions, value migrations. Feed these to `repair apply` to update document frontmatter.

### Workflow

```bash
# Generate analyze report
rootline analyze docs/ -o json > analyze.json

# Preview schema changes
rootline schema apply --report analyze.json --dry-run

# Apply schema changes (--incremental to skip already-covered inferences)
rootline schema apply --report analyze.json

# Preview data repairs
rootline repair apply --report analyze.json --dry-run

# Apply data repairs
rootline repair apply --report analyze.json
```

Inferences with `requires_agent: true` are logged in the report but skipped by both `schema apply` and `repair apply`. Human or agent review is needed to resolve them; they remain actionable for future tooling.

## Filtering with --incremental

By default, analyze reports all inferences. Use `--incremental` to report only inferences not already covered by existing `.stem` files:

```bash
rootline analyze docs/ --incremental -o json
```

This is useful in iterative schema refinement: each run shows only the new patterns not yet encoded in `.stem` files, avoiding re-reporting known patterns.

## Threshold Tuning

Section-pattern detection sensitivity is controlled by `--threshold` (default `0.60`, range `0.0-1.0`). Higher threshold = fewer pattern proposals:

```bash
rootline analyze docs/ --threshold 0.80   # Conservative
rootline analyze docs/ --threshold 0.40   # Aggressive
```

Structural naming analysis scores directory names and Markdown record-file
stems as separate populations. An unrelated directory therefore cannot become
an outlier merely because the files beside it follow a record naming pattern.
