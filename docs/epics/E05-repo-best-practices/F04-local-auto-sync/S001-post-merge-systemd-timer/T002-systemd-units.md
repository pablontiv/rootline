---
estado: Obsolete
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T002: Crear systemd user service y timer

**Story**: [S001 Post-merge Hook y Systemd Timer](README.md)

## Contexto

Se necesita un mecanismo que ejecute `git pull` automáticamente cada 5 minutos en el repo. systemd user timers son nativos de Arch Linux, tienen logs via journalctl, y son más robustos que cron.

## Dependencias

- Ninguna (independiente de T001, pero el flujo completo requiere ambos)

## Alcance

**In**:
1. Crear `~/.config/systemd/user/rootline-sync.service` — oneshot service que ejecuta `git pull --ff-only`
2. Crear `~/.config/systemd/user/rootline-sync.timer` — timer que dispara el service cada 5 minutos
3. PATH del service debe incluir `/usr/local/go/bin` para que el post-merge hook pueda compilar

**Out**: Activar el timer (eso es T003), mecanismos de retry o notificación

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: systemd
triggers:
  - timer (OnBootSec=1min, OnUnitActiveSec=5min)
jobs:
  - nombre: rootline-sync.service
    pasos:
      - git pull --ff-only en /home/rootline
artefactos:
  - ~/.config/systemd/user/rootline-sync.service
  - ~/.config/systemd/user/rootline-sync.timer
```

### Service unit

- `Type=oneshot` — ejecuta y termina
- `WorkingDirectory=/home/rootline`
- `ExecStart=/usr/bin/git pull --ff-only` — fast-forward only, falla limpio si hay conflictos
- `Environment=PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin`
- `Environment=HOME=/home/rootline`

### Timer unit

- `OnBootSec=1min` — primera ejecución 1 min después de boot
- `OnUnitActiveSec=5min` — cada 5 min después de la última ejecución
- `Persistent=true` — si el sistema estuvo apagado, ejecuta al volver
- `WantedBy=timers.target`

## Estado inicial esperado

- `~/.config/systemd/user/` puede no existir (crear si falta)
- No hay units de rootline previos

## Criterios de Aceptacion

- `systemd-analyze verify ~/.config/systemd/user/rootline-sync.service` sin errores
- `systemd-analyze verify ~/.config/systemd/user/rootline-sync.timer` sin errores
- Service y timer existen en `~/.config/systemd/user/`

## Fuente de verdad

- systemd.service(5), systemd.timer(5) — man pages de referencia
