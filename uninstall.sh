#!/usr/bin/env bash
set -euo pipefail
PREFIX="${PREFIX:-$HOME/.local}"
rm -f "$PREFIX/bin/nexus"
rm -f "$PREFIX/bin/ai"
rm -f "$PREFIX/bin/maestro"
rm -f "$PREFIX/bin/orquestrador"
rm -rf "$PREFIX/lib/nexus" "$PREFIX/lib/ai-manager"
if command -v systemctl >/dev/null 2>&1; then
  systemctl --user stop nexus-capacity-monitor.service 2>/dev/null || true
  systemctl --user disable nexus-capacity-monitor.service 2>/dev/null || true
fi
rm -f "${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user/nexus-capacity-monitor.service"
echo "Removed executable/helpers. Profiles and data were NOT deleted."
echo "To delete profiles and data too: rm -rf ~/.local/share/ai-manager ~/.config/ai-manager ~/.local/share/nexus ~/.config/nexus"

