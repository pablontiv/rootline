---
tipo: historia
cliente: Platform Owner
---
# S001: Core Skills

**Feature**: [F07 Claude Code Plugin](../README.md)
**Capacidad**: Claude Code tiene skills /validate, /describe, /new-doc que wrappean rootline CLI para AI-assisted document authorship

## Antes / Despues

**Antes**: Developer usa rootline CLI manualmente. Claude Code no conoce rootline ni sus schemas. Crear documentos validos requiere leer .stem files y armar frontmatter a mano.

**Despues**: `/validate` valida el archivo actual o directorio. `/describe` muestra el schema efectivo. `/new-doc` scaffoldea documento con frontmatter correcto y auto-numbering. Plugin distribuible que cualquier proyecto con .stem puede instalar.

## Criterios de Aceptacion (semanticos)

- [ ] Plugin scaffold con plugin.json valido
- [ ] /validate ejecuta rootline validate y presenta errores legiblemente
- [ ] /describe muestra schema efectivo como tabla markdown
- [ ] /new-doc crea documento con frontmatter correcto segun .stem

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-plugin-scaffold.md) | Crear plugin scaffold con plugin.json y estructura |
| [T002](T002-validate-skill.md) | Implementar skill /validate |
| [T003](T003-describe-skill.md) | Implementar skill /describe |
| [T004](T004-new-doc-skill.md) | Implementar skill /new-doc |

## Fuente de verdad

- `.claude/skills/roadmap/` (patron de skill existente)
- Claude Code plugin docs (plugin.json schema)
