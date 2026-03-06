---
estado: Pending
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T003: Configurar hooks path, activar timer y verificar

**Story**: [S001 Post-merge Hook y Systemd Timer](README.md)

## Contexto

Con el post-merge hook (T001) y los systemd units (T002) creados, falta configurar git para que encuentre el hook y activar el timer.

## Dependencias

- T001 (post-merge hook debe existir)
- T002 (systemd units deben existir)

## Alcance

**In**:
1. Configurar `git config core.hooksPath .githooks` para que git encuentre post-merge en `.githooks/`
2. `systemctl --user daemon-reload` para que systemd detecte los nuevos units
3. `systemctl --user enable --now rootline-sync.timer` para activar el timer
4. Verificar que todo funciona

**Out**: Monitoreo continuo, alertas de fallos

## Estado inicial esperado

- `.githooks/post-merge` existe y es ejecutable (T001 completado)
- `~/.config/systemd/user/rootline-sync.{service,timer}` existen (T002 completado)
- `core.hooksPath` puede no estar configurado
- Timer no está activo

## Criterios de Aceptacion

- `git config core.hooksPath` retorna `.githooks`
- `systemctl --user is-active rootline-sync.timer` retorna `active`
- `systemctl --user start rootline-sync.service` ejecuta sin error
- `journalctl --user -u rootline-sync.service -n 5` muestra logs del pull

## Fuente de verdad

- `git config --local --list` — verificar core.hooksPath
- `systemctl --user` — verificar estado del timer
