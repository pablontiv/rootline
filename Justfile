# Justfile for Rootline
set shell := ["bash", "-c"]

# Default recipe
default: check test

# Run all checks (fmt, lint, build)
check:
    gofmt -l . | grep . && { echo "ERROR: Code not formatted. Run: just fmt"; exit 1; } || true
    golangci-lint run
    go build ./...

# Format code
fmt:
    gofmt -l -w .

# Run tests
test:
    go test ./... -race

# Show coverage per package and total
coverage:
    go test ./... -coverprofile=coverage.out
    go tool cover -func=coverage.out

# Check coverage meets per-package floors
coverage-check: coverage
    scripts/check-coverage-floors.sh coverage.out .coverage-floors.toml

# Validate docs
validate:
    rootline validate --all docs/roadmap/

# Fix docs (propagate aggregates)
fix-docs:
    rootline fix --all docs/roadmap/

