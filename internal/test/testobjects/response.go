package testobjects

import (
	"fmt"
)

//nolint:lll // This is a test file
const (
	SimpleSelectGoogleComputeInstanceResponse string = `{
		"id": "projects/testing-project/zones/australia-southeast1-b/instances",
		"items": [
			{
				"id": "0001",
				"creationTimestamp": "2021-02-20T15:55:46.907-08:00",
				"name": "demo-vm-tt1",
				"tags": {
					"fingerprint": "z="
				},
				"machineType": "https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-b/machineTypes/f1-micro",
				"status": "RUNNING",
				"zone": "https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-b",
				"networkInterfaces": [
					{
						"network": "https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/testing-vpc-01",
						"subnetwork": "https://www.googleapis.com/compute/v1/projects/testing-project/regions/australia-southeast1/subnetworks/aus-sn-01",
						"networkIP": "10.0.0.13",
						"name": "nic0",
						"fingerprint": "z=",
						"kind": "compute#networkInterface"
					}
				],
				"disks": [
					{
						"type": "PERSISTENT",
						"mode": "READ_WRITE",
						"source": "https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-b/disks/demo-disk-qq1",
						"deviceName": "persistent-disk-0",
						"index": 0,
						"boot": true,
						"autoDelete": false,
						"interface": "SCSI",
						"diskSizeGb": "10",
						"kind": "compute#attachedDisk"
					}
				],
				"metadata": {
					"fingerprint": "z=",
					"kind": "compute#metadata"
				},
				"selfLink": "https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-b/instances/demo-vm-tt1",
				"scheduling": {
					"onHostMaintenance": "MIGRATE",
					"automaticRestart": true,
					"preemptible": false
				},
				"cpuPlatform": "Intel Broadwell",
				"labelFingerprint": "z=",
				"startRestricted": false,
				"deletionProtection": false,
				"fingerprint": "z=",
				"lastStartTimestamp": "2021-03-10T11:28:58.562-08:00",
				"kind": "compute#instance"
			},
			{
				"id": "8852892103879695477",
				"creationTimestamp": "2021-02-20T16:00:27.118-08:00",
				"name": "demo-vm-tt2",
				"tags": {
					"fingerprint": "z="
				},
				"machineType": "https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-b/machineTypes/f1-micro",
				"status": "RUNNING",
				"zone": "https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-b",
				"networkInterfaces": [
					{
						"network": "https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/testing-vpc-01",
						"subnetwork": "https://www.googleapis.com/compute/v1/projects/testing-project/regions/australia-southeast1/subnetworks/aus-sn-01",
						"networkIP": "10.0.0.14",
						"name": "nic0",
						"fingerprint": "z=",
						"kind": "compute#networkInterface"
					}
				],
				"disks": [
					{
						"type": "PERSISTENT",
						"mode": "READ_WRITE",
						"source": "https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-b/disks/demo-disk-qq2",
						"deviceName": "persistent-disk-0",
						"index": 0,
						"boot": true,
						"autoDelete": false,
						"interface": "SCSI",
						"diskSizeGb": "10",
						"kind": "compute#attachedDisk"
					}
				],
				"metadata": {
					"fingerprint": "z=",
					"kind": "compute#metadata"
				},
				"selfLink": "https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-b/instances/demo-vm-tt2",
				"scheduling": {
					"onHostMaintenance": "MIGRATE",
					"automaticRestart": true,
					"preemptible": false
				},
				"cpuPlatform": "Intel Broadwell",
				"labelFingerprint": "z=",
				"startRestricted": false,
				"deletionProtection": false,
				"fingerprint": "z=",
				"lastStartTimestamp": "2021-03-10T11:02:37.848-08:00",
				"kind": "compute#instance"
			}
		],
		"selfLink": "https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-b/instances",
		"kind": "compute#instanceList"
	}
	`
	SimpleSelectGoogleContainerAggregatedSubnetworksResponse string = `
	{
		"subnetworks": [
			{
				"subnetwork": "projects/testing-project/regions/australia-southeast1/subnetworks/sn-02",
				"network": "projects/testing-project/global/networks/vpc-01",
				"ipCidrRange": "10.0.1.0/24"
			},
			{
				"subnetwork": "projects/testing-project/regions/australia-southeast1/subnetworks/sn-01",
				"network": "projects/testing-project/global/networks/vpc-01",
				"ipCidrRange": "10.0.0.0/24"
			}
		]
	}
	`
	//nolint:gosec // this is a test token
	GoogleAuthTokenResponse string = `{
		"access_token": "some-access-token",
		"token_type": "access_token",
		"id_token": "eyJhbGciOiJSUzI1NiIsImN0eSI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaWF0IjoxNjAzMzc2MDExLCJleHAiOjExNjI4NTY1MzY0LCJhdWQiOiJhdWQteCIsImlzcyI6Imdvb2dsZSIsInNjb3BlIjoiZ29vZ2xlYXBpcyJ9.g_MHGMGbRt0MZaOyKPA7zQNrYRDgabBJwEzUGlCHlWlidWnYSG9mo5YixHwk1AfeDsRTnxxyT9Ki1mSppamKbS_QHj-o54PMLibP6jcQV4aLxwug9cKbzIvQTndWPm41gBT4Bxfip4ZI9DZUtVZ4nv89reDdmZ_WLG_HuDw-3p4E5L_5iIJGGEnfyko8Da1LiHZg6tNGzpmMyjxUhocvUdM5iEHeppkLlGlu9Lw38UVxUCvskKy6WRCnLU7uCZxeoA-Ah8jg-Ie6IPdKm2UvqUflQbfG-Ga7LqzMxSVE_KvRD9_02mYZykjuWQiEAWqMYnBqK4TtoFfAZLTa1cFGvQ",
		"expires_in": 3600 
	}`
	SimpleOktaApplicationsAppsListResponseFile              string = "test/assets/response/okta/application/apps-list.json"
	SimpleGoogleComputeDisksListResponseFile                string = "test/assets/response/google/compute/disks/disks-list.json"
	SimpleGoogleComputeDisksListResponsePaginated5Page1File string = "test/assets/response/google/compute/disks/disks-list-paginated-5-max-page-01.json"
	SimpleGoogleComputeDisksListResponsePaginated5Page2File string = "test/assets/response/google/compute/disks/disks-list-paginated-5-max-page-02.json"
	SimpleGoogleComputeDisksListResponsePaginated5Page3File string = "test/assets/response/google/compute/disks/disks-list-paginated-5-max-page-03.json"
	GoogleCloudResourceManagerGetIamPolicyResponseFile      string = "test/assets/response/google/cloudresourcemanager/projects/organizations-getIamPolicy.json"
	GoogleCloudResourceManagerProjectsListResponseFile      string = "test/assets/response/google/cloudresourcemanager/projects/projects-list.json"
	SimpleGoogleBQDatasetInsertResponseFile01               string = "test/assets/response/google/bigquery/dataset/create/dataset-01.json"
	SimpleGoogleBQDatasetInsertResponseFile02               string = "test/assets/response/google/bigquery/dataset/create/dataset-02.json"
	GoogleContainerHost                                     string = "container.googleapis.com"
	GoogleComputeHost                                       string = "compute.googleapis.com"
	GoogleBQHost                                            string = "bigquery.googleapis.com"
	GoogleBQPRoject01                                       string = "testing-dummy-project"
	GoogleBQPRoject02                                       string = "testing-project"
	GoogleCloudResourceManagerHost                          string = "cloudresourcemanager.googleapis.com"
	GoogleProjectDefault                                    string = "stackql-demo"
	DiskInsertPath                                          string = "/compute/v1/projects/testing-project/zones/australia-southeast1-b/disks"
	NetworkInsertPath                                       string = "/compute/v1/projects/stackql-demo/global/networks"
	networkDeletePath                                       string = "/compute/v1/projects/%s/global/networks/%s"
	DiskInsertURL                                           string = "https://" + GoogleComputeHost + DiskInsertPath
	NetworkInsertURL                                        string = "https://" + GoogleComputeHost + NetworkInsertPath
	BQPRoject01InsertURL                                    string = "/bigquery/v2/projects/" + GoogleBQPRoject01 + "/datasets"
	BQPRoject02InsertURL                                    string = "/bigquery/v2/projects/" + GoogleBQPRoject02 + "/datasets"
	SubnetworkInsertPath                                    string = "/compute/v1/projects/stackql-demo/regions/australia-southeast1/subnetworks"
	IPInsertPath                                            string = "/compute/v1/projects/stackql-demo/regions/australia-southeast1/addresses"
	FirewallInsertPath                                      string = "/compute/v1/projects/stackql-demo/global/firewalls"
	ComputeInstanceInsertPath                               string = "/compute/v1/projects/stackql-demo/zones/australia-southeast1-a/instances"
	SubnetworkInsertURL                                     string = "https://" + GoogleComputeHost + NetworkInsertPath
	GoogleApisHost                                          string = "www.googleapis.com"
	GoogleComputeInsertOperationPath                        string = "/compute/v1/projects/stackql-demo/global/operations/operation-xxxxx-yyyyy-0001"
	GoogleComputeInsertOperationURL                         string = "https://" + GoogleApisHost + GoogleComputeInsertOperationPath
	simpleGoogleComputeOperationInitialResponse             string = `
	{
		"id": "8485551673440766140",
		"name": "operation-xxxxx-yyyyy-0001",
		"operationType": "%s",
		"targetLink": "%s",
		"targetId": "6645238333082165609",
		"status": "%s",
		"user": "test-user@gmail.com",
		"progress": 0,
		"insertTime": "2021-03-21T02:24:38.285-07:00",
		"startTime": "2021-03-21T02:24:38.293-07:00",
		"selfLink": "%s",
		"kind": "compute#operation"
	}
	`
	simpleGoogleComputePollOperationResponse string = `
	{
		"id": "8485551673440766140",
		"name": "operation-xxxxx-yyyyy-0001",
		"operationType": "%s",
		"targetLink": "%s",
		"targetId": "6645238333082165609",
		"status": "%s",
		"user": "test-user@gmail.com",
		"progress": 100,
		"insertTime": "2021-03-21T02:24:38.285-07:00",
		"startTime": "2021-03-21T02:24:38.293-07:00",
		"endTime": "2021-03-21T02:24:45.870-07:00",
		"selfLink": "%s",
		"kind": "compute#operation"
	}
	`
)

