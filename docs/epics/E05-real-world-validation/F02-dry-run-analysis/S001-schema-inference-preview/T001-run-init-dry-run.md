---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T001: Ejecutar init --dry-run contra datos del homeserver

**Story**: [S001 Schema Inference Preview](README.md)

## Contexto

El comando `init --dry-run` analiza los archivos de un directorio e infiere un schema `.stem` sin escribir a disco. El homeserver tiene 3 campos reales en frontmatter: `estado`, `tipo`, `ejecutable_en`. Este es el primer test de inferencia contra datos no controlados.

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
comando: init
flags: --dry-run
target: /opt/homeserver/automation/docs/epics/
campos_esperados: [estado, tipo, ejecutable_en]
```

## Alcance

**In**:
1. `rootline init --dry-run /opt/homeserver/automation/docs/epics/` (directorio raiz)
2. `rootline init --dry-run` en subdirectorio con tasks (ej: una Story)
3. Verificar que muestra .stem inferido con los 3 campos
4. Verificar que NO crea archivo en disco

**Out**: Creacion real de .stem, modificacion de archivos

## Criterios de Aceptacion

- Muestra .stem inferido con campos `estado`, `tipo`, `ejecutable_en`
- NO crea archivo .stem en disco (verificar con `ls`)
- Exit code 0
- Sin panics ni stack traces
