# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Rootline, open a public issue using the bug report template.

Include:

- Description of the vulnerability
- Steps to reproduce
- Potential impact

You will receive acknowledgment within 48 hours and a detailed response within 7 days.

## Scope

Rootline is a CLI tool that reads local files. Security concerns may include:

- Path traversal via `.stem` file references
- Unsafe YAML parsing
- Command injection via hook scripts
