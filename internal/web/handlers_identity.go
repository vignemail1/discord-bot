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

// identityStateResponse est la vue JSON d'un état membre.
type identityStateResponse struct {
	UserID          string  `json:"user_id"`
	Username        *string `json:"username"`
	GlobalName      *string `json:"global_name"`
	GuildNick       *string `json:"guild_nick"`
	AvatarHash      *string `json:"avatar_hash"`
	GuildAvatarHash *string `json:"guild_avatar_hash"`
	FirstSeenAt     string  `json:"first_seen_at"`
	LastSeenAt      string  `json:"last_seen_at"`
}

// identityEventResponse est la vue JSON d'un événement d'identité.
type identityEventResponse struct {
	ID          int64   `json:"id"`
	EventType   string  `json:"event_type"`
	OldValue    *string `json:"old_value"`
	NewValue    *string `json:"new_value"`
	ChangedAt   string  `json:"changed_at"`
	SourceEvent string  `json:"source_event"`
}

// identityMemberResponse regroupe l'état courant et l'historique d'un membre.
type identityMemberResponse struct {
	State      identityStateResponse   `json:"state"`
	Events     []identityEventResponse `json:"events"`
	NextCursor int64                   `json:"next_cursor"`
	Count      int                     `json:"count"`
}

func stateToResponse(s repository.IdentityState) identityStateResponse {
	return identityStateResponse{
		UserID:          s.UserID,
		Username:        s.Username,
		GlobalName:      s.GlobalName,
		GuildNick:       s.GuildNick,
		AvatarHash:      s.AvatarHash,
		GuildAvatarHash: s.GuildAvatarHash,
		FirstSeenAt:     s.FirstSeenAt.UTC().Format(time.RFC3339Nano),
		LastSeenAt:      s.LastSeenAt.UTC().Format(time.RFC3339Nano),
	}
}

func identityEventToResp(e repository.IdentityEvent) identityEventResponse {
	return identityEventResponse{
		ID:          e.ID,
		EventType:   e.EventType,
		OldValue:    e.OldValue,
		NewValue:    e.NewValue,
		ChangedAt:   e.ChangedAt.UTC().Format(time.RFC3339Nano),
		SourceEvent: e.SourceEvent,
	}
}

// handleListIdentity retourne la liste des membres connus d'une guilde avec leur état courant.
//
// GET /guilds/{guildID}/identity
func (srv *Server) handleListIdentity(w http.ResponseWriter, r *http.Request) {
	guildID := chi.URLParam(r, "guildID")

	members, err := srv.identityRepo.ListMembers(r.Context(), guildID)
	if err != nil {
		slog.Error("handlers: ListMembers échoué",
			"guild_id", guildID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	result := make([]identityStateResponse, 0, len(members))
	for _, m := range members {
		result = append(result, stateToResponse(m))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// handleGetMemberIdentity retourne l'état courant + l'historique paginé d'un membre.
//
// GET /guilds/{guildID}/identity/{userID}
//
// Query params :
//   - limit    : 1-200, défaut 50
//   - before   : curseur ID exclusif
//   - type     : filtre event_type
func (srv *Server) handleGetMemberIdentity(w http.ResponseWriter, r *http.Request) {
	guildID := chi.URLParam(r, "guildID")
	userID := chi.URLParam(r, "userID")
	q := r.URL.Query()

	limit := 50
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 || n > 200 {
			http.Error(w, "limit must be between 1 and 200", http.StatusBadRequest)
			return
		}
		limit = n
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

	// État courant.
	state, err := srv.identityRepo.GetMember(r.Context(), guildID, userID)
	if err != nil {
		slog.Error("handlers: GetMember échoué",
			"guild_id", guildID, "user_id", userID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if state == nil {
		http.Error(w, "member not found", http.StatusNotFound)
		return
	}

	// Historique paginé.
	events, err := srv.identityRepo.ListMemberEvents(r.Context(), guildID, userID, repository.IdentityFilter{
		EventType: q.Get("type"),
		Before:    before,
		Limit:     limit,
	})
	if err != nil {
		slog.Error("handlers: ListMemberEvents échoué",
			"guild_id", guildID, "user_id", userID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resps := make([]identityEventResponse, 0, len(events))
	for _, e := range events {
		resps = append(resps, identityEventToResp(e))
	}

	var nextCursor int64
	if len(events) == limit {
		nextCursor = events[len(events)-1].ID
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(identityMemberResponse{
		State:      stateToResponse(*state),
		Events:     resps,
		NextCursor: nextCursor,
		Count:      len(resps),
	})
}
