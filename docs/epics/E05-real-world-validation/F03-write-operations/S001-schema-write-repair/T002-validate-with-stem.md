---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T002: Validar con .stem generado incluyendo --strict

**Story**: [S001 Schema Write & Repair](README.md)

## Contexto

Con el `.stem` generado en T001, ahora validate tiene reglas contra las cuales validar. Deberia detectar errores reales: READMEs sin frontmatter que no cumplen el schema, campos faltantes, etc.

## Dependencias

- T001: generate-stem-file (necesita .stem generado)

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
comando: validate
flags: --all, --strict
target: mismo subdirectorio de T001 (con .stem)
```

## Alcance

**In**:
1. `rootline validate --all` en el subdirectorio con .stem
2. `rootline validate --all --strict` en el mismo directorio
3. Verificar que reporta errores reales (READMEs sin frontmatter)
4. Verificar JSON output con severity

**Out**: Modificacion de archivos

## Criterios de Aceptacion

- Reporta errores reales (archivos sin frontmatter que violan el schema)
- JSON output con `"version": 1` y campo severity
- `--strict` reporta mas errores que sin --strict
- Errores son coherentes con la realidad del directorio
- Sin panics ni stack traces
