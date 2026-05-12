package web

import (
	"context"
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
	cfg          *config.Config
	sessions     *SessionStore
	guildRepo    repository.GuildRepository
	moduleRepo   repository.ModuleRepository
	auditRepo    repository.AuditRepository
	identityRepo repository.IdentityRepository
	dbPinger     DBPinger
	httpClient   *http.Client
	server       *http.Server
}

// NewServer crée un Server initialisé mais non démarré.
func NewServer(
	cfg *config.Config,
	guildRepo repository.GuildRepository,
	moduleRepo repository.ModuleRepository,
	auditRepo repository.AuditRepository,
	identityRepo repository.IdentityRepository,
	dbPinger DBPinger,
) *Server {
	srv := &Server{
		cfg:          cfg,
		sessions:     NewSessionStore(),
		guildRepo:    guildRepo,
		moduleRepo:   moduleRepo,
		auditRepo:    auditRepo,
		identityRepo: identityRepo,
		dbPinger:     dbPinger,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}

	r := chi.NewRouter()

	// Middlewares globaux.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(slogRequest)
	r.Use(securityHeaders)
	r.Use(metricsMiddleware)
	r.Use(RateLimitMiddleware(20, 50)) // 20 req/s par IP, burst 50

	// Routes publiques (pas de rate limit supplémentaire ni auth).
	r.Get("/healthz", srv.handleHealthz)
	r.Get("/metrics", metricsHandler().ServeHTTP)
	r.Get("/auth/login", srv.handleLogin)
	r.Get("/auth/callback", srv.handleCallback)
	r.Get("/auth/logout", srv.handleLogout)

	// Routes protégées.
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

		// Audit log.
		r.Get("/guilds/{guildID}/audit", srv.handleListAudit)

		// Identity history.
		r.Get("/guilds/{guildID}/identity", srv.handleListIdentity)
		r.Get("/guilds/{guildID}/identity/{userID}", srv.handleGetMemberIdentity)
	})

	srv.server = &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return srv
}

// Start démarre le serveur HTTP et lance le GC des sessions en arrière-plan.
func (srv *Server) Start(ctx context.Context) error {
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.server.Shutdown(shutdownCtx); err != nil {
		slog.Error("web: shutdown incomplet", "err", err)
		return err
	}
	slog.Info("web: arrêt propre")
	return nil
}
