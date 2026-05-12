package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/vignemail1/discord-bot/internal/repository"
)

// auditEventResponse est la vue JSON d'un événement d'audit.
type auditEventResponse struct {
	ID          int64   `json:"id"`
	UserID      string  `json:"user_id"`
	EventType   string  `json:"event_type"`
	OldValue    *string `json:"old_value"`
	NewValue    *string `json:"new_value"`
	ChangedAt   string  `json:"changed_at"`
	SourceEvent string  `json:"source_event"`
}

// auditListResponse enveloppe la liste avec les méta-données de pagination.
type auditListResponse struct {
	Events     []auditEventResponse `json:"events"`
	NextCursor int64                `json:"next_cursor"` // 0 = dernière page
	Count      int                  `json:"count"`
}

func auditEventToResponse(e repository.AuditEvent) auditEventResponse {
	return auditEventResponse{
		ID:          e.ID,
		UserID:      e.UserID,
		EventType:   e.EventType,
		OldValue:    e.OldValue,
		NewValue:    e.NewValue,
		ChangedAt:   e.ChangedAt.UTC().Format(time.RFC3339Nano),
		SourceEvent: e.SourceEvent,
	}
}

// handleListAudit retourne les événements d'audit d'une guilde avec pagination cursor-based.
//
// GET /guilds/{guildID}/audit
//
// Query params :
//   - limit    : nombre d'événements (1-200, défaut 50)
//   - before   : curseur — ID exclusif (récupère les événements antérieurs)
//   - user_id  : filtre sur un membre
//   - type     : filtre sur un event_type
func (srv *Server) handleListAudit(w http.ResponseWriter, r *http.Request) {
	guildID := chi.URLParam(r, "guildID")
	q := r.URL.Query()

	limit := 50
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		} else if err != nil || n <= 0 || n > 200 {
			http.Error(w, "limit must be between 1 and 200", http.StatusBadRequest)
			return
		}
	}

	var before int64
	if v := q.Get("before"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			http.Error(w, "before must be a positive integer", http.StatusBadRequest)
			return
		}
		before = n
	}

	filter := repository.AuditFilter{
		UserID:    q.Get("user_id"),
		EventType: q.Get("type"),
		Before:    before,
		Limit:     limit,
	}

	events, err := srv.auditRepo.ListEvents(r.Context(), guildID, filter)
	if err != nil {
		slog.Error("handlers: ListEvents échoué",
			"guild_id", guildID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resps := make([]auditEventResponse, 0, len(events))
	for _, e := range events {
		resps = append(resps, auditEventToResponse(e))
	}

	// Le curseur pour la page suivante est l'ID du dernier élément (le plus petit, ordre DESC).
	// Si on a reçu moins que limit événements, on est sur la dernière page → cursor = 0.
	var nextCursor int64
	if len(events) == limit {
		nextCursor = events[len(events)-1].ID
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(auditListResponse{
		Events:     resps,
		NextCursor: nextCursor,
		Count:      len(resps),
	})
}
