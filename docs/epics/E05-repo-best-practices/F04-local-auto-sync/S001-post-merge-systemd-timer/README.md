---
tipo: historia
cliente: Developer
---
# S001: Post-merge Hook y Systemd Timer

**Feature**: [F04 Local Auto-sync Pipeline](../README.md)
**Capacidad**: El repo hace pull automático cada 5 minutos y reconstruye binario + sincroniza skills si hay cambios

## Antes / Despues

**Antes**: Para actualizar el repo hay que hacer `git pull` manualmente. El rebuild del binario y sync de skills solo ocurre al hacer `git push` (via pre-push hook). Si alguien pushea desde otra máquina, esta máquina queda desactualizada hasta un pull manual.

**Despues**: Un systemd timer ejecuta `git pull --ff-only` cada 5 minutos. Si el pull trae cambios, un post-merge hook reconstruye el binario en `/usr/local/bin/rootline` y sincroniza skills a `~/.claude/skills/rootline/`. Todo automático, sin intervención.

## Criterios de Aceptacion (semanticos)

- [ ] `systemctl --user is-active rootline-sync.timer` retorna `active`
- [ ] Tras push desde otra máquina + esperar 5 min, `rootline --version` refleja el nuevo tag
- [ ] `diff -r .claude/skills/rootline ~/.claude/skills/rootline` no muestra diferencias tras pull con cambios

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-post-merge-hook.md) | Crear post-merge git hook con rebuild y sync de skills |
| [T002](T002-systemd-units.md) | Crear systemd user service y timer para pull automático |
| [T003](T003-activate-verify.md) | Configurar core.hooksPath, activar timer y verificar |

## Fuente de verdad

- `.githooks/pre-push` — código fuente del rebuild + sync (líneas 48-61)
- `~/.config/systemd/user/` — destino de los unit files
