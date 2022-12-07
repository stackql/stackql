package stackqltestutil

import (
	"fmt"
	"io/ioutil"

	"net/url"
	"testing"

	"github.com/stackql/stackql/internal/stackql/asyncmonitor"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql/internal/test/testhttpapi"
	"github.com/stackql/stackql/internal/test/testobjects"
	"github.com/stackql/stackql/internal/test/testutil"
)

func SetupSimpleSelectGoogleComputeInstance(t *testing.T) {
	path := "/compute/v1/projects/testing-project/zones/australia-southeast1-b/instances"
	url := &url.URL{
		Path: path,
	}
	ex := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", url, testobjects.GoogleComputeHost, testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	expectations := testhttpapi.NewExpectationStore(1)
	expectations.Put(testobjects.GoogleComputeHost+path, ex)
	testhttpapi.StartServer(t, expectations)
	provider.DummyAuth = true
}

func SetupSelectOktaApplicationApps(t *testing.T) {
	responseFile1, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleOktaApplicationsAppsListResponseFile)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	responseBytes1, err := ioutil.ReadFile(responseFile1)
	if err != nil {
		t.Fatalf("%v", err)
	}
	path := "/api/v1/apps"
	url := &url.URL{
		Path: path,
	}
	ex := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", url, "some-silly-subdomain.okta.com", string(responseBytes1), nil)
	expectations := testhttpapi.NewExpectationStore(1)
	expectations.Put("some-silly-subdomain.okta.com"+path, ex)
	testhttpapi.StartServer(t, expectations)
	provider.DummyAuth = true
}

func getDisksSelectExpectations(t *testing.T) map[string]testhttpapi.HTTPRequestExpectations {
	path := "/compute/v1/projects/testing-project/zones/australia-southeast1-b/disks"
	url := &url.URL{
		Path: path,
	}
	responseFile, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleGoogleComputeDisksListResponseFile)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	responseBytes, err := ioutil.ReadFile(responseFile)
	if err != nil {
		t.Fatalf("%v", err)
	}
	ex := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", url, testobjects.GoogleComputeHost, string(responseBytes), nil)

	return map[string]testhttpapi.HTTPRequestExpectations{
		testobjects.GoogleComputeHost + path: ex,
	}
}

func getCloudResourceManagerOrganizationsGetIamPolicyExpectations(t *testing.T) map[string]testhttpapi.HTTPRequestExpectations {
	path := "/v3/organizations/123456789012:getIamPolicy"
	url := &url.URL{
		Path: path,
	}

	responseFile, err := util.GetFilePathFromRepositoryRoot(testobjects.GoogleCloudResourceManagerGetIamPolicyResponseFile)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	responseBytes, err := ioutil.ReadFile(responseFile)
	if err != nil {
		t.Fatalf("%v", err)
	}
	ex := testhttpapi.NewHTTPRequestExpectations(nil, nil, "POST", url, testobjects.GoogleCloudResourceManagerHost, string(responseBytes), nil)

	return map[string]testhttpapi.HTTPRequestExpectations{
		testobjects.GoogleCloudResourceManagerHost + path: ex,
	}
}

func getCloudResourceManagerProjectSelectExpectations(t *testing.T) map[string]testhttpapi.HTTPRequestExpectations {
	path := "/v3/projects"
	url := &url.URL{
		Path:     path,
		RawQuery: "parent=organizations%2F123456789012",
	}
	responseFile, err := util.GetFilePathFromRepositoryRoot(testobjects.GoogleCloudResourceManagerProjectsListResponseFile)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	responseBytes, err := ioutil.ReadFile(responseFile)
	if err != nil {
		t.Fatalf("%v", err)
	}
	ex := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", url, testobjects.GoogleCloudResourceManagerHost, string(responseBytes), nil)

	return map[string]testhttpapi.HTTPRequestExpectations{
		testobjects.GoogleCloudResourceManagerHost + path + "?parent=organizations%2F123456789012": ex,
	}
}

func SetupSimpleSelectGoogleComputeDisks(t *testing.T, requestCount int) {
	expectations := testhttpapi.NewExpectationStore(requestCount)
	for k, v := range getDisksSelectExpectations(t) {
		for i := 0; i < requestCount; i++ {
			expectations.Put(k, v)
		}
	}
	testhttpapi.StartServer(t, expectations)
	provider.DummyAuth = true
}