func GetSimpleGoogleNetworkInsertResponse() string {
	return fmt.Sprintf(
		simpleGoogleComputeOperationInitialResponse,
		"insert",
		NetworkInsertURL+"/kubernetes-the-hard-way-vpc",
		"RUNNING",
		GoogleComputeInsertOperationURL,
	)
}

func GetSimplePollOperationGoogleNetworkInsertResponse() string {
	return fmt.Sprintf(
		simpleGoogleComputePollOperationResponse,
		"insert",
		NetworkInsertURL+"/kubernetes-the-hard-way-vpc",
		"DONE",
		GoogleComputeInsertOperationURL,
	)
}

func GetSimpleGoogleNetworkDeleteResponse() string {
	return fmt.Sprintf(
		simpleGoogleComputeOperationInitialResponse,
		"delete",
		NetworkInsertURL+"/kubernetes-the-hard-way-vpc",
		"RUNNING",
		GoogleComputeInsertOperationURL,
	)
}

func GetSimplePollOperationGoogleNetworkDeleteResponse() string {
	return fmt.Sprintf(
		simpleGoogleComputePollOperationResponse,
		"delete",
		NetworkInsertURL+"/kubernetes-the-hard-way-vpc",
		"DONE",
		GoogleComputeInsertOperationURL,
	)
}

