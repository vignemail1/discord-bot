package bot_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vignemail1/discord-bot/internal/bot"
	"github.com/vignemail1/discord-bot/internal/cache"
	"github.com/vignemail1/discord-bot/internal/module"
	"github.com/vignemail1/discord-bot/internal/repository"
	"github.com/vignemail1/discord-bot/internal/repository/mock"
)

func newTestHandler() (*bot.Handler, *module.Dispatcher) {
	guildRepo := mock.NewGuildRepository()
	moduleRepo := mock.NewModuleRepository()
	cc := cache.New(moduleRepo, 0)
	h := bot.NewHandler(guildRepo, moduleRepo, cc)
	disp := module.NewDispatcher(nil, nil)
	return h, disp
}

func TestNew_InvalidToken(t *testing.T) {
	h, disp := newTestHandler()
	// Un token vide doit provoquer une erreur à la création de session.
	_, err := bot.New("", h, disp)
	assert.Error(t, err)
}

func TestNew_ValidTokenFormat(t *testing.T) {
	h, disp := newTestHandler()
	// discordgo accepte n'importe quelle chaîne non vide comme token à la création.
	// La validation réelle se fait à l'ouverture de la Gateway.
	s, err := bot.New("fake-token-for-unit-test", h, disp)
	assert.NoError(t, err)
	assert.NotNil(t, s)
}

// Vérification que GuildRepository est bien l'interface attendue.
var _ repository.GuildRepository = (*mock.GuildRepositoryMock)(nil)
