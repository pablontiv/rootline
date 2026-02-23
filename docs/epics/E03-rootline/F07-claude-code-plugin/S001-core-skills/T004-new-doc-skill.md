---
estado: Completado
tipo: documentation
ejecutable_en: 1 sesion
---
# T004: Implementar skill /new-doc

**Story**: [S001 Core Skills](README.md)

[[blocks:T001-plugin-scaffold]]

## Contexto

El skill /new-doc wrappea `rootline new` para scaffold de documentos nuevos con frontmatter correcto segun el .stem del directorio. Usa auto-numbering via `rootline describe --field schema.id.next` para generar el proximo ID automaticamente.

## Alcance

**In**:
1. `claude-plugin/skills/new-doc/SKILL.md` con instrucciones completas
2. Trigger phrases: "new doc", "crear documento", "scaffold"
3. Si no se da path, preguntar directorio destino
4. Ejecutar `rootline describe <dir> --field schema.id.next` para auto-numbering
5. Ejecutar `rootline new <path>` para crear archivo
6. Ofrecer ejecutar /validate inmediatamente

**Out**: Template selection, batch creation, interactive field filling

## Estado inicial esperado

- Plugin scaffold (T001) completado
- rootline new funcional
- rootline describe --field schema.id.next funcional

## Criterios de Aceptacion

- SKILL.md existe en `claude-plugin/skills/new-doc/`
- Skill obtiene next ID automaticamente
- Skill crea archivo con frontmatter correcto via rootline new
- Archivo creado pasa rootline validate

## Fuente de verdad

- `cmd/rootline/new.go` (scaffolding logic)
- `cmd/rootline/describe.go` (auto-numbering via schema.id.next)
