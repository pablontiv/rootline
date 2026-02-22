---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Graph command carga .stem y pre-filtra links antes de Build()

**Story**: [S004 Graph Schema-Aware Link Filtering](README.md)

## Contexto

`rootline graph --check` reporta broken links falsos porque procesa todos los `[[...]]` parseados del body sin consultar el schema de links del .stem. El .stem ya define `links.allowed` y `links.rules` con target patterns, y `validateLinks()` ya existe en el pipeline de validacion. Pero el graph command no carga .stem en absoluto — solo hace `index.Scan()` + `graph.Build(records)`.

El principio: el .stem es la fuente de verdad. Solo links cuyo tipo tiene una regla `target` en `schema.Rules` son referencias estructurales que deben generar edges en el grafo. Links sin regla (como `reference` sin target) son datos, no FK constraints.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline + internal/graph
interfaces:
  - nombre: filterLinksBySchema (function, local a graph.go)
    metodos:
      - nombre: filterLinksBySchema
        input: "records []*extract.Record, schema rules.LinkSchema"
        output: "void (modifica records in-place)"
dependencias_externas: []
tests:
  - Records con links reference + blocks, filtrados a solo blocks → 0 broken links para reference
  - Records sin filtrar (schema vacio) → comportamiento actual preservado
  - graph --check docs/epics/ → 0 broken links, 0 cycles
```

## Dependencias

- `internal/rules` — WalkUp(), MergeStemFiles(), LinkSchema (ya implementados)
- `internal/graph` — Build(), BrokenLinks() (ya implementados)

## Alcance

**In**:
1. En `cmd/rootline/graph.go` `runGraph()`: despues de `index.Scan()`, resolver .stem efectivo para el scan root usando `rules.WalkUp(absRoot)` + `rules.MergeStemFiles(entries)` (mismo patron que `validate.go:89-93,122-128`)
2. Funcion local `filterLinksBySchema(records, schema)`: para cada record, filtrar `record.Links` manteniendo solo links cuyo `link.Type` tiene entrada en `schema.Rules`. Si schema esta vacio (IsEmpty), no filtrar (backward compatible)
3. Pasar records filtrados a `graph.Build(records)` — signature de Build() NO cambia
4. Tests unitarios para el filtrado
5. Test de integracion: `graph --check docs/epics/` con binario recompilado

**Out**: Cambios a graph.Build() signature, per-directory schema resolution (usar schema del scan root para todos), cambios a validateLinks()

## Estado inicial esperado

- `cmd/rootline/graph.go` existe con runGraph() funcional
- `cmd/rootline/validate.go` tiene el patron WalkUp+MergeStemFiles para referencia
- `internal/rules/rules.go` tiene LinkSchema con IsEmpty(), Rules map
- `docs/epics/.stem` tiene links section con `allowed: [blocks, reference]` y `blocks.target: "T*"`
- 3 broken links en `rootline graph --check docs/epics/` (target, target, A,B,C,A)

## Criterios de Aceptacion

- `filterLinksBySchema` con schema que tiene rules solo para `blocks` descarta links tipo `reference` de los records
- `filterLinksBySchema` con schema vacio (IsEmpty) no modifica ningun link
- `go test ./internal/graph/ -v` pasa (tests existentes + nuevos)
- `go build ./cmd/rootline/ && ./rootline graph --check docs/epics/` retorna exit 0 con 0 broken links
- `go vet ./...` limpio

## Fuente de verdad

- `cmd/rootline/graph.go` — runGraph (agregar carga .stem y filtrado)
- `cmd/rootline/validate.go:89-93,122-128` — patron de referencia para WalkUp+MergeStemFiles
- `internal/rules/rules.go:34-43,78-81` — LinkSchema, IsEmpty()
- `internal/graph/graph.go:39-65` — Build() (no modificar)
