
-- +migrate Up
CREATE OR REPLACE FUNCTION classify_mime_type(mime_type VARCHAR, id_dir BOOLEAN) RETURNS VARCHAR AS $$ BEGIN IF id_dir THEN RETURN 'folder'; ELSE RETURN CASE WHEN mime_type ~ '^text/' THEN 'text' WHEN mime_type ~ '^application/vnd\.openxmlformats-officedocument' THEN 'document' WHEN mime_type ~ '^application/vnd.oasis.opendocument' THEN 'document' WHEN mime_type IN ('application/msword', 'application/vnd.ms-excel', 'application/vnd.ms-powerpoint') THEN 'document' WHEN mime_type = 'application/pdf' THEN 'pdf' WHEN mime_type = 'application/json' THEN 'json' WHEN mime_type ~ '^image/' THEN 'image' WHEN mime_type ~ '^video/' THEN 'video' WHEN mime_type ~ '^audio/' THEN 'audio' WHEN mime_type IN ('application/zip', 'application/x-tar', 'application/x-gzip') THEN 'archive' ELSE 'other' END; END IF; END; $$ LANGUAGE plpgsql IMMUTABLE;

ALTER TABLE files ADD COLUMN type VARCHAR GENERATED ALWAYS AS (classify_mime_type(mime_type, is_dir)) STORED;

-- +migrate Down
ALTER TABLE files DROP COLUMN type;
