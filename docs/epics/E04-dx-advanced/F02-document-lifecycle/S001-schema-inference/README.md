# S001: Schema Inference

**Feature**: [F02 Document Lifecycle](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Rootline escanea archivos Markdown existentes, detecta patrones de frontmatter, e infiere un archivo .stem con campos, tipos, y valores enum

## Antes / Despues

**Antes**: Escribir un archivo .stem desde cero requiere conocer la sintaxis YAML del schema, analizar manualmente que campos usan los archivos existentes, identificar valores repetidos para enums, y decidir que campos son required. Es la mayor barrera de adopcion.

**Despues**: `rootline init docs/prd/` escanea todos los .md del directorio, detecta que 12 de 15 archivos tienen campo "estado" con valores "Pending", "Completado", "Especificado" (enum detectado), que "titulo" aparece en todos (required inferido), y genera un .stem con schema completo. `--dry-run` muestra sin escribir.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline init` escanea archivos .md y detecta campos comunes de frontmatter
- [ ] Campos con valores discretos repetidos se infieren como enum
- [ ] Campos presentes en >80% de archivos se marcan como required
- [ ] El .stem generado es YAML valido parseable por rootline

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-field-analyzer.md) | Paquete de analisis de frecuencia de campos y deteccion de tipos |
| [T002](T002-init-command.md) | Comando init que genera .stem desde analisis |

## Fuente de verdad

- `internal/extract/` — Record, Registry
- `internal/index/index.go` — Scan
- `internal/rules/rules.go` — StemFile struct (target output)
