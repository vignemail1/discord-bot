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
| `DISCORD_REDIRECT_URL` | URL de callback OAuth2 (ex : `http://localhost:8080/callback`) |
| `DB_PASSWORD` | Mot de passe MariaDB utilisateur applicatif |
| `DB_ROOT_PASSWORD` | Mot de passe root MariaDB |
| `SESSION_SECRET` | Clé HMAC des sessions web (min. 32 caractères) |

### Intents Discord requis

Activez dans **Discord Developer Portal → Application → Bot → Privileged Gateway Intents** :

- **Server Members Intent** (`GUILD_MEMBERS`)
- **Message Content Intent** (`MESSAGE_CONTENT`)

---

## Démarrage rapide (Docker Compose)

```bash
# 1. Construire les images
make docker

# 2. Démarrer la stack
make up

# 3. Vérifier que le dashboard est up
curl http://localhost:8080/healthz
# {"status":"ok"}

# 4. Consulter les logs
make logs
```

---

## Développement local (sans Docker)

```bash
# Lancer MariaDB seule
docker compose up -d mariadb

# Exporter les variables
export $(cat .env | grep -v '^#' | xargs)
export DB_HOST=localhost DB_PORT=3307

# Démarrer le dashboard
go run ./cmd/web

# Dans un autre terminal : démarrer le bot
go run ./cmd/bot
```

---

## Commandes `make`

```bash
make build      # Compile cmd/bot et cmd/web dans ./bin/
make lint       # gofmt + go vet + staticcheck + golangci-lint
make test       # go test -race ./...
make test-int   # Tests d'intégration (nécessitent une DB)
make docker     # Build images + valide docker-compose.yml
make up         # Lance la stack complète
make down       # Arrête la stack
make logs       # Tail logs tous services
make clean      # Supprime ./bin/
```

---

## Structure du projet

```
discord-bot/
├── cmd/bot/          # Point d'entrée bot Discord
├── cmd/web/          # Point d'entrée dashboard HTTP
├── internal/
│   ├── config/       # Lecture des variables d'environnement
│   ├── db/           # Connexion MariaDB + runner de migrations
│   ├── module/       # Interface Module, registre, dispatcher
│   ├── repository/   # CRUD guildes, modules, audit
│   ├── bot/          # Session Gateway Discord
│   ├── web/          # Serveur HTTP, OAuth2, handlers
│   └── audit/        # Service d'audit log
├── migrations/       # Fichiers SQL versionnés (golang-migrate)
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
| 2 | ⏳ | Bot : connexion Gateway Discord, READY |
| 3 | ⏳ | Multi-guilde : persistance, cache config |
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
