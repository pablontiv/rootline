---
estado: Pending
tipo: documentation
ejecutable_en: 1 sesion
---
# T003: Crear .editorconfig y .github/CODEOWNERS

**Story**: [S001 Cleanup and Standards](README.md)

## Contexto

No existe `.editorconfig` — editores diferentes pueden usar tabs vs spaces, line endings diferentes, o trailing whitespace inconsistente. EditorConfig es un estándar soportado nativamente por la mayoría de editores (VSCode, JetBrains, vim, etc.). No existe `CODEOWNERS` — los PRs no tienen reviewers automáticos asignados. Para un proyecto Go, la config es simple: tabs para .go, spaces para .yaml/.md, LF line endings.

## Dependencias

- Ninguna

## Alcance

**In**:
1. Crear `.editorconfig` con reglas para:
   - `*` — charset utf-8, lf line endings, trim trailing whitespace, final newline
   - `*.go` — indent_style tab
   - `*.{yml,yaml,md}` — indent_style space, indent_size 2
   - `Makefile` — indent_style tab
2. Crear `.github/CODEOWNERS` con `* @pablontiv` como owner por defecto

**Out**: Configurar branch protection rules (requiere admin access en GitHub), agregar más codeowners granulares

## Estado inicial esperado

- No existe `.editorconfig`
- No existe `.github/CODEOWNERS`
- `.github/` directorio existe

## Criterios de Aceptacion

- `.editorconfig` existe con secciones para `*`, `*.go`, `*.{yml,yaml,md}`, `Makefile`
- `.editorconfig` usa `indent_style = tab` para Go y `indent_style = space` para YAML/MD
- `.github/CODEOWNERS` existe con al menos una regla de ownership
- Ambos archivos son sintácticamente válidos

## Fuente de verdad

- `.editorconfig` — a crear
- `.github/CODEOWNERS` — a crear
