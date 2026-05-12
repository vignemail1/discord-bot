package bot_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/bot"
	"github.com/vignemail1/discord-bot/internal/repository"
	"github.com/vignemail1/discord-bot/internal/repository/mock"
)

func TestHandleGuildCreate_Upsert(t *testing.T) {
	gr := mock.NewGuildRepository()
	h := bot.NewHandler(gr)

	s, err := bot.New("fake-token", h)
	require.NoError(t, err)

	gc := &discordgo.GuildCreate{
		Guild: &discordgo.Guild{
			ID:      "111",
			Name:    "Test Server",
			OwnerID: "222",
		},
	}

	h.HandleGuildCreate(s.DG, gc)

	g, err := gr.Get(context.Background(), "111")
	require.NoError(t, err)
	require.NotNil(t, g)
	assert.Equal(t, "Test Server", g.GuildName)
	assert.True(t, g.Active)
}

func TestHandleGuildCreate_UpsertError_NoPanic(t *testing.T) {
	gr := mock.NewGuildRepository()
	gr.UpsertErr = errors.New("db down")

	h := bot.NewHandler(gr)
	s, err := bot.New("fake-token", h)
	require.NoError(t, err)

	gc := &discordgo.GuildCreate{
		Guild: &discordgo.Guild{ID: "111", Name: "Test", OwnerID: "222"},
	}

	assert.NotPanics(t, func() {
		h.HandleGuildCreate(s.DG, gc)
	})
}

func TestHandleGuildDelete_Deactivate(t *testing.T) {
	gr := mock.NewGuildRepository()

	// Pré-insérer la guilde.
	err := gr.Upsert(context.Background(), repository.Guild{
		GuildID:   "111",
		GuildName: "Test Server",
		Active:    true,
	})
	require.NoError(t, err)

	h := bot.NewHandler(gr)
	s, newErr := bot.New("fake-token", h)
	require.NoError(t, newErr)

	gd := &discordgo.GuildDelete{
		Guild: &discordgo.Guild{ID: "111"},
	}

	h.HandleGuildDelete(s.DG, gd)

	g, err := gr.Get(context.Background(), "111")
	require.NoError(t, err)
	require.NotNil(t, g)
	assert.False(t, g.Active)
}

func TestHandleGuildDelete_DeactivateError_NoPanic(t *testing.T) {
	gr := mock.NewGuildRepository()
	gr.DeactivateErr = errors.New("db down")

	h := bot.NewHandler(gr)
	s, err := bot.New("fake-token", h)
	require.NoError(t, err)

	gd := &discordgo.GuildDelete{
		Guild: &discordgo.Guild{ID: "111"},
	}

	assert.NotPanics(t, func() {
		h.HandleGuildDelete(s.DG, gd)
	})
}
