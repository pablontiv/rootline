---
estado: Specified
tipo: historia
cliente: Platform Owner
---
# S002: Release Pipeline

**Feature**: [F05 MCP Server and Distribution](../README.md)
**Capacidad**: Tags en el repositorio producen binarios multi-plataforma y formula Homebrew automaticamente

## Antes / Despues

**Antes**: No hay mecanismo de distribucion. El binario solo existe localmente. No hay forma de instalar Rootline en otra maquina sin compilar desde source.

**Despues**: `git tag v0.1.0 && git push --tags` trigger goreleaser que produce binarios para Linux/macOS/Windows (amd64/arm64). Homebrew tap permite `brew install org/tap/rootline`. Zero-friction installation.

## Criterios de Aceptacion (semanticos)

- [ ] Tag push produce release con binarios multi-plataforma
- [ ] `brew install` instala rootline correctamente
- [ ] Release incluye checksums y changelog

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-goreleaser-config.md) | Configurar goreleaser y GitHub Actions release workflow |
| [T002](T002-homebrew-tap.md) | Crear Homebrew tap y formula |
| [T003](T003-conventional-commits-hook.md) | Git hook commit-msg para validar conventional commits |
| [T004](T004-svu-version-automation.md) | Instalar svu y documentar flujo de release en CLAUDE.md |

## Fuente de verdad

