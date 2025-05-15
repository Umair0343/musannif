package resolver

import (
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"sync"

	"github.com/musannif-md/musannif/internal/config"
	"github.com/musannif-md/musannif/internal/utils"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	WS_ARR_START_CAP uint = 2
)

type sessionInfo struct {
	host     *websocket.Conn
	solver   *DiffSolver
	sockets  []*websocket.Conn
	channels []*chan error
}

type SessionInfoMap struct {
	mu    sync.Mutex
	conns map[uuid.UUID]sessionInfo
}

var (
	m = SessionInfoMap{
		conns: make(map[uuid.UUID]sessionInfo),
	}
)

func OnClientConnect(
	cfg *config.AppConfig,
	uuid uuid.UUID,
	ws *websocket.Conn,
	r *http.Request,
	readerFinished chan error,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	si, sessionExists := m.conns[uuid]

	// Session initiator must provide notename via query parameter!
	if !sessionExists {
		username := r.Context().Value("username").(string)
		noteName := r.URL.Query().Get("note_name")
		if noteName == "" {
			return fmt.Errorf("expected note name from session initiator `/note_name`")
		}

		noteName += ".md"

		path := filepath.Join(cfg.App.NoteDirectory, username, noteName)

		si = sessionInfo{
			sockets:  make([]*websocket.Conn, 0, WS_ARR_START_CAP),
			channels: make([]*chan error, 0, WS_ARR_START_CAP),
			solver:   &DiffSolver{fpath: path},
		}

		err := si.solver.initialize()
		if err != nil {
			return fmt.Errorf("failed to initialize diffSolver instance: %w", err)
		}

		si.host = ws
	}

	si.sockets = append(si.sockets, ws)
	si.channels = append(si.channels, &readerFinished)
	m.conns[uuid] = si

	return nil
}

// Write to all clients sharing the same session
func OnClientWrite(uuid uuid.UUID, ws *websocket.Conn, msg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	si, ok := m.conns[uuid]
	if !ok {
		return fmt.Errorf("connection doesn't exist in map")
	}

	// TODO: implement diffing
	_, err := si.solver.resolve()
	if err != nil {
		return fmt.Errorf("diff resolution failed: %w", err)
	}

	for _, c := range si.sockets {
		err := c.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			// TODO: log error if not of type 'client unreachable'
			OnClientDisconnect(uuid, c)
		}
	}

	return nil
}

func OnClientDisconnect(uuid uuid.UUID, ws *websocket.Conn) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	si, ok := m.conns[uuid]

	if !ok {
		return fmt.Errorf("connection being removed doesn't exist (or was already removed)")
	}

	for i, s := range si.sockets {
		if s == ws {
			si.sockets = slices.Delete(si.sockets, i, i+1)
			si.channels = slices.Delete(si.channels, i, i+1)
		}
	}

	m.conns[uuid] = si

	if ws == si.host {
		for i, s := range si.sockets {
			closeErr := fmt.Errorf("session host disconnected")

			err := utils.WriteCloseMsg(s, websocket.ClosePolicyViolation, closeErr)

			if err != nil {
				*si.channels[i] <- fmt.Errorf("failed to send close message: %w", err)
			} else {
				*si.channels[i] <- closeErr
			}
		}
	}

	if len(si.sockets) == 0 {
		si.solver.cleanup()
		delete(m.conns, uuid)
	}

	return nil
}
