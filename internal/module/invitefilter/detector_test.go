package invitefilter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vignemail1/discord-bot/internal/module/invitefilter"
)

func TestExtractInviteCodes(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		codes  []string
	}{
		{"discord.gg", "rejoins discord.gg/abc123", []string{"abc123"}},
		{"discord.com/invite", "https://discord.com/invite/xYz-99", []string{"xYz-99"}},
		{"discordapp.com/invite", "http://discordapp.com/invite/CODE", []string{"CODE"}},
		{"dis.gd", "dis.gd/shortcode", []string{"shortcode"}},
		{"avec http", "https://discord.gg/aaa et discord.gg/bbb", []string{"aaa", "bbb"}},
		{"aucun lien", "message normal sans lien", nil},
		{"majuscules", "DISCORD.GG/TEST", []string{"TEST"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := invitefilter.ExtractInviteCodes(tc.input)
			if len(tc.codes) == 0 {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tc.codes, got)
			}
		})
	}
}

func TestContainsInvite(t *testing.T) {
	assert.True(t, invitefilter.ContainsInvite("rejoins discord.gg/abc"))
	assert.False(t, invitefilter.ContainsInvite("message propre"))
}

func TestIsAllowedCode(t *testing.T) {
	allowed := []string{"monserveur", "OFFICIAL"}
	assert.True(t, invitefilter.IsAllowedCode("monserveur", allowed))
	assert.True(t, invitefilter.IsAllowedCode("official", allowed)) // case-insensitive
	assert.False(t, invitefilter.IsAllowedCode("random", allowed))
}