func GetSimpleGoogleSubnetworkInsertResponse() string {
	return fmt.Sprintf(
		simpleGoogleComputeOperationInitialResponse,
		"insert",
		NetworkInsertURL+"/kubernetes-the-hard-way-subnet",
		"RUNNING",
		GoogleComputeInsertOperationURL,
	)
}

func GetSimplePollOperationGoogleSubnetworkInsertResponse() string {
	return fmt.Sprintf(
		simpleGoogleComputePollOperationResponse,
		"insert",
		NetworkInsertURL+"/kubernetes-the-hard-way-subnet",
		"DONE",
		GoogleComputeInsertOperationURL,
	)
}

func GetSimpleGoogleIPInsertResponse() string {
	return fmt.Sprintf(
		simpleGoogleComputeOperationInitialResponse,
		"insert",
		NetworkInsertURL+"/kubernetes-the-hard-way-ip",
		"RUNNING",
		GoogleComputeInsertOperationURL,
	)
}

func GetSimplePollOperationGoogleIPInsertResponse() string {
	return fmt.Sprintf(
		simpleGoogleComputePollOperationResponse,
		"insert",
		NetworkInsertURL+"/kubernetes-the-hard-way-ip",
		"DONE",
		GoogleComputeInsertOperationURL,
	)
}

func GetSimpleGoogleFirewallInsertResponse() string {
	return fmt.Sprintf(
		simpleGoogleComputeOperationInitialResponse,
		"insert",
		NetworkInsertURL+"/kubernetes-the-hard-way-allow-internal-fw",
		"RUNNING",
		GoogleComputeInsertOperationURL,
	)
}

func GetSimplePollOperationGoogleFirewallInsertResponse() string {
	return fmt.Sprintf(
		simpleGoogleComputePollOperationResponse,
		"insert",
		NetworkInsertURL+"/kubernetes-the-hard-way-allow-internal-fw",
		"DONE",
		GoogleComputeInsertOperationURL,
	)
}

func GetSimpleGoogleComputeInstanceInsertResponse() string {
	return fmt.Sprintf(
		simpleGoogleComputeOperationInitialResponse,
		"insert",
		NetworkInsertURL+"/controller-0",
		"RUNNING",
		GoogleComputeInsertOperationURL,
	)
}

func GetSimplePollOperationGoogleComputeInstanceInsertResponse() string {
	return fmt.Sprintf(
		simpleGoogleComputePollOperationResponse,
		"insert",
		NetworkInsertURL+"/controller-0",
		"DONE",
		GoogleComputeInsertOperationURL,
	)
}

func GetSimpleNetworkDeletePath(proj string, network string) string {
	return fmt.Sprintf(networkDeletePath, proj, network)
}
