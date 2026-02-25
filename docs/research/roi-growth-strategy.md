---
estado: Pre-research
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

## Resumen de Prioridades

| Prioridad | Tarea | Esfuerzo | Impacto |
| :--- | :--- | :--- | :--- |
| ~~**1**~~ | ~~MCP Server (`serve`)~~ | ~~Bajo~~ | ~~Máximo (IA)~~ Completado |
| **2** | GitHub Action Dogfooding | Muy Bajo | Alto (Calidad) |
| **3** | Graph HTML Viewer | Bajo | Alto (Visual) |
| **4** | Starter Templates | Bajo | Medio (Adopción) |
| **5** | LSP | Alto | Crítico (Retención) |
