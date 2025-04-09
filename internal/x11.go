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

type X11 struct {
	display    *C.Display
	rootWindow C.Window
}

func (x11 *X11) ensureConnected() bool {
	if x11.display != nil {
		return true
	}

	x11.display = C.XOpenDisplay(nil)
	if x11.display == nil {
		log.Error("cannot open display for x11")
		return false
	}

	screen := C.XDefaultScreen(x11.display)
	x11.rootWindow = C.XRootWindow(x11.display, screen)

	return true

	// defer C.XCloseDisplay(display)
}

// func (x *X11) init() {
// 	go func() {
// 		updateFreq := time.Second * time.Duration(1.0/float64(FRAMERATE))

// 		for {

// 			time.Sleep(updateFreq)
// 		}
// 	}()
// }

func (x11 *X11) moveMouse(x int, y int) {
	x11.ensureConnected()

	C.XWarpPointer(
		x11.display, 0, x11.rootWindow, 0, 0, 0, 0, C.int(x), C.int(y),
	)

	C.XFlush(x11.display)
}

var JS_X11_MOUSE_MAP = map[byte]C.uint{
	0: C.uint(1), // left
	1: C.uint(2), // middle
	2: C.uint(3), // right
	// C.uint(8), // back
	// C.uint(9), // forward
}

func (x11 *X11) clickMouse(jsButton byte, down byte) {
	button, hasButton := JS_X11_MOUSE_MAP[jsButton]
	if !hasButton {
		return
	}

	x11.ensureConnected()

	err := C.XTestFakeButtonEvent(
		x11.display, button, C.int(down), C.CurrentTime,
	)

	if err == 0 {
		return
	}

	C.XFlush(x11.display)
}

func (x11 *X11) keyPress(keysym uint32, down byte) {
	x11.ensureConnected()

	keycode := C.XKeysymToKeycode(x11.display, C.KeySym(uint64(keysym)))

	err := C.XTestFakeKeyEvent(
		x11.display, C.uint(keycode), C.int(down), C.CurrentTime,
	)

	if err == 0 {
		return
	}

	C.XFlush(x11.display)
}

func (x11 *X11) scrollMouse(scrollDown bool) {
	var cButton C.uint
	if scrollDown {
		cButton = 5 // scroll down
	} else {
		cButton = 4 // scroll up
	}

	x11.ensureConnected()

	err := C.XTestFakeButtonEvent(x11.display, cButton, C.True, C.CurrentTime)
	if err == 0 {
		return
	}

	err = C.XTestFakeButtonEvent(x11.display, cButton, C.False, C.CurrentTime)
	if err == 0 {
		return
	}

	C.XFlush(x11.display)
}
