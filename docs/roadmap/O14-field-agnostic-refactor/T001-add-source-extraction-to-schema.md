---
estado: Completed
tipo: task
---
# T001: Add `source:` extraction to SchemaField

**Outcome**: [O14 Field-agnostic refactor](README.md)
**Contribuye a**: rootline puede derivar campos desde el body del documento, no solo desde frontmatter

## Preserva

- INV1: Todos los campos sin `source:` siguen leyendo desde frontmatter (comportamiento actual)
  - Verificar: `go test ./internal/rules/... ./internal/derive/...`
- INV2: Los tests existentes de EnrichBuiltins no cambian de resultado
  - Verificar: `go test ./internal/derive/... -run TestEnrich`

## Contexto

Actualmente SchemaField solo lee campos desde frontmatter YAML. Para eliminar el hardcoding de "titulo" en `query.go` y `tree.go`, necesitamos un mecanismo genérico que permita derivar el valor de un campo desde el body del documento.

El plan agrega `Extract string yaml:"source"` a `SchemaField` y `schemaFieldRaw`. Notar que ya existe un campo `Source string yaml:"-"` interno — usar nombre diferente: `Extract string yaml:"source"` en el struct raw, mapeado al campo final del SchemaField.

`EnrichBuiltins` en `internal/derive/enrich.go` itera el schema efectivo después de poblar `rec.Derived["isIndex"]`. Para cada campo con `Extract == "body.h1"`, llama `extractBodyH1(rec.Body)` y popula `rec.Derived[name]`. Para `body.section["## Heading"]`, busca en `rec.Sections` (o parsea el body) y popula `rec.Derived[name]`.

## Alcance

**In**:
1. Agregar `Extract string yaml:"source"` a `SchemaField` y `schemaFieldRaw` en `internal/rules/rules.go`
2. Implementar `extractBodyH1(body string) string` como helper package-private en `internal/derive/enrich.go`
3. Agregar soporte `body.section["## Heading"]` en `EnrichBuiltins`
4. Agregar unit tests para `extractBodyH1` y para derivación via `source: body.h1`

**Out**:
- No cambiar el comportamiento de campos sin `source:`
- No integrar con tree.go ni query.go todavía (eso es T004/T005)

## Estado inicial esperado

- `go test ./...` pasa en `/home/shared/rootline`
- No existe campo `Extract`/`source` en SchemaField

## Criterios de Aceptación

- `SchemaField` tiene campo `Extract string yaml:"source"` (o equivalente con tag yaml:"source")
- `EnrichBuiltins` popula `rec.Derived["titulo"]` cuando el schema tiene `titulo: {source: body.h1}`
- `extractBodyH1` retorna el texto del primer `# H1` del body, o string vacío si no hay
- `go test ./internal/rules/... ./internal/derive/...` verde

## Fuente de verdad

- `internal/rules/rules.go` — SchemaField, schemaFieldRaw
- `internal/derive/enrich.go` — EnrichBuiltins, nueva helper extractBodyH1
