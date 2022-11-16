import sqlalchemy
import typing


class SQLAlchemyClient(object):


  def __init__(self, connection_string :str):
    self._connection_string :str = connection_string
    self._eng = sqlalchemy.create_engine(connection_string)


  def _exec_raw_query(self, query :str) -> typing.List[typing.Dict]:
    conn = self._eng.raw_connection()
    curs = conn.cursor()
    r = self._eng.execute(query)
    return [ b for b in r.fetchall() ]


  def _run_raw_queries(self, queries :typing.List[str]) -> typing.List[typing.Dict]:
    ret_val = []
    for q in queries:
      nv = self._exec_raw_query(q)
      if nv:
        ret_val += nv
    return ret_val


  def run_raw_queries(self, queries :typing.List[str]) -> typing.List:
    return self._run_raw_queries(queries)

