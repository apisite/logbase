SET SEARCH_PATH = logs,public;

CREATE SEQUENCE url_seq;
CREATE TABLE url (
	id INTEGER PRIMARY KEY DEFAULT nextval('url_seq')
,	data TEXT NOT NULL UNIQUE
,	created_at TIMESTAMP NOT NULL
,	updated_at TIMESTAMP NOT NULL
);

CREATE SEQUENCE args_seq;
CREATE TABLE args (
	id INTEGER PRIMARY KEY DEFAULT nextval('args_seq')
,	data JSONB NOT NULL UNIQUE
,	created_at TIMESTAMP NOT NULL
,	updated_at TIMESTAMP NOT NULL
);

CREATE SEQUENCE referer_seq;
CREATE TABLE referer (
	id INTEGER PRIMARY KEY DEFAULT nextval('referer_seq')
,	data TEXT NOT NULL UNIQUE
,	created_at TIMESTAMP NOT NULL
,	updated_at TIMESTAMP NOT NULL
);

CREATE SEQUENCE agent_seq;
CREATE TABLE agent (
	id INTEGER PRIMARY KEY DEFAULT nextval('agent_seq')
,	data TEXT NOT NULL UNIQUE
,	created_at TIMESTAMP NOT NULL
,	updated_at TIMESTAMP NOT NULL
);

/*
ALTER TABLE url SET UNLOGGED;
ALTER TABLE args SET UNLOGGED;
ALTER TABLE referer SET UNLOGGED;
ALTER TABLE agent SET UNLOGGED;
*/

CREATE /* UNLOGGED */ TABLE request_data (
  file_id INTEGER REFERENCES file(id)
, stamp TIMESTAMP
, url_id INTEGER REFERENCES url(id)
, args_id INTEGER REFERENCES args(id)
, is_ssl BOOL NOT NULL DEFAULT FALSE -- TODO: Use it

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
, CONSTRAINT request_data_pkey PRIMARY KEY (file_id, stamp, addr, url_id, args_id)
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
JOIN url u ON(rd.url_id = u.id)     -- pkey, not null
JOIN args ar ON(rd.args_id = ar.id) -- pkey, not null
LEFT OUTER JOIN url u1 ON(rd.ref_url_id = u1.id)
LEFT OUTER JOIN args ar1 ON(rd.ref_args_id = ar1.id)
LEFT OUTER JOIN referer ref ON(rd.referer_id = ref.id)
LEFT OUTER JOIN agent ag ON(rd.agent_id = ag.id)
;
