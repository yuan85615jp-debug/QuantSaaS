package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/protocol"
	"go.uber.org/zap"
)

type ReportHandler interface {
	OnFill(report protocol.FillReport)
	OnDelta(report protocol.DeltaReport)
}

type NopHandler struct{}

func (NopHandler) OnFill(protocol.FillReport)   {}
func (NopHandler) OnDelta(protocol.DeltaReport) {}

type Hub struct {
	mu           sync.RWMutex
	byAgent      map[string]*Session
	byInstance   map[uint]string
	handler      ReportHandler
	log          *zap.Logger
	heartbeatTTL time.Duration
	stopCh       chan struct{}
}

func NewHub(handler ReportHandler, log *zap.Logger) *Hub {
	if handler == nil {
		handler = NopHandler{}
	}
	if log == nil {
		log = zap.NewNop()
	}
	h := &Hub{
		byAgent:      map[string]*Session{},
		byInstance:   map[uint]string{},
		handler:      handler,
		log:          log,
		heartbeatTTL: 45 * time.Second,
		stopCh:       make(chan struct{}),
	}
	go h.reaper()
	return h
}

func (h *Hub) Close() {
	select {
	case <-h.stopCh:
	default:
		close(h.stopCh)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, s := range h.byAgent {
		s.close()
	}
	h.byAgent = map[string]*Session{}
	h.byInstance = map[uint]string{}
}

func (h *Hub) Register(s *Session) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if old, ok := h.byAgent[s.AgentID]; ok && old != s {
		old.close()
		for inst, aid := range h.byInstance {
			if aid == s.AgentID {
				delete(h.byInstance, inst)
			}
		}
	}
	h.byAgent[s.AgentID] = s
	for _, id := range s.InstanceIDs {
		h.byInstance[id] = s.AgentID
	}
	h.log.Info("agent registered", zap.String("agent_id", s.AgentID), zap.Uints("instances", s.InstanceIDs))
}

func (h *Hub) Unregister(agentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.byAgent[agentID]; ok {
		s.close()
		delete(h.byAgent, agentID)
	}
	for inst, aid := range h.byInstance {
		if aid == agentID {
			delete(h.byInstance, inst)
		}
	}
}

func (h *Hub) OnlineAgents() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]string, 0, len(h.byAgent))
	for id := range h.byAgent {
		out = append(out, id)
	}
	return out
}

func (h *Hub) SendTrade(cmd protocol.TradeCommand) error {
	h.mu.RLock()
	agentID, ok := h.byInstance[cmd.InstanceID]
	if !ok {
		for id := range h.byAgent {
			agentID = id
			ok = true
			break
		}
	}
	s := h.byAgent[agentID]
	h.mu.RUnlock()
	if !ok || s == nil {
		return ErrNoAgent
	}
	return s.Send(protocol.TypeTradeCommand, cmd.ClientOrderID, cmd)
}

func (h *Hub) handleHello(s *Session, hello protocol.Hello) {
	if hello.AgentID != "" {
		s.AgentID = hello.AgentID
	}
	if len(hello.InstanceIDs) > 0 {
		s.InstanceIDs = hello.InstanceIDs
	}
	h.Register(s)
}

func (h *Hub) handleInbound(s *Session, data []byte) {
	w, err := protocol.Decode(data)
	if err != nil {
		h.log.Warn("bad frame", zap.Error(err))
		return
	}
	s.Touch()
	switch w.Type {
	case protocol.TypeHello:
		hello, err := protocol.DecodePayload[protocol.Hello](w)
		if err != nil {
			return
		}
		h.handleHello(s, hello)
	case protocol.TypeHeartbeat:
	case protocol.TypeFillReport:
		rep, err := protocol.DecodePayload[protocol.FillReport](w)
		if err != nil {
			return
		}
		h.handler.OnFill(rep)
	case protocol.TypeDeltaReport:
		rep, err := protocol.DecodePayload[protocol.DeltaReport](w)
		if err != nil {
			return
		}
		h.handler.OnDelta(rep)
	default:
		h.log.Debug("ignored inbound", zap.String("type", string(w.Type)))
	}
}

func (h *Hub) reaper() {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-h.stopCh:
			return
		case <-t.C:
			h.mu.Lock()
			now := time.Now()
			for id, s := range h.byAgent {
				if now.Sub(s.LastSeen()) > h.heartbeatTTL {
					h.log.Warn("agent heartbeat timeout", zap.String("agent_id", id))
					s.close()
					delete(h.byAgent, id)
					for inst, aid := range h.byInstance {
						if aid == id {
							delete(h.byInstance, inst)
						}
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

func MarshalTrade(cmd protocol.TradeCommand) []byte {
	b, _ := json.Marshal(cmd)
	return b
}
