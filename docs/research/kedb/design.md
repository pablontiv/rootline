# KEDB — Diseno y Arquitectura

**Fecha**: 2026-02-25
**Tipo**: Design
**Contexto**: Arquitectura de kedb como CLI de orquestacion ligero. kedb no mantiene index propio — delega Tier 1 (structured query) a Rootline y Tier 2 (full-text search) a Backscroll. Claude Code es el LLM; kedb es infraestructura de lifecycle.

---

## 1. Flujo Operativo

### 1.1 Ciclo de vida de un Known Error

```
Ocurrencia 1              Ocurrencia 2                    KEDB
──────────────            ──────────────                  ──────
Error X aparece           Error X aparece otra vez
Usuario lo resuelve       Hook: kedb search "$PROMPT"
Session guardada            │
Backscroll indexa           ├── Tier 1: rootline query /opt/kedb → no match
                            └── Tier 2: backscroll search → match!
                                  Inyecta snippet de sesion 1

                          Claude Code resuelve con
                          contexto de sesion anterior

                          Hook Stop: ¿error resuelto +
                          match previo sin KE?
                            → Si: Claude Code escribe ────▶ KE-XXXX creado
                              KE en /opt/kedb/               estado: active
                              (tiene todo el contexto)        recurrencia: 2


Ocurrencia 3+
──────────────
Error X aparece
Hook: kedb search "$PROMPT"
  └── Tier 1: rootline query → KE-XXXX match!
      Inyecta workaround directo
      + intentos que NO funcionan

Claude Code resuelve instantaneamente
Hook Stop: actualiza KE-XXXX
  → recurrencia++
  → ultima_revision = hoy
  → agrega session slug a sesiones_origen
```

Tres dominios participan en el ciclo:

1. **Claude Code** — el LLM en sesion. Resuelve errores, crea KEs (tiene todo el contexto), actualiza KEs existentes.
2. **kedb CLI** — orquestador. Ejecuta search (Tier 1 + Tier 2), gestiona lifecycle (reconcile, retire).
3. **Backscroll + Rootline** — backends de datos. Backscroll indexa sessions (ya lo hace). Rootline estructura y valida KEs (ya lo hace).

### 1.2 Quien hace que

| Accion | Quien | Cuando |
|--------|-------|--------|
| Indexar sessions + plans | Backscroll | Ya lo hace (existente) |
| Buscar KEs estructurados (Tier 1) | kedb CLI → rootline query | Hook UserPromptSubmit |
| Buscar en sessions (Tier 2) | kedb CLI → backscroll search | Hook UserPromptSubmit, si Tier 1 < 1 resultado |
| Inyectar contexto | Hook Claude Code | Pre-respuesta, <12ms |
| Resolver el error | Claude Code (el LLM) | Durante la sesion |
| Crear KE nuevo | Claude Code (escribe .md) | Hook Stop, si recurrencia confirmada |
| Actualizar recurrencia KEs | kedb reconcile | On-demand o cron |
| Detectar staleness | kedb retire | On-demand o cron |
| Validar schema | Rootline | Post-escritura, CI |

**Insight clave**: Claude Code ya es un LLM de alta calidad ejecutandose en contexto. kedb es infraestructura de orquestacion y lifecycle — la inteligencia es Claude Code.

## 2. Arquitectura

### 2.1 Tres dominios

