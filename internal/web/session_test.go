package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionStore_CreateAndGet(t *testing.T) {
	store := NewSessionStore()

	sess, err := store.Create()
	require.NoError(t, err)
	require.NotNil(t, sess)
	assert.NotEmpty(t, sess.ID)
	assert.NotEmpty(t, sess.StateToken)
	assert.False(t, sess.IsAuthenticated())

	got := store.Get(sess.ID)
	require.NotNil(t, got)
	assert.Equal(t, sess.ID, got.ID)
	assert.Equal(t, sess.StateToken, got.StateToken)
}

func TestSessionStore_GetUnknown(t *testing.T) {
	store := NewSessionStore()
	assert.Nil(t, store.Get("unknown-id"))
}

func TestSessionStore_Delete(t *testing.T) {
	store := NewSessionStore()
	sess, err := store.Create()
	require.NoError(t, err)

	store.Delete(sess.ID)
	assert.Nil(t, store.Get(sess.ID))
}

func TestSessionStore_Save(t *testing.T) {
	store := NewSessionStore()
	sess, err := store.Create()
	require.NoError(t, err)

	sess.UserID = "123456789"
	sess.Username = "testuser"
	store.Save(sess)

	got := store.Get(sess.ID)
	require.NotNil(t, got)
	assert.Equal(t, "123456789", got.UserID)
	assert.True(t, got.IsAuthenticated())
}

func TestSessionStore_Expiry(t *testing.T) {
	store := NewSessionStore()
	sess, err := store.Create()
	require.NoError(t, err)

	// Forcer l'expiration.
	store.mu.Lock()
	store.entries[sess.ID].expiresAt = time.Now().Add(-time.Second)
	store.mu.Unlock()

	assert.Nil(t, store.Get(sess.ID), "session expirée doit retourner nil")
}

func TestSessionStore_GC(t *testing.T) {
	store := NewSessionStore()
	sess1, _ := store.Create()
	sess2, _ := store.Create()

	// Expirer sess1.
	store.mu.Lock()
	store.entries[sess1.ID].expiresAt = time.Now().Add(-time.Second)
	store.mu.Unlock()

	store.GC()

	assert.Nil(t, store.Get(sess1.ID), "sess1 doit être purgée")
	assert.NotNil(t, store.Get(sess2.ID), "sess2 doit rester")
}

func TestSessionStore_UniqueIDs(t *testing.T) {
	store := NewSessionStore()
	ids := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		s, err := store.Create()
		require.NoError(t, err)
		_, dup := ids[s.ID]
		assert.False(t, dup, "ID dupliqué détecté : %s", s.ID)
		ids[s.ID] = struct{}{}
	}
}
