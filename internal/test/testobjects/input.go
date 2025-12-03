package testobjects

//nolint:lll,revive,stylecheck // This is a test file
const (
	SimpleSelectOktaApplicationApps               string = `select label, json_extract(settings, '$.notifications.vpn') st from okta.application.apps where subdomain = 'some-silly-subdomain' order by label asc;`
	SimpleSelectGoogleComputeInstance             string = `select name, zone from google.compute.instances where zone = 'australia-southeast1-b' AND /* */ project = 'testing-project';`
	SimpleSelectGoogleContainerSubnetworks        string = "select subnetwork, ipCidrRange from  google.container.\"projects.aggregated.usableSubnetworks\" where projectsId = 'testing-project' ;"
	K8STheHardWayTemplateFile                     string = "test/assets/input/k8s-the-hard-way/k8s-the-hard-way.iql"
	K8STheHardWayTemplateContextFile              string = "test/assets/input/k8s-the-hard-way/vars.jsonnet"
	SimpleShowResourcesFilteredFile               string = "test/assets/input/show/show-resources-filtered.iql"
	SimpleShowmethodsGoogleBQDatasetsFile         string = "test/assets/input/show/show-methods-google-bq-datasets.iql"
	SimpleShowmethodsGoogleStorageBucketsFile     string = "test/assets/input/show/show-methods-google-storage-buckets.iql"
	ShowInsertAddressesRequiredInputFile          string = "test/assets/input/simple-templating/show-insert-compute-addresses-required.iql"
	ShowInsertBQDatasetsFile                      string = "test/assets/input/simple-templating/show-insert-bigquery-datasets.iql"
	ShowInsertBQDatasetsRequiredFile              string = "test/assets/input/simple-templating/show-insert-bigquery-datasets-required.iql"
	SimpleInsertDependentComputeDisksFile         string = "test/assets/input/insert-dependent-compute-disk.iql"
	SimpleInsertDependentComputeDisksReversedFile string = "test/assets/input/insert-dependent-compute-disk-reversed.iql"
	SimpleInsertDependentBQDatasetFile            string = "test/assets/input/insert-dependent-bq-datasets.iql"
	SimpleSelectExecDependentOrgIamPolicyFile     string = "test/assets/input/select-exec-dependent-org-iam-policy.iql"
	SimpleInsertComputeNetwork                    string = `
	--
	-- create VPC 
	--
	INSERT /*+ AWAIT  */ INTO google.compute.networks
	(
	project,
	data__name,
	data__autoCreateSubnetworks,
	data__routingConfig
	) 
	SELECT
	'stackql-demo',
	'kubernetes-the-hard-way-vpc',
	false,
	'{"routingMode":"REGIONAL"}';
	`
	SimpleInsertExecComputeNetwork string = `EXEC /*+ AWAIT */ google.compute.networks.insert @project='stackql-demo' @@json='{ 
		"name": "kubernetes-the-hard-way-vpc",
	  "autoCreateSubnetworks": false,
	  "routingConfig": {"routingMode":"REGIONAL"}
		}';`
	SimpleDeleteComputeNetwork                                                string = `delete /*+ AWAIT  */ from google.compute.networks WHERE project = 'stackql-demo' and network = 'kubernetes-the-hard-way-vpc';`
	SimpleDeleteExecComputeNetwork                                            string = `EXEC /*+ AWAIT */ google.compute.networks.delete @project = 'stackql-demo', @network = 'kubernetes-the-hard-way-vpc';`
	SimpleAggCountGroupedGoogleContainerSubnetworkAsc                         string = "select ipCidrRange, sum(5) cc  from  google.container.\"projects.aggregated.usableSubnetworks\" where projectsId = 'testing-project' group by ipCidrRange having sum(5) >= 5 order by ipCidrRange asc;"
	SimpleAggCountGroupedGoogleContainerSubnetworkDesc                        string = "select ipCidrRange, sum(5) cc  from  google.container.\"projects.aggregated.usableSubnetworks\" where projectsId = 'testing-project' group by ipCidrRange having sum(5) >= 5 order by ipCidrRange desc;"
	SelectGoogleComputeDisksOrderCreationTmstpAsc                             string = `select d1.name, d1.sizeGb, d1.creationTimestamp from google.compute.disks d1 where zone = 'australia-southeast1-b' AND /* */ project = 'testing-project' ORDER BY creationTimestamp asc;`
	SelectGoogleComputeDisksOrderCreationTmstpAscPlusJSONExtract              string = `select name, json_extract('{"a":2,"c":[4,5,{"f":7}]}', '$.c') as json_rendition, sizeGb, creationTimestamp from google.compute.disks where zone = 'australia-southeast1-b' AND /* */ project = 'testing-project' ORDER BY creationTimestamp asc;`
	SelectGoogleComputeDisksOrderCreationTmstpAscPlusJSONExtractCoalesce      string = `select name, coalesce(json_extract(labels, '$.k1'), 'dummy_value') as json_rendition, sizeGb, creationTimestamp from google.compute.disks where zone = 'australia-southeast1-b' AND /* */ project = 'testing-project' ORDER BY creationTimestamp asc;`
	UnionSelectGoogleComputeDisksOrderCreationTmstpAscPlusJsonExtractCoalesce string = `select name, coalesce(json_extract(labels, '$.k1'), 'dummy_value') as json_rendition, sizeGb, creationTimestamp from google.compute.disks where zone = 'australia-southeast1-b' AND /* */ project = 'testing-project'
	                                                                               UNION ALL
																																								 select name, coalesce(json_extract(labels, '$.k1'), 'dummy_value') as json_rendition, sizeGb, creationTimestamp from google.compute.disks where zone = 'australia-southeast1-b' AND /* */ project = 'testing-project' 
																																								 ORDER BY creationTimestamp asc;`
	SelectGoogleComputeDisksOrderCreationTmstpAscPlusJsonExtractInstr string = `select name, INSTR(name, 'qq') as instr_rendition, sizeGb, creationTimestamp from google.compute.disks where zone = 'australia-southeast1-b' AND /* */ project = 'testing-project' ORDER BY creationTimestamp asc;`
	SelectGoogleComputeDisksAggOrderSizeAsc                           string = `select sizeGb, COUNT(1) as cc from google.compute.disks where zone = 'australia-southeast1-b' AND /* */ project = 'testing-project' GROUP BY sizeGb ORDER BY sizeGb ASC;`
	SelectGoogleComputeDisksAggOrderSizeDesc                          string = `select sizeGb, COUNT(1) as cc from google.compute.disks where zone = 'australia-southeast1-b' AND /* */ project = 'testing-project' GROUP BY sizeGb ORDER BY sizeGb DESC;`
	SelectGoogleComputeDisksAggSizeTotal                              string = `select sum(cast(sizeGb as unsigned)) - 10 as cc from google.compute.disks where zone = 'australia-southeast1-b' AND /* */ project = 'testing-project';`
	SelectGoogleComputeDisksAggStringTotal                            string = `select group_concat(substr(name, 0, 5)) || ' lalala' as cc from google.compute.disks where zone = 'australia-southeast1-b' AND /* */ project = 'testing-project';`

	// Window function test queries.
	SelectGoogleComputeDisksWindowRowNumber string = `select name, sizeGb, ROW_NUMBER() OVER (ORDER BY name) as row_num from google.compute.disks where zone = 'australia-southeast1-b' AND project = 'testing-project' ORDER BY name;`
	SelectGoogleComputeDisksWindowRank      string = `select name, sizeGb, RANK() OVER (ORDER BY sizeGb) as size_rank from google.compute.disks where zone = 'australia-southeast1-b' AND project = 'testing-project' ORDER BY name;`
	SelectGoogleComputeDisksWindowSum       string = `select name, sizeGb, SUM(cast(sizeGb as unsigned)) OVER (ORDER BY name) as running_total from google.compute.disks where zone = 'australia-southeast1-b' AND project = 'testing-project' ORDER BY name;`

	// CTE test queries.
	SelectGoogleComputeDisksCTESimple   string = `WITH disk_cte AS (SELECT name, sizeGb FROM google.compute.disks WHERE zone = 'australia-southeast1-b' AND project = 'testing-project') SELECT name, sizeGb FROM disk_cte ORDER BY name;`
	SelectGoogleComputeDisksCTEWithAgg  string = `WITH disk_cte AS (SELECT name, sizeGb FROM google.compute.disks WHERE zone = 'australia-southeast1-b' AND project = 'testing-project') SELECT COUNT(*) as disk_count FROM disk_cte;`
	SelectGoogleComputeDisksCTEMultiple string = `WITH first_cte AS (SELECT name, sizeGb FROM google.compute.disks WHERE zone = 'australia-southeast1-b' AND project = 'testing-project'), second_cte AS (SELECT name, sizeGb FROM first_cte) SELECT name, sizeGb FROM second_cte ORDER BY name;`
)

func GetGoogleProviderString() string {
	return "google"
}
