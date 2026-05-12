# Architecture — discord-bot

> Bot Discord multi-guilde en Go, avec dashboard web d'administration, modules activables par serveur, et persistance MariaDB.

---

## Table des matières

1. [Vue d'ensemble](#1-vue-densemble)
2. [Contraintes et objectifs](#2-contraintes-et-objectifs)
3. [Stack technique](#3-stack-technique)
4. [Structure du dépôt](#4-structure-du-dépôt)
5. [Modèle de données MariaDB](#5-modèle-de-données-mariadb)
6. [Architecture du bot Discord](#6-architecture-du-bot-discord)
7. [Moteur de modules](#7-moteur-de-modules)
8. [Module : invite_filter](#8-module--invite_filter)
9. [Module : identity_history](#9-module--identity_history)
10. [Architecture du dashboard web](#10-architecture-du-dashboard-web)
11. [Flux OAuth2 Discord](#11-flux-oauth2-discord)
12. [Déploiement Docker Compose](#12-déploiement-docker-compose)
13. [Intents Discord requis](#13-intents-discord-requis)
14. [Stratégie de tests](#14-stratégie-de-tests)
15. [Plan de réalisation par étapes](#15-plan-de-réalisation-par-étapes)
16. [Pipeline qualité](#16-pipeline-qualité)
17. [Évolutions prévues](#17-évolutions-prévues)

---

## 1. Vue d'ensemble

Le projet est un **bot Discord de modération multi-guilde** écrit en Go. Il repose sur trois composants distincts :

- **`cmd/bot`** — Connexion Gateway Discord, dispatch d'événements, exécution des modules.
- **`cmd/web`** — Dashboard HTTP d'administration (authentification, configuration, audit log).
- **`internal/`** — Logique partagée : config, modules, repository MariaDB, cache, audit.

Chaque guilde Discord dispose de sa propre configuration isolée en base. Un moteur de modules générique permet d'activer/désactiver et de configurer chaque fonctionnalité indépendamment, par serveur. Les modules initiaux sont `invite_filter` et `identity_history` ; d'autres peuvent être ajoutés sans modifier le cœur du bot.

```
                    ┌────────────────────────────────────────────┐
                    │              Internet                       │
                    └──────────┬─────────────────┬───────────────┘
                               │ Gateway WSS      │ REST API
                    ┌──────────▼──────────┐  ┌───▼──────────────┐
                    │    cmd/bot          │  │    cmd/web         │
                    │  (Discord Gateway)  │  │  (HTTP Dashboard)  │
                    └──────────┬──────────┘  └───────────────────┘
                               │                       │
                    ┌──────────▼───────────────────────▼──────────┐
                    │               internal/                      │
                    │  config · cache · modules · repository       │
                    │  audit                                       │
                    └──────────────────────┬──────────────────────┘
                                           │
                               ┌───────────▼──────────┐
                               │      MariaDB 10.11    │
                               └──────────────────────┘
```

---

## 2. Contraintes et objectifs

| Contrainte | Décision |
|---|---|
| Multi-guilde avec config isolée | Toutes les tables sont préfixées par `guild_id` |
| Modules activables par guilde | Interface `Module` + registre + table `guild_modules` |
| Pas de config en dur | 100 % variables d'environnement + secrets Docker |
| Robustesse opérationnelle | Retry DB, reconnect Gateway, logs structurés JSON |
| Sécurité des sessions web | CSRF `state` OAuth2, sessions en mémoire (pas de `localStorage`) |
| Fiabilité des déploiements | `healthcheck` MariaDB + `depends_on: condition: service_healthy` |
| Qualité du code | `gofmt`, `go vet`, `staticcheck`, `golangci-lint`, `go test -race` |
| Extensibilité | Ajouter un module = implémenter une interface, pas modifier le cœur |

---

## 3. Stack technique

| Composant | Choix | Justification |
|---|---|---|
| Langage | Go 1.22+ | Typage fort, goroutines, excellent tooling, binaires statiques |
| Bot Discord | `bwmarrin/discordgo` | Bibliothèque Go de référence pour l'API Discord |
| Framework HTTP | `go-chi/chi` v5 | Léger, middleware-friendly, pas de dépendances tierces |
| SQL | `database/sql` + `go-sql-driver/mysql` + `jmoiron/sqlx` | Contrôle total du SQL, compatible MariaDB |
| Migrations | `golang-migrate/migrate` | Migrations versionnées, appliquées au démarrage |
| Base de données | MariaDB 10.11 LTS | Support JSON, DATETIME(6), InnoDB, image Docker officielle |
| Conteneurisation | Docker + Docker Compose v2 | Déploiement reproductible, `service_healthy` |
| Logs | `log/slog` (stdlib Go 1.21+) | Logs structurés JSON sans dépendance externe |
| Tests | `testing` + `testify` + `go-sqlmock` | Tests unitaires, intégration, mocks DB |

---

## 4. Structure du dépôt

```
discord-bot/
├── cmd/
│   ├── bot/
│   │   └── main.go                  # Point d'entrée bot Discord
│   └── web/
│       └── main.go                  # Point d'entrée dashboard HTTP
├── internal/
│   ├── config/
│   │   └── config.go                # Lecture des variables d'environnement
│   ├── cache/
│   │   └── cache.go                 # GuildConfigCache (sync.Map, TTL, invalidation)
│   ├── db/
│   │   ├── db.go                    # Connexion MariaDB, pool, retry
│   │   └── migrations.go            # Chargement et application des migrations
│   ├── repository/
│   │   ├── guild.go                 # CRUD guildes
│   │   ├── module.go                # CRUD modules par guilde
│   │   ├── invite_filter.go         # Repository module invite_filter
│   │   ├── identity_history.go      # Repository module identity_history
│   │   └── audit.go                 # Écriture audit log
│   ├── module/
│   │   ├── module.go                # Interfaces Module, MemberUpdateHandler, UserUpdateHandler + HandlerFunc
│   │   ├── dispatcher.go            # Dispatch événements aux modules actifs
│   │   ├── invitefilter/            # (package invitefilter, sans underscore)
│   │   │   ├── invite_filter.go
│   │   │   └── invite_filter_test.go
│   │   └── identityhistory/         # (package identityhistory, sans underscore)
│   │       ├── config.go
│   │       ├── module.go
│   │       ├── purger.go
│   │       ├── repository.go
│   │       ├── module_test.go
│   │       ├── purger_test.go
│   │       └── repository_test.go
│   ├── bot/
│   │   ├── session.go               # Connexion Discord Gateway, reconnect
│   │   └── handlers.go              # Handlers Gateway → Dispatcher
│   ├── web/
│   │   ├── server.go                # Serveur HTTP chi, routing
│   │   ├── context.go               # Clés de contexte HTTP
│   │   ├── context_util.go          # Helpers d'extraction de contexte
│   │   ├── middleware.go            # Auth, logging, CSRF
│   │   ├── security.go              # Security headers
│   │   ├── ratelimit.go             # Rate limiting par IP (token bucket)
│   │   ├── metrics.go               # Métriques Prometheus + middleware
│   │   ├── health.go                # Handler /healthz
│   │   ├── oauth2.go                # Flux OAuth2 Discord
│   │   ├── session.go               # SessionStore en mémoire
│   │   ├── handlers_guild.go        # Routes guildes
│   │   ├── handlers_module.go       # Routes config modules
│   │   ├── handlers_audit.go        # Routes audit log
│   │   └── handlers_identity.go     # Routes identity history
│   └── audit/
│       └── audit.go                 # Service d'audit log
├── migrations/
│   ├── 000001_initial_schema.up.sql
│   ├── 000001_initial_schema.down.sql
│   ├── 000002_module_invite_filter.up.sql
│   ├── 000002_module_invite_filter.down.sql
│   ├── 000003_module_identity_history.up.sql
│   └── 000003_module_identity_history.down.sql
├── Dockerfile.bot
├── Dockerfile.web
├── docker-compose.yml
├── docker-compose.override.yml      # Dev : ports exposés, rebuild auto
├── Makefile
├── go.mod
├── go.sum
├── .env.example
├── .gitignore
├── ARCHITECTURE.md                  # Ce document
└── README.md
```

---

## 5. Modèle de données MariaDB

### Conventions

- Tous les IDs Discord sont stockés en `VARCHAR(32)` (Snowflakes 64-bit sérialisés en chaîne).
- Timestamps en `DATETIME(6)` pour préserver la microseconde.
- Charset `utf8mb4` + collation `utf8mb4_unicode_ci` sur toutes les tables.
- Les index composites commencent toujours par `guild_id` : toutes les requêtes admin sont bornées par guilde.

### Migration 1 — Schéma initial

```sql
-- guildes connues du bot
CREATE TABLE guilds (
    guild_id        VARCHAR(32)  NOT NULL,
    guild_name      VARCHAR(255) NOT NULL DEFAULT '',
    owner_user_id   VARCHAR(32)  NOT NULL DEFAULT '',
    bot_joined_at   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    active          TINYINT(1)   NOT NULL DEFAULT 1,
    created_at      DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at      DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (guild_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- état d'activation des modules par guilde
CREATE TABLE guild_modules (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    guild_id    VARCHAR(32)     NOT NULL,
    module_name VARCHAR(64)     NOT NULL,
    enabled     TINYINT(1)      NOT NULL DEFAULT 0,
    config_json LONGTEXT        NOT NULL DEFAULT '{}' CHECK (json_valid(config_json)),
    created_at  DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at  DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_guild_module (guild_id, module_name),
    KEY idx_gm_guild_id (guild_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- journal général des actions du bot
CREATE TABLE audit_logs (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    guild_id    VARCHAR(32)     NOT NULL,
    module_name VARCHAR(64)     NOT NULL,
    action_type VARCHAR(64)     NOT NULL,
    user_id     VARCHAR(32)     NULL,
    target_id   VARCHAR(32)     NULL,
    message_id  VARCHAR(32)     NULL,
    channel_id  VARCHAR(32)     NULL,
    detail_json LONGTEXT        NULL CHECK (detail_json IS NULL OR json_valid(detail_json)),
    occurred_at DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_al_guild_time  (guild_id, occurred_at),
    KEY idx_al_guild_mod   (guild_id, module_name, occurred_at),
    KEY idx_al_guild_user  (guild_id, user_id, occurred_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- sessions OAuth2 dashboard
CREATE TABLE oauth_sessions (
    session_id      VARCHAR(64)  NOT NULL,
    state_token     VARCHAR(64)  NOT NULL,
    user_id         VARCHAR(32)  NULL,
    username        VARCHAR(128) NULL,
    access_token    TEXT         NULL,
    refresh_token   TEXT         NULL,
    token_expiry    DATETIME(6)  NULL,
    created_at      DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at      DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (session_id),
    KEY idx_os_state (state_token)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### Migration 2 — Module invite_filter

```sql
-- whitelist rôles exemptés du filtrage
CREATE TABLE guild_invite_filter_whitelist_roles (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    guild_id    VARCHAR(32)     NOT NULL,
    role_id     VARCHAR(32)     NOT NULL,
    created_at  DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_ifwr_guild_role (guild_id, role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- whitelist utilisateurs exemptés du filtrage
CREATE TABLE guild_invite_filter_whitelist_users (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    guild_id    VARCHAR(32)     NOT NULL,
    user_id     VARCHAR(32)     NOT NULL,
    created_at  DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_ifwu_guild_user (guild_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- compteur d'infractions par utilisateur/guilde
CREATE TABLE guild_invite_filter_counters (
    guild_id    VARCHAR(32)     NOT NULL,
    user_id     VARCHAR(32)     NOT NULL,
    count       INT UNSIGNED    NOT NULL DEFAULT 0,
    last_at     DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (guild_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### Migration 3 — Module identity_history

```sql
-- dernier état connu de chaque membre par guilde
CREATE TABLE guild_member_identity_state (
    guild_id            VARCHAR(32)  NOT NULL,
    user_id             VARCHAR(32)  NOT NULL,
    username            VARCHAR(64)  NULL,
    global_name         VARCHAR(64)  NULL,
    guild_nick          VARCHAR(64)  NULL,
    avatar_hash         VARCHAR(128) NULL,
    guild_avatar_hash   VARCHAR(128) NULL,
    first_seen_at       DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    last_seen_at        DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    last_snapshot_at    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (guild_id, user_id),
    KEY idx_gmis_user_id           (user_id),
    KEY idx_gmis_guild_username    (guild_id, username),
    KEY idx_gmis_guild_global_name (guild_id, global_name),
    KEY idx_gmis_guild_nick        (guild_id, guild_nick)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- historique append-only des changements d'identité
CREATE TABLE guild_member_identity_events (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    guild_id        VARCHAR(32)     NOT NULL,
    user_id         VARCHAR(32)     NOT NULL,
    event_type      VARCHAR(64)     NOT NULL,
    old_value       VARCHAR(255)    NULL,
    new_value       VARCHAR(255)    NULL,
    changed_at      DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    source_event    VARCHAR(64)     NOT NULL,
    metadata_json   LONGTEXT        NULL CHECK (metadata_json IS NULL OR json_valid(metadata_json)),
    PRIMARY KEY (id),
    KEY idx_gmie_guild_user_time  (guild_id, user_id, changed_at),
    KEY idx_gmie_guild_type_time  (guild_id, event_type, changed_at),
    KEY idx_gmie_guild_new_value  (guild_id, new_value),
    KEY idx_gmie_guild_old_value  (guild_id, old_value)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

---

## 6. Architecture du bot Discord

### Connexion Gateway

Le bot se connecte via la Gateway Discord WebSocket avec `discordgo`. La session est créée une seule fois au démarrage, avec reconnexion automatique. Les intents sont déclarés explicitement (voir §13).

### Multi-guilde

Au démarrage, Discord envoie un événement `GUILD_CREATE` pour chaque guilde où le bot est présent. Le handler persiste ou met à jour chaque guilde dans `guilds`, puis charge la configuration des modules en mémoire.

La config est mise en cache par `guild_id` dans un `GuildConfigCache` (wrapper `sync.Map` avec TTL). La DB reste la source de vérité ; le cache est invalidé à chaque modification depuis le dashboard.

### Flux d'un événement

```
Discord Gateway
       │
       ▼
  handlers.go           — réception brute de l'événement discordgo
       │
       ▼
  dispatcher.go         — itère sur les modules actifs pour la guild_id
       │
       ├──► invite_filter.HandleMessage(ctx, event)
       ├──► identity_history.HandleMemberUpdate(ctx, event)   [GUILD_MEMBER_UPDATE]
       ├──► identity_history.HandleUserUpdate(ctx, event)     [USER_UPDATE, toutes guildes]
       └──► [modules futurs]
```

---

## 7. Moteur de modules

### Interfaces

```go
// Module est le contrat de base implémenté par chaque module.
type Module interface {
    // Name retourne le nom unique du module (ex : "invite_filter").
    // Doit correspondre exactement au module_name en base de données.
    Name() string

    // HandleMessage est appelé par le Dispatcher pour chaque message reçu
    // sur une guilde où ce module est actif.
    HandleMessage(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, cfg *cache.GuildConfig) error
}

// MemberUpdateHandler est implémenté par les modules qui consomment GUILD_MEMBER_UPDATE.
type MemberUpdateHandler interface {
    HandleMemberUpdate(ctx context.Context, s *discordgo.Session, ev *discordgo.GuildMemberUpdate, cfg *cache.GuildConfig) error
}

// UserUpdateHandler est implémenté par les modules qui consomment USER_UPDATE.
// USER_UPDATE est global (pas de guild_id) ; le dispatcher réplique l'appel
// pour chaque guilde active où ce module est activé.
type UserUpdateHandler interface {
    HandleUserUpdate(ctx context.Context, s *discordgo.Session, ev *discordgo.UserUpdate, guildID string, cfg *cache.GuildConfig) error
}
```

Les interfaces `MemberUpdateHandler` et `UserUpdateHandler` sont **optionnelles** : le dispatcher les détecte via une assertion de type (`mod.(MemberUpdateHandler)`) et ne les appelle que si le module les implémente. Cela permet d'ajouter de nouveaux types d'événements sans modifier les modules existants.

### Registre

Chaque module se déclare via `Register(module)` au démarrage de `cmd/bot`. Le dispatcher appelle uniquement les modules déclarés actifs pour la guilde émettrice de l'événement.

---

## 8. Module : invite_filter

### Logique d'escalade

| Condition | Action |
|---|---|
| Auteur dans whitelist (rôle ou user_id) | Aucune action, aucun comptage |
| Message sans lien d'invitation Discord | Aucune action |
| Lien pointant vers la guilde gérée | Aucune action |
| 1re infraction | Suppression du message + entrée audit |
| 2e infraction | Suppression + timeout 24 h + entrée audit |
| ≥ 3e infraction | Suppression + ban + entrée audit |

### Détection des liens

Patterns détectés (insensible à la casse, regex) :

```
(?i)(discord\.gg|discord\.com/invite|discordapp\.com/invite)/([A-Za-z0-9-]+)
```

Le code extrait est vérifié contre `allowed_invite_codes` et `allowed_guild_ids`.

### Configuration JSON (dans `guild_modules.config_json`)

```json
{
  "allowed_invite_codes": ["abc123"],
  "allowed_guild_ids":    ["123456789012345678"],
  "timeout_duration":     "24h",
  "ban_threshold":        3,
  "whitelist_role_ids":   [],
  "whitelist_user_ids":   []
}
```

---

## 9. Module : identity_history

### Champs suivis

| Champ | Objet source Discord | Note |
|---|---|---|
| `username` | User | Identifiant de compte global |
| `global_name` | User | Display name global |
| `guild_nick` | Guild Member | Surnom local au serveur |
| `avatar_hash` | User | Hash de l'avatar global |
| `guild_avatar_hash` | Guild Member | Hash de l'avatar de membre |

### Types d'événements

| `event_type` | Déclencheur Gateway |
|---|---|
| `initial_snapshot` | Premier contact avec ce membre sur cette guilde |
| `username_changed` | `USER_UPDATE` : `username` différent |
| `global_name_changed` | `USER_UPDATE` : `global_name` différent |
| `guild_nick_changed` | `GUILD_MEMBER_UPDATE` : `nick` différent |
| `avatar_hash_changed` | `USER_UPDATE` : `avatar` différent |
| `guild_avatar_hash_changed` | `GUILD_MEMBER_UPDATE` : `avatar` différent |

### Propagation USER_UPDATE

`USER_UPDATE` est global au compte. Le handler réplique le changement dans l'historique de **toutes les guildes** où cet `user_id` est présent et géré par le bot.

### Reconstruction des URLs d'avatar

Les hashs sont stockés bruts. L'URL est reconstruite à l'affichage :

```
Avatar global   : https://cdn.discordapp.com/avatars/{user_id}/{avatar_hash}.png
Avatar de membre: https://cdn.discordapp.com/guilds/{guild_id}/users/{user_id}/avatars/{guild_avatar_hash}.png
```

### Rétention

Chaque guilde configure `retention_days` (0 = illimité). Une goroutine de purge (`Purger`) s'exécute périodiquement et supprime les événements expirés de la table `guild_member_identity_events`.

---

## 10. Architecture du dashboard web

### Routes

```
# Publiques
GET  /healthz
GET  /metrics
GET  /auth/login
GET  /auth/callback
GET  /auth/logout

# Protégées (session valide requise)
GET  /guilds
GET  /guilds/{guildID}
POST /guilds/{guildID}/install
GET  /guilds/{guildID}/modules
PUT  /guilds/{guildID}/modules/{moduleName}
PUT  /guilds/{guildID}/modules/{moduleName}/config

GET  /guilds/{guildID}/audit
GET  /guilds/{guildID}/identity
GET  /guilds/{guildID}/identity/{userID}
```

### Autorisation

- Toutes les routes protégées nécessitent une session valide.
- Pour chaque requête sur une guilde, vérification que l'utilisateur connecté en est gestionnaire (`manage_guild` dans le scope `guilds` Discord).

### Middlewares

| Middleware | Rôle |
|---|---|
| `slogRequest` | Log structuré JSON de chaque requête |
| `securityHeaders` | Headers HTTP de sécurité (CSP, X-Frame-Options, etc.) |
| `metricsMiddleware` | Comptage des requêtes et durées (Prometheus) |
| `RateLimitMiddleware` | Token bucket par IP (20 req/s, burst 50) |
| `requireAuth` | Vérification de la session sur les routes protégées |

---

## 11. Flux OAuth2 Discord

### Auth utilisateur (accès dashboard)

```
1. GET /auth/login
   → 302 discord.com/oauth2/authorize
       ?client_id=...&scope=identify+guilds
       &response_type=code
       &redirect_uri=...&state=<csrf>

2. Discord → GET /auth/callback?code=...&state=...
   → Validation state (CSRF)
   → Échange code → access_token
   → GET /api/users/@me
   → GET /api/users/@me/guilds
   → Création session, cookie sécurisé HttpOnly
   → 302 /guilds
```

### Installation du bot sur une guilde

```
POST /guilds/{guildID}/install
→ 302 discord.com/oauth2/authorize
    ?client_id=...
    &scope=bot
    &permissions=<bitmask>
    &guild_id={guildID}
    &disable_guild_select=true
```

**Bitmask de permissions requis** :
`VIEW_CHANNEL` (1024) + `SEND_MESSAGES` (2048) + `MANAGE_MESSAGES` (8192) +
`KICK_MEMBERS` (2) + `BAN_MEMBERS` (4) + `MODERATE_MEMBERS` (1099511627776)

---

## 12. Déploiement Docker Compose

Trois services : `mariadb`, `bot`, `web`. Le démarrage de `bot` et `web` est conditionné à la santé de `mariadb` via `depends_on: condition: service_healthy`.

### Dockerfile multi-stage (bot et web)

```dockerfile
# Build
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /app ./cmd/<target>

# Runtime distroless
FROM gcr.io/distroless/static-debian12
COPY --from=builder /app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]
```

### Variables d'environnement bot

```
DISCORD_BOT_TOKEN
DB_HOST / DB_PORT / DB_NAME / DB_USER / DB_PASSWORD
LOG_LEVEL (debug|info|warn|error)
```

### Variables d'environnement web

```
DISCORD_CLIENT_ID / DISCORD_CLIENT_SECRET / DISCORD_REDIRECT_URL
DB_HOST / DB_PORT / DB_NAME / DB_USER / DB_PASSWORD
WEB_LISTEN_ADDR (défaut: :8080)
SESSION_SECRET (clé HMAC sessions)
LOG_LEVEL
```

---

## 13. Intents Discord requis

| Intent | Valeur | Privileged | Usage |
|---|---|---|---|
| `GUILDS` | 1 | Non | `GUILD_CREATE`, `GUILD_DELETE` |
| `GUILD_MEMBERS` | 2 | **Oui** | `GUILD_MEMBER_ADD`, `GUILD_MEMBER_UPDATE`, `GUILD_MEMBERS_CHUNK` |
| `GUILD_MESSAGES` | 512 | Non | `MESSAGE_CREATE` |
| `MESSAGE_CONTENT` | 32768 | **Oui** | Lecture du contenu des messages |

Les intents `GUILD_MEMBERS` et `MESSAGE_CONTENT` doivent être activés manuellement dans **Discord Developer Portal → Application → Bot → Privileged Gateway Intents**.

---

## 14. Stratégie de tests

| Niveau | Outil | Cible |
|---|---|---|
| Unitaire | `testing` + `testify/assert` | Parseurs, logique métier, config |
| Mock DB | `DATA-DOG/go-sqlmock` | Repositories sans MariaDB réelle |
| Intégration DB | `testing` + MariaDB de test Docker | Migrations, CRUD, contraintes |
| HTTP | `net/http/httptest` | Handlers, middleware auth |
| Gateway mock | Events `discordgo` construits manuellement | Modules, dispatcher |
| Race detector | `go test -race ./...` | Détection des data races |

### Commandes

```bash
make lint        # gofmt + go vet + staticcheck + golangci-lint
make test        # go test -race ./...
make test-int    # go test -race -tags=integration ./...  (nécessite TEST_DB_DSN)
make build       # build cmd/bot + cmd/web
make docker      # docker compose build + config validate
```

---

## 15. Plan de réalisation par étapes

| Étape | Objectif | Critère de validation |
|---|---|---|
| 1 | Infra : Compose, MariaDB, healthcheck, migrations, `/healthz` | Stack démarre, tables créées, `200 /healthz` |
| 2 | Bot : connexion Gateway, READY, intents | Bot en ligne, guildes listées |
| 3 | Multi-guilde : persistance, cache config | Config isolée par `guild_id` |
| 4 | Moteur de modules : interfaces, registre, dispatcher | Module mock reçoit les événements ciblés |
| 5 | Module `invite_filter` | Escalade 1/2/3 testée, whitelist testée |
| 6 | Module `identity_history` | Snapshot initial, diff, idempotence testés |
| 7 | Dashboard : auth OAuth2 Discord, liste guildes | Login complet, `/guilds` affiché |
| 8 | Dashboard : installation bot par guilde | Lien bot correct, statut installé détecté |
| 9 | Dashboard : config modules | Sauvegarde persistée, rechargée par le bot |
| 10 | Dashboard : audit log + recherche identité | Filtres, search username → historique |
| 11 | Hardening : métriques Prometheus, rate-limiting, security headers, graceful shutdown | `/metrics` répond, 429 sur dépassement, arrêt propre sans perte de requêtes |

---

## 16. Pipeline qualité

À exécuter à chaque étape, obligatoire avant tout push :

```bash
gofmt -w ./...
go vet ./...
staticcheck ./...
golangci-lint run
go test -race ./...
docker compose config   # valide la syntaxe du compose
hadolint Dockerfile.*   # lint des Dockerfiles
```

---

## 17. Évolutions prévues

Le moteur de modules est conçu pour accueillir de nouveaux modules sans modification du cœur :

| Module futur | Description |
|---|---|
| `word_filter` | Filtrage de mots/expressions interdits |
| `spam_filter` | Détection de spam par volume ou répétitions |
| `raid_protection` | Détection de vagues de joins |
| `welcome` | Message de bienvenue configurable |
| `slowmode` | Slowmode automatique selon l'activité |
| `role_guard` | Surveillance des attributions de rôles suspectes |

---

*Document maintenu dans `ARCHITECTURE.md` à la racine du dépôt `vignemail1/discord-bot`.*  
*Dernière mise à jour : 2026-05-12*
