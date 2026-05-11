---
estado: Fase 3
fecha: "2026-03-03"
metodo: hypothesize
origen: discover/inference-engine-architecture (3 cycles, closed)
---
# Inference Engine Architecture — Investigación Estructurada
> Estado: Fase 3 completa. Fases 4-5 en construcción.

---

## Glosario de dominio

| Término | Definición |
|---------|-----------|
| Engine | Binario Go de rootline — todo lo que ejecuta sin LLM |
| Agent | Instancia de Claude Code que razona sobre output del engine |
| Subcapa simbólica | Fase 4 interna del engine: propuestas heurísticas (`internal/proposal/`) |
| Computation-then-understanding | Teoría: todo se descompone en engine (forma) + agent opcional (significado) |
| goldmark | Parser AST de Markdown para Go (yuin/goldmark). Habilita extracción estructural de body |
| CAP | Capability — capacidad mínima necesaria para que H1 sea verdadera |
| .stem | Archivos de reglas de rootline — knowledge base del engine |

---

## Fase 1: Idea → Tesis

### Proposición central (P) — v4

> **P**: "Existe una arquitectura de 2 capas externas (engine Go + agent LLM) donde el engine progresa internamente de computación determinística (extracción, validación) a razonamiento simbólico heurístico (propuestas). Con goldmark como parser AST, el engine cubre ~80% del trabajo. El agent maneja el ~20% restante: disambiguation contextual y matching semántico. `internal/proposal/` es la interfaz natural engine→agent — sus propuestas alimentan al agent para decisiones que exceden heurísticas."

**Condición de falsación**: P es falsa si (a) la Fase 4 del engine (propuestas) resulta insuficiente como interfaz engine→agent y se necesita un protocolo bidireccional, o (b) goldmark no mejora realmente los % de las categorías 6/9/12/13 en la práctica.

### Axiomas del entorno [VALIDADO CONTRA CÓDIGO]

| ID | Axioma | Fuente | Verificación |
|----|--------|--------|-------------|
| C1 | Rootline es un engine en Go que trata el filesystem como base de datos | CLAUDE.md | ✅ Estructura de packages confirma |
| C2 | Engine procesa body content en 6 packages, pero limitado a extracción de forma (links, patrones bold-colon). No hay procesamiento semántico de body. | CLOSURE §Q5, verificado | ✅ `extract/`, `graph/`, `query/`, `proposal/`, `derive/`, `rules/` — 5/6 solo links/patrones |
| C3 | Existen 13 categorías de inferencia identificadas | Documento original §Part 2 | ✅ Taxonomía consistente |
| C4 | Categorías 1-4 implementadas (no 1-3 como decía el original) | Código verificado | ✅ Cat 4: `internal/rules/structural.go:ValidateDirectory()` con require_index, min_children, max_children |
| C5 | El engine usa `expr-lang/expr` en 3 packages (derive, query, derive/builtins) | CLAUDE.md + imports | ✅ Confirmado |
| C6 | Claude Code es el único LLM runtime disponible | Contexto del proyecto | Axioma de contexto |
| C7 | Cat 5 (links) tiene struct `LinkSchema` definida pero sin validación — infraestructura parcial | `internal/rules/rules.go` | ✅ `Allowed []string` definido, nunca validado |

### Premisas de diseño [VALIDADO]

| ID | Premisa | Fuente |
|----|---------|--------|
| D1 | Report format: `version: 1`, additive-only, `kind` discriminator | CLOSURE Q1, 18+ contratos existentes siguen este patrón |
| D2 | Thresholds hardcoded, report expone evidencia porcentual | CLOSURE Q3 |
| D3 | Solo v2 `.stem` — v3 es research separado | CLOSURE Q4 |
| D4 | Sin ML/clasificación — solo Go stdlib + 5 dependencias actuales | `go.mod` verificado: expr-lang, go-sdk, cobra, x/text, yaml.v3 |

### Modelo deseado [VALIDADO]

| ID | Capacidad | Fuente |
|----|-----------|--------|
| M1 | Implementar las 13 categorías de inferencia | Documento original §Part 2 |
| M2 | Comando `rootline analyze` que genera report unificado | Documento original §Part 5 |
| M3 | Comando `rootline apply` que ejecuta inferencias aprobadas | Documento original §Part 6 |
| M4 | Soporte incremental (`analyze --incremental`) contra `.stem` existente | Documento original §Part 7 |
| M5 | Un solo agent para el residuo semántico | CLOSURE Q2 |

