package internal

import (
	"bytes"
	"encoding/binary"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
)

type InuWebSocket struct {
	wsUpgrader websocket.Upgrader

	x11 X11
}

const WSEventMouseMove = 0
const WSEventMouseClick = 1
const WSEventKeyPress = 2
const WSEventScroll = 3
const WSEventPaste = 4

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

	xInt := int(x * float32(SCREEN_WIDTH))
	yInt := int(y * float32(SCREEN_HEIGHT))

	if xInt < 0 || yInt < 0 || xInt >= SCREEN_WIDTH || yInt >= SCREEN_HEIGHT {
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

		inuWs.x11.moveMouse(x, y)

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

		inuWs.x11.clickMouse(jsButton, down)

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

		inuWs.x11.keyPress(keysym, down)

		return

	case WSEventScroll:
		scrollDown, err := buf.ReadByte()
		if err != nil {
			return
		}

		inuWs.x11.scrollMouse(scrollDown == 1)

		return

	case WSEventPaste:
		inuWs.x11.paste(buf.String())
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
			log.Error("ws error:", err)
			return // close ws
		}

		if messageType != websocket.BinaryMessage {
			continue
		}

		inuWs.handleMessage(bytes.NewBuffer(message))
	}
}

func SetupWebSocket(httpMux *http.ServeMux) {
	var inuWs InuWebSocket
	httpMux.HandleFunc("GET /api/ws", inuWs.handleWebSocket)
}
