package identityhistory_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/cache"
	"github.com/vignemail1/discord-bot/internal/module/identityhistory"
	"github.com/vignemail1/discord-bot/internal/repository"
)

// buildGuildConfig construit un GuildConfig en mémoire pour les tests du module.
func buildGuildConfig(guildID string, cfg identityhistory.Config) *cache.GuildConfig {
	cfgJSON, _ := json.Marshal(cfg)
	return &cache.GuildConfig{
		GuildID: guildID,
		Modules: map[string]repository.GuildModule{
			identityhistory.ModuleName: {
				ModuleName: identityhistory.ModuleName,
				Enabled:    true,
				ConfigJSON: cfgJSON,
			},
		},
	}
}

// newMemberUpdate construit un GuildMemberUpdate en utilisant l'embedding *Member
// (discordgo.GuildMemberUpdate n'a pas de champ GuildMember — il embed *Member directement).
func newMemberUpdate(guildID, userID, nick, guildAvatar string) *discordgo.GuildMemberUpdate {
	return &discordgo.GuildMemberUpdate{
		Member: &discordgo.Member{
			GuildID: guildID,
			Nick:    nick,
			Avatar:  guildAvatar,
			User: &discordgo.User{
				ID: userID,
			},
		},
	}
}

// TestHandleMemberUpdate_NickChange vérifie qu'un changement de nickname est enregistré.
func TestHandleMemberUpdate_NickChange(t *testing.T) {
	repo := identityhistory.NewMemoryIdentityRepo()
	mod := identityhistory.New(repo)
	ctx := context.Background()

	cfg := buildGuildConfig("g1", identityhistory.Config{
		TrackNickname: true,
	})

	ev := newMemberUpdate("g1", "u1", "NewNick", "")
	require.NoError(t, mod.HandleMemberUpdate(ctx, nil, ev, cfg))

	require.Len(t, repo.Records, 1)
	assert.Equal(t, identityhistory.FieldNickname, repo.Records[0].Field)
	assert.Equal(t, "", repo.Records[0].OldValue)
	assert.Equal(t, "NewNick", repo.Records[0].NewValue)
	assert.Equal(t, "GUILD_MEMBER_UPDATE", repo.Records[0].SourceEvent)
}

// TestHandleMemberUpdate_Idempotent vérifie qu'une valeur identique ne génère pas d'entrée.
func TestHandleMemberUpdate_Idempotent(t *testing.T) {
	repo := identityhistory.NewMemoryIdentityRepo()
	mod := identityhistory.New(repo)
	ctx := context.Background()

	cfg := buildGuildConfig("g1", identityhistory.Config{
		TrackNickname: true,
	})

	ev := newMemberUpdate("g1", "u1", "SameNick", "")
	// Premier appel : enregistre la valeur initiale.
	require.NoError(t, mod.HandleMemberUpdate(ctx, nil, ev, cfg))
	assert.Len(t, repo.Records, 1)

	// Deuxième appel avec la même valeur : aucun nouvel enregistrement.
	require.NoError(t, mod.HandleMemberUpdate(ctx, nil, ev, cfg))
	assert.Len(t, repo.Records, 1, "la valeur identique ne doit pas créer de doublon")
}

// TestHandleMemberUpdate_MultipleFields vérifie que plusieurs champs activés
// génèrent chacun un enregistrement si leur valeur change.
func TestHandleMemberUpdate_MultipleFields(t *testing.T) {
	repo := identityhistory.NewMemoryIdentityRepo()
	mod := identityhistory.New(repo)
	ctx := context.Background()

	cfg := buildGuildConfig("g1", identityhistory.Config{
		TrackNickname:    true,
		TrackGuildAvatar: true,
	})

	ev := newMemberUpdate("g1", "u1", "Nick1", "avatar_hash_1")
	require.NoError(t, mod.HandleMemberUpdate(ctx, nil, ev, cfg))
	assert.Len(t, repo.Records, 2)
}

// TestHandleUserUpdate_UsernameChange vérifie qu'un changement de username est enregistré.
func TestHandleUserUpdate_UsernameChange(t *testing.T) {
	repo := identityhistory.NewMemoryIdentityRepo()
	mod := identityhistory.New(repo)
	ctx := context.Background()

	cfg := buildGuildConfig("g1", identityhistory.Config{
		TrackUsername: true,
	})

	ev := &discordgo.UserUpdate{
		User: &discordgo.User{
			ID:       "u1",
			Username: "NewUser",
		},
	}
	require.NoError(t, mod.HandleUserUpdate(ctx, nil, ev, "g1", cfg))

	require.Len(t, repo.Records, 1)
	assert.Equal(t, identityhistory.FieldUsername, repo.Records[0].Field)
	assert.Equal(t, "NewUser", repo.Records[0].NewValue)
	assert.Equal(t, "USER_UPDATE", repo.Records[0].SourceEvent)
}

// TestHandleUserUpdate_Discriminator vérifie le format username#discriminator
// pour les comptes Discord legacy.
func TestHandleUserUpdate_Discriminator(t *testing.T) {
	repo := identityhistory.NewMemoryIdentityRepo()
	mod := identityhistory.New(repo)
	ctx := context.Background()

	cfg := buildGuildConfig("g1", identityhistory.Config{
		TrackUsername: true,
	})

	ev := &discordgo.UserUpdate{
		User: &discordgo.User{
			ID:            "u1",
			Username:      "oldname",
			Discriminator: "1234",
		},
	}
	require.NoError(t, mod.HandleUserUpdate(ctx, nil, ev, "g1", cfg))

	require.Len(t, repo.Records, 1)
	assert.Equal(t, "oldname#1234", repo.Records[0].NewValue)
}
