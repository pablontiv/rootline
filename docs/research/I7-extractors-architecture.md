# I7 — Extractors Architecture: Abstracted Extraction Pipeline

**Fecha**: 2026-02-17
**Estado**: Done
**Pregunta**: How to abstract the extraction pipeline so the core ships Markdown-only but the architecture supports future formats via plugins?
**Motivación**: Problem-driven — addresses "7 independent parsing systems" (highest-frequency problem in intent doc §2)

---

## 1. Methodology

Three inputs, same evidence-first approach as I1 and I5:

1. **Internal evidence**: The 9 consumers analyzed in the intent document (§2), particularly
   the `parse_frontmatter` function in the `/roadmap pending` skill — the most complete
   extraction implementation in the codebase.

2. **External systems**: 7 comparable extraction architectures analyzed (Apache Tika, Hugo,
   MarkdownDB, Obsidian Dataview, Zola, frontmatter-mcp, EditorConfig). Each assessed for:
   interface design, registry pattern, content vs metadata separation, scope/matching, and
   record shape.

3. **Constraint-driven design**: The interface must satisfy three non-negotiable constraints:
   - Only Markdown is built-in (compiled into the binary)
   - All other formats are **plugins** (the mechanism is deferred to I2, but the interface
     must be serializable to cross process boundaries)
   - Output must be consumable uniformly by validation, derivation, and query

---

## 2. Internal Evidence: The 9 Consumers

Every consumer in the homeserver automation parses YAML frontmatter from Markdown.
None uses a proper YAML parser. Each re-implements discovery, extraction, and validation.

### Extraction techniques

| Technique | Used by | Reliability | Testable |
|-----------|---------|-------------|----------|
| Python `line.split(':', 1)` on frontmatter | `/roadmap pending` | Medium (fragile with colons in values) | Manual only |
| Bash `grep -q "pattern"` (substring) | `/service`, `/module`, `/operation` | High (simple) | Shell tests |
| Bash `grep -qE` (extended) | `/instance` | High | Shell tests |
| Bash `sort -V \| tail -1` (sequence) | `/prd` (auto-numbering) | High | Shell tests |
| LLM agent prompt | Write hook, SubagentStop hook, `/roadmap view` | Low (non-deterministic) | Not testable |
| Path-based tree traversal | `/roadmap pending` (context) | High | File tree tests |

### File discovery patterns (4 distinct)

| Pattern | Consumer | Type |
|---------|----------|------|
| `T[0-9][0-9][0-9]-*.md` | `/roadmap pending` | Python glob |
| `T${ID}-*.md` | `/service`, `/module`, `/operation`, `/instance` | Shell glob by ID |
| `T*-$ARGUMENTS*.md` | `/host-task`, `/instance-task`, `/drift` | Fuzzy name match |
| `T` + digits in filename | Write hook | Natural language |

### The silent failure problem

When `estado:` format changes, `/roadmap pending` returns an empty table with no error.
The Python splitter and bash grep patterns fail silently because they have no schema to
validate against. This is the core problem Rootline solves.

---

## 3. State of the Art

| System | Language | Interface | Input | Output | Registry | Scope |
|--------|----------|-----------|-------|--------|----------|-------|
| **Apache Tika** | Java | `Parser.parse(InputStream, ContentHandler, Metadata, ParseContext)` | InputStream | Metadata map + SAX events | `CompositeParser` by MIME type | MIME detection |
| **Hugo** | Go | `metadecoders.Format` | `io.Reader` | `map[string]any` + `[]byte` | Format enum by delimiter | Directory + archetype |
| **MarkdownDB** | Node.js | `processMarkdown(source)` | String | FileInfo object | Hardcoded to `.md`/`.mdx` | Extension filter |
| **Obsidian Dataview** | TypeScript | `extractInlineFields(text)` | String | `PageMetadata` | Hardcoded (frontmatter + inline) | All `.md` in vault |
| **Zola** | Rust | `pulldown-cmark` + TOML | File content | Page struct | Hardcoded to TOML | Section `_index.md` |
| **frontmatter-mcp** | TypeScript | `gray-matter` | String | DuckDB row | Hardcoded to YAML | Glob config |
| **EditorConfig** | Various | Section-based INI | File content | Key-value properties | N/A | Glob in sections |

