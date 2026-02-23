---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar comando graph con output DOT y mermaid

**Story**: [S003 Graph Command](README.md)

## Contexto

Con el Graph builder funcional (T001), el comando CLI `rootline graph` escanea documentos, construye el grafo, y genera output visual. Soporta DOT (para Graphviz) y mermaid (para Markdown/GitHub). El flag `--check` solo valida (ciclos + broken links) y retorna exit code sin generar diagrama.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: graphCmd
    metodos:
      - nombre: RunE
        input: "cmd *cobra.Command, args []string"
        output: "error"
dependencias_externas: []
tests:
  - graph genera output DOT con nodos y edges
  - graph --format mermaid genera output mermaid
  - graph --check con ciclos retorna exit code 1
  - graph --check sin problemas retorna exit code 0
  - graph -o json produce GraphResult versionado
```

## Dependencias

- T001 (Graph builder)
- internal/index (Scan)
- internal/extract (Registry)

## Alcance

**In**:
1. Comando `rootline graph [path]` (default: ".")
2. Flag `--format dot|mermaid` (default: dot)
3. Flag `--check` — solo validar, no generar diagrama
4. Scan records → Build graph → Output
5. DOT output: `digraph { "file1.md" -> "file2.md" [label="blocks"]; }`
6. Mermaid output: `graph TD; file1[file1.md] --> |blocks| file2[file2.md];`
7. Nodos incluyen estado del documento (color/shape por estado)
8. --check: reportar ciclos y broken links, exit code 1 si hay problemas
9. JSON output con `version: 1, kind: "rootline/graph"`

**Out**: Interactive visualization, SVG rendering, web dashboard

## Estado inicial esperado

- internal/graph/ con Build, DetectCycles, BrokenLinks (T001)
- index.Scan y extract.Registry funcionales

## Criterios de Aceptacion

- `rootline graph docs/epics/` genera output DOT valido con nodos
- `rootline graph docs/epics/ --format mermaid` genera output mermaid
- `rootline graph --check docs/epics/` retorna exit code 0 si no hay ciclos ni broken links
- `rootline graph -o json` produce JSON con version:1 y kind:"rootline/graph"
- `go build ./cmd/rootline/` compila sin errores

## Fuente de verdad

- `internal/graph/graph.go` — Graph, Build, DetectCycles, BrokenLinks
- `internal/index/index.go` — Scan
- `internal/extract/registry.go` — Registry
- DOT language reference — output format
- Mermaid syntax — output format
