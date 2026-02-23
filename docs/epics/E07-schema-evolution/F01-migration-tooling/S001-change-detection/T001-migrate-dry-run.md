---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar rootline migrate --dry-run

**Story**: [S001 Change Detection](README.md)

## Contexto

rootline necesita un comando `migrate` que compare el .stem actual contra una version anterior y reporte que cambio. La version anterior se obtiene de git (`git show HEAD:<path>`) o de un archivo especificado con `--from`. El modo `--dry-run` solo reporta, no modifica.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline + internal/migrate
interfaces:
  - nombre: SchemaDiff
    metodos:
      - nombre: Diff
        input: "before, after *rules.StemFile"
        output: "DiffResult"
dependencias_externas: []
tests:
  - .stem sin cambios → "no changes detected"
  - Campo agregado → non-breaking change
  - Campo removido → breaking change con conteo de archivos afectados
  - Enum value removido → breaking change
  - Required false→true → breaking change
```

## Dependencias

- internal/rules/ existente (StemFile parsing, LoadStem)
- git en PATH (para obtener version anterior)

## Alcance

**In**:
1. Nuevo archivo `cmd/rootline/migrate.go` con cobra command
2. Nuevo package `internal/migrate/` con SchemaDiff logic
3. Flag `--dry-run` (default true — migrate es dry-run by default)
4. Flag `--from <path>` para comparar contra archivo especifico
5. Default: comparar contra `git show HEAD:<stem-path>`
6. Output JSON con `"version": 1` (consistente con otros comandos)
7. Output table para humanos

**Out**: Apply mode (eso es S002), multi-level .stem diffs, interactive mode

## Estado inicial esperado

- internal/rules/ con LoadStem funcional
- git repositorio con historial de .stem

## Criterios de Aceptacion

- `rootline migrate --dry-run docs/epics/` parsea .stem actual y anterior
- Output lista cambios con clasificacion (breaking/non-breaking)
- Sin cambios → "no changes detected"
- `--from other.stem` compara contra archivo especificado
- `--output json` produce JSON con version: 1
- `go test ./internal/migrate/ -v` pasa

## Fuente de verdad

- `internal/rules/rules.go` (StemFile struct — before/after)
- `cmd/rootline/fix.go` (patron de dry-run CLI)
