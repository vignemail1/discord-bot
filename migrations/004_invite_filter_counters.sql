-- +migrate Up
CREATE TABLE IF NOT EXISTS guild_member_module_counters (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    guild_id    VARCHAR(20)     NOT NULL,
    user_id     VARCHAR(20)     NOT NULL,
    module_name VARCHAR(64)     NOT NULL,
    count       INT UNSIGNED    NOT NULL DEFAULT 0,
    updated_at  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_counter (guild_id, user_id, module_name),
    CONSTRAINT fk_counters_guild
        FOREIGN KEY (guild_id) REFERENCES guilds(guild_id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX IF NOT EXISTS idx_counters_guild_user ON guild_member_module_counters (guild_id, user_id);

-- +migrate Down
DROP TABLE IF EXISTS guild_member_module_counters;
