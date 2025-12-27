package driver_test

import (
	"fmt"
	"net/url"
	"testing"

	. "github.com/stackql/stackql/internal/stackql/driver"

	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/provider"

	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testhttpapi"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "github.com/stackql/stackql-parser/go/cache"
)

//nolint:lll // legacy test
func BenchmarkSelectGoogleComputeInstanceDriver(b *testing.B) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "BenchmarkSelectGoogleComputeInstanceDriver")
	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}
	path := "/compute/v1/projects/testing-project/zones/australia-southeast1-b/instances"
	url := &url.URL{
		Path: path,
	}
	ex := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", url, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	expectations := map[string]testhttpapi.HTTPRequestExpectations{
		"compute.googleapis.com" + path: ex,
	}
	exp := testhttpapi.NewExpectationStore(1)
	for k, v := range expectations {
		exp.Put(k, v)
	}
	testhttpapi.StartServer(b, exp)
	provider.DummyAuth = true

	inputBundle, err := stackqltestutil.BuildBenchInputBundle(*runtimeCtx)
	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}

	stringQuery := `select count(1) as inst_count from google.compute.instances where zone = 'australia-southeast1-b' AND /* */ project = 'testing-project';`

	runtimeCtx.LogLevelStr = "fatal"
	handlerCtx, err := handler.NewHandlerCtx(
		stringQuery, *runtimeCtx, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)),
		inputBundle, "v0.1.1")

	dr, _ := NewStackQLDriver(handlerCtx)

	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}

	dr.ProcessQuery(handlerCtx.GetRawQuery())

	// b.Logf("benchmark select driver integration test passed")
}

//nolint:lll // legacy test
func BenchmarkParallelProjectSelectGoogleComputeInstanceDriver(b *testing.B) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "BenchmarkParallelProjectSelectGoogleComputeInstanceDriver")
	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}
	path := `/compute/v1/projects/%s/zones/australia-southeast1-b/instances`
	pathOne := fmt.Sprintf(path, "testing-project")
	urlOne := &url.URL{
		Path: pathOne,
	}
	exOne := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlOne, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	pathTwo := fmt.Sprintf(path, "testing-project-two")
	urlTwo := &url.URL{
		Path: pathTwo,
	}
	exTwo := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlTwo, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	expectations := map[string]testhttpapi.HTTPRequestExpectations{
		"compute.googleapis.com" + pathOne: exOne,
		"compute.googleapis.com" + pathTwo: exTwo,
	}
	exp := testhttpapi.NewExpectationStore(2)
	for k, v := range expectations {
		exp.Put(k, v)
	}
	testhttpapi.StartServer(b, exp)
	provider.DummyAuth = true

	inputBundle, err := stackqltestutil.BuildBenchInputBundle(*runtimeCtx)
	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}

	stringQuery := `select count(1) as inst_count from google.compute.instances where zone = 'australia-southeast1-b' AND /* */ project in ('testing-project', 'testing-project-two');`

	handlerCtx, err := handler.NewHandlerCtx(
		stringQuery, *runtimeCtx, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)),
		inputBundle, "v0.1.1")

	dr, _ := NewStackQLDriver(handlerCtx)

	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}

	dr.ProcessQuery(handlerCtx.GetRawQuery())

	// b.Logf("benchmark parallel select driver integration test passed")
}

