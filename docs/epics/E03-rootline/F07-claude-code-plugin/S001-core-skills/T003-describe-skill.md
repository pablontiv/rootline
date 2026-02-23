---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T003: Implementar skill /describe

**Story**: [S001 Core Skills](README.md)

[[blocks:T001-plugin-scaffold]]

## Contexto

El skill /describe wrappea `rootline describe` para mostrar el schema efectivo de un directorio. Esto permite al LLM y al developer entender que campos son requeridos, que enums son validos, y de que .stem file hereda cada regla.

## Alcance

**In**:
1. `claude-plugin/skills/describe/SKILL.md` con instrucciones completas
2. Trigger phrases: "describe schema", "show schema", "que campos necesito"
3. Ejecutar `rootline describe <dir> --output json` via Bash tool
4. Renderizar schema como tabla markdown: campo, tipo, requerido, valores enum, origen (.stem)

**Out**: Schema editing, .stem modification, interactive schema exploration

## Estado inicial esperado

- Plugin scaffold (T001) completado
- rootline describe funcional con --output json

## Criterios de Aceptacion

- SKILL.md existe en `claude-plugin/skills/describe/`
- Skill ejecuta rootline describe y renderiza schema como tabla
- Tabla incluye: campo, tipo, required, enum values, .stem source

## Fuente de verdad

- `cmd/rootline/describe.go` (flags y output format)
