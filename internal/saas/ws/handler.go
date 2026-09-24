package ws

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/auth"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Handler struct {
	Hub  *Hub
	Auth *auth.Service
	Log  *zap.Logger
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if token == "" || h.Auth == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	claims, err := h.Auth.ParseToken(token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Role != "agent" && claims.Role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if h.Log != nil {
			h.Log.Warn("ws upgrade failed", zap.Error(err))
		}
		return
	}
	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		agentID = "agent-unknown"
	}
	sess := newSession(conn, h.Hub, agentID)
	go sess.writePump()
	h.Hub.Register(sess)
	sess.readPump()
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
