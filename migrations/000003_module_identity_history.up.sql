-- Migration 003 : module identity_history

CREATE TABLE IF NOT EXISTS guild_member_identity_state (
    guild_id          VARCHAR(32)  NOT NULL,
    user_id           VARCHAR(32)  NOT NULL,
    username          VARCHAR(64)  NULL,
    global_name       VARCHAR(64)  NULL,
    guild_nick        VARCHAR(64)  NULL,
    avatar_hash       VARCHAR(128) NULL,
    guild_avatar_hash VARCHAR(128) NULL,
    first_seen_at     DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    last_seen_at      DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    last_snapshot_at  DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (guild_id, user_id),
    KEY idx_gmis_user_id           (user_id),
    KEY idx_gmis_guild_username    (guild_id, username),
    KEY idx_gmis_guild_global_name (guild_id, global_name),
    KEY idx_gmis_guild_nick        (guild_id, guild_nick),
    CONSTRAINT fk_gmis_guild FOREIGN KEY (guild_id) REFERENCES guilds (guild_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS guild_member_identity_events (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    guild_id      VARCHAR(32)     NOT NULL,
    user_id       VARCHAR(32)     NOT NULL,
    event_type    VARCHAR(64)     NOT NULL,
    old_value     VARCHAR(255)    NULL,
    new_value     VARCHAR(255)    NULL,
    changed_at    DATETIME(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    source_event  VARCHAR(64)     NOT NULL,
    metadata_json LONGTEXT        NULL CHECK (metadata_json IS NULL OR json_valid(metadata_json)),
    PRIMARY KEY (id),
    KEY idx_gmie_guild_user_time (guild_id, user_id, changed_at),
    KEY idx_gmie_guild_type_time (guild_id, event_type, changed_at),
    KEY idx_gmie_guild_new_value (guild_id, new_value),
    KEY idx_gmie_guild_old_value (guild_id, old_value),
    CONSTRAINT fk_gmie_guild FOREIGN KEY (guild_id) REFERENCES guilds (guild_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