func SetupSimpleSelectGoogleComputeDisksPaginated(t *testing.T) {
	path := "/compute/v1/projects/testing-project/zones/australia-southeast1-b/disks"

	rawQuery1 := "maxResults=5"
	url1 := &url.URL{
		Path:     path,
		RawQuery: rawQuery1,
	}
	responseFile1, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleGoogleComputeDisksListResponsePaginated5Page1File)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	responseBytes1, err := ioutil.ReadFile(responseFile1)
	if err != nil {
		t.Fatalf("%v", err)
	}

	rawQuery2 := "maxResults=5&pageToken=Cg1jMi1zdGFuZGFyZC04"
	url2 := &url.URL{
		Path:     path,
		RawQuery: rawQuery2,
	}
	responseFile2, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleGoogleComputeDisksListResponsePaginated5Page2File)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	responseBytes2, err := ioutil.ReadFile(responseFile2)
	if err != nil {
		t.Fatalf("%v", err)
	}

	rawQuery3 := "maxResults=5&pageToken=Cg1jMi1zdGFuZGFyZC03"
	url3 := &url.URL{
		Path:     path,
		RawQuery: rawQuery3,
	}
	responseFile3, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleGoogleComputeDisksListResponsePaginated5Page3File)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	responseBytes3, err := ioutil.ReadFile(responseFile3)
	if err != nil {
		t.Fatalf("%v", err)
	}

	ex1 := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", url1, testobjects.GoogleComputeHost, string(responseBytes1), nil)
	ex2 := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", url2, testobjects.GoogleComputeHost, string(responseBytes2), nil)
	ex3 := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", url3, testobjects.GoogleComputeHost, string(responseBytes3), nil)

	expectations := testhttpapi.NewExpectationStore(3)
	expectations.Put(testobjects.GoogleComputeHost+path+"?"+rawQuery1, ex1)
	expectations.Put(testobjects.GoogleComputeHost+path+"?"+rawQuery2, ex2)
	expectations.Put(testobjects.GoogleComputeHost+path+"?"+rawQuery3, ex3)

	testhttpapi.StartServer(t, expectations)
	provider.DummyAuth = true
}

func SetupSimpleSelectGoogleContainerAggAllowedSubnetworks(t *testing.T) {
	path := "/v1/projects/testing-project/aggregated/usableSubnetworks"
	url := &url.URL{
		Path: path,
	}
	ex := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", url, testobjects.GoogleContainerHost, testobjects.SimpleSelectGoogleContainerAggregatedSubnetworksResponse, nil)
	expectations := testhttpapi.NewExpectationStore(1)
	expectations.Put(testobjects.GoogleContainerHost+path, ex)
	testhttpapi.StartServer(t, expectations)
	provider.DummyAuth = true
}

func getNetworkInsertSuccessExpectations() map[string]testhttpapi.HTTPRequestExpectations {
	networkInsertURL := &url.URL{
		Path: testobjects.NetworkInsertPath,
	}
	networkInsertExpectation := testhttpapi.NewHTTPRequestExpectations(
		testutil.CreateReadCloserFromString(testobjects.CreateGoogleComputeNetworkRequestPayload),
		nil,
		"POST",
		networkInsertURL,
		testobjects.GoogleComputeHost,
		testobjects.GetSimpleGoogleNetworkInsertResponse(),
		nil,
	)

	networkInsertOpPollURL := &url.URL{
		Path: testobjects.GoogleComputeInsertOperationPath,
	}
	networkInsertOpPollExpectation := testhttpapi.NewHTTPRequestExpectations(
		nil,
		nil,
		"GET",
		networkInsertOpPollURL,
		testobjects.GoogleApisHost,
		testobjects.GetSimplePollOperationGoogleNetworkInsertResponse(),
		nil,
	)

	return map[string]testhttpapi.HTTPRequestExpectations{
		testobjects.GoogleComputeHost + testobjects.NetworkInsertPath:             networkInsertExpectation,
		testobjects.GoogleApisHost + testobjects.GoogleComputeInsertOperationPath: networkInsertOpPollExpectation,
	}
}

