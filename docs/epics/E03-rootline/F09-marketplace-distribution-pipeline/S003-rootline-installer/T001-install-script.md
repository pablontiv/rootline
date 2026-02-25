---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Crear install script con descarga desde GitHub Releases

**Story**: [S003 Rootline Installer](README.md)

## Contexto

Los skills dependen de `rootline` en PATH. En lugar de bundlear binarios en el repo (desaconsejado por la política de Claude Code plugins), el marketplace incluye un `install.sh` que descarga el binario desde GitHub Releases al momento de ejecución.

## Alcance

**In**:
1. `install.sh` POSIX-compatible en raíz del marketplace
2. Detectar OS (linux/darwin) y arquitectura (amd64/arm64)
3. Descargar binario correcto desde GitHub Releases API usando curl (no requiere `gh` CLI)
4. Instalar en `~/.local/bin/rootline` (crear directorio si no existe)
5. Verificar instalación con `rootline --version`
6. Flag `--version vX.Y.Z` para instalar versión específica (default: latest)
7. Mensaje claro si plataforma no soportada o sin conexión

**Out**: Binarios bundled en repo, instalación automática post-skill-install, Windows support

## Estado inicial esperado

- goreleaser releases existentes con binarios multi-plataforma
- Nombres de archive: `rootline_{os}_{arch}.tar.gz`

## Criterios de Aceptacion

- `./install.sh` descarga y coloca binario correcto para linux/darwin (amd64/arm64)
- `rootline --version` funciona después de instalación
- `./install.sh --version v0.9.0` instala versión específica
- Sin conexión → mensaje claro de error (no falla silenciosamente)
- Plataforma no soportada → mensaje claro con lista de plataformas soportadas
- Script no requiere `gh` CLI (usa curl + GitHub API pública)
- Script es POSIX-compatible (no requiere bash)

## Fuente de verdad

- `.goreleaser.yml` (nombres de archive: `rootline_{os}_{arch}.tar.gz`)
- GitHub Releases API (`https://api.github.com/repos/pablontiv/rootline/releases`)
- Patrones de install scripts (rustup, Homebrew)
