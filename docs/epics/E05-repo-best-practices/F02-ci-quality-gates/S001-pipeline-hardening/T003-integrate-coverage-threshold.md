---
estado: Pending
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T003: Integrar reporte de coverage con threshold mínimo

**Story**: [S001 Pipeline Hardening](README.md)

## Contexto

El CI actual genera `coverage.out` y lo sube como artefacto, pero nadie lo lee. No hay threshold — la cobertura puede degradar silenciosamente con cada PR. Se necesita un step que parsee el coverage y falle si baja del mínimo. Go tiene `go tool cover -func` nativo que muestra porcentaje total. Se puede usar un threshold simple con script bash sin dependencias externas.

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: github-actions
triggers:
  - push
  - pr
jobs:
  - nombre: test (existente, modificar)
    pasos:
      - (existente) go test ./... -race -coverprofile=coverage.out
      - (nuevo) Parsear coverage total con go tool cover -func
      - (nuevo) Comparar contra threshold y fallar si es menor
artefactos:
  - .github/workflows/ci.yml (modificado)
```

## Dependencias

- Ninguna

## Alcance

**In**:
1. Agregar step después de `go test` en el job `test` de ci.yml
2. Usar `go tool cover -func coverage.out | grep total | awk '{print $3}'` para extraer porcentaje
3. Comparar contra threshold (60% como punto de partida conservador)
4. Fallar con exit 1 y mensaje si coverage es menor al threshold

**Out**: Integrar con Codecov/Coveralls, agregar badge de coverage al README, generar reporte HTML

## Estado inicial esperado

- Job `test` genera `coverage.out` con `-coverprofile`
- No hay verificación de threshold
- Coverage actual desconocida (verificar al implementar)

## Criterios de Aceptacion

- Job `test` en ci.yml tiene step que verifica coverage contra threshold
- El threshold está definido como variable clara (fácil de ajustar)
- `go test ./... -race -coverprofile=coverage.out && go tool cover -func coverage.out | grep total` muestra porcentaje localmente
- Workflow YAML es válido

## Fuente de verdad

- `.github/workflows/ci.yml` — a modificar
- `coverage.out` — generado por go test
