package testobjects

import (
	"fmt"
)

//nolint:lll // This is a test file
const (
	CreateGoogleComputeDiskRequestPayload01  string = `{"name":"demo-disk-qq1-new16"}`
	CreateGoogleComputeDiskRequestPayload02  string = `{"name":"demo-disk-qq2-new16"}`
	CreateGoogleComputeDiskRequestPayload03  string = `{"name":"demo-disk-xx5-new16"}`
	CreateGoogleComputeDiskRequestPayload04  string = `{"name":"demo-disk-xx4-new16"}`
	CreateGoogleComputeNetworkRequestPayload string = `
	{
		"name": "kubernetes-the-hard-way-vpc",
		"autoCreateSubnetworks": false,
		"routingConfig": {
			"routingMode": "REGIONAL"
		}
	}
	`
	CreateGoogleComputeSubnetworkRequestPayload string = `
	{
		"name": "kubernetes-the-hard-way-subnet",
		"network": "https://compute.googleapis.com/compute/v1/projects/stackql-demo/global/networks/kubernetes-the-hard-way-vpc",
		"ipCidrRange": "10.240.0.0/24",
		"privateIpGoogleAccess": false
	}
	`
	CreateGoogleComputeIPRequestPayload string = `
	{
		"name": "kubernetes-the-hard-way-ip"
	}`
	CreateGoogleComputeInternalFirewallRequestPayload string = `
	{
		"name": "kubernetes-the-hard-way-allow-internal-fw",
		"network": "https://compute.googleapis.com/compute/v1/projects/stackql-demo/global/networks/kubernetes-the-hard-way-vpc",
		"direction": "INGRESS",
		"sourceRanges": ["10.240.0.0/24","10.200.0.0/16"],
		"allowed": [{"IPProtocol":"tcp"},{"IPProtocol":"udp"},{"IPProtocol":"icmp"}]
	}
	`
	CreateGoogleComputeExternalFirewallRequestPayload string = `
	{
		"name": "kubernetes-the-hard-way-allow-external-fw",
		"network": "https://compute.googleapis.com/compute/v1/projects/stackql-demo/global/networks/kubernetes-the-hard-way-vpc",
		"direction": "INGRESS",
		"sourceRanges": ["0.0.0.0/0"],
		"allowed": [{"IPProtocol":"tcp","ports":["22"]},{"IPProtocol":"tcp","ports":["6443"]},{"IPProtocol":"icmp"}]
	}
	`
	createGoogleComputeInstancePayload string = `
	{
		"name": "%s",
		"machineType": "https://compute.googleapis.com/compute/v1/projects/stackql-demo/zones/australia-southeast1-a/machineTypes/f1-micro",
		"canIpForward": true,
		"deletionProtection": false,
		"scheduling": {"automaticRestart":true},
		"networkInterfaces": [{"network":"projects/stackql-demo/global/networks/kubernetes-the-hard-way-vpc","networkIP":"%s","subnetwork":"regions/australia-southeast1/subnetworks/kubernetes-the-hard-way-subnet"}],
		"disks": [{"autoDelete":true,"boot":true,"initializeParams":{"diskSizeGb":"10","sourceImage":"https://compute.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/family/ubuntu-2004-lts"},"mode":"READ_WRITE","type":"PERSISTENT"}],
		"serviceAccounts": [{"email":"default","scopes":["https://www.googleapis.com/auth/compute","https://www.googleapis.com/auth/devstorage.read_only","https://www.googleapis.com/auth/logging.write","https://www.googleapis.com/auth/monitoring","https://www.googleapis.com/auth/service.management.readonly","https://www.googleapis.com/auth/servicecontrol"]}],
		"tags": {"items":["kubernetes-the-hard-way","%s"]}
	}
	`
)

//nolint:revive,gochecknoglobals // This is a test file
var (
	CreateGoogleBQDatasetRequestPayload01 string = fmt.Sprintf(`
	{
		"datasetReference": {"datasetId":"test_dataset_zz","projectId":"%s"},
		"location": "US"
	}`, GoogleBQPRoject01)
	CreateGoogleBQDatasetRequestPayload02 string = fmt.Sprintf(`
	{
		"datasetReference": {"datasetId":"test_dataset_zz","projectId":"%s"},
		"location": "US"
	}`, GoogleBQPRoject02)
)

func GetCreateGoogleComputeInstancePayload(name string, secondaryTag string, netWorkIP string) string {
	return fmt.Sprintf(createGoogleComputeInstancePayload, name, netWorkIP, secondaryTag)
}