func getDiskInsertSuccessExpectations(expectedRequestBody string) map[string]testhttpapi.HTTPRequestExpectations {
	diskInsertURL := &url.URL{
		Path: testobjects.DiskInsertPath,
	}
	diskInsertExpectation := testhttpapi.NewHTTPRequestExpectations(
		testutil.CreateReadCloserFromString(expectedRequestBody),
		nil,
		"POST",
		diskInsertURL,
		testobjects.GoogleComputeHost,
		testobjects.GetSimpleGoogleNetworkInsertResponse(),
		nil,
	)

	diskInsertOpPollURL := &url.URL{
		Path: testobjects.GoogleComputeInsertOperationPath,
	}
	networkInsertOpPollExpectation := testhttpapi.NewHTTPRequestExpectations(
		nil,
		nil,
		"GET",
		diskInsertOpPollURL,
		testobjects.GoogleApisHost,
		testobjects.GetSimplePollOperationGoogleNetworkInsertResponse(),
		nil,
	)

	return map[string]testhttpapi.HTTPRequestExpectations{
		testobjects.GoogleComputeHost + testobjects.DiskInsertPath:                diskInsertExpectation,
		testobjects.GoogleApisHost + testobjects.GoogleComputeInsertOperationPath: networkInsertOpPollExpectation,
	}
}

func getBQDatasetInsertSuccessExpectations(bqInsertPath, expectedRequestBody, responseBody string) map[string]testhttpapi.HTTPRequestExpectations {
	datasetInsertURL := &url.URL{
		Path: bqInsertPath,
	}
	datasetInsertExpectation := testhttpapi.NewHTTPRequestExpectations(
		testutil.CreateReadCloserFromString(expectedRequestBody),
		nil,
		"POST",
		datasetInsertURL,
		testobjects.GoogleBQHost,
		responseBody,
		nil,
	)

	return map[string]testhttpapi.HTTPRequestExpectations{
		testobjects.GoogleBQHost + bqInsertPath: datasetInsertExpectation,
	}
}

func getSubnetworkInsertSuccessExpectations() map[string]testhttpapi.HTTPRequestExpectations {
	networkInsertURL := &url.URL{
		Path: testobjects.SubnetworkInsertPath,
	}
	networkInsertExpectation := testhttpapi.NewHTTPRequestExpectations(
		testutil.CreateReadCloserFromString(testobjects.CreateGoogleComputeSubnetworkRequestPayload),
		nil,
		"POST",
		networkInsertURL,
		testobjects.GoogleComputeHost,
		testobjects.GetSimpleGoogleSubnetworkInsertResponse(),
		nil,
	)

	networkInsertOpPollURL := &url.URL{
		Path: testobjects.GoogleComputeInsertOperationPath,
	}
	networkInsertOpPollExpectation := testhttpapi.NewHTTPRequestExpectations(
		nil,
		nil,
		"GET",
		networkInsertOpPollURL,
		testobjects.GoogleApisHost,
		testobjects.GetSimplePollOperationGoogleSubnetworkInsertResponse(),
		nil,
	)

	return map[string]testhttpapi.HTTPRequestExpectations{
		testobjects.GoogleComputeHost + testobjects.SubnetworkInsertPath:          networkInsertExpectation,
		testobjects.GoogleApisHost + testobjects.GoogleComputeInsertOperationPath: networkInsertOpPollExpectation,
	}
}

func getIPInsertSuccessExpectations() map[string]testhttpapi.HTTPRequestExpectations {
	networkInsertURL := &url.URL{
		Path: testobjects.IPInsertPath,
	}
	networkInsertExpectation := testhttpapi.NewHTTPRequestExpectations(
		testutil.CreateReadCloserFromString(testobjects.CreateGoogleComputeIPRequestPayload),
		nil,
		"POST",
		networkInsertURL,
		testobjects.GoogleComputeHost,
		testobjects.GetSimpleGoogleIPInsertResponse(),
		nil,
	)

	networkInsertOpPollURL := &url.URL{
		Path: testobjects.GoogleComputeInsertOperationPath,
	}
	networkInsertOpPollExpectation := testhttpapi.NewHTTPRequestExpectations(
		nil,
		nil,
		"GET",
		networkInsertOpPollURL,
		testobjects.GoogleApisHost,
		testobjects.GetSimplePollOperationGoogleIPInsertResponse(),
		nil,
	)

	return map[string]testhttpapi.HTTPRequestExpectations{
		testobjects.GoogleComputeHost + testobjects.IPInsertPath:                  networkInsertExpectation,
		testobjects.GoogleApisHost + testobjects.GoogleComputeInsertOperationPath: networkInsertOpPollExpectation,
	}
}

