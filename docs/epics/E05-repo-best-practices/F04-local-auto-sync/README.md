---
tipo: feature
---
# F04: Local Auto-sync Pipeline

**Epic**: [E05](../README.md)
**Objetivo**: El repo local se mantiene sincronizado con origin sin intervención manual, reconstruyendo el binario y sincronizando skills tras cada pull con cambios
**Beneficio**: Elimina la necesidad de hacer pull + rebuild manual; el binario y skills siempre están al día
**Milestone**: systemd timer activo que ejecuta `git pull` cada 5 minutos, con post-merge hook que reconstruye automáticamente

## Scope

**In**: post-merge git hook, systemd user timer y service, configuración de core.hooksPath
**Out**: Mecanismos de notificación de fallos, auto-resolve de conflictos, pull de múltiples repos

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-post-merge-systemd-timer/) | Post-merge Hook y Systemd Timer | El repo hace pull automático cada 5 min y reconstruye binario + skills si hay cambios |

## Dependencias

- Ninguna

## Fuente de verdad

- `.githooks/pre-push` — código de rebuild + sync a reutilizar (líneas 48-61)
- `/usr/local/bin/rootline` — binario destino del rebuild
- `~/.claude/skills/rootline/` — destino del sync de skills
