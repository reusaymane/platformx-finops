package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/reusaymane/platformx-finops/data-collector/internal/db"
	"go.uber.org/zap"
)

type Server struct {
	srv    *http.Server
	db     *db.DB
	logger *zap.Logger
}

func New(port string, database *db.DB, logger *zap.Logger) *Server {
	s := &Server{db: database, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReady)
	mux.HandleFunc("/metrics", s.handleMetrics)
	s.srv = &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return s
}

func (s *Server) Start() error {
	s.logger.Info("http server listening", zap.String("addr", s.srv.Addr))
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.db.HealthCheck(ctx); err != nil {
		s.logger.Error("readiness check failed", zap.Error(err))
		http.Error(w, `{"status":"not ready","reason":"db unreachable"}`, http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Prometheus metrics endpoint — in prod this uses OTel exporter
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("# platformx data-collector metrics\n"))
}
