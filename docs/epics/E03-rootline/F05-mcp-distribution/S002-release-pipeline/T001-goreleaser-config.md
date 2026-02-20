---
estado: Completado
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T001: Configurar goreleaser y GitHub Actions release workflow

**Story**: [S002 Release Pipeline](README.md)

## Contexto

goreleaser es el estandar para releases de CLIs Go. Produce binarios estaticos multi-plataforma, checksums, changelogs, y GitHub releases automaticamente. El workflow de GitHub Actions se triggerea en tag push.

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: github-actions
triggers:
  - tag
jobs:
  - nombre: release
    pasos:
      - Checkout con fetch-depth 0
      - Setup Go
      - Run goreleaser release
artefactos:
  - rootline_linux_amd64
  - rootline_linux_arm64
  - rootline_darwin_amd64
  - rootline_darwin_arm64
  - rootline_windows_amd64.exe
  - checksums.txt
```

## Dependencias

- F01/S002 completado (CI pipeline base)
- Proyecto compilable y testeable

## Alcance

**In**:
1. `.goreleaser.yaml` con builds para linux/darwin/windows x amd64/arm64
2. CGO_ENABLED=0 para binarios estaticos
3. Checksums y changelog automatico
4. GitHub Actions workflow `.github/workflows/release.yml` triggered on tag
5. `goreleaser --snapshot --clean` para verificacion local

**Out**: Docker image, Snap package, Scoop manifest

## Estado inicial esperado

- Go project compilable
- GitHub Actions CI funcional (F01/S002)

## Criterios de Aceptacion

- `.goreleaser.yaml` existe y es valido (`goreleaser check`)
- `goreleaser --snapshot --clean` produce binarios en dist/
- Binarios para 5 targets (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
- GitHub Actions release workflow existe y se triggerea en tag push
- Checksums.txt incluido en release

## Fuente de verdad

