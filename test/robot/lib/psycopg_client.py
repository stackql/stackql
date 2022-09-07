import psycopg
import typing

from psycopg.rows import dict_row


class PsycoPGClient(object):


  def __init__(self, connection_string :str):
    self._connection_string :str = connection_string
    self._connection = psycopg.connect(
      connection_string, 
      autocommit = True,
      row_factory=dict_row
    )


  def _exec_query(self, query :str) -> typing.List[typing.Dict]:
    r = self._connection.execute(query)
    return r.fetchall()


  def _run_queries(self, queries :typing.List[str]) -> typing.List[typing.Dict]:
    ret_val = []
    for q in queries:
      ret_val += self._exec_query(q)
    return ret_val


  def run_queries(self, queries :typing.List[str]) -> typing.List[typing.Dict]:
    return self._run_queries(queries)

