-- +migrate Up
CREATE TABLE IF NOT EXISTS guild_modules (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    guild_id    VARCHAR(20)     NOT NULL,
    module_name VARCHAR(64)     NOT NULL,
    enabled     TINYINT(1)      NOT NULL DEFAULT 1,
    config_json JSON            NOT NULL DEFAULT ('{}'),
    created_at  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_guild_module (guild_id, module_name),
    CONSTRAINT fk_guild_modules_guild
        FOREIGN KEY (guild_id) REFERENCES guilds(guild_id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Index utile pour les lectures par guilde
CREATE INDEX IF NOT EXISTS idx_guild_modules_guild_id ON guild_modules (guild_id);

-- +migrate Down
DROP TABLE IF EXISTS guild_modules;
