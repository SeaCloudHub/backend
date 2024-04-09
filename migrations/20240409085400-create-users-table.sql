
-- +migrate Up
CREATE TABLE IF NOT EXISTS "users"
(
    "id"                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "email"                 VARCHAR(255) UNIQUE NOT NULL,
    "first_name"            VARCHAR(255) NOT NULL DEFAULT '',
    "last_name"             VARCHAR(255) NOT NULL DEFAULT '',
    "avatar_url"            VARCHAR(255) NOT NULL DEFAULT '',
    "is_active"             BOOLEAN DEFAULT TRUE,
    "is_admin"              BOOLEAN DEFAULT FALSE,
    "password_changed_at"   TIMESTAMPTZ NULL,
    "last_signin_at"        TIMESTAMPTZ NULL,
    "created_at"            TIMESTAMPTZ DEFAULT NOW(),
    "updated_at"            TIMESTAMPTZ DEFAULT NOW(),
    "deleted_at"            TIMESTAMPTZ NULL
);

-- +migrate Down
DROP TABLE "users";