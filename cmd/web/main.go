// cmd/web est le point d'entrée du dashboard HTTP.
// Étape 7 : serveur HTTP complet — OAuth2 Discord, sessions, middleware auth, routes guildes.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vignemail1/discord-bot/internal/config"
	"github.com/vignemail1/discord-bot/internal/db"
	"github.com/vignemail1/discord-bot/internal/repository/mariadb"
	"github.com/vignemail1/discord-bot/internal/web"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config: échec du chargement", "err", err)
		os.Exit(1)
	}

	setLogLevel(cfg.LogLevel)

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

	guildRepo := mariadb.NewGuildRepository(conn)
	moduleRepo := mariadb.NewModuleRepository(conn)

	srv := web.NewServer(cfg, guildRepo, moduleRepo)

	if err := srv.Start(ctx); err != nil {
		slog.Error("web: erreur fatale", "err", err)
		os.Exit(1)
	}
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
