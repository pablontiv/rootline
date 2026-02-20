---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Insertar derivacion en pipeline de query, tree, y stats

**Story**: [S002 Derivation Pipeline](README.md)

## Contexto

Con DeriveRecord funcional (T001), falta conectarlo al pipeline de los comandos CLI. Los comandos query, tree, y stats deben ejecutar derivacion despues de validacion y antes de presentar resultados. Los campos derivados deben ser accesibles via --field y en output JSON.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: deriveRecords (helper)
    metodos:
      - nombre: deriveAll
        input: "records []*extract.Record, effective *rules.StemFile"
        output: "error"
dependencias_externas: []
tests:
  - query con .stem derive incluye campos derivados en output
  - tree con .stem derive incluye campos derivados
  - --field derived.slug extrae campo derivado
  - comando sin .stem derive funciona sin cambios (backward-compatible)
```

## Dependencias

- T001 (DeriveRecord)
- Comandos query, tree, stats existentes

## Alcance

**In**:
1. Helper `deriveAll(records, effective)` que itera records y llama DeriveRecord
2. Insertar en query.go despues de index.Scan y antes de query.Execute
3. Insertar en tree.go despues de scan
4. Insertar en stats.go despues de scan
5. Record.Derived aparece en JSON output
6. --field soporta `derived.fieldname` path
7. Backward-compatible: sin derive en .stem, comportamiento identico

**Out**: Derivation en validate, derivation en describe, derived field indexing

## Estado inicial esperado

- T001 completado (DeriveRecord funcional)
- Comandos query, tree, stats funcionales

## Criterios de Aceptacion

- `rootline query --where 'tipo eq X' -o json` con .stem derive incluye "derived" en output
- `rootline query --field derived.slug` extrae campo derivado
- `rootline query` sin .stem derive funciona identico a antes
- `rootline tree` y `rootline stats` incluyen campos derivados cuando aplica
- `go test ./cmd/rootline/ -race` pasa
- `go build ./cmd/rootline/` compila sin errores

## Fuente de verdad

- `cmd/rootline/query.go` — query pipeline
- `cmd/rootline/tree.go` — tree pipeline
- `cmd/rootline/stats.go` — stats pipeline
- `internal/derive/` — DeriveRecord
