# S001: CLI Polish

**Feature**: [F01 CLI Experience](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Rootline ofrece autocompletado nativo en bash/zsh/fish y output tabla formateado en todos los comandos principales

## Antes / Despues

**Antes**: No hay autocompletado — el usuario debe recordar comandos, flags y su sintaxis. Todos los comandos (excepto stats) solo producen JSON, que es dificil de leer visualmente en terminal para uso interactivo.

**Despues**: `rootline <tab>` autocompleta comandos y flags en bash, zsh y fish. `rootline validate --all -o table` muestra tabla alineada con archivos, status y errores. `rootline query -o table` muestra registros en columnas. `rootline describe -o table` muestra campos del schema con tipo y source.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline completion bash` genera script funcional que autocompleta subcomandos y flags
- [ ] `rootline validate --all -o table` produce tabla con columnas File, Status, Errors
- [ ] `rootline query -o table` produce tabla con columnas derivadas de los campos del resultado
- [ ] `rootline describe path -o table` muestra campos del schema efectivo en formato tabla

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-shell-completions.md) | Subcomando completion usando Cobra built-in |
| [T002](T002-table-output-formatter.md) | Helper compartido de tabla y aplicar a validate, query, describe |

## Fuente de verdad

- `cmd/rootline/root.go` — outputFormat flag
- `cmd/rootline/stats.go` — renderStatsTable() como referencia
- `cmd/rootline/validate.go`, `query.go`, `describe.go` — comandos a modificar
