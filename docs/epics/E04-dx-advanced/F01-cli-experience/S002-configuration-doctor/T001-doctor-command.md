---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar comando doctor con 6 checks diagnosticos

**Story**: [S002 Configuration Doctor](README.md)

## Contexto

El comando `rootline doctor` inspecciona todos los archivos .stem del proyecto y ejecuta checks de salud para detectar problemas de configuracion antes de que causen errores de validacion confusos. Es un complemento de `rootline init` (F02) — init crea .stem, doctor verifica que estan bien configurados.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: doctorCmd
    metodos:
      - nombre: RunE
        input: "cmd *cobra.Command, args []string"
        output: "error"
dependencias_externas: []
tests:
  - doctor con .stem YAML invalido reporta error
  - doctor con scope que no matchea archivos reporta warning
  - doctor con herencia consistente reporta check
  - doctor con campo redefinido por hijo reporta warning
  - doctor sin problemas reporta todo OK
```

## Dependencias

- internal/rules (ParseStemFile, WalkUp)
- internal/index (Scan, MatchesScope)

## Alcance

**In**:
1. Comando `rootline doctor [path]` (default: ".")
2. Check 1: .stem files son YAML valido (parse sin error)
3. Check 2: No hay .stem huerfanos (scope.match no matchea ningun archivo en el directorio)
4. Check 3: Schema inheritance es consistente (hijo no cambia type de un campo heredado)
5. Check 4: Campos enum tienen al menos 2 valores
6. Check 5: No hay reglas validate referenciando campos que no existen en schema
7. Check 6: Warning si hijo redefine campo que padre ya define (informativo, no error)
8. Output: lista de checks con icono (check/cross/warning), resumen final
9. JSON output con `--output json`

**Out**: Auto-fix de problemas detectados (eso es `rootline fix`), checks de contenido de documentos

## Estado inicial esperado

- Archivos .stem existentes en el proyecto
- internal/rules y internal/index funcionales

## Criterios de Aceptacion

- `rootline doctor` contra directorio con .stem validos muestra todos checks passed
- `rootline doctor` contra directorio con .stem YAML invalido reporta error con path del archivo
- `rootline doctor -o json` produce JSON con version:1 y kind:"rootline/doctor"
- `go build ./cmd/rootline/` compila sin errores
- `go test ./cmd/rootline/ -race` pasa

## Fuente de verdad

- `internal/rules/rules.go` — StemFile, SchemaField structs
- `internal/rules/discovery.go` — WalkUp
- `internal/rules/merge.go` — MergeStemFiles
- `internal/index/index.go` — Scan
- `internal/index/scope.go` — MatchesScope
