---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T001: Generar .stem file con init en subdirectorio acotado

**Story**: [S001 Schema Write & Repair](README.md)

## Contexto

Primer test de escritura real contra datos externos. Se ejecuta `init` en un subdirectorio acotado (1 Story, no el epic completo) para minimizar blast radius. El .stem generado debe contener los campos inferidos de los tasks del directorio.

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
comando: init
target: subdirectorio Story del homeserver (1 Story acotada)
escritura: si (genera .stem)
limpieza: requerida
```

## Alcance

**In**:
1. Seleccionar 1 Story del homeserver con tasks que tengan frontmatter
2. `rootline init` en ese subdirectorio
3. Verificar que crea `.stem` con campos inferidos
4. `rootline describe` para confirmar que el schema se lee correctamente

**Out**: Modificacion de archivos existentes, operaciones en directorio raiz

## Criterios de Aceptacion

- `init` crea archivo `.stem` en el subdirectorio
- `.stem` contiene campos `estado`, `tipo`, `ejecutable_en`
- `describe` muestra el schema del `.stem` generado
- Exit code 0
- Sin panics ni stack traces