//nolint:lll // legacy test
func BenchmarkHighlyParallelProjectSelectGoogleComputeInstanceDriver(b *testing.B) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "BenchmarkHighlyParallelProjectSelectGoogleComputeInstanceDriver")
	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}
	path := `/compute/v1/projects/%s/zones/australia-southeast1-b/instances`
	// scenario one
	pathOne := fmt.Sprintf(path, "testing-project")
	urlOne := &url.URL{
		Path: pathOne,
	}
	exOne := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlOne, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario two
	pathTwo := fmt.Sprintf(path, "testing-project-two")
	urlTwo := &url.URL{
		Path: pathTwo,
	}
	exTwo := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlTwo, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario three
	pathThree := fmt.Sprintf(path, "testing-project-three")
	urlThree := &url.URL{
		Path: pathThree,
	}
	exThree := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlThree, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario four
	pathFour := fmt.Sprintf(path, "testing-project-four")
	urlFour := &url.URL{
		Path: pathFour,
	}
	exFour := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlFour, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario five
	pathFive := fmt.Sprintf(path, "testing-project-five")
	urlFive := &url.URL{
		Path: pathFive,
	}
	exFive := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlFive, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario six
	pathSix := fmt.Sprintf(path, "testing-project-six")
	urlSix := &url.URL{
		Path: pathSix,
	}
	exSix := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlSix, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario seven
	pathSeven := fmt.Sprintf(path, "testing-project-seven")
	urlSeven := &url.URL{
		Path: pathSeven,
	}
	exSeven := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlSeven, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario eight
	pathEight := fmt.Sprintf(path, "testing-project-eight")
	urlEight := &url.URL{
		Path: pathEight,
	}
	exEight := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlEight, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario nine
	pathNine := fmt.Sprintf(path, "testing-project-nine")
	urlNine := &url.URL{
		Path: pathNine,
	}
	exNine := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlNine, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario ten
	pathTen := fmt.Sprintf(path, "testing-project-ten")
	urlTen := &url.URL{
		Path: pathTen,
	}
	exTen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlTen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario eleven
	pathEleven := fmt.Sprintf(path, "testing-project-eleven")
	urlEleven := &url.URL{
		Path: pathEleven,
	}
	exEleven := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlEleven, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario twelve
	pathTwelve := fmt.Sprintf(path, "testing-project-twelve")
	urlTwelve := &url.URL{
		Path: pathTwelve,
	}
	exTwelve := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlTwelve, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario thirteen
	pathThirteen := fmt.Sprintf(path, "testing-project-thirteen")
	urlThirteen := &url.URL{
		Path: pathThirteen,
	}
	exThirteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlThirteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario fourteen
	pathFourteen := fmt.Sprintf(path, "testing-project-fourteen")
	urlFourteen := &url.URL{
		Path: pathFourteen,
	}
	exFourteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlFourteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario fifteen
	pathFifteen := fmt.Sprintf(path, "testing-project-fifteen")
	urlFifteen := &url.URL{
		Path: pathFifteen,
	}
	exFifteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlFifteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario sixteen
	pathSixteen := fmt.Sprintf(path, "testing-project-sixteen")
	urlSixteen := &url.URL{
		Path: pathSixteen,
	}
	exSixteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlSixteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario seventeen
	pathSeventeen := fmt.Sprintf(path, "testing-project-seventeen")
	urlSeventeen := &url.URL{
		Path: pathSeventeen,
	}
	exSeventeen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlSeventeen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario eighteen
	pathEighteen := fmt.Sprintf(path, "testing-project-eighteen")
	urlEighteen := &url.URL{
		Path: pathEighteen,
	}
	exEighteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlEighteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario nineteen
	pathNineteen := fmt.Sprintf(path, "testing-project-nineteen")
	urlNineteen := &url.URL{
		Path: pathNineteen,
	}
	exNineteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlNineteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario twenty
	pathTwenty := fmt.Sprintf(path, "testing-project-twenty")
	urlTwenty := &url.URL{
		Path: pathTwenty,
	}
	exTwenty := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlTwenty, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)

	// expectations
	expectations := map[string]testhttpapi.HTTPRequestExpectations{
		"compute.googleapis.com" + pathOne:       exOne,
		"compute.googleapis.com" + pathTwo:       exTwo,
		"compute.googleapis.com" + pathThree:     exThree,
		"compute.googleapis.com" + pathFour:      exFour,
		"compute.googleapis.com" + pathFive:      exFive,
		"compute.googleapis.com" + pathSix:       exSix,
		"compute.googleapis.com" + pathSeven:     exSeven,
		"compute.googleapis.com" + pathEight:     exEight,
		"compute.googleapis.com" + pathNine:      exNine,
		"compute.googleapis.com" + pathTen:       exTen,
		"compute.googleapis.com" + pathEleven:    exEleven,
		"compute.googleapis.com" + pathTwelve:    exTwelve,
		"compute.googleapis.com" + pathThirteen:  exThirteen,
		"compute.googleapis.com" + pathFourteen:  exFourteen,
		"compute.googleapis.com" + pathFifteen:   exFifteen,
		"compute.googleapis.com" + pathSixteen:   exSixteen,
		"compute.googleapis.com" + pathSeventeen: exSeventeen,
		"compute.googleapis.com" + pathEighteen:  exEighteen,
		"compute.googleapis.com" + pathNineteen:  exNineteen,
		"compute.googleapis.com" + pathTwenty:    exTwenty,
	}
	exp := testhttpapi.NewExpectationStore(20)
	for k, v := range expectations {
		exp.Put(k, v)
	}
	testhttpapi.StartServer(b, exp)
	provider.DummyAuth = true

	// runtimeCtx.ExecutionConcurrencyLimit = -1

	inputBundle, err := stackqltestutil.BuildBenchInputBundle(*runtimeCtx)
	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}

	stringQuery := `
	  select count(1) as inst_count 
	  from google.compute.instances 
	  where 
	    zone = 'australia-southeast1-b' 
		AND /* */ project in 
		  (
			'testing-project', 
			'testing-project-two',
			'testing-project-three',
			'testing-project-four',
			'testing-project-five',
			'testing-project-six',
			'testing-project-seven',
			'testing-project-eight',
			'testing-project-nine',
			'testing-project-ten',
			'testing-project-eleven',
			'testing-project-twelve',
			'testing-project-thirteen',
			'testing-project-fourteen',
			'testing-project-fifteen',
			'testing-project-sixteen',
			'testing-project-seventeen',
			'testing-project-eighteen',
			'testing-project-nineteen',
			'testing-project-twenty'
		  )
	  ;`

	handlerCtx, err := handler.NewHandlerCtx(
		stringQuery, *runtimeCtx, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)),
		inputBundle, "v0.1.1")

	dr, _ := NewStackQLDriver(handlerCtx)

	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}

	dr.ProcessQuery(handlerCtx.GetRawQuery())

	// b.Logf("benchmark parallel select driver integration test passed")
}

