// cmd/bot est le point d'entrée du bot Discord.
// Étape 1 : bootstrap config + DB + migrations.
// La connexion Gateway Discord sera ajoutée à l'étape 2.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vignemail1/discord-bot/internal/config"
	"github.com/vignemail1/discord-bot/internal/db"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

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

	slog.Info("bot: prêt (connexion Gateway à venir à l'étape 2)")

	// Attendre le signal d'arrêt.
	<-ctx.Done()
	slog.Info("bot: arrêt demandé")
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
