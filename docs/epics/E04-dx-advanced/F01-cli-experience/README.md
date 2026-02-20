# F01: CLI Experience

**Epic**: [E04](../README.md)
**Objetivo**: Los comandos CLI existentes son comodos de usar diariamente con autocompletado, output legible, y diagnosticos de configuracion
**Beneficio**: Reduce friccion de uso — el usuario no necesita recordar comandos ni parsear JSON visualmente
**Milestone**: `rootline <tab>` autocompleta comandos/flags, `rootline validate --all -o table` muestra tabla formateada, `rootline doctor` reporta 6 checks de salud

## Scope

**In**: Shell completions (bash/zsh/fish), table output formatter para validate/query/describe, doctor command con checks diagnosticos
**Out**: Rich terminal colors/themes, interactive TUI, output format YAML/CSV

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [CLI Polish](S001-cli-polish/) | Tab-completion funcional y output tabla legible en todos los comandos |
| S002 | [Configuration Doctor](S002-configuration-doctor/) | Diagnostico automatizado de problemas en .stem files |

## Dependencias

- Ninguna (usa comandos existentes de E03)

## Fuente de verdad

- `cmd/rootline/root.go` — cobra setup, global flags
- `cmd/rootline/stats.go` — referencia de table output existente (renderStatsTable)
- Cobra completion docs: https://github.com/spf13/cobra/blob/main/completions.go
