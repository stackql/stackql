package registry

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-provider-registry/signing/pqsig"
	"gopkg.in/yaml.v3"
)

const (
	defaultRegistryUrlString = "https://raw.githubusercontent.com/stackql/" +
		"stackql-provider-registry/intial-devel/providers"
	defaultSrcPrefix       string = "src"
	defaultDistPrefix      string = "dist"
	httpSchemeRegexpString string = `(?i)^https?$`
	fileSchemeRegexpString string = `(?i)^file$`
	remoteProviderListPath string = `providers.yaml`
)

var (
	httpSchemeRegexp *regexp.Regexp = regexp.MustCompile(httpSchemeRegexpString)
	fileSchemeRegexp *regexp.Regexp = regexp.MustCompile(fileSchemeRegexpString)
)

type RegistryAPI interface {
	PullAndPersistProviderArchive(string, string) error
	ListAllAvailableProviders() (map[string]ProviderDescription, error)
	ListAllProviderVersions(string) (map[string]ProviderDescription, error)
	ListLocallyAvailableProviders() map[string]ProviderDescription
	GetDocBytes(string) ([]byte, error)
	GetLatestAvailableVersion(string) (string, error)
	GetLatestPublishedVersion(string) (string, error)
	LoadProviderByName(string, string) (ProviderDocument, error)
	RemoveProviderVersion(string, string) error
	ClearProviderCache(string) error
	GetLocalProviderDocPath(string, string) string
}

type RegistryConfig = omnisdk.RegistryConfig

type standardRegistry struct {
	allowSrcDownload bool
	regUrl           *url.URL
	srcUrl           *url.URL
	distUrl          *url.URL
	localDocRoot     string
	localSrcPrefix   string
	localDistPrefix  string
	transport        http.RoundTripper
	verifier         pqsig.Verifier
	nopVerifier      bool
}

func NewRegistry(registryCfg RegistryConfig, transport http.RoundTripper) (RegistryAPI, error) {
	return newRegistry(registryCfg, transport)
}

func newRegistry(registryCfg RegistryConfig, transport http.RoundTripper) (RegistryAPI, error) {
	registryUrl := registryCfg.RegistryURL
	if registryUrl == "" {
		registryUrl = defaultRegistryUrlString
	}
	srcUrlStr := registryUrl
	var srcPrefix string
	if registryCfg.SrcPrefix == nil {
		srcPrefix = defaultSrcPrefix
	} else {
		srcPrefix = *registryCfg.SrcPrefix
	}
	if srcPrefix != "" {
		srcUrlStr = fmt.Sprintf("%s/%s", registryUrl, srcPrefix)
	}
	distUrlStr := registryUrl
	var distPrefix string
	if registryCfg.DistPrefix == nil {
		distPrefix = defaultDistPrefix
	} else {
		distPrefix = *registryCfg.DistPrefix
	}
	if distPrefix != "" {
		distUrlStr = fmt.Sprintf("%s/%s", registryUrl, distPrefix)
	}
	regUrl, err := url.Parse(registryUrl)
	if err != nil {
		return nil, err
	}
	srcUrl, err := url.Parse(srcUrlStr)
	if err != nil {
		return nil, err
	}
	distUrl, err := url.Parse(distUrlStr)
	if err != nil {
		return nil, err
	}
	var ver pqsig.Verifier
	nopVerify := false
	var keys map[string]string
	if registryCfg.VerifyConfig != nil {
		nopVerify = registryCfg.VerifyConfig.NopVerify
		keys = registryCfg.VerifyConfig.PublicKeys
	}
	if !nopVerify {
		ver, err = pqsig.NewVerifier(keys)
		if err != nil {
			return nil, err
		}
	}
	rv := &standardRegistry{
		allowSrcDownload: registryCfg.AllowSrcDownload,
		regUrl:           regUrl,
		srcUrl:           srcUrl,
		distUrl:          distUrl,
		localDocRoot:     registryCfg.LocalDocRoot,
		localSrcPrefix:   srcPrefix,
		localDistPrefix:  distPrefix,
		transport:        transport,
		verifier:         ver,
		nopVerifier:      nopVerify,
	}
	return rv, nil
}

func (r *standardRegistry) GetLocalDocRoot() string {
	return r.getLocalDocRoot()
}

func (r *standardRegistry) ListLocallyAvailableProviders() map[string]ProviderDescription {
	return r.listLocalProviders()
}

type ProviderDescription = omnisdk.ProviderDescription

type ProvidersList struct {
	_         struct{}
	Providers map[string]ProviderDescription `json:"providers" yaml:"providers"`
}

