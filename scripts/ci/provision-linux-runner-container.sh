#!/usr/bin/env bash
set -euo pipefail

# Provision the toolchain needed by the `nexus-ci` self-hosted Linux runner.
# The runner lives inside a container, so this script runs as root inside that
# container and is idempotent: running it again must not change package state.

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root inside the runner container" >&2
  exit 2
fi

APT_OPTS=(
  -o APT::Sandbox::User=root
  -o Dir::Cache=/tmp/aptcache
  -o Dir::Cache::archives=/tmp/aptcache/archives
)

PACKAGES=(
  pkg-config
  python3
  build-essential
  jq
  zip
  unzip
  xdotool
  xvfb
  dbus-x11
  imagemagick
  libglib2.0-dev
  libgtk-3-dev
  libwebkit2gtk-4.1-dev
  libgtk-4-dev
  libwebkitgtk-6.0-dev
  libsoup-3.0-dev
  libcairo2-dev
  libpango1.0-dev
  libx11-dev
  libxtst-dev
  libxkbcommon-dev
  libgbm-dev
  libasound2-dev
  libatk1.0-dev
  libatspi2.0-dev
  libnss3
  libnspr4
  libatk-bridge2.0-0
  libdrm2
  libxcomposite1
  libxdamage1
  libxfixes3
  libxrandr2
  libgbm1
  libasound2t64
  libatspi2.0-0
  libxshmfence1
  libcups2t64
  fonts-liberation
  fonts-dejavu-core
)

export DEBIAN_FRONTEND=noninteractive
# A container restart can leave packages unpacked but unconfigured; apt then
# refuses every install until dpkg finishes its pending work.
dpkg --configure -a || true
apt-get "${APT_OPTS[@]}" update
apt-get "${APT_OPTS[@]}" install -y --no-install-recommends "${PACKAGES[@]}"

echo "provisioning complete"
