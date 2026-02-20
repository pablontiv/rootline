# I9: Áreas de Oportunidad — Exploración Creativa

**Date**: 2026-02-18
**Status**: Done
**Podado**: 2026-02-20 — se eliminaron secciones ya implementadas (init, doctor, fix, new, hooks, completions, table, severity)

---

## 1. Hallazgo Central

La primitiva `.stem` con herencia padre→hijo es **genuinamente novel**. Ninguna herramienta existente combina:

1. Frontmatter indexing
2. Reglas por directorio con herencia jerárquica
3. Validación determinística con source tracking
4. State derivation (reservado)

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

---

## 3. Integración con el Ecosistema de Desarrollo

### 3.1 Language Server Protocol (LSP)

Un LSP para `.stem`-aware editing sería transformador:

- **Autocompletado** de valores enum en frontmatter (`Estado: ` → sugiere `Pending`, `Completado`...)
- **Diagnósticos en tiempo real** (subrayado rojo si falta campo required)
- **Hover info**: pasar el mouse sobre un campo muestra de qué `.stem` hereda y su schema
- **Go to definition**: click en un campo → navega al `.stem` que lo define
- **Code actions**: "Add missing required field" como quick fix

### 3.2 GitHub Actions / CI First-Class

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

## 4. Migraciones de Schema

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

---

## 5. AI/LLM como Ciudadano de Primera Clase

### 5.1 MCP Server Expandido

Más allá del wrapper 1:1 de CLI commands:

- **Tool: `rootline_create`** — el LLM pide crear un documento y Rootline garantiza que sea válido antes de escribirlo
- **Tool: `rootline_suggest`** — dado un documento parcial, sugiere campos faltantes con valores válidos
- **Resource: schema context** — expone el schema efectivo como contexto para que el LLM sepa las restricciones *antes* de generar

### 5.2 Rootline como Knowledge Base Estructurado para RAG

El combo query + schema + hierarchy es perfecto para RAG:

- Rootline reemplaza el "grep + hope" que usan la mayoría de RAG pipelines sobre docs
- El schema le dice al LLM qué campos existen y qué significan
- La jerarquía le da contexto (este task pertenece a este feature, que pertenece a este epic)

### 5.3 `.stem` como Prompt Engineering Implícito

Un `.stem` file es, en esencia, un prompt estructurado. Es **constraint-based prompt engineering** — más confiable que instrucciones en lenguaje natural.

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
# Emits events: document.created, document.updated, document.invalid, schema.changed
```

- Webhooks configurables por evento
- Dashboard web en tiempo real (via SSE desde `rootline serve`)

### 6.2 Multi-Repo Federation

```yaml
# .stem
extends: github.com/org/shared-schemas/docs.stem@v2
```

- Schemas compartidos entre repositorios
- Central governance con local override

### 6.3 Time-Travel y Auditoría

- Git-aware: correlaciona cambios de `.stem` con commits
- Blame-style tracing: "this field became required in commit X"
- Compliance trail: prueba que un documento fue válido en fecha Y

### 6.4 Graph de Dependencias (links)

El campo `links` en `.stem` está reservado pero no implementado:

- Detección de ciclos
- Estado derivado por propagación
- Visualización interactiva (DOT/mermaid)

---

## 7. Nuevos Dominios de Aplicación

### 7.1 Static Site Generators

Hugo, Jekyll, Astro — todos usan frontmatter. Rootline puede ser el **linter de contenido**.

### 7.2 Compliance y Governance

Regulaciones (SOC2, GDPR, ISO) requieren documentación estructurada. `.stem` es auditable, determinístico, versionado en git.

### 7.3 API Documentation Enforcement

Schema enforcement para metadata de APIs (version, deprecation, owner).

### 7.4 Monorepo Package Metadata

`.stem` para validar README frontmatter de packages en un monorepo.

---

## 8. Oportunidades Técnicas

### 8.1 Incremental Validation

- Content-addressable cache (hash de archivo + hash de .stem chain)
- Solo re-valida si el archivo o algún `.stem` ancestro cambió

### 8.2 Custom Validation Rules

```yaml
validate:
  - rule: custom
    script: ./validators/check-links.sh
    field: references
```

- Reglas de validación como scripts externos
- Interface: stdin=record JSON, exit code=pass/fail, stdout=error message

---

## 9. Distribución

### 9.1 Homebrew + Binarios Multi-plataforma

goreleaser para releases multi-plataforma + Homebrew tap.

### 9.2 Playground Web (WASM)

Go compila a WASM nativamente. Un playground interactivo bajaría la barrera de entrada.
