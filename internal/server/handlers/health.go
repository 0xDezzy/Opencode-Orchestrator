package handlers

import (
	"encoding/json"
	"net/http"

	"issue-orchestrator/internal/db"
)

type Handlers struct{ repo *db.Repository }

func New(r *db.Repository) *Handlers { return &Handlers{repo: r} }
func write(w http.ResponseWriter, v any) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	write(w, map[string]string{"status": "ok"})
}
func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	sql, _ := h.repo.DB().DB()
	if err := sql.PingContext(r.Context()); err != nil {
		w.WriteHeader(503)
		write(w, map[string]string{"error": err.Error()})
		return
	}
	write(w, map[string]string{"status": "ready"})
}
