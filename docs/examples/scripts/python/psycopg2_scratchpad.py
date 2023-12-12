import importlib.util
import sys
import os

_THIS_DIR = os.path.abspath(os.path.dirname(__file__))

_REPOSITORY_ROOT = os.path.abspath(
  os.path.join(
    _THIS_DIR,
    "..",
    "..",
    ".."
  )
)


spec = importlib.util.spec_from_file_location(
  "module.name", 
  os.path.join(
    _REPOSITORY_ROOT, 
    "test/robot/lib/psycopg2_client.py"
  )
)
foo = importlib.util.module_from_spec(spec)
sys.modules["module.name"] = foo
spec.loader.exec_module(foo)

psycopg2_client = foo.PsycoPG2Client("host=127.0.0.1 port=5466 user=silly dbname=silly")

rv = psycopg2_client.run_queries(["show transaction isolation level"])

print(rv)

rv = psycopg2_client.run_queries(["SELECT t.oid, typarray FROM pg_type t JOIN pg_namespace ns ON typnamespace = ns.oid WHERE typname = 'hstore'"])

print(rv)
