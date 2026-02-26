# Backscroll - Claude Code Session & Plan Search CLI

**Fecha**: 2026-02-18
**Tipo**: Research
**Contexto**: Investigacion de herramienta CLI para busqueda full-text en sesiones y planes de Claude Code
**Ecosistema**: Backscroll provee Tier 2 search para [Kedral](kedral/README.md) (Known Error Database). Backscroll = event store + busqueda. Kedral = bridge entity + lifecycle. Rootline = structured store + validacion.

---

## 1. Problema

Claude Code almacena conversaciones como archivos JSONL en `~/.claude/projects/` y planes de implementacion como markdown en `~/.claude/plans/`. No existe forma nativa de buscar full-text dentro de ninguno. El `sessions-index.json` que mantiene automaticamente solo contiene `summary` + `firstPrompt` — insuficiente para encontrar discusiones especificas, decisiones, codigo o razonamiento tecnico. Los planes no tienen ningun indice.

### Problemas verificados con el skill `/sessions` actual (Python embebido):

| Problema | Evidencia |
|----------|-----------|
| Lento (~5s), sin indice persistente | Parsea ~270 JSONL (~1 GB) en cada invocacion |
| Ruido masivo en output | `<local-command-caveat>`, `<local-command-stdout>` pasan filtros |
| Solo indexa mensajes de usuario | Respuestas de Claude (implementaciones, decisiones, codigo) no se buscan |
| Muestra primeros 3 mensajes, no los que matchean | Imposible saber por que una sesion es relevante |
| No hay forma de leer contenido | Encuentra sesion pero no puede extraer contexto |

## 2. Estado del Arte

### Herramientas especificas Claude Code

| Tool | Lenguaje | Search | Index | Sync | Export | Estado |
|------|----------|--------|-------|------|--------|--------|
| **search-sessions** | Rust/Tantivy | Lexical weighted | JSON index files | No | Snippet | Repo no encontrado en GitHub |
| **cc-sessions** | Rust | Via fzf+jq | Lee sessions-index.json | No | No | Activo, TUI interactivo |
| **cass** | Rust/Tantivy | Hybrid BM25+embeddings | Tantivy segments | No | Snippet | Alpha, 11+ agents |
| **claude-code-sync** | Rust | No | Git smart merge | Git bidireccional | Git repo | Activo |
| **claude-code-log** | Python | No | Parser JSONL | No | HTML/Markdown | Activo, 736 stars |
| **claude-JSONL-browser** | TypeScript | No | Browser-side | No | Markdown | Activo |
| **CTK** | Rust/SQLite | SQLite FTS5 | SQLite | No | JSON/MD/HTML | Multi-provider |

### Herramientas analogas (shell history)

| Tool | Lenguaje | Arquitectura | Relevancia |
|------|----------|-------------|------------|
| **atuin** | Rust/SQLite | SQLite FTS5, sync daemon opcional, encrypted | Patron de referencia mas cercano |
| **mcfly** | Rust/SQLite | Neural network ranking, SQLite | Context-aware suggestions |

### Gap identificado

Ninguna herramienta combina las 3 capacidades: **busqueda indexada full-text + output para LLM + recuperacion de contexto de sesion**. Ademas, ninguna indexa plan files.

```
                Search    LLM Output    Context Recovery
search-sessions   ✅         ✅              ❌
cc-sessions       ~          ❌              ❌
cass              ✅         ❌(TUI)         ❌
claude-code-log   ❌         ❌              ❌
CTK               ✅         ❌              ❌
Backscroll        ✅         ✅              ✅  ← el gap
```

## 3. Descubrimiento: sessions-index.json

Claude Code mantiene automaticamente un indice en `~/.claude/projects/*/sessions-index.json`:

```json
{
  "version": 1,
  "entries": [{
    "sessionId": "UUID",
    "fullPath": "/absolute/path/to/session.jsonl",
    "fileMtime": 1768957737046,
    "firstPrompt": "primer mensaje del usuario",
    "summary": "Auto-generated summary",
    "messageCount": 35,
    "created": "2026-01-20T20:57:53.184Z",
    "modified": "2026-01-20T21:12:51.482Z",
    "gitBranch": "master",
    "projectPath": "/opt/homeserver/automation",
    "isSidechain": false
  }]
}
```

