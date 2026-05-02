package server

import (
	"context"
	"net/http"
	"time"

	"issue-orchestrator/internal/db"
	"issue-orchestrator/internal/server/routes"
)

type Server struct{ srv *http.Server }

func New(addr string, repo *db.Repository) *Server {
	return &Server{srv: &http.Server{Addr: addr, Handler: routes.New(repo)}}
}
func (s *Server) Start() error { return s.srv.ListenAndServe() }
func (s *Server) Shutdown(ctx context.Context) error {
	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.srv.Shutdown(c)
}
