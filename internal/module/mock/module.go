// Package mock fournit un Module factice pour les tests.
package mock

import (
	"context"
	"errors"

	"github.com/bwmarrin/discordgo"

	"github.com/vignemail1/discord-bot/internal/cache"
	"github.com/vignemail1/discord-bot/internal/module"
)

// MockModule est un module factice configurable.
type MockModule struct {
	name    string
	retErr  error
	// OnHandle est appelé à chaque HandleMessage (si non nil).
	OnHandle func()
}

// New crée un MockModule sans erreur.
func New(name string) *MockModule {
	return &MockModule{name: name}
}

// NewWithError crée un MockModule qui retourne toujours une erreur.
func NewWithError(name string) *MockModule {
	return &MockModule{name: name, retErr: errors.New("mock module error")}
}

func (m *MockModule) Name() string { return m.name }

func (m *MockModule) HandleMessage(_ context.Context, _ *discordgo.Session, _ *discordgo.MessageCreate, _ *cache.GuildConfig) error {
	if m.OnHandle != nil {
		m.OnHandle()
	}
	return m.retErr
}

// Vérification statique.
var _ module.Module = (*MockModule)(nil)
