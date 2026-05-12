package identityhistory_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/module/identityhistory"
)

func marshalConfig(cfg identityhistory.Config) ([]byte, error) {
	return json.Marshal(cfg)
}

func TestMemoryRepo_InsertAndListByUser(t *testing.T) {
	repo := identityhistory.NewMemoryIdentityRepo()
	ctx := context.Background()

	rec := identityhistory.IdentityRecord{
		GuildID:  "g1",
		UserID:   "u1",
		Field:    identityhistory.FieldUsername,
		OldValue: "OldName",
		NewValue: "NewName",
	}
	require.NoError(t, repo.Insert(ctx, rec))

	list, err := repo.ListByUser(ctx, "g1", "u1", 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "OldName", list[0].OldValue)
	assert.Equal(t, "NewName", list[0].NewValue)
}

func TestMemoryRepo_LastValue(t *testing.T) {
	repo := identityhistory.NewMemoryIdentityRepo()
	ctx := context.Background()

	// Aucun enregistrement → chaîne vide.
	val, err := repo.LastValue(ctx, "g1", "u1", identityhistory.FieldNickname)
	require.NoError(t, err)
	assert.Empty(t, val)

	_ = repo.Insert(ctx, identityhistory.IdentityRecord{
		GuildID: "g1", UserID: "u1",
		Field: identityhistory.FieldNickname, OldValue: "", NewValue: "Nick1",
	})
	_ = repo.Insert(ctx, identityhistory.IdentityRecord{
		GuildID: "g1", UserID: "u1",
		Field: identityhistory.FieldNickname, OldValue: "Nick1", NewValue: "Nick2",
	})

	val, err = repo.LastValue(ctx, "g1", "u1", identityhistory.FieldNickname)
	require.NoError(t, err)
	assert.Equal(t, "Nick2", val)
}

func TestMemoryRepo_Purge(t *testing.T) {
	repo := identityhistory.NewMemoryIdentityRepo()
	ctx := context.Background()

	old := identityhistory.IdentityRecord{
		GuildID: "g1", UserID: "u1",
		Field: identityhistory.FieldUsername, OldValue: "a", NewValue: "b",
	}
	require.NoError(t, repo.Insert(ctx, old))
	// Forcer une date ancienne.
	repo.Records[0].CreatedAt = time.Now().AddDate(0, 0, -100)

	recent := identityhistory.IdentityRecord{
		GuildID: "g1", UserID: "u1",
		Field: identityhistory.FieldUsername, OldValue: "b", NewValue: "c",
	}
	require.NoError(t, repo.Insert(ctx, recent))

	// Purger les enregistrements antérieurs à 30 jours.
	before := time.Now().AddDate(0, 0, -30)
	n, err := repo.Purge(ctx, "g1", before)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	assert.Len(t, repo.Records, 1)
	assert.Equal(t, "c", repo.Records[0].NewValue)
}

func TestMemoryRepo_IsolationByGuild(t *testing.T) {
	repo := identityhistory.NewMemoryIdentityRepo()
	ctx := context.Background()

	_ = repo.Insert(ctx, identityhistory.IdentityRecord{
		GuildID: "g1", UserID: "u1",
		Field: identityhistory.FieldUsername, OldValue: "", NewValue: "nameA",
	})

	list, err := repo.ListByUser(ctx, "g2", "u1", 10)
	require.NoError(t, err)
	assert.Empty(t, list, "g2 ne doit pas voir les données de g1")
}

func TestMemoryRepo_Limit(t *testing.T) {
	repo := identityhistory.NewMemoryIdentityRepo()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_ = repo.Insert(ctx, identityhistory.IdentityRecord{
			GuildID: "g1", UserID: "u1",
			Field: identityhistory.FieldUsername,
			OldValue: "", NewValue: "x",
		})
	}
	list, err := repo.ListByUser(ctx, "g1", "u1", 3)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}
