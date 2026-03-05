# Q5: Body Content as First-Class Data

## Question

Should body-content analysis (categories 6-13) live in rootline core, or remain in the skill/agent layer?

## Source

`[[intake/inference-engine-architecture]]` — Part 8, Q5

## Context

Rootline currently validates frontmatter only. Categories 6-13 operate on body content (headings, sections, embedded YAML, wiki-links in prose). Extending the engine to validate body structure would require a new validation dimension. However, `internal/extract/` already parses wiki-links from body content, suggesting the boundary is already porous.

## Why it matters for roadmap

If body content enters the engine, categories 6-8 become engine features (significant Go work). If it stays outside, they become skills/agents (different implementation path, different testing strategy).

## Topic

architecture, scope, validation
