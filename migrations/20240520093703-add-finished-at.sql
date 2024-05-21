
-- +migrate Up
ALTER TABLE "files" ADD COLUMN "finished_at" TIMESTAMPTZ NULL DEFAULT NOW();
UPDATE "files" SET "finished_at" = "created_at";

-- +migrate Down
ALTER TABLE "files" DROP COLUMN "finished_at";