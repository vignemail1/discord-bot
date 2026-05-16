package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/repository"
	"github.com/vignemail1/discord-bot/internal/repository/mock"
)

func newTestServerIdentity(idRepo *mock.IdentityRepositoryMock) *Server {
	return &Server{
		sessions:     NewSessionStore(),
		identityRepo: idRepo,
	}
}

func doIdentityRequest(srv *Server, path string, params map[string]string) (*httptest.ResponseRecorder, *http.Request) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return rr, req
}

func ptr(s string) *string { return &s }

func makeState(guildID, userID, username string) repository.IdentityState {
	return repository.IdentityState{
		GuildID:     guildID,
		UserID:      userID,
		Username:    ptr(username),
		FirstSeenAt: time.Now().UTC(),
		LastSeenAt:  time.Now().UTC(),
	}
}

func makeIdentityEvent(id int64, guildID, userID, evType string) repository.IdentityEvent {
	return repository.IdentityEvent{
		ID: id, GuildID: guildID, UserID: userID,
		EventType: evType, SourceEvent: "GUILD_MEMBER_UPDATE",
		ChangedAt: time.Now().UTC(),
	}
}

// --- handleListIdentity ---

func TestHandleListIdentity_OK(t *testing.T) {
	idRepo := mock.NewIdentityRepository()
	idRepo.AddState(makeState("guild1", "user1", "Alice"))
	idRepo.AddState(makeState("guild1", "user2", "Bob"))

	srv := newTestServerIdentity(idRepo)
	rr, req := doIdentityRequest(srv, "/guilds/guild1/identity", map[string]string{"guildID": "guild1"})
	srv.handleListIdentity(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []identityStateResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Len(t, result, 2)
}

func TestHandleListIdentity_Empty(t *testing.T) {
	idRepo := mock.NewIdentityRepository()
	srv := newTestServerIdentity(idRepo)
	rr, req := doIdentityRequest(srv, "/guilds/guild1/identity", map[string]string{"guildID": "guild1"})
	srv.handleListIdentity(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []identityStateResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Empty(t, result)
}

func TestHandleListIdentity_RepoError(t *testing.T) {
	idRepo := mock.NewIdentityRepository()
	idRepo.ListMembersErr = assert.AnError
	srv := newTestServerIdentity(idRepo)
	rr, req := doIdentityRequest(srv, "/guilds/guild1/identity", map[string]string{"guildID": "guild1"})
	srv.handleListIdentity(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- handleGetMemberIdentity ---

func TestHandleGetMemberIdentity_OK(t *testing.T) {
	idRepo := mock.NewIdentityRepository()
	idRepo.AddState(makeState("guild1", "user1", "Alice"))
	for i := int64(1); i <= 3; i++ {
		idRepo.AddEvent(makeIdentityEvent(i, "guild1", "user1", "username_changed"))
	}

	srv := newTestServerIdentity(idRepo)
	rr, req := doIdentityRequest(srv, "/guilds/guild1/identity/user1",
		map[string]string{"guildID": "guild1", "userID": "user1"})
	srv.handleGetMemberIdentity(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp identityMemberResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "user1", resp.State.UserID)
	assert.Equal(t, 3, resp.Count)
	assert.Equal(t, int64(0), resp.NextCursor)
	assert.Equal(t, int64(3), resp.Events[0].ID)
}

func TestHandleGetMemberIdentity_NotFound(t *testing.T) {
	idRepo := mock.NewIdentityRepository()
	srv := newTestServerIdentity(idRepo)
	rr, req := doIdentityRequest(srv, "/guilds/guild1/identity/unknown",
		map[string]string{"guildID": "guild1", "userID": "unknown"})
	srv.handleGetMemberIdentity(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandleGetMemberIdentity_RepoError(t *testing.T) {
	idRepo := mock.NewIdentityRepository()
	idRepo.GetMemberErr = assert.AnError
	srv := newTestServerIdentity(idRepo)
	rr, req := doIdentityRequest(srv, "/guilds/guild1/identity/user1",
		map[string]string{"guildID": "guild1", "userID": "user1"})
	srv.handleGetMemberIdentity(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
