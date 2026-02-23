---
estado: Pending
tipo: documentation
ejecutable_en: 1 sesion
---
# T002: Implementar skill /validate

**Story**: [S001 Core Skills](README.md)

[[blocks:T001-plugin-scaffold]]

## Contexto

El skill /validate wrappea `rootline validate` para validar documentos contra su .stem schema. Smart default: si se da un archivo, valida ese archivo; si se da un directorio o nada, usa --all. Presenta errores formateados para consumo del LLM.

## Alcance

**In**:
1. `claude-plugin/skills/validate/SKILL.md` con instrucciones completas
2. Trigger phrases: "validate", "check schema", "validar"
3. Ejecutar `rootline validate <path> --output json` via Bash tool
4. Parsear JSON y presentar errores como lista legible
5. Si no hay errores, confirmar documento valido

**Out**: Auto-fix (eso es /fix futuro), watch mode, batch validation de multiples paths

## Estado inicial esperado

- Plugin scaffold (T001) completado
- rootline validate funcional con --output json

## Criterios de Aceptacion

- SKILL.md existe en `claude-plugin/skills/validate/`
- Skill ejecuta rootline validate y presenta errores formateados
- Archivo valido reporta "sin errores"
- Archivo invalido lista cada error con path y campo

## Fuente de verdad

- `cmd/rootline/validate.go` (flags y output format)
- `internal/rules/result.go` (JSON schema de ValidationResult)
