---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T003: Migrar tasks existentes de blocks frontmatter a wiki-links

**Story**: [S002 Dependency Wiki-Links](README.md)

[[blocks:T001-add-blocks-schema-to-stem]]

## Contexto

5 tasks en F09 declaran dependencias usando `blocks:` en frontmatter YAML. Con el nuevo modelo de wiki-links, estas dependencias deben migrarse al body como `[[blocks:TXXX-name]]`. Esto permite que `rootline graph` las detecte nativamente.

## Dependencias

- T001 completado (links schema en .stem para que graph valide)

## Alcance

**In**:
1. Para cada uno de los 5 archivos:
   - Quitar `blocks:` y su contenido del frontmatter YAML
   - Agregar `[[blocks:TXXX-name]]` en la seccion Contexto del body (debajo del titulo o al inicio del contexto)
2. Archivos a migrar:
   - `S001/T002-implement-validate-directory.md` — `blocks: [T001-extend-stemfile-structural-types]`
   - `S001/T003-integrate-structural-into-validate-all.md` — `blocks: [T002-implement-validate-directory]`
   - `S002/T002-validate-dependency-targets-exist.md` — `blocks: [T001-add-blocks-schema-to-stem]`
   - `S002/T003-integrate-deps-with-graph-check.md` — `blocks: [T002-validate-dependency-targets-exist]`
   - `S003/T002-update-loop-dependency-order.md` — `blocks: [T001-update-task-guide-with-blocks]`
3. Verificar con `rootline validate` que cada archivo migrado es valido
4. Verificar con `rootline graph --check` que las dependencias se detectan

**Out**: No migrar archivos fuera de F09. No crear archivos nuevos.

## Estado inicial esperado

- 5 archivos tienen `blocks:` en frontmatter YAML
- `rootline graph` no detecta estas dependencias (frontmatter no genera edges)

## Criterios de Aceptacion

- Ninguno de los 5 archivos tiene `blocks:` en frontmatter
- Cada archivo tiene `[[blocks:TXXX-name]]` en el body
- `rootline validate --all docs/epics/E04-dx-advanced/F09-planning-structure-validation/` pasa sin errores
- `rootline graph docs/epics/E04-dx-advanced/F09-planning-structure-validation/ --format mermaid` muestra edges (no edges vacios)
- `rootline graph docs/epics/E04-dx-advanced/F09-planning-structure-validation/ --check` pasa sin ciclos ni broken links

## Fuente de verdad

- Los 5 archivos listados en Alcance
- `docs/epics/.stem` — links schema (de T001)
