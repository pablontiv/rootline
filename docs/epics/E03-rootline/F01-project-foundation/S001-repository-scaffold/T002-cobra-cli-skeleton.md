---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Crear cobra root command con subcommand stubs

**Story**: [S001 Repository Scaffold](README.md)

## Contexto

Rootline expone 7 comandos CLI (query, validate, describe, tree, stats, serve, explain). Este Task crea el wiring de cobra con stubs vacios para cada subcomando. La implementacion real de cada comando ocurre en Tasks posteriores (F03, F04, F05).

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: Root command
    metodos:
      - nombre: Execute
        input: (none)
        output: error
dependencias_externas:
  - github.com/spf13/cobra
  - github.com/spf13/viper
tests:
  - rootline --help muestra todos los subcomandos
  - rootline --version muestra version
  - cada subcomando existe y muestra su propio --help
```

## Dependencias

- T001 completado (Go module inicializado)

## Alcance

**In**:
1. Root command (`rootline`) con description, version flag
2. 7 subcommand stubs: `query`, `validate`, `describe`, `tree`, `stats`, `serve`, `explain`
3. Cada stub imprime "not implemented yet" y retorna exit 0
4. Global flags: `--output json|table` (default json), `--field` (dot-path extraction)
5. `cmd/rootline/main.go` wires root command

**Out**: Implementacion de logica de comandos, tests de integracion

## Estado inicial esperado

- T001 completado: Go module existe, `cmd/rootline/main.go` existe
- `go build ./cmd/rootline/` exitoso

## Criterios de Aceptacion

- `rootline --help` lista 7 subcomandos
- `rootline --version` muestra version string
- `rootline query --help` muestra flags del query command
- `rootline validate` imprime "not implemented yet" y retorna exit 0
- Cada subcomando acepta `--help` sin error

## Fuente de verdad

- `src/rootline/docs/intent/v0-rootline.md` seccion 3 (Commands table)
- `src/rootline/docs/research/I1-query-operators.md` seccion 5 (CLI Flag Mapping)