**179 entries** en nuestro proyecto. Util para listar sesiones recientes sin parsear JSONL, pero insuficiente para busqueda full-text (solo `summary` + `firstPrompt`).

## 4. Formato JSONL de Sesiones

No existe schema oficial. Campos estables observados:

### Record types

| Type | Contenido | Indexar? |
|------|-----------|---------|
| `user` | Mensajes del usuario | Si |
| `assistant` | Respuestas de Claude (codigo, decisiones, razonamiento) | Si |
| `progress` | Tool execution progress | No |
| `file-history-snapshot` | File change tracking | No |
| `system` | System events | No |
| `summary` | Session summary | No (ya en sessions-index.json) |
| `queue-operation` | Queue operations | No |

### Estructura de mensaje

```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": "string OR [{type: 'text', text: '...'}]"
  },
  "uuid": "message-uuid",
  "slug": "session-slug-human-readable",
  "timestamp": "2026-02-18T08:41:00Z",
  "version": 42
}
```

`content` puede ser string o array de ContentBlocks — parser debe manejar ambos.

### Patrones de ruido verificados

Mensajes que deben filtrarse (verificado en 271 sesiones reales):

- `<local-command-caveat>Caveat:...` — hooks de comandos locales
- `<local-command-stdout></local-command-stdout>` — output vacio de hooks
- `<command...` — command XML tags
- `Caveat:` — prefijo de caveats
- `[Request interrupted` — interrupciones de usuario
- `Base directory for this skill:` — dumps de prompts de skills

## 5. Decisiones Tecnicas

### Stack: Go + SQLite FTS5

| Opcion | Evaluacion |
|--------|-----------|
| **Python stdlib** | Ya disponible, pero requiere runtime, no portable |
| **Go + modernc.org/sqlite** | Binario estatico, FTS5 built-in (pure Go, zero CGO), cross-compile trivial |
| **Rust + Tantivy** | Mas potente pero mas complejo, no aporta valor para este scope |
| **Rust + SQLite** | Similar a Go pero ecosistema menos familiar |

**Eleccion: Go** — binario estatico, zero deps, FTS5 via `modernc.org/sqlite` (verificado: FTS5 disponible en Host Proxmox).

### FTS5 features clave

- `snippet()` — extrae fragmento con match highlighted: `...configurar >>>qemu<<< guest agent...`
- `bm25()` — ranking por relevancia automatico
- Ambos nativos en SQLite FTS5, sin deps externas.

### Index: SQLite vs alternativas

| Opcion | Pros | Contras |
|--------|------|---------|
| **SQLite FTS5** | Nativo, ACID, WAL concurrencia, snippet/bm25 | Solo lexical |
| **Tantivy** | Mas potente, segment-based | Overkill, mas deps |
| **Bleve** | Go native FTS | Index grande, overkill |
| **Flat file** | Simple | Sin persistencia, re-parsea |

**Eleccion: SQLite FTS5** — resuelve exactamente el problema sin overhead.

### Daemon vs CLI on-demand

| | CLI on-demand | Daemon |
|--|--------------|--------|
| Indexacion | Al invocar (incremental <50ms) | Continua (inotify) |
| Recursos | Zero cuando no se usa | RAM permanente |
| Complejidad | Baja | Media (PID, systemd, logging) |

**Eleccion: CLI on-demand** — incremental sync es tan rapido (<50ms para stat 270 files) que no justifica daemon.

## 6. Diseño de Backscroll

### Intencion

Backscroll es una CLI en Go que indexa incrementalmente sesiones y planes de Claude Code en SQLite FTS5, habilitando busqueda full-text instantanea sobre todo el historial conversacional y de planificacion. Binario estatico, zero runtime deps, cualquier usuario de Claude Code puede usarlo.

### Modos

