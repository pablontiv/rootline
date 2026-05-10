I did **not** write `/home/shared/rootline/research/pi-extension-surface.md` because the task also said “Do not edit files.” Below is the concise research brief content.

# Research: Pi extension surface for Rootline

## Summary
Pi has a broad native extension surface: TypeScript extensions can register LLM tools, slash commands, command argument completions, CLI flags, shortcuts, model providers, status/footer/widgets, custom autocomplete providers, and event hooks. Distribution is via Pi packages installed from npm, git, or local paths; for Rootline, the cleanest path is a dedicated Pi package published to npm, with optional project-local install through `.pi/settings.json`.

## Findings
1. **Extensions are TypeScript modules loaded directly, no build step required** — Pi loads extensions with `jiti`, so `.ts` files work without compilation; extensions export a default factory receiving `ExtensionAPI`. Runtime deps must be in `dependencies`, while Pi core packages should be peer deps. [Pi extensions docs](https://pi.dev/docs/latest/extensions), source: `packages/coding-agent/src/core/extensions/loader.ts:15-57,335-347`.

2. **Extension API supports native tools and commands** — `pi.registerTool()` adds LLM-callable tools; `pi.registerCommand()` adds slash commands; `getArgumentCompletions` supports command argument autocomplete. Extensions can also inspect commands via `pi.getCommands()`. Source: `packages/coding-agent/src/core/extensions/types.ts:1036-1217`; [extensions API docs](https://pi.dev/docs/latest/extensions#extensionapi-methods).

3. **Extensions can customize UI/status/autocomplete** — `ctx.ui.setStatus`, `setWidget`, `setFooter`, `setHeader`, `setEditorComponent`, and `addAutocompleteProvider()` are available. This is enough for Rootline status indicators and field/path/record autocomplete. Source: `packages/coding-agent/src/core/extensions/types.ts:100-260`; [custom UI docs](https://pi.dev/docs/latest/extensions#custom-ui).

4. **Extensions can intercept and augment agent behavior** — hooks include `before_agent_start`, `context`, `tool_call`, `tool_result`, `input`, session lifecycle, compaction, model selection, provider request/response, and user bash. `tool_call` can block/mutate args; `tool_result` can patch results. [Pi extensions docs](https://pi.dev/docs/latest/extensions#events), source: `types.ts:1036-1089`.

5. **Skills and prompt templates are first-class package resources** — Skills follow the Agent Skills standard and register `/skill:name`; prompt templates are Markdown snippets invoked as `/name` and support arguments plus `argument-hint` for autocomplete. [Skills docs](https://pi.dev/docs/latest/skills), [prompt templates docs](https://pi.dev/docs/latest/prompt-templates).

6. **Pi package model supports npm/git/local installs** — `pi install npm:@scope/pkg`, `pi install git:github.com/user/repo@ref`, URLs, and local paths are supported. Global installs write `~/.pi/agent/settings.json`; `-l` writes project `.pi/settings.json`, and project settings can auto-install missing packages on startup. [Packages docs](https://pi.dev/docs/latest/packages).

7. **Package layout is simple** — packages declare resources under `package.json` `pi` key (`extensions`, `skills`, `prompts`, `themes`) or use conventional dirs. Include `keywords: ["pi-package"]` for discovery/gallery. [Packages docs](https://pi.dev/docs/latest/packages#creating-a-pi-package).

8. **Install locations and dependency constraints matter** — global npm uses `npm install -g`; project npm goes under `.pi/npm`; git packages clone under `~/.pi/agent/git/...` or `.pi/git/...`; git packages run `npm install` when `package.json` exists, usually omitting dev deps. Source: `packages/coding-agent/src/core/package-manager.ts:1670-1775,1830-1875`; [packages docs](https://pi.dev/docs/latest/packages#package-sources).

## Decision implications for Rootline
- **Recommended:** create a dedicated Pi package, e.g. `@pablontiv/rootline-pi`, published to npm.
- Package should contain:
  - `extensions/rootline.ts` registering tools like `rootline_query`, `rootline_validate`, `rootline_tree`, maybe `rootline_set`.
  - `skills/rootline/SKILL.md` for Rootline workflows and docs conventions.
  - `prompts/*.md` for common review/validation/migration prompts.
- Extension should probably call the existing `rootline` CLI via `pi.exec()` and require Rootline on `PATH`, rather than bundling the Go binary.
- For teams, document `pi install -l npm:@pablontiv/rootline-pi` so `.pi/settings.json` can share the package requirement.
- For development, use the external Pi package repository rather than this Rootline repo.
- If installing from a git repo, Pi expects package resources at repo root unless that repo has a root `package.json` with a `pi` manifest; a dedicated npm package is cleaner.

## Confidence
High for Pi extension/package capabilities and install model; based on current official Pi docs plus source inspection. Medium for exact best packaging location because it depends on Rootline release/distribution preferences.

## Gaps
- Did not test against a locally installed Pi version.
- Pi docs do not document git subdirectory package installs; assume unsupported unless tested.
- Need product decision: publish separate npm package vs add root-level `package.json` manifest in Rootline repo.