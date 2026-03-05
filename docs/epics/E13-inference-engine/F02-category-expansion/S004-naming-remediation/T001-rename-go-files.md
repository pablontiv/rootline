---
estado: Completed
tipo: refactor
ejecutable_en: 1 sesion
---
# T001: Renombrar archivos Go de catN a nombres descriptivos

**Story**: [S004 Naming Remediation](README.md)
**Contribuye a**: Ningun archivo Go en `internal/infer/` contiene "cat" como prefijo de nombre

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage >=85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Los archivos Go en `internal/infer/` usan nombres del dominio logico de investigacion (`cat5.go`, `cat7.go`, etc.) que no comunican que hacen sin consultar documentacion externa. Las funciones ya tienen nombres descriptivos (`DetectLinkTypes`, `DetectMissingBackReferences`), pero los archivos que las contienen no. Tambien hay comentarios que referencian "Category N" y nombres de test functions con "Cat" como prefijo.

## Alcance

**In**:
1. `git mv` de 8 archivos:
   - `cat5.go` -> `link_validation.go`
   - `cat5_test.go` -> `link_validation_test.go`
   - `cat7.go` -> `back_references.go`
   - `cat7_test.go` -> `back_references_test.go`
   - `cat8.go` -> `constant_fields.go`
   - `cat8_test.go` -> `constant_fields_test.go`
   - `cat10.go` -> `cross_references.go`
   - `cat10_test.go` -> `cross_references_test.go`
2. Actualizar comentarios de funcion:
   - `// DetectLinkTypes implements Category 5: link-type validation.` -> `// DetectLinkTypes validates observed link types against a schema's allowed list.`
   - `// DetectMissingBackReferences implements Category 7: back-reference consistency.` -> `// DetectMissingBackReferences checks that bidirectional links have reciprocal references.`
   - `// DetectConstantFields implements Category 8: constant field detection.` -> `// DetectConstantFields finds fields with identical values across all records.`
   - `// DetectCrossReferences implements Category 10: cross-epic path reference extraction.` -> `// DetectCrossReferences extracts hierarchical path references from body text and validates they exist.`
3. Renombrar test functions en `integration_test.go`:
   - `TestIntegrationCat5WithFixtures` -> `TestIntegrationLinkValidationWithFixtures`
   - `TestIntegrationCat7WithFixtures` -> `TestIntegrationBackReferencesWithFixtures`
   - `TestIntegrationCat8WithFixtures` -> `TestIntegrationConstantFieldsWithFixtures`
   - `TestIntegrationCat10WithFixtures` -> `TestIntegrationCrossReferencesWithFixtures`
4. Actualizar error messages en `integration_test.go`:
   - `"Cat5: expected..."` -> `"LinkValidation: expected..."`
   - `"Cat7: expected..."` -> `"BackReferences: expected..."`
   - `"Cat8: expected..."` -> `"ConstantFields: expected..."`
   - `"Cat10: expected..."` -> `"CrossReferences: expected..."`
5. Renombrar variable en `cross_references.go`:
   - `crossRefRe` -> `hierarchicalPathRe` (y su comentario)

**Out**: No modificar logica de codigo. No eliminar campo Category (eso es T002). No renombrar task files del roadmap (eso es T003).

## Estado inicial esperado

- 8 archivos `cat*.go` existen en `internal/infer/`
- `integration_test.go` existe con test functions `TestIntegrationCat*`
- Todos los tests pasan verde

## Criterios de Aceptacion

- `ls internal/infer/cat*.go 2>/dev/null | wc -l` retorna 0
- `ls internal/infer/link_validation.go internal/infer/back_references.go internal/infer/constant_fields.go internal/infer/cross_references.go` retorna 0 (exit code)
- `grep -r "Category [0-9]" internal/infer/*.go | wc -l` retorna 0
- `grep "TestIntegrationCat" internal/infer/integration_test.go | wc -l` retorna 0
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/infer/cat5.go`, `cat7.go`, `cat8.go`, `cat10.go` — archivos a renombrar
- `internal/infer/cat5_test.go`, `cat7_test.go`, `cat8_test.go`, `cat10_test.go` — tests a renombrar
- `internal/infer/integration_test.go` — test functions y error messages a actualizar
