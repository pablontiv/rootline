# S002: Git Integration

**Feature**: [F03 Validation Evolution](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Rootline valida automaticamente documentos staged en git antes de cada commit via hooks instalables

## Antes / Despues

**Antes**: La validacion solo se ejecuta manualmente con `rootline validate`. Documentos invalidos pueden llegar al repositorio sin deteccion. No hay integracion con el workflow de git.

**Despues**: `rootline hooks install` escribe un script pre-commit en `.git/hooks/` que ejecuta `rootline validate --staged`. Solo archivos .md en staging area se validan. Si hay errores, el commit se bloquea. `rootline hooks uninstall` remueve el hook. `rootline hooks status` muestra si el hook esta instalado.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline hooks install` crea pre-commit hook funcional
- [ ] Pre-commit hook solo valida archivos .md en staging area
- [ ] Commit se bloquea si hay errores de validacion
- [ ] `rootline hooks uninstall` remueve el hook limpiamente

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-hooks-command.md) | Comando hooks con install/uninstall/status |
| [T002](T002-staged-validation.md) | Flag --staged en validate para filtrar archivos en staging |

## Fuente de verdad

- `cmd/rootline/validate.go` — validate command
- `.git/hooks/` — directorio de hooks de git
