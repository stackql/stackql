package embed

// DefaultVersion is the stackql release this module version was developed
// and conformance-tested against. cmd/stackql-mcp-fetch defaults to it.
const DefaultVersion = "0.10.500"

// BundlePins maps platform key to the published sha256 of the release
// .mcpb bundle for DefaultVersion, copied from the release's .sha256
// assets. The fetch tool verifies downloads against these before
// extracting the binary; for other versions it fetches the pin from the
// release at download time.
var BundlePins = map[string]string{
	PlatformLinuxX64:        "6615737747156b1a8413a976afb23af2e7eec29ebc98a6f0a0f65d1b153c44be",
	PlatformLinuxARM64:      "594bedbabc3096dc3563c907724e845ce0b61a67de4b3fed4158b40c0363786c",
	PlatformWindowsX64:      "d2ce895e88f9c6b557df07073158629808f56d75598f3a701164d65506b791b0",
	PlatformDarwinUniversal: "4eed70af5cfa67295ae0b42fa3a6dca71ac9acabd0d67914fd96ad1247a9b4cc",
}
