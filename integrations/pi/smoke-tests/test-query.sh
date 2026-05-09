#!/bin/bash
# Smoke test: Verify that read-only Rootline queries work
#
# This test runs a simple read-only rootline query directly via the CLI
# to verify that query functionality works and returns valid results.
#
# Exit codes:
#   0 = Success (query executed and returned results)
#   1 = Rootline not found or query failed

# Skip if rootline is not available
if ! command -v rootline >/dev/null 2>&1; then
    echo "rootline not installed, skipping read-only query test"
    exit 0
fi

# Change to repo root for relative path resolution
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../" && pwd)"
cd "$REPO_ROOT"

# Run a simple read-only query to verify query tool works
# This query counts records in the roadmap that are index files
output=$(rootline query docs/roadmap/ --where "isIndex == true" --count --output json 2>&1 || true)

# Check if the output contains an error
if echo "$output" | grep -qi "error\|failed"; then
    echo "✗ Read-only query test failed: Error in output"
    echo "Output: $output"
    exit 1
fi

# Check if we got JSON output (indicates successful execution)
if echo "$output" | grep -q '{"'; then
    count=$(echo "$output" | grep -o '"count":[0-9]*' | cut -d: -f2)
    if [ -n "$count" ]; then
        echo "✓ Read-only query test passed (found $count index files)"
        exit 0
    fi
fi

# Fallback: check if any valid JSON output was returned
if echo "$output" | grep -q '^{'; then
    echo "✓ Read-only query test passed (got valid response)"
    exit 0
else
    echo "✗ Read-only query test failed: No valid response"
    echo "Output: $output"
    exit 1
fi
