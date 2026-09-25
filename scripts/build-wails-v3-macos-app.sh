#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 6 ]; then
  echo "usage: $0 <go-package> <app-path> <executable> <bundle-id> <display-name> <version>" >&2
  exit 2
fi

PACKAGE="$1"
APP="$2"
EXECUTABLE="$3"
BUNDLE_ID="$4"
DISPLAY_NAME="$5"
VERSION="$6"

: "${NEXUS_NATIVE_LDFLAGS:?NEXUS_NATIVE_LDFLAGS is required}"

rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

go build -trimpath -buildvcs=false -tags production \
  -ldflags "$NEXUS_NATIVE_LDFLAGS" \
  -o "$APP/Contents/MacOS/$EXECUTABLE" "$PACKAGE"
chmod 0755 "$APP/Contents/MacOS/$EXECUTABLE"

LS_UI_ELEMENT=false
if [ "$EXECUTABLE" = "nexus-hud" ]; then LS_UI_ELEMENT=true; fi

cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key><string>en</string>
  <key>CFBundleDisplayName</key><string>$DISPLAY_NAME</string>
  <key>CFBundleExecutable</key><string>$EXECUTABLE</string>
  <key>CFBundleIdentifier</key><string>$BUNDLE_ID</string>
  <key>CFBundleInfoDictionaryVersion</key><string>6.0</string>
  <key>CFBundleName</key><string>$DISPLAY_NAME</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>$VERSION</string>
  <key>CFBundleVersion</key><string>$VERSION</string>
  <key>LSMinimumSystemVersion</key><string>11.0</string>
  <key>LSUIElement</key><$LS_UI_ELEMENT/>
  <key>NSHighResolutionCapable</key><true/>
  <key>NSMicrophoneUsageDescription</key>
  <string>Nexus uses the microphone only when you explicitly activate voice input.</string>
</dict>
</plist>
PLIST

plutil -lint "$APP/Contents/Info.plist"
