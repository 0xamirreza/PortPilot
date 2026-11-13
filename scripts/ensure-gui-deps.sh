#!/bin/sh
# Gio's X11 backend needs the xkbcommon-x11 and x11-xcb pkg-config modules.
# When the system already provides them, including on Manjaro and Arch, this
# script does nothing. A pkg-config path left in the environment must not hide
# a missing system module. The Debian multiarch library paths are not required:
# Arch-family distros keep those libraries in /usr/lib.
# If the modules are missing and apt-get exists, the Ubuntu debs are unpacked
# under build/third_party and pkg-config is pointed at those headers only.
set -eu
if env -u PKG_CONFIG_PATH pkg-config --exists xkbcommon-x11 x11-xcb; then
  exit 0
fi
if ! command -v apt-get >/dev/null 2>&1 || ! command -v dpkg-deb >/dev/null 2>&1; then
  echo "portpilot: pkg-config modules xkbcommon-x11 and x11-xcb were not found." >&2
  echo "Install the packages in the README, then run make again." >&2
  exit 1
fi

root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)/build/third_party
mkdir -p "$root/debs" "$root/pkgconfig" "$root/lib"
ln -sfn /usr/lib/x86_64-linux-gnu/libxkbcommon-x11.so.0 "$root/lib/libxkbcommon-x11.so"
ln -sfn /usr/lib/x86_64-linux-gnu/libX11-xcb.so.1 "$root/lib/libX11-xcb.so"
if [ ! -f "$root/usr/include/xkbcommon/xkbcommon-x11.h" ] || [ ! -f "$root/usr/include/X11/Xlib-xcb.h" ]; then
  (
    cd "$root/debs"
    apt-get download libxkbcommon-x11-dev libx11-xcb-dev
    for deb in *.deb; do
      dpkg-deb -x "$deb" "$root"
    done
  )
fi

cat > "$root/pkgconfig/xkbcommon-x11.pc" <<EOF
prefix=$root/usr
libdir=$root/lib
includedir=\${prefix}/include
Name: xkbcommon-x11
Description: XKB API common to servers and clients - X11 support
Version: 1.4.0
Requires: xkbcommon
Libs: -L\${libdir} -lxkbcommon-x11
Cflags: -I\${includedir}
EOF

cat > "$root/pkgconfig/x11-xcb.pc" <<EOF
prefix=$root/usr
libdir=$root/lib
includedir=\${prefix}/include
Name: X11 XCB
Description: X Library XCB interface
Version: 1.7.5
Requires: x11 xcb
Libs: -L\${libdir} -lX11-xcb
Cflags: -I\${includedir}
EOF

export PKG_CONFIG_PATH="$root/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
pkg-config --exists xkbcommon-x11 x11-xcb
echo "using local X11 headers from $root" >&2
