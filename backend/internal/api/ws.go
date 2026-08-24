package api

import (
	"net/http"

	"github.com/gorilla/websocket"

	"goraft/internal/logutil"
	"goraft/internal/observability"
)

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	up := websocket.Upgrader{CheckOrigin: checkOrigin, ReadBufferSize: 1024, WriteBufferSize: 4096}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		logutil.Warn("ws upgrade failed", logutil.Err(err))
		return
	}
	id, ch := s.fan.Add()
	defer func() {
		s.fan.Remove(id)
		_ = conn.Close()
	}()
	for _, ev := range s.ring.Latest(200) {
		if err := conn.WriteJSON(ev); err != nil {
			return
		}
	}
	for ev := range ch {
		if err := conn.WriteJSON(ev); err != nil {
			return
		}
	}
}

func (s *Server) bindBus() {
	s.bus.Subscribe(func(ev observability.Event) {
		s.ring.Push(ev)
		s.fan.Broadcast(ev)
	})
}
