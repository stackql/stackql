
-- 'C' collation ensures parity with sqlite text ordering
CREATE database "stackql" LC_COLLATE 'C' LC_CTYPE 'C' template template0;

CREATE user stackql with password 'stackql';

CREATE user stackql_intel with password 'stackql';

CREATE user stackql_ops with password 'stackql';

GRANT ALL PRIVILEGES on DATABASE stackql to stackql;

\connect stackql;

CREATE schema stackql_raw;

CREATE schema stackql_control;

CREATE schema stackql_intel;

CREATE schema stackql_ops;

GRANT ALL PRIVILEGES on SCHEMA stackql_raw to stackql;

GRANT ALL PRIVILEGES on SCHEMA stackql_control to stackql;

GRANT ALL PRIVILEGES on SCHEMA stackql_intel to stackql;

GRANT ALL PRIVILEGES on SCHEMA stackql_ops to stackql;

GRANT ALL PRIVILEGES on SCHEMA stackql_intel to stackql_intel;

GRANT ALL PRIVILEGES on SCHEMA stackql_ops to stackql_ops;
