---
estado: Feasible
fecha: "2026-02-23"
metodo: market-analysis
---
# Estrategia de Crecimiento y ROI (Fases de Implementación)

Este documento detalla el orden de implementación recomendado para maximizar el retorno de inversión (ROI) y la adopción de Rootline, aprovechando que el motor principal ya está completo.

---

## Fase 1: IA-First (Quick Win) — Completado
**Objetivo**: Convertir Rootline en la fuente de verdad para asistentes de IA.

*   **Acción**: ~~Completar `rootline serve` (MCP Server).~~ Completado — 8 MCP tools registrados (query, validate, describe, tree, stats, explain, fix, graph).
*   **Por qué**: Permite que Claude Desktop y otros agentes consuman la documentación estructurada sin lógica nueva.
*   **Impacto**: Posiciona a Rootline como infraestructura crítica para RAG estructurado.

## Fase 2: Impacto Visual y Adopción
**Objetivo**: Reducir la fricción de entrada y mejorar la "visibilidad" del conocimiento.

*   **Acción 1: Visualización Interactiva**. Comando `rootline graph --open` para abrir un visor HTML con Mermaid.
*   **Acción 2: Starter Packs**. Incluir plantillas (`rootline init --template zettelkasten`) dentro del binario.
*   **Impacto**: Los usuarios pueden "ver" su base de conocimiento de inmediato, facilitando la venta interna en equipos.

## Fase 3: Gobernanza y Workflow (CI/CD)
**Objetivo**: Integrar Rootline en el ciclo de vida del desarrollo profesional.

*   **Acción 1: GitHub Action Oficial**. Publicar y documentar la acción para validar documentación en PRs.
*   **Acción 2: Dogfooding**. Migrar el job `docs-validate` de este mismo repositorio para que use la Action local (`uses: ./`).
    *   *Beneficio*: Habilita anotaciones en línea en los PRs de este repo y valida la propia infraestructura de la Action.
*   **Impacto**: Establece a Rootline como el "TypeScript para Markdown" (linter de contenido).

## Fase 4: Expansión de Dominio
**Objetivo**: Salir del nicho de Markdown puro.

*   **Acción**: Plugins de extracción para JSON, TOML y comentarios en código fuente.
*   **Impacto**: Rootline puede validar la metadata de paquetes en monorepos o configuraciones complejas de infraestructura.

## Fase 5: Retención y Professional DX
**Objetivo**: Crear un foso defensivo a través de la experiencia de usuario.

*   **Acción**: Implementar un Language Server Protocol (LSP).
*   **Capacidades**: Autocompletado de enums, validación en tiempo real y "Go to definition" hacia archivos `.stem` desde VS Code o Neovim.
*   **Impacto**: Convierte a Rootline en una herramienta nativa del entorno de edición del desarrollador.

---

## Execution Status (assessed 2026-03-20)

| Fase | Estado | Evidencia |
|------|--------|-----------|
| Fase 1: MCP Server | **Completada** | 8 tools en `internal/mcp/`, `rootline serve` funcional |
| Fase 2: Visual + Adopción | **Parcial** | `rootline graph` soporta DOT + Mermaid, pero NO HTML viewer ni `--open`. No hay starter templates (`--template`) |
| Fase 3: CI/CD Governance | **Completada** | `action.yml` publicada (362 líneas), composite action con checksums, PR annotations, step summary. Dogfooding activo en CI |
| Fase 4: Plugin Expansion | **No iniciada** | Extractor registry compile-time existe, pero no hay plugins dinámicos. Ver `plugin-architecture.md` |
| Fase 5: LSP | **No iniciada** | No hay código LSP ni dependencias relacionadas |

**Siguiente paso natural**: Fase 2 — completar Graph HTML Viewer (`--open` flag con Mermaid embed) y Starter Templates (`rootline init --template`). Ambos son de bajo esfuerzo y alto impacto visual.

---

## Resumen de Prioridades

| Prioridad | Tarea | Esfuerzo | Impacto | Estado |
| :--- | :--- | :--- | :--- | :--- |
| ~~**1**~~ | ~~MCP Server (`serve`)~~ | ~~Bajo~~ | ~~Máximo (IA)~~ | Completado |
| ~~**2**~~ | ~~GitHub Action Dogfooding~~ | ~~Muy Bajo~~ | ~~Alto (Calidad)~~ | Completado |
| **3** | Graph HTML Viewer | Bajo | Alto (Visual) | Pendiente |
| **4** | Starter Templates | Bajo | Medio (Adopción) | Pendiente |
| **5** | LSP | Alto | Crítico (Retención) | Pendiente |
