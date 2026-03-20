---
estado: Deferred
fecha: "2026-02-27"
metodo: collaborative-research
---
# SDD: Specs que una AI lee para generar tests

**Contexto**: Evaluar como escribir documentos de especificacion que una AI pueda leer y, a partir de ellos, generar tests Go ejecutables para rootline. Tanto para features nuevas como para cubrir gaps en codigo existente.

**Objetivo**: Un documento spec → AI lo lee → produce `_test.go` listos para `go test`.

---

## Part 1 — Como funciona spec-kit

### Arquitectura

spec-kit es un framework de proceso, no un generador de tests. Provee:

1. **Constitution** (`memory/constitution.md`): Principios inmutables del proyecto — testing standards, stack, quality gates. Funciona como ley arquitectonica que todo artefacto debe respetar.

2. **Specification** (`specs/{feature}/spec.md`): Define QUE y POR QUE. User stories con acceptance criteria en formato Given/When/Then. Explicitamente evita detalles de implementacion.

3. **Plan** (`specs/{feature}/plan.md`): Define COMO. Stack, data model, contratos. Incluye constitution check gate.

4. **Tasks** (`specs/{feature}/tasks.md`): Breakdown ejecutable. Patron test-first: tests se escriben ANTES de implementacion. Cada task tiene ID, story ref, y path exacto.

### Principios clave para generacion de tests

spec-kit impone:
- "No implementation code shall be written before unit tests are written, validated, approved, and confirmed to FAIL"
- Acceptance criteria deben ser "INDEPENDENTLY TESTABLE"
- Tests siguen orden: contract tests → integration tests → e2e tests → unit tests
- Tasks marcadas con `[P]` pueden ejecutarse en paralelo

### Formato de specs

```
specs/001-feature-name/
├── spec.md          # User stories + acceptance criteria (Given/When/Then)
├── plan.md          # Technical decisions + data model + API contracts
├── research.md      # Context investigation
├── data-model.md    # Entity definitions
├── contracts/       # Interface contracts
├── quickstart.md    # Validation scenarios
└── tasks.md         # Ordered task list (test-first)
```

### Flujo con AI agent

```
/speckit.specify  → AI genera spec.md desde descripcion libre
/speckit.plan     → AI genera plan.md respetando constitution
/speckit.tasks    → AI genera tasks.md desde plan + contracts
/speckit.implement → AI ejecuta tasks (test-first) → codigo
```

La AI lee spec.md + plan.md + constitution.md y genera tests que:
- Cubren cada acceptance criterion del spec
- Respetan los principios de la constitution
- Fallan antes de implementar (TDD)

---

## Part 2 — Que tiene rootline hoy vs que necesita

### Lo que ya existe

Las tasks de rootline tienen estructura similar a specs:

```markdown
# T001: Add HierarchyLevel struct

## Especificacion Tecnica
```yaml
tipo: software-module
paquete: internal/rules
interfaces:
  - nombre: HierarchyLevel
    metodos:
      - input: Match string, Children []string, Schema map[string]SchemaField
tests:
  - TestHierarchyLevelParsing: YAML with levels unmarshals correctly
  - TestHierarchyLevelEmpty: StemFile without levels has nil Levels field
```

## Criterios de Aceptacion
- `go test ./internal/rules/ -run TestHierarchyLevel -v` passes
- StemFile with `levels:` YAML unmarshals with correct HierarchyLevel values
- StemFile without `levels:` YAML has `Levels == nil`

## Fuente de verdad
- `internal/rules/rules.go` — StemFile struct (lines 14-25)
```

### Lo que falta para que una AI genere tests

| Aspecto | spec-kit | Rootline tasks hoy | Gap |
|---------|----------|---------------------|-----|
| Acceptance criteria | Given/When/Then formal | Prosa libre + comandos go test | No hay formato parseable |
| Input/Output examples | Tabla de test cases | Solo nombres de test | Faltan datos concretos |
| Contract definitions | `contracts/` con interfaces | `interfaces:` en YAML inline | Suficiente pero inconsistente |
| Constitution (invariants) | Archivo dedicado | Dispersos en `## Preserva` | No centralizados |
| Data model | `data-model.md` dedicado | Implícito en `Fuente de verdad` | AI debe leer el codigo fuente |
| Error cases / edge cases | Seccion explicita en spec | No sistematico | Faltan |
| Existing test patterns | No aplica (greenfield) | 69 archivos _test.go | AI necesita ver patrones existentes |

