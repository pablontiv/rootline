---
estado: Pre-research
fecha: "2026-02-23"
metodo: brainstorm
---
# Áreas de Oportunidad — Ideas No Implementadas

**Origen**: Rescatado de I9 (eliminado en 630f597). Se excluyeron secciones ya implementadas: GitHub Actions (F06), Migraciones (E07), Claude Code Plugin (F07), Graph (F05), Homebrew (release pipeline), MCP Server (F05-mcp).

---

## 1. Schema Registry Publicable

Hoy cada proyecto define sus `.stem` desde cero. Oportunidad: un registro de schemas compartibles.

```bash
rootline schema pull pablontiv/epic-tracking
# descarga .stem files para estructura Epic→Feature→Story→Task
```

- Schemas versionados y componibles (como helm charts pero para estructura documental)
- `rootline schema init --from template/zettelkasten` para arrancar un knowledge base
- Namespace por comunidad: `rootline schema pull obsidian-community/daily-notes`

---

## 2. Multi-Repo Federation

```yaml
# .stem
extends: github.com/org/shared-schemas/docs.stem@v2
```

- Schemas compartidos entre repositorios
- Central governance con local override
- Complementa el Schema Registry: uno distribuye, el otro enlaza

---

## 3. `.stem` como Contrato entre Humanos y Máquinas

El `.stem` no solo valida — **define el contrato** de lo que un directorio contiene. Esto es valioso para:

- **Onboarding**: un developer nuevo lee `.stem` y entiende qué documentos existen y qué campos tienen
- **AI Agents**: un LLM lee `.stem` y sabe exactamente cómo crear/editar documentos válidos
- **CI/CD**: el pipeline valida que los PRs no rompan el contrato documental

---

## 4. Language Server Protocol (LSP)

Un LSP para `.stem`-aware editing sería transformador:

- **Autocompletado** de valores enum en frontmatter (`Estado: ` → sugiere `Pending`, `Completado`...)
- **Diagnósticos en tiempo real** (subrayado rojo si falta campo required)
- **Hover info**: pasar el mouse sobre un campo muestra de qué `.stem` hereda y su schema
- **Go to definition**: click en un campo → navega al `.stem` que lo define
- **Code actions**: "Add missing required field" como quick fix

---

## 5. Rootline como Knowledge Base Estructurado para RAG

El combo query + schema + hierarchy es perfecto para RAG:

- Rootline reemplaza el "grep + hope" que usan la mayoría de RAG pipelines sobre docs
- El schema le dice al LLM qué campos existen y qué significan
- La jerarquía le da contexto (este task pertenece a este feature, que pertenece a este epic)

---

## 6. `.stem` como Prompt Engineering Implícito

Un `.stem` file es, en esencia, un prompt estructurado. Es **constraint-based prompt engineering** — más confiable que instrucciones en lenguaje natural.

---

## 7. Watch Mode + Event System

```bash
rootline watch docs/
# Watches filesystem, re-validates on change
# Emits events: document.created, document.updated, document.invalid, schema.changed
```

- Webhooks configurables por evento
- Dashboard web en tiempo real (via SSE desde `rootline serve`)

---

## 8. Time-Travel y Auditoría

- Git-aware: correlaciona cambios de `.stem` con commits
- Blame-style tracing: "this field became required in commit X"
- Compliance trail: prueba que un documento fue válido en fecha Y

---

## 9. Nuevos Dominios de Aplicación

### 9.1 Static Site Generators

Hugo, Jekyll, Astro — todos usan frontmatter. Rootline puede ser el **linter de contenido**.

### 9.2 Compliance y Governance

Regulaciones (SOC2, GDPR, ISO) requieren documentación estructurada. `.stem` es auditable, determinístico, versionado en git.

### 9.3 API Documentation Enforcement

Schema enforcement para metadata de APIs (version, deprecation, owner).

### 9.4 Monorepo Package Metadata

`.stem` para validar README frontmatter de packages en un monorepo.

---

## 10. Incremental Validation

- Content-addressable cache (hash de archivo + hash de .stem chain)
- Solo re-valida si el archivo o algún `.stem` ancestro cambió

---

## 11. WASM Playground

Go compila a WASM nativamente. Un playground interactivo bajaría la barrera de entrada.
