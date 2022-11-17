import sqlalchemy

eng = sqlalchemy.create_engine('postgresql://stackql:stackql@127.0.0.1:5888/stackql')

## this is the sticking point for now
conn = eng.raw_connection()

curs = conn.cursor()

SHOW_TRANSACTION_ISOLATION_LEVEL = "show transaction isolation level"
SELECT_HSTORE_DETAILS = "SELECT t.oid, typarray FROM pg_type t JOIN pg_namespace ns ON typnamespace = ns.oid WHERE typname = 'hstore'"

SHOW_TRANSACTION_ISOLATION_LEVEL_JSON_EXPECTED = [{"transaction_isolation": "read committed"}]
SELECT_HSTORE_DETAILS_JSON_EXPECTED = []

curs.execute("show transaction isolation level")

# rv = curs.fetchall()

if curs.rowcount > 0:
    rv = curs.fetchall()
    e1 = None
    for entry in rv:
        dir(entry)
        print(entry)
        e1 = entry
else:
    print("empty result")
    