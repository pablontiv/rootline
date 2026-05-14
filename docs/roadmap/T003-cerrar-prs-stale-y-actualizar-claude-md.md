---
estado: Specified
tipo: task
---
# T003: Cerrar PRs stale y actualizar CLAUDE.md

**Contribuye a**: Limpiar PRs que no pueden mergearse y corregir docs desactualizadas

## Contexto

**PR #28 — Refactor validation** (+222/-187): Válido en su momento pero ahora irrecuperable.
La rama modifica `internal/mcp/tools.go` que fue eliminado en commit 9f61481 (`feat!: remove Rootline MCP support`, May 9 2026).
La motivación del PR ("ensure consistency between CLI and MCP tools") ya no aplica.
El código de CLI refactoring (internal/validation/pipeline.go) es valioso pero necesita un fresh PR sin las referencias a MCP.

**PR #32 — golang.org/x/text bump**: go.sum tiene conflicts con master. Cerrar y dejar que dependabot regenere.

**CLAUDE.md**: Todavía dice "MCP server complete — all CLI commands and 9 MCP tools functional."
Esto es incorrecto post-commit 9f61481.

## Alcance

**In**:
1. Cerrar PR #28 con comentario: explicar remoción de MCP y sugerir fresh PR solo con CLI validation refactoring
2. Cerrar PR #32 con comentario: conflictos en go.sum, dependabot regenerará automáticamente
3. Editar CLAUDE.md: remover/corregir la mención de MCP como funcional
4. Commit en rootline con el CLAUDE.md corregido

**Out**:
- No restaurar el código MCP
- No intentar rebasar #28 (es tarea del autor crear el fresh PR)

## Estado inicial esperado

- PR #28 y #32 abiertos
- CLAUDE.md menciona MCP como funcional

## Criterios de Aceptación

- PR #28 y #32 no aparecen en `gh pr list --repo pablontiv/rootline --state open`
- `grep -i "mcp" /home/shared/rootline/CLAUDE.md` no muestra referencias a "MCP server complete" o "9 MCP tools"
- `git -C /home/shared/rootline log --oneline -1` muestra el commit de actualización

## Fuente de verdad

- `/home/shared/rootline/CLAUDE.md`
- `gh pr list --repo pablontiv/rootline --state open --json number,title`
