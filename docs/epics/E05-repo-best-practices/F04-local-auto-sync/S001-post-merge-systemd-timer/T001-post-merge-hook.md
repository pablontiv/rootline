---
estado: Obsolete
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T001: Crear post-merge git hook

**Story**: [S001 Post-merge Hook y Systemd Timer](README.md)

## Contexto

El pre-push hook (`.githooks/pre-push`, líneas 48-61) ya contiene la lógica de rebuild del binario y sync de skills, pero solo se ejecuta al hacer push. Se necesita un post-merge hook que ejecute la misma lógica después de un `git pull` que trae cambios.

## Dependencias

- Ninguna

## Alcance

**In**:
1. Crear `.githooks/post-merge` con lógica de sync de skills (`cp -r .claude/skills/rootline/ ~/.claude/skills/rootline/`) y rebuild del binario (`go build -ldflags "-X main.version=..." -o /usr/local/bin/rootline ./cmd/rootline/`)
2. Reutilizar el código exacto de `.githooks/pre-push` líneas 48-61
3. Hacer el archivo ejecutable (`chmod +x`)

**Out**: Modificar el pre-push hook existente, crear lógica de notificación de fallos

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: git-hooks
triggers:
  - post-merge (git pull con cambios)
jobs:
  - nombre: sync-skills
    pasos:
      - Copiar .claude/skills/rootline/ a ~/.claude/skills/rootline/
  - nombre: rebuild-binary
    pasos:
      - go build con ldflags de version a /usr/local/bin/rootline
artefactos:
  - .githooks/post-merge
```

## Estado inicial esperado

- `.githooks/pre-push` existe con lógica de rebuild (líneas 48-61)
- `.githooks/post-merge` NO existe
- `core.hooksPath` puede o no estar configurado (eso es T003)

## Criterios de Aceptacion

- `.githooks/post-merge` existe y es ejecutable
- Ejecutar `.githooks/post-merge` manualmente no produce errores
- El contenido de sync + rebuild es equivalente a `.githooks/pre-push` líneas 48-61

## Fuente de verdad

- `.githooks/pre-push` — código fuente a reutilizar
