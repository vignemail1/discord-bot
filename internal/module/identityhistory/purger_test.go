package identityhistory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/module/identityhistory"
)

func TestPurger_DeletesOldRecords(t *testing.T) {
	repo := identityhistory.NewMemoryIdentityRepo()
	ctx := context.Background()

	// 1 enregistrement ancien.
	require.NoError(t, repo.Insert(ctx, identityhistory.IdentityRecord{
		GuildID: "g1", UserID: "u1",
		Field: identityhistory.FieldNickname, OldValue: "", NewValue: "old",
	}))
	repo.Records[0].CreatedAt = time.Now().AddDate(0, 0, -91)

	// 1 enregistrement récent.
	require.NoError(t, repo.Insert(ctx, identityhistory.IdentityRecord{
		GuildID: "g1", UserID: "u1",
		Field: identityhistory.FieldNickname, OldValue: "old", NewValue: "new",
	}))

	purger := identityhistory.NewPurger(
		repo,
		func() []string { return []string{"g1"} },
		func(id string) identityhistory.Config {
			return identityhistory.Config{
				TrackNickname: true,
				RetentionDays: 90,
			}
		},
		1*time.Millisecond, // intervalle très court pour le test
	)

	// Appeler run() directement (exporté uniquement pour les tests via RunOnce).
	purger.RunOnce(ctx)

	assert.Len(t, repo.Records, 1)
	assert.Equal(t, "new", repo.Records[0].NewValue)
}

func TestPurger_ZeroRetention_NoDelete(t *testing.T) {
	repo := identityhistory.NewMemoryIdentityRepo()
	ctx := context.Background()

	require.NoError(t, repo.Insert(ctx, identityhistory.IdentityRecord{
		GuildID: "g1", UserID: "u1",
		Field: identityhistory.FieldUsername, OldValue: "", NewValue: "x",
	}))
	repo.Records[0].CreatedAt = time.Now().AddDate(-1, 0, 0)

	purger := identityhistory.NewPurger(
		repo,
		func() []string { return []string{"g1"} },
		func(id string) identityhistory.Config {
			return identityhistory.Config{RetentionDays: -1} // conservation indéfinie
		},
		1*time.Millisecond,
	)
	purger.RunOnce(ctx)
	assert.Len(t, repo.Records, 1, "RetentionDays=-1 ne doit pas purger")
}
