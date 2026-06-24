# Rootline Fase 1 — Sincerización docs/skill/código post-O14 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hacer que la documentación, el skill y el código de rootline digan la verdad post-O14 (14 detectores reales, `domain:` removido, `apply` deprecado, `type: section` vivo) y documentar el contrato engine/agente.

**Architecture:** Edición de docs + remoción de código muerto. Sin lógica nueva. La verificación de cada task es por `rg` (afirmaciones eliminadas) y/o `just test` verde.

**Tech Stack:** Go 1.25+, just, golangci-lint, rootline CLI.

## Global Constraints

- Cada task termina con `just check` (gofmt + lint + build) y `just test` (`go test ./... -race`) en exit 0.
- Cobertura ≥85% por `.coverage-floors.toml` (las tasks de docs no afectan; la de código muerto no baja cobertura porque borra código sin tests propios).
- Commits convencionales. **NUNCA** agregar `Co-Authored-By` ni atribución AI.
- Gitflow del repo: `direct_push` a master. Reality verificada (no re-derivar): 14 detectores (12 data + 2 gov) en `cmd/rootline/analyze.go:111-162`; `internal/rules/domains.go` no existe; `domain:` removido en O14; `apply` deprecado cubierto por `schema apply`+`repair apply`; `type: section` vivo (`new.go:144-160`), solo el append `+=` fue removido.

**Setup previo (una vez):** El spec quedó commiteado en la rama `chore/sincerize-docs-skill-post-o14`. Confirmar rama de trabajo antes de empezar: `git -C /Users/Shared/harness/rootline branch --show-current`. Trabajar sobre la rama que el usuario indique (master por gitflow, o continuar la chore branch). El skill vive **fuera del repo** en `/Users/pones/.claude/skills/rootline/` — sus ediciones no entran en los commits del repo; commitearlas aparte si el usuario lo pide.

---

### Task 1: Corregir el conteo de detectores a 14 (12 data + 2 governance)

**Files:**
- Modify: `CLAUDE.md:9`, `CLAUDE.md:31`, `CLAUDE.md:38`
- Modify: `README.md:185`
- Modify: `CHANGELOG.md:21`
- Modify: `docs/roadmap/O09-separate-command-responsibilities-and-replace-legacy-apply/T001-codify-command-responsibility-contracts.md:96`

- [ ] **Step 1: Editar CLAUDE.md:9**

Reemplazar:
```
**Status**: CLI engine complete — all core commands functional. 15 inference detectors (13 data + 2 governance). Requires Go 1.25+.
```
por:
```
**Status**: CLI engine complete — all core commands functional. 14 inference detectors (12 data + 2 governance). Requires Go 1.25+.
```

- [ ] **Step 2: Editar CLAUDE.md:31** (solo el fragmento del conteo)

Reemplazar la subcadena:
```
runs 15 detectors (13 data inference + 2 governance: schema coverage, validation gaps).
```
por:
```
runs 14 detectors (12 data inference + 2 governance: schema coverage, validation gaps).
```

- [ ] **Step 3: Editar CLAUDE.md:38** (fragmento del conteo)

Reemplazar la subcadena:
```
Schema inference from existing documents (15 detectors: 13 data + 2 governance).
```
por:
```
Schema inference from existing documents (14 detectors: 12 data + 2 governance).
```

- [ ] **Step 4: Editar README.md:185**

Reemplazar:
```
rootline analyze [path] [--incremental]   # Run 16 detectors (data + governance), produce report
```
por:
```
rootline analyze [path] [--incremental]   # Run 14 detectors (12 data + 2 governance), produce report
```

- [ ] **Step 5: Editar CHANGELOG.md:21**

Reemplazar:
```
- 16 inference detectors (13 data + 3 governance: domain coverage, schema coverage, validation gaps)
```
por:
```
- 14 inference detectors (12 data + 2 governance: schema coverage, validation gaps)
```

- [ ] **Step 6: Editar docs/roadmap/O09-.../T001-...md:96**

