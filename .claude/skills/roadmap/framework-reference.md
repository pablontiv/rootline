# Marco de Trabajo para Desarrollo con Agentes AI

**Versión:** 1.0
**Audiencia:** Agentes AI y operadores humanos
**Cliente principal:** Platform Owner (Pones)

---

## 1. Propósito del sistema

Este marco define **cómo se descompone, ejecuta y valida el trabajo** cuando el desarrollo es realizado total o parcialmente por **agentes AI con contexto limitado por sesión**.

El objetivo es garantizar que cada unidad de trabajo sea:

* comprensible de forma aislada
* ejecutable en una sesión
* validable sin memoria histórica
* trazable a una intención superior

---

## 2. Modelo mental fundamental

### 2.1 Cliente

El **cliente** de todas las unidades es el *Platform Owner*.
El valor entregado puede ser:

* funcional (capacidad del sistema)
* operacional (reducción de riesgo, reproducibilidad)
* técnico (habilitación de futuros pasos)

No se asume UX ni usuario final.

### 2.2 Principio clave

> **El agente AI no "recuerda", solo "observa estado actual".**
> Por lo tanto, **toda intención debe estar explícita en la unidad que ejecuta**.

---

## 3. Jerarquía de trabajo (canónica)

```text
Épica
 └─ Feature
     └─ Story
         └─ Task   ← unidad ejecutable por un agente AI
```

Cada nivel tiene una **responsabilidad distinta**.
Ningún nivel reemplaza a otro.

---

## 4. Definición formal de cada unidad

### 4.1 Épica

**Rol:** Dirección estratégica
**Pregunta que responde:** *¿Qué objetivo sistémico estamos persiguiendo?*

* Vive semanas o meses
* Tiene métrica de éxito
* Agrupa features coherentes

**NO contiene:** tareas, pasos, comandos

### 4.2 Feature

**Rol:** Milestone técnico o fase coherente
**Pregunta:** *¿Qué bloque completo debe existir para avanzar?*

* Tiene un **Objetivo** claro (capacidad funcional que habilita)
* Tiene un **Beneficio** explícito (qué problema resuelve o qué habilita)
* Tiene un **Milestone** medible (condición observable de "done")
* Puede cerrarse de forma independiente
* Mejora el estado global del sistema
* Agrupa stories relacionadas (target: 1-4 Stories)

**NO es Feature si**: tiene 1 sola Story (absorber en Feature vecino) o si no tiene objetivo propio distinto del Epic.

### 4.3 Story

**Rol:** Unidad de valor para el cliente (humano)
**Pregunta:** *¿Qué capacidad nueva existe cuando esto está hecho?*

Una story **DEBE**:

* entregar una **capacidad observable**
* ser testeable
* tener un "antes / después" claro
* poder fallar de forma significativa

**Importante:**
Una story **NO está pensada para ser ejecutada en una sola sesión de AI**.

La story es:

* contrato semántico
* checkpoint de validación
* agregador de tasks

### 4.4 Task (unidad AI-native)

**Rol:** Unidad mínima ejecutable
**Pregunta:** *¿Qué puede hacer un agente AI completamente en una sesión?*

La **task es el átomo del sistema**.

Tasks may include an **Especificacion Tecnica** section for IaC work (services, modules, operations, infrastructure). This makes each Task self-contained: both specification and execution instructions live in one file. The `Tipo` field indicates the nature of the Task and determines which technical specification template applies.

---

## 5. Definición estricta de una Task

Una **task válida para agentes AI** cumple **todas**:

1. Ejecutable en **una sola sesión**
2. No depende de memoria histórica
3. Tiene **criterios de aceptación binarios**
4. Es verificable solo con estado actual
5. Es idempotente o descartable
6. Tiene input y output explícitos
7. Para tipos IaC: incluye especificación técnica suficiente para implementar sin referencias externas

Si no cumple una → **no es task**

### 5.1 Task Types and Technical Specifications

Tasks are classified by `Tipo` in three categories:

* **IaC types** (servicio-docker, modulo-sistema, operacion-sistema, lxc, vm, modulo-infraestructura, host-script, instance-script): Include `Especificacion Tecnica` with infrastructure YAML.
* **Software types** (software-module, software-test, ci-cd): Include `Especificacion Tecnica` with software development YAML (paquete, interfaces, tests, etc.).
* **General types** (documentation): Omit `Especificacion Tecnica`.

See `task-guide.md` for the complete template and type-specific YAML blocks.

---

## 6. Criterios de aceptación (obligatorios)

### 6.1 Task

Los criterios de una task son **operativos**, no semánticos.

Deben ser:

* observables
* automáticos
* pass / fail
* sin interpretación humana

Ejemplo válido:

* `flux --version` devuelve `2.2.x`
* recurso Kubernetes en estado `Ready`
* archivo existe con checksum esperado

Ejemplo inválido:

* "Configurado correctamente"
* "Listo para usar"
* "Bien integrado"

### 6.2 Story

Los criterios de una story son **de capacidad**.

Ejemplo:

* *El cluster se gobierna por GitOps*
* *El sistema se recupera en < X minutos*
* *Existe rollback verificable*

Una story se considera **DONE** solo si:

* todas sus tasks cumplen
* y sus criterios semánticos se verifican

---

## 7. Relación entre Story y Task

```text
Story: "Cluster bootstrappeado con Flux"

 ├─ Task 1: Instalar Flux CLI
 │   └─ CA: flux --version = X
 │
 ├─ Task 2: Ejecutar flux bootstrap
 │   └─ CA: flux check OK
 │
 └─ Task 3: Validar reconciliación
     └─ CA: drift corregido automáticamente
```

El **agente AI ejecuta tasks**.
El **humano valida stories**.

---

## 8. Qué NO debe hacer un agente AI

Un agente AI **no debe**:

* redefinir alcance
* crear nuevas stories
* tomar decisiones arquitectónicas no explícitas
* continuar "porque parece útil"

El agente **termina cuando los criterios se cumplen**.

---

## 9. Contexto mínimo que TODA unidad debe incluir

Toda Story y Task debe declarar explícitamente:

* Cliente
* Tipo (para Tasks)
* Alcance (in / out)
* Estado inicial esperado
* Resultado esperado
* Fuente de verdad (repo, doc, path)

Si falta uno → **riesgo de deriva**

---

## 10. Regla de corte por contexto (crítica)

> **Si una unidad requiere explicar "por qué" más de un párrafo → es demasiado grande para una sesión de AI.**

Dividir hasta que:

* solo quede "qué hacer" y "cómo validar"

---

## 11. Principios finales (no negociables)

1. **Contexto explícito > memoria implícita**
2. **Criterio de aceptación = contrato**
3. **Tasks pequeñas, stories significativas**
4. **El progreso se mide por capacidades, no por pasos**
5. **Un agente AI nunca adivina intención**

---

## 12. Resumen operativo

* El roadmap se piensa en **épicas**
* El avance se mide en **stories**
* La ejecución ocurre en **tasks**
* El agente vive en la **task**
* El humano vive en la **story**
