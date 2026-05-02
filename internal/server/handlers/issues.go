package handlers

import "net/http"

func (h *Handlers) Issues(w http.ResponseWriter, r *http.Request) {
	snap, err := h.repo.RuntimeSnapshot(r.Context())
	if err != nil {
		w.WriteHeader(500)
		write(w, map[string]string{"error": err.Error()})
		return
	}
	write(w, snap.Issues)
}
