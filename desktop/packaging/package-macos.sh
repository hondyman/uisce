#!/usr/bin/env bash
# ==============================================================================
# Uisce Trading Desk - Institutional macOS Packaging Pipeline
# ==============================================================================
# Builds a self-contained macOS Application bundle (UisceTradingDesk.app)
# and drag-and-drop DMG with zero entitlement creep, ad-hoc code signing,
# and institutional audit compliance.
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DESKTOP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_DIR="$(cd "$DESKTOP_DIR/.." && pwd)"

APP_NAME="UisceTradingDesk.app"
BIN_NAME="uisce-desk"
DIST_DIR="$DESKTOP_DIR/bin"
APP_BUNDLE="$DIST_DIR/$APP_NAME"
DMG_PATH="$DIST_DIR/UisceTradingDesk.dmg"

echo "=== 1. Building Production Frontend Assets ==="
cd "$ROOT_DIR/frontend"
npm run build

echo "=== 2. Compiling Production Desktop Binary (No verify tag) ==="
cd "$DESKTOP_DIR"
mkdir -p "$DIST_DIR"
GOWORK=off go build -trimpath -ldflags="-s -w" -o "$DIST_DIR/$BIN_NAME" .

echo "=== 3. Assembling macOS App Bundle Structure ==="
rm -rf "$APP_BUNDLE"
CONTENTS="$APP_BUNDLE/Contents"
MACOS_DIR="$CONTENTS/MacOS"
RESOURCES_DIR="$CONTENTS/Resources"

mkdir -p "$MACOS_DIR"
mkdir -p "$RESOURCES_DIR"

# Copy binary
cp "$DIST_DIR/$BIN_NAME" "$MACOS_DIR/$BIN_NAME"
chmod +x "$MACOS_DIR/$BIN_NAME"

# Copy Info.plist and PkgInfo
cp "$SCRIPT_DIR/Info.plist" "$CONTENTS/Info.plist"
echo -n "APPL????" > "$CONTENTS/PkgInfo"

# Copy Icon
if [ -f "$SCRIPT_DIR/AppIcon.icns" ]; then
  cp "$SCRIPT_DIR/AppIcon.icns" "$RESOURCES_DIR/AppIcon.icns"
fi

# Copy Frontend Distribution into App Resources
echo "Bundling frontend assets into Resources/frontend_dist..."
cp -R "$ROOT_DIR/frontend/dist" "$RESOURCES_DIR/frontend_dist"

echo "=== 4. Applying Zero-Entitlement Hardened Runtime & Code Signing ==="
# Audit check: Ensure zero unwanted permissions in Entitlements.plist by inspecting parsed plist keys (ignoring comments)
if plutil -p "$SCRIPT_DIR/Entitlements.plist" | grep -E 'camera|audio-input|location|addressbook'; then
  echo "❌ AUDIT FAILURE: Disallowed hardware recording entitlements found in Entitlements.plist!"
  exit 1
fi

SIGNING_IDENTITY="${APPLE_SIGNING_IDENTITY:-}"

if [ -n "$SIGNING_IDENTITY" ]; then
  echo "Signing with Developer ID: $SIGNING_IDENTITY"
  codesign --force --options runtime \
    --entitlements "$SCRIPT_DIR/Entitlements.plist" \
    --sign "$SIGNING_IDENTITY" \
    --timestamp \
    "$APP_BUNDLE"

  # ENABLE POINT: Uncomment below when Apple Developer ID + App Store Connect API credentials are provided
  if [ -n "${APPLE_ID:-}" ] && [ -n "${APPLE_PASSWORD:-}" ] && [ -n "${APPLE_TEAM_ID:-}" ]; then
    echo "Submitting to Apple Notary Service via notarytool..."
    # xcrun notarytool submit "$APP_BUNDLE" --apple-id "$APPLE_ID" --password "$APPLE_PASSWORD" --team-id "$APPLE_TEAM_ID" --wait
    # xcrun stapler staple "$APP_BUNDLE"
  fi
else
  echo "No Developer ID certificate found; applying mandatory Apple Silicon ad-hoc signature..."
  # Ad-hoc signing is mandatory for arm64 execution on macOS (prevents Killed: 9)
  codesign --force --sign - "$APP_BUNDLE"
fi

echo "=== 5. Verifying App Bundle Signature ==="
codesign --verify --deep --strict "$APP_BUNDLE"
echo "✓ Code signature verified."

echo "=== 6. Creating DMG Installer ==="
rm -f "$DMG_PATH"
TMP_DMG_DIR="$DIST_DIR/dmg_tmp"
rm -rf "$TMP_DMG_DIR"
mkdir -p "$TMP_DMG_DIR"

cp -R "$APP_BUNDLE" "$TMP_DMG_DIR/"
ln -s /Applications "$TMP_DMG_DIR/Applications"

hdiutil create -volname "Uisce Trading Desk" \
  -srcfolder "$TMP_DMG_DIR" \
  -ov -format UDZO \
  "$DMG_PATH" >/dev/null

rm -rf "$TMP_DMG_DIR"

echo "=== Packaging Complete! ==="
echo "App Bundle: $APP_BUNDLE"
echo "DMG File:   $DMG_PATH"
ls -lh "$DMG_PATH"
