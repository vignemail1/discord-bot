package web

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/vignemail1/discord-bot/internal/config"
	"github.com/vignemail1/discord-bot/internal/repository"
)

// Server est le serveur HTTP du dashboard.
type Server struct {
	cfg        *config.Config
	sessions   *SessionStore
	guildRepo  repository.GuildRepository
	moduleRepo repository.ModuleRepository
	httpClient *http.Client
	server     *http.Server
}

// NewServer crée un Server initialisé mais non démarré.
func NewServer(
	cfg *config.Config,
	guildRepo repository.GuildRepository,
	moduleRepo repository.ModuleRepository,
) *Server {
	srv := &Server{
		cfg:        cfg,
		sessions:   NewSessionStore(),
		guildRepo:  guildRepo,
		moduleRepo: moduleRepo,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(slogRequest)

	// Routes publiques.
	r.Get("/healthz", srv.handleHealthz)
	r.Get("/auth/login", srv.handleLogin)
	r.Get("/auth/callback", srv.handleCallback)
	r.Get("/auth/logout", srv.handleLogout)

	// Routes protégées (session authentifiée obligatoire).
	r.Group(func(r chi.Router) {
		r.Use(srv.requireAuth)

		// Guildes.
		r.Get("/guilds", srv.handleListGuilds)
		r.Get("/guilds/{guildID}", srv.handleGetGuild)
		r.Post("/guilds/{guildID}/install", srv.handleInstallBot)

		// Modules.
		r.Get("/guilds/{guildID}/modules", srv.handleListModules)
		r.Put("/guilds/{guildID}/modules/{moduleName}", srv.handleSetModuleEnabled)
		r.Put("/guilds/{guildID}/modules/{moduleName}/config", srv.handleUpdateModuleConfig)
	})

	srv.server = &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return srv
}

// Start démarre le serveur HTTP et lance le GC des sessions en arrière-plan.
// Bloque jusqu'à l'annulation du contexte puis effectue un graceful shutdown.
func (srv *Server) Start(ctx context.Context) error {
	// Goroutine de nettoyage des sessions expirées.
	go func() {
		ticker := time.NewTicker(sessionGCInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				srv.sessions.GC()
			}
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("web: écoute", "addr", srv.server.Addr)
		if err := srv.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	slog.Info("web: arrêt demandé, drain en cours")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.server.Shutdown(shutdownCtx); err != nil {
		slog.Error("web: shutdown incomplet", "err", err)
		return err
	}
	slog.Info("web: arrêt propre")
	return nil
}

// handleHealthz retourne 200 OK si la DB est joignable.
func (srv *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
