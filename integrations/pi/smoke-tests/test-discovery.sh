#!/bin/bash
# Smoke test: Verify that Pi tools are discoverable
#
# This test verifies that Rootline tools are defined via Pi extensions.
# It checks for tool implementations and package infrastructure.
#
# Exit codes:
#   0 = Success (tools discoverable)
#   1 = Tools not discoverable or infrastructure missing

# Don't use set -e; we want to continue checking even if one test fails
# set -e

# Skip if Pi is not available
if ! command -v pi >/dev/null 2>&1; then
    echo "pi not installed, skipping tool discovery test"
    exit 0
fi

# Change to repo root for relative path resolution
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../" && pwd)"
cd "$REPO_ROOT"

# Track overall test success
test_passed=0

# Test 1: Verify core extensions exist
# These are the tools that the package exposes via the extensions directory
echo "Checking for Rootline tool implementations..."
expected_tools=("doctor" "query" "validate" "tree" "stats" "describe")
found_tools=0

for tool in "${expected_tools[@]}"; do
    if [ -f "integrations/pi/extensions/$tool.ts" ]; then
        echo "✓ Tool implementation found: rootline-$tool"
        ((found_tools++))
    fi
done

if [ $found_tools -lt 5 ]; then
    echo "✗ Not enough tool implementations found (found $found_tools, expected at least 5)"
    exit 1
fi

# Test 2: Verify runner infrastructure is in place
if [ ! -f "integrations/pi/runner.ts" ]; then
    echo "✗ Runner infrastructure missing (runner.ts)"
    exit 1
fi
echo "✓ Runner infrastructure verified"

# Test 3: Verify package.json Pi manifest
if [ ! -f "integrations/pi/package.json" ]; then
    echo "✗ Package manifest missing"
    exit 1
fi

if ! grep -q '"pi"' "integrations/pi/package.json"; then
    echo "✗ Pi manifest not found in package.json"
    exit 1
fi
echo "✓ Pi package manifest verified"

echo "✓ Tool discovery test passed (found $found_tools Rootline tools discoverable)"
exit 0
