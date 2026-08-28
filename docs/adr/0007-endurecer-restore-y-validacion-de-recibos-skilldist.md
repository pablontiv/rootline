---
tipo: adr
estado: accepted
fecha: "2026-08-28"
contexto: "La revisión final de la distribución de skills identificó que restore publicaba candidatos directamente en el destino final y que los recibos JSONL se aceptaban con validación semántica insuficiente."
decision: "Restaurar preimágenes mediante candidatos únicos de staging antes de la publicación atómica y validar semánticamente los recibos en el límite de carga del Store antes de usarlos para autorización."
consecuencias: "Restore evita dejar árboles parciales operacionales en el destino final, preserva el estado corriente hasta que el candidato verificado está listo, y rechaza recibos con versión, kind, operación, destinos, acciones o evidencias incompatibles antes de planificar uninstall o restore."
---

## Contexto

La distribución de skills usa recibos append-only para decidir uninstall y restore. La revisión final mostró dos riesgos relacionados: un restore de directorio podía copiar directamente al destino final y dejar un candidato parcial si fallaba la copia o verificación, y los recibos leídos desde JSONL solo se revisaban por sintaxis, identificador y duplicados.

Ambos comportamientos debilitaban la garantía de que las operaciones se basan en evidencia completa y que los fallos preservan el estado observado.

## Decisión

Restore construye primero un candidato en un sibling único de staging, verifica ese candidato y solo entonces lo publica con rename atómico hacia el destino final. Cuando existe un symlink gestionado actual, se conserva en staging hasta que el candidato restaurado ya fue verificado y publicado. En caso de fallo, solo se elimina el candidato operacional y se restaura el estado corriente si el destino final sigue disponible.

El Store valida semánticamente cada recibo al cargar o escanear `receipts.jsonl`: versión exacta 1, kind `rootline/skill-receipt`, operación conocida, destinos soportados sin duplicados, acciones conocidas, evidencias `before`/`after` compatibles y backups requeridos por la operación y acción.

## Alternativas descartadas

- Mantener la copia directa al destino final y mejorar solo el rollback: descartado porque una falla de verificación puede dejar un árbol parcial que impide distinguir estado externo de estado operacional.
- Validar recibos únicamente en los comandos CLI: descartado porque `Store.Load`, `Store.Latest` y los planificadores son el límite compartido de autorización.
- Verificar existencia física de backups durante el escaneo JSONL: descartado porque la disponibilidad física se valida durante la planificación de restore; el escaneo debe rechazar contratos imposibles sin acoplarse prematuramente al estado mutable del filesystem.

## Consecuencias

- Un recibo malformado semánticamente bloquea el escaneo completo para evitar decisiones basadas en evidencia no confiable.
- Los recibos no-op de instalación siguen siendo válidos sin backups copiados cuando la evidencia del symlink correcto es suficiente.
- La implementación agrega staging e inyección mínima de operaciones de filesystem para cubrir fallos enfocados sin introducir un framework genérico.
