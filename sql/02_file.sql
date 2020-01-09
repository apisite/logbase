
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

CREATE SEQUENCE file_seq;

CREATE TABLE file (
  id        INTEGER PRIMARY KEY DEFAULT nextval('file_seq')
, config_id INTEGER NOT NULL REFERENCES config(id)
, type_id   INTEGER NOT NULL REFERENCES log_type(id)
, filename  TEXT NOT NULL
, begin_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
, end_at    TIMESTAMP
, total     INTEGER
, loaded    INTEGER
, skipped   INTEGER
, error     TEXT
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

