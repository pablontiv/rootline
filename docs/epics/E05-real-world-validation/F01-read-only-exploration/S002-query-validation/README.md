# S002: Query & Validation

**Feature**: [F01 Read-Only Exploration](../README.md)
**Estado**: Completado
**Cliente**: Platform Owner
**Capacidad**: Rootline consulta y valida datos externos coherentemente sin .stem files

## Antes / Despues

**Antes**: Los operadores de query solo se probaron en unit tests con datos sinteticos. validate y doctor nunca se ejecutaron contra un directorio sin .stem.

**Despues**: query filtra correctamente por estado/tipo, validate y doctor se comportan coherentemente sin .stem files.

## Criterios de Aceptacion (semanticos)

- [x] `query --where "estado eq Completado"` retorna resultados no vacios
- [x] `validate --all` sin .stem retorna 0 errores, exit code 0
- [x] `doctor` reporta estado coherente sin panic

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-run-query-operators.md) | Ejecutar query con multiples operadores y combinaciones |
| [T002](T002-run-validate-without-stem.md) | Ejecutar validate sin .stem files |
| [T003](T003-run-doctor-without-stem.md) | Ejecutar doctor sin .stem files |
