# S001: Validation Engine

**Feature**: [F03 Validation and Schema](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Motor de reglas deterministico valida frontmatter contra schema .stem con source tracing

## Antes / Despues

**Antes**: Validacion es prompt-based (LLM) o inline grep. No determinista, no testeable. El Write hook usa un agente LLM para validar estructura de Tasks — no se puede reproducir ni garantizar consistencia.

**Despues**: Rules engine chequea Record.Frontmatter contra el schema efectivo del .stem. 4 reglas built-in (non_empty, enum, requires, exists). Produce JSON con errores tipados incluyendo source (cual .stem definio la regla). Deterministico, testeable, reproducible.

## Criterios de Aceptacion (semanticos)

- [ ] Documento con campo enum invalido produce error con regla y source
- [ ] Documento con campo required faltante produce error
- [ ] Validacion es 100% determinista (mismo input = mismo output)

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-rules-engine.md) | Implementar 4 reglas de validacion |
| [T002](T002-validation-output-format.md) | Definir JSON output contract para validation results |

## Fuente de verdad

- `src/rootline/docs/research/I5-describe-contract.md` seccion 2.4 (Validate Section)
