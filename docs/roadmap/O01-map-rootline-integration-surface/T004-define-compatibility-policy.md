---
estado: Completed
tipo: task
---
# T004: Define compatibility expectations between rootline CLI versions and the Pi extension.

**Outcome**: [O01 Map Rootline integration surface](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T003-classify-pi-exposure.md]]

## Preserva

- INV1: No Pi tool is implemented before its Rootline command contract and risk class are documented.
  - Verificar: Check downstream tasks reference this Outcome as source of truth.

## Contexto

Esta task forma parte de O01 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Define compatibility expectations between rootline CLI versions and the Pi extension.

## Alcance

**In**:
1. A compatibility policy exists for CLI version detection and unsupported commands.
2. The policy defines failure messages for missing or too-old rootline binaries.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T003-classify-pi-exposure.md` está completada o su salida está disponible.

## Criterios de Aceptación

- A compatibility policy exists for CLI version detection and unsupported commands.
- The policy defines failure messages for missing or too-old rootline binaries.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `install.sh`
- `.goreleaser.yml`
- `README.md`

## Política de Compatibilidad Pi–Rootline

### Detección de Versión

Pi should detect the installed rootline version using:

```bash
rootline --version
```

This outputs a semver tag (e.g., `v0.9.100`). Parsing strategy:
1. Execute `rootline --version` and capture stdout
2. Strip leading `v` prefix if present
3. Parse as semantic version: `MAJOR.MINOR.PATCH[+metadata]`
4. Compare using semver rules (e.g., v0.9.0 < v0.10.0 < v1.0.0)

### Versión Mínima Requerida

**Minimum rootline version: v0.9.0**

This minimum is derived from:
- All 11 Pi Tools (analyze, describe, explain, graph, query, stats, tree, validate, fix --all, migrate --from variants) have stable JSON contracts with `version: 1` as of v0.9.0
- JSON contract stability is the primary blocker for agent-safe tool use
- v0.9.0 marks the point where core read-only commands emit versioned JSON envelopes

Pi should enforce this check at startup and reject older versions explicitly.

### Comandos sin Contrato Estable

**Policy for Unsupported Commands** (from T003 classification):

Commands marked **Unsupported** in T003 (trace, fix [single file], init, migrate --rename, migrate --scaffold, migrate --split, new, set) should:

1. **Fail explicitly** — Pi must not invoke these commands silently or fall back
2. **User-friendly error message** — output a diagnostic that explains the issue and points to the roadmap
3. **No silent partial functionality** — do not attempt workarounds (e.g., using shell text parsing)
4. **Link to contract issue** — reference T003 classification for context

Example flow for an unsupported command:
```
User attempts: /pi new docs/api/auth.md
Pi detects: Command 'new' is unsupported
Pi outputs: "⚠ Command 'rootline new' is not yet stable for agent use.
            See docs/roadmap/O01-map-rootline-integration-surface/T003-classify-pi-exposure.md
            Tracked in: https://github.com/pablontiv/rootline/issues/..."
```

### Mensaje de Error para Binario Ausente o Viejo

**Template for missing or too-old rootline:**

```
Error: rootline binary not found or incompatible

Details:
- Expected: rootline >= v0.9.0
- Found: [ACTUAL_VERSION or "not installed"]

To install or upgrade, run:
  bash <(curl -fsSL https://raw.githubusercontent.com/pablontiv/rootline/master/install.sh)

Or manually:
  go install github.com/pablontiv/rootline/cmd/rootline@latest

For help, see: https://github.com/pablontiv/rootline#installation
```

When rootline is missing entirely:
```
Error: rootline is required but not installed

Pi requires the rootline CLI (>= v0.9.0) to query and validate Rootline-managed documentation.

To install:
  bash <(curl -fsSL https://raw.githubusercontent.com/pablontiv/rootline/master/install.sh)

After installation, verify with:
  rootline --version
```

### Estrategia de Actualización

**Guidance path for users:**

1. **Detection**: On startup, Pi checks `which rootline` and runs `rootline --version`
2. **Minimum enforcement**: If missing or < v0.9.0, display error message (above template)
3. **Primary update path**: Direct users to `install.sh`:
   ```bash
   bash <(curl -fsSL https://raw.githubusercontent.com/pablontiv/rootline/master/install.sh)
   ```
   This script:
   - Auto-detects platform (linux/darwin) and arch (amd64/arm64)
   - Fetches latest release from GitHub
   - Verifies SHA256 checksum
   - Installs to `$HOME/.local/bin` or `/usr/local/bin` (with sudo if needed)
4. **Alternative path**: For users who prefer source builds:
   ```bash
   go install github.com/pablontiv/rootline/cmd/rootline@latest
   ```
5. **Release monitoring**: Rootline uses automated semantic versioning via Conventional Commits + goreleaser. New releases are published to GitHub Releases with checksums and multi-platform binaries (linux/darwin/windows × amd64/arm64).

**Version lifecycle**:
- Pre-1.0 (current): Breaking changes bump minor version (e.g., v0.9.0 → v0.10.0). Bug fixes and features bump patch. Pi can safely rely on commands from the minimum version forward.
- Post-1.0: Standard semver — breaking changes bump major. Pi may then tighten minimum version only on major updates.
