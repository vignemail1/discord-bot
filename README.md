# discord-bot

Bot Discord de modération multi-guilde en Go, avec dashboard web d'administration.

> Architecture complète : voir [`ARCHITECTURE.md`](./ARCHITECTURE.md)

---

## Prérequis

- Go 1.22+
- Docker + Docker Compose v2
- Un compte Discord Developer et une application créée sur [discord.com/developers](https://discord.com/developers/applications)

---

## Configuration

Copier `.env.example` en `.env` :

```bash
cp .env.example .env
$EDITOR .env
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

### Variables optionnelles

| Variable | Défaut | Usage |
|---|---|---|
| `CACHE_TTL_SECONDS` | `300` | TTL cache config guildes |
| `LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |
| `HTTP_ADDR` | `:8080` | Adresse dashboard |
| `DB_HOST` | `mariadb` | Hôte MariaDB |
| `DB_PORT` | `3306` | Port MariaDB |

### Intents Discord requis (Privileged)

Activer sur **Discord Developer Portal → Bot → Privileged Gateway Intents** :
- **Server Members Intent**
- **Message Content Intent**

---

## Démarrage rapide

```bash
cp .env.example .env && $EDITOR .env
make docker && make up
curl http://localhost:8080/healthz
make logs
```

## Développement local

```bash
docker compose up -d mariadb
export $(cat .env | grep -v '^#' | xargs)
export DB_HOST=localhost DB_PORT=3307
go run ./cmd/bot
```

---

## Plan de développement

| Étape | Statut | Description |
|---|---|---|
| 1 | ✅ | Infra : Compose, MariaDB, migrations, `/healthz` |
| 2 | ✅ | Bot : connexion Gateway, READY, persistance guildes |
| 3 | ✅ | Cache config par guilde (sync.Map, TTL, invalidation) |
| 4 | ✅ | Moteur de modules : interface `Module`, `Registry`, `Dispatcher` |
| 5 | ⏳ | Module `invite_filter` |
| 6 | ⏳ | Module `identity_history` |
| 7 | ⏳ | Dashboard : OAuth2 Discord, liste guildes |
| 8 | ⏳ | Dashboard : installation bot |
| 9 | ⏳ | Dashboard : configuration modules |
| 10 | ⏳ | Dashboard : audit log |
| 11 | ⏳ | Hardening, observabilité |

---

## Licence

TBD
