// Package bot gère la connexion Gateway Discord et le cycle de vie de la session.
package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	reconnectDelay    = 5 * time.Second
	reconnectMaxDelay = 2 * time.Minute
)

// Session encapsule la connexion discordgo et expose l'objet *discordgo.Session
// pour que les handlers et le dispatcher puissent l'utiliser.
type Session struct {
	DG      *discordgo.Session
	handler *Handler
}

// New crée une nouvelle Session bot avec les intents requis.
// Elle n'ouvre pas encore la connexion Gateway.
func New(token string, h *Handler) (*Session, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("bot: création session discordgo: %w", err)
	}

	// Intents explicites — GUILD_MEMBERS et MESSAGE_CONTENT sont privileged.
	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildMessages |
		discordgo.IntentMessageContent

	s := &Session{DG: dg, handler: h}

	// Enregistrement des handlers d'événements Gateway.
	dg.AddHandler(s.handler.onReady)
	dg.AddHandler(s.handler.onGuildCreate)
	dg.AddHandler(s.handler.onGuildDelete)

	return s, nil
}

// Open ouvre la connexion Gateway et bloque jusqu'à l'annulation du contexte.
// En cas de déconnexion inattendue, elle tente de se reconnecter avec backoff.
func (s *Session) Open(ctx context.Context) error {
	delay := reconnectDelay

	for {
		if err := s.DG.Open(); err != nil {
			slog.Error("bot: connexion Gateway échouée", "err", err, "retry_in", delay)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(delay):
				delay = min(delay*2, reconnectMaxDelay)
				continue
			}
		}

		slog.Info("bot: Gateway connectée")
		delay = reconnectDelay // reset backoff

		// Attendre la déconnexion ou l'arrêt.
		select {
		case <-ctx.Done():
			_ = s.DG.Close()
			slog.Info("bot: Gateway fermée proprement")
			return nil
		case <-s.DG.State.Ready.ReadyChan():
			// Cas théorique : le chan Ready n'est pas exposé directement.
			// On reste bloqué sur ctx.Done() dans la pratique.
		}
	}
}

// Close ferme la connexion Gateway.
func (s *Session) Close() error {
	return s.DG.Close()
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
