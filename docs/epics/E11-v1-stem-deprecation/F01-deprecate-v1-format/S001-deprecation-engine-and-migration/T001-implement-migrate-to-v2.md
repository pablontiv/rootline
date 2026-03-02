---
estado: Completed
tipo: software-module
ejecutable_en: rootline
---
# T001: Implement `rootline migrate --to-v2`

**Story**: [S001 V1 Deprecation Engine & Migration](README.md)
**Contribuye a**: Migración: `rootline migrate --to-v2 <path>` convierte stems v1/v0 a v2
**Preserva**: INV1 (tests existentes pasan), INV2 (pipeline verde)

## Contexto

Rootline tiene stems con `version: 1` o sin version field (defaults to 0). Se necesita un comando bulk que los migre a `version: 2` preservando comentarios y formato YAML.

Ya existe `FindStemFiles()` en `internal/migrate/source.go` para descubrir stems y `MarshalStemV2()` en `internal/migrate/levels_to_match.go` para serializar. La nueva operación es más simple: solo reescribir el campo version usando YAML AST.

## Alcance

**In scope**:
- Crear `internal/migrate/to_v2.go` con `UpgradeToV2(dir string, dryRun bool) (*ToV2Result, error)`
- Usar `FindStemFiles()` para discovery
- Parsear cada stem con `yaml.Node` para preservar formato
- Reescribir `version` field: si es 0 o 1, cambiar a 2; si ya es 2, skip
- Si no existe campo version, insertarlo como primer campo
- Respetar `--dry-run`
- Agregar flag `--to-v2` en `cmd/rootline/migrate.go`
- Tests unitarios en `internal/migrate/to_v2_test.go`

**Out of scope**: Migrar contenido de `levels:` (eso lo hace `--from-levels`)

## Especificacion Tecnica

```yaml
componente: internal/migrate/to_v2.go
funcion_principal: UpgradeToV2(dir string, dryRun bool) (*ToV2Result, error)
dependencias:
  - internal/migrate/source.go:FindStemFiles
  - gopkg.in/yaml.v3 (Node API para AST preservation)
resultado:
  type: ToV2Result
  fields:
    - Version: 1
    - Kind: "rootline/migrate-to-v2"
    - Updated: []string  # relative paths
    - Skipped: int
    - Total: int
cli_flag: --to-v2 (bool) in cmd/rootline/migrate.go
cli_routing: if migrateToV2 { return runMigrateToV2(cmd, args) }
```

## Criterios de Aceptación

- [ ] `go run ./cmd/rootline/ migrate --to-v2 --dry-run docs/` lista stems v1 que serían actualizados
- [ ] `go test ./internal/migrate/ -run TestUpgradeToV2 -v` pasa con casos: v1→v2, v0→v2, already-v2 skip, comment preservation
- [ ] `go test ./... -race` pasa verde
