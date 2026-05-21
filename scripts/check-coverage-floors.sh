#!/usr/bin/env bash
set -euo pipefail

# Check coverage against per-package floors
# Usage: check-coverage-floors.sh <coverage.out> <.coverage-floors.toml>
#
# Computes statement-level coverage per package directly from the coverage
# profile (mode: set/count/atomic). Skips packages with no source statements
# (e.g. test-only packages like internal/e2e).

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

# Compute statement-level coverage per package from the coverage profile.
# Coverage profile lines: <file>:<start>,<end> <stmts> <count>
# Aggregate covered/total statements per package prefix.
get_package_coverage() {
    local pkg="$1"
    local module_prefix="github.com/pablontiv/rootline/${pkg}/"

    # awk: sum stmts where count>0 (covered) and total stmts per package prefix
    awk -v prefix="$module_prefix" '
    /^mode:/ { next }
    {
        # field 1: file:startline.col,endline.col
        split($1, parts, ":")
        filepath = parts[1]
        if (index(filepath, prefix) != 1) next
        stmts = $2
        count = $3
        total += stmts
        if (count > 0) covered += stmts
    }
    END {
        if (total == 0) {
            print "SKIP"
        } else {
            printf "%.1f\n", 100 * covered / total
        }
    }
    ' "$COVERAGE_OUT"
}

# Check coverage for each package
echo "Coverage Report:"
echo "================"
declare -a FAILED_PACKAGES=()
for pkg in "${PACKAGES[@]}"; do
    cov=$(get_package_coverage "$pkg")

    if [[ "$cov" == "SKIP" ]]; then
        echo "SKIP: $pkg (no source statements — test-only package)"
        continue
    fi

    # Compare using awk
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
