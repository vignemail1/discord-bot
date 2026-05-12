package invitefilter_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/module/invitefilter"
)

func TestAuditRepo_InsertAndListByUser(t *testing.T) {
	repo := invitefilter.NewMemoryAuditRepo()
	ctx := context.Background()

	records := []invitefilter.AuditRecord{
		{GuildID: "g1", UserID: "u1", ChannelID: "c1", MessageID: "m1", Action: invitefilter.ActionDelete, InviteCodes: "abc", Count: 1},
		{GuildID: "g1", UserID: "u1", ChannelID: "c1", MessageID: "m2", Action: invitefilter.ActionTimeout, InviteCodes: "abc", Count: 2},
		{GuildID: "g1", UserID: "u2", ChannelID: "c1", MessageID: "m3", Action: invitefilter.ActionDelete, InviteCodes: "xyz", Count: 1},
	}
	for _, r := range records {
		require.NoError(t, repo.Insert(ctx, r))
	}

	results, err := repo.ListByUser(ctx, "g1", "u1", 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	// Ordre décroissant : dernière insertion en premier.
	assert.Equal(t, invitefilter.ActionTimeout, results[0].Action)
	assert.Equal(t, invitefilter.ActionDelete, results[1].Action)
}

func TestAuditRepo_ListByUser_Limit(t *testing.T) {
	repo := invitefilter.NewMemoryAuditRepo()
	ctx := context.Background()

	for i := range 5 {
		require.NoError(t, repo.Insert(ctx, invitefilter.AuditRecord{
			GuildID: "g1", UserID: "u1",
			MessageID: strings.Repeat("x", i+1),
			Action:    invitefilter.ActionDelete, Count: i + 1,
		}))
	}

	results, err := repo.ListByUser(ctx, "g1", "u1", 3)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestAuditRepo_ListByGuild(t *testing.T) {
	repo := invitefilter.NewMemoryAuditRepo()
	ctx := context.Background()

	for _, uid := range []string{"u1", "u2", "u3"} {
		require.NoError(t, repo.Insert(ctx, invitefilter.AuditRecord{
			GuildID: "g1", UserID: uid, Action: invitefilter.ActionDelete, Count: 1,
		}))
	}
	// Autre guilde : ne doit pas apparaître.
	require.NoError(t, repo.Insert(ctx, invitefilter.AuditRecord{
		GuildID: "g2", UserID: "u9", Action: invitefilter.ActionBan, Count: 3,
	}))

	results, err := repo.ListByGuild(ctx, "g1", 10)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	for _, r := range results {
		assert.Equal(t, "g1", r.GuildID)
	}
}

func TestAuditRepo_InsertErr(t *testing.T) {
	repo := invitefilter.NewMemoryAuditRepo()
	repo.InsertErr = assert.AnError
	err := repo.Insert(context.Background(), invitefilter.AuditRecord{})
	assert.ErrorIs(t, err, assert.AnError)
}

func TestAuditRepo_EmptyResults(t *testing.T) {
	repo := invitefilter.NewMemoryAuditRepo()
	ctx := context.Background()

	results, err := repo.ListByUser(ctx, "g1", "unknown", 10)
	require.NoError(t, err)
	assert.Empty(t, results)

	results, err = repo.ListByGuild(ctx, "unknown_guild", 10)
	require.NoError(t, err)
	assert.Empty(t, results)
}
