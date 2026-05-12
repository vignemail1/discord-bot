package identityhistory_test

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/cache"
	"github.com/vignemail1/discord-bot/internal/module/identityhistory"
)

// buildCacheWithCfg construit un GuildConfig directement en mémoire pour les tests.
func buildGuildConfig(guildID string, cfg identityhistory.Config) *cache.GuildConfig {
	import_json, _ := marshalConfig(cfg)
	mod := fakeGuildModule(guildID, identityhistory.ModuleName, true, import_json)
	return &cache.GuildConfig{
		GuildID: guildID,
		Modules: map[string]interface{}{
			identityhistory.ModuleName: mod,
		},
	}
}

func newMemberUpdate(guildID, userID, username, nick, avatar string) *discordgo.GuildMemberUpdate {
	return &discordgo.GuildMemberUpdate{
		GuildMember: &discordgo.Member{
			GuildID: guildID,
			Nick:    nick,
			Avatar:  avatar,
			User: &discordgo.User{
				ID:       userID,
				Username: username,
			},
		},
	}
}
