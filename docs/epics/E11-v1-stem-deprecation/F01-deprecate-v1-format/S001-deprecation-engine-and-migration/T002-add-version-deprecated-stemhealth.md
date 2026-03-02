---
estado: Specified
tipo: software-module
ejecutable_en: rootline
---
# T002: Add `version-deprecated` Stem Health Check

**Story**: [S001 V1 Deprecation Engine & Migration](README.md)
**Contribuye a**: Detección: `rootline validate --all docs/` muestra `version-deprecated` para stems v1
**Preserva**: INV1 (tests existentes pasan), INV2 (pipeline verde)

## Contexto

`internal/rules/stemhealth.go` tiene 7 checks que se ejecutan como Phase 1 de `validate --all`. Se necesita Check 8: `version-deprecated` que emita un warning para stems con version 0 o 1.

Patrón existente: cada check itera `parsedStems`, crea `StemHealthCheck{Name, Status, Message, Path}`, y lo appenda a `checks`.

## Alcance

**In scope**:
- Agregar Check 8 en `ValidateStemHealth()` después del bloque de Check 7 (~line 252)
- Name: `"version-deprecated"`, Status: `"warn"`
- Message: `"stem uses deprecated v1 format — run: rootline migrate --to-v2"`
- Condición: `stem.Version == 1 || stem.Version == 0`
- Tests: `TestValidateStemHealth_VersionDeprecated` + `TestValidateStemHealth_V2NoWarning`

**Out of scope**: Hacer version-deprecated un error (solo warn)

## Especificacion Tecnica

```yaml
archivo: internal/rules/stemhealth.go
insertar_despues: "Check 7: Aggregated required fields" (~line 252)
codigo: |
  // Check 8: Deprecated v1 format
  for sf, stem := range parsedStems {
      if stem.Version == 1 || stem.Version == 0 {
          relPath, _ := filepath.Rel(absRoot, sf)
          checks = append(checks, StemHealthCheck{
              Name:    "version-deprecated",
              Status:  "warn",
              Message: "stem uses deprecated v1 format — run: rootline migrate --to-v2",
              Path:    relPath,
          })
      }
  }
tests: internal/rules/stemhealth_test.go
```

## Criterios de Aceptación

- [ ] `go test ./internal/rules/ -run TestValidateStemHealth_VersionDeprecated -v` pasa
- [ ] `go test ./internal/rules/ -run TestValidateStemHealth_V2NoWarning -v` pasa
- [ ] `go test ./... -race` pasa verde
