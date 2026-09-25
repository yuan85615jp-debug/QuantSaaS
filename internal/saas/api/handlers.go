package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/agent/protocol"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/auth"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/config"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/instance"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/lab"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/market"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/ws"
	"go.uber.org/zap"
)

type Server struct {
	Users  *auth.UserService
	Auth   *auth.Service
	Inst   *instance.Service
	Market *market.Service
	Feed   *market.Feed
	Lab    *lab.Service
	Hub    *ws.Hub
	Cfg    *config.Config
	Log    *zap.Logger
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/agent/login", s.handleAgentLogin)
	authMW := AuthMiddleware(s.Auth)
	mux.Handle("GET /api/v1/instances", authMW(http.HandlerFunc(s.handleListInstances)))
	mux.Handle("POST /api/v1/instances", authMW(http.HandlerFunc(s.handleCreateInstance)))
	mux.Handle("GET /api/v1/instances/{id}", authMW(http.HandlerFunc(s.handleGetInstance)))
	mux.Handle("POST /api/v1/instances/{id}/start", authMW(http.HandlerFunc(s.handleStartInstance)))
	mux.Handle("POST /api/v1/instances/{id}/stop", authMW(http.HandlerFunc(s.handleStopInstance)))
	mux.Handle("GET /api/v1/instances/{id}/portfolio", authMW(http.HandlerFunc(s.handleGetPortfolio)))
	mux.Handle("POST /api/v1/instances/{id}/trades", authMW(http.HandlerFunc(s.handleSendTrade)))
	mux.Handle("POST /api/v1/klines/import", authMW(http.HandlerFunc(s.handleImportKlines)))
	mux.Handle("POST /api/v1/klines/seed", authMW(http.HandlerFunc(s.handleSeedKlines)))
	mux.Handle("GET /api/v1/klines", authMW(http.HandlerFunc(s.handleListKlines)))
	mux.Handle("GET /api/v1/klines/last", authMW(http.HandlerFunc(s.handleLastKline)))
	mux.Handle("POST /api/v1/klines/sync", authMW(http.HandlerFunc(s.handleSyncKlines)))
	mux.Handle("POST /api/v1/lab/tasks", authMW(http.HandlerFunc(s.handleCreateLabTask)))
	mux.Handle("GET /api/v1/lab/tasks", authMW(http.HandlerFunc(s.handleListLabTasks)))
	mux.Handle("GET /api/v1/lab/tasks/{id}", authMW(http.HandlerFunc(s.handleGetLabTask)))
	mux.Handle("GET /api/v1/research/regime", authMW(http.HandlerFunc(s.handleResearchRegime)))
	mux.Handle("POST /api/v1/lab/tasks/{id}/run", authMW(http.HandlerFunc(s.handleRunLabTask)))
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "role": string(s.Cfg.AppRole)})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req auth.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, err := s.Users.Register(req)
	if err != nil {
		if errors.Is(err, auth.ErrUserExists) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": u})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	resp, err := s.Users.Login(req)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCreds) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAgentLogin(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	resp, err := s.Users.AgentLogin(req)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCreds) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		if errors.Is(err, auth.ErrRoleNotAllowed) {
			writeError(w, http.StatusForbidden, "agent role required")
			return
		}
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": resp.Token})
}

func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	c := ClaimsFrom(r.Context())
	list, err := s.Inst.ListByUser(c.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list})
}

type createInstanceBody struct {
	TemplateID   string  `json:"template_id"`
	Symbol       string  `json:"symbol"`
	CapitalQuota float64 `json:"capital_quota"`
}

func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	c := ClaimsFrom(r.Context())
	var body createInstanceBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	inst, err := s.Inst.Create(instance.CreateRequest{
		UserID: c.UserID, TemplateID: body.TemplateID, Symbol: body.Symbol, CapitalQuota: body.CapitalQuota,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, instance.ErrTemplate) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, inst)
}

func (s *Server) parseInstanceID(r *http.Request) (uint, error) {
	raw := r.PathValue("id")
	id64, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id64 == 0 {
		return 0, errors.New("invalid instance id")
	}
	return uint(id64), nil
}

func (s *Server) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	c := ClaimsFrom(r.Context())
	id, err := s.parseInstanceID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID := c.UserID
	if c.Role == "admin" {
		userID = 0
	}
	inst, err := s.Inst.Get(id, userID)
	if err != nil {
		if errors.Is(err, instance.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, inst)
}