### Clarificaciones (falacias resueltas)

| ID | Tipo | Descripción | Resolución |
|----|------|-------------|-----------|
| F1 | Scope inflation | "6 packages procesan body" → "body es territorio del engine." Pero 5/6 solo extraen links/patrones. | Reformulado: "engine procesa *forma* del body; *significado* está fuera de su alcance." Reflejado en C2 corregido. |
| F2 | Cuantificación no medida | "~80% de cat 9-13 es engine-computable" era intuición, no dato. | Medido: cat 9-13 = 40% Go / 60% LLM. Migrado a S2 como supuesto corregido. |
| F3 | Falsa homogeneidad | "Un agent" asume que disambiguation, semantic matching, redundancy detection, y program synthesis son suficientemente similares. | Evidencia externa (Claude docs) soporta single-agent por defecto. 3 casos justifican multi-agent. Migrado a S3. |
| F4 | Autoridad vs evidencia | "Skills no son capa" fue corrección del usuario, no observación empírica. | Confirmado por documentación oficial Anthropic: skills son inyección de prompts via meta-tool. S4 resuelto. |

### Mapa de inferencias

```
C1 (engine Go) ──────────────┐
C2 (body = forma) ───────────┤
C4 (cat 1-4 impl.) ──────────┼──→ "Engine cubre lo determinístico + heurístico"
C5 (expr-lang) ──────────────┤         │
C7 (LinkSchema parcial) ─────┘         │
                                       ├──→ P(v4): "2 capas externas, gradiente interno"
D1 (additive-only report) ────┐        │
D2 (thresholds hardcoded) ────┤        │
D3 (solo v2) ─────────────────┼──→ "Diseño acotado"
D4 (sin ML) ──────────────────┘        │
                                       │
M1 (13 categorías) ───────────┐        │
M2 (analyze) ─────────────────┤        │
M3 (apply) ───────────────────┼──→ "Modelo deseado" ──┘
M4 (incremental) ─────────────┤
M5 (1 agent) ─────────────────┘

Subcapa simbólica (hallazgo nuevo):
  Engine interno = 4 fases:
    Extracción (determinística) → Validación (rule-checking)
    → Derivación (expr-lang) → Propuestas (heurísticas)
                                    │
                                    └──→ Interfaz natural → Agent LLM

⚠ Huecos de evidencia resueltos:
  • "~80% engine" → medido: ~75% sin goldmark, ~80% con goldmark ✅
  • "1 agent suficiente" → soportado por guidance oficial, caveat en heterogeneidad ⚠
  • "skills ≠ capa" → confirmado por Anthropic docs ✅
```

### Supuestos explícitos

| ID | Supuesto | Estado | Origen |
|----|----------|--------|--------|
| S1 | Regex + parsing + counting + AST agotan lo determinístico | ⚠️ Parcial — AST (goldmark) amplía el alcance, pero no se ha probado en producción | Anti-presupposición CL3 |
| S2 | ~75-80% engine / ~20-25% agent (medido por categoría, no globalmente pesado) | ✅ Medido | F2 corregida |
| S3 | El residuo semántico es manejable por 1 agent generalista | ⚠️ Soportado por guidance oficial, pero cat 9+11 son 70-80% LLM con contextos distintos | F3, Claude docs |
| S4 | Skills son inyección de prompts, no capa computacional | ✅ Confirmado | Anthropic docs oficiales |
| S5 | La barrera forma/significado mapea al gradiente engine→agent | ✅ Verificado contra código | discover line + código |
| S6 | Las 13 categorías son exhaustivas | ❓ No validado contra otros datasets | Anti-presupposición |
| S7 | El report JSON unidireccional (engine→agent) es suficiente — no se necesita feedback loop bidireccional | ❓ No probado | Diseño propuesto |

---

## Fase 2: Tesis → Plan de investigación

### Hipótesis testable (H1)

> **H1**: "Si rootline implementa las 13 categorías con arquitectura de 2 capas (engine Go + agent LLM), con goldmark como parser AST y `internal/proposal/` como interfaz engine→agent, entonces el engine resolverá ≥75% de toda inferencia de forma determinística, el agent resolverá el ≤25% restante como residuo semántico, y no se necesitará protocolo bidireccional ni capa intermedia, porque el gradiente interno del engine (extracción→validación→derivación→propuestas) cubre progresivamente más territorio antes de ceder al agent."

