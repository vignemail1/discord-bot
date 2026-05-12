package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/vignemail1/discord-bot/internal/repository"
)

type moduleKey struct{ guildID, moduleName string }

// ModuleRepositoryMock est une implémentation en mémoire de repository.ModuleRepository.
type ModuleRepositoryMock struct {
	mu      sync.RWMutex
	modules map[moduleKey]repository.GuildModule

	GetErr        error
	ListErr       error
	UpsertErr     error
	SetEnabledErr error
	UpdateCfgErr  error
	DeleteErr     error
}

func NewModuleRepository() *ModuleRepositoryMock {
	return &ModuleRepositoryMock{
		modules: make(map[moduleKey]repository.GuildModule),
	}
}

func (m *ModuleRepositoryMock) Get(ctx context.Context, guildID, moduleName string) (*repository.GuildModule, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.modules[moduleKey{guildID, moduleName}]
	if !ok {
		return nil, nil
	}
	copy := v
	return &copy, nil
}

func (m *ModuleRepositoryMock) ListByGuild(ctx context.Context, guildID string) ([]repository.GuildModule, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []repository.GuildModule
	for k, v := range m.modules {
		if k.guildID == guildID {
			out = append(out, v)
		}
	}
	return out, nil
}

func (m *ModuleRepositoryMock) Upsert(ctx context.Context, mod repository.GuildModule) error {
	if m.UpsertErr != nil {
		return m.UpsertErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(mod.ConfigJSON) == 0 {
		mod.ConfigJSON = json.RawMessage(`{}`)
	}
	m.modules[moduleKey{mod.GuildID, mod.ModuleName}] = mod
	return nil
}

func (m *ModuleRepositoryMock) SetEnabled(ctx context.Context, guildID, moduleName string, enabled bool) error {
	if m.SetEnabledErr != nil {
		return m.SetEnabledErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	k := moduleKey{guildID, moduleName}
	v, ok := m.modules[k]
	if !ok {
		return fmt.Errorf("mock: module %s/%s introuvable", guildID, moduleName)
	}
	v.Enabled = enabled
	m.modules[k] = v
	return nil
}

func (m *ModuleRepositoryMock) UpdateConfig(ctx context.Context, guildID, moduleName string, config json.RawMessage) error {
	if m.UpdateCfgErr != nil {
		return m.UpdateCfgErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	k := moduleKey{guildID, moduleName}
	v, ok := m.modules[k]
	if !ok {
		return fmt.Errorf("mock: module %s/%s introuvable", guildID, moduleName)
	}
	v.ConfigJSON = config
	m.modules[k] = v
	return nil
}

func (m *ModuleRepositoryMock) Delete(ctx context.Context, guildID, moduleName string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.modules, moduleKey{guildID, moduleName})
	return nil
}

var _ repository.ModuleRepository = (*ModuleRepositoryMock)(nil)
