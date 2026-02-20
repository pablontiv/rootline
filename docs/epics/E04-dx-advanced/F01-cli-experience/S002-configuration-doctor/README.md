# S002: Configuration Doctor

**Feature**: [F01 CLI Experience](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Rootline diagnostica automaticamente problemas de configuracion en archivos .stem con checks de salud y reportes claros

## Antes / Despues

**Antes**: Si un .stem tiene errores sutiles (YAML invalido, scope que no matchea ningun archivo, hijo que redefine campo con tipo incompatible al padre), el usuario descubre los problemas indirectamente a traves de errores de validacion confusos o resultados inesperados.

**Despues**: `rootline doctor` ejecuta 6 checks diagnosticos sobre todos los .stem files del proyecto y reporta con iconos claros (check/cross/warning) que esta bien, que esta mal, y que es sospechoso. El usuario puede correr `rootline doctor` antes de `rootline validate` para asegurarse de que su configuracion es correcta.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline doctor` reporta .stem files con YAML invalido
- [ ] `rootline doctor` detecta .stem files cuyo scope no matchea ningun archivo en su directorio
- [ ] `rootline doctor` warn cuando un hijo redefine un campo ya definido por el padre
- [ ] Output muestra resumen con conteo de checks passed/failed/warnings

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-doctor-command.md) | Comando doctor con 6 checks diagnosticos |

## Fuente de verdad

- `internal/rules/` — ParseStemFile, WalkUp, MergeStemFiles
- `internal/index/` — Scan, MatchesScope
