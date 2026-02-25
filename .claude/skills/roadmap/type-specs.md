# Templates de Especificación Técnica por Tipo

## Descubrimiento de tipos

Los tipos válidos se descubren dinámicamente desde el schema del proyecto:

```bash
rootline describe <story-dir> --field schema.tipo
```

## Templates por proyecto

Buscar templates específicos del proyecto en `.claude/roadmap.local.md`. Si existe, usar esos templates. Si no existe, usar los templates genéricos de abajo como fallback.

## Templates genéricos (fallback)

Usar cuando no hay `.claude/roadmap.local.md` o cuando el tipo no tiene template en el archivo local.

### Software (módulos, librerías, servicios)

```yaml
tipo: # software-module, servicio-docker, etc.
proyecto: # nombre del proyecto
lenguaje: # go | python | typescript | rust
paquete: # path del paquete/módulo
interfaces:
  - nombre: # interfaz/tipo a implementar
    metodos:
      - nombre: # nombre del método
        input: # parámetros
        output: # retorno
dependencias_externas:
  - # librería externa
tests:
  - # descripción del test case principal
```

### Tests

```yaml
tipo: # software-test
proyecto: # nombre del proyecto
paquete: # path del paquete a testear
cobertura_objetivo: # porcentaje target
tipos_test:
  - unit | integration | e2e
fixtures:
  - # archivos de test data necesarios
```

### CI/CD

```yaml
tipo: # ci-cd
plataforma: # github-actions | gitlab | forgejo | etc.
triggers:
  - push | pr | tag | schedule
jobs:
  - nombre: # nombre del job
    pasos:
      - # descripción del paso
artefactos:
  - # binarios, imágenes, paquetes producidos
```

### Infraestructura (IaC, sistema, contenedores)

```yaml
tipo: # modulo-sistema, modulo-infraestructura, lxc, vm, host-script, instance-script, etc.
nombre: # nombre del componente
archivos_gestionados:
  - path: # path del archivo
    contenido: # template o literal
    permisos: # "0644"
comandos:
  - # comandos a ejecutar
verificaciones:
  - comando: # comando de verificación
    resultado_esperado: # descripción del resultado OK
```

### Documentación

No requiere bloque YAML de especificación técnica. Describir contenido y estructura esperada en prosa dentro de la sección Alcance del task.

### Agrupación (feature, historia)

Estos tipos se usan en READMEs de agrupación. No llevan Especificación Técnica — su contenido se define por los templates de [epic-guide.md](epic-guide.md) y [story-guide.md](story-guide.md).