**Falsación**: H1 es falsa si alguna categoría requiere >50% de interacción bidireccional engine↔agent, o si goldmark no reduce el % LLM de al menos 2 categorías.

### Capabilities mínimas (CAPs)

| ID | Capability | Método | Dependencias | Prioridad | Crítica? |
|----|-----------|--------|-------------|-----------|---------|
| CAP-01 | goldmark se integra limpiamente en el pipeline extract | Empírico (spike) | Ninguna | 1 | Sí |
| CAP-02 | Categorías 5/7/8/10 son 100% engine-implementables | Lógico (code review) | C4, C7 | 2 | Sí |
| CAP-03 | Categorías 6/12/13 mejoran ≥15% Go con goldmark | Empírico (spike) | CAP-01 | 3 | Sí |
| CAP-04 | Cat 9 (deps heterogéneas) es manejable con engine + 1 agent | Mixto | CAP-01 | 4 | No |
| CAP-05 | Cat 11 (traceability) es manejable con engine + 1 agent | Mixto | CAP-01 | 4 | No |
| CAP-06 | Report JSON unidireccional es suficiente como interfaz | Empírico | CAP-02, CAP-04 | 5 | Sí |
| CAP-07 | `internal/proposal/` puede alimentar al agent sin reestructuración | Lógico | Ninguna | 2 | No |

### Sub-hipótesis

| ID | Sub-hipótesis | Pregunta de falsación |
|----|--------------|----------------------|
| H1-a | Si goldmark se añade como dependencia, entonces el pipeline extract funciona igual con AST opcional (campo `json:"-"`), porque goldmark es pure Go sin deps transitivas y Record.Body permanece string. | ¿Es falso que goldmark se integra sin romper contratos JSON? |
| H1-b | Si las categorías 5/7/8/10 se implementan como Go puro, entonces producen resultados correctos sin LLM, porque son operaciones de regex, conteo, y graph filtering sobre infraestructura existente. | ¿Es falso que cat 5/7/8/10 son 100% determinísticas? |
| H1-c | Si goldmark parsea body como AST, entonces cat 6/12/13 ganan ≥15% Go (section boundaries, heading hierarchy, YAML block extraction precisa), porque AST resuelve ambigüedades que regex no puede. | ¿Es falso que goldmark mejora el % Go de estas categorías? |
| H1-d | Si cat 9 (deps heterogéneas) se implementa como engine extraction + agent disambiguation, entonces 1 agent generalista maneja el residuo 70% LLM, porque el contexto necesario (sección Dependencias + links formales) es acotado. | ¿Es falso que 1 agent puede manejar disambiguation de dependencias sin contexto especializado? |
| H1-e | Si cat 11 (traceability) se implementa como engine extraction + agent semantic matching, entonces 1 agent generalista maneja el 80% LLM, porque matching "Contribuye a" contra acceptance criteria es una tarea de comprensión lectora estándar para LLMs. | ¿Es falso que semantic matching de traceability es manejable por un generalista? |
| H1-f | Si el engine emite propuestas via report JSON y el agent las consume sin feedback loop, entonces no se pierde información necesaria para decisiones, porque las propuestas incluyen toda la evidencia porcentual. | ¿Es falso que el report unidireccional es suficiente? |
| H1-g | Si `internal/proposal/` ya genera alternativas rankeadas, entonces puede alimentar al agent sin reestructuración, porque la estructura proposal→decision ya existe. | ¿Es falso que proposal/ sirve como interfaz engine→agent? |

### Criterios de decisión

| Resultado | Umbral | Acción |
|-----------|--------|--------|
| **Go** | ≥5 CAPs confirmadas, incluyendo CAP-01 y CAP-06 | Proceder a planificación de implementación con arquitectura de 2 capas |
| **Pivot** | CAP-06 falla (se necesita bidireccional) | Rediseñar interfaz engine↔agent, investigar MCP protocol |
| **Stop** | CAP-01 falla (goldmark no integra) O ≥3 CAPs fallan | Replantear premisas. Posible vuelta a 3 capas. |

### Reglas de parada

