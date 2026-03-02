---
estado: Completed
tipo: software-module
ejecutable_en: rootline
---
# T003: Update `rootline init` Flat-Mode to Generate V2

**Story**: [S001 V1 Deprecation Engine & Migration](README.md)
**Contribuye a**: Init: `rootline init --dry-run` genera `version: 2` (no `version: 1`)
**Preserva**: INV1 (tests existentes pasan), INV2 (pipeline verde)

## Contexto

`cmd/rootline/init.go` line 271 en `generateStemYAML()` emite `version: 1` para schemas planos (no jerárquicos). La versión jerárquica (`generateHierarchicalRootYAML`) ya emite `version: 2`. Hay que alinear ambos.

## Alcance

**In scope**:
- Cambiar `generateStemYAML()` de `"version: 1\n..."` a `"version: 2\n..."`
- Actualizar tests en `cmd/rootline/init_test.go` que aserten `version: 1` output

**Out of scope**: Cambiar la lógica de generación de schemas

## Especificacion Tecnica

```yaml
archivo: cmd/rootline/init.go
linea: 271
cambio: |
  // antes
  b.WriteString("version: 1\nscope:\n  match: \"*.md\"\nschema:\n")
  // después
  b.WriteString("version: 2\nscope:\n  match: \"*.md\"\nschema:\n")
```

## Criterios de Aceptación

- [ ] `go run ./cmd/rootline/ init --dry-run /tmp/test-dir/` genera `version: 2` (con archivos de prueba)
- [ ] `go test ./cmd/rootline/ -run TestInit -v` pasa (con expectativas actualizadas)
- [ ] `go test ./... -race` pasa verde
