#!/usr/bin/env bash
# IAPro Nexus installer (Linux / macOS) — prebuilt signed-release path.
# Usage:
#   ./install.sh --version=vX.Y.Z
#   ./install.sh --version=latest
#   ./install.sh --build-from-source [--yes]
set -euo pipefail

# Closed-beta clients install from the public binaries-only repository.
# NEXUS_RELEASE_REPO overrides it for internal/dev distributions.
REPO="${NEXUS_RELEASE_REPO:-IAPro-Community/nexus-releases}"
GITHUB_URL="https://github.com/${REPO}"
GITHUB_API="https://api.github.com/repos/${REPO}"
RELEASE_DOWNLOAD_BASE="${NEXUS_RELEASE_BASE_URL:-${GITHUB_URL}/releases/download}"
# Source checkouts (--build-from-source) always come from the private source
# repository, never from the binaries mirror.
SOURCE_REPO="${NEXUS_SOURCE_REPO:-IAPro-Community/nexus}"
SOURCE_URL="https://github.com/${SOURCE_REPO}"

VERSION="${NEXUS_VERSION:-}"
BUILD_FROM_SOURCE=false
SOURCE_REF="${NEXUS_SOURCE_REF:-}"
INSTALL_DESKTOP=true
WITH_MAESTRO=false
NO_PATH=false
YES_MODE=false
MAESTRO_EXIT=0
INSTALL_DIR="${NEXUS_INSTALL_DIR:-}"

for arg in "$@"; do
    case "$arg" in
        --version=*) VERSION="${arg#--version=}" ;;
        --build-from-source) BUILD_FROM_SOURCE=true ;;
        --source-ref=*) SOURCE_REF="${arg#--source-ref=}" ;;
        --with-desktop) INSTALL_DESKTOP=true ;;
        --no-desktop) INSTALL_DESKTOP=false ;;
        --with-maestro) WITH_MAESTRO=true ;;
        --no-path) NO_PATH=true ;;
        --install-dir=*) INSTALL_DIR="${arg#--install-dir=}" ;;
        --yes|-y) YES_MODE=true ;;
        --help|-h)
            cat <<'EOF'
IAPro Nexus installer (prebuilt signed-release path)

  --version=vX.Y.Z|latest   Pin a release, or resolve the newest tag via GitHub API
  --no-desktop              Skip native desktop artifact
  --no-path                 Do not append ~/.local/bin to shell rc files
  --install-dir=<path>       Install CLI/Desktop binaries into an explicit directory
  --with-maestro            Opt-in Maestro npm install
  --build-from-source       Build with Go+Bun+make (dev path)
  --source-ref=<ref>        Clone ref when building from source without a checkout
  --yes                     Non-interactive toolchain install for source builds

Environment:
  NEXUS_VERSION, NEXUS_SOURCE_REF, NEXUS_RELEASE_REPO, NEXUS_RELEASE_BASE_URL, NEXUS_INSTALL_DIR
EOF
            exit 0
            ;;
        *) echo "Unknown option: $arg" >&2; exit 2 ;;
    esac
done

if [ -z "$VERSION" ] && [ "$BUILD_FROM_SOURCE" != true ]; then
    echo "A version is required for verified installation: use --version=vX.Y.Z, --version=latest, or NEXUS_VERSION." >&2
    echo "For an explicit source build, use --build-from-source from a checkout." >&2
    exit 2
fi

http_get() {
    local url="$1"
    local out="$2"
    local attempt=1
    while [ "$attempt" -le 3 ]; do
        if command -v curl >/dev/null 2>&1; then
            if curl -fsSL --connect-timeout 15 --max-time 120 "$url" -o "$out"; then
                return 0
            fi
            local code
            code="$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 15 --max-time 30 "$url" || true)"
            if [ "$code" = "403" ] || [ "$code" = "429" ]; then
                echo "GitHub API rate limited (HTTP ${code}); retry ${attempt}/3..." >&2
                sleep $((attempt * 2))
            fi
        elif command -v wget >/dev/null 2>&1; then
            if wget -q -O "$out" "$url"; then
                return 0
            fi
        else
            echo "Need curl or wget to download releases." >&2
            return 1
        fi
        attempt=$((attempt + 1))
    done
    return 1
}

