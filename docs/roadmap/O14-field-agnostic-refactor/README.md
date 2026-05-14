---
tipo: outcome
---
# O14: Field-agnostic refactor

rootline mezcla tipos de almacenamiento con semántica de dominio. roadmapctl hardcodea nombres de campo al parsear resultados de rootline. Este outcome elimina esa mezcla: rootline se convierte en un engine genérico que no conoce "estado", "tipo" ni "blocked_by", y roadmapctl declara el vocabulario en su propia config.

El resultado observable es que rootline puede indexar cualquier corpus de Markdown sin asumir un esquema de dominio, y roadmapctl funciona con cualquier nombre de campo configurado en `.roadmapctl.toml`.

Cambios en rootline: nuevo atributo `source:` en SchemaField, extracción de campos desde body, eliminación de `domain:`, `type: section`, y todos los hardcodings de "estado"/"titulo"/"blocked_by" en los subcomandos. Cambios en roadmapctl: field mapping configurable en `config.go` con defaults retrocompatibles, todos los consumidores actualizados para usar `cfg.Fields.*`.
