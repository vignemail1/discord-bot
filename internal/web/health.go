package web

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// DBPinger est l'interface minimale requise pour le health check de la base de données.
type DBPinger interface {
	PingContext(ctx context.Context) error
}

// healthResponse est la réponse JSON du endpoint /healthz.
type healthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// handleHealthz retourne l'état du service et de ses dépendances.
// HTTP 200 si tout est opérationnel, 503 si une dépendance critique est indisponible.
func (srv *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := make(map[string]string)
	overall := "ok"

	// Vérification de la base de données si un pinger est disponible.
	if srv.dbPinger != nil {
		if err := srv.dbPinger.PingContext(ctx); err != nil {
			checks["database"] = "error: " + err.Error()
			overall = "degraded"
		} else {
			checks["database"] = "ok"
		}
	}

	resp := healthResponse{
		Status:    overall,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}

	code := http.StatusOK
	if overall != "ok" {
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(resp)
}