```
┌─────────────────────────────────────────────────────────────────┐
│                        Claude Code                               │
│                                                                  │
│  Hook UserPromptSubmit ──▶ kedb search "$PROMPT"                │
│                              ├── Tier 1: rootline query         │
│                              └── Tier 2: backscroll search      │
│                                                                  │
│  Hook Stop ──▶ ¿recurrencia confirmada? ──▶ escribe KE .md     │
│                                                                  │
└──────────┬──────────────────┬──────────────────┬────────────────┘
           │                  │                  │
┌──────────▼─────────┐ ┌─────▼──────────┐ ┌────▼───────────────┐
│ kedb CLI (Go)      │ │ Rootline       │ │ Backscroll         │
│ ~5MB, pure Go      │ │ (ya existe)    │ │ (ya existe)        │
│ lifecycle manager  │ │                │ │                    │
│                    │ │ /opt/kedb/     │ │ Sessions JSONL     │
│ search (orquesta)  │ │ .stem schema   │ │ Plans .md          │
│ reconcile          │ │ KE-XXXX.md     │ │ SQLite FTS5        │
│ retire             │ │ rootline query │ │ backscroll search  │
│ report             │ │ rootline valid │ │                    │
│ init               │ │                │ │                    │
│                    │ │                │ │                    │
│ State file (JSON)  │ │                │ │                    │
│ ~/.kedb/state.json │ │                │ │                    │
└────────────────────┘ └────────────────┘ └────────────────────┘
```

### 2.2 Flujo de datos

```
Sessions JSONL ──▶ Backscroll (indexa) ──▶ Backscroll SQLite FTS5
Plans .md ─────▶ Backscroll (indexa) ──▶       │
                                                │
                                                │ (read-only, Tier 2)
                                                ▼
kedb search "$QUERY" ──────────────────────────┘
         │
         ├── Tier 1: rootline query /opt/kedb
         │   --where "estado != 'resolved'"
         │   + field matching contra keywords del prompt
         │   → Si match: return KE con workaround
         │
         └── Tier 2: backscroll search "$QUERY"
             → Si match recurrente sin KE: return snippets
             → Flag: "candidato a KE si se resuelve"
```

### 2.3 Modos CLI

| Modo | Comando | Descripcion | Latencia |
|------|---------|-------------|----------|
| Search | `kedb search QUERY` | Orquesta rootline query (Tier 1) + backscroll search (Tier 2) | <12ms |
| Reconcile | `kedb reconcile` | Lee Backscroll DB → actualiza KEs → git commit | <3s |
| Retire | `kedb retire` | Detecta KEs stale desde Backscroll DB | <3s |
| Report | `kedb report` | Dashboard: rootline query + backscroll stats | <1s |
| Init | `kedb init` | Crea /opt/kedb/ con .stem, git init | <1s |

**Nota**: No hay `kedb sync`. kedb no mantiene index propio — lee directamente del filesystem (via rootline) y de Backscroll DB (read-only).

## 3. Decisiones Tecnicas

### 3.1 Stack

| Componente | Eleccion | Razon |
|-----------|----------|-------|
| Lenguaje | Go | Consistente con Backscroll y Rootline; binario estatico |
| Tier 1 matching | rootline query (Go library import) | Query estructurado sobre frontmatter, ya existe |
| Tier 2 matching | backscroll search (CLI o Go library) | FTS5/BM25 sobre sessions + plans, ya existe |
| Normalizacion | Regex (patron Sentry) | Deterministico, <1ms |
| Storage KE | Rootline filesystem (/opt/kedb/) | .stem schema, validacion, rootline query |
| State | JSON file (~/.kedb/state.json) | Recurrence counters, timestamps |
| Creacion KE | Claude Code (el LLM en sesion) | Mejor calidad que cualquier modelo local |
| Validacion | Rootline CLI | Ya existe |
| Distribucion | `go build` → binario estatico | ~5MB, zero deps externas |

**Cambio clave vs diseno original**: kedb no mantiene SQLite/FTS5 propio. Rootline query resuelve Tier 1 (structured). Backscroll FTS5 resuelve Tier 2 (full-text). kedb orquesta ambos.

### 3.2 FTS5 como motor de matching (Backscroll)

FTS5 es responsabilidad de Backscroll, no de kedb. Se documenta aqui porque la eleccion impacta la efectividad del Tier 2 search.

