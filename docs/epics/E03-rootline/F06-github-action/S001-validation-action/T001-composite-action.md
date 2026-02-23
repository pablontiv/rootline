---
estado: Completed
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T001: Crear composite GitHub Action definition

**Story**: [S001 Validation Action](README.md)

## Contexto

rootline necesita una GitHub Action reutilizable que cualquier repositorio pueda usar para validar documentacion estructurada en CI. La action descarga el binary de rootline desde GitHub Releases y ejecuta `rootline validate` con parametros configurables.

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: github-actions
triggers:
  - pr (via workflow del usuario)
jobs:
  - nombre: validate
    pasos:
      - Detectar si rootline ya esta en PATH
      - Si no, descargar de GitHub Releases (version configurable)
      - Ejecutar rootline validate --all --output json en path configurado
      - Setear outputs (valid, error_count)
artefactos:
  - action.yml (composite action definition)
```

## Dependencias

- rootline binary publicado en GitHub Releases (goreleaser ya configurado)

## Alcance

**In**:
1. `action.yml` como composite action
2. Input parameters: `version` (default latest), `path` (default `.`), `fail-on` (error|warning)
3. Output parameters: `valid` (bool), `error_count` (number)
4. Descarga de binary desde GitHub Releases

**Out**: Docker action, marketplace publishing, caching del binary

## Estado inicial esperado

- goreleaser configurado y funcional
- rootline binarios disponibles en GitHub Releases

## Criterios de Aceptacion

- `action.yml` existe con inputs/outputs documentados
- Action descarga rootline binary correcto para la plataforma
- `rootline validate --all` ejecuta exitosamente en el path configurado
- Exit code refleja resultado de validacion (0=valid, 1=errors found)

## Fuente de verdad

- `.goreleaser.yml` (nombres de artefactos)
- GitHub Releases API (URL pattern para descarga)
