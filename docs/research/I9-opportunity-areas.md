# I9: Áreas de Oportunidad — Exploración Creativa

**Date**: 2026-02-18
**Status**: Done
**Method**: Análisis exhaustivo del codebase actual (~2,870 líneas, 11 test files), documentación de diseño (v0-rootline.md), investigaciones previas (I1, I5, I7), y roadmap de epics. Exploración divergente de oportunidades por dimensión.

---

## 1. Hallazgo Central

La primitiva `.stem` con herencia padre→hijo es **genuinamente novel**. Ninguna herramienta existente combina:

1. Frontmatter indexing
2. Reglas por directorio con herencia jerárquica
3. Validación determinística con source tracking
4. State derivation (reservado)

Esto abre territorio virgen en múltiples dimensiones.

---

## 2. El `.stem` como Producto Independiente

### 2.1 Schema Registry Publicable

Hoy cada proyecto define sus `.stem` desde cero. Oportunidad: un registro de schemas compartibles.

```bash
rootline schema pull pablontiv/epic-tracking
# descarga .stem files para estructura Epic→Feature→Story→Task
```

- Schemas versionados y componibles (como helm charts pero para estructura documental)
- `rootline schema init --from template/zettelkasten` para arrancar un knowledge base
- Namespace por comunidad: `rootline schema pull obsidian-community/daily-notes`

### 2.2 `.stem` como Contrato entre Humanos y Máquinas

El `.stem` no solo valida — **define el contrato** de lo que un directorio contiene. Esto es valioso para:

- **Onboarding**: un developer nuevo lee `.stem` y entiende qué documentos existen y qué campos tienen
- **AI Agents**: un LLM lee `.stem` y sabe exactamente cómo crear/editar documentos válidos
- **CI/CD**: el pipeline valida que los PRs no rompan el contrato documental
- **Generación**: `rootline scaffold docs/prd/` crea un archivo con todos los campos required pre-poblados

---

## 3. Integración con el Ecosistema de Desarrollo

### 3.1 Language Server Protocol (LSP)

Un LSP para `.stem`-aware editing sería transformador:

- **Autocompletado** de valores enum en frontmatter (`Estado: ` → sugiere `Pending`, `Completado`...)
- **Diagnósticos en tiempo real** (subrayado rojo si falta campo required)
- **Hover info**: pasar el mouse sobre un campo muestra de qué `.stem` hereda y su schema
- **Go to definition**: click en un campo → navega al `.stem` que lo define
- **Code actions**: "Add missing required field" como quick fix

Esto convertiría a Rootline de CLI tool a **experiencia de authoring**.

### 3.2 Git Hooks Nativos

```bash
rootline hooks install  # configura pre-commit
```

- **Pre-commit**: `rootline validate --staged` solo valida archivos en staging
- **Pre-push**: validación batch completa
- **Commit-msg**: verifica que el mensaje siga un .stem de commits (meta-meta)

### 3.3 GitHub Actions / CI First-Class

```yaml
# .github/workflows/docs.yml
- uses: pablontiv/rootline-action@v1
  with:
    command: validate --all
    fail-on: error  # o warning para soft enforcement
```

- Annotaciones inline en PRs (como ESLint en GitHub Actions)
- Bot comment con summary de validación
- Status check bloqueante para documentación

---

## 4. Más Allá de la Validación: Generación y Migración

### 4.1 Scaffolding Inteligente

```bash
rootline new docs/prd/nueva-feature.md
# Lee el .stem efectivo, genera:
# ---
# Estado: Pending
# Tipo:
# Prioridad:
# ---
# con todos los required fields y defaults aplicados
```

- Templates derivados del schema (no archivos template separados)
- Valores default del `.stem` ya poblados
- Campos enum con comentarios inline mostrando valores válidos

### 4.2 Migraciones de Schema

Cuando un `.stem` cambia, los documentos existentes pueden quedar inválidos:

