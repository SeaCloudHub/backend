
-- +migrate Up
CREATE TABLE IF NOT EXISTS "files"
(
    "id"            SERIAL PRIMARY KEY,
    "name"          VARCHAR(255) NOT NULL,
    "path"          TEXT NOT NULL,
    "full_path"     TEXT UNIQUE NOT NULL,
    "size"          BIGINT NOT NULL,
    "mode"          BIGINT NOT NULL,
    "mime_type"     VARCHAR(255) NOT NULL,
    "md5"           VARCHAR(32) NOT NULL,
    "is_dir"        BOOLEAN NOT NULL DEFAULT FALSE,
    "created_at"    TIMESTAMPTZ DEFAULT NOW(),
    "updated_at"    TIMESTAMPTZ DEFAULT NOW(),
    "deleted_at"    TIMESTAMPTZ NULL
);

-- +migrate Down
DROP TABLE "files";