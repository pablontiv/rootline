---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar rootline migrate --rename

**Story**: [S002 Bulk Operations](README.md)

## Contexto

Renombrar un campo en un .stem es un breaking change que invalida todos los documentos existentes. Esta task implementa `rootline migrate --rename old_field=new_field [path]` que actualiza el frontmatter de todos los documentos afectados y el .stem schema atomicamente.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline + internal/migrate
interfaces:
  - nombre: RenameOperation
    metodos:
      - nombre: Execute
        input: "oldField, newField, path string, dryRun bool"
        output: "RenameResult, error"
dependencias_externas: []
tests:
  - Rename campo existente → N archivos actualizados
  - Rename campo inexistente → "no files affected"
  - Dry-run → reporta sin modificar
  - .stem schema actualizado con nuevo nombre
  - Archivo sin el campo → no modificado
```

## Dependencias

- S001 completado (migrate command infrastructure)
- internal/proposal/ (rewriteFrontmatter)

## Alcance

**In**:
1. Flag `--rename old=new` en rootline migrate command
2. Scan records que tienen old_field en frontmatter
3. Para cada record: rename key en frontmatter, rewrite file (reusar rewriteFrontmatter de proposal)
4. Actualizar .stem: rename schema key (AST node rename, similar a addEnumValueToNode pero para keys)
5. `--dry-run` reporta archivos que se modificarian
6. Output: lista de archivos modificados + conteo

**Out**: Bulk type changes, bulk value remapping, undo/rollback

## Estado inicial esperado

- migrate command de S001
- rewriteFrontmatter en internal/proposal/
- addEnumValueToNode pattern en cmd/rootline/fix.go

## Criterios de Aceptacion

- `rootline migrate --rename titulo=title docs/epics/` actualiza todos los archivos con campo "titulo"
- .stem schema.titulo renombrado a schema.title
- `--dry-run` muestra archivos sin modificar
- `rootline validate --all` pasa despues de rename
- `go test ./internal/migrate/ -run TestRename -v` pasa

## Fuente de verdad

- `internal/proposal/proposal.go` (rewriteFrontmatter)
- `cmd/rootline/fix.go` (addEnumValueToNode — pattern para AST modification)
- `internal/rules/rules.go` (StemFile schema structure)