| Modo | Comando | Fuente |
|------|---------|--------|
| List | `backscroll` | sessions-index.json (instantaneo) |
| Search | `backscroll KEYWORD` | SQLite FTS5 con snippet + bm25 (sessions + plans) |
| Search plans | `backscroll KEYWORD --plans` | Solo plan_sections_fts |
| Search sessions | `backscroll KEYWORD --sessions` | Solo messages_fts |
| Read | `backscroll --read ID` | JSONL filtrado o markdown crudo (auto-detecta) |
| Topics | `backscroll --topics` | Analisis de keywords en DB |
| Stats | `backscroll --stats` | Metadata del index (sessions + plans) |

### Arquitectura

```
src/backscroll/
├── main.go       # Entry point, arg parsing, --plans/--sessions/--all flags
├── db.go         # SQLite schema, WAL, sync, FTS5 queries (sessions + plans)
├── parser.go     # JSONL defensive parsing + noise filtering
├── plans.go      # Plan discovery, markdown parsing, section splitting
├── scope.go      # Project detection (git root → project path), plan-project association
├── reader.go     # --read mode: JSONL filtrado o plan markdown (auto-detecta)
├── output.go     # Compact formatter con prefijo [SESSION]/[PLAN]
└── topics.go     # Topic analysis (sessions + plans)
```

### Schema SQLite — Sessions

```sql
PRAGMA journal_mode=WAL;

files (path TEXT PK, mtime REAL, size INT, slug TEXT, session_date TEXT, msg_count INT)
messages (id INT PK, file_path TEXT FK→files, ordinal INT, role TEXT, text TEXT, timestamp TEXT)
messages_fts USING fts5(text, content=messages, content_rowid=id)
-- + triggers INSERT/DELETE para sync FTS
```

### Schema SQLite — Plans

```sql
plans (path TEXT PK, mtime REAL, size INT, title TEXT, plan_date TEXT, section_count INT)
plan_sections (id INT PK, plan_path TEXT FK→plans, ordinal INT, heading TEXT, text TEXT)
plan_sections_fts USING fts5(heading, text, content=plan_sections, content_rowid=id)
-- + triggers INSERT/DELETE para sync FTS
```

Tablas separadas de sessions (no se modifican las existentes). La busqueda unificada usa UNION ALL de ambas FTS5 tables con bm25() para ranking comparable.

### Performance esperado

| Operacion | Tiempo |
|-----------|--------|
| Primera indexacion sessions (~270 files, 1 GB) | ~3-5s |
| Primera indexacion plans (~147 files, 935 KB) | <500ms |
| Indexacion incremental sessions (0-2 files) | <50ms |
| Indexacion incremental plans (0-1 files) | <10ms |
| Busqueda FTS5 unificada | <1ms |
| Stat 270+147 files | ~4ms |

## 7. Plan Files — Formato y Descubrimiento

Claude Code genera plan files en `~/.claude/plans/` durante sesiones con plan mode activo.

### Datos verificados (2026-02-18)

| Dato | Valor |
|------|-------|
| Ubicacion | `~/.claude/plans/*.md` |
| Cantidad | 147 archivos |
| Tamaño total | 935 KB (~23K lineas) |
| Formato | Markdown puro (headers, code blocks, tablas, listas) |
| Naming | Random adjective-verb-noun (ej: `binary-waddling-kitten.md`) |
| Subagent plans | 2 archivos con sufijo `-agent-UUID.md` |
| Metadata index | No existe (a diferencia de sessions-index.json) |

### Estructura tipica

```markdown
# Plan: Titulo del Plan

## Context
Por que se hace este cambio...

## Steps/Tasks
### Step 1: ...
### Step 2: ...

## Verification
Como validar...
```

No todos empiezan con `# Plan:` (~30% tienen titulos diferentes). El parser no debe asumir el prefijo.

### Discovery

Sin equivalente a sessions-index.json. Discovery directa via glob:

```go
func DiscoverPlans(homeDir string) ([]string, error) {
    return filepath.Glob(filepath.Join(homeDir, ".claude", "plans", "*.md"))
}
```

