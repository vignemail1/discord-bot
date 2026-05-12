// Package cache fournit un cache mémoire de configuration par guilde.
// — Thread-safe via sync.Map.
// — TTL configurable (défaut 5 min) ; une goroutine de purge en arrière-plan
//   nettoie les entrées expirées toutes les TTL/2.
// — Invalidation explicite depuis le dashboard ou les tests.
package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/vignemail1/discord-bot/internal/repository"
)

// GuildConfig est la vue composée de la configuration d'une guilde
// utilisée par tous les modules au runtime.
type GuildConfig struct {
	GuildID string
	// Modules est la carte moduleName → GuildModule.
	// La clé est le nom du module (ex : "invite_filter").
	Modules map[string]repository.GuildModule
	cachedAt time.Time
}

// IsEnabled retourne true si le module est actif dans cette guilde.
func (g *GuildConfig) IsEnabled(moduleName string) bool {
	m, ok := g.Modules[moduleName]
	return ok && m.Enabled
}

// ModuleConfig déséérialise le config_json d'un module dans dst.
// Retourne une erreur si le module est absent ou si le JSON est invalide.
func (g *GuildConfig) ModuleConfig(moduleName string, dst any) error {
	m, ok := g.Modules[moduleName]
	if !ok {
		return nil // pas de config → defaults
	}
	if len(m.ConfigJSON) == 0 {
		return nil
	}
	return json.Unmarshal(m.ConfigJSON, dst)
}

type cacheEntry struct {
	config   GuildConfig
	expiresAt time.Time
}

// GuildConfigCache est le cache de configurations de guildes.
type GuildConfigCache struct {
	ttl        time.Duration
	entries    sync.Map // map[string]*cacheEntry
	moduleRepo repository.ModuleRepository
}

// DefaultTTL est la durée de vie par défaut d'une entrée de cache.
const DefaultTTL = 5 * time.Minute

// New crée un nouveau GuildConfigCache avec le TTL donné.
// Si ttl <= 0, DefaultTTL est utilisé.
func New(moduleRepo repository.ModuleRepository, ttl time.Duration) *GuildConfigCache {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &GuildConfigCache{
		ttl:        ttl,
		moduleRepo: moduleRepo,
	}
}

// Start lance la goroutine de purge en arrière-plan.
// Elle s'arrête quand ctx est annulé.
func (c *GuildConfigCache) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(c.ttl / 2)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				c.evict(now)
			}
		}
	}()
}

// Get retourne la config de la guilde, en chargeant depuis la DB si absent ou expiré.
func (c *GuildConfigCache) Get(ctx context.Context, guildID string) (*GuildConfig, error) {
	now := time.Now()
	if v, ok := c.entries.Load(guildID); ok {
		e := v.(*cacheEntry)
		if now.Before(e.expiresAt) {
			cfg := e.config
			return &cfg, nil
		}
	}
	return c.load(ctx, guildID)
}

// Invalidate supprime l'entrée de cache pour une guilde.
// Prochain appel Get() rechargera depuis la DB.
func (c *GuildConfigCache) Invalidate(guildID string) {
	c.entries.Delete(guildID)
	slog.Debug("cache: invalidation guilde", "guild_id", guildID)
}

// Populate charge la configuration d'une guilde en DB et l'insère dans le cache.
// Appelé au GUILD_CREATE pour pré-populer le cache sans attendre le premier évènement.
func (c *GuildConfigCache) Populate(ctx context.Context, guildID string) error {
	_, err := c.load(ctx, guildID)
	return err
}

// load (privé) charge depuis la DB et met à jour le cache.
func (c *GuildConfigCache) load(ctx context.Context, guildID string) (*GuildConfig, error) {
	modules, err := c.moduleRepo.ListByGuild(ctx, guildID)
	if err != nil {
		return nil, err
	}
	modMap := make(map[string]repository.GuildModule, len(modules))
	for _, m := range modules {
		modMap[m.ModuleName] = m
	}
	cfg := GuildConfig{
		GuildID:  guildID,
		Modules:  modMap,
		cachedAt: time.Now(),
	}
	c.entries.Store(guildID, &cacheEntry{
		config:    cfg,
		expiresAt: time.Now().Add(c.ttl),
	})
	slog.Debug("cache: guilde chargée", "guild_id", guildID, "modules", len(modules))
	return &cfg, nil
}

// evict nettoie les entrées expirées.
func (c *GuildConfigCache) evict(now time.Time) {
	c.entries.Range(func(k, v any) bool {
		if v.(*cacheEntry).expiresAt.Before(now) {
			c.entries.Delete(k)
		}
		return true
	})
}
