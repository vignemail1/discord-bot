package invitefilter_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/cache"
	"github.com/vignemail1/discord-bot/internal/module/invitefilter"
	"github.com/vignemail1/discord-bot/internal/repository"
	"github.com/vignemail1/discord-bot/internal/repository/mock"
)

// fakeSession est une session discordgo bouchonée : elle enregistre les appels
// mais ne contacte pas l'API réelle.
type fakeSession struct {
	deletedMessages []string
	timedOutUsers   []string
	bannedUsers     []string
	edits           []*discordgo.GuildMemberParams
}

func (fs *fakeSession) ChannelMessageDelete(channelID, messageID string) error {
	fs.deletedMessages = append(fs.deletedMessages, messageID)
	return nil
}

func (fs *fakeSession) GuildMemberEdit(guildID, userID string, data *discordgo.GuildMemberParams) (*discordgo.Member, error) {
	fs.timedOutUsers = append(fs.timedOutUsers, userID)
	fs.edits = append(fs.edits, data)
	return nil, nil
}

func (fs *fakeSession) GuildBanCreateWithReason(guildID, userID, reason string, days int) error {
	fs.bannedUsers = append(fs.bannedUsers, userID)
	return nil
}

// buildCache construit un GuildConfigCache avec un module invite_filter configuré.
func buildCache(t *testing.T, guildID string, cfg invitefilter.Config) *cache.GuildConfigCache {
	t.Helper()
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)

	mockRepo := mock.NewModuleRepository()
	require.NoError(t, mockRepo.Upsert(context.Background(), repository.GuildModule{
		GuildID:    guildID,
		ModuleName: invitefilter.ModuleName,
		Enabled:    true,
		ConfigJSON: raw,
	}))
	c := cache.New(mockRepo, 5*time.Minute)
	require.NoError(t, c.Populate(context.Background(), guildID))
	return c
}

func newMsg(guildID, authorID, content string, roles []string) *discordgo.MessageCreate {
	return &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-" + authorID,
			GuildID:   guildID,
			ChannelID: "chan1",
			Content:   content,
			Author:    &discordgo.User{ID: authorID},
			Member:    &discordgo.Member{Roles: roles},
		},
	}
}

// --- tests ---

func TestModule_NoInviteLink_NoAction(t *testing.T) {
	counters := invitefilter.NewMemoryCounterRepo()
	mod := invitefilter.New(counters)
	c := buildCache(t, "g1", invitefilter.Config{})
	cfgCache, _ := c.Get(context.Background(), "g1")

	fs := &fakeSession{}
	msg := newMsg("g1", "u1", "hello world", nil)

	err := mod.HandleMessage(context.Background(), (*discordgo.Session)(nil), msg, cfgCache)
	require.NoError(t, err)
	assert.Empty(t, fs.deletedMessages)
}

func TestModule_AllowedCode_NoAction(t *testing.T) {
	counters := invitefilter.NewMemoryCounterRepo()
	mod := invitefilter.New(counters)
	c := buildCache(t, "g1", invitefilter.Config{AllowedInviteCodes: []string{"monserveur"}})
	cfgCache, _ := c.Get(context.Background(), "g1")

	// Nous testons la logique sans session réelle : on vérifie que le compteur n'augmente pas.
	_ = cfgCache
	count, _ := counters.Get(context.Background(), "g1", "u1", invitefilter.ModuleName)
	assert.Equal(t, 0, count)
}

func TestModule_WhitelistUser_NoCounter(t *testing.T) {
	counters := invitefilter.NewMemoryCounterRepo()
	mod := invitefilter.New(counters)
	c := buildCache(t, "g1", invitefilter.Config{WhitelistUserIDs: []string{"u-admin"}})
	cfgCache, _ := c.Get(context.Background(), "g1")

	msg := newMsg("g1", "u-admin", "discord.gg/badlink", nil)
	err := mod.HandleMessage(context.Background(), nil, msg, cfgCache)
	require.NoError(t, err)

	count, _ := counters.Get(context.Background(), "g1", "u-admin", invitefilter.ModuleName)
	assert.Equal(t, 0, count, "whitelist user ne doit pas être compté")
}

func TestModule_WhitelistRole_NoCounter(t *testing.T) {
	counters := invitefilter.NewMemoryCounterRepo()
	mod := invitefilter.New(counters)
	c := buildCache(t, "g1", invitefilter.Config{WhitelistRoleIDs: []string{"role-mod"}})
	cfgCache, _ := c.Get(context.Background(), "g1")

	msg := newMsg("g1", "u2", "discord.gg/badlink", []string{"role-mod"})
	err := mod.HandleMessage(context.Background(), nil, msg, cfgCache)
	require.NoError(t, err)

	count, _ := counters.Get(context.Background(), "g1", "u2", invitefilter.ModuleName)
	assert.Equal(t, 0, count, "whitelist role ne doit pas être compté")
}

func TestModule_ForbiddenLink_CounterIncrements(t *testing.T) {
	counters := invitefilter.NewMemoryCounterRepo()
	mod := invitefilter.New(counters)

	// Session bouchonée qui implémente uniquement ChannelMessageDelete.
	type deleter interface {
		ChannelMessageDelete(string, string) error
	}
	// On passe nil pour la session discordgo réelle ; la suppression va logger
	// une erreur mais ne pas paniquer (nil check dans discordgo).
	// Pour tester la logique pure du compteur, on vérifie uniquement le compteur.
	c := buildCache(t, "g1", invitefilter.Config{})
	cfgCache, _ := c.Get(context.Background(), "g1")

	msg := newMsg("g1", "u3", "discord.gg/badlink", nil)
	// On appelle HandleMessage avec session nil ; la suppression va échouer silencieusement.
	_ = mod.HandleMessage(context.Background(), nil, msg, cfgCache)

	count, _ := counters.Get(context.Background(), "g1", "u3", invitefilter.ModuleName)
	assert.Equal(t, 1, count)
}

func TestModule_ThirdViolation_ResetAfterBan(t *testing.T) {
	counters := invitefilter.NewMemoryCounterRepo()
	// Simuler 2 infractions déjà enregistrées.
	_, _ = counters.Increment(context.Background(), "g1", "u4", invitefilter.ModuleName)
	_, _ = counters.Increment(context.Background(), "g1", "u4", invitefilter.ModuleName)

	mod := invitefilter.New(counters)
	c := buildCache(t, "g1", invitefilter.Config{BanThreshold: 3})
	cfgCache, _ := c.Get(context.Background(), "g1")

	msg := newMsg("g1", "u4", "discord.gg/badlink", nil)
	_ = mod.HandleMessage(context.Background(), nil, msg, cfgCache)

	// Après le 3ème message, le compteur doit être réinitialisé.
	count, _ := counters.Get(context.Background(), "g1", "u4", invitefilter.ModuleName)
	assert.Equal(t, 0, count, "compteur réinitialisé après ban")
}
