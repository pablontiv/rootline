---
estado: Completed
tipo: task
---
# T007: Activar gate de coverage en pre-push hook

**Outcome**: [O15 Coverage feedback loop](README.md)
**Contribuye a**: bloquear regresiones de coverage antes del push, no en CI (INV3)

[[blocked_by:./T001-raise-internal-proposal-coverage.md]]
[[blocked_by:./T002-raise-internal-fix-coverage.md]]
[[blocked_by:./T003-raise-cmd-rootline-coverage.md]]
[[blocked_by:./T004-raise-internal-templates-coverage.md]]
[[blocked_by:./T005-remove-deprecated-dead-code.md]]
[[blocked_by:./T006-add-coverage-tooling.md]]

## Preserva

- INV3 del outcome: el feedback ocurre antes del push
  - Verificar: simular una regresión (borrar tests) → `git push` se bloquea con mensaje de coverage; restaurar → push pasa
- INV2 del outcome: piso uniforme de 85
  - Verificar: el hook invoca `just coverage-check` que aplica tanto el total como el per-package

## Contexto

`.githooks/pre-push` ya tiene precedentes de gate condicional: valida `docs/roadmap/` si cambian archivos en esa ruta; falla si CLI code cambia sin docs sync. El patrón que se reusa es:

```bash
if git diff --name-only "$range" -- <paths> 2>/dev/null | grep -q .; then
  echo "Checking <thing>..."
  if ! <command> > /tmp/log 2>&1; then
    cat /tmp/log
    echo "<actionable message>"
    exit 1
  fi
fi
```

Aplicado a coverage:

```bash
# Only run coverage check if Go code changed (skip docs-only pushes).
if git diff --name-only "$range" -- '*.go' 2>/dev/null | grep -q .; then
  echo "Checking coverage..."
  if ! just coverage-check > /tmp/rootline-cov.log 2>&1; then
    cat /tmp/rootline-cov.log
    echo "Coverage below threshold. Run: just coverage"
    exit 1
  fi
fi
```

Riesgos:
- **Latencia**: `go test ./... -coverprofile=...` añade ~30s al push. Aceptable; CI tarda más en dar la misma señal.
- **Falsos positivos**: el hook corre sobre el árbol local — si el dev no commiteó tests aún, el push se bloquea. Eso es feature, no bug.
- **Bypass**: `git push --no-verify` lo salta. Documentar que está prohibido bypass salvo emergencia con explicación en commit.

T001-T005 son hard blockers: si se activa el hook con paquetes por debajo de 85, todo push se bloquea inmediatamente (incluyendo el push del propio fix). T006 es hard blocker porque el hook invoca `just coverage-check` que debe existir.

## Alcance

**In**:
1. Editar `.githooks/pre-push` añadiendo el bloque de coverage check después de las validaciones existentes (docs, CLI sync) y antes del install/rebuild de la binary de rootline.
2. Documentar en comentario inline del hook por qué corre solo si cambian `.go`.

**Out**:
- No reescribir el hook completo; solo añadir el bloque nuevo.
- No tocar el pre-commit hook (gofmt/lint son separados).
- No introducir bypass automático.

## Estado inicial esperado

- T001-T005 completadas: `just coverage-check` exit 0 sobre el árbol actual.
- T006 completada: `just coverage-check` recipe existe.
- `.githooks/pre-push` no tiene gate de coverage.

## Criterios de Aceptación

- Crear branch local, borrar un test: `git rm internal/fuzzy/fuzzy_test.go && git commit -m 'test' && git push` → el push falla con mensaje de coverage.
- Restaurar el archivo, retomar: push pasa.
- Push de cambio docs-only (`docs/**/*.md`) no dispara el coverage check.
- `.githooks/pre-push` mantiene comportamiento previo (docs validation, CLI sync, gitleaks, install).

## Fuente de verdad

- `/home/shared/rootline/.githooks/pre-push` — hook actual + nuevo bloque
- `/home/shared/rootline/Justfile` — recipe `coverage-check` (debe existir vía T006)
