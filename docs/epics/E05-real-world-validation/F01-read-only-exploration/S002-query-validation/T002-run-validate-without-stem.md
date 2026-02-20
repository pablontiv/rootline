---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T002: Ejecutar validate sin .stem files

**Story**: [S002 Query & Validation](README.md)

## Contexto

El comando `validate` verifica que los archivos cumplen con las reglas definidas en `.stem` files. Sin `.stem` files, no deberia haber reglas que validar, por lo tanto 0 errores. Este es un edge case importante: validate debe comportarse gracefully cuando no hay schema definido.

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
comando: validate
target: /opt/homeserver/automation/docs/epics/
condicion: sin .stem files
```

## Alcance

**In**:
1. `rootline validate --all /opt/homeserver/automation/docs/epics/ --output json`
2. Verificar que retorna 0 errores
3. Verificar exit code 0

**Out**: Generacion de .stem, modificacion de archivos

## Criterios de Aceptacion

- Sin `.stem` files → 0 errores reportados
- Exit code 0
- JSON output con `"version": 1`
- Sin panics ni stack traces
