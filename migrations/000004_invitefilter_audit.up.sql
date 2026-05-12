-- Migration 000004 : table d'audit des actions du module invite_filter.
-- Chaque ligne correspond à une action de modération (suppression, timeout, ban)
-- déclenchée par le module sur un message contenant un lien d'invitation interdit.

CREATE TABLE IF NOT EXISTS invite_filter_audit (
    id             BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
    guild_id       VARCHAR(20)      NOT NULL COMMENT 'Snowflake Discord de la guilde',
    user_id        VARCHAR(20)      NOT NULL COMMENT 'Snowflake Discord de l\'auteur',
    channel_id     VARCHAR(20)      NOT NULL DEFAULT '' COMMENT 'Snowflake du salon',
    message_id     VARCHAR(20)      NOT NULL DEFAULT '' COMMENT 'Snowflake du message supprimé',
    action         ENUM('delete','timeout','ban') NOT NULL COMMENT 'Type de sanction',
    invite_codes   TEXT             NOT NULL DEFAULT '' COMMENT 'Codes extraits, séparés par des virgules',
    counter_value  SMALLINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Valeur du compteur au moment de l\'action',
    created_at     DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    INDEX idx_audit_guild_user  (guild_id, user_id, created_at DESC),
    INDEX idx_audit_guild_time  (guild_id, created_at DESC)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci
  COMMENT='Audit des actions de modération du module invite_filter';
