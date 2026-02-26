# Kedral — Roadmap de Implementacion

Roadmap simplificado para KEDB v1. Cambio principal vs diseño original: Kedral ya no mantiene su propio indice FTS5/SQLite. En su lugar, Kedral es un **orquestador liviano** que delega busqueda a rootline (Tier 1, structured search) y backscroll (Tier 2, full-text search).

## 1. Estructura Jerarquica

```
E01: Kedral — Known Error Database Reactiva
│   Intent: CLI Go que orquesta busqueda cross-proyecto de known errors.
│   Tier 1 via rootline query, Tier 2 via backscroll search.
│   Lifecycle: reconcile, retire, report.
│   Pure Go, ~5MB, zero deps externas propias.
│
├── F01: Search Orchestrator
│   │   Milestone: kedral search retorna matches de Rootline y Backscroll
│   │
│   ├── S001: Core
│   │   ├── T001: go-module-and-init
│   │   └── T002: query-normalization
│   │
│   └── S002: Search
│       └── T001: tiered-search-orchestrator
│
├── F02: Lifecycle Management
│   │   Milestone: kedral reconcile/retire mantienen la KEDB
│   │
│   ├── S001: Reconciliation
│   │   └── T001: reconcile-from-backscroll
│   │
│   └── S002: Retirement
│       └── T001: staleness-detection
│
├── F03: CLI Assembly & Deployment
│   │   Milestone: kedral instalado y funcional end-to-end
│   │
│   ├── S001: CLI & Reporting
│   │   ├── T001: cli-entrypoint-and-report
│   │   └── T002: unit-tests
│   │
│   └── S002: Deployment
│       └── T001: build-and-install
│
└── F04: Claude Code Integration
    │   Milestone: hooks activos, KEDB responde en sesiones
    │
    └── S001: Hooks
        ├── T001: user-prompt-submit-hook
        └── T002: stop-hook-ke-creation
```

## 2. Tasks Detallados

| # | Task | Tipo | Criterio de Aceptacion |
|---|------|------|------------------------|
| F01/S001/T001 | go-module-and-init | software-module | go.mod (importa rootline como library), `kedral init` crea /opt/kedral/ + .stem + state.json. Sin SQLite. |
| F01/S001/T002 | query-normalization | software-module | Regex normalizer (IPs, UUIDs, line numbers, timestamps, hex addrs). Unit tests para cada regex. |
| F01/S002/T001 | tiered-search-orchestrator | software-module | Tier 1 (rootline query, Go library import) + Tier 2 (backscroll search, CLI invocation). Output compacto. State file update. |
| F02/S001/T001 | reconcile-from-backscroll | software-module | Lee KEs via rootline, busca nuevas sessions en Backscroll DB (read-only), actualiza frontmatter + git commit |
| F02/S002/T001 | staleness-detection | software-module | Detecta KEs sin recurrencia 6+ meses, componentes obsoletos |
| F03/S001/T001 | cli-entrypoint-and-report | software-module | Arg dispatch (search/reconcile/retire/report/init), report output |
| F03/S001/T002 | unit-tests | software-test | go test ./... pasa; coverage >=70% en normalization, search orchestrator |
| F03/S002/T001 | build-and-install | ci-cd | Makefile, go build, /usr/local/bin/kedral, kedral --version |
| F04/S001/T001 | user-prompt-submit-hook | claude-hook | Hook busca KEDB en cada prompt con heuristica de error detection |
| F04/S001/T002 | stop-hook-ke-creation | claude-hook | Hook prompt-based evalua sesion, crea KE si recurrencia confirmada |

## 3. Cadena de Dependencias

```
F01/S001/T001 go-module-and-init
  → F01/S001/T002 query-normalization
    → F01/S002/T001 tiered-search-orchestrator
      → F02/S001/T001 reconcile-from-backscroll
        → F02/S002/T001 staleness-detection
          → F03/S001/T001 cli-entrypoint-and-report
            → F03/S001/T002 unit-tests
              → F03/S002/T001 build-and-install
                → F04/S001/T001 user-prompt-submit-hook
                  → F04/S001/T002 stop-hook-ke-creation
```

