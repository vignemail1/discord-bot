-- Migration 002 : module invite_filter

CREATE TABLE IF NOT EXISTS guild_invite_filter_whitelist_roles (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    guild_id   VARCHAR(32)     NOT NULL,
    role_id    VARCHAR(32)     NOT NULL,
    created_at DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_ifwr_guild_role (guild_id, role_id),
    KEY idx_ifwr_guild_id (guild_id),
    CONSTRAINT fk_ifwr_guild FOREIGN KEY (guild_id) REFERENCES guilds (guild_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS guild_invite_filter_whitelist_users (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    guild_id   VARCHAR(32)     NOT NULL,
    user_id    VARCHAR(32)     NOT NULL,
    created_at DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_ifwu_guild_user (guild_id, user_id),
    KEY idx_ifwu_guild_id (guild_id),
    CONSTRAINT fk_ifwu_guild FOREIGN KEY (guild_id) REFERENCES guilds (guild_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS guild_invite_filter_counters (
    guild_id VARCHAR(32)     NOT NULL,
    user_id  VARCHAR(32)     NOT NULL,
    count    INT UNSIGNED    NOT NULL DEFAULT 0,
    last_at  DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (guild_id, user_id),
    CONSTRAINT fk_ifc_guild FOREIGN KEY (guild_id) REFERENCES guilds (guild_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
