---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T003: Fix, revision y limpieza completa

**Story**: [S001 Schema Write & Repair](README.md)

## Contexto

Ultimo task del ciclo write. Ejecuta `fix --dry-run` para previsualizar reparaciones, luego `fix` para aplicarlas, y finalmente limpia todo para dejar el homeserver intacto. Este task valida el pipeline completo init→validate→fix y la reversibilidad de las operaciones.

## Dependencias

- T001: generate-stem-file (.stem debe existir)
- T002: validate-with-stem (errores identificados)

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
comando: fix
target: mismo subdirectorio de T001/T002
escritura: si (modifica archivos, luego limpia)
limpieza: rm .stem + git checkout
```

## Alcance

**In**:
1. `rootline fix --dry-run` — previsualizar reparaciones
2. Revisar que los cambios propuestos son coherentes
3. `rootline fix` — aplicar reparaciones
4. Verificar que los archivos fueron modificados
5. Limpiar: `rm .stem` + `git checkout .` en el subdirectorio
6. `git status` en homeserver confirma 0 cambios residuales

**Out**: Cambios permanentes al homeserver

## Criterios de Aceptacion

- `fix --dry-run` muestra cambios propuestos coherentes
- `fix` aplica cambios sin errores
- Limpieza con `rm .stem` + `git checkout` deja el directorio intacto
- `git status` en homeserver muestra working tree clean
- Sin panics ni stack traces
- Exit code 0 en todas las operaciones
