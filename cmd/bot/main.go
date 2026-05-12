// cmd/bot est le point d'entrée du bot Discord.
// Étape 3 : cache de configuration par guilde (sync.Map + TTL + goroutine de purge).
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

	// Wiring des dépendances.
	guildRepo := mariadb.NewGuildRepo(conn)
	moduleRepo := mariadb.NewModuleRepo(conn)

	configCache := cache.New(moduleRepo, cfg.CacheTTL)
	configCache.Start(ctx) // goroutine de purge des entrées expirées

	handler := bot.NewHandler(guildRepo, moduleRepo, configCache)

	session, err := bot.New(cfg.DiscordBotToken, handler)
	if err != nil {
		slog.Error("bot: création session échouée", "err", err)
		os.Exit(1)
	}

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
