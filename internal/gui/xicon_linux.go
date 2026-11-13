package gui

/*
#cgo pkg-config: x11
#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <X11/Xutil.h>
#include <stdlib.h>
*/
import "C"

import (
	"image"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/0xamirreza/portpilot/assets/brand"
)

// The icon is applied from the window goroutine. A second goroutine calling
// Xlib while Gio is using it is not safe unless XInitThreads ran first.
var windowIcon []C.ulong

func (ui *UI) publishIcon() {
	if ui.iconSet || ui.iconTries > 30 {
		return
	}
	ui.iconTries++
	if windowIcon == nil {
		var imgs []image.Image
		for _, size := range []int{16, 24, 32, 48, 64, 128, 256} {
			img, err := brand.ImageAt(size)
			if err != nil {
				ui.iconSet = true
				return
			}
			imgs = append(imgs, img)
		}
		windowIcon = iconCardinals(imgs)
	}
	ui.iconSet = setWindowIcon(filepath.Base(os.Args[0]), windowIcon)
}

func setWindowIcon(class string, data []C.ulong) bool {
	if len(data) == 0 {
		return false
	}
	dpy := C.XOpenDisplay(nil)
	if dpy == nil {
		return false
	}
	defer C.XCloseDisplay(dpy)
	var found bool
	walkWindows(dpy, C.XDefaultRootWindow(dpy), class, func(win C.Window) {
		atomName := C.CString("_NET_WM_ICON")
		atom := C.XInternAtom(dpy, atomName, C.False)
		C.free(unsafe.Pointer(atomName))
		C.XChangeProperty(dpy, win, atom, C.XA_CARDINAL, 32, C.PropModeReplace,
			(*C.uchar)(unsafe.Pointer(&data[0])), C.int(len(data)))
		found = true
	})
	if found {
		C.XFlush(dpy)
	}
	return found
}

func walkWindows(dpy *C.Display, win C.Window, class string, fn func(C.Window)) {
	if windowClass(dpy, win) == class {
		fn(win)
	}
	var root, parent C.Window
	var children *C.Window
	var n C.uint
	if C.XQueryTree(dpy, win, &root, &parent, &children, &n) == 0 || children == nil {
		return
	}
	defer C.XFree(unsafe.Pointer(children))
	list := unsafe.Slice(children, int(n))
	for _, child := range list {
		walkWindows(dpy, child, class, fn)
	}
}

func windowClass(dpy *C.Display, win C.Window) string {
	var hint C.XClassHint
	if C.XGetClassHint(dpy, win, &hint) == 0 {
		return ""
	}
	if hint.res_name != nil {
		defer C.XFree(unsafe.Pointer(hint.res_name))
	}
	if hint.res_class != nil {
		defer C.XFree(unsafe.Pointer(hint.res_class))
	}
	if hint.res_name == nil {
		return ""
	}
	return C.GoString(hint.res_name)
}

func iconCardinals(imgs []image.Image) []C.ulong {
	var out []C.ulong
	for _, img := range imgs {
		b := img.Bounds()
		out = append(out, C.ulong(b.Dx()), C.ulong(b.Dy()))
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bv, a := img.At(x, y).RGBA()
				out = append(out, C.ulong(a>>8)<<24|C.ulong(r>>8)<<16|C.ulong(g>>8)<<8|C.ulong(bv>>8))
			}
		}
	}
	return out
}
