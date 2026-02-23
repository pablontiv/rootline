---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Diagnóstico en doctor para aggregate+required

**Story**: [S001 Aggregate Field Exclusion](README.md)

[[blocks:T001-implicit-aggregate-exclusion]]

## Contexto

Las capas 1 y 2 implementan la exclusión automática y declarativa. Pero el comportamiento implícito (campos con aggregate se auto-excluyen de required en index files) puede confundir a usuarios que no lo conocen. `rootline doctor` ya ejecuta 6 checks de salud sobre `.stem` files. Se necesita un check 7 que detecte cuando un campo es `required: true` Y tiene expresión `aggregate:` — indicando que el required se relaja automáticamente en index files.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: DoctorCheck
    metodos:
      - nombre: check 7 logic
        input: "parsedStems map, merged StemFile"
        output: "[]DoctorCheck"
dependencias_externas: []
tests:
  - .stem con campo required + aggregate → warning emitido
  - .stem con campo required sin aggregate → no warning
  - .stem con campo aggregate sin required → no warning
  - Output incluye nombre del campo y sugerencia
```

## Alcance

**In**:
1. Nuevo check "Aggregated Required Fields" en `cmd/rootline/doctor.go` (~línea 237, después de check 6)
2. Para cada `.stem` parseado, iterar Schema fields
3. Si `field.Required == true` y el campo name existe como key en `Aggregate` map
4. Emitir `DoctorCheck{Status: "warn", Name: "aggregated-required", Message: "..."}` con mensaje explicativo
5. Tests

**Out**: Cambios al aggregate engine, auto-fix de frontmatter, cambios a validate

## Estado inicial esperado

- T001 completado (comportamiento implícito funciona)
- `cmd/rootline/doctor.go` tiene 6 checks implementados con patrón claro
- `DoctorCheck` struct con Name, Status, Message, Path

## Criterios de Aceptacion

- `go test ./cmd/rootline/ -run TestDoctor -race` pasa (o test e2e equivalente)
- `rootline doctor` sobre directorio con .stem que tiene required+aggregate → output incluye warning "aggregated-required"
- `rootline doctor` sobre directorio con .stem sin conflicto → no warning
- Warning message incluye nombre del campo y sugiere quitar `required: true` o usar `excludes`
- `golangci-lint run ./cmd/rootline/` sin errores

## Fuente de verdad

- `cmd/rootline/doctor.go` — checks existentes (líneas 92-236), patrón DoctorCheck
- `internal/rules/rules.go` — StemFile struct (Schema + Aggregate maps)
