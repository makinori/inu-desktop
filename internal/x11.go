package internal

/*
#cgo LDFLAGS: -lX11 -lXtst
#include <X11/Xlib.h>
#include <X11/extensions/XTest.h>
*/
import "C"
import (
	"github.com/charmbracelet/log"
)

var (
	display    *C.Display
	rootWindow C.Window
)

func ensureX11Connected() {
	if display == nil {
		display = C.XOpenDisplay(nil)
		if display == nil {
			log.Error("cannot open display")
		}
		// TODO: cleanup properly
		// defer C.XCloseDisplay(display)

		screen := C.XDefaultScreen(display)
		rootWindow = C.XRootWindow(display, screen)
	}
}

func moveMouse(x int, y int) {
	ensureX11Connected()

	C.XWarpPointer(
		display, 0, rootWindow, 0, 0, 0, 0, C.int(x), C.int(y),
	)

	C.XFlush(display)
}

func clickMouse(jsButton byte, down byte) {
	var cButton C.uint

	switch jsButton {
	case 0:
		cButton = C.uint(1) // left
	case 2:
		cButton = C.uint(3) // right
	default:
		return
	}

	ensureX11Connected()

	var cErr C.int

	if down == 1 {
		cErr = C.XTestFakeButtonEvent(display, cButton, C.True, C.CurrentTime)
	} else {
		cErr = C.XTestFakeButtonEvent(display, cButton, C.False, C.CurrentTime)
	}

	if cErr == 0 {
		return
	}

	C.XFlush(display)
}

func keyPress(keysym uint32, down byte) {
	ensureX11Connected()

	keycode := C.XKeysymToKeycode(display, C.KeySym(uint64(keysym)))

	var cErr C.int

	if down == 1 {
		cErr = C.XTestFakeKeyEvent(display, C.uint(keycode), C.True, C.CurrentTime)
	} else {
		cErr = C.XTestFakeKeyEvent(display, C.uint(keycode), C.False, C.CurrentTime)
	}

	if cErr == 0 {
		return
	}

	C.XFlush(display)
}

func scrollMouse(scrollDown bool) {
	var cButton C.uint

	if scrollDown {
		cButton = 5 // scroll down
	} else {
		cButton = 4 // scroll up
	}

	ensureX11Connected()

	cErr := C.XTestFakeButtonEvent(display, cButton, C.True, C.CurrentTime)
	if cErr == 0 {
		return
	}

	cErr = C.XTestFakeButtonEvent(display, cButton, C.False, C.CurrentTime)
	if cErr == 0 {
		return
	}

	C.XFlush(display)
}
