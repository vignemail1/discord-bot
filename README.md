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

Copier `.env.example` en `.env` et remplir toutes les valeurs :

```bash
cp .env.example .env
$EDITOR .env
```

### Variables obligatoires

| Variable | Usage |
|---|---|
| `DISCORD_BOT_TOKEN` | Token du bot (Discord Developer Portal → Bot) |
| `DISCORD_CLIENT_ID` | Client ID OAuth2 (dashboard web) |
| `DISCORD_CLIENT_SECRET` | Client Secret OAuth2 (dashboard web) |
| `DISCORD_REDIRECT_URL` | URL de callback OAuth2 (ex : `http://localhost:8080/auth/callback`) |
| `DB_PASSWORD` | Mot de passe MariaDB utilisateur applicatif |
| `DB_ROOT_PASSWORD` | Mot de passe root MariaDB |
| `SESSION_SECRET` | Clé HMAC des sessions web (min. 32 caractères) |

### Variables optionnelles

| Variable | Défaut | Usage |
|---|---|---|
| `CACHE_TTL_SECONDS` | `300` | TTL en secondes du cache de config par guilde |
| `LOG_LEVEL` | `info` | Niveau de log (`debug`, `info`, `warn`, `error`) |
| `HTTP_ADDR` | `:8080` | Adresse d'écoute du dashboard |
| `DB_HOST` | `mariadb` | Hôte MariaDB |
| `DB_PORT` | `3306` | Port MariaDB |

### Intents Discord requis (Privileged)

Activer dans **Discord Developer Portal → Application → Bot → Privileged Gateway Intents** :

| Intent | Raison |
|---|---|
| **Server Members Intent** (`GUILD_MEMBERS`) | Réception des événements membres |
| **Message Content Intent** (`MESSAGE_CONTENT`) | Lecture du contenu des messages pour le filtrage |

---

## Démarrage rapide (Docker Compose)

```bash
make docker
make up
curl http://localhost:8080/healthz   # {"status":"ok"}
make logs
```

---

## Développement local (sans Docker)

```bash
docker compose up -d mariadb
export $(cat .env | grep -v '^#' | xargs)
export DB_HOST=localhost DB_PORT=3307
go run ./cmd/bot
# dans un autre terminal :
go run ./cmd/web
```

---

## Commandes `make`

```bash
make build      # Compile cmd/bot et cmd/web dans ./bin/
make lint       # gofmt + go vet + staticcheck + golangci-lint
make test       # go test -race ./...
make test-int   # Tests d'intégration (nécessitent une DB)
make docker     # Build images
make up         # Lance la stack complète
make down       # Arrête la stack
make logs       # Tail logs
make clean      # Supprime ./bin/
```

---

## Structure du projet

```
discord-bot/
├── cmd/bot/          # Point d'entrée bot Discord
├── cmd/web/          # Point d'entrée dashboard HTTP
├── internal/
│   ├── config/       # Variables d'environnement
│   ├── db/           # Connexion MariaDB + migrations
│   ├── cache/        # GuildConfigCache (sync.Map + TTL)
│   ├── module/       # Interface Module, registre, dispatcher (step 4)
│   ├── repository/   # Interfaces de persistance
│   │   ├── mariadb/  # Implémentations MariaDB (sqlx)
│   │   └── mock/     # Mocks en mémoire pour les tests
│   ├── bot/          # Session Gateway, handlers READY/GUILD_CREATE/DELETE
│   ├── web/          # Serveur HTTP, OAuth2 (steps 7-10)
│   └── audit/        # Service d'audit log (step 7)
├── migrations/       # SQL versionnés (golang-migrate)
├── Dockerfile.bot
├── Dockerfile.web
├── docker-compose.yml
└── Makefile
```

---

## Plan de développement

| Étape | Statut | Description |
|---|---|---|
| 1 | ✅ | Infra : Compose, MariaDB, migrations, `/healthz` |
| 2 | ✅ | Bot : connexion Gateway Discord, READY, persistance guildes |
| 3 | ✅ | Multi-guilde : cache config par guilde (sync.Map, TTL, invalidation) |
| 4 | ⏳ | Moteur de modules : interface, registre, dispatcher |
| 5 | ⏳ | Module `invite_filter` |
| 6 | ⏳ | Module `identity_history` |
| 7 | ⏳ | Dashboard : OAuth2 Discord, liste guildes |
| 8 | ⏳ | Dashboard : installation bot par guilde |
| 9 | ⏳ | Dashboard : configuration des modules |
| 10 | ⏳ | Dashboard : audit log + recherche identité |
| 11 | ⏳ | Hardening : retry, purge, observabilité |

---

## Licence

TBD
