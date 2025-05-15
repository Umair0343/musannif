package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/musannif-md/musannif/internal/config"
	"github.com/musannif-md/musannif/internal/logger"
	"github.com/musannif-md/musannif/internal/resolver"
	"github.com/musannif-md/musannif/internal/utils"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	pongWait   = 20 * time.Second
	pingPeriod = pongWait * 9 / 10
)

var (
	upgrader = websocket.Upgrader{
		EnableCompression: true,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func validateSid(r *http.Request) (uuid.UUID, error) {
	uuidStr := r.URL.Query().Get("sid")
	if uuidStr == "" {
		return uuid.Nil, fmt.Errorf("expected session ID via query parameter `/sid`")
	}

	sidUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("couldn't parse session ID: %w", err)
	}

	return sidUUID, nil
}

func CreateWsConn(cfg *config.AppConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			_, ok := err.(websocket.HandshakeError)
			if !ok {
				logger.Log.Error().Err(err).Msg("error establishing websocket connection")
			}
			return
		}
		defer ws.Close()

		sidUUID, err := validateSid(r)
		if err != nil {
			logger.Log.Err(err).Msgf("session ID (uuid) error")

			err := utils.WriteCloseMsg(ws, websocket.CloseUnsupportedData, fmt.Errorf("missing/invalid session ID"))

			if err != nil {
				logger.Log.Err(err).Msgf(utils.UnableToSendCloseMsg)
			}

			return
		}

		err = wsTransmission(cfg, sidUUID, ws, r)
		if err != nil {
			logger.Log.Err(err).Msg("websocket died")
		}
	}
}

func wsTransmission(cfg *config.AppConfig, sid uuid.UUID, ws *websocket.Conn, r *http.Request) error {
	stopReading := make(chan error)
	pingTicker := time.NewTicker(pingPeriod)
	connectErr := resolver.OnClientConnect(cfg, sid, ws, r, stopReading)
	if connectErr != nil {
		err := utils.WriteCloseMsg(ws, websocket.ClosePolicyViolation, connectErr)

		if err != nil {
			logger.Log.Err(err).Msgf(utils.UnableToSendCloseMsg)
		}

		return connectErr
	}

	defer func() {
		resolver.OnClientDisconnect(sid, ws)
		pingTicker.Stop()
		ws.Close()
	}()

	go readFromWs(sid, ws, stopReading)

	for {
		select {
		case reason := <-stopReading: // if someone signalled the end of reading or wants us to be closed
			return reason
		case <-pingTicker.C: // Send sporadic pings
			err := ws.WriteControl(
				websocket.PingMessage,
				[]byte{},
				time.Now().Add(utils.CloseDeadline),
			)

			if err != nil {
				return fmt.Errorf("failed to send ping: %w", err)
			}
		}
	}
}

func readFromWs(sid uuid.UUID, ws *websocket.Conn, readerFinished chan error) {
	defer close(readerFinished)

	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(appdata string) error {
		ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msgJson map[string]any

		// Blocks on read call. Closes return an error here.
		err := ws.ReadJSON(&msgJson)
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				readerFinished <- nil
			} else {
				readerFinished <- fmt.Errorf("reading from connection with session id [%s] failed: %w", sid.String(), err)
			}
			break
		}

		// Parse recieved message
		textStr, ok := msgJson["text"].(string)
		if !ok {
			closeErr := fmt.Errorf("json object didn't contain key 'text'")

			err := utils.WriteCloseMsg(ws, websocket.CloseUnsupportedData, closeErr)

			if err != nil {
				readerFinished <- fmt.Errorf("failed to send close message: %w", err)
			} else {
				readerFinished <- closeErr
			}

			break
		}

		// Propagate message to diff resolver
		err = resolver.OnClientWrite(sid, ws, textStr)
		if err != nil {
			readerFinished <- fmt.Errorf("resolver failed to write in session id [%s] w/ err: %w", sid.String(), err)
			break
		}
	}
}
