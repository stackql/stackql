
## Running the MCP server

If necessary, rebuild stackql with:

```bash
python cicd/python/build.py --build
```

**Note**: before starting an MCP server, remember to export all appropriate auth env vars.

We have a nice debug config for running an MCP server with `vscode`, please see [the `vscode` debug launch config](/.vscode/launch.json) for that.  Otherwise, you can run with stackql (assuming locally built into `./build/stackql`):


```bash

./build/stackql mcp --mcp.server.type=http --mcp.config '{"server": {"transport": "http", "address": "127.0.0.1:9992"} }'


```


## Using the MCP Client

This is very much a development tool, not currently recommended for production.  That said, it emulates agent actions and allows us to run regression tests.

Build:

```bash
python cicd/python/build.py --build-mcp-client
```

Then, assuming you have a `stackql` MCP server serving streamable HTTP on port `9992`, you can access any edpoint.  The below examples are somewhat illustrative of a canonical agent pattern for agent behaviour.


```bash

## List available providers.
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      list_providers

## List available services.  
## **must** supply <provider>
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      list_services --exec.args '{"provider": "google"}'

## List available resources.  
## **must** supply <provider>, <service>
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      list_resources --exec.args '{"provider": "google", "service": "compute"}'

## List access methods.  
## **must** supply <provider>, <service>, <resource>
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      list_methods --exec.args '{"provider": "google", "service": "compute", "resource": "networks"}'

## Describe published relation
## **must** supply <provider>, <service>, <resource>
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      meta_describe_table --exec.args '{"provider": "google", "service": "compute", "resource": "networks"}'

## Validate query AOT.  Only works for SELECT at this stage.
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action validate_query_json_v2      --exec.args '{"sql": "select name from google.compute.networks where project = '"'"'stackql-demo'"'"';"}'

## Run query
./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action query_json_v2      --exec.args '{"sql": "select name from google.compute.networks where project = '"'"'stackql-demo'"'"';"}'

## Exec query pattern; for non-read operations
## Tread carefully!!!
## These are almost always mutations
# /build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action exec_query_json_v2      --exec.args '{"sql": "delete from google.compute.networks where project = '"'"'stackql-demo'"'"' and network = '"'"'returning-test-03'"'"' ;"}'

```


## Canonical agent tools

All return `json` responses if any.

- `list_providers` -> List (locally) available providers.  Top of the `stackql` hierarchy.  No request data needed. 
- `list_services` -> List out sertives in a given provider. **must** supply <provider>.
- `list_resources` -> List out resources. **must** supply <provider>, <service>.
- `list_methods` -> List out access methods; invaluable for inference of required WHERE parameters. **must** supply <provider>, <service>, <resource>.
- `meta_describe_table` -> Describe published relation; useful for getting projections right. **must** supply <provider>, <service>, <resource>.
- `validate_query_json_v2` -> AOT validation of SELECT queries.  Returns a json object including `"result": "OK"` is successful.  No response / error / response lackig OK if not successful.
- `query_json_v2` -> Run SQL query and receive json rendition of result set.
- `exec_query_json_v2` -> Exec query pattern; for non-read operations. Tread carefully!!! These are almost always mutations.  At present no AOT validation supported.

Example:

