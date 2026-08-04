#!/bin/sh
set -e

REPO="pablontiv/rootline"
BINARY="rootline"
INSTALL_DIR="${ROOTLINE_INSTALL_DIR:-}"

main() {
    detect_platform
    detect_arch
    resolve_install_dir
    get_latest_version
    download_and_install
    verify_installation
}

detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$OS" in
        linux)  OS="linux" ;;
        darwin) OS="darwin" ;;
        *)      abort "Unsupported operating system: $OS" ;;
    esac
}

detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)   ARCH="amd64" ;;
        aarch64|arm64)   ARCH="arm64" ;;
        *)               abort "Unsupported architecture: $ARCH" ;;
    esac
}

resolve_install_dir() {
    # ~/.local/bin is the canonical destination: per-user, no sudo, and the same
    # place `just install` and the git hooks use, so a machine never ends up with
    # two rootline binaries in different directories. verify_installation warns
    # if it is not yet on PATH. Override with ROOTLINE_INSTALL_DIR.
    if [ -z "$INSTALL_DIR" ]; then
        INSTALL_DIR="$HOME/.local/bin"
    fi

    # A root-owned override (e.g. /usr/local/bin) needs sudo; the default under
    # $HOME never does.
    if [ ! -w "$INSTALL_DIR" ] && [ ! -w "$(dirname "$INSTALL_DIR")" ]; then
        NEED_SUDO=1
    fi
}

get_latest_version() {
    log "Fetching latest version..."
    RELEASE_URL="$(fetch_latest_release_url 2>/dev/null || true)"
    VERSION="$(version_from_release_url "$RELEASE_URL" || true)"
    if [ -z "$VERSION" ]; then
        VERSION="$(fetch_latest_version_from_api || true)"
    fi
    if [ -z "$VERSION" ]; then
        abort "Could not determine latest version. Check https://github.com/${REPO}/releases"
    fi
    log "Latest version: $VERSION"
}

fetch_latest_release_url() {
    LATEST_URL="https://github.com/${REPO}/releases/latest"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSIL -o /dev/null -w '%{url_effective}\n' "$LATEST_URL"
    elif command -v wget >/dev/null 2>&1; then
        wget --server-response --spider "$LATEST_URL" 2>&1 |
            awk 'tolower($1) == "location:" { location=$2 } END { sub(/\r$/, "", location); print location }'
    else
        return 1
    fi
}

version_from_release_url() {
    RELEASE_PREFIX="https://github.com/${REPO}/releases/tag/"
    case "$1" in
        "${RELEASE_PREFIX}"*)
            RELEASE_VERSION="${1#"$RELEASE_PREFIX"}"
            case "$RELEASE_VERSION" in
                ""|*/*) return 1 ;;
                *) printf '%s\n' "$RELEASE_VERSION" ;;
            esac
            ;;
        *) return 1 ;;
    esac
}

fetch_latest_version_from_api() {
    fetch "https://api.github.com/repos/${REPO}/releases/latest" |
        grep '"tag_name"' |
        sed -E 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/'
}

download_and_install() {
    VERSION_NUM="${VERSION#v}"
    ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

    TMPDIR="$(mktemp -d)"
    trap 'rm -rf "$TMPDIR"' EXIT

    log "Downloading ${ARCHIVE}..."
    fetch "$URL" > "$TMPDIR/$ARCHIVE"

    # Verify checksum — mandatory, abort on failure.
    CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"
    log "Verifying checksum..."
    if ! fetch "$CHECKSUM_URL" > "$TMPDIR/checksums.txt" 2>/dev/null; then
        abort "Could not fetch checksums.txt from ${CHECKSUM_URL}"
    fi
    # Compute-and-compare rather than `sha256sum --check`: the --check/-c flags
    # differ across GNU coreutils and the BSD/macOS sha256sum, so relying on them
    # breaks the installer on macOS. Extracting the expected hash and comparing
    # hex strings is portable and mirrors install.ps1's Get-FileHash approach.
    EXPECTED="$(grep -F "$ARCHIVE" "$TMPDIR/checksums.txt" | awk '{print $1}')"
    if [ -z "$EXPECTED" ]; then
        abort "No checksum listed for ${ARCHIVE} in checksums.txt"
    fi
    ACTUAL="$(sha256_hex "$TMPDIR/$ARCHIVE")"
    if [ "$ACTUAL" != "$EXPECTED" ]; then
        abort "Checksum mismatch for ${ARCHIVE}: expected ${EXPECTED}, got ${ACTUAL}"
    fi
    log "Checksum verified."

    log "Extracting..."
    tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"

    if [ ! -f "$TMPDIR/$BINARY" ]; then
        abort "Binary not found in archive"
    fi

    log "Installing to ${INSTALL_DIR}..."
    mkdir -p "$INSTALL_DIR" 2>/dev/null || true

    if [ "${NEED_SUDO:-}" = "1" ]; then
        log "Requires sudo for /usr/local/bin"
        sudo mkdir -p "$INSTALL_DIR"
        sudo install -m 755 "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
    else
        install -m 755 "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
    fi
}

verify_installation() {
    if command -v "$BINARY" >/dev/null 2>&1; then
        log "Installed $($BINARY --version 2>/dev/null || echo "$BINARY") to $INSTALL_DIR/$BINARY"
    else
        log "Installed to $INSTALL_DIR/$BINARY"
        log "Note: $INSTALL_DIR may not be in your PATH. Add it with:"
        log "  export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
}

fetch() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$1"
    else
        abort "curl or wget is required"
    fi
}

# Print the lowercase hex SHA-256 of a file. Prefers shasum (present on macOS
# and most Linux via Perl); falls back to sha256sum (GNU coreutils / busybox).
# Both print "<hex>  <file>", so the first field is the digest.
sha256_hex() {
    if command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    elif command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    else
        abort "shasum or sha256sum is required to verify the download"
    fi
}

log() {
    printf '%s\n' "$1"
}

abort() {
    printf 'Error: %s\n' "$1" >&2
    exit 1
}

if [ "${ROOTLINE_INSTALLER_TESTING:-}" != "1" ]; then
    main
fi
