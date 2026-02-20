---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: init advierte sobre contenido mixto

**Story**: [S002 Guidance UX](README.md)

## Contexto

Cuando `rootline init` escanea un directorio con mezcla de archivos con y sin frontmatter (ej: READMEs de Features + Tasks con frontmatter), infiere schema sin advertir que el resultado puede ser suboptimo. Los archivos sin frontmatter diluyen los thresholds de inferencia. E05 T001 confirmo este comportamiento.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline + internal/infer
interfaces:
  - nombre: AnalyzeResult (nuevo o extender)
    metodos:
      - nombre: metadata
        input: "TotalFiles, FilesWithFrontmatter, FilesWithout int"
        output: "usado por init para decidir warning"
dependencias_externas: []
tests:
  - init en directorio con >20% archivos sin frontmatter emite warning a stderr
  - init en directorio donde todos tienen frontmatter no emite warning
  - warning sugiere usar subdirectorio mas acotado
  - warning no afecta el .stem generado (se genera igual)
```

## Dependencias

- init command existente (`cmd/rootline/init.go`)
- infer.Analyze (`internal/infer/infer.go`)

## Alcance

**In**:
1. Modificar `infer.Analyze()` para retornar metadata adicional (total files, files with FM, files without)
2. En `runInit()`: despues de Analyze, calcular ratio sin frontmatter
3. Si >20% de archivos no tienen frontmatter, emitir warning a stderr:
   `"Warning: X of Y files have no frontmatter. Consider running init on a more specific subdirectory."`
4. El warning NO afecta la generacion del .stem (es informativo)
5. Tests

**Out**: Cambiar threshold de required inference, filtrar archivos sin frontmatter, interactive mode

## Estado inicial esperado

- `cmd/rootline/init.go` con runInit funcional
- `internal/infer/infer.go` con Analyze() que retorna InferredSchema

## Criterios de Aceptacion

- `rootline init /tmp/mixed/ 2>&1` incluye "Warning" cuando >20% sin frontmatter
- `rootline init /tmp/clean/` no emite warning cuando todos tienen frontmatter
- El .stem generado es identico con y sin warning (warning no afecta output)
- `rootline init --dry-run /tmp/mixed/ 2>/dev/null` muestra schema normal (warning va a stderr)
- `go test ./... -race` pasa

## Fuente de verdad

- `cmd/rootline/init.go` — init command
- `internal/infer/infer.go` — Analyze function