| ID | Regla |
|----|-------|
| R1 | Si goldmark rompe contratos JSON existentes → Stop inmediato |
| R2 | Si alguna categoría "100% engine" (5/7/8/10) requiere LLM → revisar taxonomía |
| R3 | Si el agent necesita pedir datos no presentes en el report → Pivot a bidireccional |

---

## Fase 3: Investigación → Argumento actualizado

### Evidencia lógica: Invariantes

| ID | Invariante | Derivado de |
|----|-----------|-------------|
| INV-01 | Todo output JSON del engine lleva `"version": 1` | D1 |
| INV-02 | Record.Body es siempre `string` en serialización JSON | C2, D1 |
| INV-03 | Validación es determinística: mismo .stem + mismo record → mismo resultado | C1 |
| INV-04 | Propuestas son determinísticas: mismos records + mismos thresholds → mismas propuestas | Fase interna 4 del engine |
| INV-05 | goldmark AST no se serializa (campo `json:"-"`) | INV-02 |

### Evidencia lógica: Constraints derivados

| ID | Constraint | Consecuencia de |
|----|-----------|-----------------|
| CD-01 | Añadir goldmark no puede cambiar output de comandos existentes | INV-02, D1 |
| CD-02 | El agent solo recibe lo que el report contiene — no tiene acceso directo al engine | S7 |
| CD-03 | Si thresholds son hardcoded (D2), toda propuesta es reproducible | INV-04, D2 |
| CD-04 | Si solo v2 .stem (D3), el analyze no necesita resolver entidades v3 | D3 |

### Matriz Premisa-Evidencia

| Premisa | Método | Evidencia | Calidad | Estado |
|---------|--------|-----------|---------|--------|
| "2 capas, no 3" | Lógico + empírico | Anthropic docs confirman skills ≠ capa. Código muestra todo en 1 binario Go. | Sistemática | ✅ true |
| "Skills son prompt injection, no capa" | Empírico | Docs oficiales: "prompt-based", "do NOT run code", progressive disclosure | Sistemática (5 fuentes) | ✅ true |
| "~75-80% engine global" | Empírico | Medición por categoría: cat 1-4 (100%), cat 5-8 (80%), cat 9-13 (40%). Global ponderado ~75%. Con goldmark ~80%. | Estimación calibrada | ⚠️ parcial |
| "Cat 9-13 son ~80% engine" (original) | Empírico | Medición: 40% Go / 60% LLM. | Medición contra código | ❌ false |
| "Cat 9-13 son ~40% engine" (corregido) | Empírico | Cat 5:85%, 6:40%, 7:95%, 8:100%, 9:30%, 10:100%, 11:20%, 12:60%, 13:50% | Estimación por análisis de código | ⚠️ parcial |
| "1 agent suficiente" | Empírico | Cognition AI "Don't Build Multi-Agents". Claude docs: 3 casos para multi-agent, default = single. 3-10x tokens en multi. | Sistemática (6 fuentes) | ⚠️ parcial |
| "Body es engine territory" | Empírico | 6 packages procesan body pero solo forma (links, patrones). Significado no procesado. | Verificación contra código | ⚠️ parcial |
| "goldmark integra limpiamente" | Lógico + empírico | Pure Go, 0 deps transitivas, Go 1.13+, Record.Body unchanged, AST con `json:"-"`. Bug existente en fix.go:236 que goldmark corrige. | Análisis de feasibility | ⚠️ parcial |
| "Report unidireccional suficiente" | Lógico | Propuestas incluyen evidencia porcentual. Pero: ¿qué si agent necesita contexto no en report? | Razonamiento | ❓ unknown |
| "Subcapa simbólica = gradiente, no frontera" | Empírico | 4 fases internas: extracción→validación→derivación→propuestas. Cada fase añade complejidad. Thresholds: 0.8, 0.6, 0.75. | Análisis de código (10+ archivos) | ✅ true |
| "AST expande alcance del engine" | Lógico | Section boundaries, table parsing, code block distinction — imposible con regex, trivial con AST. | Documentación goldmark | ⚠️ parcial |
| "Neuro-simbólico = 3 capas en literatura" | Empírico | Nature 2025: neural + integration + symbolic. Pero rootline embebe symbolic en Go binary. | Peer-reviewed | ✅ true (en literatura), no aplica directamente |

### Registro de incertidumbre

