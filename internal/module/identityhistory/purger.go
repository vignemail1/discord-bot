package identityhistory

import (
	"context"
	"log/slog"
	"time"
)

// Purger est une goroutine périodique qui supprime les enregistrements
// d'identité antérieurs au RetentionDays configuré pour chaque guilde.
type Purger struct {
	repo     IdentityRepository
	guilds   func() []string // injecte la liste des guild_id actifs
	cfgFn    func(guildID string) Config // retourne la config du module pour une guilde
	interval time.Duration
}

// NewPurger crée un nouveau Purger.
// - guilds : fonction retournant la liste des guild_id actifs.
// - cfgFn : fonction retournant la Config du module pour un guild_id.
// - interval : durée entre deux passes de purge (typiquement 24h en prod).
func NewPurger(
	repo IdentityRepository,
	guilds func() []string,
	cfgFn func(string) Config,
	interval time.Duration,
) *Purger {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &Purger{
		repo:     repo,
		guilds:   guilds,
		cfgFn:    cfgFn,
		interval: interval,
	}
}

// Start lance la goroutine de purge. Elle s'arrête quand ctx est annulé.
func (p *Purger) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.run(ctx)
			}
		}
	}()
}

func (p *Purger) run(ctx context.Context) {
	for _, guildID := range p.guilds() {
		cfg := p.cfgFn(guildID)
		cfg.defaults()
		if cfg.RetentionDays <= 0 {
			continue
		}
		before := time.Now().AddDate(0, 0, -cfg.RetentionDays)
		n, err := p.repo.Purge(ctx, guildID, before)
		if err != nil {
			slog.Warn("identity_history.purger: purge échouée",
				"guild_id", guildID, "err", err)
			continue
		}
		if n > 0 {
			slog.Info("identity_history.purger: enregistrements supprimés",
				"guild_id", guildID, "count", n)
		}
	}
}
