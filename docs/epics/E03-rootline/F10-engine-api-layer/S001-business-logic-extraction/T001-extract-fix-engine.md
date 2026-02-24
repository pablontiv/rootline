---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Extract fix engine to internal/fix

**Story**: [S001 Business Logic Extraction](README.md)

## Contexto

`cmd/rootline/fix.go` (731 LOC) contiene ~365 líneas de lógica de negocio que no es accesible fuera del CLI. Las funciones `applyProposals`, `applyFixes`, `rewriteFrontmatter`, `closestMatch`, y `levenshtein` son lógica pura sin dependencia de cobra ni I/O de consola. Esta lógica debe vivir en `internal/fix/` para que el futuro MCP server pueda reutilizarla.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/fix
interfaces:
  - nombre: ApplyProposals
    metodos:
      - nombre: ApplyProposals
        input: "report *proposal.Report, root string, records []*extract.Record"
        output: "error"
  - nombre: ApplyFixes
    metodos:
      - nombre: ApplyFixes
        input: "record *extract.Record, effective *rules.StemFile, errs []rules.ValidationError"
        output: "added []string, corrected []string"
  - nombre: RewriteFrontmatter
    metodos:
      - nombre: RewriteFrontmatter
        input: "original string, fm map[string]any"
        output: "string"
dependencias_externas:
  - gopkg.in/yaml.v3
tests:
  - ApplyFixes corrige campo required faltante con default value
  - ApplyFixes corrige enum inválido al valor más cercano (levenshtein)
  - ApplyProposals aplica extend_enum seguido de correct_value
  - RewriteFrontmatter preserva body después de reescribir frontmatter
  - RewriteFrontmatter maneja archivos sin frontmatter existente
  - closestMatch retorna mejor match por distancia de edición
```

## Dependencias

- `internal/rules` — tipos StemFile, ValidationError
- `internal/extract` — tipo Record
- `internal/proposal` — tipos Report, Proposal

## Alcance

**In**:
1. Crear paquete `internal/fix/` con archivo `fix.go`
2. Mover funciones: `applyProposals`, `applyExtendEnum`, `addEnumValueToNode`, `applyMigrateValue`, `applyCorrectValue`, `applySetField`, `applyCorrectLink`, `rewriteRecordFile`, `applyFixes`, `closestMatch`, `levenshtein`, `rewriteFrontmatter`, `writeFrontmatterFields`
3. Exportar funciones públicas: `ApplyProposals`, `ApplyFixes`, `RewriteFrontmatter`, `ClosestMatch`
4. Mantener helpers internos como unexported: `applyExtendEnum`, `addEnumValueToNode`, etc.
5. Actualizar `cmd/rootline/fix.go` para delegar a `internal/fix`
6. Crear `internal/fix/fix_test.go` con tests unitarios
7. Verificar que tests existentes en `cmd/rootline/fix_test.go` siguen pasando

**Out**: No refactorizar la lógica interna de las funciones. No cambiar signatures de funciones helper privadas. No tocar `proposalsToFixResults` ni `renderProposalTable` (son CLI puro).

## Estado inicial esperado

- `cmd/rootline/fix.go` contiene todas las funciones listadas
- `internal/fix/` no existe
- `cmd/rootline/fix_test.go` y `cmd/rootline/fix_apply_test.go` pasan

## Criterios de Aceptacion

- `go test ./internal/fix/ -race` pasa con >= 80% coverage del nuevo paquete
- `go test ./cmd/rootline/ -run TestFix -race` pasa (tests existentes no rotos)
- `go test ./... -race` pasa sin errores
- `go vet ./...` limpio
- `cmd/rootline/fix.go` no contiene las funciones `levenshtein`, `rewriteFrontmatter`, `applyProposals`
- `internal/fix/fix.go` exporta `ApplyProposals`, `ApplyFixes`, `RewriteFrontmatter`

## Fuente de verdad

- `cmd/rootline/fix.go` — funciones a extraer (líneas 280-731)
- `cmd/rootline/fix_test.go` — tests existentes
- `cmd/rootline/fix_apply_test.go` — tests de apply