func (s *Server) handleStartInstance(w http.ResponseWriter, r *http.Request) {
	c := ClaimsFrom(r.Context())
	id, err := s.parseInstanceID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID := c.UserID
	if c.Role == "admin" {
		userID = 0
	}
	inst, err := s.Inst.Start(id, userID)
	if err != nil {
		switch {
		case errors.Is(err, instance.ErrNotFound):
			writeError(w, http.StatusNotFound, "not found")
		case errors.Is(err, instance.ErrAlreadyRunning):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, inst)
}

func (s *Server) handleStopInstance(w http.ResponseWriter, r *http.Request) {
	c := ClaimsFrom(r.Context())
	id, err := s.parseInstanceID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID := c.UserID
	if c.Role == "admin" {
		userID = 0
	}
	inst, err := s.Inst.Stop(id, userID)
	if err != nil {
		switch {
		case errors.Is(err, instance.ErrNotFound):
			writeError(w, http.StatusNotFound, "not found")
		case errors.Is(err, instance.ErrNotRunning):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, inst)
}

func (s *Server) handleGetPortfolio(w http.ResponseWriter, r *http.Request) {
	c := ClaimsFrom(r.Context())
	id, err := s.parseInstanceID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID := c.UserID
	if c.Role == "admin" {
		userID = 0
	}
	if _, err := s.Inst.Get(id, userID); err != nil {
		if errors.Is(err, instance.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	port, err := s.Inst.GetPortfolio(id)
	if err != nil {
		if errors.Is(err, instance.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, port)
}

type sendTradeBody struct {
	Side          string  `json:"side"`
	Engine        string  `json:"engine"`
	Qty           float64 `json:"qty"`
	OrderType     string  `json:"order_type"`
	ClientOrderID string  `json:"client_order_id"`
}

func (s *Server) handleSendTrade(w http.ResponseWriter, r *http.Request) {
	if s.Cfg != nil && !s.Cfg.AllowTradeDispatch() {
		writeError(w, http.StatusForbidden, "trade dispatch disabled for this app_role")
		return
	}
	c := ClaimsFrom(r.Context())
	id, err := s.parseInstanceID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID := c.UserID
	if c.Role == "admin" {
		userID = 0
	}
	inst, err := s.Inst.Get(id, userID)
	if err != nil {
		if errors.Is(err, instance.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var body sendTradeBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	side := strings.ToUpper(strings.TrimSpace(body.Side))
	engine := strings.ToUpper(strings.TrimSpace(body.Engine))
	if side != "BUY" && side != "SELL" {
		writeError(w, http.StatusBadRequest, "side must be BUY or SELL")
		return
	}
	if engine != "MACRO" && engine != "MICRO" {
		writeError(w, http.StatusBadRequest, "engine must be MACRO or MICRO")
		return
	}
	if body.Qty <= 0 {
		writeError(w, http.StatusBadRequest, "qty must be > 0")
		return
	}
	orderType := strings.ToUpper(strings.TrimSpace(body.OrderType))
	if orderType == "" {
		orderType = "MARKET"
	}
	clientOID := strings.TrimSpace(body.ClientOrderID)
	if clientOID == "" {
		clientOID = fmt.Sprintf("api-%d-%d-%s", id, time.Now().UnixMilli(), randomSuffix())
	}
	cmd := protocol.TradeCommand{
		ClientOrderID: clientOID, InstanceID: inst.ID, Symbol: inst.Symbol,
		Side: side, Engine: engine, Qty: body.Qty, OrderType: orderType,
	}
	if s.Hub == nil {
		writeError(w, http.StatusServiceUnavailable, "ws hub not available")
		return
	}
	_ = s.Inst.RecordPendingExecution(inst.ID, cmd.ClientOrderID, cmd.Symbol, store.TradeAction(side), cmd.Qty)
	if err := s.Hub.SendTrade(cmd); err != nil {
		if errors.Is(err, ws.ErrNoAgent) {
			writeError(w, http.StatusServiceUnavailable, "no online agent for instance")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status": "dispatched", "client_order_id": cmd.ClientOrderID,
		"instance_id": cmd.InstanceID, "symbol": cmd.Symbol,
		"side": cmd.Side, "engine": cmd.Engine, "qty": cmd.Qty,
	})
}

func randomSuffix() string {
	return strconv.FormatInt(int64(newRequestSeq()), 36)
}

var requestSeq uint64

func newRequestSeq() uint64 {
	requestSeq++
	return requestSeq
}
