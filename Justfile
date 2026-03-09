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

# --- Versioning & Release ---

# Sync root.go version with latest git tag
sync-version:
    #!/usr/bin/env bash
    LATEST=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
    if [ -n "$LATEST" ]; then
      sed -i "s/var version = \".*\"/var version = \"$LATEST\"/" cmd/rootline/root.go
      echo "Synced root.go to $LATEST"
    fi

# Increment version (patch) — derives from latest tag
bump-patch:
    #!/usr/bin/env bash
    LATEST=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
    IFS='.' read -r MAJOR MINOR PATCH <<< "$LATEST"
    NEXT="${MAJOR}.${MINOR}.$((PATCH + 1))"
    sed -i "s/var version = \".*\"/var version = \"$NEXT\"/" cmd/rootline/root.go
    echo "$NEXT"

# Increment version (minor) — derives from latest tag
bump-minor:
    #!/usr/bin/env bash
    LATEST=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
    IFS='.' read -r MAJOR MINOR PATCH <<< "$LATEST"
    NEXT="${MAJOR}.$((MINOR + 1)).0"
    sed -i "s/var version = \".*\"/var version = \"$NEXT\"/" cmd/rootline/root.go
    echo "$NEXT"

# Create a new release (patch)
release-patch: check test
    just bump-patch
    VERSION=$(grep 'var version' cmd/rootline/root.go | sed 's/.*"\(.*\)"/\1/'); \
    git add cmd/rootline/root.go; \
    git commit -m "chore: release v$${VERSION}"; \
    git tag -a "v$${VERSION}" -m "Release v$${VERSION}"; \
    git push origin master --tags

# Create a new release (minor)
release-minor: check test
    just bump-minor
    VERSION=$(grep 'var version' cmd/rootline/root.go | sed 's/.*"\(.*\)"/\1/'); \
    git add cmd/rootline/root.go; \
    git commit -m "chore: release v$${VERSION}"; \
    git tag -a "v$${VERSION}" -m "Release v$${VERSION}"; \
    git push origin master --tags
