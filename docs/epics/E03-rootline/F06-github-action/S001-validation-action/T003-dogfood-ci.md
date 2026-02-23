---
estado: Completed
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T003: Agregar validacion de docs a CI de rootline

**Story**: [S001 Validation Action](README.md)

## Contexto

rootline's own CI (.github/workflows/ci.yml) no valida sus propios docs/epics/. Esta task agrega un job `docs-validate` que construye rootline y ejecuta validate en cada push. Es la forma mas directa de dogfooding — no depende de la action publicada.

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: github-actions
triggers:
  - push
  - pr
jobs:
  - nombre: docs-validate
    pasos:
      - Checkout
      - Setup Go
      - go build -o rootline ./cmd/rootline/
      - ./rootline validate --all docs/epics/
artefactos:
  - Job adicional en ci.yml
```

## Dependencias

- Ninguna (usa binary compilado localmente, no la action)

## Alcance

**In**:
1. Nuevo job `docs-validate` en `.github/workflows/ci.yml`
2. Build rootline binary desde source
3. Ejecutar `rootline validate --all docs/epics/`
4. Fail si hay errores de validacion

**Out**: Usar la GitHub Action (eso viene despues), PR annotations (esta task solo valida)

## Estado inicial esperado

- CI existente con jobs: build, test, lint
- docs/epics/ con .stem files y documentos

## Criterios de Aceptacion

- Job `docs-validate` existe en ci.yml
- Job compila rootline y ejecuta validate --all
- CI pasa en green en master (docs actuales son validos)
- PR que introduce documento invalido falla en CI

## Fuente de verdad

- `.github/workflows/ci.yml` (CI existente)
- `docs/epics/` (documentos a validar)
