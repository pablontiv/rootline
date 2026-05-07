# PR Workflow — Branch per Outcome

Lógica de branching, PR creation y merge para `/roadmap loop --pr`.

> Workspace mode: usar `git -C <repo-path>`. `gh` se ejecuta desde `<repo-path>`.

## Branch & PR Detection

1. Detectar base branch:
   ```bash
   git -C <repo-path> symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@'
   ```
   Fallback: `main`, luego `master`.
2. Verificar `gh`:
   ```bash
   command -v gh && gh auth status
   ```
   Si no disponible, degradar a modo sin PR.
3. Registrar `base_branch`.

## Variables

- `base_branch`
- `current_branch_scope`: Outcome actual o `direct-tasks`
- `prs_created`: `[{number, url, scope, status}]`

## Outcome Setup

Al detectar que la siguiente task pertenece a otro Outcome/direct-task scope:

1. Si hay branch activo anterior, su PR ya debe haberse creado o quedar pendiente de cierre.
2. Derivar branch:
   - Outcome: `feat/OXX-slug`
   - Tasks directas: `feat/direct-roadmap-tasks`
3. Crear branch desde base actualizado:
   ```bash
   git -C <repo-path> checkout <base_branch>
   git -C <repo-path> pull origin <base_branch>
   git -C <repo-path> checkout -b feat/<scope>
   ```
4. Registrar scope activo.

## Outcome PR

Se activa al cambiar de Outcome/direct-task scope o al terminar el loop.

### Push

```bash
git -C <repo-path> push -u origin feat/<scope>
```

### Crear PR

```bash
gh pr create --base <base_branch> --title "<titulo>" --body "$(cat <<'EOF'
## Scope
[OXX: nombre](link al Outcome) o tasks directas

## Cambios
- lista de tasks completadas con commits

## Verificación
- ACs: N/N passed
- Invariantes preservadas
EOF
)"
```

- Título: conventional commit style, ej. `feat(roadmap): O01 simplify planning model`.
- Usar `<pr-merge-strategy>` de `roadmap.local.md`.

### Merge

Modo autónomo:

```bash
gh pr merge <number> --auto --<pr-merge-strategy> --delete-branch
```

Modo interactivo: preguntar si mergear ahora o dejar abierto.

### Post-merge cleanup

```bash
git -C <repo-path> checkout <base_branch>
git -C <repo-path> pull origin <base_branch>
```

Registrar `{number, url, scope, status}` en `prs_created`.
