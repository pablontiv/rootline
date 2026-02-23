---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Apply Proposals with Stem Rewrite

**Story**: [S003 Fix Command Integration](README.md)

[[blocks:T001-wire-proposals-into-fix-dry-run]]

## Contexto

Con dry-run mostrando propuestas (T001), `rootline fix` sin `--dry-run` necesita aplicar las propuestas, incluyendo la capacidad nueva de modificar el archivo `.stem` (para extend_enum). El orden de aplicacion importa: extend_enum primero (modifica .stem, lo que hace validos los valores antes invalidos), luego data fixes.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: applyProposals
    metodos:
      - nombre: applyProposals
        input: "report *proposal.Report, root string"
        output: "error"
dependencias_externas:
  - gopkg.in/yaml.v3
tests:
  - extend_enum → .stem reescrito con nuevo valor en enum
  - migrate_value → frontmatter con Bloqueada + [[blocks:T001]] en body
  - extract_body → frontmatter con valor extraido del body
  - infer_from_children → frontmatter con valor inferido
  - add_field + correct_value → comportamiento existente
  - validate despues de fix → 0 errores
```

## Alcance

**In**:
1. Implementar `applyProposals()` en `cmd/rootline/fix.go`
2. Para `extend_enum`: leer .stem YAML, agregar valor al array `values` del campo, reescribir .stem
3. Para `migrate_value`: cambiar frontmatter field + insertar `[[blocks:TARGET]]` antes del primer `## ` heading en body
4. Para `extract_body`: agregar valor (con mapping aplicado) a frontmatter
5. Para `infer_from_children`: agregar valor inferido a frontmatter
6. Para `add_field` / `correct_value`: reusar logica existente de `rewriteFrontmatter()`
7. Aplicar en orden de prioridad (extend_enum primero, luego el resto)
8. Output: lista de cambios aplicados

**Out**: No eliminar `**Key**: Value` del body en extract_body (preservar informacion legacy). No implementar undo/rollback.

## Estado inicial esperado

- `cmd/rootline/fix.go` tiene `runFixAll()` con proposal integration de T001
- `rewriteFrontmatter()` ya existe y funciona
- `internal/proposal/` completo con todos los detectores
- `go build ./cmd/rootline/` compila

## Criterios de Aceptacion

- `go build ./cmd/rootline/` compila sin errores
- En test fixture: `rootline fix --all` seguido de `rootline validate --all` produce 0 errores
- .stem file contiene el nuevo valor de enum despues de extend_enum fix
- Archivo con "Pending (blocked by T001)" despues de fix tiene `estado: Bloqueada` y `[[blocks:T001]]` en body
- `go vet ./cmd/rootline/` sin errores
- `golangci-lint run ./...` limpio

## Fuente de verdad

- `cmd/rootline/fix.go` — archivo a modificar
- `internal/proposal/proposal.go` — Proposal type con campos para cada tipo
- `internal/rules/rules.go` — StemFile YAML structure para rewrite