| | FTS5/BM25 | Embeddings (MiniLM) |
|--|-----------|---------------------|
| Latencia | <1ms | ~5ms/doc + model load |
| Dependencias | modernc.org/sqlite (pure Go) | ONNX Runtime (.so nativa) |
| Binario | ~10MB | ~10MB + 23MB modelo |
| Cross-compile | Trivial | Complejo (native lib per-platform) |
| Efectividad en tokens tecnicos | **Superior** (tokens exactos) | Inferior (tokens raros = ruido) |
| Efectividad semantica | Inferior (solo lexica) | Superior (sinonimos, parafraseo) |
| Mantenimiento | Zero | Version de modelo, compatibilidad ONNX |

**Dato**: BM25 supera dense retrieval en tokens tecnicos exactos (BEIR benchmark). Los errores tecnicos tienen vocabulario repetitivo y especifico: `encryption_key_fingerprint`, `bpg/proxmox`, `tofu untaint`. FTS5 matchea esto con precision superior a embeddings.

**Dato**: Character n-gram TF-IDF (sin ML) alcanza 92.4% micro F1 en clasificacion de eventos (ACL 2020). Para el dominio de errores tecnicos, la similaridad lexica es suficiente.

**Eleccion: FTS5.** Embeddings son overkill para este dominio y agregan complejidad de distribucion innecesaria. Si se necesitan en el futuro, se agregan como capa adicional sin cambiar la arquitectura.

### 3.3 Normalizacion pre-search (patron Sentry)

kedb normaliza el query antes de pasarlo a backscroll search y antes de comparar contra campos de rootline query.

Antes de buscar, normalizar query y documentos:

```go
var (
    ipRegex        = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(:\d+)?`)
    uuidRegex      = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
    lineNumRegex   = regexp.MustCompile(`(line |:)\d+`)
    timestampRegex = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`)
    hexAddrRegex   = regexp.MustCompile(`0x[0-9a-f]+`)
    pathNumRegex   = regexp.MustCompile(`/\d+/`)
)

func Normalize(text string) string {
    text = ipRegex.ReplaceAllString(text, "<IP>")
    text = uuidRegex.ReplaceAllString(text, "<UUID>")
    text = lineNumRegex.ReplaceAllString(text, "<LINE>")
    text = timestampRegex.ReplaceAllString(text, "<TS>")
    text = hexAddrRegex.ReplaceAllString(text, "<ADDR>")
    text = pathNumRegex.ReplaceAllString(text, "/<N>/")
    return text
}
```

Esto mejora matching: `"error at 192.168.1.1:5432"` y `"error at 10.0.0.5:5432"` normalizan a `"error at <IP>"` y matchean el mismo KE.

### 3.4 State file

```json
// ~/.kedb/state.json
{
  "version": 1,
  "last_reconcile": "2026-02-25T14:30:00Z",
  "last_retire": "2026-02-20T10:00:00Z",
  "recurrence": {
    "encryption_key_fingerprint bpg/proxmox": {
      "count": 3,
      "last_seen": "2026-02-25",
      "ke_id": "KE-0001"
    },
    "cloud-init network config proxmox": {
      "count": 2,
      "last_seen": "2026-02-15",
      "ke_id": null
    }
  }
}
```

Lightweight: JSON file, ~1KB for 50 entries. Read/write atomically (write to tmp + rename). No SQLite needed for this volume.

### 3.5 Formato de output (para inyeccion en hook)

Compacto, optimizado para consumo por Claude Code:

```
# Si hay KE formal (Tier 1):

⚠ Known Error KE-0001 | medium | workaround disponible
  bpg/proxmox encryption_key_fingerprint unknown after apply
  Workaround: tofu untaint despues del error. El recurso SI se crea.
  NO funciona: tofu taint + re-apply (recrea recurso, downtime)
  Recurrencia: 3 | Ultima: 2026-02-20

# Si no hay KE pero hay sesiones previas (Tier 2):

⚠ Sin KE formal, pero hay sesiones previas relevantes:
  [SESSION] copper-running-kitten (2026-02-05):
    ...tofu untaint despues del error. El recurso >>>SI<<< se crea...
  [SESSION] binary-waddling-fox (2026-02-10):
    ...mismo issue con >>>encryption_key_fingerprint<<<...
  Si se resuelve este caso, se creara KE automaticamente.
```