## 8. Parser de Plan Files

### Decision: Split por secciones `##`

| Opcion | Pros | Contras |
|--------|------|---------|
| **File completo como 1 registro** | Simple | `snippet()` inutil en archivos de 958 lineas |
| **Split por `##` headers** | Snippets con contexto de seccion, heading searchable | Mas rows |
| **Split por `###` headers** | Granularidad maxima | ~700+ secciones, fragmenta contexto |

**Eleccion: Split por `##`** — ~6 secciones promedio por archivo = ~900 rows total (trivial para FTS5). Cada seccion tiene un heading que da contexto inmediato en el snippet. Headers `###` quedan como texto dentro de la seccion `##` padre.

### Algoritmo

```go
type PlanSection struct {
    Ordinal int    // 0 = preambulo (antes del primer ##), 1+ = secciones
    Heading string // texto del ## header (sin ##), "" si es preambulo
    Text    string // contenido completo de la seccion
}

type Plan struct {
    Path         string
    Title        string // primera linea # (sin "Plan: " si existe)
    PlanDate     string // extraido de "**Fecha**: YYYY-MM-DD" si existe
    SectionCount int
    Sections     []PlanSection
}
```

Parsing:
1. Iterar lineas del archivo
2. Primera `# ` line → titulo (quitar prefijo `Plan: ` si existe)
3. `**Fecha**: YYYY-MM-DD` → plan_date
4. Cada `## ` → flush seccion actual, abrir nueva con heading
5. Todo lo demas → acumular en seccion actual
6. Fallback titulo: nombre del archivo sin extension

### Diferencias con parser JSONL

| Aspecto | Sessions (JSONL) | Plans (Markdown) |
|---------|------------------|------------------|
| Formato | JSON lines, defensivo | Texto plano, simple |
| Noise filtering | Extensivo (caveats, XML tags, interruptions) | Ninguno (texto limpio) |
| Code blocks | Se indexan | Se indexan (alto valor en planes) |
| Unidad de indexacion | Mensaje individual | Seccion `##` |
| Metadata | Timestamp, role, version | Title, date (opcional) |

## 9. Scoping por Proyecto

### El problema

Sessions tienen scope natural por proyecto — cada `~/.claude/projects/<proyecto>/` contiene sus JSONL. Plans son globales — `~/.claude/plans/` es un directorio flat sin separacion por proyecto.

Backscroll busca por defecto solo en el **proyecto actual** (detectado via cwd). Esto requiere asociar plans a proyectos.

### Deteccion del proyecto actual

1. Leer `.git` en cwd para obtener repo root
2. Buscar en `~/.claude/projects/` el directorio cuyo `sessions-index.json` tiene `projectPath` == repo root
3. El project path es la key para filtrar sessions Y plans

### Asociacion plan→proyecto (dos capas)

| Capa | Metodo | Cobertura verificada | Mecanismo |
|------|--------|---------------------|-----------|
| Primary | Session refs | 47% (71/149) | Regex `\.claude/plans/([\w-]+)\.md` en JSONL |
| Fallback | Content heuristics | +35% (52 orphans) | Markers de proyecto en contenido del plan |
| **Total** | **Combinada** | **82% (123/149)** | |
| Orphaned | Sin contexto | 18% (26/149) | Planes genericos, excluidos por defecto |

**Session refs**: Al indexar sessions, extraer plan filenames referenciados en tool calls (Write/Read). La session ya es proyecto-scoped, por lo que el plan hereda la asociacion.

**Content heuristics**: Para plans sin referencia en sessions, buscar markers del proyecto. Markers auto-detectados del repo (paths, keywords del codebase) o configurables. Ejemplo para homeserver: `homeserver`, `terraform/`, `docs/epics/`, `tofu`, `CLAUDE.md`, `locals-services`.

**Truly orphaned**: Plans genericos sin ningun contexto de proyecto (ej: "organizar commits"). Excluidos del search por defecto, visibles con `--all`.

### Schema: tabla de asociacion

