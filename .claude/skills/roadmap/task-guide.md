# Task Guide — Crear Task AI-Ready

## Workflow

### Paso 1: Parsear Argumentos

De `$ARGUMENTS`, extraer:
- **story-path**: ruta a la Story padre (ej: `E01/F13/S001`)
- **task-name**: slug kebab-case (ej: `add-k8s-phase`)
- **descripción**: qué debe hacer el agente AI

### Paso 2: Determinar Tipo de Task

Evaluar qué tipo de Task se necesita según la descripción:

| Tipo | Cuándo |
|------|--------|
| **IaC Types** | |
| `servicio-docker` | Nuevo servicio Docker o actualización mayor |
| `modulo-sistema` | Configuración OS gestionada por IaC |
| `operacion-sistema` | Remoción, migración, actualización |
| `lxc` | Nueva instancia LXC en Proxmox |
| `vm` | Nueva VM en Proxmox |
| `modulo-infraestructura` | Módulo IaC de capa Proxmox |
| `host-script` | Script para ejecutar en Host Proxmox |
| `instance-script` | Script para ejecutar dentro de LXC/VM |
| **Software Types** | |
| `software-module` | Implementar módulo/paquete de código en un proyecto de software |
| `software-test` | Escribir tests para funcionalidad existente |
| `ci-cd` | Configurar pipeline CI/CD, releases, distribución |
| **General Types** | |
| `documentation` | Documentación, knowledge capture, o refactoring simple |

**Cuándo usar cada categoría:**
- **IaC Types**: Tasks que producen infraestructura materializada (contenedores, módulos, scripts).
- **Software Types**: Tasks que producen código fuente (módulos Go/Python/etc., tests, pipelines).
- **General Types**: Tasks de documentación, knowledge capture, o refactoring simple.

Todos los tipos pueden incluir `## Especificacion Tecnica` cuando se beneficien de una spec estructurada. Usar el bloque YAML del tipo correspondiente, o un bloque libre si no hay template.

### Paso 3: Verificar Story Padre

```bash
# Resolver path real
rootline describe docs/epics/E01-*/F13-*/S001-*/
```

Si no existe → informar al usuario. Sugerir crear con `/roadmap story` primero.

Si existe → leer el README de la Story para:
- Entender la capacidad objetivo
- Extraer contexto relevante para el Task
- Ver Tasks existentes (evitar duplicación)

### Paso 4: Auto-numbering

```bash
# Detectar próximo TXXX en la Story (requiere .stem con id: {type: sequence, prefix: T, digits: 3})
rootline describe <story-dir> --field schema.id.next
```

El comando retorna directamente el próximo identificador (ej: `"T004"`). Requiere que el directorio padre tenga un `.stem` con `id: {type: sequence}` configurado.

### Paso 5: Generar Task File

**5.1**: Crear el archivo con frontmatter correcto usando `rootline new`:

```bash
rootline new <story-dir>/TXXX-task-name.md
```

Esto genera el frontmatter según el `.stem` del directorio, con valores de enum correctos y comentados. El agente edita el contenido del task (contexto, alcance, ACs) pero NO modifica el schema del frontmatter — solo selecciona el valor correcto de cada enum.

**5.2**: Editar el contenido con toda la información necesaria para que un agente AI lo ejecute sin contexto adicional.

**CRÍTICO**: El Task debe ser auto-contenido. Un agente que lea SOLO este archivo debe poder ejecutar el trabajo completo.

### Paso 6: Actualizar Story README

Agregar fila en la tabla "Tasks" del Story README padre:

```markdown
| [TXXX](TXXX-task-name.md) | Descripción breve |
```

**Nota**: NO incluir columna Estado en la tabla. El estado se lee del YAML frontmatter del Task file. `/roadmap view` lo deriva automáticamente.

---

## Template: Task File

**Notas sobre el template**:
- **Wiki-links de dependencia**: Si el Task depende de otro, agregar `[[blocks:TXXX-name]]` debajo del link a la Story. `rootline graph` lee estos links automaticamente para detectar ciclos y resolver orden de ejecucion. Omitir si no hay dependencias.
- **Especificacion Tecnica**: Incluir cuando el Task se beneficie de una especificacion estructurada — aplica a cualquier tipo (IaC, software, ci-cd, etc.). Omitir solo si el Task es puramente textual (ej: documentation sin componente tecnico). Usar el bloque YAML correspondiente al tipo, o un bloque libre si no hay template predefinido.

```markdown
---
estado: Pending
tipo: servicio-docker | modulo-sistema | operacion-sistema | lxc | vm | modulo-infraestructura | host-script | instance-script | software-module | software-test | ci-cd | documentation
ejecutable_en: 1 sesion
---
# TXXX: [Descripción accionable del task]

**Story**: [SXXX Story Name](README.md)

[[blocks:TXXX-prerequisite-task]]

## Contexto

[Párrafo breve explicando el contexto necesario. Extraído de la Story padre pero auto-contenido. El agente no necesita leer otro archivo para entender qué hacer.]

## Especificacion Tecnica

Usar el bloque YAML correspondiente al tipo:

### servicio-docker

```yaml
tipo: servicio-docker
nombre: $TASK_NAME
imagen: # imagen:tag
puertos:
  - interno: 80
    externo: 8080
