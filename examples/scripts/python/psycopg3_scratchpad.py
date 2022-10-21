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
    "test/robot/lib/psycopg_client.py"
  )
)
foo = importlib.util.module_from_spec(spec)
sys.modules["module.name"] = foo
spec.loader.exec_module(foo)

psycopg3_client = foo.PsycoPGClient("host=127.0.0.1 port=5888 user=admin dbname=postgres")

table_name = "okta.application.Application.generation_1"

SELECT_OKTA_APPS = "select name, status, label, id from okta.application.apps apps where apps.subdomain = 'example-subdomain' order by name asc;"
NATIVEQUERY_OKTA_APPS_ROW_COUNT = f"NATIVEQUERY 'SELECT COUNT(*) as object_count FROM \"{table_name}\"' ;"
PURGE_CONSERVATIVE = "purge conservative;"


rv = psycopg3_client.run_queries([
  SELECT_OKTA_APPS, 
  NATIVEQUERY_OKTA_APPS_ROW_COUNT, 
  PURGE_CONSERVATIVE, 
  NATIVEQUERY_OKTA_APPS_ROW_COUNT, 
  SELECT_OKTA_APPS, 
  SELECT_OKTA_APPS, 
  NATIVEQUERY_OKTA_APPS_ROW_COUNT, 
  PURGE_CONSERVATIVE, 
  NATIVEQUERY_OKTA_APPS_ROW_COUNT
])

print(rv)

rv = psycopg3_client.run_queries(["SELECT t.oid, typarray FROM pg_type t JOIN pg_namespace ns ON typnamespace = ns.oid WHERE typname = 'hstore'"])

print(rv)
