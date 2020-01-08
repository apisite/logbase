INSERT INTO logs.config(id, key, type_id, data) VALUES
  (1, md5(random()::text), 1, jsonb_build_object(
  'channels', 4
    , 'skip',   '\.(js|gif|png|css|ico|jpg|eot)$'
    , 'format', '$remote_addr $user1 $user2 [$time_local] "$request" "$status" $size "$referer" "$user_agent" "$t_size" $fresp $fload $end'
  ))
 ;

SELECT key as "Upload key" FROM logs.config WHERE id = 1;
