// cmd/bot est le point d'entrée du bot Discord.
// Étape 6 : enregistrement du module identity_history + purger.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vignemail1/discord-bot/internal/bot"
	"github.com/vignemail1/discord-bot/internal/cache"
	"github.com/vignemail1/discord-bot/internal/config"
	"github.com/vignemail1/discord-bot/internal/db"
	"github.com/vignemail1/discord-bot/internal/module"
	"github.com/vignemail1/discord-bot/internal/module/identityhistory"
	"github.com/vignemail1/discord-bot/internal/module/invitefilter"
	"github.com/vignemail1/discord-bot/internal/repository/mariadb"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config: échec", "err", err)
		os.Exit(1)
	}
	setLogLevel(cfg.LogLevel)

	if cfg.DiscordBotToken == "" {
		slog.Error("config: DISCORD_BOT_TOKEN manquant")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	conn, err := db.Connect(ctx, cfg.DSN())
	if err != nil {
		slog.Error("db: connexion échouée", "err", err)
		os.Exit(1)
	}
	defer conn.Close()

	if err = db.RunMigrations(conn, "./migrations"); err != nil {
		slog.Error("migrations: échec", "err", err)
		os.Exit(1)
	}

	guildRepo    := mariadb.NewGuildRepo(conn)
	moduleRepo   := mariadb.NewModuleRepo(conn)
	counterRepo  := invitefilter.NewMariaDBCounterRepo(conn)
	identityRepo := identityhistory.NewMariaDBIdentityRepo(conn)

	configCache := cache.New(moduleRepo, cfg.CacheTTL)
	configCache.Start(ctx)

	// Moteur de modules.
	reg := module.NewRegistry()
	reg.MustRegister(invitefilter.New(counterRepo))
	reg.MustRegister(identityhistory.New(identityRepo))

	disp := module.NewDispatcher(reg, configCache)

	// Purger identity_history : tourne toutes les 24h.
	purger := identityhistory.NewPurger(
		identityRepo,
		func() []string { return configCache.ActiveGuildIDs() },
		func(guildID string) identityhistory.Config {
			cfgCtx := context.Background()
			gc, err := configCache.Get(cfgCtx, guildID)
			if err != nil {
				return identityhistory.Config{}
			}
			var modCfg identityhistory.Config
			_ = gc.ModuleConfig(identityhistory.ModuleName, &modCfg)
			return modCfg
		},
		24*time.Hour,
	)
	purger.Start(ctx)

	handler := bot.NewHandler(guildRepo, moduleRepo, configCache)
	session, err := bot.New(cfg.DiscordBotToken, handler, disp)
	if err != nil {
		slog.Error("bot: création session échouée", "err", err)
		os.Exit(1)
	}

	slog.Info("bot: démarrage", "modules", reg.Names())

	if err = session.Open(ctx); err != nil {
		slog.Error("bot: Gateway err", "err", err)
		os.Exit(1)
	}

	slog.Info("bot: arrêt propre")
}

func setLogLevel(level string) {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l})))
}
