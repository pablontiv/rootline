---
estado: Completado
tipo: documentation
---
# T003: Configurar structural rules en docs/epics/.stem y crear READMEs faltantes

**Story**: [S003 Skill and Loop Integration](README.md)

## Contexto

Con S001 completada, rootline soporta bloque `structural:` en .stem files. Ahora se necesita configurar `docs/epics/.stem` con las reglas estructurales apropiadas para el arbol de planificacion, y crear los README.md faltantes en E03 y E04 para que la validacion pase.

## Dependencias

- S001 completada (structural rules implementadas en rootline)

## Alcance

**In**:
1. Agregar bloque `structural:` a `docs/epics/.stem`:
   ```yaml
   structural:
     subdirs:
       require_index: README.md
       min_children: 2
       severity: warn
   ```
2. Crear `docs/epics/E03-rootline/README.md` con template de epic (del epic-guide.md)
3. Crear `docs/epics/E04-dx-advanced/README.md` con template de epic
4. Ejecutar `rootline validate --all docs/epics/` para verificar
5. E03 seguira mostrando warning de min_children (tiene 1 feature) — esto es correcto y esperado

**Out**: No resolver el warning de E03 (1 feature). No renumerar features existentes. No reestructurar epics.

## Estado inicial esperado

- `docs/epics/.stem` no tiene bloque structural
- `docs/epics/E03-rootline/README.md` no existe
- `docs/epics/E04-dx-advanced/README.md` no existe

## Criterios de Aceptacion

- `docs/epics/.stem` tiene bloque `structural:` con require_index y min_children
- `docs/epics/E03-rootline/README.md` existe con template de epic valido
- `docs/epics/E04-dx-advanced/README.md` existe con template de epic valido
- `rootline validate --all docs/epics/` no reporta errores de require_index
- `rootline validate --all docs/epics/` reporta warning en E03 por min_children (1 < 2) — esperado
- Ambos READMEs pasan `rootline validate` (frontmatter valido contra schema)

## Fuente de verdad

- `docs/epics/.stem` — schema a modificar
- `.claude/skills/roadmap/epic-guide.md` — template de epic README
