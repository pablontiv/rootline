---
estado: Specified
tipo: task
---
# T005: Remover workflow Scorecard local y bumpear a `crossbeam@v2`

**Contribuye a**: eliminar el 50% startup_failure rate de Scorecard en rootline (y los 4 fixes de emergencia recurrentes por SHA pinning churn). Heredar los cambios saneados de crossbeam.

## Preserva

- INV1: CodeQL, gitleaks, go-release y `docs-validate` siguen funcionando.
- INV2: pre-push hook local + `.coverage-floors.toml` no se tocan.

## Contexto

rootline tiene el caso especial: `scorecard.yml` está **inlined localmente** (no via crossbeam reusable) porque el v1 reusable de crossbeam capa permissions a `read-all` y `scorecard-action` necesita `security-events: write` a nivel de job (ver `rootline/CLAUDE.md` sección "CI Workflows"). La copia local ha requerido 4 fixes de emergencia en 48h (SHA churn) y tiene 50% startup_failure.

La decisión global de O01 (eliminar Scorecard del ecosistema) aplica también a rootline: eliminar la copia inline, no reemplazarla. Independientemente, bumpear referencias `@v1` → `@v2` para heredar coverage default=0.

## Alcance

**In**:
1. Eliminar `.github/workflows/scorecard.yml` (la copia inline local).
2. Bumpear todas las referencias `pablontiv/crossbeam/...@v1` a `@v2`.
3. Actualizar `CLAUDE.md` sección "CI Workflows": remover descripción extensa de scorecard inline (puede mencionar la decisión histórica de removerlo).

**Out**:
- No tocar `docs-validate` (custom job, sigue inline).
- No modificar packaging ni release flow.

## Estado inicial esperado

- `crossbeam@v2` publicado.
- `.github/workflows/scorecard.yml` existe inline.
- Workflows referencian `@v1`.

## Criterios de Aceptación

- `ls .github/workflows/scorecard.yml` retorna "No such file or directory".
- `grep -rE 'pablontiv/crossbeam/.*@v1' .github/workflows/` retorna 0 matches.
- `CLAUDE.md` ya no contiene la sección que describe el workflow scorecard inline (sí puede mencionar histórico).
- Próximo push a master: `gh run list --repo pablontiv/rootline --branch master --limit 5` no muestra `startup_failure`.

## Fuente de verdad

- `/home/shared/rootline/.github/workflows/scorecard.yml`
- `/home/shared/rootline/.github/workflows/ci.yml`
- `/home/shared/rootline/CLAUDE.md`
