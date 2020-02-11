
CREATE OR REPLACE VIEW logs.request_at AS
SELECT
  stamp
, stamp+(fresp||' sec')::interval as fin
, url
,fresp
,size
,request_size
,status
FROM logs.request
ORDER BY stamp
;

-- Show all requests active at timestamp
CREATE OR REPLACE FUNCTION logs.nginx_at(
  a_stamp TIMESTAMP
) RETURNS SETOF logs.request_at LANGUAGE sql AS
$_$
 SELECT *
   FROM logs.request_at
   WHERE stamp < a_stamp
   AND fin > a_stamp
$_$;
