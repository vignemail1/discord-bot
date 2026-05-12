// Package identityhistory implémente le module de suivi des changements
// de pseudo (username, display_name, nickname) et d'avatar des membres.
package identityhistory

// Config est la configuration du module identity_history désérialisée depuis config_json.
type Config struct {
	// TrackUsername surveille les changements de username Discord global.
	TrackUsername bool `json:"track_username"`

	// TrackDisplayName surveille les changements de display_name (global name Discord).
	TrackDisplayName bool `json:"track_display_name"`

	// TrackNickname surveille les changements de surnom de guilde (nickname).
	TrackNickname bool `json:"track_nickname"`

	// TrackAvatar surveille les changements d'avatar global Discord.
	TrackAvatar bool `json:"track_avatar"`

	// TrackGuildAvatar surveille les changements d'avatar de guilde.
	TrackGuildAvatar bool `json:"track_guild_avatar"`

	// RetentionDays est la durée de conservation des enregistrements en jours.
	// 0 = conservation indéfinie.
	RetentionDays int `json:"retention_days"`
}

// defaults applique les valeurs par défaut manquantes.
func (c *Config) defaults() {
	// Par défaut : tout activé, conservation 90 jours.
	if !c.TrackUsername && !c.TrackDisplayName && !c.TrackNickname && !c.TrackAvatar && !c.TrackGuildAvatar {
		c.TrackUsername = true
		c.TrackDisplayName = true
		c.TrackNickname = true
		c.TrackAvatar = true
		c.TrackGuildAvatar = true
	}
	if c.RetentionDays == 0 {
		c.RetentionDays = 90
	}
}
