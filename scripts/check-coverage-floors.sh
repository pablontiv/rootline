#!/usr/bin/env bash
set -euo pipefail

# Check coverage against per-package floors
# Usage: check-coverage-floors.sh <coverage.out> <.coverage-floors.toml>

COVERAGE_OUT="${1:-coverage.out}"
FLOORS_TOML="${2:-.coverage-floors.toml}"

if [[ ! -f "$COVERAGE_OUT" ]]; then
    echo "ERROR: Coverage file not found: $COVERAGE_OUT" >&2
    exit 1
fi

if [[ ! -f "$FLOORS_TOML" ]]; then
    echo "ERROR: Floors config not found: $FLOORS_TOML" >&2
    exit 1
fi

# Parse default threshold from TOML
DEFAULT_FLOOR=$(grep '^default' "$FLOORS_TOML" | awk -F'=' '{print $2}' | tr -d ' ')

# Parse package list from TOML
# Extract lines between 'packages = [' and ']', strip quotes and whitespace
PACKAGES=()
in_packages=false
while IFS= read -r line; do
    if [[ "$line" =~ ^packages[[:space:]]*=[[:space:]]*\[ ]]; then
        in_packages=true
        continue
    fi
    if [[ "$in_packages" == true ]]; then
        if [[ "$line" =~ \] ]]; then
            break
        fi
        # Strip leading/trailing whitespace, quotes, and commas
        pkg=$(echo "$line" | tr -d '[:space:]",' | grep -v '^$' || true)
        if [[ -n "$pkg" ]]; then
            PACKAGES+=("$pkg")
        fi
    fi
done < "$FLOORS_TOML"

# Function to get coverage for a package
get_package_coverage() {
    local pkg="$1"
    # Use 'go tool cover -func' to get per-function coverage
    # Lines are formatted: github.com/pablontiv/rootline/<pkg>/file.go:func  N  X.X%
    local coverage_lines
    coverage_lines=$(go tool cover -func="$COVERAGE_OUT" | grep "github.com/pablontiv/rootline/${pkg}/" || true)

    if [[ -z "$coverage_lines" ]]; then
        # Package not in coverage (e.g., test build failed)
        echo "0.0"
        return
    fi

    # Extract percentages, remove '%' suffix, compute average
    local total=0.0
    local count=0
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            # Extract percentage value (last field minus %)
            local pct
            pct=$(echo "$line" | awk '{print $NF}' | sed 's/%$//')
            total=$(awk "BEGIN {print $total + $pct}")
            ((count++))
        fi
    done <<< "$coverage_lines"

    if [[ $count -gt 0 ]]; then
        awk "BEGIN {printf \"%.1f\", $total / $count}"
    else
        echo "0.0"
    fi
}

# Check coverage for each package
echo "Coverage Report:"
echo "================"
declare -a FAILED_PACKAGES
for pkg in "${PACKAGES[@]}"; do
    cov=$(get_package_coverage "$pkg")
    # Compare: bash arithmetic requires integers, so we'll use awk
    below_threshold=$(awk "BEGIN {print ($cov < $DEFAULT_FLOOR)}")

    if [[ "$below_threshold" == "1" ]]; then
        echo "FAIL: $pkg = ${cov}% (threshold: ${DEFAULT_FLOOR}%)"
        FAILED_PACKAGES+=("$pkg")
    else
        echo "PASS: $pkg = ${cov}%"
    fi
done

# Show total coverage
echo ""
TOTAL_COV=$(go tool cover -func="$COVERAGE_OUT" | grep total | awk '{print substr($3,1,length($3)-1)}')
echo "TOTAL: ${TOTAL_COV}%"

# Exit with failure if any package is below threshold
if [[ ${#FAILED_PACKAGES[@]} -gt 0 ]]; then
    echo ""
    echo "ERROR: Coverage floors not met for:" >&2
    for pkg in "${FAILED_PACKAGES[@]}"; do
        echo "  - $pkg" >&2
    done
    exit 1
fi

exit 0