func NewProvidersList() ProvidersList {
	return ProvidersList{
		Providers: make(map[string]ProviderDescription),
	}
}

func (pl ProvidersList) GetLatestList() (ProvidersList, error) {
	m := make(map[string]ProviderDescription)
	for k, v := range pl.Providers {
		latest, err := latestVersion(v.Versions)
		if err != nil {
			return NewProvidersList(), err
		}
		m[k] = ProviderDescription{Versions: []string{latest}}
	}
	return ProvidersList{Providers: m}, nil
}

func (pl ProvidersList) GetSingleProviderList(prov string) ProvidersList {
	m := make(map[string]ProviderDescription)
	for k, v := range pl.Providers {
		if k == prov {
			m[k] = v
		}
	}
	return ProvidersList{Providers: m}
}

func (r *standardRegistry) ListAllAvailableProviders() (map[string]ProviderDescription, error) {
	if r.isFile() {
		return nil, fmt.Errorf("'registry list' is meaningless in local mode")
	}
	regProvs := NewProvidersList()
	rc, err := r.getRemoteProviderList()
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(b, &regProvs)
	if err != nil {
		return nil, err
	}
	latest, err := regProvs.GetLatestList()
	if err != nil {
		return nil, err
	}
	return latest.Providers, nil
}

func (r *standardRegistry) ListAllProviderVersions(prov string) (map[string]ProviderDescription, error) {
	return r.listAllProviderVersions(prov)
}

func (r *standardRegistry) listAllProviderVersions(prov string) (map[string]ProviderDescription, error) {
	if r.isFile() {
		return nil, fmt.Errorf("'registry list' is meaningless in local mode")
	}
	regProvs := NewProvidersList()
	rc, err := r.getRemoteProviderList()
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(b, &regProvs)
	if err != nil {
		return nil, err
	}
	singleProvList := regProvs.GetSingleProviderList(prov)
	return singleProvList.Providers, nil
}

func (r *standardRegistry) isHttp() bool {
	return httpSchemeRegexp.MatchString(r.regUrl.Scheme)
}

func (r *standardRegistry) isFile() bool {
	return fileSchemeRegexp.MatchString(r.regUrl.Scheme)
}

func (r *standardRegistry) isLocalFile() bool {
	return r.isFile() && strings.HasPrefix(r.regUrl.Path, "/")
}

func (r *standardRegistry) GetDocBytes(docPath string) ([]byte, error) {
	return r.getVerifiedDocBytes(docPath)
}

func (r *standardRegistry) getProviderDocBytes(prov string, version string) ([]byte, error) {
	prov = r.renameProvider(prov)
	return r.getVerifiedDocBytes(path.Join(prov, version, "provider.yaml"))
}

func (r *standardRegistry) PullProviderArchive(prov string, version string) (io.ReadCloser, error) {
	return r.pullProviderArchive(prov, version)
}

func (r *standardRegistry) pullProviderArchive(prov string, version string) (io.ReadCloser, error) {
	prov = r.renameProvider(prov)
	fp := path.Join(prov, fmt.Sprintf("%s.tgz", version))
	return r.pullArchive(fp)
}

func (r *standardRegistry) PullAndPersistProviderArchive(prov string, version string) error {
	return r.pullAndPersistProviderArchive(prov, version)
}

func (r *standardRegistry) pullAndPersistProviderArchive(prov string, version string) error {
	if r.localDocRoot == "" {
		return fmt.Errorf("cannot pull provider without local doc location")
	}
	rdr, err := r.pullProviderArchive(prov, version)
	if err != nil {
		return err
	}
	pr := r.renameProvider(prov)
	err = os.RemoveAll(path.Join(r.getLocalDocRoot(), pr, version))
	if err != nil {
		return err
	}
	return decompressToPath(rdr, path.Join(r.getLocalDocRoot(), pr))
}

func (r *standardRegistry) LoadProviderByName(prov, version string) (ProviderDocument, error) {
	if version == "" {
		var err error
		version, err = r.GetLatestAvailableVersion(prov)
		if err != nil {
			return nil, err
		}
	}
	b, err := r.getProviderDocBytes(prov, version)
	if err != nil {
		return nil, err
	}
	return parseProviderDocument(b)
}

func (r *standardRegistry) pullArchive(archivePath string) (io.ReadCloser, error) {
	return r.getUnVerifiedArchive(archivePath)
}