```sql
plan_projects (
    plan_path    TEXT,
    project_path TEXT,
    method       TEXT,  -- 'session_ref' | 'content_heuristic'
    PRIMARY KEY(plan_path, project_path)
)
```

Se puebla durante sync. Un plan puede pertenecer a multiples proyectos (verificado: 0 casos en practica actual).

### CLI con scoping

```bash
backscroll KEYWORD           # sessions + plans del proyecto actual
backscroll KEYWORD --all     # TODOS los proyectos + todos los plans
backscroll KEYWORD --plans   # solo plans del proyecto actual
backscroll KEYWORD --sessions # solo sessions del proyecto actual
```

## 10. Busqueda Unificada Sessions + Plans

### Query SQL con project scoping

```sql
SELECT 'session' AS source, f.slug, f.session_date AS date,
       f.slug AS title, '' AS section_heading,
       snippet(messages_fts, 0, '>>>', '<<<', '...', 10) AS snip,
       bm25(messages_fts) AS score
FROM messages_fts
JOIN messages m ON messages_fts.rowid = m.id
JOIN files f ON m.file_path = f.path
WHERE messages_fts MATCH ?
  AND f.path LIKE ?  -- project session path prefix

UNION ALL

SELECT 'plan' AS source, p.path, COALESCE(p.plan_date, '') AS date,
       p.title, COALESCE(ps.heading, '(preambulo)') AS section_heading,
       snippet(plan_sections_fts, 1, '>>>', '<<<', '...', 10) AS snip,
       bm25(plan_sections_fts) AS score
FROM plan_sections_fts
JOIN plan_sections ps ON plan_sections_fts.rowid = ps.id
JOIN plans p ON ps.plan_path = p.path
WHERE plan_sections_fts MATCH ?
  AND ps.plan_path IN (SELECT plan_path FROM plan_projects WHERE project_path = ?)

ORDER BY score
LIMIT 20;
```

Nota: `bm25()` retorna valores negativos (mas negativo = mas relevante). Ambas fuentes usan la misma convencion, UNION es directamente comparable. Con `--all`, los WHERE de scoping se omiten.

Para plans, snippet usa columna 1 (`text`) porque columna 0 es `heading`. Si el match esta en el heading, se muestra el heading completo.

### Output diferenciado

```
[SESSION] 2026-02-10 · binary-waddling-kitten · E02/F03/S003
  user: ...erradicar todas las referencias a >>>PRD<<< de .claude/...

[PLAN] binary-waddling-kitten · Erradicar PRD > S003 Tasks
  `security-checklist.md`: >>>PRD<<<-296→remove ref, Task295/301→replace...
```

El prefijo `[SESSION]`/`[PLAN]` + heading de seccion dan contexto suficiente para decidir si leer el item completo.

### --read ampliado

```bash
backscroll --read binary-waddling-kitten
```

Auto-detecta si el slug matchea en `plans` o `files`. Para planes, imprime markdown crudo (ya legible). Para sessions, aplica el filtrado de ruido existente.

### --stats con cobertura

```
Project: /opt/homeserver/automation
Sessions: 179 indexed (1.2 GB)
Plans: 123 associated (of 149 total, 82% coverage)
  via session refs: 71
  via content heuristics: 52
  orphaned: 26
Index size: 45 MB
Last sync: 2026-02-18 08:41
```

## 11. Sync Incremental de Plans

### Paso 1: Indexar contenido (mtime-based)

Mismo patron mtime que sessions:

```go
func (db *DB) SyncPlans(paths []string) error {
    for _, path := range paths {
        info, _ := os.Stat(path)
        mtime := float64(info.ModTime().UnixNano()) / 1e9

        var storedMtime float64
        db.conn.QueryRow("SELECT mtime FROM plans WHERE path=?", path).Scan(&storedMtime)

        if mtime <= storedMtime { continue } // sin cambios

        content, _ := os.ReadFile(path)
        plan := ParsePlan(path, content)

        tx := db.conn.Begin()
        tx.Exec("DELETE FROM plan_sections WHERE plan_path=?", path)
        tx.Exec(`INSERT OR REPLACE INTO plans VALUES (?,?,?,?,?,?)`,
            path, mtime, info.Size(), plan.Title, plan.PlanDate, len(plan.Sections))
        for _, s := range plan.Sections {
            tx.Exec(`INSERT INTO plan_sections(plan_path,ordinal,heading,text) VALUES (?,?,?,?)`,
                path, s.Ordinal, s.Heading, s.Text)
        }
        tx.Commit()
    }
    return nil
}
```

