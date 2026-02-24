---
estado: Specified
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T003: Agregar validación pre-push del marketplace

**Story**: [S002 Cross-Repo Sync Pipeline](README.md)

## Contexto

Antes de pushear al agent-marketplace, el workflow debe verificar que la estructura resultante es válida: marketplace.json es JSON válido, todos los skills referenciados existen con SKILL.md, frontmatter tiene campos requeridos (name, description). Esto previene publicar un marketplace roto.

## Alcance

**In**:
1. Step de validación después del rsync y antes del push
2. Verificar marketplace.json es JSON válido (jq)
3. Verificar que cada skill directory contiene SKILL.md
4. Verificar que cada SKILL.md tiene frontmatter con `name` y `description`
5. Fail del workflow si validación falla (no pushear marketplace roto)

**Out**: Validación semántica de contenido de skills, rootline validate

## Estado inicial esperado

- T001 completado: workflow base funcional
- Marketplace tiene estructura con marketplace.json y skills/

## Criterios de Aceptacion

- Workflow falla si marketplace.json no es JSON válido
- Workflow falla si un skill directory no tiene SKILL.md
- Workflow falla si un SKILL.md no tiene frontmatter name/description
- Workflow pasa y pushea cuando estructura es válida
- Error message claro indicando qué falló la validación

## Fuente de verdad

- `.github/workflows/publish-marketplace.yml` (workflow a modificar)
- skills.sh spec (campos requeridos de frontmatter)
