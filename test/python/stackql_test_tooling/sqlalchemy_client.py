import sqlalchemy
import typing


class SQLAlchemyClient(object):


  def __init__(self, connection_string :str, autocommit :bool = False):
    self._connection_string :str = connection_string
    options = {"isolation_level": "AUTOCOMMIT"} if autocommit else {}
    self._eng = sqlalchemy.create_engine(connection_string, **options)


  def _exec_raw_query(self, query :str) -> typing.List[typing.Dict]:
    r = self._eng.execute(query)
    rv = []
    try:
      for row in r:
        rv.append(row)
    except Exception as err:
      pass
    return rv


  def _run_raw_queries(self, queries :typing.List[str]) -> typing.List[typing.Dict]:
    ret_val = []
    for q in queries:
      nv = self._exec_raw_query(q)
      if nv:
        ret_val += nv
    return ret_val


  def run_raw_queries(self, queries :typing.List[str]) -> typing.List:
    return self._run_raw_queries(queries)

