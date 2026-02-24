---
estado: Specified
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T001: Extender workflow para descargar binarios de release

**Story**: [S003 Binary Bundling](README.md)

## Contexto

Los skills dependen de `rootline` en PATH. Para que el marketplace sea auto-suficiente, debe incluir binarios pre-compilados. goreleaser ya produce binarios para 6 plataformas en cada release. El workflow de sync debe descargarlos y colocarlos en el marketplace.

## Alcance

**In**:
1. Nuevo job step en publish-marketplace.yml
2. Usar `gh release download` para obtener binarios del último release
3. Extraer y colocar en `bin/{os}-{arch}/rootline`
4. Plataformas: linux-amd64, linux-arm64, darwin-amd64, darwin-arm64

**Out**: Windows binaries, install script (T002)

## Estado inicial esperado

- S002/T001 completado: workflow base funcional
- goreleaser releases existentes con binarios multi-plataforma

## Criterios de Aceptacion

- Workflow descarga binarios de último release exitosamente
- Binarios colocados en `bin/` con subdirectorio por plataforma
- Binarios son ejecutables (permisos correctos)
- Workflow no falla si no hay releases (primer run antes de release)

## Fuente de verdad

- `.goreleaser.yml` (nombres de archive: `rootline_{os}_{arch}.tar.gz`)
- GitHub Releases API (`gh release download`)