```bash

$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      list_providers 2>/dev/null | jq
{
  "rows": [
    {
      "name": "aws",
      "version": "v24.07.00246"
    },
    {
      "name": "azure",
      "version": "v24.06.00242"
    },
    {
      "name": "datadog",
      "version": "v00.00.00000"
    },
    {
      "name": "deno",
      "version": "v25.09.00347"
    },
    {
      "name": "digitalocean",
      "version": "v24.11.00274"
    },
    {
      "name": "github",
      "version": "v25.07.00320"
    },
    {
      "name": "google",
      "version": "v25.11.00355"
    },
    {
      "name": "googleadmin",
      "version": "v23.07.00153"
    }
  ],
  "row_count": 8,
  "format": "json"
}

$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      list_services --exec.args '{"provider": "google"}' 2>/dev/null | jq
{
  "rows": [
    {
      "id": "accessapproval:v25.11.00355",
      "name": "accessapproval",
      "title": "Access Approval API"
    },
    {
      "id": "accesscontextmanager:v25.11.00355",
      "name": "accesscontextmanager",
      "title": "Access Context Manager API"
    },
    {
      "id": "addressvalidation:v25.11.00355",
      "name": "addressvalidation",
      "title": "Address Validation API"
    },
    {
      "id": "advisorynotifications:v25.11.00355",
      "name": "advisorynotifications",
      "title": "Advisory Notifications API"
    },
    {
      "id": "aiplatform:v25.11.00355",
      "name": "aiplatform",
      "title": "Vertex AI API"
    },
    {
      "id": "airquality:v25.11.00355",
      "name": "airquality",
      "title": "Air Quality API"
    },
    {
      "id": "alloydb:v25.11.00355",
      "name": "alloydb",
      "title": "AlloyDB API"
    },
    {
      "id": "analyticshub:v25.11.00355",
      "name": "analyticshub",
      "title": "Analytics Hub API"
    },
    {
      "id": "apigateway:v25.11.00355",
      "name": "apigateway",
      "title": "API Gateway API"
    },
    {
      "id": "apigee:v25.11.00355",
      "name": "apigee",
      "title": "Apigee API"
    },
    {
      "id": "apigeeregistry:v25.11.00355",
      "name": "apigeeregistry",
      "title": "Apigee Registry API"
    },
    {
      "id": "apihub:v25.11.00355",
      "name": "apihub",
      "title": "API hub API"
    },
    {
      "id": "apikeys:v25.11.00355",
      "name": "apikeys",
      "title": "API Keys API"
    },
    {
      "id": "apim:v25.11.00355",
      "name": "apim",
      "title": "API Management API"
    },
    {
      "id": "appengine:v25.11.00355",
      "name": "appengine",
      "title": "App Engine Admin API"
    },
    {
      "id": "apphub:v25.11.00355",
      "name": "apphub",
      "title": "App Hub API"
    },
    {
      "id": "areainsights:v25.11.00355",
      "name": "areainsights",
      "title": "Places Aggregate API"
    },
    {
      "id": "artifactregistry:v25.11.00355",
      "name": "artifactregistry",
      "title": "Artifact Registry API"
    },
    {
      "id": "assuredworkloads:v25.11.00355",
      "name": "assuredworkloads",
      "title": "Assured Workloads API"
    },
    {
      "id": "backupdr:v25.11.00355",
      "name": "backupdr",
      "title": "Backup and DR Service API"
    },
    {
      "id": "baremetalsolution:v25.11.00355",
      "name": "baremetalsolution",
      "title": "Bare Metal Solution API"
    },
    {
      "id": "batch:v25.11.00355",
      "name": "batch",
      "title": "Batch API"
    },
    {
      "id": "beyondcorp:v25.11.00355",
      "name": "beyondcorp",
      "title": "BeyondCorp API"
    },
    {
      "id": "biglake:v25.11.00355",
      "name": "biglake",
      "title": "BigLake API"
    },
    {
      "id": "bigquery:v25.11.00355",
      "name": "bigquery",
      "title": "BigQuery API"
    },
    {
      "id": "bigqueryconnection:v25.11.00355",
      "name": "bigqueryconnection",
      "title": "BigQuery Connection API"
    },
    {
      "id": "bigquerydatapolicy:v25.11.00355",
      "name": "bigquerydatapolicy",
      "title": "BigQuery Data Policy API"
    },
    {
      "id": "bigquerydatatransfer:v25.11.00355",
      "name": "bigquerydatatransfer",
      "title": "BigQuery Data Transfer API"
    },
    {
      "id": "bigqueryreservation:v25.11.00355",
      "name": "bigqueryreservation",
      "title": "BigQuery Reservation API"
    },
    {
      "id": "bigtableadmin:v25.11.00355",
      "name": "bigtableadmin",
      "title": "Cloud Bigtable Admin API"
    },
    {
      "id": "billingbudgets:v25.11.00355",
      "name": "billingbudgets",
      "title": "Cloud Billing Budget API"
    },
    {
      "id": "binaryauthorization:v25.11.00355",
      "name": "binaryauthorization",
      "title": "Binary Authorization API"
    },
    {
      "id": "blockchainnodeengine:v25.11.00355",
      "name": "blockchainnodeengine",
      "title": "Blockchain Node Engine API"
    },
    {
      "id": "certificatemanager:v25.11.00355",
      "name": "certificatemanager",
      "title": "Certificate Manager API"
    },
    {
      "id": "cloudasset:v25.11.00355",
      "name": "cloudasset",
      "title": "Cloud Asset API"
    },
    {
      "id": "cloudbilling:v25.11.00355",
      "name": "cloudbilling",
      "title": "Cloud Billing API"
    },
    {
      "id": "cloudbuild:v25.11.00355",
      "name": "cloudbuild",
      "title": "Cloud Build API"
    },
    {
      "id": "cloudcommerceprocurement:v25.11.00355",
      "name": "cloudcommerceprocurement",
      "title": "Cloud Commerce Partner Procurement API"
    },
    {
      "id": "cloudcontrolspartner:v25.11.00355",
      "name": "cloudcontrolspartner",
      "title": "Cloud Controls Partner API"
    },
    {
      "id": "clouddeploy:v25.11.00355",
      "name": "clouddeploy",
      "title": "Cloud Deploy API"
    },
    {
      "id": "clouderrorreporting:v25.11.00355",
      "name": "clouderrorreporting",
      "title": "Error Reporting API"
    },
    {
      "id": "cloudfunctions:v25.11.00355",
      "name": "cloudfunctions",
      "title": "Cloud Functions API"
    },
    {
      "id": "cloudidentity:v25.11.00355",
      "name": "cloudidentity",
      "title": "Cloud Identity API"
    },
    {
      "id": "cloudkms:v25.11.00355",
      "name": "cloudkms",
      "title": "Cloud Key Management Service (KMS) API"
    },
    {
      "id": "cloudlocationfinder:v25.11.00355",
      "name": "cloudlocationfinder",
      "title": "Cloud Location Finder API"
    },
    {
      "id": "cloudprofiler:v25.11.00355",
      "name": "cloudprofiler",
      "title": "Cloud Profiler API"
    },
    {
      "id": "cloudresourcemanager:v25.11.00355",
      "name": "cloudresourcemanager",
      "title": "Cloud Resource Manager API"
    },
    {
      "id": "cloudscheduler:v25.11.00355",
      "name": "cloudscheduler",
      "title": "Cloud Scheduler API"
    },
    {
      "id": "cloudshell:v25.11.00355",
      "name": "cloudshell",
      "title": "Cloud Shell API"
    },
    {
      "id": "cloudsupport:v25.11.00355",
      "name": "cloudsupport",
      "title": "Google Cloud Support API"
    },
    {
      "id": "cloudtasks:v25.11.00355",
      "name": "cloudtasks",
      "title": "Cloud Tasks API"
    },
    {
      "id": "cloudtrace:v25.11.00355",
      "name": "cloudtrace",
      "title": "Cloud Trace API"
    },
    {
      "id": "composer:v25.11.00355",
      "name": "composer",
      "title": "Cloud Composer API"
    },
    {
      "id": "compute:v25.11.00355",
      "name": "compute",
      "title": "Compute Engine API"
    },
    {
      "id": "config:v25.11.00355",
      "name": "config",
      "title": "Infrastructure Manager API"
    },
    {
      "id": "connectors:v25.11.00355",
      "name": "connectors",
      "title": "Connectors API"
    },
    {
      "id": "contactcenteraiplatform:v25.11.00355",
      "name": "contactcenteraiplatform",
      "title": "Contact Center AI Platform API"
    },
    {
      "id": "contactcenterinsights:v25.11.00355",
      "name": "contactcenterinsights",
      "title": "Contact Center AI Insights API"
    },
    {
      "id": "container:v25.11.00355",
      "name": "container",
      "title": "Kubernetes Engine API"
    },
    {
      "id": "containeranalysis:v25.11.00355",
      "name": "containeranalysis",
      "title": "Container Analysis API"
    },
    {
      "id": "contentwarehouse:v25.11.00355",
      "name": "contentwarehouse",
      "title": "Document AI Warehouse API"
    },
    {
      "id": "datacatalog:v25.11.00355",
      "name": "datacatalog",
      "title": "Google Cloud Data Catalog API"
    },
    {
      "id": "dataflow:v25.11.00355",
      "name": "dataflow",
      "title": "Dataflow API"
    },
    {
      "id": "dataform:v25.11.00355",
      "name": "dataform",
      "title": "Dataform API"
    },
    {
      "id": "datafusion:v25.11.00355",
      "name": "datafusion",
      "title": "Cloud Data Fusion API"
    },
    {
      "id": "datalabeling:v25.11.00355",
      "name": "datalabeling",
      "title": "Data Labeling API"
    },
    {
      "id": "datalineage:v25.11.00355",
      "name": "datalineage",
      "title": "Data Lineage API"
    },
    {
      "id": "datamigration:v25.11.00355",
      "name": "datamigration",
      "title": "Database Migration API"
    },
    {
      "id": "datapipelines:v25.11.00355",
      "name": "datapipelines",
      "title": "Data pipelines API"
    },
    {
      "id": "dataplex:v25.11.00355",
      "name": "dataplex",
      "title": "Cloud Dataplex API"
    },
    {
      "id": "dataproc:v25.11.00355",
      "name": "dataproc",
      "title": "Cloud Dataproc API"
    },
    {
      "id": "datastore:v25.11.00355",
      "name": "datastore",
      "title": "Cloud Datastore API"
    },
    {
      "id": "datastream:v25.11.00355",
      "name": "datastream",
      "title": "Datastream API"
    },
    {
      "id": "deploymentmanager:v25.11.00355",
      "name": "deploymentmanager",
      "title": "Cloud Deployment Manager V2 API"
    },
    {
      "id": "developerconnect:v25.11.00355",
      "name": "developerconnect",
      "title": "Developer Connect API"
    },
    {
      "id": "dialogflow:v25.11.00355",
      "name": "dialogflow",
      "title": "Dialogflow API"
    },
    {
      "id": "discoveryengine:v25.11.00355",
      "name": "discoveryengine",
      "title": "Discovery Engine API"
    },
    {
      "id": "dlp:v25.11.00355",
      "name": "dlp",
      "title": "Sensitive Data Protection (DLP)"
    },
    {
      "id": "dns:v25.11.00355",
      "name": "dns",
      "title": "Cloud DNS API"
    },
    {
      "id": "documentai:v25.11.00355",
      "name": "documentai",
      "title": "Cloud Document AI API"
    },
    {
      "id": "domains:v25.11.00355",
      "name": "domains",
      "title": "Cloud Domains API"
    },
    {
      "id": "essentialcontacts:v25.11.00355",
      "name": "essentialcontacts",
      "title": "Essential Contacts API"
    },
    {
      "id": "eventarc:v25.11.00355",
      "name": "eventarc",
      "title": "Eventarc API"
    },
    {
      "id": "file:v25.11.00355",
      "name": "file",
      "title": "Cloud Filestore API"
    },
    {
      "id": "firestore:v25.11.00355",
      "name": "firestore",
      "title": "Cloud Firestore API"
    },
    {
      "id": "geminicloudassist:v25.11.00355",
      "name": "geminicloudassist",
      "title": "Gemini Cloud Assist API"
    },
    {
      "id": "gkebackup:v25.11.00355",
      "name": "gkebackup",
      "title": "Backup for GKE API"
    },
    {
      "id": "gkehub:v25.11.00355",
      "name": "gkehub",
      "title": "GKE Hub API"
    },
    {
      "id": "gkeonprem:v25.11.00355",
      "name": "gkeonprem",
      "title": "GKE On-Prem API"
    },
    {
      "id": "healthcare:v25.11.00355",
      "name": "healthcare",
      "title": "Cloud Healthcare API"
    },
    {
      "id": "iam:v25.11.00355",
      "name": "iam",
      "title": "Identity and Access Management (IAM) API"
    },
    {
      "id": "iamcredentials:v25.11.00355",
      "name": "iamcredentials",
      "title": "IAM Service Account Credentials API"
    },
    {
      "id": "iamv2:v25.11.00355",
      "name": "iamv2",
      "title": "Identity and Access Management (IAM) API"
    },
    {
      "id": "iamv2beta:v25.11.00355",
      "name": "iamv2beta",
      "title": "Identity and Access Management (IAM) API"
    },
    {
      "id": "iap:v25.11.00355",
      "name": "iap",
      "title": "Cloud Identity-Aware Proxy API"
    },
    {
      "id": "identitytoolkit:v25.11.00355",
      "name": "identitytoolkit",
      "title": "Google Identity Toolkit API"
    },
    {
      "id": "ids:v25.11.00355",
      "name": "ids",
      "title": "Cloud IDS API"
    },
    {
      "id": "integrations:v25.11.00355",
      "name": "integrations",
      "title": "Application Integration API"
    },
    {
      "id": "jobs:v25.11.00355",
      "name": "jobs",
      "title": "Cloud Talent Solution API"
    },
    {
      "id": "kmsinventory:v25.11.00355",
      "name": "kmsinventory",
      "title": "KMS Inventory API"
    },
    {
      "id": "language:v25.11.00355",
      "name": "language",
      "title": "Cloud Natural Language API"
    },
    {
      "id": "libraryagent:v25.11.00355",
      "name": "libraryagent",
      "title": "Library Agent API"
    },
    {
      "id": "lifesciences:v25.11.00355",
      "name": "lifesciences",
      "title": "Cloud Life Sciences API"
    },
    {
      "id": "logging:v25.11.00355",
      "name": "logging",
      "title": "Cloud Logging API"
    },
    {
      "id": "looker:v25.11.00355",
      "name": "looker",
      "title": "Looker (Google Cloud core) API"
    },
    {
      "id": "managedidentities:v25.11.00355",
      "name": "managedidentities",
      "title": "Managed Service for Microsoft Active Directory API"
    },
    {
      "id": "managedkafka:v25.11.00355",
      "name": "managedkafka",
      "title": "Managed Service for Apache Kafka API"
    },
    {
      "id": "memcache:v25.11.00355",
      "name": "memcache",
      "title": "Cloud Memorystore for Memcached API"
    },
    {
      "id": "migrationcenter:v25.11.00355",
      "name": "migrationcenter",
      "title": "Migration Center API"
    },
    {
      "id": "ml:v25.11.00355",
      "name": "ml",
      "title": "AI Platform Training & Prediction API"
    },
    {
      "id": "monitoring:v25.11.00355",
      "name": "monitoring",
      "title": "Cloud Monitoring API"
    },
    {
      "id": "netapp:v25.11.00355",
      "name": "netapp",
      "title": "NetApp API"
    },
    {
      "id": "networkconnectivity:v25.11.00355",
      "name": "networkconnectivity",
      "title": "Network Connectivity API"
    },
    {
      "id": "networkmanagement:v25.11.00355",
      "name": "networkmanagement",
      "title": "Network Management API"
    },
    {
      "id": "networksecurity:v25.11.00355",
      "name": "networksecurity",
      "title": "Network Security API"
    },
    {
      "id": "networkservices:v25.11.00355",
      "name": "networkservices",
      "title": "Network Services API"
    },
    {
      "id": "notebooks:v25.11.00355",
      "name": "notebooks",
      "title": "Notebooks API"
    },
    {
      "id": "observability:v25.11.00355",
      "name": "observability",
      "title": "Observability API"
    },
    {
      "id": "ondemandscanning:v25.11.00355",
      "name": "ondemandscanning",
      "title": "On-Demand Scanning API"
    },
    {
      "id": "oracledatabase:v25.11.00355",
      "name": "oracledatabase",
      "title": "Oracle Database@Google Cloud API"
    },
    {
      "id": "orgpolicy:v25.11.00355",
      "name": "orgpolicy",
      "title": "Organization Policy API"
    },
    {
      "id": "osconfig:v25.11.00355",
      "name": "osconfig",
      "title": "OS Config API"
    },
    {
      "id": "oslogin:v25.11.00355",
      "name": "oslogin",
      "title": "Cloud OS Login API"
    },
    {
      "id": "parallelstore:v25.11.00355",
      "name": "parallelstore",
      "title": "Parallelstore API"
    },
    {
      "id": "parametermanager:v25.11.00355",
      "name": "parametermanager",
      "title": "Parameter Manager API"
    },
    {
      "id": "places:v25.11.00355",
      "name": "places",
      "title": "Places API (New)"
    },
    {
      "id": "policyanalyzer:v25.11.00355",
      "name": "policyanalyzer",
      "title": "Policy Analyzer API"
    },
    {
      "id": "policysimulator:v25.11.00355",
      "name": "policysimulator",
      "title": "Policy Simulator API"
    },
    {
      "id": "policytroubleshooter:v25.11.00355",
      "name": "policytroubleshooter",
      "title": "Policy Troubleshooter API"
    },
    {
      "id": "pollen:v25.11.00355",
      "name": "pollen",
      "title": "Pollen API"
    },
    {
      "id": "privateca:v25.11.00355",
      "name": "privateca",
      "title": "Certificate Authority API"
    },
    {
      "id": "prod_tt_sasportal:v25.11.00355",
      "name": "prod_tt_sasportal",
      "title": "SAS Portal API (Testing)"
    },
    {
      "id": "publicca:v25.11.00355",
      "name": "publicca",
      "title": "Public Certificate Authority API"
    },
    {
      "id": "pubsub:v25.11.00355",
      "name": "pubsub",
      "title": "Cloud Pub/Sub API"
    },
    {
      "id": "pubsublite:v25.11.00355",
      "name": "pubsublite",
      "title": "Pub/Sub Lite API"
    },
    {
      "id": "rapidmigrationassessment:v25.11.00355",
      "name": "rapidmigrationassessment",
      "title": "Rapid Migration Assessment API"
    },
    {
      "id": "recaptchaenterprise:v25.11.00355",
      "name": "recaptchaenterprise",
      "title": "reCAPTCHA Enterprise API"
    },
    {
      "id": "recommendationengine:v25.11.00355",
      "name": "recommendationengine",
      "title": "Recommendations AI (Beta)"
    },
    {
      "id": "recommender:v25.11.00355",
      "name": "recommender",
      "title": "Recommender API"
    },
    {
      "id": "redis:v25.11.00355",
      "name": "redis",
      "title": "Google Cloud Memorystore for Redis API"
    },
    {
      "id": "retail:v25.11.00355",
      "name": "retail",
      "title": "Vertex AI Search for commerce API"
    },
    {
      "id": "run:v25.11.00355",
      "name": "run",
      "title": "Cloud Run Admin API"
    },
    {
      "id": "runtimeconfig:v25.11.00355",
      "name": "runtimeconfig",
      "title": "Cloud Runtime Configuration API"
    },
    {
      "id": "saasservicemgmt:v25.11.00355",
      "name": "saasservicemgmt",
      "title": "SaaS Runtime API"
    },
    {
      "id": "sasportal:v25.11.00355",
      "name": "sasportal",
      "title": "SAS Portal API"
    },
    {
      "id": "secretmanager:v25.11.00355",
      "name": "secretmanager",
      "title": "Secret Manager API"
    },
    {
      "id": "securesourcemanager:v25.11.00355",
      "name": "securesourcemanager",
      "title": "Secure Source Manager API"
    },
    {
      "id": "securitycenter:v25.11.00355",
      "name": "securitycenter",
      "title": "Security Command Center API"
    },
    {
      "id": "securityposture:v25.11.00355",
      "name": "securityposture",
      "title": "Security Posture API"
    },
    {
      "id": "serviceconsumermanagement:v25.11.00355",
      "name": "serviceconsumermanagement",
      "title": "Service Consumer Management API"
    },
    {
      "id": "servicecontrol:v25.11.00355",
      "name": "servicecontrol",
      "title": "Service Control API"
    },
    {
      "id": "servicedirectory:v25.11.00355",
      "name": "servicedirectory",
      "title": "Service Directory API"
    },
    {
      "id": "servicemanagement:v25.11.00355",
      "name": "servicemanagement",
      "title": "Service Management API"
    },
    {
      "id": "servicenetworking:v25.11.00355",
      "name": "servicenetworking",
      "title": "Service Networking API"
    },
    {
      "id": "serviceusage:v25.11.00355",
      "name": "serviceusage",
      "title": "Service Usage API"
    },
    {
      "id": "solar:v25.11.00355",
      "name": "solar",
      "title": "Solar API"
    },
    {
      "id": "spanner:v25.11.00355",
      "name": "spanner",
      "title": "Cloud Spanner API"
    },
    {
      "id": "speech:v25.11.00355",
      "name": "speech",
      "title": "Cloud Speech-to-Text API"
    },
    {
      "id": "sqladmin:v25.11.00355",
      "name": "sqladmin",
      "title": "Cloud SQL Admin API"
    },
    {
      "id": "storage:v25.11.00355",
      "name": "storage",
      "title": "Cloud Storage JSON API"
    },
    {
      "id": "storagebatchoperations:v25.11.00355",
      "name": "storagebatchoperations",
      "title": "Storage Batch Operations API"
    },
    {
      "id": "storagetransfer:v25.11.00355",
      "name": "storagetransfer",
      "title": "Storage Transfer API"
    },
    {
      "id": "texttospeech:v25.11.00355",
      "name": "texttospeech",
      "title": "Cloud Text-to-Speech API"
    },
    {
      "id": "tpu:v25.11.00355",
      "name": "tpu",
      "title": "Cloud TPU API"
    },
    {
      "id": "trafficdirector:v25.11.00355",
      "name": "trafficdirector",
      "title": "Traffic Director API"
    },
    {
      "id": "transcoder:v25.11.00355",
      "name": "transcoder",
      "title": "Transcoder API"
    },
    {
      "id": "translate:v25.11.00355",
      "name": "translate",
      "title": "Cloud Translation API"
    },
    {
      "id": "videointelligence:v25.11.00355",
      "name": "videointelligence",
      "title": "Cloud Video Intelligence API"
    },
    {
      "id": "vision:v25.11.00355",
      "name": "vision",
      "title": "Cloud Vision API"
    },
    {
      "id": "vmmigration:v25.11.00355",
      "name": "vmmigration",
      "title": "VM Migration API"
    },
    {
      "id": "vmwareengine:v25.11.00355",
      "name": "vmwareengine",
      "title": "VMware Engine API"
    },
    {
      "id": "vpcaccess:v25.11.00355",
      "name": "vpcaccess",
      "title": "Serverless VPC Access API"
    },
    {
      "id": "webrisk:v25.11.00355",
      "name": "webrisk",
      "title": "Web Risk API"
    },
    {
      "id": "websecurityscanner:v25.11.00355",
      "name": "websecurityscanner",
      "title": "Web Security Scanner API"
    },
    {
      "id": "workflowexecutions:v25.11.00355",
      "name": "workflowexecutions",
      "title": "Workflow Executions API"
    },
    {
      "id": "workflows:v25.11.00355",
      "name": "workflows",
      "title": "Workflows API"
    },
    {
      "id": "workloadmanager:v25.11.00355",
      "name": "workloadmanager",
      "title": "Workload Manager API"
    },
    {
      "id": "workstations:v25.11.00355",
      "name": "workstations",
      "title": "Cloud Workstations API"
    }
  ],
  "row_count": 178,
  "format": "json"
}

$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      list_resources --exec.args '{"provider": "google", "service": "compute"}' 2>/dev/null | jq
{
  "rows": [
    {
      "id": "google.compute.accelerator_types",
      "name": "accelerator_types"
    },
    {
      "id": "google.compute.addresses",
      "name": "addresses"
    },
    {
      "id": "google.compute.autoscalers",
      "name": "autoscalers"
    },
    {
      "id": "google.compute.backend_buckets",
      "name": "backend_buckets"
    },
    {
      "id": "google.compute.backend_buckets_iam_policies",
      "name": "backend_buckets_iam_policies"
    },
    {
      "id": "google.compute.backend_services",
      "name": "backend_services"
    },
    {
      "id": "google.compute.backend_services_aggregated",
      "name": "backend_services_aggregated"
    },
    {
      "id": "google.compute.backend_services_health",
      "name": "backend_services_health"
    },
    {
      "id": "google.compute.backend_services_iam_policies",
      "name": "backend_services_iam_policies"
    },
    {
      "id": "google.compute.backend_services_usable",
      "name": "backend_services_usable"
    },
    {
      "id": "google.compute.commitments",
      "name": "commitments"
    },
    {
      "id": "google.compute.disk_types",
      "name": "disk_types"
    },
    {
      "id": "google.compute.disks",
      "name": "disks"
    },
    {
      "id": "google.compute.disks_iam_policies",
      "name": "disks_iam_policies"
    },
    {
      "id": "google.compute.disks_resource_policies",
      "name": "disks_resource_policies"
    },
    {
      "id": "google.compute.disks_snapshot",
      "name": "disks_snapshot"
    },
    {
      "id": "google.compute.effective_firewalls",
      "name": "effective_firewalls"
    },
    {
      "id": "google.compute.external_vpn_gateways",
      "name": "external_vpn_gateways"
    },
    {
      "id": "google.compute.external_vpn_gateways_iam_policies",
      "name": "external_vpn_gateways_iam_policies"
    },
    {
      "id": "google.compute.firewall_policies",
      "name": "firewall_policies"
    },
    {
      "id": "google.compute.firewall_policies_associations",
      "name": "firewall_policies_associations"
    },
    {
      "id": "google.compute.firewall_policies_iam_policies",
      "name": "firewall_policies_iam_policies"
    },
    {
      "id": "google.compute.firewall_policies_rule",
      "name": "firewall_policies_rule"
    },
    {
      "id": "google.compute.firewalls",
      "name": "firewalls"
    },
    {
      "id": "google.compute.forwarding_rules",
      "name": "forwarding_rules"
    },
    {
      "id": "google.compute.forwarding_rules_aggregated",
      "name": "forwarding_rules_aggregated"
    },
    {
      "id": "google.compute.health_check_services",
      "name": "health_check_services"
    },
    {
      "id": "google.compute.health_checks",
      "name": "health_checks"
    },
    {
      "id": "google.compute.health_checks_aggregated",
      "name": "health_checks_aggregated"
    },
    {
      "id": "google.compute.http_health_checks",
      "name": "http_health_checks"
    },
    {
      "id": "google.compute.https_health_checks",
      "name": "https_health_checks"
    },
    {
      "id": "google.compute.image_family_views",
      "name": "image_family_views"
    },
    {
      "id": "google.compute.images",
      "name": "images"
    },
    {
      "id": "google.compute.images_iam_policies",
      "name": "images_iam_policies"
    },
    {
      "id": "google.compute.instance_group_manager_resize_requests",
      "name": "instance_group_manager_resize_requests"
    },
    {
      "id": "google.compute.instance_group_managers",
      "name": "instance_group_managers"
    },
    {
      "id": "google.compute.instance_group_managers_errors",
      "name": "instance_group_managers_errors"
    },
    {
      "id": "google.compute.instance_group_managers_instances",
      "name": "instance_group_managers_instances"
    },
    {
      "id": "google.compute.instance_group_managers_per_instance_configs",
      "name": "instance_group_managers_per_instance_configs"
    },
    {
      "id": "google.compute.instance_groups",
      "name": "instance_groups"
    },
    {
      "id": "google.compute.instance_groups_instances",
      "name": "instance_groups_instances"
    },
    {
      "id": "google.compute.instance_settings",
      "name": "instance_settings"
    },
    {
      "id": "google.compute.instance_templates",
      "name": "instance_templates"
    },
    {
      "id": "google.compute.instance_templates_aggregated",
      "name": "instance_templates_aggregated"
    },
    {
      "id": "google.compute.instance_templates_iam_policies",
      "name": "instance_templates_iam_policies"
    },
    {
      "id": "google.compute.instances",
      "name": "instances"
    },
    {
      "id": "google.compute.instances_access_config",
      "name": "instances_access_config"
    },
    {
      "id": "google.compute.instances_guest_attributes",
      "name": "instances_guest_attributes"
    },
    {
      "id": "google.compute.instances_iam_policies",
      "name": "instances_iam_policies"
    },
    {
      "id": "google.compute.instances_referrers",
      "name": "instances_referrers"
    },
    {
      "id": "google.compute.instances_resource_policies",
      "name": "instances_resource_policies"
    },
    {
      "id": "google.compute.instances_screenshot",
      "name": "instances_screenshot"
    },
    {
      "id": "google.compute.instances_serial_port_output",
      "name": "instances_serial_port_output"
    },
    {
      "id": "google.compute.instant_snapshots",
      "name": "instant_snapshots"
    },
    {
      "id": "google.compute.instant_snapshots_iam_policies",
      "name": "instant_snapshots_iam_policies"
    },
    {
      "id": "google.compute.interconnect_attachment_groups",
      "name": "interconnect_attachment_groups"
    },
    {
      "id": "google.compute.interconnect_attachment_groups_iam_policies",
      "name": "interconnect_attachment_groups_iam_policies"
    },
    {
      "id": "google.compute.interconnect_attachment_groups_operational_status",
      "name": "interconnect_attachment_groups_operational_status"
    },
    {
      "id": "google.compute.interconnect_attachments",
      "name": "interconnect_attachments"
    },
    {
      "id": "google.compute.interconnect_groups",
      "name": "interconnect_groups"
    },
    {
      "id": "google.compute.interconnect_groups_iam_policies",
      "name": "interconnect_groups_iam_policies"
    },
    {
      "id": "google.compute.interconnect_groups_operational_status",
      "name": "interconnect_groups_operational_status"
    },
    {
      "id": "google.compute.interconnect_locations",
      "name": "interconnect_locations"
    },
    {
      "id": "google.compute.interconnect_remote_locations",
      "name": "interconnect_remote_locations"
    },
    {
      "id": "google.compute.interconnects",
      "name": "interconnects"
    },
    {
      "id": "google.compute.interconnects_diagnostics",
      "name": "interconnects_diagnostics"
    },
    {
      "id": "google.compute.interconnects_macsec_config",
      "name": "interconnects_macsec_config"
    },
    {
      "id": "google.compute.license_codes",
      "name": "license_codes"
    },
    {
      "id": "google.compute.license_codes_iam_policies",
      "name": "license_codes_iam_policies"
    },
    {
      "id": "google.compute.licenses",
      "name": "licenses"
    },
    {
      "id": "google.compute.licenses_iam_policies",
      "name": "licenses_iam_policies"
    },
    {
      "id": "google.compute.machine_images",
      "name": "machine_images"
    },
    {
      "id": "google.compute.machine_images_iam_policies",
      "name": "machine_images_iam_policies"
    },
    {
      "id": "google.compute.machine_types",
      "name": "machine_types"
    },
    {
      "id": "google.compute.network_attachments",
      "name": "network_attachments"
    },
    {
      "id": "google.compute.network_attachments_iam_policies",
      "name": "network_attachments_iam_policies"
    },
    {
      "id": "google.compute.network_edge_security_services",
      "name": "network_edge_security_services"
    },
    {
      "id": "google.compute.network_endpoint_groups",
      "name": "network_endpoint_groups"
    },
    {
      "id": "google.compute.network_endpoint_groups_iam_policies",
      "name": "network_endpoint_groups_iam_policies"
    },
    {
      "id": "google.compute.network_endpoints",
      "name": "network_endpoints"
    },
    {
      "id": "google.compute.network_profiles",
      "name": "network_profiles"
    },
    {
      "id": "google.compute.networks",
      "name": "networks"
    },
    {
      "id": "google.compute.networks_effective_firewalls",
      "name": "networks_effective_firewalls"
    },
    {
      "id": "google.compute.networks_peering",
      "name": "networks_peering"
    },
    {
      "id": "google.compute.networks_peering_routes",
      "name": "networks_peering_routes"
    },
    {
      "id": "google.compute.node_groups",
      "name": "node_groups"
    },
    {
      "id": "google.compute.node_groups_iam_policies",
      "name": "node_groups_iam_policies"
    },
    {
      "id": "google.compute.node_groups_nodes",
      "name": "node_groups_nodes"
    },
    {
      "id": "google.compute.node_templates",
      "name": "node_templates"
    },
    {
      "id": "google.compute.node_templates_iam_policies",
      "name": "node_templates_iam_policies"
    },
    {
      "id": "google.compute.node_types",
      "name": "node_types"
    },
    {
      "id": "google.compute.notification_endpoints",
      "name": "notification_endpoints"
    },
    {
      "id": "google.compute.operations",
      "name": "operations"
    },
    {
      "id": "google.compute.operations_aggregated",
      "name": "operations_aggregated"
    },
    {
      "id": "google.compute.packet_mirroring_rule",
      "name": "packet_mirroring_rule"
    },
    {
      "id": "google.compute.packet_mirrorings",
      "name": "packet_mirrorings"
    },
    {
      "id": "google.compute.packet_mirrorings_iam_policies",
      "name": "packet_mirrorings_iam_policies"
    },
    {
      "id": "google.compute.projects",
      "name": "projects"
    },
    {
      "id": "google.compute.public_advertised_prefixes",
      "name": "public_advertised_prefixes"
    },
    {
      "id": "google.compute.public_delegated_prefixes",
      "name": "public_delegated_prefixes"
    },
    {
      "id": "google.compute.public_delegated_prefixes_aggregated",
      "name": "public_delegated_prefixes_aggregated"
    },
    {
      "id": "google.compute.regions",
      "name": "regions"
    },
    {
      "id": "google.compute.reservation_blocks",
      "name": "reservation_blocks"
    },
    {
      "id": "google.compute.reservation_sub_blocks",
      "name": "reservation_sub_blocks"
    },
    {
      "id": "google.compute.reservations",
      "name": "reservations"
    },
    {
      "id": "google.compute.reservations_iam_policies",
      "name": "reservations_iam_policies"
    },
    {
      "id": "google.compute.resource_policies",
      "name": "resource_policies"
    },
    {
      "id": "google.compute.resource_policies_iam_policies",
      "name": "resource_policies_iam_policies"
    },
    {
      "id": "google.compute.route_policies",
      "name": "route_policies"
    },
    {
      "id": "google.compute.router_bgp_routes",
      "name": "router_bgp_routes"
    },
    {
      "id": "google.compute.router_nat_ip_info",
      "name": "router_nat_ip_info"
    },
    {
      "id": "google.compute.router_nat_mapping_info",
      "name": "router_nat_mapping_info"
    },
    {
      "id": "google.compute.router_status",
      "name": "router_status"
    },
    {
      "id": "google.compute.routers",
      "name": "routers"
    },
    {
      "id": "google.compute.routes",
      "name": "routes"
    },
    {
      "id": "google.compute.security_policies",
      "name": "security_policies"
    },
    {
      "id": "google.compute.security_policies_aggregated",
      "name": "security_policies_aggregated"
    },
    {
      "id": "google.compute.security_policies_expression_sets",
      "name": "security_policies_expression_sets"
    },
    {
      "id": "google.compute.security_policies_rule",
      "name": "security_policies_rule"
    },
    {
      "id": "google.compute.service_attachments",
      "name": "service_attachments"
    },
    {
      "id": "google.compute.service_attachments_iam_policies",
      "name": "service_attachments_iam_policies"
    },
    {
      "id": "google.compute.shielded_instance_identity",
      "name": "shielded_instance_identity"
    },
    {
      "id": "google.compute.snapshot_settings",
      "name": "snapshot_settings"
    },
    {
      "id": "google.compute.snapshots",
      "name": "snapshots"
    },
    {
      "id": "google.compute.snapshots_iam_policies",
      "name": "snapshots_iam_policies"
    },
    {
      "id": "google.compute.ssl_certificates",
      "name": "ssl_certificates"
    },
    {
      "id": "google.compute.ssl_certificates_aggregated",
      "name": "ssl_certificates_aggregated"
    },
    {
      "id": "google.compute.ssl_policies",
      "name": "ssl_policies"
    },
    {
      "id": "google.compute.ssl_policies_aggregated",
      "name": "ssl_policies_aggregated"
    },
    {
      "id": "google.compute.ssl_policies_available_features",
      "name": "ssl_policies_available_features"
    },
    {
      "id": "google.compute.storage_pool_types",
      "name": "storage_pool_types"
    },
    {
      "id": "google.compute.storage_pools",
      "name": "storage_pools"
    },
    {
      "id": "google.compute.storage_pools_disks",
      "name": "storage_pools_disks"
    },
    {
      "id": "google.compute.storage_pools_iam_policies",
      "name": "storage_pools_iam_policies"
    },
    {
      "id": "google.compute.subnetworks",
      "name": "subnetworks"
    },
    {
      "id": "google.compute.subnetworks_iam_policies",
      "name": "subnetworks_iam_policies"
    },
    {
      "id": "google.compute.subnetworks_usable",
      "name": "subnetworks_usable"
    },
    {
      "id": "google.compute.target_grpc_proxies",
      "name": "target_grpc_proxies"
    },
    {
      "id": "google.compute.target_http_proxies",
      "name": "target_http_proxies"
    },
    {
      "id": "google.compute.target_http_proxies_aggregated",
      "name": "target_http_proxies_aggregated"
    },
    {
      "id": "google.compute.target_https_proxies",
      "name": "target_https_proxies"
    },
    {
      "id": "google.compute.target_https_proxies_aggregated",
      "name": "target_https_proxies_aggregated"
    },
    {
      "id": "google.compute.target_instances",
      "name": "target_instances"
    },
    {
      "id": "google.compute.target_pools",
      "name": "target_pools"
    },
    {
      "id": "google.compute.target_pools_health_check",
      "name": "target_pools_health_check"
    },
    {
      "id": "google.compute.target_pools_instance",
      "name": "target_pools_instance"
    },
    {
      "id": "google.compute.target_ssl_proxies",
      "name": "target_ssl_proxies"
    },
    {
      "id": "google.compute.target_tcp_proxies",
      "name": "target_tcp_proxies"
    },
    {
      "id": "google.compute.target_tcp_proxies_aggregated",
      "name": "target_tcp_proxies_aggregated"
    },
    {
      "id": "google.compute.target_vpn_gateways",
      "name": "target_vpn_gateways"
    },
    {
      "id": "google.compute.url_maps",
      "name": "url_maps"
    },
    {
      "id": "google.compute.url_maps_aggregated",
      "name": "url_maps_aggregated"
    },
    {
      "id": "google.compute.vpn_gateways",
      "name": "vpn_gateways"
    },
    {
      "id": "google.compute.vpn_gateways_iam_policies",
      "name": "vpn_gateways_iam_policies"
    },
    {
      "id": "google.compute.vpn_gateways_status",
      "name": "vpn_gateways_status"
    },
    {
      "id": "google.compute.vpn_tunnels",
      "name": "vpn_tunnels"
    },
    {
      "id": "google.compute.xpn_hosts",
      "name": "xpn_hosts"
    },
    {
      "id": "google.compute.xpn_resources",
      "name": "xpn_resources"
    },
    {
      "id": "google.compute.zones",
      "name": "zones"
    }
  ],
  "row_count": 159,
  "format": "json"
}

$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      list_methods --exec.args '{"provider": "google", "service": "compute", "resource": "networks"}' 2>/dev/null | jq
{
  "rows": [
    {
      "MethodName": "get",
      "RequiredParams": "network, project",
      "SQLVerb": "SELECT"
    },
    {
      "MethodName": "list",
      "RequiredParams": "project",
      "SQLVerb": "SELECT"
    },
    {
      "MethodName": "insert",
      "RequiredParams": "project",
      "SQLVerb": "INSERT"
    },
    {
      "MethodName": "delete",
      "RequiredParams": "network, project",
      "SQLVerb": "DELETE"
    },
    {
      "MethodName": "patch",
      "RequiredParams": "network, project",
      "SQLVerb": "UPDATE"
    },
    {
      "MethodName": "request_remove_peering",
      "RequiredParams": "network, project",
      "SQLVerb": "EXEC"
    },
    {
      "MethodName": "switch_to_custom_mode",
      "RequiredParams": "network, project",
      "SQLVerb": "EXEC"
    }
  ],
  "row_count": 7,
  "format": "json"
}

$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action      meta_describe_table --exec.args '{"provider": "google", "service": "compute", "resource": "networks"}' 2>/dev/null | jq
{
  "rows": [
    {
      "name": "id",
      "type": "string"
    },
    {
      "name": "name",
      "type": "string"
    },
    {
      "name": "description",
      "type": "string"
    },
    {
      "name": "IPv4Range",
      "type": "string"
    },
    {
      "name": "autoCreateSubnetworks",
      "type": "boolean"
    },
    {
      "name": "creationTimestamp",
      "type": "string"
    },
    {
      "name": "enableUlaInternalIpv6",
      "type": "boolean"
    },
    {
      "name": "firewallPolicy",
      "type": "string"
    },
    {
      "name": "gatewayIPv4",
      "type": "string"
    },
    {
      "name": "internalIpv6Range",
      "type": "string"
    },
    {
      "name": "kind",
      "type": "string"
    },
    {
      "name": "mtu",
      "type": "integer"
    },
    {
      "name": "networkFirewallPolicyEnforcementOrder",
      "type": "string"
    },
    {
      "name": "networkProfile",
      "type": "string"
    },
    {
      "name": "params",
      "type": "object"
    },
    {
      "name": "peerings",
      "type": "array"
    },
    {
      "name": "routingConfig",
      "type": "object"
    },
    {
      "name": "selfLink",
      "type": "string"
    },
    {
      "name": "selfLinkWithId",
      "type": "string"
    },
    {
      "name": "subnetworks",
      "type": "array"
    }
  ],
  "row_count": 20,
  "format": "json"
}

$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action validate_query_json_v2      --exec.args '{"sql": "select name from google.compute.networks where project = '"'"'stackql-demo'"'"';"}' 2>/dev/null | jq
{
  "rows": [
    {
      "query": "EXPLAIN select name from google.compute.networks where project = 'stackql-demo'",
      "result": "OK",
      "timestamp": "2025-11-13T20:02:57+11:00 AEDT",
      "valid": "true"
    }
  ],
  "row_count": 1,
  "format": "json"
}


$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action query_json_v2      --exec.args '{"sql": "select name from google.compute.networks where project = '"'"'stackql-demo'"'"';"}' 2>/dev/null | jq
{
  "rows": [
    {
      "name": "pathfinders-test-01"
    },
    {
      "name": "pathfinders-test-02"
    },
    {
      "name": "returning-test-01"
    },
    {
      "name": "returning-test-03"
    }
  ],
  "row_count": 4,
  "format": "json"
}


## There are currently no guarantees about the contents of the response body for the tool.  Could vary wildly for a while.
$ ./build/stackql_mcp_client exec --client-type=http  --url=http://127.0.0.1:9992 --exec.action exec_query_json_v2      --exec.args '{"sql": "delete from google.compute.networks where project = '"'"'stackql-demo'"'"' and network = '"'"'returning-test-01'"'"' ;"}' 2>/dev/null | jq
{
  "messages": [
    "The operation was despatched successfully"
  ],
  "timestamp": "2025-11-13T22:46:09+11:00 AEDT"
}



```