## 4. Schema KEDB (.stem)

### 4.1 Rootline .stem para /opt/kedb/

```yaml
fields:
  id:
    type: string
    required: true
  titulo:
    type: string
    required: true
  estado:
    type: string
    required: true
    enum: [active, workaround_available, pending_fix, resolved, accepted_risk]
  severidad:
    type: string
    required: true
    enum: [critical, high, medium, low]
  tipo:
    type: string
    required: true
    enum:
      - bug_externo
      - error_procedimiento
      - limitacion_plataforma
      - race_condition
      - anti_patron
      - error_diseno
  componentes:
    type: array
    required: true
    non_empty: true
  proyectos:
    type: array
    required: true
  sesiones_origen:
    type: array
  recurrencia:
    type: number
  fecha_creacion:
    type: string
    required: true
  ultima_revision:
    type: string
    required: true
  owner:
    type: string
    required: true
  regla_derivada:
    type: string
  workaround_disponible:
    type: boolean

sequence:
  prefix: "KE"
  digits: 4
```

**Nota**: No hay estado `draft`. Los KEs se crean solo cuando la recurrencia esta confirmada (2+ ocurrencias + workaround validado). Nacen como `active` o `workaround_available`.

### 4.2 Secciones del body

```
## Sintomas
## Intentos fallidos
## Causa raiz
## Workaround
## Regla derivada
## Plan de resolucion
## Referencias
```

### 4.3 Ejemplo de KE

```markdown
---
id: KE-0001
titulo: bpg/proxmox encryption_key_fingerprint unknown after apply
estado: workaround_available
severidad: medium
tipo: bug_externo
componentes:
  - bpg/proxmox
  - proxmox_virtual_environment_storage_pbs
proyectos:
  - homeserver
sesiones_origen:
  - copper-running-kitten
  - binary-waddling-fox
recurrencia: 2
fecha_creacion: "2026-02-05"
ultima_revision: "2026-02-25"
owner: pablo
workaround_disponible: true
---

## Sintomas

`tofu apply` del recurso `proxmox_virtual_environment_storage_pbs` reporta
`encryption_key_fingerprint: (known after apply)` en cada plan, aunque el
recurso ya existe y no ha cambiado.

## Intentos fallidos

- `tofu taint` + re-apply: recrea el recurso innecesariamente, downtime de backup
- Ignorar el diff: `lifecycle { ignore_changes }` rompe deteccion de cambios reales

## Causa raiz

Bug en el provider bpg/proxmox: el API de PVE no retorna `encryption_key_fingerprint`
en GET, solo en POST. El provider lo ve como "unknown" en cada refresh.

## Workaround

`tofu untaint` despues del error. El recurso SI se crea correctamente — el
fingerprint es cosmetico en el state.

## Regla derivada

Verificar siempre el state real (`pct config`, `qm config`) antes de asumir
que un "known after apply" indica un problema real.

## Plan de resolucion

- Issue upstream: pendiente de reporte en github.com/bpg/terraform-provider-proxmox
- Fix esperado: provider debe leer fingerprint de la respuesta POST y persistir en state

## Referencias

- Sesiones: copper-running-kitten (2026-02-05), binary-waddling-fox (2026-02-10)
- Provider: github.com/bpg/terraform-provider-proxmox
```

## 5. Operaciones Detalladas

### 5.1 `kedb init`

```
1. Crear /opt/kedb/ si no existe
2. git init
3. Escribir .stem (schema KEDB, seccion 4.1)
4. Escribir .stemignore (README.md, LICENSE)
5. Crear ~/.kedb/state.json (state file vacio)
6. git add . && git commit -m "feat: initialize KEDB"
```

No SQLite. No kedb.db. Just filesystem + state file.

### 5.2 `kedb search QUERY`

