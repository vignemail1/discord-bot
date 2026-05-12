package bot_test

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/bot"
	"github.com/vignemail1/discord-bot/internal/repository/mock"
)

func TestOnGuildCreate_Upsert(t *testing.T) {
	gr := mock.NewGuildRepository()
	h := bot.NewHandler(gr)

	// Appel direct du handler via la session factice.
	s, err := bot.New("fake-token", h)
	require.NoError(t, err)

	gc := &discordgo.GuildCreate{
		Guild: &discordgo.Guild{
			ID:      "111",
			Name:    "Test Server",
			OwnerID: "222",
		},
	}

	// Simulation de l'événement GUILD_CREATE.
	s.DG.State.GuildAdd(gc.Guild)

	// Appel direct à travers l'interface du handler (méthode exportée pour les tests).
	h.HandleGuildCreate(s.DG, gc)

	g, err := gr.Get(nil, "111") //nolint:staticcheck
	require.NoError(t, err)
	require.NotNil(t, g)
	assert.Equal(t, "Test Server", g.GuildName)
	assert.True(t, g.Active)
}

func TestOnGuildCreate_UpsertError(t *testing.T) {
	gr := mock.NewGuildRepository()
	gr.UpsertErr = errors.New("db down")

	h := bot.NewHandler(gr)
	s, err := bot.New("fake-token", h)
	require.NoError(t, err)

	gc := &discordgo.GuildCreate{
		Guild: &discordgo.Guild{ID: "111", Name: "Test", OwnerID: "222"},
	}

	// Ne doit pas paniquer même si le repo échoue.
	assert.NotPanics(t, func() {
		h.HandleGuildCreate(s.DG, gc)
	})
}

func TestOnGuildDelete_Deactivate(t *testing.T) {
	gr := mock.NewGuildRepository()
	// Pré-insérer la guilde.
	_ = gr.Upsert(nil, repository_guild_for_test("111")) //nolint:staticcheck

	h := bot.NewHandler(gr)
	s, err := bot.New("fake-token", h)
	require.NoError(t, err)

	gd := &discordgo.GuildDelete{
		Guild: &discordgo.Guild{ID: "111"},
	}

	h.HandleGuildDelete(s.DG, gd)

	g, err := gr.Get(nil, "111") //nolint:staticcheck
	require.NoError(t, err)
	require.NotNil(t, g)
	assert.False(t, g.Active)
}

// Helper local pour réduire la duplication dans les tests.
func repository_guild_for_test(id string) interface{} {
	return nil // remplacé par l'import repository dans l'implémentation finale
}
