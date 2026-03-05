# Q1: Report Schema Versioning

## Question

How should the inference report format (`"version": 1`) handle forward compatibility as new inference categories are added?

## Source

`[[intake/inference-engine-architecture]]` — Part 8, Q1

## Context

Options: strict versioning (bump version when format changes, consumers must handle each version) or additive-only (new categories don't break old consumers, like JSON with unknown-field tolerance). Rootline already uses `"version": 1` in all JSON output for contract stability.

## Why it matters for roadmap

Strict versioning means `apply` needs version negotiation. Additive-only means consumers ignore unknown categories — simpler but may miss breaking changes in existing categories.

## Topic

api-design, versioning, compatibility
