package inuws

import (
	"bytes"
	"context"
	"encoding/binary"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/makinori/inu-desktop/src/config"
	"github.com/makinori/inu-desktop/src/webrtc"
	"github.com/makinori/inu-desktop/src/x11"
	"github.com/maniartech/signals"
)

var (
	upgrader websocket.Upgrader

	conns      []*websocket.Conn
	connsMutex sync.RWMutex

	viewerCount *atomic.Uint32
)

const (
	WSEventMouseMove = iota
	WSEventMouseClick
	WSEventKeyPress
	WSEventScroll
	WSEventClipboardUpload
	WSEventClipboardDownload
	WSEventViewerCount
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

		x, y, ok := getMousePos(buf)
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

func sendViewerCountMessage(conn *websocket.Conn, value uint32) {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(WSEventViewerCount)
	binary.Write(buf, binary.LittleEndian, value)
	conn.WriteMessage(websocket.BinaryMessage, buf.Bytes())
}

func onConnected(conn *websocket.Conn) {
	sendViewerCountMessage(conn, webrtc.ViewerCount.Load())
}

func onDisconnected(conn *websocket.Conn) {
}

func handleEndpoint(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	connsMutex.Lock()
	conns = append(conns, conn)
	connsMutex.Unlock()

	conn.SetCloseHandler(func(_ int, _ string) error {
		connsMutex.Lock()
		defer connsMutex.Unlock()

		i := slices.Index(conns, conn)
		if i < 0 {
			return nil
		}

		conns = slices.Delete(conns, i, i+1)

		onDisconnected(conn)

		return nil
	})

	onConnected(conn)

	// TODO: limit by framerate

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway) {
				return
			}
			slog.Error("websocket", "err", err.Error())
			return // close ws
		}

		if messageType != websocket.BinaryMessage {
			continue
		}

		handleMessage(conn, bytes.NewBuffer(message))
	}
}

func onViewerCountChanged(value uint32) {
	connsMutex.RLock()
	defer connsMutex.RUnlock()

	for _, conn := range conns {
		sendViewerCountMessage(conn, value)
	}
}

func Init(
	httpMux *http.ServeMux,
	viewerCountPtr *atomic.Uint32,
	viewerCountSignal signals.Signal[uint32],
) {
	httpMux.HandleFunc("GET /api/ws", handleEndpoint)

	viewerCount = viewerCountPtr

	viewerCountSignal.AddListener(func(_ context.Context, value uint32) {
		onViewerCountChanged(value)
	})
}
