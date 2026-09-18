#!/usr/bin/env bash
set -euo pipefail

if [ "${I_ACCEPT_HOST_PARITY:-}" != "1" ]; then
  echo "Docker host-parity requires I_ACCEPT_HOST_PARITY=1" >&2
  echo "Blast radius equals giving a shell as your user (home RW mount)." >&2
  echo "See docs/getting-started/docker-compat.md" >&2
  exit 2
fi

# Prefer host-installed provider bins when present (same-path home mount).
export PATH="${HOST_HOME_DIR:-$HOME}/.local/bin:${HOST_HOME_DIR:-$HOME}/bin:${HOME}/.opencode/bin:${PATH}"

exec nexus "$@"