Reemplazar la subcadena `16 inference detectors (13 data + 3 governance)` por `14 inference detectors (12 data + 2 governance)`.

- [ ] **Step 7: Verificar que no quedan conteos viejos**

Run:
```bash
rg -n "15 detectors|16 detectors|15 inference|16 inference|13 data inference|13 data \+ 3 governance" CLAUDE.md README.md CHANGELOG.md docs/
```
Expected: sin resultados (exit 1).

- [ ] **Step 8: Verificar que docs/roadmap sigue validando**

Run: `rootline validate --all docs/roadmap/`
Expected: exit 0.

- [ ] **Step 9: Commit**

```bash
git add CLAUDE.md README.md CHANGELOG.md docs/roadmap/O09-separate-command-responsibilities-and-replace-legacy-apply/T001-codify-command-responsibility-contracts.md
git commit -m "docs: correct inference detector count to 14 (12 data + 2 governance)"
```

---

### Task 2: Eliminar la documentación stale de `domain:`

`domains.go` y el campo `domain:` fueron removidos en O14. Eliminar todo lo que los describa como vivos.

**Files:**
- Modify: `README.md:16`, `README.md:137-159` (sección "Domain Types")
- Modify: `CLAUDE.md:31` (oración `describe --by-domain`), `CLAUDE.md:33` (mención `domains.go`)

- [ ] **Step 1: Borrar la fila de tabla en README.md:16**

Eliminar la línea completa:
```
| Domain type | `domain:` property (semantic type) |
```

- [ ] **Step 2: Borrar la sección "Domain Types" en README.md**

Eliminar desde la línea `### Domain Types` hasta el bullet final `- **Governance**: ...governance gaps` inclusive (todo el bloque: encabezado, párrafo, bloque YAML, lista "12 core domains" y bullets "Why domains matter"). Conservar la línea previa `> Sections (\`type: section\`) ...` (es correcta) y el `---`/`## CLI` posteriores.

- [ ] **Step 3: Borrar la oración `describe --by-domain` en CLAUDE.md:31**

Eliminar la subcadena (incluido el espacio previo):
```
 `describe --by-domain` filters schema output by semantic domain.
```

- [ ] **Step 4: Borrar la mención `domains.go` en CLAUDE.md:33**

Eliminar la subcadena:
```
 domain semantic types (`domains.go` — 12 core domains with base type inference, scope-aware field lookup, virtual alias support),
```

- [ ] **Step 5: Verificar la lista de stem-health checks en CLAUDE.md:33**

CLAUDE.md:33 lista "11 checks" incluyendo `domain-type-compat` y `domain-duplicate-scope`. Verificar si esos checks aún existen:
Run: `rg -n "domain-type-compat|domain-duplicate-scope|domainTypeCompat|domainDuplicate" internal/rules/`
- Si **no** aparecen (removidos con domains): editar CLAUDE.md:33 para quitar `domain-type-compat, domain-duplicate-scope` de la lista y corregir el número "11 checks" al conteo real (`rg -c` de los checks reales en `internal/rules/`, p.ej. stem_health).
- Si **sí** aparecen: dejar la lista como está (anotar el hallazgo).

- [ ] **Step 6: Verificar que no quedan claims vivos de domain**

Run:
```bash
rg -n -i "domain:|12 core domains|by-domain|domains\.go|semantic domain" README.md CLAUDE.md
```
Expected: sin claims que presenten `domain:` como feature vigente (puede quedar prosa histórica solo si dice explícitamente "removed").

