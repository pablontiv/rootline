# Type Specs — Deprecated

El modelo simple de roadmap ya no clasifica tasks por tipo técnico.

El campo canónico es:

```yaml
tipo: outcome | task
```

Si una task necesita especificación técnica, incluirla directamente en el cuerpo del archivo bajo una sección libre:

```markdown
## Especificación Técnica

[Detalles suficientes para implementar la task.]
```

No agregar taxonomías obligatorias como `docs`, `ci`, `test` o `refactor` al `.stem` base.
