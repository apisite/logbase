CREATE OR REPLACE FUNCTION logs.url_update(
  a_file_id INTEGER
) RETURNS VOID LANGUAGE 'sql' AS $_$
  with stamps as (
    select url_id, min(stamp) as mi, max(stamp) as ma
    from logs.request_data where file_id = a_file_id
    group by url_id
  )
  update logs.url SET
    created_at = CASE WHEN created_at > stamps.mi THEN stamps.mi ELSE created_at END
  , updated_at = CASE WHEN updated_at < stamps.ma THEN stamps.ma ELSE updated_at END
  FROM stamps
  WHERE url.id = stamps.url_id
  ;
$_$;

CREATE OR REPLACE FUNCTION logs.file_after(
  a_id INTEGER
, a_total INTEGER
, a_loaded INTEGER
, a_skipped INTEGER
, a_first INTEGER
, a_last INTEGER
, a_error TEXT
) RETURNS BOOL LANGUAGE 'plpgsql' AS $_$
BEGIN
  -- update logs.url timestamps
  -- TODO: do the same for others
  PERFORM logs.url_update(a_id);

  UPDATE logs.file SET
    end_at = now()
  , total = a_total
  , loaded = a_loaded
  , skipped = a_skipped
  , first = a_first
  , last = a_last
  , error = a_error
  WHERE id = a_id
  ;
  RETURN FOUND;
END
$_$;