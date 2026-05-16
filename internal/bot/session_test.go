package bot_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vignemail1/discord-bot/internal/bot"
	"github.com/vignemail1/discord-bot/internal/cache"
	"github.com/vignemail1/discord-bot/internal/repository"
	"github.com/vignemail1/discord-bot/internal/repository/mock"
)

func TestNew_InvalidToken(t *testing.T) {
	guildRepo := mock.NewGuildRepository()
	moduleRepo := mock.NewModuleRepository()
	cc := cache.New(moduleRepo, 0)
	h := bot.NewHandler(guildRepo, moduleRepo, cc)
	// Un token vide doit provoquer une erreur à la création de session.
	_, err := bot.New("", h)
	assert.Error(t, err)
}

func TestNew_ValidTokenFormat(t *testing.T) {
	guildRepo := mock.NewGuildRepository()
	moduleRepo := mock.NewModuleRepository()
	cc := cache.New(moduleRepo, 0)
	h := bot.NewHandler(guildRepo, moduleRepo, cc)
	// discordgo accepte n'importe quelle chaîne non vide comme token à la création.
	// La validation réelle se fait à l'ouverture de la Gateway.
	s, err := bot.New("fake-token-for-unit-test", h)
	assert.NoError(t, err)
	assert.NotNil(t, s)
}

// Vérification que GuildRepository est bien l'interface attendue.
var _ repository.GuildRepository = (*mock.GuildRepositoryMock)(nil)
