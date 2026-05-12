package cache_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/cache"
	"github.com/vignemail1/discord-bot/internal/repository"
	"github.com/vignemail1/discord-bot/internal/repository/mock"
)

func setup() (*mock.ModuleRepositoryMock, *cache.GuildConfigCache) {
	mockRepo := mock.NewModuleRepository()
	c := cache.New(mockRepo, 100*time.Millisecond)
	return mockRepo, c
}

func TestCache_Get_EmptyGuild(t *testing.T) {
	_, c := setup()
	cfg, err := c.Get(context.Background(), "guild1")
	require.NoError(t, err)
	assert.Equal(t, "guild1", cfg.GuildID)
	assert.Empty(t, cfg.Modules)
}

func TestCache_Get_WithModules(t *testing.T) {
	mockRepo, c := setup()
	_ = mockRepo.Upsert(context.Background(), repository.GuildModule{
		GuildID: "g1", ModuleName: "invite_filter",
		Enabled: true, ConfigJSON: json.RawMessage(`{"key":"val"}`),
	})

	cfg, err := c.Get(context.Background(), "g1")
	require.NoError(t, err)
	assert.True(t, cfg.IsEnabled("invite_filter"))
	assert.False(t, cfg.IsEnabled("other_module"))
}

func TestCache_Get_Cached(t *testing.T) {
	mockRepo, c := setup()
	// Premier appel : charge depuis la DB mock
	_, err := c.Get(context.Background(), "g2")
	require.NoError(t, err)
	// Inject une erreur : le deuxième appel ne doit PAS aller en DB (cache valide)
	mockRepo.ListErr = errors.New("db down")
	_, err = c.Get(context.Background(), "g2")
	require.NoError(t, err, "should have served from cache")
}

func TestCache_Invalidate(t *testing.T) {
	mockRepo, c := setup()
	_, _ = c.Get(context.Background(), "g3")

	c.Invalidate("g3")
	// Après invalidation + erreur DB : le rechargement doit échouer
	mockRepo.ListErr = errors.New("db down")
	_, err := c.Get(context.Background(), "g3")
	require.Error(t, err, "should have tried DB after invalidation")
}

func TestCache_TTL_Expiry(t *testing.T) {
	mockRepo, c := setup() // TTL = 100ms
	_, err := c.Get(context.Background(), "g4")
	require.NoError(t, err)

	time.Sleep(150 * time.Millisecond)
	// Injecter un module après l'expiration : il doit être visible
	_ = mockRepo.Upsert(context.Background(), repository.GuildModule{
		GuildID: "g4", ModuleName: "new_module", Enabled: true,
	})
	cfg, err := c.Get(context.Background(), "g4")
	require.NoError(t, err)
	assert.True(t, cfg.IsEnabled("new_module"), "new module should be visible after TTL expiry")
}

func TestCache_ModuleConfig_Unmarshal(t *testing.T) {
	mockRepo, c := setup()
	_ = mockRepo.Upsert(context.Background(), repository.GuildModule{
		GuildID: "g5", ModuleName: "invite_filter",
		Enabled: true, ConfigJSON: json.RawMessage(`{"timeout_hours":24}`),
	})
	cfg, err := c.Get(context.Background(), "g5")
	require.NoError(t, err)

	var dst struct {
		TimeoutHours int `json:"timeout_hours"`
	}
	require.NoError(t, cfg.ModuleConfig("invite_filter", &dst))
	assert.Equal(t, 24, dst.TimeoutHours)
}

func TestCache_Populate(t *testing.T) {
	mockRepo, c := setup()
	_ = mockRepo.Upsert(context.Background(), repository.GuildModule{
		GuildID: "g6", ModuleName: "invite_filter", Enabled: true,
	})
	err := c.Populate(context.Background(), "g6")
	require.NoError(t, err)

	// Après Populate, Get doit servir depuis le cache (pas de recharge)
	mockRepo.ListErr = errors.New("db down")
	cfg, err := c.Get(context.Background(), "g6")
	require.NoError(t, err)
	assert.True(t, cfg.IsEnabled("invite_filter"))
}