func (r *standardRegistry) getRemoteDoc(docPath string) (io.ReadCloser, error) {
	cl := &http.Client{}
	if r.transport != nil {
		cl.Transport = r.transport
	}
	response, err := cl.Get(fmt.Sprintf("%s/%s", r.srcUrl.String(), docPath))
	if err != nil {
		return nil, err
	}
	if response.Body == nil {
		return nil, fmt.Errorf("no response body from remote")
	}
	return response.Body, nil
}

func (r *standardRegistry) getRemoteArchive(docPath string) (io.ReadCloser, error) {
	cl := &http.Client{}
	if r.transport != nil {
		cl.Transport = r.transport
	}
	response, err := cl.Get(fmt.Sprintf("%s/%s", r.distUrl.String(), docPath))
	if err != nil {
		return nil, err
	}
	if response.Body == nil {
		return nil, fmt.Errorf("no response body from remote")
	}
	return response.Body, nil
}

func (r *standardRegistry) getRemoteProviderList() (io.ReadCloser, error) {
	cl := &http.Client{}
	if r.transport != nil {
		cl.Transport = r.transport
	}
	response, err := cl.Get(fmt.Sprintf("%s/%s", r.distUrl.String(), remoteProviderListPath))
	if err != nil {
		return nil, err
	}
	if response.Body == nil {
		return nil, fmt.Errorf("no response body from remote")
	}
	return response.Body, nil
}

func (r *standardRegistry) GetLocalDocTrunk() string {
	return r.getLocalDocTrunk()
}

func (r *standardRegistry) getLocalDocTrunk() string {
	return r.localDocRoot
}

func (r *standardRegistry) getLocalDocRoot() string {
	switch r.localSrcPrefix {
	case "":
		return r.localDocRoot
	default:
		return path.Join(r.localDocRoot, r.localSrcPrefix)
	}
}

type ProviderInfo struct {
	Name    string
	Version string
}

//nolint:gocognit // Preserve the registry directory traversal and filtering.
func (r *standardRegistry) listLocalProviders() map[string]ProviderDescription {
	dr := r.getLocalDocRoot()
	switch dr {
	case "":
		return map[string]ProviderDescription{}
	default:
		provs, err := os.ReadDir(dr)
		if err != nil {
			return map[string]ProviderDescription{}
		}
		rv := make(map[string]ProviderDescription)
		for _, p := range provs {
			if p.IsDir() && !strings.HasPrefix(p.Name(), ".") {
				versions, versionErr := os.ReadDir(path.Join(dr, p.Name()))
				var val ProviderDescription
				if versionErr == nil {
					for _, v := range versions {
						if v.IsDir() && !strings.HasPrefix(v.Name(), ".") {
							val.Versions = append(val.Versions, v.Name())
						}
					}
				}
				rv[p.Name()] = val
			}
		}
		return rv
	}
}

func (r *standardRegistry) getLocalArchiveRoot() string {
	switch r.localDistPrefix {
	case "":
		return r.localDocRoot
	default:
		return path.Join(r.localDocRoot, r.localDistPrefix)
	}
}

func (r *standardRegistry) renameProvider(prov string) string {
	if prov == "google" {
		prov = "googleapis.com"
	}
	return prov
}

func (r *standardRegistry) GetLocalProviderDocPath(prov string, version string) string {
	return r.getLocalProviderDocPath(prov, version)
}

func (r *standardRegistry) getLocalProviderDocPath(prov string, version string) string {
	return path.Join(r.getLocalProviderRootDirPath(prov, version), "provider.yaml")
}

func (r *standardRegistry) getLocalProviderRootDirPath(prov string, version string) string {
	prov = r.renameProvider(prov)
	return path.Join(r.getLocalDocRoot(), prov, version)
}

func (r *standardRegistry) getLocalDocPath(docPath string) string {
	return path.Join(r.getLocalDocRoot(), docPath)
}

func (r *standardRegistry) getLocalArchivePath(docPath string) string {
	return path.Join(r.getLocalArchiveRoot(), docPath)
}

func (r *standardRegistry) getLocalDoc(docPath string) (io.ReadCloser, error) {
	// localPath := r.getLocalDocPath(docPath)
	fi, err := os.Open(docPath)
	if fi != nil {
		defer fi.Close()
	} else {
		return nil, fmt.Errorf("nil file")
	}
	if err != nil {
		return nil, err
	}
	fb, readErr := io.ReadAll(fi)
	if readErr != nil {
		return nil, fmt.Errorf("cannot read local file: '%s'", readErr.Error())
	}
	return io.NopCloser(bytes.NewReader(fb)), nil
}

