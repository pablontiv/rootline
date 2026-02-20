---
estado: Completado
tipo: documentation
ejecutable_en: 1 sesion
---
# T002: Ampliar enum tipo en docs/epics/.stem

**Story**: [S001 CLI & Schema Readiness](README.md)

## Contexto

El enum `tipo` en `docs/epics/.stem` actualmente tiene 4 valores: software-module, ci-cd, feature, historia. El task-guide del skill /roadmap define 14 tipos de task (IaC: servicio-docker, modulo-sistema, operacion-sistema, lxc, vm, modulo-infraestructura, host-script, instance-script; Software: software-module, software-test, ci-cd; General: documentation; Framework: feature, historia). El .stem debe cubrir todos para que `rootline validate` y `rootline new` funcionen con cualquier tipo.

## Dependencias

- Ninguna

## Alcance

**In**:
1. Agregar 10 valores faltantes al enum `tipo` en `docs/epics/.stem`
2. Verificar que documentos existentes siguen pasando validacion
3. Verificar que `rootline new --dry-run` muestra todos los tipos

**Out**: Cambios a task-guide.md, nuevos tipos de task, cambios a reglas de validacion

## Estado inicial esperado

- `docs/epics/.stem` existe con enum tipo de 4 valores
- Documentos existentes usan solo software-module, ci-cd, feature, historia

## Criterios de Aceptacion

- `docs/epics/.stem` contiene 14 valores en el enum tipo
- `rootline validate --all docs/epics/` pasa sin errores
- `rootline new docs/epics/E04-dx-advanced/F06-rootline-skill-backend/S001-cli-schema-readiness/T099-test.md --dry-run` muestra los 14 tipos en comment
- Orden de valores: IaC primero, luego software, luego general, luego framework

## Fuente de verdad

- `docs/epics/.stem` — archivo a modificar
- `.claude/skills/roadmap/task-guide.md` — definicion canonica de los 14 tipos
