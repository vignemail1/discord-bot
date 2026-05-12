package invitefilter

import (
	"regexp"
	"strings"
)

// invitePattern capture les formes courantes de liens d'invitation Discord.
// Formes supportées :
//   - discord.gg/<code>
//   - discord.com/invite/<code>
//   - discordapp.com/invite/<code>
//   - dis.gd/<code>
var invitePattern = regexp.MustCompile(
	`(?i)(?:https?://)?(?:www\.)?` +
		`(?:discord\.gg|discord\.com/invite|discordapp\.com/invite|dis\.gd)` +
		`/([a-zA-Z0-9-]{2,32})`,
)

// ExtractInviteCodes retourne tous les codes d'invitation trouvés dans le texte.
func ExtractInviteCodes(text string) []string {
	matches := invitePattern.FindAllStringSubmatch(text, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			out = append(out, m[1])
		}
	}
	return out
}

// ContainsInvite retourne true si le texte contient au moins un lien d'invitation.
func ContainsInvite(text string) bool {
	return invitePattern.MatchString(text)
}

// IsAllowedCode retourne true si le code est dans la liste des codes autorisés.
func IsAllowedCode(code string, allowedCodes []string) bool {
	for _, c := range allowedCodes {
		if strings.EqualFold(c, code) {
			return true
		}
	}
	return false
}
