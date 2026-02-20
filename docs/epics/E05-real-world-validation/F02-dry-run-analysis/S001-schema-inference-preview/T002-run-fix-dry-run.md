---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T002: Ejecutar fix --dry-run sin .stem files

**Story**: [S001 Schema Inference Preview](README.md)

## Contexto

El comando `fix --dry-run` muestra que reparaciones haria sin ejecutarlas. Sin `.stem` files, no deberia haber nada que reparar porque no hay reglas definidas.

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
comando: fix
flags: --dry-run
target: /opt/homeserver/automation/docs/epics/
condicion: sin .stem files
```

## Alcance

**In**:
1. `rootline fix --dry-run /opt/homeserver/automation/docs/epics/`
2. Verificar que reporta 0 reparaciones necesarias
3. Verificar exit code 0

**Out**: Modificacion de archivos

## Criterios de Aceptacion

- Sin .stem → nada que reparar
- Exit code 0
- Sin panics ni stack traces
- No modifica ningun archivo en disco
