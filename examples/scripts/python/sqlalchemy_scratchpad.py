import sqlalchemy

eng = sqlalchemy.create_engine('postgresql://sillyuser:sillypw@127.0.0.1:5466/sillydb')

## this is the sticking point for now
conn = eng.raw_connection()

curs = conn.cursor()

curs.execute("show transaction isolation level")

curs.fetchall()