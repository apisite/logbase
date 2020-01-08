SET SEARCH_PATH = logs,public;

CREATE SEQUENCE url_seq;
CREATE TABLE url (
	id INTEGER PRIMARY KEY DEFAULT nextval('url_seq')
,	data TEXT NOT NULL
,	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
,	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE SEQUENCE args_seq;
CREATE TABLE args (
	id INTEGER PRIMARY KEY DEFAULT nextval('args_seq')
,	data TEXT NOT NULL
,	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
,	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE SEQUENCE referer_seq;
CREATE TABLE referer (
	id INTEGER PRIMARY KEY DEFAULT nextval('referer_seq')
,	data TEXT NOT NULL
,	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
,	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE SEQUENCE agent_seq;
CREATE TABLE agent (
	id INTEGER PRIMARY KEY DEFAULT nextval('agent_seq')
,	data TEXT NOT NULL
,	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
,	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE url SET UNLOGGED;
ALTER TABLE args SET UNLOGGED;
ALTER TABLE referer SET UNLOGGED;
ALTER TABLE agent SET UNLOGGED;

CREATE UNLOGGED TABLE request_data (
  id INTEGER NOT NULL REFERENCES file(id)
, stamp TIMESTAMP
, url_id INTEGER REFERENCES url(id)
, args_id INTEGER REFERENCES args(id)
, is_ssl BOOL NOT NULL DEFAULT FALSE

-- internal referer
, ref_url_id INTEGER REFERENCES url(id)
, ref_args_id INTEGER REFERENCES args(id)

-- external referer
, referer_id INTEGER REFERENCES referer(id)

, addr INET -- remote_addr = 37.9.113.95
, agent_id INTEGER REFERENCES agent(id)
, method TEXT
, status INTEGER
, size INTEGER NOT NULL
, fresp NUMERIC
, fload NUMERIC
--proto = HTTP/1.1
--t_size = -
, CONSTRAINT request_data_pkey PRIMARY KEY (stamp,addr,url_id,args_id)
);

CREATE OR REPLACE VIEW request AS
  SELECT rd.*
,  u.data as url
,  ar.data as args
,  u1.data as ref_url
,  ar1.data as ref_args
,  ref.data as referer
,  ag.data as agent
FROM request_data rd
JOIN url u ON(rd.url_id=u.id)
LEFT OUTER JOIN args ar ON(rd.args_id=ar.id)
LEFT OUTER JOIN url u1 ON(rd.ref_url_id=u1.id)
LEFT OUTER JOIN args ar1 ON(rd.ref_args_id=ar1.id)
LEFT OUTER JOIN referer ref ON(rd.referer_id=ref.id)
LEFT OUTER JOIN agent ag ON(rd.agent_id=ag.id)
;

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
  v_args_a TEXT;
  v_ref_args_a TEXT;
  v_referer INTEGER;
  v_url INTEGER;
  v_args INTEGER;
  v_ref_url INTEGER;
  v_ref_args INTEGER;
  v_agent INTEGER;
begin

-- insert into logs.agent(id,data) values(2535,'xxx') on conflict on constraint agent_pkey do update set id=agent.id returning id;

	v_url_a := COALESCE(a_url, '');
  SELECT INTO v_url id FROM logs.url WHERE data = v_url_a;
  IF NOT FOUND THEN
    INSERT INTO logs.url (data) VALUES (v_url_a) RETURNING id INTO v_url;
  END IF;
	v_args_a := COALESCE(a_args, '');
	  SELECT INTO v_args id FROM logs.args WHERE data = v_args_a;
	  IF NOT FOUND THEN
	    INSERT INTO logs.args (data) VALUES (v_args_a) RETURNING id INTO v_args;
	  END IF;
	IF a_agent <> '-' THEN
	  SELECT INTO v_agent id FROM logs.agent WHERE data = a_agent;
	  IF NOT FOUND THEN
	    INSERT INTO logs.agent (data) VALUES (a_agent) RETURNING id INTO v_agent;
	  END IF;
  END IF;
	IF COALESCE(a_referer,'-') <> '-' THEN
	  SELECT INTO v_referer id FROM logs.referer WHERE data = a_referer;
	  IF NOT FOUND THEN
	    INSERT INTO logs.referer (data) VALUES (a_referer) RETURNING id INTO v_referer;
	  END IF;
  ELSIF a_ref_url IS NOT NULL THEN
	  SELECT INTO v_ref_url id FROM logs.url WHERE data = a_ref_url;
	  IF NOT FOUND THEN
	    INSERT INTO logs.url (data) VALUES (a_ref_url) RETURNING id INTO v_ref_url;
	  END IF;
		v_ref_args_a := COALESCE(a_ref_args, '');
	  SELECT INTO v_ref_args id FROM logs.args WHERE data = v_ref_args_a;
	  IF NOT FOUND THEN
	    INSERT INTO logs.args (data) VALUES (v_ref_args_a) RETURNING id INTO v_ref_args;
	  END IF;
  END IF;

  INSERT INTO logs.request_data (
  stamp 	
, id
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

/*
  TODO
  ф-я по завершении загрузки файла
  * сохраняет отдельно статистику загрузки
  * обновляет (url,args,referer).updated_at, чтобы можно было отрезать устаревшие строки
*/

/*
-- show unlogged:
SELECT relname FROM pg_class WHERE relpersistence = 'u';
*/
