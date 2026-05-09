#!/bin/bash
# Smoke test: Verify that rootline validate --all returns exit 0
#
# This test runs the rootline CLI directly to verify the core validation
# workflow passes, ensuring the roadmap is in a valid state.
#
# Exit codes:
#   0 = Success (validation passed)
#   1 = Rootline not found or validation failed

set -e

# Skip if rootline is not available
if ! command -v rootline >/dev/null 2>&1; then
    echo "rootline not installed, skipping validation test"
    exit 0
fi

# Change to repo root for relative path resolution
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../" && pwd)"
cd "$REPO_ROOT"

# Run validation
if rootline validate --all docs/roadmap/ > /dev/null 2>&1; then
    echo "✓ Validation test passed"
    exit 0
else
    exit_code=$?
    echo "✗ Validation test failed with exit code $exit_code"
    # Still show the validation output for debugging
    rootline validate --all docs/roadmap/ --output json | head -20 || true
    exit 1
fi
