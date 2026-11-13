# Build

## Host libraries

Install the packages in the README before `make build`. The binary is dynamically linked on purpose. Gio loads the host X11, Wayland, and EGL libraries, so a fully static executable cannot open the window.

Gio 0.8 draws with cgo. On Linux the link line asks pkg-config for:

- `x11`, `xkbcommon`, `xkbcommon-x11`, `x11-xcb`, `xcursor`, `xfixes`
- `wayland-client`, `wayland-cursor`, `egl` when Wayland support is compiled
- `vulkan` when the Vulkan backend is compiled

SQLite uses `modernc.org/sqlite` and does not need cgo. The C toolchain is required because of the window, not the database.

Ubuntu 22.04 packages that satisfied the build of the tested `linux/amd64` binary:

```sh
sudo apt install gcc pkg-config libx11-dev libx11-xcb-dev libxcursor-dev \
  libxfixes-dev libxkbcommon-dev libxkbcommon-x11-dev libwayland-dev \
  libegl1-mesa-dev libvulkan-dev
```

Manjaro and Arch use the pacman line in the README. Those packages supply both the shared libraries and the pkg-config files, except Vulkan: `vulkan-icd-loader` provides `vulkan.pc` and `vulkan-headers` provides `vulkan/vulkan.h`. Both are required. `scripts/ensure-gui-deps.sh` exits immediately when `xkbcommon-x11` and `x11-xcb` are already visible to pkg-config.

On Debian and Ubuntu those two modules are sometimes missing even though the shared libraries are installed. The script then downloads `libxkbcommon-x11-dev` and `libx11-xcb-dev` into `build/third_party` and points pkg-config at the headers. `make build` runs that script. `build/` is gitignored. Without apt, a missing module is an error that names the README, not a silent fallback.

## Commands

```sh
make build
make test
make lint
make doctor
make appimage
make release
make clean
```

`make release` builds `linux/amd64` and attempts `linux/arm64`, then writes `dist/SHA256SUMS` for the binaries that exist. Cross-compiling Gio needs a C cross toolchain and the matching libraries. On this Ubuntu 22.04 machine the amd64 build is the one that can be produced locally. A missing arm64 toolchain is a failed release step, not a hidden success. A binary built on Ubuntu 22.04 is not a promise that the same file runs on every older distribution.

Gio needs cgo and the host graphics libraries. `doctor` reports the binary as dynamic. A fully static executable cannot draw the window. The single-binary release is this one dynamically linked file: it does not embed Node.js, a web server, or a database service. It still needs glibc 2.34 or newer and the packages in the README.

The README table is the list of systems this tree ran on, with a separate package command for each. Manjaro compiled and opened the window. Ubuntu 22.04 compiled the `linux/amd64` file and opened the window. Debian 12, Fedora 41, Arch Linux, and openSUSE Leap 15.6 ran that binary's `doctor` with no display. Linux Mint was not run.

## Fonts

`assets/fonts` embeds Liberation Sans and Liberation Mono from the Ubuntu `fonts-liberation2` package. Both are SIL Open Font License 1.1. The license text is embedded beside the fonts. The reserved font names are unchanged.

## Versions pinned for this tree

- Go 1.25.1 on the development machine
- gioui.org v0.8.0
- modernc.org/sqlite v1.34.5
- golang.org/x/sys v0.36.0