resolve_latest_version() {
    local tmp_json
    tmp_json="$(mktemp)"
    if ! http_get "${GITHUB_API}/releases/latest" "$tmp_json"; then
        rm -f "$tmp_json"
        echo "Could not resolve latest release tag from GitHub API." >&2
        exit 1
    fi
    local tag
    tag="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$tmp_json" | head -n1)"
    rm -f "$tmp_json"
    if [ -z "$tag" ]; then
        echo "GitHub API response did not contain a tag_name." >&2
        exit 1
    fi
    # Integrity only: tag is resolved then assets are checksum-verified. Not strong authenticity.
    echo "Resolved --version=latest to ${tag} (tag resolve only; install still requires signed update-manifest)." >&2
    printf '%s' "$tag"
}

if [ "$VERSION" = "latest" ]; then
    VERSION="$(resolve_latest_version)"
fi

if [ -n "$VERSION" ] && ! printf '%s' "$VERSION" | grep -Eq '^v?[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9.-]+)?$'; then
    echo "Invalid Nexus version: $VERSION" >&2
    exit 2
fi

echo "=== IAPro Nexus Installer (Signed Prebuilt Release) ==="
echo "Release repo: ${REPO}"

OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux) OS_NAME="Linux" ;;
    Darwin) OS_NAME="Darwin" ;;
    *)
        echo "Unsupported OS: $OS. Use install.ps1 on Windows." >&2
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64) ARCH_NAME="x86_64" ;;
    arm64|aarch64) ARCH_NAME="arm64" ;;
    *)
        echo "Unsupported architecture: $ARCH. Please build from source." >&2
        exit 1
        ;;
esac

# Install into the directory that the current shell resolves first. This keeps
# `type -a nexus` and the binary we just installed in sync when an older copy
# exists in a higher-priority PATH entry. A fresh installation still defaults
# to ~/.local/bin.
resolve_nexus_target_dir() {
    local path_entry candidate
    local -a path_entries
    IFS=: read -r -a path_entries <<< "${PATH:-}"
    for path_entry in "${path_entries[@]}"; do
        [ -n "$path_entry" ] || path_entry="."
        candidate="${path_entry}/nexus"
        if [ -x "$candidate" ] && [ ! -d "$candidate" ]; then
            dirname "$candidate"
            return 0
        fi
    done
    printf '%s/.local/bin\n' "$HOME"
}

if [ -n "$INSTALL_DIR" ]; then
    TARGET_DIR="$INSTALL_DIR"
else
    TARGET_DIR="$(resolve_nexus_target_dir)"
fi
mkdir -p "$TARGET_DIR"

INSTALL_SUCCESS=0
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

sha256_file() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        echo "No SHA-256 utility found (need sha256sum or shasum)." >&2
        return 1
    fi
}

install_cli_from_dir() {
    local dir="$1"
    local expected_version="$2"
    local source_binary=""

    if [ -f "${dir}/nexus" ]; then
        source_binary="${dir}/nexus"
    elif [ -f "${dir}/ai" ]; then
        source_binary="${dir}/ai"
    else
        echo "ERROR: verified archive does not contain nexus/ai executable" >&2
        return 1
    fi

    chmod +x "$source_binary"
    local actual_version
    actual_version="$("$source_binary" version --json 2>/dev/null | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
    if [ -z "$actual_version" ] || [ "$actual_version" != "$expected_version" ]; then
        echo "ERROR: signed artifact binary version mismatch: expected ${expected_version}, got ${actual_version:-unknown}" >&2
        return 1
    fi

    install -m 0755 "$source_binary" "${TMP_DIR}/nexus.install"
    mv -f "${TMP_DIR}/nexus.install" "${TARGET_DIR}/nexus"
    INSTALL_SUCCESS=1
}

# Ed25519 public key for manifest signature verification (matches internal/update ProductionTrustRoot).
# Only the corresponding private key (release CI secret) can produce valid signatures.
NEXUS_PUBKEY="ee2e268394ce6a0634a9ea2f414ee831dd536d12f7e2b80107f94566e58aa334"

