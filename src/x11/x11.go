package x11

/*
#cgo LDFLAGS: -lX11 -lXtst
#include <X11/Xlib.h>
#include <X11/extensions/XTest.h>
*/
import "C"
import "log/slog"

var (
	display    *C.Display
	rootWindow C.Window
)

func ensureConnected() bool {
	if display != nil {
		return true
	}

	display = C.XOpenDisplay(nil)
	if display == nil {
		slog.Error("cannot open display for x11")
		return false
	}

	screen := C.XDefaultScreen(display)
	rootWindow = C.XRootWindow(display, screen)

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

func MoveMouse(x int, y int) {
	ensureConnected()

	C.XWarpPointer(
		display, 0, rootWindow, 0, 0, 0, 0, C.int(x), C.int(y),
	)

	C.XFlush(display)
}

var JS_X11_MOUSE_MAP = map[byte]C.uint{
	0: C.uint(1), // left
	1: C.uint(2), // middle
	2: C.uint(3), // right
	// C.uint(8), // back
	// C.uint(9), // forward
}

func ClickMouse(jsButton byte, down byte) {
	button, hasButton := JS_X11_MOUSE_MAP[jsButton]
	if !hasButton {
		return
	}

	ensureConnected()

	err := C.XTestFakeButtonEvent(
		display, button, C.int(down), C.CurrentTime,
	)

	if err == 0 {
		return
	}

	C.XFlush(display)
}

func keyPressNoFlush(keysym uint32, down byte) {
	ensureConnected()

	keycode := C.XKeysymToKeycode(display, C.KeySym(uint64(keysym)))

	err := C.XTestFakeKeyEvent(
		display, C.uint(keycode), C.int(down), C.CurrentTime,
	)

	if err == 0 {
		return
	}
}

func KeyPress(keysym uint32, down byte) {
	keyPressNoFlush(keysym, down)
	C.XFlush(display)
}

func ScrollMouse(scrollDown bool) {
	var cButton C.uint
	if scrollDown {
		cButton = 5 // scroll down
	} else {
		cButton = 4 // scroll up
	}

	ensureConnected()

	err := C.XTestFakeButtonEvent(display, cButton, C.True, C.CurrentTime)
	if err == 0 {
		return
	}

	err = C.XTestFakeButtonEvent(display, cButton, C.False, C.CurrentTime)
	if err == 0 {
		return
	}

	C.XFlush(display)
}

func Paste(text string) {
	ensureConnected()

	// let go of keys first
	keyPressNoFlush(0xffe3, 0) // left ctrl
	keyPressNoFlush(0xffe4, 0) // right ctrl
	keyPressNoFlush(0x76, 0)   // v
	keyPressNoFlush(0x56, 0)   // V

	// TODO: not accurate. '>' turns into '.'

	for _, char := range text {
		keyPressNoFlush(uint32(char), 1)
		keyPressNoFlush(uint32(char), 0)
	}

	C.XFlush(display)
}
