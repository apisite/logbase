
CREATE OR REPLACE FUNCTION request_add(
  a_stamp_id INTEGER
, a_file_id INTEGER
, a_line_num INTEGER
, a_time_local TIMESTAMP
, a_remote_addr INET
, a_url TEXT
, a_args TEXT

-- internal referer
, a_ref_url TEXT
, a_ref_args TEXT
-- external referer
, a_referer TEXT

, a_user_agent TEXT
, a_method TEXT
, a_status INTEGER
, a_size INTEGER
, a_fresp NUMERIC
, a_fload NUMERIC DEFAULT NULL
, a_request_size INTEGER DEFAULT NULL
, a_request_id TEXT DEFAULT NULL
) RETURNS BOOL LANGUAGE plpgsql AS $_$
declare
  v_url_id INTEGER;
  v_args_id INTEGER;
  v_referer_id INTEGER;
  v_ref_url_id INTEGER;
  v_ref_args_id INTEGER;
  v_agent_id INTEGER;
  v_referer TEXT;
begin
  IF a_url IS NOT NULL THEN
    INSERT INTO logs.url (data, created_at, updated_at) VALUES (a_url, a_time_local, a_time_local)
      ON CONFLICT DO NOTHING
      RETURNING id INTO v_url_id;
    IF NOT FOUND THEN
      SELECT INTO v_url_id id FROM logs.url WHERE data = a_url;
    END IF;
  END IF;

  IF a_args IS NOT NULL THEN
    INSERT INTO logs.args (data, created_at, updated_at) VALUES (a_args::JSONB, a_time_local, a_time_local)
      ON CONFLICT DO NOTHING
      RETURNING id INTO v_args_id;
    IF NOT FOUND THEN
      SELECT INTO v_args_id id FROM logs.args WHERE data = a_args::JSONB;
    END IF;
  END IF;

	IF a_user_agent <> '-' THEN
    INSERT INTO logs.agent (data, created_at, updated_at) VALUES (a_user_agent, a_time_local, a_time_local)
      ON CONFLICT DO NOTHING
      RETURNING id INTO v_agent_id;
    IF NOT FOUND THEN
      SELECT INTO v_agent_id id FROM logs.agent WHERE data = a_user_agent;
    END IF;
  END IF;

	IF COALESCE(a_referer,'-') <> '-' THEN
    v_referer := a_referer::VARCHAR(2048); --  limit length
    INSERT INTO logs.referer (data, created_at, updated_at) VALUES (v_referer, a_time_local, a_time_local)
      ON CONFLICT DO NOTHING
      RETURNING id INTO v_referer_id;
    IF NOT FOUND THEN
      SELECT INTO v_referer_id id FROM logs.referer WHERE data = v_referer;
    END IF;
  ELSIF a_ref_url IS NOT NULL THEN
    INSERT INTO logs.url (data, created_at, updated_at) VALUES (a_ref_url, a_time_local, a_time_local)
      ON CONFLICT DO NOTHING
      RETURNING id INTO v_ref_url_id;
    IF NOT FOUND THEN
      SELECT INTO v_ref_url_id id FROM logs.url WHERE data = a_ref_url;
    END IF;
    IF a_ref_args IS NOT NULL THEN
      INSERT INTO logs.args (data, created_at, updated_at) VALUES (a_ref_args::JSONB, a_time_local, a_time_local)
        ON CONFLICT DO NOTHING
        RETURNING id INTO v_ref_args_id;
      IF NOT FOUND THEN
        SELECT INTO v_ref_args_id id FROM logs.args WHERE data = a_ref_args::JSONB;
      END IF;
    END IF;
  END IF;

  INSERT INTO logs.request_data (
    stamp_id
  , file_id
  , line_num
  , stamp
  , url_id
  , args_id
  , referer_id
  , ref_url_id
  , ref_args_id
  , agent_id
  , method
  , status
  , addr
  , size
  , fresp
  , fload
  , request_size
  , request_id
  ) VALUES (
    a_stamp_id
  , a_file_id
  , a_line_num
  , a_time_local
  , v_url_id
  , v_args_id
  , v_referer_id
  , v_ref_url_id
  , v_ref_args_id
  , v_agent_id
  , nullif(a_method, 'GET')
  , nullif(a_status, 200)
  , a_remote_addr
  , a_size
  , a_fresp
  , a_fload
  , a_request_size
  , a_request_id
  )
  ON CONFLICT DO NOTHING
;
  RETURN FOUND; -- true means 'inserted'
end
$_$;