verify_manifest_signature() {
    local manifest_path="$1"
    local sig_path="$2"
    if [ ! -f "$sig_path" ]; then
        echo "ERROR: update-manifest.sig missing — refusing unsigned install" >&2
        return 1
    fi
    local sig_hex
    sig_hex="$(tr -d '[:space:]' < "$sig_path")"
    if [ -z "$sig_hex" ]; then
        echo "ERROR: update-manifest.sig is empty — refusing unsigned install" >&2
        return 1
    fi

    if command -v python3 >/dev/null 2>&1; then
        if python3 -c "
import sys
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey
pub = Ed25519PublicKey.from_public_bytes(bytes.fromhex('$NEXUS_PUBKEY'))
sig = bytes.fromhex('$sig_hex')
data = open('$manifest_path', 'rb').read()
pub.verify(sig, data)
" 2>/dev/null; then
            echo "Manifest signature VERIFIED (Ed25519)"
            return 0
        fi
        echo "ERROR: manifest signature verification failed" >&2
        return 1
    fi

    echo "ERROR: python3+cryptography required to verify Ed25519 release manifests" >&2
    return 1
}

extract_sha_from_manifest() {
    local manifest_path="$1"
    local artifact_name="$2"
    python3 -c "
import json, sys
from urllib.parse import unquote, urlparse
m = json.load(open(sys.argv[1]))
want = sys.argv[2]
arts = m.get('artifacts', {})
for k, v in arts.items():
    url = unquote(urlparse(v.get('url', '')).path)
    if url.endswith('/' + want) or url.endswith(want):
        print(v.get('sha256', ''))
        sys.exit(0)
key = want.lower().replace('-', '_')
if key.endswith('.tar.gz'):
    key = key[:-7]
elif '.' in key:
    key = key.rsplit('.', 1)[0]
for k, v in arts.items():
    if k == key or key in k or k in key:
        print(v.get('sha256', ''))
        sys.exit(0)
print('', end='')
" "$manifest_path" "$artifact_name" 2>/dev/null
}

download_and_verify_archive() {
    local archive_name="$1"
    local version_plain="$2"
    local release_url="${RELEASE_DOWNLOAD_BASE}/v${version_plain}"
    local archive_path="${TMP_DIR}/${archive_name}"
    local manifest_path="${TMP_DIR}/update-manifest.json"
    local sig_path="${TMP_DIR}/update-manifest.sig"

    echo "Downloading Nexus v${version_plain}: ${archive_name}..."

    local expected=""

    # 1. Fetch signed manifest (required). checksum integrity is bound to this
    # verified manifest; an unsigned release is never an acceptable fallback.
    if ! http_get "${release_url}/update-manifest.json" "$manifest_path"; then
        echo "ERROR: signed update-manifest.json unavailable — refusing install" >&2
        return 1
    fi
    echo "Signed manifest downloaded"
    if ! http_get "${release_url}/update-manifest.sig" "$sig_path"; then
        echo "ERROR: update-manifest.sig unavailable — refusing install" >&2
        return 1
    fi
    if ! verify_manifest_signature "$manifest_path" "$sig_path"; then
        return 1
    fi
    expected="$(extract_sha_from_manifest "$manifest_path" "$archive_name")"
    if [ -z "$expected" ]; then
        echo "ERROR: artifact ${archive_name} missing from signed manifest" >&2
        return 1
    fi
    echo "Using SHA-256 from signed manifest"

    # 2. Download artifact
    if ! http_get "${release_url}/${archive_name}" "$archive_path"; then
        echo "Failed to download ${archive_name}" >&2
        return 1
    fi

    # 3. Verify SHA-256 from signed manifest only
    local actual
    actual="$(sha256_file "$archive_path")"
    if [ "$expected" != "$actual" ]; then
        echo "Release checksum verification FAILED for ${archive_name}." >&2
        echo "  expected: $expected" >&2
        echo "  actual:   $actual" >&2
        return 1
    fi
    echo "SHA-256 verified"

    case "$archive_name" in
        *.tar.gz|*.tgz)
            tar -xzf "$archive_path" -C "$TMP_DIR"
            ;;
        *.zip)
            if command -v ditto >/dev/null 2>&1; then
                ditto -x -k "$archive_path" "$TMP_DIR"
            elif command -v unzip >/dev/null 2>&1; then
                unzip -q -o "$archive_path" -d "$TMP_DIR"
            else
                echo "Need ditto (macOS) or unzip to extract ${archive_name}." >&2
                return 1
            fi
            ;;
        *)
            echo "Unsupported verified archive format: ${archive_name}" >&2
            return 1
            ;;
    esac
    case "$archive_name" in
        nexus_*) install_cli_from_dir "$TMP_DIR" "$version_plain" ;;
    esac
}

