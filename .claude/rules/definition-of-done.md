# Definition of Done — rootline

> Una tarea NO está terminada hasta que cumple TODOS estos criterios.

- [ ] `just check` pasa (vet + lint)
- [ ] `just test` pasa (0 failures)
- [ ] Commit convencional + push
- [ ] CLI instalado y verificado en el sistema:
  ```bash
  just install          # build local de desarrollo → ~/.local/bin, con ldflags
  just doctor-install   # falla si hay más de un rootline en PATH
  rootline --version    # muestra versión nueva, nunca "dev"
  ```
  `doctor-install` usa `which -a`, no `which`. `which` a secas devuelve solo
  la primera coincidencia del PATH, así que un binario viejo sombreado en
  otro directorio pasaba el check sin ser visto.

  Las tres vías de instalación convergen en `~/.local/bin`: `install.sh` (el
  instalador público) lo usa por default, y los hooks `post-merge`/`pre-push`
  delegan en `just install`. Un único destino, un único esquema de versión.
