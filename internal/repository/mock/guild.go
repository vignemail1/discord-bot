// Package mock fournit des implémentations en mémoire des repositories pour les tests.
package mock

import (
	"context"
	"sync"

	"github.com/vignemail1/discord-bot/internal/repository"
)

// GuildRepositoryMock est une implémentation en mémoire thread-safe de repository.GuildRepository.
type GuildRepositoryMock struct {
	mu     sync.RWMutex
	guilds map[string]repository.Guild

	// Injections d'erreurs pour les tests.
	UpsertErr    error
	DeactivateErr error
	GetErr       error
	ListActiveErr error
}

// NewGuildRepository crée un mock vide.
func NewGuildRepository() *GuildRepositoryMock {
	return &GuildRepositoryMock{
		guilds: make(map[string]repository.Guild),
	}
}

func (m *GuildRepositoryMock) Upsert(ctx context.Context, g repository.Guild) error {
	if m.UpsertErr != nil {
		return m.UpsertErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guilds[g.GuildID] = g
	return nil
}

func (m *GuildRepositoryMock) Deactivate(ctx context.Context, guildID string) error {
	if m.DeactivateErr != nil {
		return m.DeactivateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if g, ok := m.guilds[guildID]; ok {
		g.Active = false
		m.guilds[guildID] = g
	}
	return nil
}

func (m *GuildRepositoryMock) Get(ctx context.Context, guildID string) (*repository.Guild, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.guilds[guildID]
	if !ok {
		return nil, nil
	}
	copy := g
	return &copy, nil
}

func (m *GuildRepositoryMock) ListActive(ctx context.Context) ([]repository.Guild, error) {
	if m.ListActiveErr != nil {
		return nil, m.ListActiveErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []repository.Guild
	for _, g := range m.guilds {
		if g.Active {
			out = append(out, g)
		}
	}
	return out, nil
}

// Vérification statique : GuildRepositoryMock implémente repository.GuildRepository.
var _ repository.GuildRepository = (*GuildRepositoryMock)(nil)