func getInternalFirewallInsertSuccessExpectations() map[string]testhttpapi.HTTPRequestExpectations {
	networkInsertURL := &url.URL{
		Path: testobjects.FirewallInsertPath,
	}
	networkInsertExpectation := testhttpapi.NewHTTPRequestExpectations(
		testutil.CreateReadCloserFromString(testobjects.CreateGoogleComputeInternalFirewallRequestPayload),
		nil,
		"POST",
		networkInsertURL,
		testobjects.GoogleComputeHost,
		testobjects.GetSimpleGoogleFirewallInsertResponse(),
		nil,
	)

	networkInsertOpPollURL := &url.URL{
		Path: testobjects.GoogleComputeInsertOperationPath,
	}
	networkInsertOpPollExpectation := testhttpapi.NewHTTPRequestExpectations(
		nil,
		nil,
		"GET",
		networkInsertOpPollURL,
		testobjects.GoogleApisHost,
		testobjects.GetSimplePollOperationGoogleFirewallInsertResponse(),
		nil,
	)

	return map[string]testhttpapi.HTTPRequestExpectations{
		testobjects.GoogleComputeHost + testobjects.FirewallInsertPath:            networkInsertExpectation,
		testobjects.GoogleApisHost + testobjects.GoogleComputeInsertOperationPath: networkInsertOpPollExpectation,
	}
}

func getExternalFirewallInsertSuccessExpectations() map[string]testhttpapi.HTTPRequestExpectations {
	networkInsertURL := &url.URL{
		Path: testobjects.FirewallInsertPath,
	}
	networkInsertExpectation := testhttpapi.NewHTTPRequestExpectations(
		testutil.CreateReadCloserFromString(testobjects.CreateGoogleComputeExternalFirewallRequestPayload),
		nil,
		"POST",
		networkInsertURL,
		testobjects.GoogleComputeHost,
		testobjects.GetSimpleGoogleFirewallInsertResponse(),
		nil,
	)

	networkInsertOpPollURL := &url.URL{
		Path: testobjects.GoogleComputeInsertOperationPath,
	}
	networkInsertOpPollExpectation := testhttpapi.NewHTTPRequestExpectations(
		nil,
		nil,
		"GET",
		networkInsertOpPollURL,
		testobjects.GoogleApisHost,
		testobjects.GetSimplePollOperationGoogleFirewallInsertResponse(),
		nil,
	)

	return map[string]testhttpapi.HTTPRequestExpectations{
		testobjects.GoogleComputeHost + testobjects.FirewallInsertPath:            networkInsertExpectation,
		testobjects.GoogleApisHost + testobjects.GoogleComputeInsertOperationPath: networkInsertOpPollExpectation,
	}
}

func getComputeInstanceInsertSuccessExpectations(name string, secondaryTag string, networkIP string) map[string]testhttpapi.HTTPRequestExpectations {
	networkInsertURL := &url.URL{
		Path: testobjects.ComputeInstanceInsertPath,
	}
	networkInsertExpectation := testhttpapi.NewHTTPRequestExpectations(
		testutil.CreateReadCloserFromString(testobjects.GetCreateGoogleComputeInstancePayload(name, secondaryTag, networkIP)),
		nil,
		"POST",
		networkInsertURL,
		testobjects.GoogleComputeHost,
		testobjects.GetSimpleGoogleComputeInstanceInsertResponse(),
		nil,
	)

	networkInsertOpPollURL := &url.URL{
		Path: testobjects.GoogleComputeInsertOperationPath,
	}
	networkInsertOpPollExpectation := testhttpapi.NewHTTPRequestExpectations(
		nil,
		nil,
		"GET",
		networkInsertOpPollURL,
		testobjects.GoogleApisHost,
		testobjects.GetSimplePollOperationGoogleComputeInstanceInsertResponse(),
		nil,
	)

	return map[string]testhttpapi.HTTPRequestExpectations{
		testobjects.GoogleComputeHost + testobjects.ComputeInstanceInsertPath:     networkInsertExpectation,
		testobjects.GoogleApisHost + testobjects.GoogleComputeInsertOperationPath: networkInsertOpPollExpectation,
	}
}