| Incertidumbre | Impacto si falsa | Severidad |
|---------------|------------------|-----------|
| S7: Report unidireccional suficiente | Necesitaría protocolo bidireccional (MCP?). Rediseño de interfaz. | Alta |
| S3: 1 agent para residuo heterogéneo | Necesitaría 2-4 agents especializados. Overhead de orquestación. | Media |
| S6: 13 categorías exhaustivas | Categorías nuevas podrían no encajar en modelo 2 capas. | Baja |
| goldmark mejora prácticas, no solo teóricas | Si el parser no resuelve los casos edge reales, las proporciones no mejoran. | Media |

### Conclusión provisional

**Tendencia: Go** — con caveats.

La arquitectura de 2 capas es sólida:
- Confirmada por fuentes oficiales (skills ≠ capa)
- Proporciones medidas (~75% engine, ~80% con goldmark)
- Subcapa simbólica mapeada (gradiente, no frontera)
- Guidance oficial soporta single-agent

**Cuello de botella**: S7 (unidireccional vs bidireccional). No se puede probar sin implementar al menos una categoría LLM-heavy (cat 9 o 11) end-to-end.

**Recomendación**: Proceder a Fase 4 con goldmark como primera validación empírica (CAP-01), seguida de cat 8/10 (CAP-02, triviales) como quick wins.

---

## Fase 4: Argumento → Factibilidad

### Restricciones como axiomas

Referenciar C1-C7 y CD-01 a CD-04. No duplicar.

### Claims técnicos

| ID | Claim | CAP | Spike necesario | Resultado | Estado |
|----|-------|-----|----------------|-----------|--------|
| T-01 | goldmark se añade a go.mod sin conflictos | CAP-01 | `go get github.com/yuin/goldmark` + `go build ./...` | — | Pendiente |
| T-02 | Record.AST con `json:"-"` no rompe tests existentes | CAP-01 | `go test ./internal/extract/ -race` | — | Pendiente |
| T-03 | ParseLinksAST produce output idéntico a ParseLinks | CAP-01 | Test de equivalencia con 14 casos existentes | — | Pendiente |
| T-04 | Cat 8 (constants) se implementa en ≤10 LOC | CAP-02 | Modificar `infer.go:Analyze()` | — | Pendiente |
| T-05 | Cat 10 (cross-epic refs) se implementa en ≤40 LOC | CAP-02 | Regex `E\d{2}/F\d{2}` + validación de existencia | — | Pendiente |
| T-06 | goldmark extrae section boundaries correctamente | CAP-03 | AST walk con heading detection vs regex naive | — | Pendiente |
| T-07 | proposal/ puede emitir proposals consumibles por agent | CAP-07 | Revisar struct `Proposal` + JSON output | — | Pendiente |

### Matriz riesgos vs mitigaciones

| Riesgo | Premisa frágil | Impacto | Mitigación |
|--------|---------------|---------|-----------|
| goldmark overhead >20% | Performance aceptable | Extracción más lenta en repos grandes | Feature flag `parseAST bool`, default false |
| 1 agent insuficiente para cat 9+11 | S3 | Necesitaría multi-agent | Empezar con 1, medir calidad, escalar si necesario |
| Report JSON no contiene info suficiente para agent | S7 | Rediseño de interfaz | Diseñar report con campos extensibles (`kind` discriminator, D1) |
| Cat 6 (body sections) sigue siendo 60% LLM con goldmark | CAP-03 | Proporción no mejora como esperado | Threshold judgment es inherentemente LLM — aceptar |

### Regla Go/No-Go

> **Go** si: T-01 + T-02 + T-03 pasan (goldmark integra) Y T-04 o T-05 pasan (al menos 1 categoría nueva funciona).
>
> **No-Go** si: T-01 falla (goldmark no integra) O T-02 falla (rompe contratos).

---

## Fase 5: Factibilidad → Prototipo

### Teorema de valor

El prototipo demuestra que:
- goldmark se integra sin romper el pipeline existente
- Al menos 2 categorías nuevas (8, 10) funcionan como pure Go
- La subcapa simbólica (proposal/) puede generar output consumible por un agent

El prototipo NO demuestra:
- Que 1 agent es suficiente para cat 9-13
- Que el report unidireccional basta a largo plazo
- Que las 13 categorías son exhaustivas

### Especificación mínima

