INSERT INTO logs.config(id, key, type_id, data) VALUES
  (1, md5(random()::text), 1, jsonb_build_object(
  'channels', 4
    , 'skip',   '\.(js|gif|png|css|ico|jpg|eot)$'
    , 'format', '$remote_addr $user1 $user2 [$time_local] "$request" "$status" $size "$referer" "$user_agent" "$t_size" $fresp $fload $end $request_size $request_id'
    , 'fields', json_build_array('stamp_id', 'file_id', 'line_num', 'time_local', 'remote_addr', 'url', 'args', 'ref_url', 'ref_args', 'referer', 'user_agent', 'method', 'status', 'size', 'fresp', 'fload', 'request_size', 'request_id')
  ))
 ;

SELECT key as "Upload key" FROM logs.config WHERE id = 1;
