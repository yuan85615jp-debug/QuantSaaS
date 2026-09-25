package api

import (
	"net/http"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/research"
)

func (s *Server) handleResearchRegime(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	vix, err := research.ParseOptionalFloat(q.Get("vix"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	fear, err := research.ParseOptionalFloat(q.Get("fear_score"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cycle, err := research.ParseOptionalFloat(q.Get("marks_cycle"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	running := q.Get("instance_running") == "1" || q.Get("instance_running") == "true"
	adv := research.Advise(research.Input{
		VIX: vix, FearScore: fear, MarksCycle: cycle, InstanceRunning: running,
	})
	writeJSON(w, http.StatusOK, adv)
}