- [ ] **Step 7: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: remove stale domain: subsystem references (removed in O14)"
```

---

### Task 3: Eliminar código muerto (`missing_domain` + tumba `subschema_detection.go`)

**Files:**
- Modify: `cmd/rootline/analyze.go:44`
- Modify: `internal/e2e/analyze_test.go:42`
- Delete: `internal/infer/subschema_detection.go`
- Modify: `CLAUDE.md:38` (mención `subschema_detection.go`)

**Interfaces:**
- Produces: nada nuevo. `missing_domain` no es emitido por ningún detector (ver `cmd/rootline/analyze.go:111-162`), así que removerlo del map `agentRequiredTypes` no cambia comportamiento.

- [ ] **Step 1: Verificar baseline verde**

Run: `just test`
Expected: exit 0 (todo verde antes de tocar).

- [ ] **Step 2: Confirmar que `missing_domain` no lo emite ningún detector**

Run: `rg -n "missing_domain" internal/ cmd/`
Expected: solo dos hits — `cmd/rootline/analyze.go:44` y `internal/e2e/analyze_test.go:42`. Ningún `Type: "missing_domain"` productor.

- [ ] **Step 3: Borrar la línea en cmd/rootline/analyze.go:44**

Eliminar la línea dentro del map `agentRequiredTypes`:
```go
	"missing_domain":          true,
```

- [ ] **Step 4: Borrar la clave en internal/e2e/analyze_test.go:42**

Eliminar la línea correspondiente del fixture:
```go
		"missing_domain":                true,
```

- [ ] **Step 5: Borrar el archivo-tumba**

Run: `git rm internal/infer/subschema_detection.go`
(El archivo solo contiene un comentario documentando la remoción en O14; no exporta nada.)

- [ ] **Step 6: Quitar la mención en CLAUDE.md:38**

Eliminar la subcadena de la lista de body-aware detectors:
```
`subschema_detection.go` (per-type field groups), 
```
(quitar también la coma/espacio para no dejar lista mal formada).

- [ ] **Step 7: Verificar verde post-cambio**

Run: `just test`
Expected: exit 0.

- [ ] **Step 8: Verificar build/lint**

Run: `just check`
Expected: exit 0 (sin referencias colgadas a `subschema_detection`).

- [ ] **Step 9: Commit**

```bash
git add cmd/rootline/analyze.go internal/e2e/analyze_test.go CLAUDE.md
git rm internal/infer/subschema_detection.go
git commit -m "refactor: remove dead missing_domain refs and subschema_detection tombstone"
```

---

### Task 4: Corregir las afirmaciones falsas sobre `type: section`

`type: section` está vivo (genera sección en `new`, lo parsea `rules.go`, lo maneja `set.go`). Solo el append `+=` fue removido. Corregir el mensaje de error y el skill.

**Files:**
- Modify: `cmd/rootline/set.go:263`
- Test: `cmd/rootline/set_test.go` (o el test que cubra el mensaje `+=`)
- Modify (skill, fuera del repo): `/Users/pones/.claude/skills/rootline/SKILL.md:109`

- [ ] **Step 1: Localizar tests que afirmen el mensaje viejo**

Run: `rg -n "type:section has been removed|no longer supported" cmd/rootline/`
Expected: identifica `cmd/rootline/set.go:263` y cualquier test que assert-ee ese string.

- [ ] **Step 2: Si hay un test del mensaje, actualizar la expectativa primero (red)**

Si un test verifica `"append (+=) is no longer supported (type:section has been removed)"`, cambiar la expectativa del test a `"append (+=) is no longer supported"`.
Run: `go test ./cmd/rootline/ -run TestSet -v`
Expected: FAIL (el código aún emite el string viejo).
(Si no existe tal test, saltar al Step 3.)

- [ ] **Step 3: Corregir el mensaje en set.go:263**

Reemplazar:
```go
		return proposal.Proposal{}, fmt.Errorf("append (+=) is no longer supported (type:section has been removed)")
```
por:
```go
		return proposal.Proposal{}, fmt.Errorf("append (+=) is no longer supported")
