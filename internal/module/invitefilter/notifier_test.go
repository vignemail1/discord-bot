package invitefilter_test

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/module/invitefilter"
)

// fakeSender est un DiscordSender bouchoné qui enregistre les embeds envoyés.
type fakeSender struct {
	sentEmbeds []*discordgo.MessageEmbed
	sendErr    error
}

func (f *fakeSender) ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	if f.sendErr != nil {
		return nil, f.sendErr
	}
	f.sentEmbeds = append(f.sentEmbeds, embed)
	return &discordgo.Message{}, nil
}

func baseMsg(guildID, authorID, channelID, content string) *discordgo.MessageCreate {
	return &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg1",
			GuildID:   guildID,
			ChannelID: channelID,
			Content:   content,
			Author:    &discordgo.User{ID: authorID, Username: "testuser"},
		},
	}
}

func TestNotifyAction_Disabled(t *testing.T) {
	sender := &fakeSender{}
	cfg := invitefilter.Config{} // NotifyChannelID vide
	msg := baseMsg("g1", "u1", "c1", "discord.gg/test")

	invitefilter.NotifyAction(context.Background(), sender, cfg, msg, []string{"test"}, invitefilter.ActionDelete, 1)

	assert.Empty(t, sender.sentEmbeds, "aucun embed ne doit être envoyé si NotifyChannelID est vide")
}

func TestNotifyAction_Delete(t *testing.T) {
	sender := &fakeSender{}
	cfg := invitefilter.Config{NotifyChannelID: "log-chan"}
	msg := baseMsg("g1", "u1", "c1", "discord.gg/bad")

	invitefilter.NotifyAction(context.Background(), sender, cfg, msg, []string{"bad"}, invitefilter.ActionDelete, 1)

	require.Len(t, sender.sentEmbeds, 1)
	embed := sender.sentEmbeds[0]
	assert.Contains(t, embed.Title, "Suppression")
	assert.Equal(t, 0x607D8B, embed.Color)
	// Aucun champ "Contenu original" car NotifyIncludeContent est false.
	for _, f := range embed.Fields {
		assert.NotEqual(t, "Contenu original", f.Name)
	}
}

func TestNotifyAction_Timeout(t *testing.T) {
	sender := &fakeSender{}
	cfg := invitefilter.Config{NotifyChannelID: "log-chan"}
	msg := baseMsg("g1", "u2", "c1", "discord.gg/spam")

	invitefilter.NotifyAction(context.Background(), sender, cfg, msg, []string{"spam"}, invitefilter.ActionTimeout, 2)

	require.Len(t, sender.sentEmbeds, 1)
	assert.Contains(t, sender.sentEmbeds[0].Title, "Timeout")
	assert.Equal(t, 0xF4A124, sender.sentEmbeds[0].Color)
}

func TestNotifyAction_Ban(t *testing.T) {
	sender := &fakeSender{}
	cfg := invitefilter.Config{NotifyChannelID: "log-chan"}
	msg := baseMsg("g1", "u3", "c1", "discord.gg/evil")

	invitefilter.NotifyAction(context.Background(), sender, cfg, msg, []string{"evil"}, invitefilter.ActionBan, 3)

	require.Len(t, sender.sentEmbeds, 1)
	assert.Contains(t, sender.sentEmbeds[0].Title, "Ban")
	assert.Equal(t, 0xE53935, sender.sentEmbeds[0].Color)
}

func TestNotifyAction_IncludeContent(t *testing.T) {
	sender := &fakeSender{}
	cfg := invitefilter.Config{
		NotifyChannelID:      "log-chan",
		NotifyIncludeContent: true,
	}
	msg := baseMsg("g1", "u1", "c1", "rejoins mon serveur discord.gg/spam !")

	invitefilter.NotifyAction(context.Background(), sender, cfg, msg, []string{"spam"}, invitefilter.ActionDelete, 1)

	require.Len(t, sender.sentEmbeds, 1)
	var found bool
	for _, f := range sender.sentEmbeds[0].Fields {
		if f.Name == "Contenu original" {
			found = true
			assert.Contains(t, f.Value, "spam")
		}
	}
	assert.True(t, found, "le champ Contenu original doit être présent")
}

func TestNotifyAction_ContentTruncated(t *testing.T) {
	sender := &fakeSender{}
	cfg := invitefilter.Config{
		NotifyChannelID:      "log-chan",
		NotifyIncludeContent: true,
	}
	// Contenu de 1100 caractères.
	content := "discord.gg/x " + string(make([]byte, 1100))
	msg := baseMsg("g1", "u1", "c1", content)

	invitefilter.NotifyAction(context.Background(), sender, cfg, msg, []string{"x"}, invitefilter.ActionDelete, 1)

	require.Len(t, sender.sentEmbeds, 1)
	for _, f := range sender.sentEmbeds[0].Fields {
		if f.Name == "Contenu original" {
			assert.LessOrEqual(t, len(f.Value), 1030, "embed field ne doit pas dépasser 1024 chars utiles + backticks")
		}
	}
}

func TestNotifyAction_SendErr_NoPropagate(t *testing.T) {
	sender := &fakeSender{sendErr: assert.AnError}
	cfg := invitefilter.Config{NotifyChannelID: "log-chan"}
	msg := baseMsg("g1", "u1", "c1", "discord.gg/x")

	// Ne doit pas paniquer ni retourner d'erreur (signature void).
	assert.NotPanics(t, func() {
		invitefilter.NotifyAction(context.Background(), sender, cfg, msg, []string{"x"}, invitefilter.ActionDelete, 1)
	})
}