volumenes:
  - host: /opt/homeserver/data/$TASK_NAME
    container: /data
variables_entorno:
  TZ: America/New_York
red: homelab-network
recursos:
  memoria: # MB o null
  cpu: # cores o null
```

### modulo-sistema

```yaml
tipo: modulo-sistema
nombre: $TASK_NAME
archivos_gestionados:
  - path: /etc/example.conf
    contenido: # template o literal
    permisos: "0644"
comandos_sistema:
  - "systemctl mask example"
directorios:
  - path: /opt/homeserver/data
    owner: root
    mode: "0755"
providers_requeridos:
  - hashicorp/local
  - hashicorp/null
```

### operacion-sistema

```yaml
tipo: operacion-sistema
nombre: $TASK_NAME
accion: remocion | migracion | actualizacion
componentes_afectados:
  - nombre: nombre-componente
    tipo: paquete-apt | systemd-service | archivo | directorio
verificaciones_pre:
  - comando: "comando a ejecutar"
    resultado_esperado: "descripción del resultado OK"
pasos:
  - orden: 1
    descripcion: "Descripción del paso"
    comando: "comando a ejecutar"
    reversible: true | false
verificaciones_post:
  - comando: "comando a ejecutar"
    resultado_esperado: "descripción del resultado OK"
rollback:
  comandos:
    - "comando para revertir"
  tiempo_estimado: "X minutos"
```

### software-module

```yaml
tipo: software-module
proyecto: # nombre del proyecto (ej: rootline)
lenguaje: # go | python | typescript | rust
paquete: # path del paquete (ej: internal/extract)
interfaces:
  - nombre: # nombre de interfaz/tipo a implementar
    metodos:
      - nombre: # nombre del método
        input: # parámetros
        output: # retorno
dependencias_externas:
  - # librería externa (ej: gopkg.in/yaml.v3)
tests:
  - # descripción del test case principal
```

### software-test

```yaml
tipo: software-test
proyecto: # nombre del proyecto
paquete: # path del paquete a testear
cobertura_objetivo: # porcentaje target (ej: 80%)
tipos_test:
  - unit | integration | e2e
fixtures:
  - # archivos de test data necesarios
```

### ci-cd

```yaml
tipo: ci-cd
plataforma: # github-actions | forgejo | gitlab
triggers:
  - push | pr | tag | schedule
jobs:
  - nombre: # nombre del job
    pasos:
      - # descripción del paso
artefactos:
  - # binarios, imágenes, paquetes producidos
```

## Dependencias

> Contexto humano complementario. Las dependencias machine-readable se declaran arriba con `[[blocks:TXXX-name]]`.

- [Task/componente que debe existir antes de ejecutar este Task — contexto adicional]
- [Servicio, módulo o config que este Task requiere]

## Alcance

**In**: [Lista específica de lo que el agente DEBE hacer]
1. [Acción concreta 1]
2. [Acción concreta 2]

**Out**: [Lo que NO debe hacer — límites explícitos]

## Estado inicial esperado

- [Prerrequisito 1 que debe existir antes de ejecutar]
- [Prerrequisito 2]

## Criterios de Aceptacion

- [Criterio binario 1 — comando o check específico con resultado esperado]
- [Criterio binario 2 — observable, automático, pass/fail]
- [Criterio binario 3 — sin términos vagos]

## Fuente de verdad

- [Path a código/config que el agente necesita leer/modificar]
- [Path a documentación de referencia]
```

---

## Estados del Task

| Estado | Emoji | Cuándo |
|--------|-------|--------|
| Pending | - | Task creado, sin especificación técnica |
| Especificado | 📋 | Especificación técnica completa, listo para implementar |
| Completado | ✅ | Ejecutado y verificado exitosamente |
| Obsoleto | ❌ | Reemplazado o ya no relevante |

---

## Checklist de Validación (6 Condiciones)

Antes de finalizar el Task, verificar mentalmente:

| # | Condición | Pregunta de validación |
|---|-----------|----------------------|
| 1 | Sesión única | ¿Un agente puede completar esto en una sesión? |
| 2 | Sin memoria | ¿El archivo contiene TODO el contexto necesario? |
| 3 | Criterios binarios | ¿Cada criterio es pass/fail sin interpretación? |
| 4 | Verificable | ¿Los criterios referencian comandos/checks reales? |
| 5 | Idempotente | ¿Se puede re-ejecutar sin daño? |
| 6 | I/O explícitos | ¿Estado inicial, resultado y fuentes están declarados? |

**Nota**: Agent hooks PreToolUse y PostToolUse validarán automáticamente estas condiciones al escribir el archivo. Si un hook bloquea, corregir según sus indicaciones.

---

## Anti-patrones a Evitar

- **Criterio vago**: "Servicio configurado correctamente" → Usar: "`systemctl is-active service` retorna active"
- **Scope inflado**: Task que toca 5 archivos en 3 capas → Dividir en Tasks más pequeños
- **Dependencia implícita**: "Después de configurar X..." → Declarar en "Estado inicial esperado"
- **Sin fuente de verdad**: Agente no sabe qué archivos mirar → Siempre listar paths
