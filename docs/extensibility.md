---
estado: Completado
---
# Extensibility

Rootline is built around **extractors**.

Markdown is the built-in extractor.

The architecture is designed so that other extractors can be added
without changing the core model. Future extractors may include:

- YAML / JSON / TOML files
- MDX
- API specifications (OpenAPI, AsyncAPI)

All extractors feed the same pipeline:
rules by directory, inheritance, validation, derivation, querying.

> LSP integration has been considered but carries very high complexity.
> It is not in scope.