DELETE + reinsert para updates: cuando un plan se edita, el trigger FTS5 de DELETE limpia el indice antes de la reinsercion. Identidad del plan = `path`.

### Paso 2: Asociar plans a proyectos

Despues de indexar sessions y plans, construir la tabla `plan_projects`:

1. **Session refs**: Para cada session indexada, extraer plan filenames del contenido JSONL via regex `\.claude/plans/([\w-]+)\.md`. INSERT en `plan_projects` con method='session_ref'.

2. **Content heuristics**: Para plans sin asociacion (orphans), buscar markers del proyecto en el contenido del plan. Markers derivados del `projectPath` del sessions-index.json (repo name, subdirectories). INSERT con method='content_heuristic'.

El paso 2 se ejecuta solo durante primera indexacion o cuando se detectan nuevos plans sin asociacion. Es O(n) sobre orphans, no sobre todos los plans.

## 12. Mantenimiento del Schema JSONL

- No existe schema oficial ni changelog de formato
- Campo `version` (CLI version) sirve como proxy para detectar cambios
- Monitorear: github.com/anthropics/claude-code/releases
- Parseo defensivo: ignorar campos desconocidos, skip con warning
- Referencia comunitaria: github.com/daaain/claude-code-log (736 stars, parser activo)

## 13. Futuro (v2+)

| Feature | Complejidad | Valor |
|---------|-------------|-------|
| `--export markdown/json` | Baja | Export sesiones y plans |
| `--since DATE` | Baja | Filtro por fecha (sessions y plans) |
| `--context N` | Baja | Mensajes alrededor del match |
| `--read-section N` | Baja | Leer seccion especifica de plan grande |
| `--sync remote` | Media | Sync a storage externo |
| Indexar otros artifacts | Baja | `~/.claude/todos/` u otros si aparecen |
| Semantic search | Alta | Embeddings-based |
| Multi-agent | Media | Cursor, Copilot, etc. |
| Custom markers config | Baja | Archivo de config para content heuristics por proyecto |

## 15. Roadmap de Implementacion

### Estructura Jerarquica

```
E03: Backscroll — Session & Plan Search CLI
│   Intent: CLI para busqueda full-text indexada sobre sesiones y planes
│   de Claude Code, reemplazando el skill /sessions actual.
│
├── F01: Core Engine
│   │   Milestone: Binary indexa sessions + plans en SQLite FTS5
│   │
│   ├── S001: Session Indexing Pipeline
│   │   ├── T001: go-module-sessions-core
│   │   └── T002: session-sync-and-scope
│   │
│   └── S002: Plan Indexing Pipeline
│       ├── T001: plan-parser-and-schema
│       └── T002: plan-sync-and-association
│
├── F02: Search & CLI Interface
│   │   Milestone: Binary funcional con todos los modos CLI
│   │
│   ├── S001: Search Engine
│   │   ├── T001: unified-search-and-output
│   │   └── T002: read-mode
│   │
│   └── S002: CLI Assembly & Tests
│       ├── T001: cli-entrypoint-and-topics
│       └── T002: unit-tests
│
└── F03: Deployment & Integration
    │   Milestone: backscroll instalado, /sessions retirado
    │
    └── S001: Ship & Replace
        ├── T001: build-and-install
        └── T002: replace-sessions-skill
```

### Tasks Detallados

