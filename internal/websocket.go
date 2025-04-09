package internal

import (
	"bytes"
	"encoding/binary"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
)

var (
	wsUpgrader = &websocket.Upgrader{}
)

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

const WSEventMouseMove = 0
const WSEventMouseClick = 1
const WSEventKeyPress = 2
const WSEventScroll = 3

func handleMessage(buf *bytes.Buffer) {
	eventType, err := buf.ReadByte()
	if err != nil {
		return
	}

	switch eventType {
	case WSEventMouseMove:
		x, y, ok := getMousePos(buf)
		if !ok {
			return
		}

		moveMouse(x, y)

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

		clickMouse(jsButton, down)

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

		keyPress(keysym, down)

		return

	case WSEventScroll:
		scrollDown, err := buf.ReadByte()
		if err != nil {
			return
		}

		scrollMouse(scrollDown == 1)

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
