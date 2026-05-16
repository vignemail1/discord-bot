package mock

import (
	"context"
	"fmt"
	"sync"

	"github.com/vignemail1/discord-bot/internal/repository"
)

// GuildRepositoryMock est l'implémentation en mémoire de repository.GuildRepository.
type GuildRepositoryMock struct {
	mu            sync.RWMutex
	guilds        map[string]repository.Guild
	UpsertErr     error
	DeactivateErr error
	GetErr        error
	ListErr       error
}

func NewGuildRepository() *GuildRepositoryMock {
	return &GuildRepositoryMock{
		guilds: make(map[string]repository.Guild),
	}
}

// NewGuild est un helper de test.
func NewGuild(id, name string) repository.Guild {
	return repository.Guild{GuildID: id, GuildName: name, Active: true}
}

// NewModule est un helper de test utilisé par les tests cross-packages.
func NewModule(guildID, name string, enabled bool) repository.GuildModule {
	return repository.GuildModule{GuildID: guildID, ModuleName: name, Enabled: enabled}
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
		return nil, fmt.Errorf("mock: guilde %s introuvable", guildID)
	}
	copy := g
	return &copy, nil
}

func (m *GuildRepositoryMock) ListActive(ctx context.Context) ([]repository.Guild, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
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

var _ repository.GuildRepository = (*GuildRepositoryMock)(nil)
