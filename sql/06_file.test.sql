begin;

SAVEPOINT test_begin;

-- select pgmig.assert_eq('first', 1, 1);

INSERT INTO logs.config(id, type_id, key, data) VALUES (1, 1, 'testkey','{}');

select logs.file_after(
  logs.file_before(
    1
  , 1
  , 'test.file'
  )
, 5
, 3
, 2
, 10
, 20
, 'err'
);

select * from logs.file;


ROLLBACK TO SAVEPOINT test_begin;

-- rollback;
