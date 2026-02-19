# S002: Validate Command

**Feature**: [F03 Validation and Schema](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: `rootline validate` valida documentos contra .stem schemas desde la linea de comandos

## Antes / Despues

**Antes**: No existe comando de validacion. Las verificaciones son inline en skills (grep) o prompt-based (LLM agent). No hay forma de validar un documento desde terminal.

**Despues**: `rootline validate <file>` valida un archivo. `rootline validate --all` valida todos los archivos en scope. Output JSON con errores tipados. Exit code 1 si hay errores.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline validate docs/prd/PRD-001.md` reporta errores de schema
- [ ] `rootline validate --all` valida todos los archivos .md en scope
- [ ] Exit code 0 para documentos validos, 1 para invalidos

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-validate-command.md) | Implementar cobra command `rootline validate` |

## Fuente de verdad

- `src/rootline/docs/intent/v0-rootline.md` seccion 3 (Commands table)
