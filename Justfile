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

# Validate docs
validate:
    rootline validate --all docs/epics/

# Fix docs (propagate aggregates)
fix-docs:
    rootline fix --all docs/epics/

