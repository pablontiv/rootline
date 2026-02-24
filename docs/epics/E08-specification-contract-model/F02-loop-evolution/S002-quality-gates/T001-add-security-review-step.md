---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Agregar security review selectivo al loop

**Story**: [S002 Quality Gates](README.md)
**Contribuye a**: SKILL.md loop tiene paso de security review selectivo (post-ACs, pre-commit)

## Contexto

El loop de /roadmap ejecuta tasks y hace commit+push sin ningun analisis de seguridad. /security-review es un comando built-in de Claude Code que analiza cambios por vulnerabilidades (injection, auth, secrets, crypto, etc.).

## Alcance

**In**:
1. Agregar variables de estado al inicio de Fase 3: checkpoint_commit (SHA base), checkpoint_task_count (contador), current_story_path (para detectar cambio), checkpoint_interval (default 5)
2. Insertar nuevo paso entre ACs (paso 6) y Commit (paso 5) — reordenar a: ACs → Security Review → Commit
3. Security review aplica si: archivos modificados incluyen patterns sensibles (`**/secret*`, `**/*credentials*`, `**/.env*`, `**/auth*`) O si el tipo de task lo requiere
4. Ejecutar `/security-review` sobre archivos modificados
5. Si findings HIGH → parar (vulnerability pre-push). Si MEDIUM → warning, continuar. Si nada → continuar silenciosamente

**Out**: No agregar /review (eso es T002). No agregar flags (eso es T002).

## Preserva

- INV1: El loop sigue funcionando para tasks que no tienen superficie de ataque
- Verificar: un task tipo documentation no dispara security review

## Estado inicial esperado

- SKILL.md loop tiene pasos: 1-verificar deps, 2-marcar inicio, 3-leer, 4-implementar, 5-commit, 6-ACs, 7-marcar completado, 8-resumen, 9-confirmar, 10-reintentar
- No existe security review en el loop

## Criterios de Aceptacion

- SKILL.md loop tiene variables de estado (checkpoint_commit, checkpoint_task_count, current_story_path)
- Existe paso de security review entre ACs y commit
- Paso describe condiciones de activacion (patterns sensibles)
- Paso describe acciones por severidad (HIGH → parar, MEDIUM → warning)

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md`
