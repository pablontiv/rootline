---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar subcomando completion con soporte bash/zsh/fish

**Story**: [S001 CLI Polish](README.md)

## Contexto

Cobra tiene soporte built-in para generar scripts de shell completion. El subcomando `completion` debe generar el script apropiado para bash, zsh o fish y escribirlo a stdout para que el usuario lo redirija al archivo de completions de su shell.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: completionCmd
    metodos:
      - nombre: RunE
        input: "cmd *cobra.Command, args []string"
        output: "error"
dependencias_externas: []
tests:
  - completion bash genera output no vacio
  - completion zsh genera output no vacio
  - completion fish genera output no vacio
  - completion sin argumento muestra usage
```

## Dependencias

- Cobra v1.10.2 (ya en go.mod)

## Alcance

**In**:
1. Subcomando `rootline completion <shell>` donde shell es bash, zsh, o fish
2. Output a stdout del script de completion
3. Help text con instrucciones de instalacion por shell

**Out**: Completions dinamicas por valor de campo, powershell

## Estado inicial esperado

- Cobra CLI funcional con subcomandos existentes
- `cmd/rootline/root.go` con rootCmd configurado

## Criterios de Aceptacion

- `rootline completion bash` produce output que empieza con `# bash completion`
- `rootline completion zsh` produce output que contiene `#compdef rootline`
- `rootline completion fish` produce output que contiene `complete -c rootline`
- `rootline completion` sin argumento retorna error con usage
- `go build ./cmd/rootline/` compila sin errores

## Fuente de verdad

- `cmd/rootline/root.go` — rootCmd al que agregar el subcomando
- Cobra GenBashCompletionV2, GenZshCompletion, GenFishCompletion APIs
