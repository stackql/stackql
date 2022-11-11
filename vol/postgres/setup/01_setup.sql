
-- 'C' collation ensures parity with sqlite text ordering
CREATE database "stackql" LC_COLLATE 'C' LC_CTYPE 'C' template template0;

CREATE user stackql with password 'stackql';

GRANT ALL PRIVILEGES on DATABASE stackql to stackql;
