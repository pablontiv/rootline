---
estado: Deferred
fecha: "2026-02-20"
metodo: agent-team-research
---
# I3 Research: Derivation Engine — Decision Record

---

## 1. Decision

**Diferido**. F04 (Derivation Engine) queda en estado Diferida hasta que haya evidencia de demanda real de campos derivados por parte de usuarios.

**Razón**: Rootline ya tiene validation + query completos. `derive:` se parsea en .stem pero es 100% no-op. Zero archivos .stem en el repo usan `derive:`. La research original (Feb 2026) recomendaba diferir hasta tener "evidence from real usage patterns" — esa evidencia aún no existe.

**Motor recomendado cuando se retome**: `expr-lang/expr` (v1.17+). Zero deps, non-Turing complete, 70ns/op, probado en Google/Uber/Argo.

---

## 2. Contexto de la Investigación

Investigación conducida por team de 4 agentes (Feb 2026):
- **needs-analyst**: Analizó qué necesita rootline del codebase actual
- **expr-evaluator**: Deep-dive en expr-lang/expr API, safety, adoption
- **alternatives-evaluator**: Comparó CEL, Starlark, Rego, go-native
- **critic**: Cuestionó si se necesita un engine en absoluto

---

## 3. Hallazgos Clave

### Estado actual del codebase
- `StemFile.Derive` es `map[string]any` en `internal/rules/rules.go:21`
- Se parsea via YAML unmarshal pero no se evalúa en ningún lugar
- Pipeline slot reservado: Extraction → Validation → **[DERIVATION]** → Query
- `cmd/rootline/explain.go` es stub
- `tree.go` ya hace derivación hardcodeada (cuenta estados, calcula completados/total)

### Derivation Levels

| Level | Ejemplo | Necesario v1? |
|-------|---------|---------------|
| 1. Field transform | `slug = slugify(title)` | Posible, pero nadie lo ha pedido |
| 2. Cross-record aggregation | `progress = count(done) / total` | Parcial — requiere contexto de hijos que el pipeline no provee aún |
| 3. Link traversal | `blocked = any(links.blocks, .status != "done")` | No — F05 (links) no implementado |
| 4. Recursive | `health = all(descendants, .has_owner)` | No — requiere graph traversal |

### Dependencia F04 ↔ F05

F05 (Dependency Graph) declaraba dependencia en F04 para "estado derivado por propagación". Análisis demuestra que F05 core (link extraction, validation, graph, cycle detection) NO necesita F04. La propagación de estado es Level 3 derivation y requiere ambos features. Dependencia corregida.

---

## 4. Evaluación de Engines

| Engine | Size | Deps | YAML inline | Safety | Performance | Verdict |
|--------|------|------|-------------|--------|-------------|---------|
| **Expr** | 3 MB | 0 | Excelente | Non-Turing, sandboxed | 70 ns/op | **Recomendado** |
| CEL | 15+ MB | Protobuf (heavy) | Awkward | Non-Turing | 91 ns/op | Solo si K8s integration |
| Starlark | 8-10 MB | 0 | Multi-block only | Bounded (SetMaxSteps) | Interpreter | Solo Architecture A |
| Rego | 20+ MB | 15 transitivas | Multi-block only | Non-Turing | Heavy | No (oversized) |
| Go-native | 0 MB | 0 | Pure YAML | Full control | N/A | Alternativa keyword |

### Expr-lang/expr (Recomendado)
- v1.17.7 (Dec 2025), actively maintained
- Builtins: `lower`, `upper`, `trim`, `replace`, `len`, `any`, `all`, `filter`, `map`, `split`, `contains`, `startsWith`, `endsWith`
- Custom functions via `expr.Function()` (para slugify, etc.)
- Compile-time type checking, bytecode VM
- Security advisory GHSA-93mq-9ffx-83m2 (parser memory, fixed v1.17.0)
- Adoption: Google Cloud, Uber, Argo Workflows, ByteDance, GoDaddy

### Alternativa: Keywords Declarativos (go-native)

Si se necesita MVP minimal sin engine externo:
```yaml
derive:
  slug:
    fn: slugify
    from: title
  status_lower:
    fn: lower
    from: estado
```
Zero deps, scope limitado by design, mejor UX en YAML. Limitado a Level 1.

---

## 5. Argumentos del Critic

1. **`tree.go` ya hace derivación** — cuenta estados, calcula ratios. Es derivación en Go.
2. **CLAUDE.md dice** "convenience, not blocking" sobre derivación.
3. **Inline expressions en YAML** = UX cuestionable (escaping, no IDE support, multiline ugly).
4. **Scope creep**: shipear Expr invita presión por Levels 3-4 antes de validar 1-2.
5. **Keyword system cubre 80%** de los casos sin engine externo.

---

## 6. Arquitecturas

### Architecture A: Single Language (Starlark/Rego)
- Archivo separado `.derive.star` o `.derive.rego`
- Turing-complete, puede expresar todo
- Risk: complejidad excesiva para L1-2

### Architecture B: Engine + Expression (Expr inline)
- Expressions inline en .stem YAML: `expr: "lower(replace(title, ' ', '-'))"`
- Rootline controla traversal, Expr evalúa hojas
- Pro: coherente con inheritance model de rootline
- Con: dos conceptos (engine + expression syntax)

### Architecture C: Keywords declarativos (go-native)
- YAML puro: `{ fn: slugify, from: title }`
- Zero deps, scope forzado a L1-2
- Pro: simple, no UX issues
- Con: cada función nueva = código Go

**Decisión diferida** — cuando se retome, empezar con Architecture B (Expr) o C (keywords) según complejidad de los casos de uso reales.

---

## 7. Criterios para Reactivar

Reactivar F04 cuando:
1. Usuarios pidan campos derivados en .stem (demanda real)
2. Se identifiquen > 3 expresiones concretas que no se pueden resolver con query/tree existentes
3. F05 (links) esté implementado (habilita Level 3 use cases)

---

## 8. References

- Expr: https://expr-lang.org/, https://github.com/expr-lang/expr
- CEL: https://cel.dev/
- Starlark: https://github.com/google/starlark-go
- OPA/Rego: https://www.openpolicyagent.org/
- Expr security: https://github.com/expr-lang/expr/security/advisories/GHSA-93mq-9ffx-83m2
