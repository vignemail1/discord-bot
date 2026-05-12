# discord-bot

Bot Discord de modération multi-guilde en Go, avec dashboard web d'administration.

---

## Prérequis

- Go 1.22+, Docker + Docker Compose v2
- Application créée sur [discord.com/developers](https://discord.com/developers/applications)

---

## Configuration

```bash
cp .env.example .env && $EDITOR .env
```

### Variables obligatoires

| Variable | Usage |
|---|---|
| `DISCORD_BOT_TOKEN` | Token du bot |
| `DISCORD_CLIENT_ID` | Client ID OAuth2 |
| `DISCORD_CLIENT_SECRET` | Client Secret OAuth2 |
| `DISCORD_REDIRECT_URL` | URL callback OAuth2 |
| `DB_PASSWORD` | Mot de passe MariaDB |
| `DB_ROOT_PASSWORD` | Mot de passe root MariaDB |
| `SESSION_SECRET` | Clé HMAC sessions (min. 32 chars) |

### Intents Discord requis (Privileged)

Activer sur **Discord Developer Portal → Bot → Privileged Gateway Intents** :
- **Server Members Intent** (`GUILD_MEMBERS`)
- **Message Content Intent** (`MESSAGE_CONTENT`)

---

## Démarrage rapide

```bash
make docker && make up
curl http://localhost:8080/healthz
make logs
```

---

## Module `invite_filter`

Filtrage des liens d'invitation Discord non autorisés.

### Comportement

| Infraction | Action |
|---|---|
| 1er message interdit | Suppression du message |
| 2ème message interdit | Suppression + timeout 24h |
| 3ème message interdit (et +) | Suppression + ban permanent + reset compteur |
| Utilisateur/rôle whitelisté | Aucune action, aucun compteur |

### Configuration (config_json dans `guild_modules`)

```json
{
  "allowed_invite_codes": ["monserveur"],
  "allowed_guild_ids": [],
  "whitelist_role_ids": ["123456789"],
  "whitelist_user_ids": ["987654321"],
  "timeout_duration": "24h",
  "ban_threshold": 3
}
```

---

## Plan de développement

| Étape | Statut | Description |
|---|---|---|
| 1 | ✅ | Infra : Compose, MariaDB, migrations, `/healthz` |
| 2 | ✅ | Bot : Gateway Discord, READY, persistance guildes |
| 3 | ✅ | Cache config par guilde (sync.Map, TTL, invalidation) |
| 4 | ✅ | Moteur de modules : interface `Module`, `Registry`, `Dispatcher` |
| 5 | ✅ | Module `invite_filter` |
| 6 | ⏳ | Module `identity_history` |
| 7 | ⏳ | Dashboard : OAuth2 Discord, liste guildes |
| 8 | ⏳ | Dashboard : installation bot |
| 9 | ⏳ | Dashboard : configuration modules |
| 10 | ⏳ | Dashboard : audit log |
| 11 | ⏳ | Hardening, observabilité |

---

## Licence

TBD
