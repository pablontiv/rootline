---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Unify Validate and Doctor Commands

**Story**: [S004 Validate & Doctor Unification](README.md)
**Contribuye a**: validate absorbe doctor checks, doctor queda como alias deprecado

## Preserva

- INV1: Tests existentes siguen pasando
  - Verificar: `go test ./cmd/rootline/ -run "TestValidate|TestDoctor" -v`

## Contexto

Actualmente rootline tiene 3 comandos read-only + write con overlap:
- `validate` — valida documentos contra .stem schema
- `doctor` — 7 checks de salud del .stem (yaml-valid, scope-match, type-consistency, enum-values, rule-field-exists, field-override, aggregated-required)
- `fix` — detecta proposals y escribe correcciones

El patrón estándar en herramientas declarativas (Terraform, ESLint, Prettier) es 2 comandos: **check** (read-only) y **fix** (write). `doctor` y `validate` son ambos read-only pero con scope diferente: doctor opera sobre .stem files, validate sobre documentos. Unificarlos en un solo `validate` que sea el CI gate completo.

## Especificacion Tecnica

**Modificar**: `cmd/rootline/validate.go`

Agregar stem health checks como fase que corre antes de la validación de documentos, cuando el target es un directorio (no un archivo individual).

Checks que migran de `doctor.go`:
1. `yaml-valid` → ya implícito (validate parsea .stem)
2. `scope-match` → agregar como warning: scope patterns son coherentes entre niveles
3. `type-consistency` → agregar como error: campos con mismo nombre tienen tipos compatibles
4. `enum-values` → agregar como warning: valores de enum son subset/superset coherente
5. `rule-field-exists` → agregar como warning: campos referenciados en validate rules existen en schema
6. `field-override` → agregar como warning: child .stem no overridea tipo de campo parent
7. `aggregated-required` → agregar como warning: campo con aggregate debería tener required en niveles hoja

Estos checks corren sobre los `.stem` encontrados por walk-up discovery, NO sobre documentos individuales. Se ejecutan como primera fase en `runValidate()` cuando el argumento es un directorio.

**Modificar**: `cmd/rootline/doctor.go`

Convertir en alias de validate con deprecation warning:
```go
RunE: func(cmd *cobra.Command, args []string) error {
    fmt.Fprintln(cmd.ErrOrStderr(), "Warning: 'doctor' is deprecated, use 'validate' instead")
    return runValidate(cmd, args)
}
```

**Tests**:
- Mover tests de `doctor_test.go` a `validate_test.go` (renombrar `TestDoctor*` → `TestValidateStemHealth*`)
- En `doctor_test.go`: nuevo test `TestDoctorDeprecationWarning` que verifica el warning a stderr
- Verificar que `rootline validate dir/` produce output que incluye stem health checks

## Dependencias

- Ninguna directa — este task es independiente de S001/S002/S003

## Alcance

**In**:
1. Extraer lógica de checks de `doctor.go` a funciones reutilizables (o mover directamente a validate)
2. Integrar checks en `runValidate()` como fase pre-documentos para directorios
3. Convertir `doctor` en alias con deprecation warning
4. Migrar tests de doctor a validate, agregar test de deprecation

**Out**: No tocar `fix.go`, `proposal.go`, `init.go`, `migrate.go`.

## Estado inicial esperado

- `cmd/rootline/doctor.go` tiene 7 checks funcionales
- `cmd/rootline/validate.go` tiene `runValidate()` funcional
- `cmd/rootline/doctor_test.go` tiene tests para los 7 checks

## Criterios de Aceptacion

- `go test ./cmd/rootline/ -run TestValidateStemHealth -v` pasa (7 checks migrados)
- `go test ./cmd/rootline/ -run TestDoctor -v` pasa (deprecation warning test)
- `rootline validate docs/epics/` muestra stem health checks + document validation
- `rootline doctor docs/epics/` emite "Warning: 'doctor' is deprecated, use 'validate' instead" a stderr y produce misma output que validate

## Fuente de verdad

- `cmd/rootline/doctor.go` — 7 checks, runDoctor function
- `cmd/rootline/validate.go` — runValidate function
- `cmd/rootline/doctor_test.go` — tests existentes de los 7 checks
- `cmd/rootline/validate_test.go` — tests existentes de validación