func getNetworkDeleteSuccessExpectations() map[string]testhttpapi.HTTPRequestExpectations {
	path := testobjects.GetSimpleNetworkDeletePath(testobjects.GoogleProjectDefault, "kubernetes-the-hard-way-vpc")
	networkDeleteURL := &url.URL{
		Path: path,
	}
	networkDeleteExpectation := testhttpapi.NewHTTPRequestExpectations(
		nil,
		nil,
		"DELETE",
		networkDeleteURL,
		testobjects.GoogleComputeHost,
		testobjects.GetSimpleGoogleNetworkDeleteResponse(),
		nil,
	)

	networkInsertOpPollURL := &url.URL{
		Path: testobjects.GoogleComputeInsertOperationPath,
	}
	networkInsertOpPollExpectation := testhttpapi.NewHTTPRequestExpectations(
		nil,
		nil,
		"GET",
		networkInsertOpPollURL,
		testobjects.GoogleApisHost,
		testobjects.GetSimplePollOperationGoogleNetworkDeleteResponse(),
		nil,
	)

	return map[string]testhttpapi.HTTPRequestExpectations{
		testobjects.GoogleComputeHost + path:                                      networkDeleteExpectation,
		testobjects.GoogleApisHost + testobjects.GoogleComputeInsertOperationPath: networkInsertOpPollExpectation,
	}
}

func SetupSimpleInsertGoogleComputeNetworks(t *testing.T) {

	expectations := testhttpapi.NewExpectationStore(3)
	for k, v := range getNetworkInsertSuccessExpectations() {
		expectations.Put(k, v)
	}
	testhttpapi.StartServer(t, expectations)
	provider.DummyAuth = true
	asyncmonitor.MonitorPollIntervalSeconds = 2
}

func SetupDependentInsertGoogleComputeDisks(t *testing.T) {

	expectations := testhttpapi.NewExpectationStore(5)
	for k, v := range getDisksSelectExpectations(t) {
		expectations.Put(k, v)
	}
	for k, v := range getDiskInsertSuccessExpectations(testobjects.CreateGoogleComputeDiskRequestPayload01) {
		expectations.Put(k, v)
	}
	for k, v := range getDiskInsertSuccessExpectations(testobjects.CreateGoogleComputeDiskRequestPayload02) {
		expectations.Put(k, v)
	}
	testhttpapi.StartServer(t, expectations)
	provider.DummyAuth = true
	asyncmonitor.MonitorPollIntervalSeconds = 2
}

func SetupExecGoogleOrganizationsGetIamPolicy(t *testing.T) {

	expectations := testhttpapi.NewExpectationStore(1)
	for k, v := range getCloudResourceManagerOrganizationsGetIamPolicyExpectations(t) {
		expectations.Put(k, v)
	}
	testhttpapi.StartServer(t, expectations)
	provider.DummyAuth = true
}

func SetupDependentInsertGoogleComputeDisksReversed(t *testing.T) {

	expectations := testhttpapi.NewExpectationStore(5)
	for k, v := range getCloudResourceManagerProjectSelectExpectations(t) {
		expectations.Put(k, v)
	}
	for k, v := range getDiskInsertSuccessExpectations(testobjects.CreateGoogleComputeDiskRequestPayload03) {
		expectations.Put(k, v)
	}
	for k, v := range getDiskInsertSuccessExpectations(testobjects.CreateGoogleComputeDiskRequestPayload04) {
		expectations.Put(k, v)
	}
	testhttpapi.StartServer(t, expectations)
	provider.DummyAuth = true
	asyncmonitor.MonitorPollIntervalSeconds = 2
}