### Insight clave

El gap principal no es de estructura sino de **datos concretos**. Las specs dicen "YAML with levels unmarshals correctly" pero no dicen:

```yaml
# Input concreto:
input: |
  levels:
    epic:
      match: "E*"
      children: [feature]

# Output esperado:
expected:
  Levels["epic"].Match: "E*"
  Levels["epic"].Children: ["feature"]
```

Una AI necesita **ejemplos concretos de input/output** para generar tests correctos. Los nombres de test no bastan.

---

## Part 3 — Opciones

### Opcion A: Adoptar spec-kit directamente

**Como**: `specify init rootline`, crear constitution desde CLAUDE.md, escribir specs para features y para retroactive coverage.

**Flujo**:
```
1. Crear .specify/constitution.md desde CLAUDE.md + invariantes de E08
2. Para feature nueva: /speckit.specify → /speckit.plan → /speckit.tasks → implement
3. Para retroactive: escribir spec de paquete existente → /speckit.tasks → AI genera tests
```

**Ventajas**:
- Framework probado por GitHub, con comunidad activa
- Templates ya optimizados para consumo por AI agents
- Slash commands integrados con Claude Code, Copilot, Cursor
- Constitution centraliza invariantes (el gap de E08)
- Extension system permite customizar (V-Model extension existe para trazabilidad)

**Desventajas**:
- CLI es Python (`uvx`) — agrega dependencia no-Go al proyecto
- Las specs viven en `.specify/`, separadas de `docs/epics/` — dos sistemas de planning
- No integrado con rootline validate — las specs no tienen `.stem` schema
- Formato generico: no aprovecha que rootline ya sabe sobre sus propios schemas
- Requiere aprender un workflow nuevo

**Esfuerzo**: Bajo para empezar (init + constitution). Medio para adoptarlo completamente.

### Opcion B: Adaptar el formato spec-kit a docs de rootline

**Como**: Tomar los templates de spec-kit (spec-template.md, plan-template.md, tasks-template.md) y crear un `.stem` schema + template para docs/specs/ que rootline valide nativamente.

**Flujo**:
```
1. Crear docs/specs/ con .stem que valide campos SDD
2. Crear constitution.md en la raiz del proyecto
3. Para cada feature/paquete: escribir spec con Given/When/Then + input/output examples
4. AI lee spec → genera _test.go
```

**Formato propuesto** para un spec doc:

```markdown
---
estado: Draft
paquete: internal/rules
tipo: spec
scope: [merge, hierarchy]
---
# Spec: StemFile Levels Parsing

## Constitution Check
- [x] Test-first: tests se generan antes de implementar
- [x] Coverage >= 85%

## Acceptance Criteria

### AC-001: Basic levels parsing
**Given** a .stem file with `levels:` section containing epic and feature definitions
**When** the YAML is unmarshaled into StemFile
**Then** `Levels["epic"].Match` equals `"E*"` and `Levels["epic"].Children` equals `["feature"]`

```yaml
input: |
  version: 1
  levels:
    epic:
      match: "E*"
      children: [feature]
      schema:
        id: { type: sequence, prefix: E, digits: 2 }
expected:
  Levels.epic.Match: "E*"
  Levels.epic.Children: ["feature"]
  Levels.epic.Schema.id.Type: "sequence"
```

### AC-002: Empty levels
**Given** a .stem file without `levels:` section
**When** the YAML is unmarshaled into StemFile
**Then** `Levels` is nil

### AC-003: Levels merge (child overrides parent)
**Given** parent .stem with `levels.epic.schema.id.digits: 2`
**And** child .stem with `levels.epic.schema.id.digits: 3`
**When** merged via MergeStemFiles
**Then** result has `levels.epic.schema.id.digits: 3`

## Edge Cases
- AC-004: `levels:` with empty map → `Levels` is empty map, not nil
- AC-005: `levels:` with unknown fields → ignored (forward compat)

## Test Patterns (reference)
- See `internal/rules/merge_test.go` for merge test patterns
- See `internal/e2e/hierarchy_test.go` for e2e integration patterns
- Use `setupProject(t, map[string]string{...})` for filesystem fixtures
```

**Ventajas**:
- Validable por rootline (`.stem` con `required: true` en `paquete`, `scope`)
- Formato optimizado para AI: Given/When/Then + YAML input/output concretos
- Vive en el mismo repo, misma toolchain
- Reusa patrones de test existentes (AI los ve como referencia)
- No agrega dependencias externas