```

- [ ] **Step 4: Verificar verde**

Run: `go test ./cmd/rootline/ -race`
Expected: exit 0.

- [ ] **Step 5: Smoke test de que `type: section` sigue funcionando**

Run:
```bash
d=$(mktemp -d); cd "$d"; git init -q
printf 'version: 2\nscope:\n  match: "*.md"\nschema:\n  titulo:\n    type: string\n    required: true\n  notas:\n    type: section\n    heading: "## Notas"\n    required: true\n' > .stem
rootline new doc.md >/dev/null 2>&1; cat doc.md; cd - >/dev/null
```
Expected: `doc.md` contiene `## Notas` y `<!-- TODO: Add content -->`.

- [ ] **Step 6: Corregir SKILL.md:109** (archivo del skill, fuera del repo)

Reemplazar:
```
Note: `type: section` and section append (`+=`) are removed; use `source: body.section[...]` + `type: string` in the `.stem` instead.
```
por:
```
Note: section append (`+=`) is removed. `type: section` remains supported (scaffolds a body section in `new`); `source: body.section[...]` + `type: string` is an alternative.
```

- [ ] **Step 7: Commit (repo)**

```bash
git add cmd/rootline/set.go cmd/rootline/set_test.go
git commit -m "fix: correct += error message; type:section is still supported"
```
(Incluir `set_test.go` solo si se editó en Step 2.)

---

### Task 5: Documentar el contrato engine/agente y la guía de `apply` en el skill

**Files:**
- Modify: `README.md` (sección `## AI-Native`, ~`:298`)
- Modify (skill, fuera del repo): `/Users/pones/.claude/skills/rootline/SKILL.md:118`, `:127`

- [ ] **Step 1: Agregar el contrato engine/agente en README, sección AI-Native**

Tras el párrafo de `### CLI-first automation` (~`:304`), agregar:
```markdown

### Engine vs. agent: division of labor

Rootline's engine decides everything resolvable **from form** — frequency
thresholds (a field present in ≥80% of records is `required`), unanimous or
majority value agreement, and structural conventions (directory naming, type
consistency). Decisions that need **meaning** — is this value semantically the
same as that one? — are not guessed: `analyze` marks those proposals
`requires_agent` for a human or agent to resolve. The report exposes
**percentage evidence, not opinions**; consumers apply their own thresholds.
```

- [ ] **Step 2: Verificar el contrato en README**

Run: `rg -n "requires_agent|percentage evidence|division of labor" README.md`
Expected: al menos un hit.

- [ ] **Step 3: Corregir la guía de `apply` en SKILL.md:118** (skill)

Reemplazar:
```
Do not apply results automatically. For `apply`, inspect the report first and treat the command as a write to `.stem` and documents.
```
por:
```
Do not apply results automatically. Inspect the report first, then use `rootline schema apply --report <file>` for schema-surface proposals (writes `.stem`) or `rootline repair apply --report <file>` for document repairs (writes frontmatter only). The legacy `apply` command is deprecated.
```

- [ ] **Step 4: Corregir la línea de referencia en SKILL.md:127** (skill)

Reemplazar:
```
- Graph, migrate, init, analyze, apply: `ref-advanced.md`
```
por:
```
- Graph, migrate, init, analyze, schema apply, repair apply: `ref-advanced.md`
```

- [ ] **Step 5: Commit (repo)**

```bash
git add README.md
git commit -m "docs: document engine/agent division of labor in README"
```

---

## Self-Review (cobertura del spec — Fase 1)

- Conteo 14 detectores → Task 1 ✓ (CLAUDE.md, README, CHANGELOG, roadmap T001)
- Remover docs `domain:` stale → Task 2 ✓ (README Domain Types + tabla; CLAUDE.md by-domain + domains.go)
- Código muerto `missing_domain` + tumba subschema → Task 3 ✓
- Corregir `type: section` (set.go:263 + SKILL.md) → Task 4 ✓
- Contrato engine/agente en README + guía apply en SKILL → Task 5 ✓

Sin placeholders: cada edición muestra texto exacto antes/después; cada task verifica con `rg`/`just test`/`just check`/`rootline validate`. Fases 2-4 reciben su propio plan (Fase 4 requiere exploración de código adicional antes de pasos sin placeholder).
```
