package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestHandleListAudit_Pagination(t *testing.T) {
	auditRepo := mock.NewAuditRepository()
	for i := int64(1); i <= 10; i++ {
		auditRepo.Add(makeEvent(i, "guild1", "user1", "username_changed"))
	}

	srv := newTestServerAudit(auditRepo)

	// Page 1 : limit=3, before=0 (depuis le début)
	rr := doAuditRequest(srv, "guild1", "limit=3")
	assert.Equal(t, http.StatusOK, rr.Code)
	var resp auditListResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 3, resp.Count)
	// NextCursor pointe vers le dernier ID retourné pour la page suivante.
	nextCursor := resp.NextCursor
	assert.Greater(t, nextCursor, int64(0))

	// Page 2 : before=nextCursor
	rr2 := doAuditRequest(srv, "guild1", "limit=3&before="+strconv.FormatInt(nextCursor, 10))
	assert.Equal(t, http.StatusOK, rr2.Code)
	var resp2 auditListResponse
	require.NoError(t, json.NewDecoder(rr2.Body).Decode(&resp2))
	assert.Equal(t, 3, resp2.Count)
	// Les IDs de la page 2 doivent être inférieurs au cursor.
	for _, e := range resp2.Events {
		assert.Less(t, e.ID, nextCursor)
	}
}

func TestHandleListAudit_LimitValidation(t *testing.T) {
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
