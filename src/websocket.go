package src

import (
	"bytes"
	"encoding/binary"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/makinori/inu-desktop/src/config"
	"github.com/makinori/inu-desktop/src/x11"
)

type InuWebSocket struct {
	wsUpgrader websocket.Upgrader
}

const (
	WSEventMouseMove = iota
	WSEventMouseClick
	WSEventKeyPress
	WSEventScroll
	WSEventPaste
)

func getWsMousePos(buf *bytes.Buffer) (int, int, bool) {
	var x, y float32

	err := binary.Read(buf, binary.LittleEndian, &x)
	if err != nil {
		return 0, 0, false
	}

	err = binary.Read(buf, binary.LittleEndian, &y)
	if err != nil {
		return 0, 0, false
	}

	xInt := int(x * float32(config.SCREEN_WIDTH))
	yInt := int(y * float32(config.SCREEN_HEIGHT))

	if xInt < 0 || yInt < 0 ||
		xInt >= config.SCREEN_WIDTH || yInt >= config.SCREEN_HEIGHT {
		return 0, 0, false
	}

	return xInt, yInt, true
}

func (inuWs *InuWebSocket) handleMessage(buf *bytes.Buffer) {
	eventType, err := buf.ReadByte()
	if err != nil {
		return
	}

	switch eventType {
	case WSEventMouseMove:
		x, y, ok := getWsMousePos(buf)
		if !ok {
			return
		}

		x11.MoveMouse(x, y)

		return

	case WSEventMouseClick:
		jsButton, err := buf.ReadByte()
		if err != nil {
			return
		}

		down, err := buf.ReadByte()
		if err != nil {
			return
		}

		x11.ClickMouse(jsButton, down)

		return

	case WSEventKeyPress:
		var keysym uint32
		err := binary.Read(buf, binary.LittleEndian, &keysym)
		if err != nil {
			return
		}

		down, err := buf.ReadByte()
		if err != nil {
			return
		}

		x11.KeyPress(keysym, down)

		return

	case WSEventScroll:
		scrollDown, err := buf.ReadByte()
		if err != nil {
			return
		}

		x11.ScrollMouse(scrollDown == 1)

		return

	case WSEventPaste:
		x11.Paste(buf.String())
		return
	}
}

func (inuWs *InuWebSocket) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := inuWs.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer ws.Close()

	// TODO: limit by framerate

	for {
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			slog.Error("websocket", "err", err.Error())
			return // close ws
		}

		if messageType != websocket.BinaryMessage {
			continue
		}

		inuWs.handleMessage(bytes.NewBuffer(message))
	}
}

func initWebSocket(httpMux *http.ServeMux) {
	var inuWs InuWebSocket
	httpMux.HandleFunc("GET /api/ws", inuWs.handleWebSocket)
}
