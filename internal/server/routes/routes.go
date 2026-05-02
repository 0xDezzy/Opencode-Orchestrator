package routes

import (
	"net/http"

	"issue-orchestrator/internal/db"
	"issue-orchestrator/internal/server/handlers"
)

func New(repo *db.Repository) http.Handler {
	mux := http.NewServeMux()
	h := handlers.New(repo)
	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("GET /readyz", h.Ready)
	mux.HandleFunc("GET /runs", h.Runs)
	mux.HandleFunc("GET /runs/{id}", h.Run)
	mux.HandleFunc("GET /issues", h.Issues)
	mux.HandleFunc("GET /events", h.Events)
	return mux
}
