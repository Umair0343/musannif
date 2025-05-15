package utils

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	CloseDeadline = 5 * time.Second

	UnableToSendCloseMsg = "couldn't send close message to connection"
)

func WriteCloseMsg(ws *websocket.Conn, closeCode int, err error) error {
	return ws.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(closeCode, err.Error()),
		time.Now().Add(CloseDeadline),
	)
}