ensure_go() {
    if command -v go >/dev/null 2>&1; then
        return 0
    fi
    if [ "$YES_MODE" != true ]; then
        echo "--build-from-source requires Go >=1.25. Install Go and re-run, or pass --yes to attempt package install." >&2
        echo "  https://go.dev/dl/" >&2
        return 1
    fi
    echo "Go not found; attempting package install (--yes)..."
    if command -v apt-get >/dev/null 2>&1; then
        sudo apt-get update -y && sudo apt-get install -y golang-go
    elif command -v dnf >/dev/null 2>&1; then
        sudo dnf install -y golang
    elif command -v brew >/dev/null 2>&1; then
        brew install go
    else
        echo "No supported package manager for Go. Install from https://go.dev/dl/" >&2
        return 1
    fi
    command -v go >/dev/null 2>&1
}

ensure_bun() {
    if command -v bun >/dev/null 2>&1; then
        return 0
    fi
    if [ "$YES_MODE" != true ]; then
        echo "--build-from-source requires Bun >=1.3.9. Install Bun and re-run, or pass --yes." >&2
        echo "  https://bun.sh" >&2
        return 1
    fi
    echo "Bun not found; installing via official script (--yes)..."
    curl -fsSL https://bun.sh/install | bash
    # shellcheck disable=SC1090
    [ -f "$HOME/.bun/bin/bun" ] && export PATH="$HOME/.bun/bin:$PATH"
    command -v bun >/dev/null 2>&1
}

ensure_make_git() {
    local missing=0
    if ! command -v make >/dev/null 2>&1; then
        echo "make is required for --build-from-source." >&2
        missing=1
    fi
    if ! command -v git >/dev/null 2>&1; then
        echo "git is required when cloning --source-ref or for make workflows." >&2
        missing=1
    fi
    return "$missing"
}

VERSION_PLAIN="${VERSION#v}"

if [ -n "$VERSION" ] && [ "$BUILD_FROM_SOURCE" != true ]; then
    ARCHIVE_NAME="nexus_${OS_NAME}_${ARCH_NAME}.tar.gz"
    if ! download_and_verify_archive "$ARCHIVE_NAME" "$VERSION_PLAIN"; then
        echo "Installation failed: verified release artifact unavailable." >&2
        exit 1
    fi
elif [ "$BUILD_FROM_SOURCE" = true ]; then
    echo "Building from source (Go + Bun + make)..."
    ensure_go || exit 1
    ensure_bun || exit 1
    ensure_make_git || exit 1

    BUILD_DIR=""
    if [ -f "./go.mod" ] && [ -d "./cmd/nexus" ]; then
        BUILD_DIR="."
    elif [ -n "$SOURCE_REF" ]; then
        case "$SOURCE_REF" in *[!A-Za-z0-9._/-]*) echo "Invalid source ref." >&2; exit 2 ;; esac
        echo "Cloning explicitly requested source ref ${SOURCE_REF}..."
        git clone --depth 1 --branch "$SOURCE_REF" "${SOURCE_URL}.git" "${TMP_DIR}/repo"
        BUILD_DIR="${TMP_DIR}/repo"
    else
        echo "--build-from-source requires a Nexus checkout or --source-ref=<tag-or-commit>." >&2
        exit 1
    fi

    (
        cd "$BUILD_DIR"
        bun --cwd web install --frozen-lockfile
        make build
        install -m 0755 ./nexus "${TARGET_DIR}/nexus"
    )
    INSTALL_SUCCESS=1
fi

if [ "$INSTALL_SUCCESS" -eq 0 ]; then
    echo "Installation failed: no verified release artifact was installed." >&2
    exit 1
fi

echo "✓ Successfully installed IAPro Nexus to ${TARGET_DIR}/nexus"

