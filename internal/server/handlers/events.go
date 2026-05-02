package handlers

import "net/http"

func (h *Handlers) Events(w http.ResponseWriter, r *http.Request) {
	ev, err := h.repo.ListEventsByRun(r.Context(), r.URL.Query().Get("run_id"), 100)
	if err != nil {
		w.WriteHeader(500)
		write(w, map[string]string{"error": err.Error()})
		return
	}
	write(w, ev)
}
