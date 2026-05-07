# /roadmap loop [--filter PATTERN] [--max N] [--pr]

> **Pre-requisito**: Leer [common-logic.md](common-logic.md) para auto-numbering, cascading links y comandos rootline.

Ejecutar Tasks pendientes en loop con confirmacion entre cada uno.

**Opciones**:
- `--filter PATTERN`: Filtrar por path (ej: `E02/F04`, `E01`)
- `--max N`: Limitar a N tasks
- `--checkpoint-interval N`: Intervalo de tasks entre checkpoints de calidad (default 5)
- `--skip-reviews`: Desactivar quality gates (security review y checkpoint review)
- `--pr`: Crear feature branch por Story y PR por Story (requiere `gh` CLI). Sin este flag, push depende de `<auto-push>` config.

## Workspace mode

En workspace mode, el loop opera en **un solo repo a la vez** (default).

- **Con `--repo <name>`** (o ya resuelto en bootstrap): proceder como single-repo
  usando `<abs-roadmap-root>` y `<repo-path>` de ese repo. Todos los `git` commands
  usan `git -C <repo-path>` (ej: `git -C /opt/backscroll add ...`).

- **Sin `--repo`**: ejecutar discovery rápida y pedir selección:
  1. Para cada repo en `<repos>`, ejecutar en paralelo:
     `rootline tree <abs-roadmap-root>/ --where '<where-leaf> && <where-not-done>' --output table`
  2. Contar pendientes por repo
  3. Pedir selección con `AskUserQuestion`:
     "En qué repo querés ejecutar el loop?"
     Opciones: repos con pendientes > 0 (mostrar conteo)
  4. Una vez seleccionado → proceder como single-repo con ese repo

## Fase 1: Discovery

1. Ejecutar `rootline graph --check <roadmap-root>/` para validar dependencias antes de empezar
   - Si hay ciclos → reportar y **parar** (dependencias circulares impiden ejecucion)
   - Si hay broken links → reportar como warning (pueden ser tasks aun no creados)
2. Ejecutar `rootline query <roadmap-root>/ --where "<where-leaf>" --where "<where-active>" --output table` para obtener tasks pendientes
3. Si `--filter PATTERN` proporcionado, filtrar resultados por Epic/Feature path match
4. Si `--max N`, tomar solo los primeros N tasks
5. Mostrar tabla de tasks encontradas al usuario
6. **Si 0 tasks encontradas** → informar:
   "No hay tasks materializadas en `<roadmap-root>/`.
   Ejecutar `/roadmap plan` para crear los archivos de task primero."
   → **STOP**. NO implementar desde el contexto de conversación.

## Fase 2: TodoList Setup

Para cada task encontrada, crear entrada con `TaskCreate`:
- **subject**: `TXXX: titulo`
- **description**: `Path: <filepath> | Tipo: <tipo>`
- **activeForm**: `Implementando TXXX`

Mostrar TodoList con `TaskList`.

## Fase 2.5: Branch & PR Setup

Solo si `--pr`. Sin este flag → skip esta fase entera (push depende de `<auto-push>`).

Si `--pr` → leer [pr-workflow.md](pr-workflow.md) y ejecutar **Branch & PR Detection**.

## Fase 3: Loop de Ejecucion

**Variables de estado del loop:**
- `checkpoint_commit`: SHA del ultimo checkpoint (inicializar con HEAD al inicio)
- `checkpoint_task_count`: Contador de tasks desde ultimo checkpoint (inicializar en 0)
- `current_story_path`: Path de la Story actual (para detectar cambio de contexto)
- `checkpoint_interval`: Intervalo entre checkpoints (default 5, configurable con --checkpoint-interval)

### Story context change (trigger: cambio de Story)

Al detectar que el task actual pertenece a una Story diferente a `current_story_path`:

- **Default (sin `--pr`)**: Solo actualizar `current_story_path`, continuar normalmente.
- **`--pr` mode**: Ejecutar **Story Setup** de [pr-workflow.md](pr-workflow.md).

---

Para cada task en orden:

1. **Verificar dependencias**: Leer el archivo .md del task y buscar `[[blocks:TXXX-name]]` en el body.
   Para cada dependencia encontrada:
   - Buscar el task referenciado y verificar que su frontmatter tiene `estado` con valor en `<done-statuses>`
   - Si alguna dependencia no esta en `<done-statuses>` → **skip** con mensaje: `Bloqueado por: TXXX (estado: <valor>)`
   - Tasks bloqueados se reintentaran al final de la cola

2. **Marcar inicio**: `TaskUpdate` → status: `in_progress`

3. **Leer Task**: `Read` del archivo .md completo para entender que pide

3.5. **Prior attempts** (opcional, si `command -v backscroll >/dev/null 2>&1`):
   `backscroll search "TASK_TITLE_KEYWORDS" --robot --max-tokens 1000`
   Si este task fue intentado antes y revertido/bloqueado, surfacear ese contexto.

4. **Implementar**:
   - Si el Task tiene `tipo:` en frontmatter que corresponde a un skill
     conocido del proyecto, invocarlo via `Skill` tool
   - Si no tiene skill asociado, implementar directamente siguiendo
     las instrucciones del Task
   - Consultar `.claude/roadmap.local.md` (si existe) para templates de
     especificacion tecnica del proyecto, o [type-specs.md](type-specs.md) como fallback

