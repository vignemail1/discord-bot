package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPinger struct{ err error }

func (m *mockPinger) PingContext(_ context.Context) error { return m.err }

func TestHandleHealthz_OK(t *testing.T) {
	srv := &Server{dbPinger: &mockPinger{}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	srv.handleHealthz(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp healthResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "ok", resp.Checks["database"])
}

func TestHandleHealthz_DBError(t *testing.T) {
	srv := &Server{dbPinger: &mockPinger{err: errors.New("connection refused")}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	srv.handleHealthz(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	var resp healthResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "degraded", resp.Status)
	assert.Contains(t, resp.Checks["database"], "error")
}

func TestHandleHealthz_NoPinger(t *testing.T) {
	srv := &Server{}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	srv.handleHealthz(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp healthResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "ok", resp.Status)
}
