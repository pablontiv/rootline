# Hallazgos

Descubrimientos e insights del proyecto. Se actualiza continuamente.

---

### 2026-04-02 23:30
**Insight:** El merge de `.stem` es driven por tipo YAML, no por nombre de campo — maps hacen merge a nivel key, arrays/scalars reemplazan
**Contexto:** Reversa arquitectural de rootline — decisión de diseño del motor de reglas
**Categoría:** Arquitectura

### 2026-04-02 23:30
**Insight:** expr-lang se usa unificadamente para derivation, query filtering y aggregation — mismo evaluator en las 3 fases
**Contexto:** Pipeline: Derivation → Aggregation → Query
**Categoría:** Diseño

### 2026-04-02 23:30
**Insight:** CLI y MCP server llaman al motor core directamente — sin serialización entre layers, misma API
**Contexto:** `/dc:reversa` — internal/mcp/ y cmd/rootline/ comparten packages internos
**Categoría:** Arquitectura

### 2026-04-02 23:30
**Insight:** Aggregation es bottom-up deepest-first — index files (README.md) procesadas después de sus children
**Contexto:** derive.AggregateAll() asume children ya tienen Record.Derived populado
**Categoría:** Flujo de datos

### 2026-04-02 23:30
**Insight:** Fuzzy matching con threshold adaptativo `max(2, len/3)` — compartido por validation (enum), graph (broken links), query (unknown fields), fix
**Contexto:** Paquete internal/fuzzy usado en 4 lugares diferentes
**Categoría:** Patrón
