package internal

import (
	"bytes"
	"encoding/binary"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
)

/*
#cgo LDFLAGS: -lX11 -lXtst
#include <X11/Xlib.h>
#include <X11/extensions/XTest.h>
*/
import "C"

var (
	wsUpgrader = &websocket.Upgrader{}

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

func getMousePos(buf *bytes.Buffer) (int, int, bool) {
	var x, y float32

	err := binary.Read(buf, binary.LittleEndian, &x)
	if err != nil {
		return 0, 0, false
	}

	err = binary.Read(buf, binary.LittleEndian, &y)
	if err != nil {
		return 0, 0, false
	}

	xInt := int(x * float32(SCREEN_WIDTH))
	yInt := int(y * float32(SCREEN_HEIGHT))

	if xInt < 0 || yInt < 0 || xInt >= SCREEN_WIDTH || yInt >= SCREEN_HEIGHT {
		return 0, 0, false
	}

	return xInt, yInt, true
}

const EventTypeMouseMove = 0
const EventTypeMouseClick = 1

func handleMessage(buf *bytes.Buffer) {
	eventType, err := buf.ReadByte()
	if err != nil {
		return
	}

	switch eventType {
	case EventTypeMouseMove:
		x, y, ok := getMousePos(buf)
		if !ok {
			return
		}

		ensureX11Connected()

		C.XWarpPointer(
			display, 0, rootWindow, 0, 0, 0, 0, C.int(x), C.int(y),
		)

		C.XFlush(display)

		return

	case EventTypeMouseClick:
		jsButton, err := buf.ReadByte()
		if err != nil {
			return
		}

		var cButton C.uint

		switch jsButton {
		case 0:
			cButton = C.uint(1) // left
		case 2:
			cButton = C.uint(3) // right
		default:
			return
		}

		down, err := buf.ReadByte()
		if err != nil {
			return
		}

		ensureX11Connected()

		var cErr C.int

		if down == 1 {
			cErr = C.XTestFakeButtonEvent(display, cButton, C.True, C.ulong(0))
		} else {
			cErr = C.XTestFakeButtonEvent(display, cButton, C.False, C.ulong(0))
		}

		if cErr == 0 {
			return
		}

		C.XFlush(display)

		return
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer ws.Close()

	// TODO: limit by framerate

	for {
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			log.Error("ws error:", err)
			return // close ws
		}

		if messageType != websocket.BinaryMessage {
			continue
		}

		handleMessage(bytes.NewBuffer(message))
	}
}

func SetupWebSocket(httpMux *http.ServeMux) {
	httpMux.HandleFunc("GET /api/ws", handleWebSocket)
}
