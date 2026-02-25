---
tipo: historia
cliente: Plugin Consumer
---
# S003: Rootline Installer

**Feature**: [F09 Agent Marketplace](../README.md)
**Capacidad**: Marketplace incluye un install script que descarga rootline desde GitHub Releases, eliminando la necesidad de instalación manual

## Antes / Despues

**Antes**: Instalar skills desde marketplace requiere instalar rootline por separado. Skills fallan silenciosamente si rootline no está en PATH.

**Despues**: Marketplace incluye `install.sh`. El usuario lo corre una vez, el script detecta OS/arch, descarga el binario correcto desde GitHub Releases, y lo coloca en `~/.local/bin/`. No se almacenan binarios en el repo (cumple políticas de Claude Code plugins).

## Criterios de Aceptacion (semanticos)

- [ ] Install script descarga binario correcto desde GitHub Releases
- [ ] Soporta linux y darwin (amd64/arm64)
- [ ] Install script detecta OS/arch automáticamente
- [ ] `rootline --version` funciona después de correr el script

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-install-script.md) | Crear install script con descarga desde GitHub Releases |

## Fuente de verdad

- `.goreleaser.yml` (plataformas y nombres de archivos)
- GitHub Releases de rootline (binarios fuente)
