---
estado: Pending
tipo: feature
---
# F01: Supply Chain Security

**Epic**: [E05](../README.md)
**Objetivo**: Dependencias y GitHub Actions tienen escaneo automático de vulnerabilidades y están pinneadas para prevenir supply chain attacks
**Beneficio**: Elimina el riesgo de dependencias vulnerables no detectadas y de tag-hijacking en GitHub Actions
**Milestone**: Dependabot abre PRs automáticos, `govulncheck` y `gosec` corren en CI, todas las Actions están SHA-pinneadas

## Scope

**In**: Configurar Dependabot, agregar govulncheck y gosec al pipeline CI, SHA-pin todas las GitHub Actions en ci.yml y release.yml
**Out**: Auditoría manual de dependencias, políticas de branch protection, signing de commits

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-automated-vulnerability-scanning/) | Automated Vulnerability Scanning | El repositorio detecta y notifica vulnerabilidades en dependencias y código automáticamente |

## Dependencias

- Ninguna

## Fuente de verdad

- `.github/workflows/ci.yml` — pipeline CI actual
- `.github/workflows/release.yml` — pipeline de release
- `.golangci.yml` — configuración de linters
- `go.mod` — dependencias del proyecto