```bash
rootline migrate --dry-run docs/prd/
# Shows:
#   12 files need migration
#   - "Estado" renamed to "Status" in 12 files
#   - "Prioridad" default changed: null → "Media" (affects 5 files)

rootline migrate --apply docs/prd/
```

- Detección automática de breaking changes entre versiones de `.stem`
- Migraciones reversibles (genera `.stem-migration` log)
- `rootline diff --schema` para ver cómo cambió el schema efectivo

### 4.3 Auto-fix

```bash
rootline fix docs/prd/feature.md
# Adds missing required fields with defaults
# Fixes enum values to closest valid match
# Sorts frontmatter fields to match schema order
```

---

## 5. AI/LLM como Ciudadano de Primera Clase

### 5.1 MCP Server Expandido

Más allá del wrapper 1:1 de CLI commands (ya planeado en I8):

- **Tool: `rootline_create`** — el LLM pide crear un documento y Rootline garantiza que sea válido antes de escribirlo
- **Tool: `rootline_suggest`** — dado un documento parcial, sugiere campos faltantes con valores válidos
- **Resource: schema context** — expone el schema efectivo como contexto para que el LLM sepa las restricciones *antes* de generar

### 5.2 Rootline como Knowledge Base Estructurado para RAG

El combo query + schema + hierarchy es perfecto para RAG:

```
User: "¿Cuáles son los servicios docker pendientes?"
→ rootline query --where 'tipo eq servicio-docker' --where 'estado eq Pending'
→ Returns structured results with metadata
→ LLM synthesizes answer from structured data, not raw file grep
```

- Rootline reemplaza el "grep + hope" que usan la mayoría de RAG pipelines sobre docs
- El schema le dice al LLM qué campos existen y qué significan
- La jerarquía le da contexto (este task pertenece a este feature, que pertenece a este epic)

### 5.3 `.stem` como Prompt Engineering Implícito

Un `.stem` file es, en esencia, un prompt estructurado:

```yaml
schema:
  Estado:
    type: enum
    values: [Pending, In Progress, Completado]
    required: true
```

Esto le dice a un LLM: "cuando crees un documento aquí, DEBES incluir Estado y DEBE ser uno de estos tres valores." Es **constraint-based prompt engineering** — más confiable que instrucciones en lenguaje natural.

### 5.4 Plugin de Claude Code

Un skill de Claude Code que use Rootline como backend:

```
/validate          → rootline validate en el archivo actual
/describe          → muestra schema efectivo del directorio
/new-doc prd       → scaffold interactivo con rootline new
```

---

## 6. Oportunidades Arquitectónicas

### 6.1 Watch Mode + Event System

```bash
rootline watch docs/
# Watches filesystem, re-validates on change
# Emits events:
#   - document.created
#   - document.updated
#   - document.invalid
#   - schema.changed
```

- Webhooks configurables por evento
- Integration con notification systems
- Dashboard web en tiempo real (via SSE desde `rootline serve`)

### 6.2 Multi-Repo Federation

```yaml
# .stem
extends: github.com/org/shared-schemas/docs.stem@v2
```

- Schemas compartidos entre repositorios
- Central governance con local override
- Organizaciones que estandarizan estructura documental

### 6.3 Time-Travel y Auditoría

```bash
rootline history docs/prd/feature.md
# Shows schema evolution:
#   2026-01-15: Estado added (source: docs/.stem@abc123)
#   2026-02-01: Prioridad made required (source: docs/prd/.stem@def456)
#   2026-02-18: Current state: 2 errors
```

- Git-aware: correlaciona cambios de `.stem` con commits
- Blame-style tracing: "this field became required in commit X"
- Compliance trail: prueba que un documento fue válido en fecha Y

### 6.4 Graph de Dependencias (links)

El campo `links` en `.stem` está reservado pero no implementado. Oportunidad enorme:

```yaml
# .stem
links:
  blocks:
    target: "../tasks/*.md"
    field: blocked_by
  parent:
    target: "../"
    field: epic
```