### Key findings

1. **Tika is the only system with a true extractor abstraction.** Its `Parser` interface with
   `CompositeParser` registry is the gold standard, but over-engineered for documentation
   files (<100KB). MIME detection and SAX events are unnecessary.

2. **Hugo is closest to Rootline's needs.** Detects frontmatter format by delimiter
   (`---` = YAML, `+++` = TOML), unmarshals to `map[string]any`, returns content separately.
   But formats are hardcoded in a switch statement — not pluggable.

3. **No system separates file I/O from content extraction.** Every system reads the file
   itself. Rootline should separate: engine handles I/O, extractor receives `[]byte`.
   This makes extractors testable without filesystem access and serializable for plugins.

4. **All systems produce metadata as `map[string]any` or equivalent.** No system uses typed
   structs. Correct for Rootline: schemas live in `.stem`, not in Go types.

5. **Scope matching universally belongs to the engine, not the extractor.** EditorConfig uses
   globs in config. Hugo uses directory types. Tika uses MIME detection. Rootline's
   `scope.match` follows the same separation.

---

## 4. Core Specification

### 4.1 The `Extractor` Interface

```go
// Extractor extracts structured metadata and body text from file content.
// Implementations are format-specific (Markdown built-in; others via plugins).
// Extractors receive content as bytes — they do NOT perform file I/O.
//
// This interface is designed to be serializable: []byte in, Record out.
// Plugin extractors (I2) will use the same contract across process boundaries.
type Extractor interface {
    // Extract parses content and returns a Record.
    // The path is for error reporting only, not file access.
    Extract(path string, content []byte) (*Record, error)

    // Extensions returns file extensions this extractor handles.
    // Example: [".md", ".markdown"]
    Extensions() []string

    // Name returns the extractor identifier.
    // Surfaces in query output ("type": "markdown") and registry keys.
    Name() string
}
```

### 4.2 The `Record` Type

```go
// Record is the universal output of all extractors and the universal input
// to validation, derivation, and query. All fields are JSON-serializable.
type Record struct {
    // Path relative to repository root.
    Path string `json:"path"`

    // Type is the extractor name that produced this record.
    Type string `json:"type"`

    // Frontmatter contains extracted structured metadata.
    // For Markdown: YAML frontmatter fields.
    // For YAML files: top-level document keys.
    // For JSON files: top-level object keys.
    Frontmatter map[string]any `json:"frontmatter"`

    // Body is document content excluding metadata.
    // For Markdown: everything after closing --- of frontmatter.
    // For pure-metadata formats (YAML/JSON/TOML): empty string.
    // Used by `contains` operator on `body` field (I1).
    Body string `json:"body"`

    // Errors contains non-fatal extraction issues.
    // Fatal errors are returned from Extract(), not stored here.
    Errors []ExtractionError `json:"errors,omitempty"`
}

// ExtractionError represents a non-fatal issue during extraction.
type ExtractionError struct {
    Line    int    `json:"line"`    // 0 if unknown
    Message string `json:"message"`
}
```

### 4.3 The Registry

```go
// Registry maps file extensions and names to extractors.
type Registry struct {
    byName      map[string]Extractor
    byExtension map[string]Extractor
}

// NewRegistry creates a registry with the built-in extractors.
func NewRegistry() *Registry {
    r := &Registry{
        byName:      make(map[string]Extractor),
        byExtension: make(map[string]Extractor),
    }
    r.Register(&MarkdownExtractor{})
    return r
}

// Register adds an extractor to the registry.
// Panics on duplicate (programmer error, not user error).
func (r *Registry) Register(e Extractor) {
    if _, exists := r.byName[e.Name()]; exists {
        panic(fmt.Sprintf("extractor %q already registered", e.Name()))
    }
    r.byName[e.Name()] = e
    for _, ext := range e.Extensions() {
        if existing, exists := r.byExtension[ext]; exists {
            panic(fmt.Sprintf("extension %q already registered by %q", ext, existing.Name()))
        }
        r.byExtension[ext] = e
    }
}

// ForFile returns the extractor for a given file path.
// Returns nil if no extractor matches (file is skipped, not an error).
func (r *Registry) ForFile(path string, stemExtractor string) Extractor {
    if stemExtractor != "" {
        return r.byName[stemExtractor]
    }
    ext := filepath.Ext(path)
    return r.byExtension[ext]
}
```

