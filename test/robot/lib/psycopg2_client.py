import psycopg2
import typing

from psycopg2.extras import RealDictCursor


class PsycoPG2Client(object):


  def __init__(self, connection_string :str):
    self._connection_string :str = connection_string
    self._connection = psycopg2.connect(
      connection_string
    )
    self._connection.set_session(autocommit=True)


  def _exec_query(self, query :str) -> typing.List[typing.Dict]:
    cur = self._connection.cursor(cursor_factory=RealDictCursor)
    cur.execute(query)
    return [ dict(b) for b in cur.fetchall() ]


  def _run_queries(self, queries :typing.List[str]) -> typing.List[typing.Dict]:
    ret_val = []
    for q in queries:
      ret_val += self._exec_query(q)
    return ret_val


  def run_queries(self, queries :typing.List[str]) -> typing.List[typing.Dict]:
    return self._run_queries(queries)

