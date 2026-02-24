---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T002: Agregar checkpoint review con /review

**Story**: [S002 Quality Gates](README.md)
**Contribuye a**: SKILL.md loop tiene checkpoint detection con /review

[[blocks:T001-add-security-review-step]]

## Contexto

El loop puede ejecutar decenas de tasks sin feedback de calidad de codigo. /review analiza diffs acumulados para detectar problemas de calidad. Un checkpoint cada N tasks o al cambiar de Story balancea costo vs feedback.

## Alcance

**In**:
1. Agregar paso de checkpoint detection despues del resumen de iteracion
2. Triggers (OR): a) Story context change (siguiente task de otra Story), b) Safety net (N tasks desde ultimo checkpoint, default 5), c) Loop interrumpido (usuario elige "Parar")
3. Al activar: calcular diff acumulado (`git diff <checkpoint_commit>..HEAD`), ejecutar `/review`, reportar findings, registrar nuevo checkpoint
4. Findings de /review son informativos (no bloquean el loop)
5. Agregar flags: `--checkpoint-interval N` (default 5), `--skip-reviews` (desactivar ambos gates)
6. Documentar flags en la seccion de opciones del loop

**Out**: No modificar security review (T001). No modificar resumen final (T003).

## Preserva

- INV1: El loop sigue funcionando con --skip-reviews
- Verificar: la seccion describe --skip-reviews como flag que desactiva quality gates

## Estado inicial esperado

- T001 completado: variables de estado y security review existen
- Loop tiene resumen de iteracion (paso 8) seguido de confirmacion (paso 9)

## Criterios de Aceptacion

- Existe paso de checkpoint detection despues de resumen de iteracion
- Documenta 3 triggers (story change, N tasks, loop interrumpido)
- Describe ejecucion de /review sobre diff acumulado
- Findings son informativos (explicitamente "no bloquean")
- Flags --checkpoint-interval y --skip-reviews documentados en seccion de opciones

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md`
