---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T004: Agregar --where flag a validate --all

**Story**: [S001 Core Implementation](README.md)

[[blocks:T001-shared-filter-helper]]

## Contexto

El comando `validate` en `cmd/rootline/validate.go` tiene dos modos: single-file y `--all` (batch). El modo `--all` (`runValidateAll()`) hace scan → derivar → validar cada record → reportar. El filtrado debe insertarse en `runValidateAll()` despues de scan+derivacion, antes de validacion, usando `filterRecords()` de T001. Esto permite validar solo un subset de records (ej: solo tasks pendientes).

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: validateCmd
    metodos:
      - nombre: runValidateAll
        input: "cmd *cobra.Command, args []string"
        output: error
dependencias_externas: []
tests:
  - validate --all --where filtra records antes de validar
  - validate --all --where con expresion invalida retorna error
  - validate --all sin --where funciona igual que antes
  - validate single-file ignora --where (no aplica)
```

## Dependencias

- T001 completado (filterRecords helper disponible)

## Alcance

**In**:
1. Agregar flag `--where` (StringArrayVar) al validate command
2. En `runValidateAll()`, llamar `filterRecords(records, wheres)` despues de scan+derivacion
3. Agregar tests en `validate_test.go` para --where filtering en modo --all

**Out**: Filtrado en single-file mode (no tiene sentido), documentacion (S002)

## Estado inicial esperado

- T001 completado (filter.go con filterRecords)
- `cmd/rootline/validate.go` sin --where flag

## Criterios de Aceptacion

- `rootline validate --all docs/epics/ --where "estado == 'Specified'" -o table` valida solo records con estado Specified
- `rootline validate --all docs/epics/ --where "tipo == 'software-module'"` valida solo software-module records
- `go test ./cmd/rootline/ -run TestValidate -v` pasa
- `go build ./cmd/rootline/` compila sin errores

## Fuente de verdad

- `cmd/rootline/validate.go`
- `cmd/rootline/validate_test.go`
- `cmd/rootline/filter.go` (T001)
