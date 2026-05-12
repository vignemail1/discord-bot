package invitefilter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/module/invitefilter"
)

func TestMemoryCounterRepo_IncrementAndGet(t *testing.T) {
	repo := invitefilter.NewMemoryCounterRepo()
	ctx := context.Background()

	n, err := repo.Increment(ctx, "g1", "u1", "invite_filter")
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = repo.Increment(ctx, "g1", "u1", "invite_filter")
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	count, err := repo.Get(ctx, "g1", "u1", "invite_filter")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestMemoryCounterRepo_Reset(t *testing.T) {
	repo := invitefilter.NewMemoryCounterRepo()
	ctx := context.Background()

	_, _ = repo.Increment(ctx, "g1", "u1", "invite_filter")
	_, _ = repo.Increment(ctx, "g1", "u1", "invite_filter")

	err := repo.Reset(ctx, "g1", "u1", "invite_filter")
	require.NoError(t, err)

	count, _ := repo.Get(ctx, "g1", "u1", "invite_filter")
	assert.Equal(t, 0, count)
}

func TestMemoryCounterRepo_IsolationByGuild(t *testing.T) {
	repo := invitefilter.NewMemoryCounterRepo()
	ctx := context.Background()

	_, _ = repo.Increment(ctx, "g1", "u1", "invite_filter")
	_, _ = repo.Increment(ctx, "g1", "u1", "invite_filter")

	// même user, autre guilde → compteur indépendant
	count, _ := repo.Get(ctx, "g2", "u1", "invite_filter")
	assert.Equal(t, 0, count)
}
