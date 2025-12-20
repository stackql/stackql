

# Analytics with stackql

The canonical pattern is a postgres backend.  To meaningfully develop analytics capability, **real** authenticated access to providers plus a postgres backend is needed. Therefore for local development:

- Ensure that all env var secrets are exported from the `.gitignore`d file `cicd/vol/vendor-secrets/secrets.sh`.
- Run and kill development containers with `docker compose -f docker-compose-live.yml down --volumes` / `docker compose -f docker-compose-live.yml up --force-recreate`.
- Connect and develop queries with `psql "postgresql://stackql:stackql@127.0.0.1:8632/stackql"`.


## TODO

Robot tests for:

- Support for `current_date`.
- Support for `current_timestamp`.
- Support for multi-layered table valued functions in subqueries with outside filters, per Figure MLS-01.

---

```sql

-- sqlite version

CREATE OR REPLACE MATERIALIZED VIEW gcp_compute_public_ip_exposure AS
select
  resource_type,
  resource_id,
  resource_name,
  cloud,
  region,
  protocol,
  from_port,
  to_port,
  cidr,
  direction,
  public_access_type,
  public_principal,
  access_mechanism
from 
(
SELECT
  'compute'                          AS resource_type,
  vms.id                                 AS resource_id,
  vms.name                               AS resource_name,
  'google'                              AS cloud,
  split_part(vms.zone, '/', -1)      AS region,
  NULL                               AS protocol,
  NULL                               AS from_port,
  NULL                               AS to_port,
  NULL                               AS cidr,
  NULL                               AS direction,
  NULL                               AS public_access_type,
  NULL                               AS public_principal,
  NULL                               AS access_mechanism,
  json_extract(ac.value, '$.natIP') as external_ip
FROM google.compute.instances vms,
  json_each(vms.networkInterfaces) AS ni,
  json_each(json_extract(ni.value, '$.accessConfigs')) AS ac
WHERE 
  vms.project in (
    'testing-project'
  )
  ) foo
  where external_ip != ''
;


-- postgres version

CREATE OR REPLACE MATERIALIZED VIEW gcp_compute_public_ip_exposure AS
select
  resource_type,
  resource_id,
  resource_name,
  cloud,
  region,
  protocol,
  from_port,
  to_port,
  cidr,
  direction,
  public_access_type,
  public_principal,
  access_mechanism
from 
(
SELECT
  'compute'                          AS resource_type,
  vms.id                                 AS resource_id,
  vms.name                               AS resource_name,
  'google'                              AS cloud,
  split_part(vms.zone, '/', -1)      AS region,
  NULL                               AS protocol,
  NULL                               AS from_port,
  NULL                               AS to_port,
  NULL                               AS cidr,
  NULL                               AS direction,
  NULL                               AS public_access_type,
  NULL                               AS public_principal,
  NULL                               AS access_mechanism,
  json_extract_path_text(ac.value, 'natIP') as external_ip
FROM google.compute.instances vms,
  json_array_elements_text(vms.networkInterfaces) AS ni,
  json_array_elements_text(json_extract_path_text(ni.value, 'accessConfigs')) AS ac
WHERE 
  vms.project in (
    'stackql-interesting'
  )
  ) foo
  where external_ip != ''
;

```

**Figure MLS-01**: Multi-layered table valued functions in subqueries with outside filters.

---