func (r *standardRegistry) getUnVerifiedArchive(docPath string) (io.ReadCloser, error) {
	if r.isLocalFile() {
		return os.Open(path.Join(r.distUrl.Path, docPath))
	}
	if r.localDocRoot != "" {
		localPath := r.getLocalArchivePath(docPath)
		lf, err := r.getLocalDoc(localPath)
		if err == nil {
			return lf, nil
		}
	}
	if r.isHttp() {
		return r.getRemoteArchive(docPath)
	}
	return nil, fmt.Errorf("registry scheme '%s' currently not supported", r.regUrl.Scheme)
}

func (r *standardRegistry) getDocReader(docPath string) (io.ReadCloser, error) {
	if r.isLocalFile() {
		return os.Open(path.Join(r.localSourcePath(), docPath))
	}
	if r.localDocRoot != "" {
		localPath := r.getLocalDocPath(docPath)
		lf, err := r.getLocalDoc(localPath)
		if err == nil {
			return lf, nil
		}
	}
	if r.isHttp() {
		if !r.allowSrcDownload {
			return nil, fmt.Errorf("download of individual docs disallowed; please attempt to pull provider docs")
		}
		response, err := r.getRemoteDoc(docPath)
		if err != nil {
			return nil, err
		}
		if response == nil {
			return nil, fmt.Errorf("no response body from remote")
		}
		return response, nil
	}
	return nil, fmt.Errorf("registry scheme '%s' currently not supported", r.regUrl.Scheme)
}

func (r *standardRegistry) getVerifiedDocBytes(docPath string) ([]byte, error) {
	reader, err := r.getDocReader(docPath)
	if err != nil {
		return nil, fmt.Errorf("registry document %q: %w", docPath, err)
	}
	defer reader.Close()
	doc, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read registry document %q: %w", docPath, err)
	}
	if r.nopVerifier {
		return doc, nil
	}
	doc, err = r.verifier.Verify(doc)
	if err != nil {
		return nil, fmt.Errorf("verify registry document %q: %w", docPath, err)
	}
	return doc, nil
}

func (r *standardRegistry) GetLatestAvailableVersion(providerName string) (string, error) {
	return r.getLatestAvailableVersion(providerName)
}

func (r *standardRegistry) localSourcePath() string {
	localPath := r.srcUrl.Path
	if r.srcUrl.Host == "." {
		localPath = "." + localPath
	}
	if strings.HasPrefix(localPath, "/") && filepath.VolumeName(localPath[1:]) != "" {
		localPath = localPath[1:]
	}
	return localPath
}

func (r *standardRegistry) getLatestAvailableVersion(providerName string) (string, error) {
	providerName = r.renameProvider(providerName)
	if r.isLocalFile() {
		deSlice, err := os.ReadDir(path.Join(r.localSourcePath(), providerName))
		if err != nil {
			return "", err
		}
		var deStrSlice []string
		for _, e := range deSlice {
			if e.IsDir() {
				deStrSlice = append(deStrSlice, e.Name())
			}
		}
		return latestVersion(deStrSlice)
	}

	deSlice, err := os.ReadDir(path.Join(r.getLocalDocRoot(), providerName))
	if err != nil {
		return "", err
	}
	var deStrSlice []string
	for _, e := range deSlice {
		if e.IsDir() {
			deStrSlice = append(deStrSlice, e.Name())
		}
	}
	return latestVersion(deStrSlice)
}

func (r *standardRegistry) GetLatestPublishedVersion(providerName string) (string, error) {
	return r.getLatestPublishedVersion(providerName)
}

func (r *standardRegistry) getLatestPublishedVersion(providerName string) (string, error) {
	p, err := r.listAllProviderVersions(providerName)
	if err != nil {
		return "", err
	}
	description, ok := p[providerName]
	if !ok {
		return "", fmt.Errorf("could not resolve provider '%s'", providerName)
	}
	latestVersion, err := latestVersion(description.Versions)
	if err != nil {
		return "", err
	}
	return latestVersion, nil
}

// RemoveProviderVersion removes a specific version of a provider
// For example: RemoveProviderVersion("okta", "v23.09.00169").
func (r *standardRegistry) RemoveProviderVersion(providerId string, version string) error {
	providerPath := r.getLocalProviderRootDirPath(providerId, version)
	return os.RemoveAll(providerPath)
}

// ClearProviderCache clears the cache for a specific provider
// For example: ClearProviderCache("aws").
func (r *standardRegistry) ClearProviderCache(providerId string) error {
	cachePath := path.Join(r.getLocalDocRoot(), providerId)
	return os.RemoveAll(cachePath)
}
