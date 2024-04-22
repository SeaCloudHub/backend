
-- +migrate Up
CREATE TEXT SEARCH DICTIONARY english_stem_nostop (
    Template = snowball
    , Language = english
);
CREATE TEXT SEARCH CONFIGURATION public.english_nostop ( COPY = pg_catalog.english );
ALTER TEXT SEARCH CONFIGURATION public.english_nostop
   ALTER MAPPING FOR asciiword, asciihword, hword_asciipart, hword, hword_part, word WITH english_stem_nostop;

ALTER TABLE users ADD COLUMN search_vector tsvector GENERATED ALWAYS AS (to_tsvector('english_nostop', coalesce(REPLACE(REPLACE(email, '@', ' '), '.', ' '), '') || ' ' || coalesce(first_name, '')  || ' ' || coalesce(last_name, ''))) STORED;
CREATE INDEX search_vector_idx ON users USING GIN (search_vector);

-- +migrate Down
ALTER TABLE users DROP COLUMN search_vector;