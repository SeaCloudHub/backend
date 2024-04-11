
-- +migrate Up
CREATE TABLE IF NOT EXISTS "files"
(
    "id"                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "name"              VARCHAR(255) NOT NULL,
    "path"              TEXT NOT NULL,
    "full_path"         TEXT UNIQUE NOT NULL,
    "previous_path"     TEXT NULL,
    "size"              BIGINT NOT NULL,
    "mode"              BIGINT NOT NULL,
    "mime_type"         VARCHAR(255) NOT NULL,
    "md5"               VARCHAR(32) NOT NULL,
    "is_dir"            BOOLEAN NOT NULL DEFAULT FALSE,
    "general_access"    VARCHAR(255) NOT NULL DEFAULT 'restricted', -- restricted, everyone-can-view, everyone-can-edit
    "owner_id"          UUID NOT NULL,
    "created_at"        TIMESTAMPTZ DEFAULT NOW(),
    "updated_at"        TIMESTAMPTZ DEFAULT NOW(),
    "deleted_at"        TIMESTAMPTZ NULL
);

-- +migrate Down
DROP TABLE "files";