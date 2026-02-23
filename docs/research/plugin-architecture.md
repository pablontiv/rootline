---
estado: Pre-research
fecha: "2026-02-17"
metodo: web-research
---
# I2 — Plugin Architecture: Pre-Research Notes

**Contexto**: Capturado durante I7. Artículo de referencia + análisis de opciones.

---

## Decisión previa (I7)

D28 establece que solo Markdown es built-in. Todos los demás extractors son plugins.
D29 establece que el contrato `Extractor→Record` es serializable (JSON over process boundary).

Esto significa que I2 no es futureproofing — es el mecanismo que habilita extensibilidad real.

---

## Opciones identificadas

### 1. Go `plugin` package (`.so` nativo)

**Referencia**: "Building Extensible Go Applications with Plugins" (Thisara Weerakoon, Medium, Apr 2025)

**Patrón**:
```go
// Protocol package (shared)
type Extractor interface { ... }

// Plugin (.so)
var Plugin = MyYAMLExtractor{}

// Host
p, _ := plugin.Open("yaml-extractor.so")
sym, _ := p.Lookup("Plugin")
ext := sym.(protocol.Extractor)
```

**Pros**: Nativo, type-safe, zero serialization overhead
**Contras**:
- Solo Linux/macOS/FreeBSD — no Windows
- Plugin y host deben compilar con **exact same Go version**
- No se puede descargar un plugin (no unload)
- Dependencias compartidas deben ser versiones idénticas
- Frágil en distribución — el usuario necesita compilar plugins con el mismo toolchain

### 2. External process (stdin/stdout JSON)

**Patrón**:
```
host → stdin: {"path": "file.yaml", "content": "base64..."}
plugin → stdout: {"path": "file.yaml", "type": "yaml", "frontmatter": {...}, "body": "", "errors": []}
```

**Pros**: Cross-platform, any language, simple, aislamiento natural
**Contras**: IPC overhead, process management, no type safety en compile time

**Nota**: D29 (contrato serializable) fue diseñado explícitamente para este mecanismo.

### 3. WASM (wazero runtime)

**Patrón**: Plugin compila a `.wasm`, host ejecuta con wazero (pure Go, no CGO)

**Pros**: Cross-platform, sandboxed, determinístico
**Contras**: Complexity, limited Go stdlib disponible en WASM, serialización en boundary

### 4. HashiCorp go-plugin (gRPC)

**Patrón**: Plugin como proceso separado, comunicación via gRPC. Usado por Terraform/Vault/Nomad.

**Pros**: Battle-tested (es lo que usa OpenTofu para providers), health checks, versioning, multiplexing
**Contras**: Heavy dependency (gRPC + protobuf), overhead para algo simple como extraer frontmatter

---

## Evaluación preliminar

| Criterio | `.so` nativo | External process | WASM | go-plugin (gRPC) |
|----------|-------------|-----------------|------|-------------------|
| Cross-platform | No (no Windows) | Si | Si | Si |
| Any language | No (solo Go) | Si | Si (compile to WASM) | Si |
| Simplicidad | Media | **Alta** | Baja | Baja |
| Type safety | **Compile-time** | Runtime (JSON schema) | Runtime | Compile-time (proto) |
| Distribución | Frágil | **Simple** (cualquier binario) | Media (.wasm files) | Media (binarios) |
| Overhead | Ninguno | Bajo (1 exec por file) | Bajo | Medio (gRPC startup) |
| Precedente Rootline | — | Alineado con D29 | — | Usado por OpenTofu |

**Hipótesis preliminar**: External process es el candidato más fuerte para Rootline. Simple, cross-platform, alineado con D29, y permite plugins en cualquier lenguaje. El overhead de un exec por archivo es negligible para documentación (<1000 files).

---

## Para investigar en I2

- Benchmarking: overhead real de exec por file vs batch mode (pasar N files en un solo exec)
- Discovery: ¿cómo encuentra Rootline los plugins? PATH, directorio `~/.rootline/plugins/`, config en `.stem`?
- Versioning: ¿cómo se valida compatibilidad entre plugin y host?
- Error handling: ¿qué pasa si el plugin crashea mid-extraction?
- Security: ¿sandboxing necesario para plugins de terceros?