//nolint:lll // legacy test
func BenchmarkLoosenedHighlyParallelProjectSelectGoogleComputeInstanceDriver(b *testing.B) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "BenchmarkLoosenedHighlyParallelProjectSelectGoogleComputeInstanceDriver")
	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}
	path := `/compute/v1/projects/%s/zones/australia-southeast1-b/instances`
	// scenario one
	pathOne := fmt.Sprintf(path, "testing-project")
	urlOne := &url.URL{
		Path: pathOne,
	}
	exOne := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlOne, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario two
	pathTwo := fmt.Sprintf(path, "testing-project-two")
	urlTwo := &url.URL{
		Path: pathTwo,
	}
	exTwo := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlTwo, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario three
	pathThree := fmt.Sprintf(path, "testing-project-three")
	urlThree := &url.URL{
		Path: pathThree,
	}
	exThree := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlThree, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario four
	pathFour := fmt.Sprintf(path, "testing-project-four")
	urlFour := &url.URL{
		Path: pathFour,
	}
	exFour := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlFour, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario five
	pathFive := fmt.Sprintf(path, "testing-project-five")
	urlFive := &url.URL{
		Path: pathFive,
	}
	exFive := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlFive, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario six
	pathSix := fmt.Sprintf(path, "testing-project-six")
	urlSix := &url.URL{
		Path: pathSix,
	}
	exSix := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlSix, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario seven
	pathSeven := fmt.Sprintf(path, "testing-project-seven")
	urlSeven := &url.URL{
		Path: pathSeven,
	}
	exSeven := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlSeven, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario eight
	pathEight := fmt.Sprintf(path, "testing-project-eight")
	urlEight := &url.URL{
		Path: pathEight,
	}
	exEight := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlEight, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario nine
	pathNine := fmt.Sprintf(path, "testing-project-nine")
	urlNine := &url.URL{
		Path: pathNine,
	}
	exNine := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlNine, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario ten
	pathTen := fmt.Sprintf(path, "testing-project-ten")
	urlTen := &url.URL{
		Path: pathTen,
	}
	exTen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlTen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario eleven
	pathEleven := fmt.Sprintf(path, "testing-project-eleven")
	urlEleven := &url.URL{
		Path: pathEleven,
	}
	exEleven := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlEleven, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario twelve
	pathTwelve := fmt.Sprintf(path, "testing-project-twelve")
	urlTwelve := &url.URL{
		Path: pathTwelve,
	}
	exTwelve := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlTwelve, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario thirteen
	pathThirteen := fmt.Sprintf(path, "testing-project-thirteen")
	urlThirteen := &url.URL{
		Path: pathThirteen,
	}
	exThirteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlThirteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario fourteen
	pathFourteen := fmt.Sprintf(path, "testing-project-fourteen")
	urlFourteen := &url.URL{
		Path: pathFourteen,
	}
	exFourteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlFourteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario fifteen
	pathFifteen := fmt.Sprintf(path, "testing-project-fifteen")
	urlFifteen := &url.URL{
		Path: pathFifteen,
	}
	exFifteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlFifteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario sixteen
	pathSixteen := fmt.Sprintf(path, "testing-project-sixteen")
	urlSixteen := &url.URL{
		Path: pathSixteen,
	}
	exSixteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlSixteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario seventeen
	pathSeventeen := fmt.Sprintf(path, "testing-project-seventeen")
	urlSeventeen := &url.URL{
		Path: pathSeventeen,
	}
	exSeventeen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlSeventeen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario eighteen
	pathEighteen := fmt.Sprintf(path, "testing-project-eighteen")
	urlEighteen := &url.URL{
		Path: pathEighteen,
	}
	exEighteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlEighteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario nineteen
	pathNineteen := fmt.Sprintf(path, "testing-project-nineteen")
	urlNineteen := &url.URL{
		Path: pathNineteen,
	}
	exNineteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlNineteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario twenty
	pathTwenty := fmt.Sprintf(path, "testing-project-twenty")
	urlTwenty := &url.URL{
		Path: pathTwenty,
	}
	exTwenty := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlTwenty, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)

	// expectations
	expectations := map[string]testhttpapi.HTTPRequestExpectations{
		"compute.googleapis.com" + pathOne:       exOne,
		"compute.googleapis.com" + pathTwo:       exTwo,
		"compute.googleapis.com" + pathThree:     exThree,
		"compute.googleapis.com" + pathFour:      exFour,
		"compute.googleapis.com" + pathFive:      exFive,
		"compute.googleapis.com" + pathSix:       exSix,
		"compute.googleapis.com" + pathSeven:     exSeven,
		"compute.googleapis.com" + pathEight:     exEight,
		"compute.googleapis.com" + pathNine:      exNine,
		"compute.googleapis.com" + pathTen:       exTen,
		"compute.googleapis.com" + pathEleven:    exEleven,
		"compute.googleapis.com" + pathTwelve:    exTwelve,
		"compute.googleapis.com" + pathThirteen:  exThirteen,
		"compute.googleapis.com" + pathFourteen:  exFourteen,
		"compute.googleapis.com" + pathFifteen:   exFifteen,
		"compute.googleapis.com" + pathSixteen:   exSixteen,
		"compute.googleapis.com" + pathSeventeen: exSeventeen,
		"compute.googleapis.com" + pathEighteen:  exEighteen,
		"compute.googleapis.com" + pathNineteen:  exNineteen,
		"compute.googleapis.com" + pathTwenty:    exTwenty,
	}
	exp := testhttpapi.NewExpectationStore(20)
	for k, v := range expectations {
		exp.Put(k, v)
	}
	testhttpapi.StartServer(b, exp)
	provider.DummyAuth = true

	runtimeCtx.ExecutionConcurrencyLimit = 30

	inputBundle, err := stackqltestutil.BuildBenchInputBundle(*runtimeCtx)
	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}

	stringQuery := `
	  select count(1) as inst_count 
	  from google.compute.instances 
	  where 
	    zone = 'australia-southeast1-b' 
		AND /* */ project in 
		  (
			'testing-project', 
			'testing-project-two',
			'testing-project-three',
			'testing-project-four',
			'testing-project-five',
			'testing-project-six',
			'testing-project-seven',
			'testing-project-eight',
			'testing-project-nine',
			'testing-project-ten',
			'testing-project-eleven',
			'testing-project-twelve',
			'testing-project-thirteen',
			'testing-project-fourteen',
			'testing-project-fifteen',
			'testing-project-sixteen',
			'testing-project-seventeen',
			'testing-project-eighteen',
			'testing-project-nineteen',
			'testing-project-twenty'
		  )
	  ;`

	handlerCtx, err := handler.NewHandlerCtx(
		stringQuery, *runtimeCtx, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)),
		inputBundle, "v0.1.1")

	dr, _ := NewStackQLDriver(handlerCtx)

	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}

	dr.ProcessQuery(handlerCtx.GetRawQuery())

	// b.Logf("benchmark parallel select driver integration test passed")
}

