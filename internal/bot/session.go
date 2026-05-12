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

// Session encapsule la connexion discordgo.
type Session struct {
	DG      *discordgo.Session
	handler *Handler
}

// New crée une nouvelle Session bot avec les intents requis.
// Ne connecte pas encore la Gateway.
func New(token string, h *Handler) (*Session, error) {
	if token == "" {
		return nil, fmt.Errorf("bot: token Discord vide")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("bot: création session discordgo: %w", err)
	}

	// Intents explicites.
	// GUILD_MEMBERS et MESSAGE_CONTENT sont "Privileged" et doivent être
	// activés manuellement sur le Discord Developer Portal.
	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildMessages |
		discordgo.IntentMessageContent

	s := &Session{DG: dg, handler: h}

	// Enregistrement des handlers Gateway.
	dg.AddHandler(s.handler.onReady)
	dg.AddHandler(s.handler.onGuildCreate)
	dg.AddHandler(s.handler.onGuildDelete)

	return s, nil
}

// Open ouvre la connexion Gateway et bloque jusqu'à l'annulation du contexte.
// Reconnexion automatique avec backoff exponentiel plafonné à 2 minutes.
func (s *Session) Open(ctx context.Context) error {
	delay := reconnectDelay

	for {
		// Vérifier l'annulation avant chaque tentative.
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := s.DG.Open(); err != nil {
			slog.Error("bot: connexion Gateway échouée", "err", err, "retry_in", delay)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(delay):
				delay = minDuration(delay*2, reconnectMaxDelay)
				continue
			}
		}

		slog.Info("bot: Gateway connectée")
		delay = reconnectDelay // reset backoff après succès

		// Attendre l'arrêt du contexte.
		<-ctx.Done()
		_ = s.DG.Close()
		slog.Info("bot: Gateway fermée proprement")
		return nil
	}
}

// Close ferme la connexion Gateway.
func (s *Session) Close() error {
	return s.DG.Close()
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
