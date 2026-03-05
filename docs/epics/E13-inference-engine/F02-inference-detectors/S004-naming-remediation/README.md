---
estado: Specified
tipo: historia
---
# S004: Naming Remediation — Eliminar terminología de investigación

**Feature**: [F02 Inference Category Expansion](../README.md)
**Capacidad**: Código y docs usan vocabulario autodescriptivo del dominio de implementación
**Cubre**: Calidad de código — nombres comunican qué hacen sin consultar docs externos

## Antes / Despues

**Antes**: Archivos Go se llaman `cat5.go`, `cat7.go`. Task files usan slugs como `T001-cat-5-link-validation.md`. Struct Inference tiene campo `Category int`. La terminología "Cat N" viene del documento de investigación y no es autodescriptiva en el código.

**Despues**: Archivos Go se llaman `link_validation.go`, `back_references.go`. Task files usan slugs descriptivos. Struct Inference no tiene campo `Category`. Directorios del roadmap usan nombres de producto (`F02-inference-detectors`, `S001-structural-detectors`, `S003-semantic-extraction`). Cero referencias a "Cat N", "porciones Go/LLM", porcentajes de research, o teorías de investigación en código ni documentos de producto.

## Criterios de Aceptacion (semanticos)

- [ ] Ningún archivo Go en `internal/infer/` contiene "cat" como prefijo de nombre
- [ ] Struct Inference no tiene campo `Category int`
- [ ] Task files en F02 no contienen "Cat N" en slugs
- [ ] Cero comentarios o dirs con "category" en código Go
- [ ] Directorios del roadmap no usan nomenclatura de investigación
- [ ] Cero referencias "Cat N", "porciones Go/LLM", porcentajes research en docs de producto
- [ ] `go test ./... -race` pasa verde
- [ ] `rootline validate --all docs/epics/E13-inference-engine/` pasa

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-rename-go-files.md) | Renombrar archivos Go de catN a nombres descriptivos |
| [T002](T002-remove-category-field.md) | Eliminar campo Category del struct Inference |
| [T003](T003-rename-task-files.md) | Renombrar task files y limpiar referencias Cat N en docs |
| [T004](T004-fix-go-residuals.md) | Fix Go code residuals: comentario y testdata dir |
| [T005](T005-rename-roadmap-dirs.md) | Renombrar directorios del roadmap con nombres de producto |
| [T006](T006-replace-research-terms.md) | Reemplazar terminología de investigación en docs del roadmap |
