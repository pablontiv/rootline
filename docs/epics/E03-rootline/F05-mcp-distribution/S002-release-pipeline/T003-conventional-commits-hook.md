---
estado: Pending
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T003: Git hook para validar conventional commits

**Story**: [S002 Release Pipeline](README.md)

## Contexto

El proyecto usa conventional commits de facto (feat:, fix:, docs:, refactor:, etc.) y `svu` los lee para calcular la version semantica automaticamente. Sin validacion, un commit mal formateado rompe el calculo de version. El hook `commit-msg` intercepta el mensaje antes de que el commit sea creado y lo rechaza si no cumple el formato. El directorio `.githooks/` se versiona en el repo para que el hook este disponible para todos.

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: github-actions
triggers:
  - commit (via git hook commit-msg)
jobs:
  - nombre: validate-commit-msg
    pasos:
      - Leer mensaje del archivo $1 (arg del hook)
      - Validar regex contra patron conventional commits
      - Exit 1 con mensaje de ayuda si no cumple
artefactos:
  - .githooks/commit-msg
```

## Dependencias

- Ninguna

## Alcance

**In**:
1. Crear `.githooks/commit-msg` con script bash que valide el patron: `^(feat|fix|chore|docs|refactor|perf|test|style|ci)(\(.+\))?!?: .+`
2. El hook debe aceptar mensajes multi-linea (solo valida primera linea)
3. Permisos de ejecucion: `chmod +x .githooks/commit-msg`
4. Configurar git: `git config core.hooksPath .githooks`
5. Actualizar paso 4 del loop en `.claude/skills/roadmap/SKILL.md` para especificar que el commit debe seguir conventional commits con el tipo adecuado segun el contenido del task

**Out**: No modificar .pre-commit-config.yaml, no instalar commitizen ni herramientas adicionales

## Estado inicial esperado

- Repo git inicializado
- `.githooks/` directorio no existe (se crea en este task)

## Criterios de Aceptacion

- `git commit -m "mensaje sin tipo"` → rechazado (exit 1 con mensaje explicativo)
- `git commit -m "feat: agrega nueva funcionalidad"` → aceptado (exit 0)
- `git commit -m "fix(index): corrige traversal"` → aceptado
- `git commit -m "feat!: breaking change en CLI"` → aceptado
- `git config core.hooksPath` retorna `.githooks`
- `.githooks/commit-msg` tiene permisos de ejecucion (`ls -la .githooks/commit-msg` muestra `-rwxr-xr-x`)
- El paso 4 del loop en `SKILL.md` menciona conventional commits con guia de tipos

## Fuente de verdad

- `.githooks/commit-msg` — script de validacion
- `.claude/skills/roadmap/SKILL.md` — paso 4 del loop de ejecucion
- `git config core.hooksPath` — configuracion activa
