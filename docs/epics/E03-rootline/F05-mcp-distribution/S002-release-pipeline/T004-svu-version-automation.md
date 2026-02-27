---
ejecutable_en: 1 sesion
estado: Completed
tipo: ci-cd
---
# T004: Instalar svu y documentar flujo de release

**Story**: [S002 Release Pipeline](README.md)

## Contexto

`svu` (semantic version utility, github.com/caarlos0/svu) lee el git log desde el ultimo tag y calcula la siguiente version semantica basandose en conventional commits. Con goreleaser ya configurado (T001), el flujo de release queda como un one-liner: `git tag "$(svu next)" && git push --tags`. Este task instala svu y documenta la convencion de commits y el flujo de release en CLAUDE.md para que el agente AI lo aplique consistentemente.

## Dependencias

- T001 completado (goreleaser configurado)
- T003 completado (conventional commits validados)

## Alcance

**In**:
1. Instalar svu: `go install github.com/caarlos0/svu/v3/cmd/svu@latest`
2. Agregar seccion "## Commit Convention" en `CLAUDE.md` con tabla de tipos → impacto semver
3. Agregar seccion "## Release Flow" en `CLAUDE.md` con el comando `git tag "$(svu next)" && git push --tags`
4. Marcar T001 como Completado (frontmatter `estado: Completado`)

**Out**: No modificar go.mod, no crear .svu.yml (defaults son suficientes)

## Estado inicial esperado

- T001 implementado (.goreleaser.yml y .github/workflows/release.yml existen)
- T003 implementado (git hook activo)
- `~/go/bin` en PATH

## Criterios de Aceptacion

- `svu current` ejecuta sin error y retorna version (ej: `v0.1.0` o `v0.0.0` si no hay tags)
- `CLAUDE.md` contiene seccion "Commit Convention" con tabla feat/fix/chore → minor/patch/none
- `CLAUDE.md` contiene el comando de release con svu
- `T001-goreleaser-config.md` tiene `estado: Completado` en frontmatter

## Fuente de verdad

- `/opt/rootline/CLAUDE.md` — documentacion de convencion y release flow
- `~/go/bin/svu` — binario instalado
- `docs/epics/E03-rootline/F05-mcp-distribution/S002-release-pipeline/T001-goreleaser-config.md`
