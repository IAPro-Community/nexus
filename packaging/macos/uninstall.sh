#!/usr/bin/env bash
# IAPro Nexus macOS system-package uninstaller.
# Removes only package-managed binaries/app/receipt. User data is preserved.
set -euo pipefail

if [ "$(uname -s)" != "Darwin" ]; then
  echo "This uninstaller is for macOS only." >&2
  exit 2
fi
if [ "$(id -u)" -ne 0 ]; then
  echo "Run with sudo: sudo nexus-uninstall" >&2
  exit 2
fi

rm -f /usr/local/bin/nexus
rm -f /usr/local/bin/.nexus-pkg
rm -rf "/Applications/IAPro Nexus.app"
rm -rf "/Applications/IAPro Nexus HUD.app"
pkgutil --forget com.iapro.nexus >/dev/null 2>&1 || true

SELF="/usr/local/bin/nexus-uninstall"
if [ -f "$SELF" ]; then
  rm -f "$SELF"
fi

echo "IAPro Nexus system package removed. User profiles and project data were preserved."