//nolint:lll // legacy test
func BenchmarkUnlimitedHighlyParallelProjectSelectGoogleComputeInstanceDriver(b *testing.B) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "BenchmarkUnlimitedHighlyParallelProjectSelectGoogleComputeInstanceDriver")
	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}
	path := `/compute/v1/projects/%s/zones/australia-southeast1-b/instances`
	// scenario one
	pathOne := fmt.Sprintf(path, "testing-project")
	urlOne := &url.URL{
		Path: pathOne,
	}
	exOne := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlOne, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario two
	pathTwo := fmt.Sprintf(path, "testing-project-two")
	urlTwo := &url.URL{
		Path: pathTwo,
	}
	exTwo := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlTwo, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario three
	pathThree := fmt.Sprintf(path, "testing-project-three")
	urlThree := &url.URL{
		Path: pathThree,
	}
	exThree := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlThree, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario four
	pathFour := fmt.Sprintf(path, "testing-project-four")
	urlFour := &url.URL{
		Path: pathFour,
	}
	exFour := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlFour, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario five
	pathFive := fmt.Sprintf(path, "testing-project-five")
	urlFive := &url.URL{
		Path: pathFive,
	}
	exFive := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlFive, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario six
	pathSix := fmt.Sprintf(path, "testing-project-six")
	urlSix := &url.URL{
		Path: pathSix,
	}
	exSix := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlSix, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario seven
	pathSeven := fmt.Sprintf(path, "testing-project-seven")
	urlSeven := &url.URL{
		Path: pathSeven,
	}
	exSeven := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlSeven, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario eight
	pathEight := fmt.Sprintf(path, "testing-project-eight")
	urlEight := &url.URL{
		Path: pathEight,
	}
	exEight := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlEight, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario nine
	pathNine := fmt.Sprintf(path, "testing-project-nine")
	urlNine := &url.URL{
		Path: pathNine,
	}
	exNine := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlNine, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario ten
	pathTen := fmt.Sprintf(path, "testing-project-ten")
	urlTen := &url.URL{
		Path: pathTen,
	}
	exTen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlTen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario eleven
	pathEleven := fmt.Sprintf(path, "testing-project-eleven")
	urlEleven := &url.URL{
		Path: pathEleven,
	}
	exEleven := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlEleven, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario twelve
	pathTwelve := fmt.Sprintf(path, "testing-project-twelve")
	urlTwelve := &url.URL{
		Path: pathTwelve,
	}
	exTwelve := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlTwelve, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario thirteen
	pathThirteen := fmt.Sprintf(path, "testing-project-thirteen")
	urlThirteen := &url.URL{
		Path: pathThirteen,
	}
	exThirteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlThirteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario fourteen
	pathFourteen := fmt.Sprintf(path, "testing-project-fourteen")
	urlFourteen := &url.URL{
		Path: pathFourteen,
	}
	exFourteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlFourteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario fifteen
	pathFifteen := fmt.Sprintf(path, "testing-project-fifteen")
	urlFifteen := &url.URL{
		Path: pathFifteen,
	}
	exFifteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlFifteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario sixteen
	pathSixteen := fmt.Sprintf(path, "testing-project-sixteen")
	urlSixteen := &url.URL{
		Path: pathSixteen,
	}
	exSixteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlSixteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario seventeen
	pathSeventeen := fmt.Sprintf(path, "testing-project-seventeen")
	urlSeventeen := &url.URL{
		Path: pathSeventeen,
	}
	exSeventeen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlSeventeen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario eighteen
	pathEighteen := fmt.Sprintf(path, "testing-project-eighteen")
	urlEighteen := &url.URL{
		Path: pathEighteen,
	}
	exEighteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlEighteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario nineteen
	pathNineteen := fmt.Sprintf(path, "testing-project-nineteen")
	urlNineteen := &url.URL{
		Path: pathNineteen,
	}
	exNineteen := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlNineteen, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	// scenario twenty
	pathTwenty := fmt.Sprintf(path, "testing-project-twenty")
	urlTwenty := &url.URL{
		Path: pathTwenty,
	}
	exTwenty := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", urlTwenty, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)

	// expectations
	expectations := map[string]testhttpapi.HTTPRequestExpectations{
		"compute.googleapis.com" + pathOne:       exOne,
		"compute.googleapis.com" + pathTwo:       exTwo,
		"compute.googleapis.com" + pathThree:     exThree,
		"compute.googleapis.com" + pathFour:      exFour,
		"compute.googleapis.com" + pathFive:      exFive,
		"compute.googleapis.com" + pathSix:       exSix,
		"compute.googleapis.com" + pathSeven:     exSeven,
		"compute.googleapis.com" + pathEight:     exEight,
		"compute.googleapis.com" + pathNine:      exNine,
		"compute.googleapis.com" + pathTen:       exTen,
		"compute.googleapis.com" + pathEleven:    exEleven,
		"compute.googleapis.com" + pathTwelve:    exTwelve,
		"compute.googleapis.com" + pathThirteen:  exThirteen,
		"compute.googleapis.com" + pathFourteen:  exFourteen,
		"compute.googleapis.com" + pathFifteen:   exFifteen,
		"compute.googleapis.com" + pathSixteen:   exSixteen,
		"compute.googleapis.com" + pathSeventeen: exSeventeen,
		"compute.googleapis.com" + pathEighteen:  exEighteen,
		"compute.googleapis.com" + pathNineteen:  exNineteen,
		"compute.googleapis.com" + pathTwenty:    exTwenty,
	}
	exp := testhttpapi.NewExpectationStore(20)
	for k, v := range expectations {
		exp.Put(k, v)
	}
	testhttpapi.StartServer(b, exp)
	provider.DummyAuth = true

	runtimeCtx.ExecutionConcurrencyLimit = -1

	inputBundle, err := stackqltestutil.BuildBenchInputBundle(*runtimeCtx)
	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}

	stringQuery := `
	  select count(1) as inst_count 
	  from google.compute.instances 
	  where 
	    zone = 'australia-southeast1-b' 
		AND /* */ project in 
		  (
			'testing-project', 
			'testing-project-two',
			'testing-project-three',
			'testing-project-four',
			'testing-project-five',
			'testing-project-six',
			'testing-project-seven',
			'testing-project-eight',
			'testing-project-nine',
			'testing-project-ten',
			'testing-project-eleven',
			'testing-project-twelve',
			'testing-project-thirteen',
			'testing-project-fourteen',
			'testing-project-fifteen',
			'testing-project-sixteen',
			'testing-project-seventeen',
			'testing-project-eighteen',
			'testing-project-nineteen',
			'testing-project-twenty'
		  )
	  ;`

	handlerCtx, err := handler.NewHandlerCtx(
		stringQuery, *runtimeCtx, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)),
		inputBundle, "v0.1.1")

	dr, _ := NewStackQLDriver(handlerCtx)

	if err != nil {
		b.Fatalf("Test failed: %v", err)
	}

	dr.ProcessQuery(handlerCtx.GetRawQuery())

	// b.Logf("benchmark parallel select driver integration test passed")
}
