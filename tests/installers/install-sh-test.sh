#!/bin/sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
TEST_TMP="$(mktemp -d)"
trap 'rm -rf "$TEST_TMP"' EXIT

mkdir -p "$TEST_TMP/bin" "$TEST_TMP/home"
cat > "$TEST_TMP/bin/curl" <<'EOF'
#!/bin/sh
case "$*" in
    *api.github.com*)
        : > "$TEST_API_CALLED_FILE"
        printf '{"tag_name":"%s"}\n' "$TEST_API_VERSION"
        ;;
    *)
        printf '%s\n' "$TEST_REDIRECT_URL"
        ;;
esac
EOF
chmod +x "$TEST_TMP/bin/curl"

export HOME="$TEST_TMP/home"
export PATH="$TEST_TMP/bin:$PATH"
export ROOTLINE_INSTALLER_TESTING=1
export TEST_API_CALLED_FILE="$TEST_TMP/api-called"

# shellcheck source=../../install.sh
. "$ROOT_DIR/install.sh"

assert_equal() {
    expected="$1"
    actual="$2"
    message="$3"
    if [ "$actual" != "$expected" ]; then
        printf 'FAIL: %s: expected %s, got %s\n' "$message" "$expected" "$actual" >&2
        exit 1
    fi
}

TEST_REDIRECT_URL="https://github.com/pablontiv/rootline/releases/tag/v9.8.7"
TEST_API_VERSION="v0.0.0"
export TEST_REDIRECT_URL TEST_API_VERSION
rm -f "$TEST_API_CALLED_FILE"
get_latest_version >/dev/null
assert_equal "v9.8.7" "$VERSION" "redirect tag"
if [ -e "$TEST_API_CALLED_FILE" ]; then
    printf 'FAIL: redirect success called the REST fallback\n' >&2
    exit 1
fi

TEST_REDIRECT_URL="https://github.com/pablontiv/rootline/releases"
TEST_API_VERSION="v7.6.5"
export TEST_REDIRECT_URL TEST_API_VERSION
rm -f "$TEST_API_CALLED_FILE"
get_latest_version >/dev/null
assert_equal "v7.6.5" "$VERSION" "REST fallback tag"
if [ ! -e "$TEST_API_CALLED_FILE" ]; then
    printf 'FAIL: malformed redirect did not call the REST fallback\n' >&2
    exit 1
fi

printf 'install.sh resolver tests passed\n'
