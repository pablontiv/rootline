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
    if [ -n "$INSTALL_DIR" ]; then
        return
    fi

    if echo "$PATH" | tr ':' '\n' | grep -qx "$HOME/.local/bin"; then
        INSTALL_DIR="$HOME/.local/bin"
    elif [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
    else
        INSTALL_DIR="/usr/local/bin"
        NEED_SUDO=1
    fi
}

get_latest_version() {
    log "Fetching latest version..."
    VERSION="$(fetch "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/')"
    if [ -z "$VERSION" ]; then
        abort "Could not determine latest version. Check https://github.com/${REPO}/releases"
    fi
    log "Latest version: $VERSION"
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
    if ! grep -F "$ARCHIVE" "$TMPDIR/checksums.txt" | sha256sum --check --status; then
        abort "Checksum verification failed for ${ARCHIVE}"
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

log() {
    printf '%s\n' "$1"
}

abort() {
    printf 'Error: %s\n' "$1" >&2
    exit 1
}

main
