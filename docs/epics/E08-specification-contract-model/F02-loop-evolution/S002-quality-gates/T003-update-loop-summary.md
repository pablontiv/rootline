---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T003: Actualizar resumen final + verificar invariantes en loop

**Story**: [S002 Quality Gates](README.md)
**Contribuye a**: Resumen final incluye metricas de reviews; paso 6 verifica invariantes

[[blocks:T002-add-checkpoint-review]]

## Contexto

El resumen final del loop (Fase 4) solo reporta tasks completadas, saltadas, ACs y commits. Con los quality gates agregados, debe incluir metricas de security reviews y checkpoints. Ademas, el paso de verificacion de ACs debe tambien verificar invariantes (seccion Preserva del task).

## Alcance

**In**:
1. Actualizar Fase 4 resumen final para incluir: security reviews ejecutados + findings (H/M), quality checkpoints ejecutados + findings
2. Modificar paso de verificacion de ACs para tambien leer seccion "Preserva" del task y ejecutar cada invariante
3. Reportar invariantes: INV1 HOLDS / INV2 VIOLATED
4. Si algun invariante se viola → parar (igual que AC fail)

**Out**: No agregar nuevos pasos. No modificar otros subcomandos.

## Preserva

- INV1: El resumen final sigue reportando las metricas existentes (tasks, ACs, commits)
- Verificar: las lineas existentes del resumen no se eliminan, solo se agregan nuevas

## Estado inicial esperado

- T001 y T002 completados: security review y checkpoint review existen
- Fase 4 tiene formato de resumen con: Tasks completadas, saltadas, ACs, Commits, restantes
- Paso 6 verifica solo ACs

## Criterios de Aceptacion

- Resumen final tiene lineas para: security reviews (N ejecutados, M findings H/M), quality checkpoints (N ejecutados, M findings)
- Paso de verificacion menciona leer seccion "Preserva" del task
- Paso de verificacion reporta invariantes como HOLDS/VIOLATED
- Invariante violado produce parada (documentado)

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md`
