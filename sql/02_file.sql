
SET SEARCH_PATH = logs,public;

CREATE TABLE log_type (
	id INTEGER PRIMARY KEY
,	code TEXT NOT NULL
);

CREATE TABLE config (
	id INTEGER PRIMARY KEY
,	key TEXT NOT NULL UNIQUE
,	type_id INTEGER NOT NULL REFERENCES log_type(id)
,	data JSONB NOT NULL
);

CREATE SEQUENCE stamp_seq;
CREATE TABLE stamp (
  id INTEGER PRIMARY KEY DEFAULT nextval('stamp_seq')
, data TIMESTAMP(0) NOT NULL UNIQUE
);

CREATE SEQUENCE file_seq;

CREATE TABLE file (
  id        INTEGER PRIMARY KEY DEFAULT nextval('file_seq')
, config_id INTEGER NOT NULL REFERENCES config(id)
, type_id   INTEGER NOT NULL REFERENCES log_type(id)
, stamp_id  INTEGER REFERENCES stamp(id)
, filename  TEXT NOT NULL
, first     INTEGER
, last      INTEGER
, total     INTEGER DEFAULT 0 -- updated while loading
, loaded    INTEGER DEFAULT 0 -- updated while loading
, skipped   INTEGER
, error     TEXT
, begin_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
, end_at    TIMESTAMP
);

INSERT INTO log_type(id, code) VALUES
  (1, 'nginx')
;

CREATE OR REPLACE FUNCTION file_before(
  a_type_id INTEGER
, a_config_id INTEGER
, a_filename TEXT
) RETURNS INTEGER LANGUAGE 'plpgsql' AS $_$
DECLARE
  v_id INTEGER;
BEGIN
  INSERT INTO logs.file (type_id, config_id, filename) VALUES
    (a_type_id, a_config_id, a_filename)
    RETURNING id INTO v_id
  ;
  RETURN v_id;
END
$_$;

-- logs.stamp_register(a_file_id => $1, a_stamp => $2)

CREATE OR REPLACE FUNCTION stamp_register(
  a_stamp TIMESTAMP
, a_file_id INTEGER
) RETURNS INTEGER LANGUAGE 'plpgsql' AS $_$
DECLARE
  v_id INTEGER;
BEGIN
  INSERT INTO logs.stamp (data) VALUES (a_stamp)
    ON CONFLICT DO NOTHING
    RETURNING id INTO v_id
  ;
  IF NOT FOUND THEN
    SELECT INTO v_id id
    FROM logs.stamp WHERE data = a_stamp;
  END IF;
  UPDATE logs.file SET stamp_id = v_id WHERE id = a_file_id;
  RETURN v_id;
END
$_$;

CREATE OR REPLACE FUNCTION logs.file_update_stat(
  a_id INTEGER
, a_total INTEGER
, a_loaded INTEGER
-- TODO: ? add stamp
) RETURNS VOID LANGUAGE 'sql' AS $_$
-- TODO: NOTIFY
UPDATE logs.file SET
  total = total + a_total
, loaded = loaded + a_loaded
WHERE id = a_id
$_$;
