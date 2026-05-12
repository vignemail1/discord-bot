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

## Module `identity_history`

Suivi des changements d'identité des membres (username, display name, pseudo de guilde, avatar).

### Événements écoutés

| Événement Discord | Champs suivis |
|---|---|
| `GUILD_MEMBER_UPDATE` | `nickname`, `guild_avatar` |
| `USER_UPDATE` | `username`, `global_name`, `avatar` |

### Configuration (config_json dans `guild_modules`)

```json
{
  "track_username": true,
  "track_display_name": true,
  "track_nickname": true,
  "track_avatar": true,
  "track_guild_avatar": true,
  "retention_days": 90
}
```

---

## Dashboard API

Serveur HTTP démarré automatiquement avec le bot.

### Authentification

OAuth2 Discord. Tous les endpoints sous `/guilds/` requièrent une session valide.

```
GET  /auth/login     → redirige vers Discord OAuth2
GET  /auth/callback  → échange le code, crée la session
GET  /auth/logout    → supprime la session
```

### Endpoints

| Méthode | Route | Description |
|---|---|---|
| GET | `/healthz` | Santé du service + état base de données |
| GET | `/metrics` | Métriques Prometheus |
| GET | `/guilds` | Liste des guildes de l'utilisateur connecté |
| GET | `/guilds/{id}` | Détail d'une guilde |
| POST | `/guilds/{id}/install` | Installe le bot sur une guilde |
| GET | `/guilds/{id}/modules` | Liste des modules et leur config |
| PUT | `/guilds/{id}/modules/{name}` | Active/désactive un module |
| PUT | `/guilds/{id}/modules/{name}/config` | Met à jour la config d'un module |
| GET | `/guilds/{id}/audit` | Audit log paginé |
| GET | `/guilds/{id}/identity` | Liste des membres avec état courant |
| GET | `/guilds/{id}/identity/{userID}` | Historique d'identité d'un membre |

### Pagination (identity history)

```
GET /guilds/{id}/identity/{userID}?limit=50&before=<id>&type=<event_type>
→ { state, events[], next_cursor, count }
```

---

## Observabilité

### Métriques Prometheus

Exposeé sur `GET /metrics`. Métriques disponibles :

| Métrique | Type | Description |
|---|---|---|
| `discordbot_http_requests_total` | Counter | Requêtes HTTP par méthode, route et statut |
| `discordbot_http_request_duration_seconds` | Histogram | Durée des requêtes HTTP |

Intégration Prometheus (scrape config) :

```yaml
scrape_configs:
  - job_name: discord-bot
    static_configs:
      - targets: ['localhost:8080']
```

### Health check

`GET /healthz` retourne `200 ok` si toutes les dépendances sont disponibles, `503 degraded` sinon.

```json
{
  "status": "ok",
  "timestamp": "2026-05-12T14:00:00Z",
  "checks": {
    "database": "ok"
  }
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
| 6 | ✅ | Module `identity_history` |
| 7 | ✅ | Dashboard : OAuth2 Discord, liste guildes |
| 8 | ✅ | Dashboard : installation bot |
| 9 | ✅ | Dashboard : configuration modules |
| 10 | ✅ | Dashboard : audit log + identity history routes |
| 11 | ✅ | Hardening, observabilité |

---

## Licence

TBD
