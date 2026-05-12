package invitefilter

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// DiscordSender est l'interface minimale de discordgo.Session utilisée par le notifier.
// Elle facilite le bouchonnage dans les tests.
type DiscordSender interface {
	ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed) (*discordgo.Message, error)
}

// actionLabel retourne un libellé lisible et une couleur Discord pour chaque ActionKind.
func actionLabel(a ActionKind) (label, emoji string, color int) {
	switch a {
	case ActionTimeout:
		return "Timeout", "⏱️", 0xF4A124 // orange
	case ActionBan:
		return "Ban", "🔨", 0xE53935 // rouge
	default: // ActionDelete
		return "Suppression", "🗑️", 0x607D8B // gris bleu
	}
}

// NotifyAction publie un embed de notification dans le salon configuré.
// Les erreurs d'envoi sont loguées mais ne font pas remonter d'erreur :
// la notification est best-effort et ne doit jamais bloquer la sanction.
func NotifyAction(
	ctx context.Context,
	s DiscordSender,
	cfg Config,
	msg *discordgo.MessageCreate,
	codes []string,
	action ActionKind,
	count int,
) {
	if cfg.NotifyChannelID == "" {
		return
	}

	label, emoji, color := actionLabel(action)

	title := fmt.Sprintf("%s %s — %s", emoji, label, msg.Author.Username)

	fields := []*discordgo.MessageEmbedField{
		{Name: "Utilisateur", Value: fmt.Sprintf("<@%s> (`%s`)", msg.Author.ID, msg.Author.ID), Inline: true},
		{Name: "Salon", Value: fmt.Sprintf("<#%s>", msg.ChannelID), Inline: true},
		{Name: "Infraction n°", Value: fmt.Sprintf("%d", count), Inline: true},
		{Name: "Codes détectés", Value: "`" + strings.Join(codes, "`, `") + "`", Inline: false},
	}

	if cfg.NotifyIncludeContent {
		content := msg.Content
		if len(content) > 1024 {
			content = content[:1021] + "..."
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Contenu original",
			Value: fmt.Sprintf("```\n%s\n```", content),
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Color:       color,
		Fields:      fields,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "invite_filter"},
	}

	if _, err := s.ChannelMessageSendEmbed(cfg.NotifyChannelID, embed); err != nil {
		slog.Error("invite_filter: notification salon échouée",
			"channel_id", cfg.NotifyChannelID,
			"guild_id", msg.GuildID,
			"user_id", msg.Author.ID,
			"err", err,
		)
	}
}