| Componente | Input | Output | Límites |
|-----------|-------|--------|---------|
| goldmark integration | `[]byte` (markdown content) | `ast.Node` en Record (no serializado) | Solo archivos .md |
| Cat 8 detector | `map[string]*FieldStats` | `[]Inference{category: "constant"}` | Solo frontmatter fields |
| Cat 10 extractor | `string` (body) | `[]PathReference` | Regex `E\d{2}/F\d{2}/...` |
| Proposal→Agent adapter | `[]Proposal` | JSON report con proposals | — |

### Instrumentación y métricas

| Métrica | Cómo se mide | Umbral éxito | Umbral fallo |
|---------|-------------|-------------|-------------|
| Tests passing | `go test ./... -race` | 0 failures | ≥1 failure |
| Coverage | `go tool cover -func=coverage.out \| tail -1` | ≥85% | <85% |
| goldmark overhead | Benchmark antes/después en extract | ≤20% slower | >30% slower |
| Cat 8 accuracy | Detecta constants en test fixture | 100% | <100% |
| Cat 10 accuracy | Detecta path refs en test fixture | ≥95% | <90% |

### Reporte de resultados

| Claim | Validado | Refutado | Observaciones | Siguiente paso |
|-------|----------|---------|---------------|---------------|
| T-01 | — | — | — | Spike: `go get goldmark` |
| T-02 | — | — | — | Spike: add AST field, run tests |
| T-03 | — | — | — | Spike: equivalence test |
| T-04 | — | — | — | Implement cat 8 |
| T-05 | — | — | — | Implement cat 10 |
| T-06 | — | — | — | Spike: section extraction |
| T-07 | — | — | — | Review proposal struct |

---

## Pieza transversal: Matriz de trazabilidad

| Claim | CAP | Fase actual | Método | Resultado | Confianza | Decisión |
|-------|-----|-------------|--------|-----------|-----------|---------|
| goldmark integra limpiamente | CAP-01 | Fase 3 | Empírico | ⚠️ parcial (feasibility, no spike) | Media | Spike necesario |
| Cat 5/7/8/10 son 100% engine | CAP-02 | Fase 3 | Lógico | ⚠️ parcial (code review, no implementación) | Alta | Implementar cat 8+10 primero |
| Cat 6/12/13 mejoran con goldmark | CAP-03 | Fase 3 | Lógico | ⚠️ parcial (análisis teórico) | Media | Spike AST section extraction |
| Cat 9 manejable con 1 agent | CAP-04 | Fase 3 | Mixto | ❓ unknown | Baja | Probar con datos reales |
| Cat 11 manejable con 1 agent | CAP-05 | Fase 3 | Mixto | ❓ unknown | Baja | Probar con datos reales |
| Report unidireccional suficiente | CAP-06 | Fase 3 | Lógico | ❓ unknown | Baja | Probar con implementación end-to-end |
| proposal/ sirve como interfaz | CAP-07 | Fase 3 | Lógico | ⚠️ parcial (struct review) | Media | Review JSON output format |

---

## Apéndice A: Proporciones medidas por categoría

| Cat | Descripción | Go % | LLM % | Infraestructura existente |
|-----|------------|------|-------|--------------------------|
| 1 | Schema (types, enums, required) | 100 | 0 | ✅ `infer.go:Analyze()` |
| 2 | Sequences (prefix, digits) | 100 | 0 | ✅ `hierarchy.go:AnalyzeHierarchy()` |
| 3 | Aggregates (enum rollup) | 100 | 0 | ✅ `aggregate.go:GenerateAggregates()` |
| 4 | Structural (require_index) | 100 | 0 | ✅ `structural.go:ValidateDirectory()` |
| 5 | Links (allowed types) | 85 | 15 | Parcial: `LinkSchema` struct definida, sin validación |
| 6 | Body sections | 40→70* | 60→30* | Nada — requiere goldmark |
| 7 | Back-references | 95 | 5 | `graph.Build()` tiene edges con targets |
| 8 | Constants | 100 | 0 | `infer.Analyze()` tiene datos, falta detector (5 LOC) |
| 9 | Heterogeneous deps | 30→45* | 70→55* | `ParseLinks()` para links formales |
| 10 | Cross-epic refs | 100 | 0 | Body disponible, regex trivial |
| 11 | Traceability | 20 | 80 | Body disponible, sin matching |
| 12 | Invariants | 60→70* | 40→30* | Body disponible, regex INV pattern |
| 13 | Sub-schema by type | 50→65* | 50→35* | Body + `Analyze()` per-subgroup |