func SetupDependentInsertGoogleBQDatasets(t *testing.T) {

	expectations := testhttpapi.NewExpectationStore(3)
	for k, v := range getCloudResourceManagerProjectSelectExpectations(t) {
		expectations.Put(k, v)
	}
	responseFile01, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleGoogleBQDatasetInsertResponseFile01)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	responseBytes01, err := ioutil.ReadFile(responseFile01)
	if err != nil {
		t.Fatalf("%v", err)
	}
	responseFile02, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleGoogleBQDatasetInsertResponseFile02)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	responseBytes02, err := ioutil.ReadFile(responseFile02)
	if err != nil {
		t.Fatalf("%v", err)
	}
	for k, v := range getBQDatasetInsertSuccessExpectations(testobjects.BQPRoject01InsertURL, testobjects.CreateGoogleBQDatasetRequestPayload01, string(responseBytes01)) {
		expectations.Put(k, v)
	}
	for k, v := range getBQDatasetInsertSuccessExpectations(testobjects.BQPRoject02InsertURL, testobjects.CreateGoogleBQDatasetRequestPayload02, string(responseBytes02)) {
		expectations.Put(k, v)
	}
	testhttpapi.StartServer(t, expectations)
	provider.DummyAuth = true
	asyncmonitor.MonitorPollIntervalSeconds = 2
}

func SetupSimpleDeleteGoogleComputeNetworks(t *testing.T) {

	expectations := testhttpapi.NewExpectationStore(3)
	for k, v := range getNetworkDeleteSuccessExpectations() {
		expectations.Put(k, v)
	}
	testhttpapi.StartServer(t, expectations)
	provider.DummyAuth = true
	asyncmonitor.MonitorPollIntervalSeconds = 2
}

func SetupK8sTheHardWayE2eSuccess(t *testing.T) {

	computeControllerInstanceCount := 3
	computeWorkerInstanceCount := 3

	expectations := testhttpapi.NewExpectationStore(30)
	for k, v := range getNetworkInsertSuccessExpectations() {
		expectations.Put(k, v)
	}
	for k, v := range getSubnetworkInsertSuccessExpectations() {
		expectations.Put(k, v)
	}
	for k, v := range getIPInsertSuccessExpectations() {
		expectations.Put(k, v)
	}
	for k, v := range getInternalFirewallInsertSuccessExpectations() {
		expectations.Put(k, v)
	}
	for k, v := range getExternalFirewallInsertSuccessExpectations() {
		expectations.Put(k, v)
	}
	for i := 0; i < computeControllerInstanceCount; i++ {
		for k, v := range getComputeInstanceInsertSuccessExpectations(fmt.Sprintf("controller-%d", i), "controller", fmt.Sprintf("10.240.0.%d", 10+i)) {
			expectations.Put(k, v)
		}
	}
	for i := 0; i < computeWorkerInstanceCount; i++ {
		for k, v := range getComputeInstanceInsertSuccessExpectations(fmt.Sprintf("worker-%d", i), "worker", fmt.Sprintf("10.240.0.%d", 20+i)) {
			expectations.Put(k, v)
		}
	}
	testhttpapi.StartServer(t, expectations)
	provider.DummyAuth = true
	asyncmonitor.MonitorPollIntervalSeconds = 2
}

func SetupInsertDependentComputeDiskSuccess(t *testing.T) {

	computeControllerInstanceCount := 3
	computeWorkerInstanceCount := 3

	expectations := testhttpapi.NewExpectationStore(30)
	for k, v := range getNetworkInsertSuccessExpectations() {
		expectations.Put(k, v)
	}
	for k, v := range getSubnetworkInsertSuccessExpectations() {
		expectations.Put(k, v)
	}
	for k, v := range getIPInsertSuccessExpectations() {
		expectations.Put(k, v)
	}
	for k, v := range getInternalFirewallInsertSuccessExpectations() {
		expectations.Put(k, v)
	}
	for k, v := range getExternalFirewallInsertSuccessExpectations() {
		expectations.Put(k, v)
	}
	for i := 0; i < computeControllerInstanceCount; i++ {
		for k, v := range getComputeInstanceInsertSuccessExpectations(fmt.Sprintf("controller-%d", i), "controller", fmt.Sprintf("10.240.0.%d", 10+i)) {
			expectations.Put(k, v)
		}
	}
	for i := 0; i < computeWorkerInstanceCount; i++ {
		for k, v := range getComputeInstanceInsertSuccessExpectations(fmt.Sprintf("worker-%d", i), "worker", fmt.Sprintf("10.240.0.%d", 20+i)) {
			expectations.Put(k, v)
		}
	}
	testhttpapi.StartServer(t, expectations)
	provider.DummyAuth = true
	asyncmonitor.MonitorPollIntervalSeconds = 2
}
