package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/lab"
)

func (s *Server) requireEvolution(w http.ResponseWriter) bool {
	if s.Cfg == nil || !s.Cfg.AllowEvolution() {
		writeError(w, http.StatusForbidden, "evolution disabled for this app_role (need lab|dev)")
		return false
	}
	if s.Lab == nil {
		writeError(w, http.StatusServiceUnavailable, "lab service not configured")
		return false
	}
	return true
}

type createLabTaskBody struct {
	StrategyID string          `json:"strategy_id"`
	Symbol     string          `json:"symbol"`
	Config     json.RawMessage `json:"config"`
}

func (s *Server) handleCreateLabTask(w http.ResponseWriter, r *http.Request) {
	if !s.requireEvolution(w) {
		return
	}
	var body createLabTaskBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	task, err := s.Lab.CreateTask(lab.CreateTaskRequest{
		StrategyID: body.StrategyID, Symbol: body.Symbol, Config: body.Config,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) handleListLabTasks(w http.ResponseWriter, r *http.Request) {
	if !s.requireEvolution(w) {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := s.Lab.ListTasks(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list})
}

func (s *Server) handleGetLabTask(w http.ResponseWriter, r *http.Request) {
	if !s.requireEvolution(w) {
		return
	}
	id, err := parseUintID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	task, err := s.Lab.GetTask(id)
	if err != nil {
		if errors.Is(err, lab.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleRunLabTask(w http.ResponseWriter, r *http.Request) {
	if !s.requireEvolution(w) {
		return
	}
	id, err := parseUintID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	task, err := s.Lab.StartTask(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, lab.ErrNotFound):
			writeError(w, http.StatusNotFound, "not found")
		case errors.Is(err, lab.ErrBadStatus):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusAccepted, task)
}

func parseUintID(raw string) (uint, error) {
	id64, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id64 == 0 {
		return 0, errors.New("invalid id")
	}
	return uint(id64), nil
}
