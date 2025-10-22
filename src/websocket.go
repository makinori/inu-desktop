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

var wsUpgrader websocket.Upgrader

const (
	WSEventMouseMove = iota
	WSEventMouseClick
	WSEventKeyPress
	WSEventScroll
	WSEventClipboardUpload
	WSEventClipboardDownload
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

func handleMessage(conn *websocket.Conn, buf *bytes.Buffer) {
	eventType, err := buf.ReadByte()
	if err != nil {
		return
	}

	switch eventType {
	case WSEventMouseMove:
		if !config.IN_CONTAINER {
			return
		}

		x, y, ok := getWsMousePos(buf)
		if !ok {
			return
		}

		x11.MoveMouse(x, y)

	case WSEventMouseClick:
		if !config.IN_CONTAINER {
			return
		}

		jsButton, err := buf.ReadByte()
		if err != nil {
			return
		}

		down, err := buf.ReadByte()
		if err != nil {
			return
		}

		x11.ClickMouse(jsButton, down)

	case WSEventKeyPress:
		if !config.IN_CONTAINER {
			return
		}

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

	case WSEventScroll:
		if !config.IN_CONTAINER {
			return
		}

		scrollDown, err := buf.ReadByte()
		if err != nil {
			return
		}

		x11.ScrollMouse(scrollDown == 1)

	case WSEventClipboardUpload:
		x11.SetClipboard(buf.String())

	case WSEventClipboardDownload:
		value, err := x11.GetClipboard()
		if err != nil {
			return
		}

		conn.WriteMessage(websocket.BinaryMessage, append([]byte{
			WSEventClipboardDownload,
		}, value...))

	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// TODO: limit by framerate

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			slog.Error("websocket", "err", err.Error())
			return // close ws
		}

		if messageType != websocket.BinaryMessage {
			continue
		}

		handleMessage(conn, bytes.NewBuffer(message))
	}
}

func initWebSocket(httpMux *http.ServeMux) {
	httpMux.HandleFunc("GET /api/ws", handleWebSocket)
}
