# S001: Schema Write & Repair

**Feature**: [F03 Write Operations](../README.md)
**Estado**: Completado
**Cliente**: Platform Owner
**Capacidad**: Rootline genera .stem, detecta discrepancias, repara, y se limpia todo

## Antes / Despues

**Antes**: El ciclo completo init→validate→fix nunca se ejecuto contra datos externos. No hay evidencia de que el write path funcione end-to-end.

**Despues**: rootline genera .stem en subdirectorio acotado, detecta errores reales (campos faltantes en READMEs sin frontmatter), repara, y la limpieza deja el repo intacto.

## Criterios de Aceptacion (semanticos)

- [x] `init` crea .stem con campos inferidos en subdirectorio acotado
- [x] `validate --all --strict` reporta errores reales con severity en JSON
- [x] `fix` propone cambios coherentes, limpieza deja repo intacto

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-generate-stem-file.md) | Generar .stem con init en subdirectorio acotado |
| [T002](T002-validate-with-stem.md) | Validar con .stem generado, incluyendo --strict |
| [T003](T003-fix-and-cleanup.md) | Fix, revision, y limpieza completa |
