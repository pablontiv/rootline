---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar comando init que genera .stem desde analisis

**Story**: [S001 Schema Inference](README.md)

## Contexto

El comando `rootline init [path]` usa el field analyzer (T001) para escanear archivos existentes y generar un archivo .stem con el schema inferido. Es el punto de entrada de adopcion — un usuario con archivos Markdown existentes puede generar su primer .stem sin conocer la sintaxis.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: initCmd
    metodos:
      - nombre: RunE
        input: "cmd *cobra.Command, args []string"
        output: "error"
dependencias_externas: []
tests:
  - init en directorio con .md genera .stem YAML valido
  - init --dry-run imprime a stdout sin escribir archivo
  - init en directorio sin .md retorna error descriptivo
  - init en directorio con .stem existente advierte y no sobreescribe
```

## Dependencias

- T001 (field analyzer en internal/infer/)
- internal/index (Scan)
- internal/extract (Registry)

## Alcance

**In**:
1. Comando `rootline init [path]` (default: ".")
2. Escanea archivos .md en el path con index.Scan + extract.Registry
3. Pasa Records al analyzer (internal/infer)
4. Genera YAML .stem con version, scope, schema
5. Flag `--dry-run`: imprime a stdout sin escribir
6. Sin flag: escribe `.stem` en el directorio target
7. Si `.stem` ya existe: advertir y no sobreescribir (usar `--force` para override)

**Out**: Inferencia de validate rules, inferencia de derive expressions, interactive mode

## Estado inicial esperado

- internal/infer/ con Analyze() funcional (T001)
- index.Scan y extract.Registry funcionales

## Criterios de Aceptacion

- `rootline init docs/epics/ --dry-run` imprime YAML valido a stdout con campos inferidos
- `rootline init /tmp/test-dir/` crea archivo .stem en el directorio
- `rootline init /tmp/empty/` retorna error "no markdown files found"
- El .stem generado es parseable por `rootline describe`
- `go build ./cmd/rootline/` compila sin errores

## Fuente de verdad

- `internal/infer/` — Analyze (T001)
- `internal/index/index.go` — Scan
- `internal/extract/registry.go` — Registry
- `internal/rules/rules.go` — StemFile struct (output format reference)