**Desventajas**:
- Hay que diseñar y mantener el formato propio
- No tiene la comunidad/iteracion de spec-kit
- Requiere crear templates y documentar el formato

**Esfuerzo**: Medio. Diseñar schema + template + escribir primer spec como prueba.

### Opcion C: Usar spec-kit para specs + rootline para validar

**Como**: Usar spec-kit como herramienta de authoring (sus slash commands y templates) pero almacenar las specs en una carpeta que rootline tambien valide.

**Flujo**:
```
1. specify init rootline
2. Agregar .stem en .specify/ para validar campos minimos
3. /speckit.specify genera specs → rootline validate .specify/ las chequea
4. AI lee specs + constitution → genera tests
```

**Ventajas**: Lo mejor de ambos mundos — templates probados + validacion rootline.
**Desventajas**: Complejidad de mantener dos herramientas sincronizadas.

---

## Part 4 — Aplicacion retroactiva: cubrir gaps de testing

Para codigo existente sin suficiente cobertura, el flujo seria:

### 1. Identificar gaps

```bash
go test ./... -coverprofile=c.out
go tool cover -func=c.out | grep -v "100.0%" | sort -t: -k3 -n
```

### 2. Escribir spec del paquete

Para cada paquete con baja cobertura, escribir un spec doc que describa:
- Que hace el paquete (resumen del API publico)
- Inputs/outputs de cada funcion publica
- Edge cases conocidos
- Invariantes (ej: "Extract nunca retorna nil record sin error")

### 3. AI genera tests

La AI lee:
1. El spec doc (acceptance criteria + input/output examples)
2. El codigo fuente del paquete (para entender tipos y signatures)
3. Los tests existentes (para seguir patrones: table-driven, setupProject, etc.)
4. La constitution (invariantes del proyecto)

Y produce archivos `_test.go` que:
- Cubren cada acceptance criterion
- Siguen los patrones existentes del proyecto
- Usan standard `testing` package (sin frameworks externos)
- Pasan `go vet` y `golangci-lint`

### El valor de la constitution

Para rootline, una constitution capturaria:

```markdown
# Rootline Test Constitution

## Article I: Standard Library Only
All tests use Go standard `testing` package. No testify, no gomock, no external test frameworks.

## Article II: Filesystem Fixtures
E2E tests use `setupProject(t, map[string]string{...})` with inline filesystem.
Always create `.git` directory as boundary marker for WalkUp.

## Article III: Assertion Patterns
- Use `t.Fatalf` for precondition failures (wrong count, missing setup)
- Use `t.Errorf` for assertion failures (wrong value)
- Use `t.Helper()` in all helper functions

## Article IV: Table-Driven Tests
Prefer table-driven tests for functions with multiple input/output combinations.
Pattern: `tests := []struct{ name string; input X; want Y }{...}`

## Article V: No Mocks
Test against real implementations. Use TempDir for filesystem, real YAML for parsing.
Only exception: MCP tests use `mcp.NewInMemoryTransports()`.

## Article VI: Coverage Gate
Coverage must stay >= 85%. Check with:
`go test ./... -coverprofile=c.out && go tool cover -func=c.out | tail -1`

## Article VII: Naming
Test functions: `Test{Subject}_{Scenario}` (e.g., `TestMerge_ChildOverridesParent`)
Benchmark functions: `Benchmark{Subject}` with `b.Loop()` (Go 1.24+)
```

Este documento, combinado con los specs, le da a la AI todo el contexto para generar tests idiomaticos.

---

## Part 5 — Comparativa final

| Criterio | A: spec-kit directo | B: Formato propio | C: Hibrido |
|----------|--------------------|--------------------|------------|
| Tiempo de setup | Bajo (init) | Medio (diseñar) | Medio |
| Calidad de specs | Alta (templates probados) | Depende del diseño | Alta |
| Validacion rootline | No | Si | Si |
| Dependencia externa | Python (uvx) | Ninguna | Python (uvx) |
| Retroactive coverage | Posible | Posible | Posible |
| Curva de aprendizaje | Media (nuevo workflow) | Baja (mismo repo) | Alta (dos tools) |
| AI test generation | Bueno (formato estandar) | Bueno (si incluye I/O) | Bueno |
| Evolucion | Comunidad GitHub | Propia | Ambas |

### Recomendacion

**Empezar con Opcion A (spec-kit directo)** para validar el concepto rapidamente:

1. `specify init rootline` — 5 minutos
2. Crear constitution desde CLAUDE.md + invariantes — 1 hora
3. Escribir spec de un paquete con baja cobertura (ej: `internal/graph/`) — 1 hora
4. Pedir a la AI que genere tests desde el spec — evaluar calidad
5. Si funciona bien: decidir si seguir con spec-kit o migrar a formato propio (Opcion B)

La razon: spec-kit ya tiene templates optimizados para AI consumption. No tiene sentido reinventar el formato sin antes probar que el concepto funciona. Si despues el formato propio es mejor, la migracion es trivial (son archivos Markdown).

**El factor critico no es la herramienta sino los datos**: el spec necesita **input/output concretos** en cada acceptance criterion, no solo descripciones en prosa. Esto aplica independientemente de si usas spec-kit, formato propio, o cualquier otra herramienta.

---

## Part 6 — Estado actual de rootline y siguiente paso sugerido

### Feasibility Assessment (updated 2026-03-20)

**Urgency has decreased significantly.** When this doc was written, several packages had low coverage. Current state:

| Paquete | Coverage actual | Tests | Cambio desde Feb 2026 |
|---------|----------------|-------|----------------------|
| `internal/infer/` | 97.8% | 32 archivos _test.go | Era 2 archivos → ahora 32. Masivamente expandido |
| `internal/graph/` | 95.1% | Well covered | Era candidato #1 → ya cubierto |
| `internal/fix/` | — | Expanded | Era candidato #2 → coverage mejorado |
| `internal/mcp/` | — | 2+ archivos | Posible candidato aún |
| `internal/rules/` | — | 12 archivos | Ya bien cubierto |
| `internal/extract/` | — | 4 archivos + fuzz | Ya bien cubierto |

**Conclusion**: El concepto de SDD→test generation sigue siendo valido pero la urgencia es baja. Coverage global esta por encima del gate de 85%. El valor principal seria para **features nuevas** (TDD desde spec), no para retroactive coverage.

**Deferred because**: spec-kit no adoptado, no constitution.md, no `docs/specs/`. La Opcion B (formato nativo) sigue siendo la mas alineada con rootline si se reactiva.

### Paquetes candidatos para retroactive spec (original, for reference)

| Paquete | Tests existentes (Feb 2026) | Que cubre | Candidato para spec retroactivo |
|---------|-----------------|-----------|--------------------------------|
| `internal/graph/` | 1 archivo | Cycle detection, broken links | ~~Si~~ Cubierto (95.1%) |
| `internal/fix/` | 1 archivo | Frontmatter rewriting | Posible aún |
| `internal/infer/` | 2 archivos | Schema inference | ~~Si~~ Cubierto (97.8%) |
| `internal/mcp/` | 2 archivos | MCP server tools | Posible — 8 tools, variedad de edge cases |
| `internal/rules/` | 12 archivos | Validation, merge, hierarchy | No — ya bien cubierto |
| `internal/extract/` | 4 archivos + fuzz | YAML extraction | No — ya bien cubierto |

### Siguiente paso

Escribir un spec de `internal/graph/` como piloto. Es el paquete ideal porque:
- Solo tiene 1 archivo de test (graph_test.go)
- Tiene funcionalidad bien definida (cycle detection, broken links, DOT/Mermaid output)
- Los inputs son grafos de wiki-links — faciles de expresar como Given/When/Then
- El output es deterministico — facil de verificar

---

## Referencias

- [spec-kit (GitHub)](https://github.com/github/spec-kit) — Toolkit SDD para agentes AI
- [spec-driven.md](https://github.com/github/spec-kit/blob/main/spec-driven.md) — Guia completa de metodologia
- [Spec-Driven Development with AI (GitHub Blog)](https://github.blog/ai-and-ml/generative-ai/spec-driven-development-with-ai-get-started-with-a-new-open-source-toolkit/)
- [Spec-Driven Development: spec-kit (Microsoft)](https://developer.microsoft.com/blog/spec-driven-development-spec-kit)
- [SuperOptiX BDD Guide](https://superagenticai.github.io/superoptix-ai/guides/bdd/#evaluation-metrics) — Evaluacion multi-criterio (no directamente aplicable)
- E08: Specification Contract Model — `docs/epics/E08-specification-contract-model/README.md`
- Intrinsic Hierarchy Principle — `docs/research/intrinsic-hierarchy-principle.md`
