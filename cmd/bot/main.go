// cmd/bot est le point d'entrée du bot Discord.
// Étape 4 : moteur de modules (Registry + Dispatcher).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vignemail1/discord-bot/internal/bot"
	"github.com/vignemail1/discord-bot/internal/cache"
	"github.com/vignemail1/discord-bot/internal/config"
	"github.com/vignemail1/discord-bot/internal/db"
	"github.com/vignemail1/discord-bot/internal/module"
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

	// Connexion DB + migrations.
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

	// Repositories.
	guildRepo  := mariadb.NewGuildRepo(conn)
	moduleRepo := mariadb.NewModuleRepo(conn)

	// Cache.
	configCache := cache.New(moduleRepo, cfg.CacheTTL)
	configCache.Start(ctx)

	// Moteur de modules.
	// Les modules concrets (invite_filter, …) seront enregistrés ici au step 5+.
	reg  := module.NewRegistry()
	disp := module.NewDispatcher(reg, configCache)

	// Handler Gateway + session.
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
