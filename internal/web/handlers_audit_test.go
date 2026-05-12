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

func newTestServerAudit(auditRepo *mock.AuditRepositoryMock) *Server {
	return &Server{
		sessions:  NewSessionStore(),
		auditRepo: auditRepo,
	}
}

func makeEvent(id int64, guildID, userID, evType string) repository.AuditEvent {
	return repository.AuditEvent{
		ID:          id,
		GuildID:     guildID,
		UserID:      userID,
		EventType:   evType,
		SourceEvent: "GUILD_MEMBER_UPDATE",
		ChangedAt:   time.Now().UTC(),
	}
}

func doAuditRequest(srv *Server, guildID, query string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	path := "/guilds/" + guildID + "/audit"
	if query != "" {
		path += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("guildID", guildID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	srv.handleListAudit(rr, req)
	return rr
}

// --- Tests ---

func TestHandleListAudit_OK(t *testing.T) {
	auditRepo := mock.NewAuditRepository()
	for i := int64(1); i <= 5; i++ {
		auditRepo.Add(makeEvent(i, "guild1", "user1", "username_changed"))
	}

	srv := newTestServerAudit(auditRepo)
	rr := doAuditRequest(srv, "guild1", "")

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp auditListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 5, resp.Count)
	assert.Equal(t, int64(0), resp.NextCursor) // moins que limit=50 → dernière page
	// Ordre décroissant : premier élément = ID le plus élevé.
	assert.Equal(t, int64(5), resp.Events[0].ID)
}

func TestHandleListAudit_EmptyGuild(t *testing.T) {
	auditRepo := mock.NewAuditRepository()
	srv := newTestServerAudit(auditRepo)
	rr := doAuditRequest(srv, "guild1", "")

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp auditListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Empty(t, resp.Events)
	assert.Equal(t, int64(0), resp.NextCursor)
}

func TestHandleListAudit_FilterUserID(t *testing.T) {
	auditRepo := mock.NewAuditRepository()
	auditRepo.Add(makeEvent(1, "guild1", "user1", "username_changed"))
	auditRepo.Add(makeEvent(2, "guild1", "user2", "username_changed"))
	auditRepo.Add(makeEvent(3, "guild1", "user1", "nick_changed"))

	srv := newTestServerAudit(auditRepo)
	rr := doAuditRequest(srv, "guild1", "user_id=user1")

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp auditListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 2, resp.Count)
	for _, e := range resp.Events {
		assert.Equal(t, "user1", e.UserID)
	}
}

func TestHandleListAudit_FilterEventType(t *testing.T) {
	auditRepo := mock.NewAuditRepository()
	auditRepo.Add(makeEvent(1, "guild1", "user1", "username_changed"))
	auditRepo.Add(makeEvent(2, "guild1", "user1", "nick_changed"))
	auditRepo.Add(makeEvent(3, "guild1", "user2", "nick_changed"))

	srv := newTestServerAudit(auditRepo)
	rr := doAuditRequest(srv, "guild1", "type=nick_changed")

	var resp auditListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 2, resp.Count)
	for _, e := range resp.Events {
		assert.Equal(t, "nick_changed", e.EventType)
	}
}

func TestHandleListAudit_PaginationCursor(t *testing.T) {
	auditRepo := mock.NewAuditRepository()
	for i := int64(1); i <= 10; i++ {
		auditRepo.Add(makeEvent(i, "guild1", "user1", "username_changed"))
	}

	srv := newTestServerAudit(auditRepo)

	// Première page : limit=4
	rr := doAuditRequest(srv, "guild1", "limit=4")
	var page1 auditListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&page1))
	assert.Equal(t, 4, page1.Count)
	assert.Equal(t, int64(10), page1.Events[0].ID)         // plus récent en premier
	assert.NotEqual(t, int64(0), page1.NextCursor)          // pas dernière page

	// Deuxième page via curseur.
	rr2 := doAuditRequest(srv, "guild1", "limit=4&before="+strconv.FormatInt(page1.NextCursor, 10))
	var page2 auditListResponse
	require.NoError(t, json.NewDecoder(rr2.Body).Decode(&page2))
	assert.Equal(t, 4, page2.Count)
	// Aucun événement de page2 ne doit être dans page1.
	p1ids := make(map[int64]struct{}, 4)
	for _, e := range page1.Events {
		p1ids[e.ID] = struct{}{}
	}
	for _, e := range page2.Events {
		_, dup := p1ids[e.ID]
		assert.False(t, dup, "ID %d dupliqué entre page1 et page2", e.ID)
	}
}

func TestHandleListAudit_LimitInvalid(t *testing.T) {
	auditRepo := mock.NewAuditRepository()
	srv := newTestServerAudit(auditRepo)

	// limit=0 invalide.
	rr := doAuditRequest(srv, "guild1", "limit=0")
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// limit=201 invalide.
	rr = doAuditRequest(srv, "guild1", "limit=201")
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleListAudit_BeforeInvalid(t *testing.T) {
	auditRepo := mock.NewAuditRepository()
	srv := newTestServerAudit(auditRepo)

	rr := doAuditRequest(srv, "guild1", "before=abc")
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	rr = doAuditRequest(srv, "guild1", "before=-5")
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleListAudit_RepoError(t *testing.T) {
	auditRepo := mock.NewAuditRepository()
	auditRepo.ListErr = assert.AnError
	srv := newTestServerAudit(auditRepo)

	rr := doAuditRequest(srv, "guild1", "")
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