if [ "$OS_NAME" = "Linux" ] && [ -n "$VERSION_PLAIN" ] && [ "$BUILD_FROM_SOURCE" != true ]; then
    echo "Optional system packages (when published on the same release):"
    echo "  # Debian/Ubuntu: sudo dpkg -i nexus_*_linux_amd64.deb"
    echo "  # Fedora/RHEL:   sudo rpm -i nexus_*_linux_amd64.rpm"
fi

install_linux_desktop() {
    local source_binary="$1"
    local desktop_target="${TARGET_DIR}/nexus-desktop"
    install -m 0755 "$source_binary" "$desktop_target"

    local applications_dir="${HOME}/.local/share/applications"
    mkdir -p "$applications_dir"
    local desktop_entry="${applications_dir}/iapro-nexus.desktop"
    cat > "$desktop_entry" <<EOF
[Desktop Entry]
Type=Application
Version=1.0
Name=IAPro Nexus
Comment=IAPro Nexus Workspace OS
Exec=${desktop_target}
Icon=application-x-executable
Terminal=false
Categories=Development;
EOF
    chmod +x "$desktop_entry"

    local desktop_dir
    if command -v xdg-user-dir >/dev/null 2>&1; then
        desktop_dir="$(xdg-user-dir DESKTOP 2>/dev/null || true)"
    else
        desktop_dir="${HOME}/Desktop"
    fi
    if [ -z "$desktop_dir" ] || [ "$desktop_dir" = "$HOME" ]; then
        desktop_dir="${HOME}/Desktop"
    fi
    mkdir -p "$desktop_dir"
    cp "$desktop_entry" "${desktop_dir}/IAPro Nexus.desktop"
    chmod +x "${desktop_dir}/IAPro Nexus.desktop"
    echo "✓ Native Desktop installed to ${desktop_target}"
    echo "✓ Launcher created at ${desktop_dir}/IAPro Nexus.desktop"
}

warn_webkitgtk() {
    if [ "$OS_NAME" != "Linux" ]; then
        return 0
    fi
    if ldconfig -p 2>/dev/null | grep -Eq 'libwebkit2gtk-4\.[01]'; then
        return 0
    fi
    echo "⚠️  WebKitGTK not detected. Desktop may fail at runtime; CLI remains usable." >&2
    if command -v apt-get >/dev/null 2>&1; then
        echo "    sudo apt-get install -y libwebkit2gtk-4.1-0" >&2
    elif command -v dnf >/dev/null 2>&1; then
        echo "    sudo dnf install -y webkit2gtk4.1" >&2
    fi
}

if [ "$INSTALL_DESKTOP" = true ] && [ -n "$VERSION_PLAIN" ]; then
    if [ "$OS_NAME" = "Darwin" ]; then
        DESKTOP_ARCHIVE_NAME="nexus-desktop_${OS_NAME}_${ARCH_NAME}.zip"
    else
        DESKTOP_ARCHIVE_NAME="nexus-desktop_${OS_NAME}_${ARCH_NAME}.tar.gz"
    fi

    echo "Attempting to install the native IAPro Nexus Desktop shell..."
    if download_and_verify_archive "$DESKTOP_ARCHIVE_NAME" "$VERSION_PLAIN"; then
        if [ "$OS_NAME" = "Darwin" ]; then
            source_app="$(find "$TMP_DIR" -maxdepth 2 -type d -name '*.app' -print -quit)"
            if [ -z "$source_app" ]; then
                echo "ERROR: macOS Desktop .app bundle missing from verified archive ${DESKTOP_ARCHIVE_NAME}" >&2
                exit 1
            fi
            applications_dir="${HOME}/Applications"
            mkdir -p "$applications_dir"
            app_target="${applications_dir}/$(basename "$source_app")"
            rm -rf "$app_target"
            ditto "$source_app" "$app_target"
            echo "✓ Native Desktop installed to ${app_target}"
        else
            tar -xzf "${TMP_DIR}/${DESKTOP_ARCHIVE_NAME}" -C "$TMP_DIR"
            DESKTOP_BINARY="${TMP_DIR}/nexus-desktop"
            if [ -x "$DESKTOP_BINARY" ]; then
                install_linux_desktop "$DESKTOP_BINARY"
                warn_webkitgtk
            else
                echo "ERROR: Native Desktop executable missing from verified archive ${DESKTOP_ARCHIVE_NAME}" >&2
                exit 1
            fi
        fi
    elif [ "$BUILD_FROM_SOURCE" = true ]; then
        for source_desktop in "./nexus-desktop" "./cmd/nexus-desktop/build/bin/nexus-desktop"; do
            if [ -x "$source_desktop" ]; then
                install_linux_desktop "$source_desktop"
                warn_webkitgtk
                break
            fi
        done
    else
        echo "ERROR: Native Desktop requires an entry in the signed update-manifest (refusing unsigned desktop-checksums.txt fallback)" >&2
        exit 1
    fi
