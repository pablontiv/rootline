---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Tipificar links en .stem con allowed types y target patterns

**Story**: [S002 Link Schema and Validation](README.md)

## Contexto

StemFile.Links es actualmente `map[string]any`. Necesita tipificarse para soportar: lista de tipos permitidos (allowed), y reglas por tipo con target pattern (glob) y campo de referencia. Esto sigue el modelo de schema: definir restricciones en .stem que la validacion aplica.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: LinkSchema
    metodos: []
  - nombre: LinkRule
    metodos: []
dependencias_externas: []
tests:
  - ParseStemFile con links.allowed parsea lista de tipos
  - ParseStemFile con links.blocks.target parsea glob pattern
  - Merge de links sigue reglas type-driven (maps merge)
  - .stem sin links section retorna LinkSchema vacio
```

## Dependencias

- Ninguna (modifica struct existente)

## Alcance

**In**:
1. Struct `LinkSchema` con: Allowed []string, Rules map[string]LinkRule
2. Struct `LinkRule` con: Target string (glob), Field string
3. Reemplazar `Links map[string]any` con `Links LinkSchema` en StemFile
4. Parse .stem YAML a LinkSchema
5. Merge de LinkSchema sigue reglas existentes (Allowed: array replace, Rules: map merge)
6. Ejemplo .stem:
   ```yaml
   links:
     allowed: [blocks, parent, reference]
     blocks:
       target: "../tasks/*.md"
       field: blocked_by
   ```

**Out**: Link validation (T002), link resolution, bidirectional linking

## Estado inicial esperado

- StemFile.Links es map[string]any
- MergeStemFiles funcional

## Criterios de Aceptacion

- ParseStemFile con links.allowed: [blocks, parent] retorna LinkSchema.Allowed == ["blocks", "parent"]
- ParseStemFile con links.blocks.target retorna LinkSchema.Rules["blocks"].Target
- ParseStemFile sin links section retorna LinkSchema vacio (Allowed nil, Rules nil)
- MergeStemFiles con padre y hijo links sections mergea correctamente
- Tests existentes siguen pasando (backward-compatible con map[string]any → LinkSchema)
- `go test ./internal/rules/ -race` pasa

## Fuente de verdad

- `internal/rules/rules.go` — StemFile struct, Links field
- `internal/rules/merge.go` — mergeAnyMap (adaptar para LinkSchema)
