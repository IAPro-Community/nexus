#!/usr/bin/env bash
# IAPro Nexus uninstaller — Nexus v1 supported release target: Linux amd64.
# Removes Nexus-managed executables and desktop launchers while preserving
# profiles/project data by default.
set -euo pipefail

PREFIX="${PREFIX:-$HOME/.local}"
PURGE_DATA=false

for arg in "$@"; do
    case "$arg" in
        --purge-data) PURGE_DATA=true ;;
        --help|-h)
            cat <<'EOF'
IAPro Nexus uninstaller

  --purge-data   Also remove Nexus-owned user configuration and local data.

Environment:
  PREFIX         Installation prefix (default: ~/.local)
EOF
            exit 0
            ;;
        *) echo "Unknown option: $arg" >&2; exit 2 ;;
    esac
done

remove_file() {
    local path="$1"
    if [ -L "$path" ] || [ -f "$path" ]; then
        rm -f "$path"
        echo "Removed $path"
    fi
}

remove_file "$PREFIX/bin/nexus"
remove_file "$PREFIX/bin/ai"
remove_file "$PREFIX/bin/nexus-desktop"
remove_file "$PREFIX/bin/maestro"
remove_file "$PREFIX/bin/orquestrador"
rm -rf "$PREFIX/lib/nexus" "$PREFIX/lib/ai-manager"

# Linux desktop integration created by install.sh.
remove_file "$HOME/.local/share/applications/iapro-nexus.desktop"
remove_file "$HOME/Desktop/IAPro Nexus.desktop"

# macOS user-local Wails app bundle.
for app in "$HOME/Applications/IAPro Nexus.app" "$HOME/Applications/nexus-desktop.app"; do
    if [ -d "$app" ]; then
        rm -rf "$app"
        echo "Removed $app"
    fi
done

# Remove only PATH lines that the Nexus installer itself marked. Leave any
# pre-existing ~/.local/bin configuration untouched.
cleanup_shell_rc() {
    local rc="$1"
    [ -f "$rc" ] || return 0
    local tmp
    tmp="$(mktemp)"
    awk '
      $0 == "# IAPro Nexus" { skip=1; next }
      skip == 1 && $0 == "export PATH=\"\$HOME/.local/bin:\$PATH\"" { skip=0; next }
      { skip=0; print }
    ' "$rc" > "$tmp"
    cat "$tmp" > "$rc"
    rm -f "$tmp"
}
cleanup_shell_rc "$HOME/.bashrc"
cleanup_shell_rc "$HOME/.zshrc"

if [ "$PURGE_DATA" = true ]; then
    rm -rf         "$HOME/.local/share/nexus"         "$HOME/.config/nexus"         "$HOME/.cache/nexus"
    echo "Removed Nexus user configuration/data (--purge-data)."
else
    echo "Profiles and data were preserved."
    echo "Re-run with --purge-data to remove Nexus-owned user configuration/data."
fi

echo "IAPro Nexus uninstall complete."