---

## 5. Practical Application: Markdown Extractor

```go
type MarkdownExtractor struct{}

func (m *MarkdownExtractor) Name() string        { return "markdown" }
func (m *MarkdownExtractor) Extensions() []string { return []string{".md", ".markdown"} }

func (m *MarkdownExtractor) Extract(path string, content []byte) (*Record, error) {
    record := &Record{
        Path:        path,
        Type:        m.Name(),
        Frontmatter: make(map[string]any),
    }

    text := string(content)

    // Strip BOM if present
    text = stripBOM(text)

    // No frontmatter — entire content is body
    if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
        record.Body = text
        return record, nil
    }

    // Find closing delimiter
    closeIdx := findClosingDelimiter(text)
    if closeIdx < 0 {
        record.Body = text
        record.Errors = append(record.Errors, ExtractionError{
            Line:    1,
            Message: "unclosed frontmatter delimiter",
        })
        return record, nil
    }

    // Parse YAML frontmatter
    fmContent := text[4:closeIdx]
    if err := yaml.Unmarshal([]byte(fmContent), &record.Frontmatter); err != nil {
        record.Errors = append(record.Errors, ExtractionError{
            Line:    1,
            Message: fmt.Sprintf("malformed YAML frontmatter: %v", err),
        })
        // Line-by-line fallback (matches existing consumer resilience)
        record.Frontmatter = fallbackParseFrontmatter(fmContent)
    }

    // Body is everything after closing delimiter
    bodyStart := closeIdx + 4
    if bodyStart < len(text) {
        record.Body = strings.TrimLeft(text[bodyStart:], "\r\n")
    }

    return record, nil
}
```

The **fallback parser** is significant. The existing consumer in `/roadmap pending` uses
`line.split(':', 1)` as its entire strategy. This works on partially-valid YAML that
`gopkg.in/yaml.v3` would reject. The Markdown extractor includes a line-by-line fallback
that extracts what it can from malformed frontmatter, reporting issues in `Record.Errors`.

---

## 6. Pipeline Integration

How a Record flows through the full pipeline:

```
1. File Discovery      → Scanner finds files matching scope.match
2. File I/O            → Scanner reads []byte content
3. Extractor Selection → Registry.ForFile(path, stemExtractor)
4. Extraction          → extractor.Extract(path, content) → *Record
5. Rule Loading        → Engine resolves effective .stem for Record.Path
6. Validation          → Rules engine checks Record.Frontmatter against schema
7. Derivation          → Rules engine computes derived fields
8. Query               → Query engine filters/returns Records matching WHERE clause
```

The extractor is step 4 of 8. It knows nothing about steps 5-8.

