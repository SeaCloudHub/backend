-- +migrate Up
CREATE TABLE IF NOT EXISTS "books"
(
    "id"         SERIAL PRIMARY KEY,
    "isbn"       VARCHAR(255),
    "name"       TEXT,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +migrate Down
DROP TABLE "books";