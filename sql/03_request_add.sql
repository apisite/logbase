
CREATE OR REPLACE FUNCTION request_add(
  a_stamp TIMESTAMP
, a_file_id INTEGER
, a_addr INET
, a_url TEXT
, a_args TEXT

-- internal referer
, a_ref_url TEXT
, a_ref_args TEXT
-- external referer
, a_referer TEXT

, a_agent TEXT
, a_method TEXT
, a_status INTEGER
, a_size INTEGER
, a_fresp NUMERIC
, a_fload NUMERIC
) RETURNS BOOL LANGUAGE plpgsql AS $_$
declare
  v_url_a TEXT;
  v_args_a JSONB;
  v_ref_args_a JSONB;
  v_referer_a TEXT;
  v_referer INTEGER;
  v_url INTEGER;
  v_args INTEGER;
  v_ref_url INTEGER;
  v_ref_args INTEGER;
  v_agent INTEGER;
begin
  v_url_a := COALESCE(a_url, '');
  INSERT INTO logs.url (data, created_at, updated_at) VALUES (v_url_a, a_stamp, a_stamp) 
    ON CONFLICT DO NOTHING
    RETURNING id INTO v_url;
  IF NOT FOUND THEN
    SELECT INTO v_url id FROM logs.url WHERE data = v_url_a;
  END IF;

  v_args_a := COALESCE(a_args, '{}')::JSONB;
  INSERT INTO logs.args (data, created_at, updated_at) VALUES (v_args_a, a_stamp, a_stamp) 
    ON CONFLICT DO NOTHING
    RETURNING id INTO v_args;
  IF NOT FOUND THEN
    SELECT INTO v_args id FROM logs.args WHERE data = v_args_a;
  END IF;

	IF a_agent <> '-' THEN
    INSERT INTO logs.agent (data, created_at, updated_at) VALUES (a_agent, a_stamp, a_stamp) 
      ON CONFLICT DO NOTHING
      RETURNING id INTO v_agent;
    IF NOT FOUND THEN
      SELECT INTO v_agent id FROM logs.agent WHERE data = a_agent;
    END IF;
  END IF;

	IF COALESCE(a_referer,'-') <> '-' THEN
  v_referer_a := a_referer::VARCHAR(2048); --  limit length
    INSERT INTO logs.referer (data, created_at, updated_at) VALUES (v_referer_a, a_stamp, a_stamp) 
      ON CONFLICT DO NOTHING
      RETURNING id INTO v_referer;
    IF NOT FOUND THEN
      SELECT INTO v_referer id FROM logs.referer WHERE data = v_referer_a;
    END IF;
  ELSIF a_ref_url IS NOT NULL THEN
    INSERT INTO logs.url (data, created_at, updated_at) VALUES (a_ref_url, a_stamp, a_stamp) 
      ON CONFLICT DO NOTHING
      RETURNING id INTO v_ref_url;
    IF NOT FOUND THEN
      SELECT INTO v_ref_url id FROM logs.url WHERE data = a_ref_url;
    END IF;
    
		v_ref_args_a := COALESCE(a_ref_args, '{}')::JSONB;
    INSERT INTO logs.args (data, created_at, updated_at) VALUES (v_ref_args_a, a_stamp, a_stamp) 
      ON CONFLICT DO NOTHING
      RETURNING id INTO v_ref_args;
    IF NOT FOUND THEN
      SELECT INTO v_ref_args id FROM logs.args WHERE data = v_ref_args_a;
    END IF;
  END IF;

  INSERT INTO logs.request_data (
  stamp 	
, file_id
, addr 		
, url_id 		
, args_id
, referer_id
, ref_url_id
, ref_args_id
, agent_id
, method 
, status 
, size 		
, fresp 	
, fload 	
  ) VALUES (
  a_stamp 	
, a_file_id  
, a_addr 		
, v_url
, v_args 		
, v_referer
, v_ref_url
, v_ref_args
, v_agent 	
, nullif(a_method, 'GET') 
, nullif(a_status, 200) 
, a_size 		
, a_fresp 	
, a_fload 	  
  )
  ON CONFLICT DO NOTHING;
  RETURN FOUND; -- true means 'inserted'
end
$_$;

