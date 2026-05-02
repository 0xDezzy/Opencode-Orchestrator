package handlers

import "net/http"

func (h *Handlers) Runs(w http.ResponseWriter, r *http.Request) {
	runs, err := h.repo.ListRecentRuns(r.Context(), 100)
	if err != nil {
		w.WriteHeader(500)
		write(w, map[string]string{"error": err.Error()})
		return
	}
	write(w, runs)
}
func (h *Handlers) Run(w http.ResponseWriter, r *http.Request) {
	events, err := h.repo.ListEventsByRun(r.Context(), r.PathValue("id"), 100)
	if err != nil {
		w.WriteHeader(500)
		write(w, map[string]string{"error": err.Error()})
		return
	}
	write(w, events)
}
