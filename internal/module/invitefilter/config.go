// Package invitefilter implémente le module de filtrage des liens d'invitation Discord.
package invitefilter

// Config est la configuration du module invite_filter désérialisée depuis config_json.
type Config struct {
	// AllowedGuildIDs est la liste des guild_id dont les invitations sont autorisées.
	// Si vide, seule la guilde courante est autorisée.
	AllowedGuildIDs []string `json:"allowed_guild_ids"`

	// AllowedInviteCodes liste des codes d'invitation explicitement autorisés
	// (ex : code permanent de la guilde gérée).
	AllowedInviteCodes []string `json:"allowed_invite_codes"`

	// WhitelistRoleIDs : rôles exempts de tout filtrage et de tout compteur.
	WhitelistRoleIDs []string `json:"whitelist_role_ids"`

	// WhitelistUserIDs : utilisateurs exempts de tout filtrage et de tout compteur.
	WhitelistUserIDs []string `json:"whitelist_user_ids"`

	// TimeoutDuration est la durée du timeout au 2ème message (format Go : "24h").
	// Défaut : 24h.
	TimeoutDuration string `json:"timeout_duration"`

	// BanThreshold est le nombre de messages interdits à partir duquel le ban est appliqué.
	// Défaut : 3.
	BanThreshold int `json:"ban_threshold"`

	// NotifyChannelID est l'identifiant du salon textuel Discord dans lequel le
	// module publie un récapitulatif pour chaque action de modération.
	// Si vide, la fonctionnalité est désactivée.
	NotifyChannelID string `json:"notify_channel_id"`

	// NotifyIncludeContent indique si le contenu original du message supprimé
	// doit être inclus dans la notification. Défaut : false.
	// Attention : activer cette option expose le contenu dans le salon de log.
	NotifyIncludeContent bool `json:"notify_include_content"`
}

// defaults applique les valeurs par défaut manquantes.
func (c *Config) defaults() {
	if c.TimeoutDuration == "" {
		c.TimeoutDuration = "24h"
	}
	if c.BanThreshold <= 0 {
		c.BanThreshold = 3
	}
}
