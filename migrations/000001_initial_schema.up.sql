-- Migration 001 : schéma initial
-- Tables : guilds, guild_modules, audit_logs, oauth_sessions

CREATE TABLE IF NOT EXISTS guilds (
    guild_id      VARCHAR(32)  NOT NULL,
    guild_name    VARCHAR(255) NOT NULL DEFAULT '',
    owner_user_id VARCHAR(32)  NOT NULL DEFAULT '',
    bot_joined_at DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    active        TINYINT(1)   NOT NULL DEFAULT 1,
    created_at    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (guild_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS guild_modules (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    guild_id    VARCHAR(32)     NOT NULL,
    module_name VARCHAR(64)     NOT NULL,
    enabled     TINYINT(1)      NOT NULL DEFAULT 0,
    config_json LONGTEXT        NOT NULL DEFAULT ('{}') CHECK (json_valid(config_json)),
    created_at  DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at  DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_guild_module (guild_id, module_name),
    KEY idx_gm_guild_id (guild_id),
    CONSTRAINT fk_gm_guild FOREIGN KEY (guild_id) REFERENCES guilds (guild_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS audit_logs (
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
    KEY idx_al_guild_time (guild_id, occurred_at),
    KEY idx_al_guild_mod  (guild_id, module_name, occurred_at),
    KEY idx_al_guild_user (guild_id, user_id, occurred_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS oauth_sessions (
    session_id    VARCHAR(64)  NOT NULL,
    state_token   VARCHAR(64)  NOT NULL,
    user_id       VARCHAR(32)  NULL,
    username      VARCHAR(128) NULL,
    access_token  TEXT         NULL,
    refresh_token TEXT         NULL,
    token_expiry  DATETIME(6)  NULL,
    created_at    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (session_id),
    KEY idx_os_state (state_token)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
