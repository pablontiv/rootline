---
tipo: adr
estado: accepted
fecha: "2026-08-28"
contexto: "La segunda tarea de distribución de skills debe compilar de forma independiente, pero el brief original anticipaba interfaces basadas en Receipt antes de que la tarea 3 definiera ese modelo."
decision: "Limitar la superficie pública de la tarea 2 a BuildInstallPlan(source Source, states []DestinationState) y aplazar la planificación dependiente de recibos hasta que exista el modelo Receipt."
consecuencias: "La instalación puede construir planes inmutables y verificables sin acoplarse a tipos futuros; las operaciones de uninstall y restore requerirán una extensión posterior cuando Task 3 y Task 5 incorporen recibos."
---

## Contexto

La distribución de skills se está incorporando por etapas. La tarea 2 agrega inventario de destinos soportados y planes de instalación, mientras que la tarea 3 definirá los recibos necesarios para validar uninstall y restore. El brief de la tarea 2 mencionaba una firma con `Receipt`, pero la resolución previa indicó que esta tarea debe compilar sin definir ni referenciar ese tipo.

## Decisión

La implementación de la tarea 2 expone únicamente `BuildInstallPlan(source Source, states []DestinationState) (Plan, error)` para construir planes de instalación. El modelo de plan no incluye campos dependientes de recibos en esta etapa, y el hash canónico del plan cubre la operación, la fuente y las acciones observadas.

## Alternativas descartadas

- Definir un tipo temporal `Receipt`: descartado porque introduciría un contrato prematuro antes de la tarea responsable de diseñarlo.
- Mantener un parámetro o campo de recibo sin tipo completo: descartado porque conservaría acoplamiento conceptual con una interfaz todavía no definida.
- Implementar uninstall/restore con comportamiento provisional: descartado porque esas operaciones dependen de evidencia de recibos que aún no existe.

## Consecuencias

- La tarea 2 permanece autocontenida y compilable.
- Los planes de instalación son deterministas y están ligados a la evidencia observada de fuente y destinos.
- Task 3 o Task 5 deberán agregar explícitamente los campos o planificadores relacionados con recibos si siguen siendo necesarios.