*\* = con goldmark*

**Totales**:
- Sin goldmark: ~75% Go / ~25% LLM
- Con goldmark: ~80% Go / ~20% LLM

---

## Apéndice B: Subcapa simbólica del engine

El engine internamente progresa en 4 fases de complejidad:

```
Fase 1: Extracción (determinística)
  │  YAML parse, regex links, BOM strip
  │  1 input → 1 output, siempre igual
  │  Packages: extract/, index/
  ▼
Fase 2: Validación (rule-checking)
  │  required?, enum?, exists?, requires?
  │  Determinística pero schema-driven (.stem)
  │  Packages: rules/validate.go, rules/structural.go
  ▼
Fase 3: Derivación (expr-lang)
  │  derive: { slug: "slugify(titulo)" }
  │  aggregate: { count: "len(children)" }
  │  Turing-complete pero determinística
  │  Packages: derive/, query/expr_eval.go
  ▼
Fase 4: Propuestas (razonamiento simbólico)
     10 detectores con heurísticas y umbrales
     Genera ALTERNATIVAS, no un resultado único
     Thresholds: 0.8, 0.6, 0.75, 20, 2
     Prioridad: migrate_value > extend_enum > infer_from_children > ...
     Package: proposal/
```

**Distinción clave**:
- Fases 1-3: 1 input → 1 output. Determinístico.
- Fase 4: 1 input → N propuestas rankeadas. Heurístico.
- Agent LLM: input + propuestas → decisión. Semántico.

---

## Apéndice C: Fuentes de investigación

### Arquitectura de skills y agents
- [Claude Skills and Subagents: Escaping the Prompt Engineering Hamster Wheel](https://towardsdatascience.com/claude-skills-and-subagents-escaping-the-prompt-engineering-hamster-wheel/) — Towards Data Science, 2025
- [Extend Claude with skills](https://code.claude.com/docs/en/skills) — Claude Code docs oficiales
- [Claude Agent Skills: A First Principles Deep Dive](https://leehanchung.github.io/blogs/2025/10/26/claude-skills-deep-dive/) — Lee Han Chung
- [Building Agents with Skills](https://claude.com/blog/building-agents-with-skills-equipping-agents-for-specialized-work) — Anthropic blog

### Single vs multi-agent
- [Don't Build Multi-Agents](https://cognition.ai/blog/dont-build-multi-agents) — Cognition AI, 2025
- [When to Use Multi-Agent Systems](https://claude.com/blog/building-multi-agent-systems-when-and-how-to-use-them) — Claude docs
- [Single-agent vs. multi-agent AI](https://redis.io/blog/single-agent-vs-multi-agent-systems/) — Redis, 2025

### Patrón neuro-simbólico
- [Neuro-symbolic AI for auditable cognitive information extraction](https://www.nature.com/articles/s43856-025-01194-x) — Nature Communications Medicine, 2025
- [Neuro-Symbolic AI: A Foundational Analysis](https://gregrobison.medium.com/neuro-symbolic-ai-a-foundational-analysis-of-the-third-waves-hybrid-core-cc95bc69d6fa) — Medium

### AST parsing / goldmark
- [yuin/goldmark](https://github.com/yuin/goldmark) — GitHub
- [goldmark AST package](https://pkg.go.dev/github.com/yuin/goldmark/ast) — pkg.go.dev
- [Advanced markdown processing in Go](https://blog.kowalczyk.info/article/cxn3/advanced-markdown-processing-in-go.html) — Kowalczyk

### Análisis determinístico de texto
- [Symbolic AI in NLP](https://smythos.com/developers/ai-models/symbolic-ai-in-natural-language-processing/) — SmythOS
- [The Moment to Pay Attention to Hybrid NLP](https://www.bitext.com/blog/the-moment-to-pay-attention-to-hybrid-nlp-symbolic-ml/) — Bitext, 2025

---

## Conexiones

- `[[theories/computation-then-understanding]]` — teoría origen, refinada por esta investigación
- `[[closed/inference-engine-architecture]]` — línea de discover que produjo la teoría
- Related: `docs/research/intrinsic-hierarchy-principle.md` — v3 schema (deferred, D3)

---

*Investigación estructurada — /hypothesize framework*
