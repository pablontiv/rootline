# Templates de Especificacion Tecnica por Tipo

Templates de referencia del proyecto actual — adaptar segun .stem del proyecto.

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