```
1. Normalizar query (regex, seccion 3.3)

2. Tier 1 — rootline query /opt/kedb:
   rootline query /opt/kedb --where "estado != 'resolved'" --output json
   Filtrar resultados por match de keywords/componentes contra query normalizado
   Top 3 por relevancia
   → Si hay resultados: formatear output compacto (seccion 3.5)

3. Tier 2 — Si Tier 1 < 1 resultado:
   backscroll search "$QUERY_NORMALIZADO"
   Top 5 por bm25()
   Detectar recurrencia: ¿aparece en 2+ sessions distintas?
   → Si hay resultados: formatear con flag de recurrencia

4. Actualizar state file: incrementar recurrence counters

5. Output a stdout (consumido por hook de Claude Code)
```

Latencia total: <12ms (rootline query ~5ms + backscroll search ~5ms + overhead ~2ms).

### 5.3 `kedb reconcile`

```
1. Para cada KE en /opt/kedb/ con estado != resolved:
   a. Extraer componentes y titulo del frontmatter
   b. Buscar en Backscroll DB: FTS5 MATCH componentes + keywords del titulo
   c. Comparar sessions encontradas vs sesiones_origen del KE
   d. Si hay sessions nuevas no registradas:
      - Agregar slugs a sesiones_origen (rootline fix o edit directo)
      - Incrementar recurrencia
      - Actualizar ultima_revision

2. rootline validate /opt/kedb/ (batch)

3. Si hubo cambios: git add + git commit en /opt/kedb/

4. Report:
   - X KEs actualizados con nueva evidencia
   - Y sessions nuevas vinculadas
```

### 5.4 `kedb retire`

```
1. Para cada KE con estado != resolved:
   a. Calcular dias desde ultima_revision
   b. Si > 180 dias sin nueva recurrencia en Backscroll → flag: review_needed
   c. Verificar componentes en proyectos activos:
      - git log --since="6 months ago" en cada proyecto de proyectos[]
      - Si componente no aparece en commits recientes → flag: possibly_obsolete

2. Report:
   - X KEs sin recurrencia en 6+ meses
   - Y KEs con componente posiblemente obsoleto
```

### 5.5 `kedb report`

```
Estado de la KEDB                                    2026-02-25
──────────────────────────────────────────────────────────────
KEs totales:       15
  active:           5
  workaround:       6
  pending_fix:      2
  resolved:         2

Salud:
  Sin revision 6+ meses:    2 (KE-0004, KE-0011)
  Sin owner:                 0
  Recurrencia alta (5+):     3 (KE-0001, KE-0007, KE-0012)

Proyectos:
  homeserver:      10 KEs
  rootline:         3 KEs
  incubadora:       2 KEs

Fuentes:
  KE files:        15 archivos en /opt/kedb/
  Backscroll:      270 sessions (read-only)
  Ultimo reconcile: 2026-02-25 14:30
```

## 6. Integracion con Claude Code

### 6.1 Hook UserPromptSubmit (busqueda reactiva)

El hook intercepta cada prompt del usuario y busca matches en la KEDB:

```bash
#!/bin/bash
# Recibe el prompt del usuario via stdin o arg
QUERY=$(echo "$CLAUDE_USER_PROMPT" | head -c 500)

# Solo buscar si parece un error/problema (heuristica simple)
if echo "$QUERY" | grep -qiE 'error|fail|broke|issue|bug|not work|problema|falla'; then
    RESULTS=$(kedb search "$QUERY" --compact 2>/dev/null)
    if [ -n "$RESULTS" ]; then
        echo "Known errors relevantes encontrados en KEDB:"
        echo "$RESULTS"
        echo ""
        echo "Considera estos workarounds e intentos fallidos antes de troubleshootear."
    fi
fi
```

### 6.2 Hook Stop (creacion automatica de KE)

Hook prompt-based que Claude Code evalua al final de la sesion:

```
Analiza esta sesion:

1. ¿Se resolvio un error tecnico? (no una feature, no una pregunta)
2. Si se resolvio, ejecuta: kedb search "[descripcion del error]"
3. Evalua el resultado:

   a. Si kedb retorno un KE formal (Tier 1 — rootline query match):
      → Ejecuta: kedb reconcile
      → El KE se actualiza automaticamente

   b. Si kedb retorno sesiones previas sin KE (Tier 2 — backscroll search match):
      → Este error es recurrente y ahora esta confirmado
      → Crea un archivo KE-XXXX.md en /opt/kedb/ con:
        - Frontmatter completo (.stem schema)
        - Sintomas de ambas ocurrencias
        - Intentos fallidos de ambas ocurrencias
        - Workaround validado (funciono 2+ veces)
        - sesiones_origen con slugs de todas las sesiones
        - estado: workaround_available (si hay workaround) o active (si no)
      → Ejecuta: rootline validate /opt/kedb/
      → Ejecuta: cd /opt/kedb && git add . && git commit

   c. Si kedb no retorno nada:
      → Primera ocurrencia. No crear KE. Backscroll lo indexara.
```

### 6.3 Rootline MCP Server

`rootline serve /opt/kedb/` expone 8 tools MCP sobre la KEDB. Cualquier agente Claude Code puede hacer queries estructuradas sobre KEs sin necesidad de un MCP server adicional de kedb:

```
query: "estado != 'resolved' && componentes contains 'bpg/proxmox'"
→ KE-0001: bpg/proxmox encryption_key_fingerprint...
```

kedb no necesita MCP propio — Rootline ya provee acceso MCP a /opt/kedb/.

## 7. Dependencias

| Dependencia | Version | Uso |
|------------|---------|-----|
| Go | 1.24+ | Lenguaje |
| rootline | latest | Tier 1 query, validation, sequence (Go library import) |
| backscroll | latest | Tier 2 search (CLI invocation o Go library import) |

**Zero SQLite en kedb. Zero modelos. Zero native libs. Zero servicios externos.**

Binario estatico de ~5MB. `go build` y listo. Mas ligero que el diseno original (~10MB) porque no incluye modernc.org/sqlite.

## 8. Performance Esperado

| Operacion | Tiempo |
|-----------|--------|
| `kedb search` (Tier 1 + Tier 2) | <12ms |
| `kedb reconcile` (15 KEs vs Backscroll) | <3s |
| `kedb retire` (15 KEs, git log check) | <3s |
| `kedb report` | <1s |
| `kedb init` | <1s |

Uso de recursos:

| Componente | RAM | Disco |
|-----------|-----|-------|
| kedb CLI (ejecucion) | ~10MB | ~5MB (binario) |
| state.json (50 entries) | — | <1KB |
| /opt/kedb/ (50 KEs) | — | <500KB |

**RAM cuando no se ejecuta: 0.** CLI on-demand, no daemon.

## 9. Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigacion |
|--------|-------------|---------|------------|
| Tier 2 no matchea errores con vocabulario diferente | Media | Medio | Normalizacion Sentry-style + BM25 pondera tokens tecnicos |
| Hook genera ruido (falsos positivos) | Media | Medio | Heuristica pre-filtro (solo si prompt contiene error/fail/bug) |
| Claude Code escribe KE con formato incorrecto | Baja | Bajo | rootline validate post-escritura; .stem como guardrail |
| Backscroll no disponible (no instalado) | Media | Medio | Tier 2 se salta gracefully; Tier 1 funciona independiente |
| Rootline API cambia | Baja | Medio | kedb importa rootline como Go library; breaking changes detectados en compilacion |
| Backscroll CLI/DB schema cambia | Baja | Medio | Parseo defensivo; version check |
| KEs se acumulan sin curar | Baja | Medio | `kedb retire` detecta staleness; report muestra metricas de salud |
| State file se corrompe | Muy baja | Bajo | Atomic write (tmp + rename); state es regenerable desde filesystem |
