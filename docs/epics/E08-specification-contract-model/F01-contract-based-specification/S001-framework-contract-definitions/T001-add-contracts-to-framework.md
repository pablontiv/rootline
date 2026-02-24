---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Agregar contratos por nivel a framework-reference.md

**Story**: [S001 Framework Contract Definitions](README.md)
**Contribuye a**: framework-reference.md tiene seccion 2.3 con modelo Pre/Post/Invariantes/Trazabilidad

## Contexto

framework-reference.md define la jerarquia Epic/Feature/Story/Task con responsabilidades y criterios por nivel, pero no tiene contratos formales (Pre/Post/Invariantes). El principio "Especificar antes de descomponer" no existe. La cadena de trazabilidad es implicita.

## Alcance

**In**:
1. Agregar seccion 2.3 "Contratos por nivel" despues de seccion 2.2
2. Extender seccion 4.1 (Epic) con bloque de contratos: Postcondiciones (2-3 constraints observables), Invariantes (reglas que ningun feature puede violar), Out of scope
3. Extender seccion 4.2 (Feature) con: Satisface (→ Epic postcondiciones), Postcondicion (milestone medible), Invariantes (heredados del Epic)
4. Extender seccion 4.3 (Story) con: Cubre (→ Feature milestone), Invariantes propios (propiedades que tasks deben preservar)
5. Extender seccion 4.4 (Task) con: Contribuye a (→ Story criterio), Preserva (→ Story invariantes)
6. Agregar seccion 12.1 con diagrama de cadena de trazabilidad bidireccional

**Out**: No modificar contenido existente. No agregar ejemplos extensos. No crear nuevos archivos.

## Preserva

- INV1: El contenido existente de framework-reference.md no se elimina ni modifica
- Verificar: `diff` entre version anterior y nueva solo muestra adiciones

## Estado inicial esperado

- framework-reference.md existe con secciones 1-12
- Seccion 2.2 tiene principio "el agente no recuerda"
- Secciones 4.1-4.4 definen Epic/Feature/Story/Task sin contratos formales

## Criterios de Aceptacion

- Seccion 2.3 existe con titulo "Contratos por nivel" y contiene modelo Pre/Post/Invariantes/Trazabilidad
- Seccion 4.1 tiene bloque "Restricciones formales" con Postcondiciones + Invariantes + Out of scope
- Seccion 4.2 tiene bloque con Satisface + Postcondicion + Invariantes
- Seccion 4.3 tiene bloque con Cubre + Invariantes
- Seccion 4.4 tiene bloque con Contribuye a + Preserva
- Seccion 12.1 existe con cadena de trazabilidad: Epic.Postcondiciones → Feature.Satisface → Story.Cubre → Task.Contribuye_a

## Fuente de verdad

- `.claude/skills/roadmap/framework-reference.md`