5. **Verificar ACs**:
   - Leer seccion "Criterios de Aceptacion" del Task .md
   - Ejecutar CADA verificacion documentada (comandos, checks, observables)
   - Reportar resultado por AC: PASS / FAIL
   - Si algun AC falla → reportar y **parar** (bug encontrado)
   - Leer seccion "Preserva" del Task .md (si existe)
   - Para cada invariante listado en Preserva: ejecutar su comando/procedimiento de verificacion
   - Reportar resultado: INV1 HOLDS / INV2 VIOLATED
   - Si algun invariante se viola → **parar** (igual que AC fail)

6. **Verificacion de cierre de Story** (si es el ultimo task de la Story):
   - Determinar si es el ultimo task: no quedan tasks pendientes en la misma Story (todas las demas estan en `<done-statuses>`)
   - Leer criterios semanticos (seccion "Criterios de Aceptacion" o "Despues") del README.md de la Story padre
   - Ejecutar `<story-close-cmds>` de `roadmap.local.md` (si existen)
   - Reportar resultado por comando: PASS / FAIL
   - **Warning informativo, no bloquea** el loop — el usuario decide si actuar

7. **Security Review** (selectivo, post-ACs, pre-commit):
   - Aplica si: archivos modificados incluyen patterns sensibles (`**/secret*`, `**/*credentials*`, `**/.env*`, `**/auth*`, `**/crypto*`) O si el tipo de task lo requiere
   - Si aplica: ejecutar `/security-review` sobre archivos modificados
   - Si findings HIGH → **parar** (vulnerabilidad pre-push). Reportar findings y detener loop
   - Si findings MEDIUM → warning informativo, continuar
   - Si nada o no aplica → continuar silenciosamente

8. **Commit** (centralizado, NO delegado a skills hijos):
   - Identificar archivos modificados/creados por la implementacion
   - `git add` archivos relevantes (especificos, no `git add .`)
   - **Formato del mensaje** (segun `<commit-style>`):
     - **`conventional`** (default): `type(scope): description`
       - Elegir `type` segun el contenido del task: `feat` (nueva funcionalidad), `fix` (correccion), `test` (tests), `docs` (documentacion), `refactor` (reestructuracion), `ci` (CI/CD), `chore` (mantenimiento), `perf` (rendimiento), `style` (formato)
       - El hook `.githooks/commit-msg` rechazara mensajes que no sigan el formato
     - **`terse`**: descripcion corta en minusculas, sin type ni scope. Ej: `add user validation`, `fix retry logic on timeout`. Maximo ~50 caracteres.
   - **Push policy** (segun `<auto-push>` y `--pr`):
     - **`<auto-push>: true` (default), sin `--pr`**: Push inmediato tras commit: `git push`
     - **`<auto-push>: false`, sin `--pr`**: NO push. Los commits se acumulan localmente.
     - **`--pr` mode** (independiente de `<auto-push>`): NO push aqui. Push ocurre en Story PR ([pr-workflow.md](pr-workflow.md)).

9. **Marcar completado**: `TaskUpdate` → status: `completed`

10. **Resumen de iteracion**:
    ```
    ITERACION N/TOTAL
    ├─ Task: TXXX - titulo
    ├─ Resultado: PASS/FAIL
    ├─ ACs: N/M passed
    ├─ Commit: hash
    └─ Siguiente: TXXX+1 - titulo
    ```

11. **Checkpoint Detection** (post-resumen, pre-confirmacion):
    - Incrementar `checkpoint_task_count`
    - Triggers (OR — cualquiera activa el checkpoint):
      a) **Story context change**: siguiente task pertenece a otra Story — si `--pr`, triggers Story Setup + Story PR ([pr-workflow.md](pr-workflow.md))
      b) **Safety net**: `checkpoint_task_count >= checkpoint_interval` (default 5)
      c) **Loop interrumpido**: usuario elige "Parar" en la confirmacion
    - Al activar checkpoint:
      1. Calcular diff acumulado: `git diff <checkpoint_commit>..HEAD`
      2. Ejecutar `/review` sobre el diff acumulado
      3. Reportar findings (informativos, **no bloquean** el loop)
      4. Registrar nuevo checkpoint: `checkpoint_commit = HEAD`, `checkpoint_task_count = 0`
      5. Si trigger fue Story context change y `--pr` → ejecutar **Story PR** de [pr-workflow.md](pr-workflow.md) antes de continuar

> **Al terminar el loop** (todos los tasks procesados o usuario dice "Parar"): si `--pr` y hay commits pendientes en el feature branch actual → ejecutar **Story PR** de [pr-workflow.md](pr-workflow.md).

12. **Confirmar**: `AskUserQuestion` con opciones:
    - Si, continuar (Recommended)
    - Saltar siguiente y continuar
    - Parar aqui

13. **Reintentar bloqueados**: Al terminar la cola, si quedan tasks que fueron skipped por dependencias bloqueadas y ahora sus dependencias estan Completadas → reintentar. Si ningun task progreso en la pasada → parar (deadlock de dependencias).

## Fase 4: Resumen Final

Al terminar todas las tasks o al parar:

```
RESUMEN LOOP
├─ Tasks completadas: N/TOTAL
├─ Tasks saltadas: M
├─ ACs: total passed / total
├─ Security reviews: N ejecutados, M findings (H: X, M: Y)
├─ Quality checkpoints: N ejecutados, M findings
├─ Pull Requests: (solo --pr)
│  ├─ #42 S043-multi-path-config — MERGED
│  ├─ #43 S044-directory-discovery — MERGED
│  └─ #44 S045-resume-subcommand — OPEN (CI pending)
├─ Commits: lista de hashes
└─ Tasks restantes: lista (si las hay)
```