## 4. Cambios vs Diseño Original

| Aspecto | Diseño original | Diseño refinado |
|---------|----------------|-----------------|
| FTS5 index | kedb mantiene SQLite FTS5 propio | Eliminado — Backscroll ya tiene FTS5 |
| SQLite schema | ke_files, ke_sections, ke_fts | Eliminado — sin SQLite en kedral |
| Sync | `kedb sync` mtime-based | Eliminado — lee filesystem directamente |
| Search Tier 1 | FTS5 sobre ke_fts | rootline query (structured, Go library) |
| Search Tier 2 | FTS5 sobre Backscroll messages_fts | backscroll search (CLI invocation) |
| Binario | ~10MB (incluye modernc.org/sqlite) | ~5MB (sin SQLite) |
| State | SQLite DB | JSON state file (~1KB) |
| MCP | Potencial kedb MCP server | Innecesario — rootline serve ya cubre /opt/kedral/ |

Tasks eliminados vs original:

- `ke-parser-and-sync` — no hay sync, kedral lee filesystem directamente via rootline
- `fts5-tiered-search` — reemplazado por `tiered-search-orchestrator` (orquesta busqueda delegada, no mantiene indice propio)
- `go-module-sqlite-schema-and-init` — simplificado a `go-module-and-init` (sin SQLite, state en JSON)

## 5. Futuro (v2+)

### v2: LLM Embebida para Analisis Batch

Agregar capacidad de analisis batch con LLM local para escenarios donde Claude Code no esta en el loop:

| Componente | Opcion | Justificacion |
|-----------|--------|---------------|
| Embeddings | ONNX Runtime via `onnxruntime-purego` | Sin CGO, in-process, 23MB modelo (all-MiniLM-L6-v2) |
| LLM inference | yzma (purego/FFI a llama.cpp) | Sin CGO, carga cualquier GGUF, in-process |
| Modelo LLM | Qwen2.5-3B Q4_K_M (2.4GB) | Mejor structured output en su rango de tamaño |

Habilitaria:

- `kedral analyze` — scan batch de Backscroll DB, detectar clusters de errores recurrentes via embedding similarity, sintetizar KE drafts con LLM
- Template mining estilo Drain para auto-detectar patrones de error
- Semantic search (embeddings) como complemento a FTS5 lexical

**Por que no en v1**: Claude Code ya esta en el loop y es mejor LLM que cualquier 3B local. El valor de kedral v1 esta en el matching rapido y la orquestacion de busqueda, no en la generacion de texto. Agregar LLM embebida suma 2.5GB de deps y complejidad de distribucion sin beneficio claro para el flujo reactivo.

### v2+ Features adicionales

| Feature | Complejidad | Valor |
|---------|-------------|-------|
| LLM embebida (analyze batch) | Alta | Deteccion proactiva sin Claude Code |
| Embeddings (semantic search) | Media | Match por sinonimos/parafraseo |
| Template mining (Drain) | Media | Auto-detectar patrones de error en sessions |
| Backscroll as Go library import | Media | Import directo en vez de CLI invocation para Tier 2 — elimina process spawn, ~5ms savings |
| Auto-link a git commits/PRs | Media | Detectar cuando un PR menciona componente de KE |
| Export KE a Confluence/Notion/Wiki | Media | Publicacion a sistemas externos |
| Multi-user (merge conflicts) | Alta | Para equipos (no MVP) |
| Metricas: % sesiones resueltas via KE | Media | Medir impacto real |
| Character n-gram TF-IDF re-ranker | Baja | Mejorar matching de errores parciales/truncados |
| BM25 field boosting (titulo^5, sintomas^3) | Baja | Mejorar precision de ranking |
