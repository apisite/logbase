
INSERT INTO logs.config(id, key, type_id, data) VALUES
  (1, md5(random()::text), 1, jsonb_build_object(
  'channels', 4
    , 'skip',   '\.(js|gif|png|css|ico|jpg|eot)$'
    , 'format', '$remote_addr - $remote_user [$time_local] "$request" '
            ||    '$status $size "$referer" '
            ||    '"$user_agent" "$forwarded_for" '
            ||    'rt=$request_time uct="$fload" uht="$upstream_header_time" urt="$fresp"'
-- log_format: ||    'rt=$request_time uct="$upstream_connect_time" uht="$upstream_header_time" urt="$upstream_response_time"'
    , 'fields', json_build_array('stamp_id', 'file_id', 'line_num', 'time_local', 'remote_addr', 'url', 'args', 'ref_url'
               , 'ref_args', 'referer', 'user_agent', 'method', 'status', 'size', 'fresp', 'fload')
  ))
 ;

SELECT key as "Upload key" FROM logs.config WHERE id = 1;
