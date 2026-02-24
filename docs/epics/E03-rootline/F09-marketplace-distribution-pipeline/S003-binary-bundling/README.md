---
tipo: historia
cliente: Plugin Consumer
---
# S003: Binary Bundling

**Feature**: [F09 Agent Marketplace](../README.md)
**Capacidad**: Marketplace incluye binarios pre-compilados de rootline y un install script, eliminando la necesidad de instalación separada

## Antes / Despues

**Antes**: Instalar skills desde marketplace requiere instalar rootline por separado. Skills fallan silenciosamente si rootline no está en PATH.

**Despues**: Marketplace incluye binarios para linux/darwin (amd64/arm64). Install script detecta plataforma, extrae binario correcto, y lo coloca en PATH. Consumer obtiene setup funcional desde un solo `npx skills add` o `claude plugin add`.

## Criterios de Aceptacion (semanticos)

- [ ] Workflow descarga binarios de último release de rootline
- [ ] Binarios disponibles para linux y darwin (amd64/arm64)
- [ ] Install script detecta OS/arch y extrae binario correcto
- [ ] Solo re-descarga binarios cuando hay nuevo release

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-download-release-binaries.md) | Extender workflow para descargar binarios de release |
| [T002](T002-install-script.md) | Crear script de instalación de binarios |
| [T003](T003-binary-freshness-check.md) | Agregar check de frescura de binarios |

## Fuente de verdad

- `.goreleaser.yml` (plataformas y nombres de archivos)
- GitHub Releases de rootline (binarios fuente)