```bash
rootline graph docs/epics/
# Genera DOT/mermaid graph de dependencias
# "Feature F01 is blocked by Task T003 which is In Progress"
```

- Detección de ciclos
- Estado derivado por propagación (Feature es "blocked" si algún task bloqueante es "Pending")
- Visualización interactiva

---

## 7. Nuevos Dominios de Aplicación

### 7.1 Static Site Generators

Hugo, Jekyll, Astro — todos usan frontmatter. Rootline puede ser el **linter de contenido**:

```bash
rootline validate content/blog/ --all
# Ensures all blog posts have: title, date, author, tags
# Ensures draft posts don't have publish_date
```

- Plugin para Hugo/Astro que corre `rootline validate` en build
- Schema definitions que reemplazan la documentación informal de "qué campos necesita un post"

### 7.2 Compliance y Governance

Regulaciones (SOC2, GDPR, ISO) requieren documentación estructurada:

```yaml
# compliance/.stem
schema:
  review_date:
    type: string
    required: true
  approved_by:
    type: string
    required: true
  classification:
    type: enum
    values: [public, internal, confidential, restricted]
    required: true
validate:
  - rule: requires
    if: { classification: "restricted" }
    then: { fields: [access_control, encryption_at_rest] }
```

- Auditable, determinístico, versionado en git
- Reemplaza spreadsheets de compliance

### 7.3 API Documentation Enforcement

```yaml
# apis/.stem
schema:
  api_version:
    type: string
    required: true
  deprecation_date:
    type: string
  owner_team:
    type: string
    required: true
```

### 7.4 Monorepo Package Metadata

```
packages/
  .stem          # schema para package metadata
  auth/
    README.md    # frontmatter: owner, status, dependencies
  billing/
    README.md    # validated against .stem
```

---

## 8. Developer Experience (Quick Wins)

### 8.1 `rootline init` Interactivo

```bash
rootline init
# Scans existing files
# Detects common frontmatter patterns
# Suggests .stem schema based on actual data
# "I found 15 .md files. 12 have 'status' field with values: draft, published, archived. Create enum?"
```

- Reverse-engineering de schema desde documentos existentes
- Zero-config onboarding

### 8.2 `rootline doctor`

```bash
rootline doctor
# Checks:
#   ✓ .stem files are valid YAML
#   ✓ No orphan .stem files (directories with .stem but no matching files)
#   ✓ No conflicting scope patterns
#   ✓ Schema inheritance is consistent
#   ⚠ docs/archive/.stem defines "status" but parent already defines it (intentional?)
```

### 8.3 Rich Terminal Output

```bash
rootline validate --all --output table
┌─────────────────────┬────────┬────────┐
│ File                │ Status │ Errors │
├─────────────────────┼────────┼────────┤
│ docs/prd/auth.md    │   ✓    │   0    │
│ docs/prd/billing.md │   ✗    │   2    │
│ docs/prd/search.md  │   ✓    │   0    │
└─────────────────────┴────────┴────────┘
Summary: 2/3 valid (1 invalid)
```

### 8.4 Shell Completions

```bash
rootline completion bash > /etc/bash_completion.d/rootline
rootline completion zsh > ~/.zsh/completions/_rootline
```

---

## 9. Distribución y Comunidad

### 9.1 Homebrew + Binarios Multi-plataforma

```bash
brew install rootline
# o
curl -sSL https://rootline.dev/install.sh | sh
```

- goreleaser ya planeado — ejecutar
- Binarios para Linux/macOS/Windows ARM64+AMD64

### 9.2 Playground Web (WASM)

```
rootline.dev/playground
├── Editor: escribe .stem + .md
├── Output: ve validation results en tiempo real
└── Share: genera URL compartible
```

- Baja la barrera de entrada enormemente
- Recurso educativo (tutorials interactivos)
- Go compila a WASM nativamente

---

## 10. Oportunidades Técnicas Profundas

