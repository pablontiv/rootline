# PR Workflow — Branch per Story

Lógica de branching, PR creation, y merge para `/roadmap loop --pr`.

> **Workspace mode**: Todos los `git` commands en este archivo usan `git -C <repo-path>`.
> En single-repo mode, `<repo-path>` = `.` (equivalente a `git` sin `-C`).
> `gh` commands se ejecutan desde `<repo-path>`: `cd <repo-path> && gh pr create ...`

## Branch & PR Detection

1. **Detectar base branch**:
   ```bash
   git -C <repo-path> symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@'
   ```
   Fallback: probar `main`, luego `master`:
   ```bash
   git -C <repo-path> rev-parse --verify origin/main 2>/dev/null && echo main || echo master
   ```
2. **Verificar `gh` CLI**:
   ```bash
   command -v gh && gh auth status
   ```
   Si no disponible → warning, degradar a modo sin PR automáticamente:
   > "`gh` CLI no disponible o no autenticado. Degradando a modo sin PR."
   → Volver a loop-subcommand.md y seguir como default (push segun `<auto-push>`).
3. Registrar `base_branch` en variables de estado.

## Variables de estado adicionales (PR mode)

- `base_branch`: Branch base para PRs (auto-detected arriba)
- `prs_created`: Lista de PRs creados `[{number, url, story_id, status}]`

## Story Setup (trigger: cambio de Story)

Al detectar que el task actual pertenece a una Story diferente a `current_story_path`:

1. Si hay feature branch activo de una Story anterior → Story PR ya fue creado (cleanup en post-merge ya ejecutó checkout a base)
2. Derivar branch name: `feat/<story-id>-<story-slug>`
   - Ej: `feat/S043-multi-path-config` (slug = título de la Story en kebab-case, max 40 chars)
3. Crear feature branch desde base actualizado:
   ```bash
   git -C <repo-path> checkout <base_branch>
   git -C <repo-path> pull origin <base_branch>
   git -C <repo-path> checkout -b feat/<story-id>-<story-slug>
   ```
4. Actualizar `current_story_path` y registrar branch activo

## Story PR (trigger: story boundary o fin de loop)

Se activa cuando el checkpoint detecta cambio de Story O cuando el loop termina. Reemplaza el concepto de "push per task" con "PR per story".

### 1. Push branch

```bash
git -C <repo-path> push -u origin feat/<story-id>-<story-slug>
```

### 2. Crear PR

```bash
gh pr create --base <base_branch> --title "<titulo>" --body "$(cat <<'EOF'
## Story
[SXXX: nombre](link al README de la Story)

## Cambios
- lista de tasks completados con sus commits

## Verificacion
- ACs: N/N passed
- Invariantes: todos preservados
- `just check` + `just test` pasan
EOF
)"
```
- Título: conventional commit style, ej: `feat(config): S043 multi-path config`
- Usar `<pr-merge-strategy>` de `roadmap.local.md` (default: `squash`)

### 3. Merge

- **Modo autónomo** (no `AskUserQuestion` disponible o usuario pidió autonomía):
  ```bash
  gh pr merge <number> --auto --<pr-merge-strategy> --delete-branch
  ```
  Esperar merge con timeout:
  ```bash
  TIMEOUT=300  # 5 minutos
  ELAPSED=0
  while [ "$(gh pr view <number> --json state -q .state)" != "MERGED" ]; do
    if [ $ELAPSED -ge $TIMEOUT ]; then
      echo "Timeout: PR #<number> not merged after ${TIMEOUT}s"
      break
    fi
    sleep 15
    ELAPSED=$((ELAPSED + 15))
  done
  ```
  Si timeout → reportar PR status con `gh pr checks <number>` y **parar** el loop.

- **Modo interactivo**:
  `AskUserQuestion`: "PR #N creado: <url>. Merge ahora o dejarlo abierto?"
  - Merge → ejecutar merge + esperar
  - Dejar abierto → registrar y continuar (siguiente Story puede fallar por dependencias)

### 4. Post-merge cleanup

```bash
git -C <repo-path> checkout <base_branch>
git -C <repo-path> pull origin <base_branch>
```
Esto asegura que la siguiente Story parte del código actualizado.

### 5. Registrar

En `prs_created`: `{number, url, story_id, status: "merged"|"open"}`
