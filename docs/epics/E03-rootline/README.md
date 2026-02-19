# E03: Rootline — File-Based Documentation Engine

**Estado**: Activa
**Metrica de exito**: `rootline validate docs/` y `rootline query --where 'estado eq Pending'` producen resultados correctos sobre el repositorio homeserver
**Timeline**: 2026-Q1 — en curso

## Intencion

Construir y publicar Rootline, un CLI standalone en Go que trata el filesystem como base de datos de documentacion. Rootline reemplaza los 7 sistemas de parsing independientes del homeserver automation con un unico motor de extraccion, validacion y query basado en archivos `.stem` con herencia parent-to-child.

El producto resuelve un problema concreto (logica dispersa en skills/hooks) y simultaneamente funciona como portfolio piece para audiencias DevOps/SRE, Software Engineering, y AI/LLM Engineering.

## Features

| ID | Nombre | Descripcion |
|----|--------|-------------|
| F01 | [Project Foundation](F01-project-foundation/) | Go module, cobra skeleton, CI pipeline |
| F02 | [Core Engine](F02-core-engine/) | .stem parser, merge algorithm, Markdown extractor, file scanner |
| F03 | [Validation and Schema](F03-validation-schema/) | Rules engine, validate command, describe command |
| F04 | [Query and Presentation](F04-query-presentation/) | Query engine, query command, tree and stats commands |
| F05 | [MCP Server and Distribution](F05-mcp-distribution/) | JSON-RPC MCP server, goreleaser, Homebrew tap |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| F01 | D11 (GitHub org/user) resuelto | Foundation — todo depende de un proyecto buildable |
| F02 | F01 | Core pipeline es la base para todos los comandos |
| F03 | F02 | Primer valor visible al usuario (validacion + describe) |
| F04 | F02 | Parallelizable con F03 (consumidores independientes del core) |
| F05 | F03, F04 | Wraps funcionalidad completada (MCP + releases) |

## Decision Log

| Fecha | Decision | Razon |
|-------|----------|-------|
| 2026-02-17 | Go como lenguaje (D1) | Portfolio signal para 3 audiencias; coherencia con IaC; menor riesgo abandono |
| 2026-02-17 | `.stem` como rules file (D3) | Metafora: "the stem is where everything grows from" |
| 2026-02-17 | MCP como protocolo (D7) | Single protocol layer. CLI calls engine directly. |
| 2026-02-17 | Solo Markdown built-in (D28) | Core pequeno. Otros formatos son plugins (I2). |

## Gaps Activos

- **D11 (GitHub org/user)**: Bloquea F01. Pones debe decidir antes de `go mod init`.
- **Dogfooding**: Migracion de consumers del homeserver pertenece a E02, no aqui.
- **`explain` command**: Diferido (I4). Se agrega cuando derivation (I3) exista.
- **Derivation engine**: Diferido (I3). Pipeline slot reservado.

## Referencias

- [Intent Document v0](../../rootline-intent-v0.md) — vision y decisiones
- [I1: Query Operators](../../research/I1-query-operators.md)
- [I5: Describe Contract](../../research/I5-describe-contract.md)
- [I7: Extractors Architecture](../../research/I7-extractors-architecture.md)
- [Source code](../../../src/rootline/) — proyecto standalone
