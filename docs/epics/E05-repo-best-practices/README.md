---
tipo: feature
---
# E05: Repo Best Practices

**Estado**: Activa
**Metrica de exito**: CI pipeline bloquea vulnerabilidades, enforce coverage, y el repo cumple estándares de producción open-source
**Timeline**: 2026-Q1 — en curso

## Intencion

Consolidar prácticas de seguridad de supply chain, calidad de CI/CD, testing avanzado y limpieza de repositorio para alcanzar estándares de producción. El repositorio tiene una base sólida (conventional commits, golangci-lint, goreleaser), pero carece de: escaneo automático de vulnerabilidades, enforcement de cobertura, fuzz/benchmarks, y tiene artefactos inconsistentes (binario committeado, dependencia archivada en pre-commit).

## Features

| ID | Nombre | Estado | Descripcion |
|----|--------|--------|-------------|
| F01 | [Supply Chain Security](F01-supply-chain-security/) | 0% | Dependabot, govulncheck, gosec, SHA-pinning de Actions |
| F02 | [CI Quality Gates](F02-ci-quality-gates/) | 0% | Pinear lint version, go mod tidy check, coverage threshold, fuzz tests, benchmarks |
| F03 | [Repository Hygiene](F03-repository-hygiene/) | 0% | Eliminar binario, reemplazar pre-commit archivado, editorconfig, CODEOWNERS, CONTRIBUTING |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| F01 | — | Independiente, habilita gosec para F02 |
| F03 | — | Limpieza base, independiente |
| F02 | F01 parcial | gosec habilitado en F01 se usa en F02 pipeline |

## Decision Log

| Fecha | Decision | Razon |
|-------|----------|-------|

## Gaps Activos

- Dockerfile y SBOM en goreleaser diferidos (baja prioridad para un CLI tool)
- CHANGELOG.md estático diferido (goreleaser genera changelog en releases)