**What the extractor does NOT do**:
- File discovery (engine's scanner, respects `.gitignore`)
- Scope matching (`scope.match` evaluated by engine before calling extractor)
- Schema validation (rules engine)
- Derived field computation (derivation engine)
- File I/O (engine reads, passes `[]byte`)

---

## 7. Edge Cases

| # | Case | Resolution | Rationale |
|---|------|------------|-----------|
| EC-1 | File has no frontmatter | Empty Frontmatter, full Body | Valid document. Validation may flag missing required fields. |
| EC-2 | Malformed YAML frontmatter | Partial parse + ExtractionError | Better to extract 4/5 fields than 0/5. Matches existing consumer resilience. |
| EC-3 | Empty file (0 bytes) | Empty Frontmatter, empty Body | Not an extraction error. Validation may flag it. |
| EC-4 | Binary file matched by glob | `Extract` returns fatal error | Cannot interpret binary as text. File skipped with warning. |
| EC-5 | TOML frontmatter (`+++`) in .md | Not extracted | Markdown extractor handles YAML (`---`) only. No real consumer uses TOML frontmatter. |
| EC-6 | Duplicate YAML keys | Last value wins | `gopkg.in/yaml.v3` behavior per YAML 1.2 spec. |
| EC-7 | BOM at start of file | Strip before parsing | Common in Windows-edited files. UTF-8 BOM detected and removed. |
| EC-8 | Frontmatter-only file (no body) | Populated Frontmatter, empty Body | Valid — some files are pure metadata. |

---

## 8. Decisions

| # | Decision | Result | Rationale | Rejected alternative |
|---|----------|--------|-----------|---------------------|
| D21 | Input type | `[]byte` content | Extractors are pure functions. Engine owns I/O. Testable without filesystem. Serializable for plugins. | `io.Reader` — adds complexity, unnecessary for <1MB doc files |
| D22 | Metadata type | `map[string]any` | Schema lives in `.stem`, not Go types. JSON-serializable. | Typed struct — would couple extractor to schema |
| D23 | Body inclusion | `Body string` in Record | Required by `contains` operator (I1). Separating body from metadata is the extractor's core job. | Omit body — breaks I1 operator coverage |
| D24 | Registry lookup | Extension-based + `.stem` override | Covers 100% of current scope (all `.md`). Override for future explicit config. | Name-only — requires `.stem` in every directory |
| D25 | Error handling | Non-fatal `Errors` in Record | Partial extraction > total failure. Matches existing consumer resilience. | Fatal-only — loses recoverable data |
| D26 | Scope ownership | Engine, not extractor | Consistent with all 7 analyzed systems. Extractor should not know about `.stem`. | Extractor filters — violates separation of concerns |
| D27 | Pure-metadata formats | `Body: ""` | A YAML file IS its metadata. No body to separate. | Invent synthetic body — meaningless |
| D28 | Built-in scope | **Only Markdown is built-in** | Other formats (YAML, JSON, TOML) are plugins — never compiled into the binary. Keeps core small and focused. | Compile all formats — bloats binary, couples to formats that have no real consumer today |
| D29 | Serializable contract | `path` + `content` in → `Record` JSON out | Plugin extractors (I2) will cross process boundaries. The contract must work without shared Go types. | Go-only interface — locks out non-Go plugins |

---

## 9. Pain Points Addressed

| Pain point (intent doc §2) | How extractors address it |
|----------------------------|--------------------------|
| 7 independent parsing systems | Single `Extractor` interface replaces all ad-hoc parsing. One implementation, tested. |
| `line.split(':', 1)` in Python | Replaced by `gopkg.in/yaml.v3` with line-by-line fallback. Proper YAML parsing. |
| `grep -q "pattern"` in Bash | Replaced by structured `Record.Frontmatter["tipo"]`. No substring matching on raw text. |
| LLM-based validation (non-deterministic) | Extractor produces deterministic Record. Validation is separate, deterministic, testable. |
| 4 different glob patterns | Engine scanner replaces all globs. Extractor not involved in discovery. |
| Silent failures (empty output) | Typed extraction + schema validation catches bad data. `ExtractionError` reports issues. |

---

## 10. What This Investigation Does NOT Cover

| Topic | Where | Why deferred |
|-------|-------|-------------|
| Plugin mechanism (WASM, external process, IPC) | I2 | I7 defines the contract. I2 defines how plugins implement it. |
| YAML/JSON/TOML extractors | Plugins (post-I2) | No real consumer needs them today. Interface is ready. |
| Inline field extraction (Dataview-style `Key:: Value`) | Future | Not in any existing consumer. |
| Link extraction from body | D9 (planned) | Links are a separate pipeline stage, not extraction. |
| Derivation functions | I3 | Extractor produces raw data. Derivation is downstream. |
| Body parsing (headings, sections, code blocks) | Future | Body is opaque string for `contains` search. |

---

## 11. Summary

| Aspect | Result |
|--------|--------|
| Interface | `Extractor` — 3 methods: `Extract`, `Extensions`, `Name` |
| Record | `Record{Path, Type, Frontmatter map[string]any, Body, Errors}` — fully JSON-serializable |
| Registry | Extension-based auto-detection + optional `.stem` override |
| Built-in | `MarkdownExtractor` only — YAML frontmatter via `gopkg.in/yaml.v3` + fallback |
| Plugins | All non-Markdown formats are plugins (D28). Contract is serializable (D29). Mechanism in I2. |
| Scope | Engine responsibility, not extractor (D26) |
| Body | Included — required by `contains` operator from I1 (D23) |
| Separation | Extraction produces raw data. Validation/derivation/query are downstream. |
| Coverage | Replaces 100% of the 9 existing ad-hoc parsing implementations |