| # | Task | Tipo | Scope | Criterio de Aceptacion |
|---|------|------|-------|------------------------|
| F01/S001/T001 | go-module-sessions-core | software-module | go.mod + db.go (sessions schema, WAL, triggers) + parser.go (JSONL parse + 6 noise filters) | `go build` OK; indexa >=100 sessions de `~/.claude/projects/` |
| F01/S001/T002 | session-sync-and-scope | software-module | scope.go (git root, projectPath lookup) + db.go SyncSessions() mtime-based incremental | `--stats` < 100ms en segunda corrida; count correcto por proyecto |
| F01/S002/T001 | plan-parser-and-schema | software-module | db.go (plans schema extension) + plans.go (discovery, `##` split, title/date) | ParsePlan() split correcto en 5 plan files conocidos |
| F01/S002/T002 | plan-sync-and-association | software-module | db.go SyncPlans() + plan_projects (session_ref 47% + content_heuristic 35%) | `--stats` muestra >=82% plan coverage |
| F02/S001/T001 | unified-search-and-output | software-module | FTS5 UNION ALL + bm25 + snippet + output.go `[SESSION]`/`[PLAN]` formatter | `backscroll velero` retorna resultados de ambas fuentes con bm25 ordering |
| F02/S001/T002 | read-mode | software-module | reader.go `--read ID` auto-detect (JSONL filtrado vs markdown crudo) | `--read <slug>` imprime contenido correcto para session y plan |
| F02/S002/T001 | cli-entrypoint-and-topics | software-module | main.go arg dispatch + topics.go + stats + wire all modules | Los 7 modos CLI funcionales end-to-end |
| F02/S002/T002 | unit-tests | software-test | go test: parser.go, plans.go, db.go queries, scope.go | `go test ./... -count=1` pasa; coverage >=70% en parser y plans |
| F03/S001/T001 | build-and-install | ci-cd | Makefile target + go build + /usr/local/bin/ | `which backscroll` retorna path; `backscroll --stats` exit 0 |
| F03/S001/T002 | replace-sessions-skill | operacion-sistema | Rewrite SKILL.md → shell calls a backscroll | `/sessions velero` retorna output backscroll; 0 Python en SKILL.md |

### Cadena de Dependencias

Ejecucion estrictamente secuencial — cada modulo construye sobre el anterior:

```
F01/S001/T001 go-module-sessions-core
  → F01/S001/T002 session-sync-and-scope
    → F01/S002/T001 plan-parser-and-schema
      → F01/S002/T002 plan-sync-and-association
        → F02/S001/T001 unified-search-and-output
          → F02/S001/T002 read-mode
            → F02/S002/T001 cli-entrypoint-and-topics
              → F02/S002/T002 unit-tests
                → F03/S001/T001 build-and-install
                  → F03/S001/T002 replace-sessions-skill
```

### Decisiones de Diseno

| Decision | Eleccion | Razon |
|----------|----------|-------|
| Merge scaffold+parser en T001 | 1 task | parser.go y db.go schema mutuamente dependientes; ni uno verificable sin el otro |
| Merge search+output en T001 F02 | 1 task | No se puede verificar FTS5 sin formatear output |
| Tests como task separado | T002 en S002 | `go test ./...` es criterio universal; separa correctness de wiring |
| `ci-cd` para build | No host-script | Makefile + go build es infra de distribucion, no config de host |
| Cadena lineal | Sin paralelismo | Cada modulo depende del anterior; Features no son independientes |

## 16. Referencias

- [cc-sessions](https://github.com/chronologos/cc-sessions) — Rust session picker
- [cass](https://github.com/Dicklesworthstone/coding_agent_session_search) — Rust hybrid search
- [claude-code-sync](https://github.com/perfectra1n/claude-code-sync) — Rust Git sync
- [claude-code-log](https://github.com/daaain/claude-code-log) — Python export
- [claude-JSONL-browser](https://github.com/withLinda/claude-JSONL-browser) — TypeScript browser
- [CTK](https://github.com/queelius/ctk) — Rust/SQLite multi-provider
- [atuin](https://github.com/atuinsh/atuin) — Rust/SQLite shell history (patron analogo)
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — Pure Go SQLite con FTS5
- [Claude Code session format](https://kentgigger.com/posts/claude-code-conversation-history)
