---
estado: Completed
tipo: task
---
# T002: Design a shared CLI runner for executing rootline from Pi.

**Outcome**: [O02 Design Pi extension architecture](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T001-define-read-only-tool-schemas.md]]

## Preserva

- INV1: The extension treats Rootline CLI JSON as the integration boundary; it does not import Go internal packages.
  - Verificar: Check architecture docs and implementation tasks.

## Contexto

Esta task forma parte de O02 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Design a shared CLI runner for executing rootline from Pi.

## Alcance

**In**:
1. Runner design covers cwd, binary path, timeouts, abort signals, JSON parsing, and stderr.
2. Runner design defines stable error objects for the model.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T001-define-read-only-tool-schemas.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Runner design covers cwd, binary path, timeouts, abort signals, JSON parsing, and stderr.
- Runner design defines stable error objects for the model.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `Pi extension docs`
- `Rootline CLI behavior`

## Runner Design

The CLI runner is the execution boundary between Pi and Rootline. All integration flows through this abstraction, ensuring consistent behavior across mutating and read-only tools.

### 1. Binary Discovery

The runner uses a lookup chain to locate the rootline binary:

1. **Configurable path**: Accept an optional `ROOTLINE_BIN` environment variable or configuration parameter.
2. **PATH lookup**: Search the system `$PATH` for `rootline` executable.
3. **Version check**: Once located, verify `rootline --version` succeeds and returns a semantic version.

**Error handling**: If the binary is not found or version check fails, return a structured error with code `BINARY_NOT_FOUND`.

```
Discovery failure → return {code: "BINARY_NOT_FOUND", message: "...", stderr: "..."}
```

### 2. Working Directory

All rootline invocations must execute with `cwd` set to the project root (the directory containing `.git`). This ensures:
- Relative paths in rootline arguments are resolved consistently.
- `.stem` file discovery walks from the correct anchor.
- File paths in JSON output are relative to the project root.

**Discovery of project root**: Walk up from the current directory until `.git` is found. If not found, return an error with code `PROJECT_ROOT_NOT_FOUND`.

### 3. Timeout

Define two timeout strategies:

1. **Default timeout**: 30 seconds per command.
2. **Per-tool override**: Allow Pi to configure custom timeouts for specific tools (e.g., `validate --all` may require 60s on large projects).

**Timeout behavior**: If a subprocess exceeds its timeout, terminate it (SIGTERM, then SIGKILL after grace period) and return a structured error with code `TIMEOUT`.

### 4. Abort/Cancellation

Propagate user abort signals to the rootline subprocess:

- On user abort (Ctrl+C, cancellation token expiration), send SIGTERM to the rootline process.
- Allow a brief grace period (e.g., 2 seconds) for graceful shutdown.
- If the process does not exit, send SIGKILL.

**Error handling**: Return a structured error with code `ABORT` if the subprocess is cancelled by the user.

### 5. JSON Parsing

Parse stdout as JSON using the following contract:

1. **Version field**: Every rootline JSON response includes a `"version": 1` field.
2. **Validation**: After parsing, validate that `version` exists and is `1`.
3. **Failure mode**: If parsing fails or version is missing/incorrect, return a structured error with code `PARSE_ERROR` and include the raw stdout in the error details.

**Example**:
```json
{
  "version": 1,
  "data": { /* tool-specific result */ }
}
```

### 6. Stderr Handling

Capture stderr separately from stdout:

- **Stdout**: Always contains tool result (JSON for success, or nothing if command fails before JSON output).
- **Stderr**: Contains diagnostic messages, warnings, and error context.
- **No mixing**: Never parse stderr as JSON or append it to stdout.

**Usage**: Include stderr in error responses (see stable error object below) to help the user diagnose subprocess failures.

### 7. Stable Error Object

Define a `RootlineError` shape for consistent error handling across all tools:

```typescript
interface RootlineError {
  code: string;           // Named error code (see below)
  message: string;        // Human-readable error message
  stderr: string;         // Captured stderr from the subprocess
  exitCode: number;       // Exit code from the rootline process (-1 if timeout/abort)
}
```

**Named error codes**:

| Code | Meaning | When to use |
|------|---------|------------|
| `BINARY_NOT_FOUND` | rootline binary not found or version check failed | Binary discovery failed |
| `PROJECT_ROOT_NOT_FOUND` | No `.git` directory found walking up from cwd | Working directory setup failed |
| `TIMEOUT` | Command exceeded configured timeout | Subprocess ran too long |
| `ABORT` | User cancelled the operation | Cancellation signal received |
| `PARSE_ERROR` | stdout is not valid JSON or missing `version` field | JSON parsing failed |
| `ROOTLINE_ERROR` | rootline exited with non-zero code after producing JSON | Command execution failed |
| `INVALID_ARGUMENTS` | Runner validation detected invalid arguments before subprocess invocation | Pre-flight validation failed |

**Example error response**:
```json
{
  "code": "TIMEOUT",
  "message": "rootline validate --all exceeded 30s timeout",
  "stderr": "...",
  "exitCode": -1
}
```

### 8. Security

Prevent arbitrary shell expansion:

- **No shell mode**: Do not invoke rootline via `/bin/sh -c` or similar.
- **Argument arrays**: Pass rootline arguments as an array, not a concatenated shell string.
- **Validation**: Validate all arguments before subprocess invocation; reject arguments containing shell metacharacters if they cannot be safely quoted.

**Example**:
```typescript
// Good: arguments as array
const args = ["query", "--where", "estado=completed"];
spawn("rootline", args, { cwd: projectRoot });

// Bad: shell string
const args = ["sh", "-c", `rootline query --where "estado=completed"`];
```

This ensures argument injection attacks are prevented and the integration remains predictable across platforms.
