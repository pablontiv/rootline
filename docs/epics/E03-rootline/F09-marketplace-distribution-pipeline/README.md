---
tipo: feature
---
# F09: Agent Marketplace

**Epic**: [E03](../README.md)
**Objetivo**: Repo `agent-marketplace` distribuye skills en formato estándar skills.sh (compatible con cualquier agente AI) y simultáneamente como Claude Code plugin marketplace
**Beneficio**: Skills se instalan con `npx skills add` (Cursor, Copilot, Claude, etc.) o `claude plugin add` (Claude Code); single source of truth en rootline, sync automático
**Milestone**: `npx skills add pablontiv/agent-marketplace` y `claude plugin add pablontiv/agent-marketplace` instalan skills funcionales; push a master sincroniza marketplace en minutos

## Scope

**In**: Repo agent-marketplace con dual format (skills.sh + Claude plugin), sync workflow CI/CD, install script para rootline
**Out**: MCP server distribution (F05), Homebrew tap (F05), skill authoring tools, binarios bundled en repo

## Arquitectura

```
agent-marketplace/
├── .claude-plugin/
│   └── marketplace.json      ← Claude Code plugin grouping
├── skills/
│   ├── rootline-roadmap/
│   │   ├── SKILL.md           ← skills.sh standard (name + description frontmatter)
│   │   └── (archivos soporte)
│   ├── rootline-validate/
│   │   └── SKILL.md
│   ├── rootline-describe/
│   │   └── SKILL.md
│   └── rootline-new-doc/
│       └── SKILL.md
├── install.sh                 ← descarga rootline desde GitHub Releases (S003)
└── README.md
```

**Formato dual**: Cada skill es una carpeta con `SKILL.md` (estándar skills.sh). `marketplace.json` agrupa esos mismos skills como Claude Code plugin. Modelo de referencia: `anthropics/skills`.

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Marketplace Repository Scaffold](S001-marketplace-repository-scaffold/) | Repo agent-marketplace existe con manifest válido, 4 skills, e instrucciones |
| S002 | [Cross-Repo Sync Pipeline](S002-cross-repo-sync-pipeline/) | Push a master sincroniza skills automáticamente al marketplace |
| S003 | [Rootline Installer](S003-rootline-installer/) | Install script descarga rootline desde GitHub Releases |

## Dependencias

- F07 completado (skills existen en .claude/skills/)
- F05/S002 completado (goreleaser release pipeline funcional)
- PAT secret (MARKETPLACE_TOKEN) con scope repo en agent-marketplace

## Fuente de verdad

- `.claude/skills/` (skills a distribuir)
- `anthropics/skills` (modelo de estructura para dual format)
- `.github/workflows/` (patrones de CI/CD existentes)
- `.goreleaser.yml` (matrix de plataformas y nombres de binarios)
