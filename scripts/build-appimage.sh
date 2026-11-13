#!/bin/sh
# Build PortPilot AppDir and standalone AppImage
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

DIST_DIR="$ROOT_DIR/dist"
APPDIR="$ROOT_DIR/dist/AppDir"
ARCH="${ARCH:-x86_64}"

echo "==> Ensuring build dependencies..."
sh scripts/ensure-gui-deps.sh

echo "==> Compiling PortPilot binary..."
mkdir -p bin "$DIST_DIR"
PKG_CONFIG_PATH="${ROOT_DIR}/build/third_party/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}" \
  go build -trimpath -ldflags="-s -w" -o bin/portpilot ./cmd/portpilot

echo "==> Preparing AppDir structure at $APPDIR..."
rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin"
mkdir -p "$APPDIR/usr/share/applications"
mkdir -p "$APPDIR/usr/share/icons/hicolor"

# Copy binary
cp -p bin/portpilot "$APPDIR/usr/bin/portpilot"
chmod 755 "$APPDIR/usr/bin/portpilot"

# Export icons using embedded brand asset generator
echo "==> Generating brand icons..."
go run assets/brand/export/main.go "$APPDIR/usr/share/icons/hicolor"

# Root icon and .DirIcon
cp -p "$APPDIR/usr/share/icons/hicolor/256x256/apps/portpilot.png" "$APPDIR/portpilot.png"
cp -p "$APPDIR/portpilot.png" "$APPDIR/.DirIcon"

# Desktop entry
cat > "$APPDIR/usr/share/applications/portpilot.desktop" <<'EOF'
[Desktop Entry]
Type=Application
Name=PortPilot
Comment=Linux sockets, firewall, services, and live network topology
Exec=portpilot %U
Icon=portpilot
Terminal=false
Categories=System;Monitor;Security;
StartupWMClass=portpilot
EOF

cp -p "$APPDIR/usr/share/applications/portpilot.desktop" "$APPDIR/portpilot.desktop"

# AppRun launcher
cat > "$APPDIR/AppRun" <<'EOF'
#!/bin/sh
HERE="$(dirname "$(readlink -f "${0}")")"
export PATH="${HERE}/usr/bin:${PATH}"
export LD_LIBRARY_PATH="${HERE}/usr/lib:${HERE}/usr/lib/x86_64-linux-gnu:${LD_LIBRARY_PATH:-}"
export XDG_DATA_DIRS="${HERE}/usr/share:${XDG_DATA_DIRS:-/usr/local/share:/usr/share}"
exec "${HERE}/usr/bin/portpilot" "$@"
EOF
chmod 755 "$APPDIR/AppRun"

echo "==> AppDir created successfully at $APPDIR"

# Package into .AppImage if appimagetool is found
APPIMAGE_BIN="${APPIMAGETOOL:-$(command -v appimagetool || true)}"

if [ -z "$APPIMAGE_BIN" ] && [ -x "$ROOT_DIR/build/appimagetool" ]; then
  APPIMAGE_BIN="$ROOT_DIR/build/appimagetool"
fi

if [ -n "$APPIMAGE_BIN" ] && [ -x "$APPIMAGE_BIN" ]; then
  echo "==> Building AppImage using $APPIMAGE_BIN..."
  ARCH="$ARCH" "$APPIMAGE_BIN" -n "$APPDIR" "$DIST_DIR/PortPilot-${ARCH}.AppImage"
  test -f "$DIST_DIR/PortPilot-${ARCH}.AppImage" || { echo "Error: Failed to create $DIST_DIR/PortPilot-${ARCH}.AppImage" >&2; exit 1; }
  echo "==> Built: $DIST_DIR/PortPilot-${ARCH}.AppImage"
else
  echo "==> Notice: appimagetool was not found on PATH."
  echo "    The AppDir is ready at: $APPDIR"
  echo "    To package into a single .AppImage file, run:"
  echo "      ARCH=x86_64 appimagetool -n $APPDIR $DIST_DIR/PortPilot-x86_64.AppImage"
fi
