---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar comando fix que repara errores de validacion automaticamente

**Story**: [S002 Scaffolding and Repair](README.md)

## Contexto

El comando `rootline fix` es el complemento de `rootline validate`. Mientras validate detecta errores, fix los corrige automaticamente. Para cada error de validacion, aplica un fix apropiado: campos required faltantes se agregan con defaults, valores enum invalidos se corrigen al match mas cercano (Levenshtein distance). Esta es la primera feature que modifica archivos fuente.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: fixCmd
    metodos:
      - nombre: RunE
        input: "cmd *cobra.Command, args []string"
        output: "error"
dependencias_externas: []
tests:
  - fix agrega campo required faltante con default
  - fix corrige enum invalido al valor mas cercano
  - fix --dry-run muestra cambios sin modificar
  - fix en archivo sin errores no modifica nada
  - fix preserva campos existentes y body
```

## Dependencias

- internal/rules (WalkUp, MergeStemFiles, Validate)
- internal/extract (MarkdownExtractor, Record)

## Alcance

**In**:
1. Comando `rootline fix <file> [files...]` o `rootline fix --all`
2. Pipeline: extract → validate → para cada error, aplicar fix
3. Fix para "required field missing": insertar campo con default value en frontmatter
4. Fix para "invalid enum value": Levenshtein distance al valor valido mas cercano
5. Flag `--dry-run`: mostrar cambios propuestos sin escribir
6. Output: resumen de fixes aplicados (N fields added, M values corrected)
7. Preservar body y campos existentes (solo modificar frontmatter)

**Out**: Fix para reglas custom, fix para scope mismatches, interactive confirmation

## Estado inicial esperado

- internal/rules con Validate retornando ValidationError con field, rule, message
- MarkdownExtractor funcional

## Criterios de Aceptacion

- `rootline fix /tmp/test/missing-field.md` agrega campo required faltante
- `rootline fix /tmp/test/bad-enum.md` corrige "Compltado" a "Completado"
- `rootline fix --dry-run /tmp/test/doc.md` muestra cambios sin modificar archivo
- `rootline validate /tmp/test/doc.md` despues de fix retorna valid=true
- `rootline fix /tmp/test/valid.md` no modifica archivo ya valido
- `go build ./cmd/rootline/` compila sin errores

## Fuente de verdad

- `internal/rules/validate.go` — Validate, ValidationError
- `internal/rules/rules.go` — SchemaField (default, values)
- `internal/extract/extract.go` — Record, MarkdownExtractor
- `cmd/rootline/validate.go` — referencia de pipeline validate