fi

echo ""
if [ "$WITH_MAESTRO" = true ]; then
    echo "Checking Orquestrador Maestro dependency (--with-maestro requested)..."
    if ! command -v orquestrador-maestro >/dev/null 2>&1 && ! command -v maestro >/dev/null 2>&1; then
        if command -v npm >/dev/null 2>&1; then
            echo "Installing Orquestrador Maestro CLI (@iapro/orquestrador-maestro-cli)..."
            if ! npm install -g @iapro/orquestrador-maestro-cli; then
                echo "⚠️  Could not install @iapro/orquestrador-maestro-cli globally with npm." >&2
                MAESTRO_EXIT=2
            fi
        else
            echo "⚠️  Node.js / npm not detected. Maestro unavailable; install npm then re-run with --with-maestro." >&2
            MAESTRO_EXIT=2
        fi
    fi
else
    echo "Maestro auto-install skipped (Nexus does not silently install third-party packages)."
    echo "To install Maestro, run with '--with-maestro' or: npm install -g @iapro/orquestrador-maestro-cli"
fi

MAESTRO_BIN="$(command -v orquestrador-maestro 2>/dev/null || echo "")"
if [ -z "$MAESTRO_BIN" ]; then
    for nvm_bin in "$HOME"/.nvm/versions/node/*/bin/orquestrador-maestro; do
        if [ -x "$nvm_bin" ]; then
            MAESTRO_BIN="$nvm_bin"
            break
        fi
    done
fi
if [ -n "$MAESTRO_BIN" ]; then
    ln -sf "$MAESTRO_BIN" "${TARGET_DIR}/maestro"
    ln -sf "$MAESTRO_BIN" "${TARGET_DIR}/orquestrador"
    echo "✓ Linked Maestro binaries (${TARGET_DIR}/maestro, ${TARGET_DIR}/orquestrador)"
fi

append_path_rc() {
    local rc="$1"
    local line='export PATH="$HOME/.local/bin:$PATH"'
    touch "$rc"
    if grep -Fqs '.local/bin' "$rc"; then
        return 0
    fi
    printf '\n# IAPro Nexus\n%s\n' "$line" >> "$rc"
    echo "✓ Added ~/.local/bin to PATH in ${rc}"
}

if [[ ":$PATH:" != *":$TARGET_DIR:"* ]]; then
    echo ""
    if [ "$NO_PATH" = true ]; then
        echo "⚠️  ${TARGET_DIR} is not in PATH (--no-path). Add manually:"
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    else
        case "${SHELL:-}" in
            */zsh) append_path_rc "${HOME}/.zshrc" ;;
            */bash|*) append_path_rc "${HOME}/.bashrc"
                [ -f "${HOME}/.zshrc" ] && append_path_rc "${HOME}/.zshrc"
                ;;
        esac
        export PATH="${TARGET_DIR}:$PATH"
    fi
fi

echo ""
if command -v nexus >/dev/null 2>&1 || [ -x "${TARGET_DIR}/nexus" ]; then
    echo "Running nexus doctor (warnings do not fail install)..."
    "${TARGET_DIR}/nexus" doctor || true
fi

echo ""
echo "Quick Start:"
echo "  nexus doctor            # Check provider & platform dependencies"
echo "  nexus web               # Launch IAPro Nexus Workspace OS (Web UI)"
echo ""
echo "Note: --version=latest resolves a GitHub tag, then still requires a signed update-manifest.json + Ed25519 signature."

exit "$MAESTRO_EXIT"