### 10.1 Expression Language para Derivación

El slot `derive` en `.stem` está reservado (ver I3). La elección de lenguaje es estratégica:

| Opción | Pro | Contra |
|--------|-----|--------|
| **CEL** (Google) | Diseñado para policies, usado en K8s | Go-native |
| **Expr** | Lightweight, embeddable | Menos conocido |
| **Starlark** | Python-like, usado en Bazel | Más pesado |
| **Custom DSL** | Exactamente lo que necesitas | Mantenimiento |

```yaml
derive:
  slug: "slugify(titulo)"
  is_blocked: "any(children, 'estado == Pending')"
  completion_pct: "count(children, 'estado == Completado') / count(children)"
```

### 10.2 Incremental Validation

Para repositorios grandes, validar todo es costoso:

- Content-addressable cache (hash de archivo + hash de .stem chain)
- Solo re-valida si el archivo o algún `.stem` ancestro cambió
- `rootline validate --all --cache` para builds incrementales

### 10.3 Custom Validation Rules

```yaml
# .stem
validate:
  - rule: custom
    script: ./validators/check-links.sh
    field: references
```

- Reglas de validación como scripts externos
- Interface: stdin=record JSON, exit code=pass/fail, stdout=error message
- Composable con las built-in rules

### 10.4 Progressive Strictness (off/warn/error)

Hoy la validación es binaria: pasa o no pasa. Inspirado por mdbase (spec v0.2), agregar severidad configurable por campo o regla:

```yaml
# .stem
schema:
  titulo:
    type: string
    required: true
    severity: error      # default — bloquea CI
  tags:
    type: list
    required: true
    severity: warn       # reporta pero no falla exit code
  deprecated_field:
    severity: off         # ignorado completamente
```

```bash
rootline validate --all
# ✗ doc.md: titulo is required (error)
# ⚠ doc.md: tags is required (warning)
# Exit code 1 (solo errors cuentan)

rootline validate --all --strict
# Exit code 1 si hay errors O warnings
```

Esto es **clave para adopción incremental**:
- Equipos migran gradualmente: primero `warn` en todo, luego `error` campo por campo
- `rootline init` puede inferir schema existente con `severity: warn` por default
- CI puede usar `--strict` mientras desarrollo local es permisivo
- Compatible con el modelo de herencia: un hijo puede tighten severity (`warn` → `error`) pero no loosear (`error` → `warn`)

---

## 11. La Visión Más Ambiciosa: Documentation as Code

Rootline puede ser el **terraform de la documentación**:

```
terraform   : infraestructura :: rootline : documentación
```

| Terraform | Rootline |
|-----------|----------|
| `.tf` files definen infraestructura | `.stem` files definen estructura |
| `terraform plan` muestra cambios | `rootline diff --schema` muestra cambios |
| `terraform apply` ejecuta | `rootline migrate --apply` ejecuta |
| `terraform validate` verifica | `rootline validate` verifica |
| Provider plugins | Extractor plugins |
| State file | Derived state (computed, never stored) |
| Module registry | Schema registry |

Diferencia clave: Rootline **nunca modifica los archivos fuente** — el estado derivado es efímero, computado al vuelo. Más limpio que terraform state.

---

## 12. Priorización Sugerida

### Impacto Inmediato (ya casi listo)

1. Completar los 5 comandos CLI stub (query, describe, tree, stats, explain)
2. `--output table` con formato rico
3. Shell completions

### Impacto Alto, Esfuerzo Medio

4. MCP Server (abre el mundo AI)
5. `rootline init` (baja barrera de entrada)
6. `rootline new` / scaffold (utilidad diaria)
7. GitHub Action para CI

### Impacto Transformador, Esfuerzo Alto

8. LSP server (experiencia de authoring)
9. Graph/links (state derivation)
10. Expression language para `derive`
11. Schema registry

### Moonshots

12. WASM playground
13. Multi-repo federation
14. Documentation-as-Code ecosystem
