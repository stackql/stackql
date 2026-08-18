//! Renders platforms.json (written by packaging/mcpb/scripts/render-platforms.sh,
//! the one pin source for every wrapper vector) into Rust consts at build time.
//! No hand-written pin table exists in this crate.

use std::env;
use std::fs;
use std::path::Path;

fn main() {
    let manifest_path = Path::new(env!("CARGO_MANIFEST_DIR")).join("platforms.json");
    println!("cargo:rerun-if-changed={}", manifest_path.display());
    let raw = fs::read_to_string(&manifest_path).unwrap_or_else(|e| {
        panic!(
            "cannot read {}: {e}\n  render it first: make cargo-manifest VERSION=X.Y.Z \
             (from packaging/mcpb)",
            manifest_path.display()
        )
    });
    let manifest: serde_json::Value =
        serde_json::from_str(&raw).expect("platforms.json is not valid JSON");
    let version = manifest["version"]
        .as_str()
        .expect("platforms.json: missing 'version'");
    let base_url = manifest["baseUrl"]
        .as_str()
        .expect("platforms.json: missing 'baseUrl'");
    let platforms = manifest["platforms"]
        .as_object()
        .expect("platforms.json: missing 'platforms'");

    let mut out = String::new();
    out.push_str(&format!(
        "/// The stackql release this crate is version-locked to.\n\
         pub const STACKQL_VERSION: &str = {version:?};\n\
         /// Front door the .mcpb bundle is downloaded from (attribution proxy).\n\
         pub const BASE_URL: &str = {base_url:?};\n\
         /// One pin per supported platform, from platforms.json.\n\
         pub const PINS: &[Pin] = &[\n"
    ));
    for (key, entry) in platforms {
        let bundle = entry["bundle"]
            .as_str()
            .unwrap_or_else(|| panic!("platforms.json: {key}: missing 'bundle'"));
        let sha256 = entry["sha256"]
            .as_str()
            .unwrap_or_else(|| panic!("platforms.json: {key}: missing 'sha256'"));
        assert!(
            sha256.len() == 64 && sha256.chars().all(|c| c.is_ascii_hexdigit()),
            "platforms.json: {key}: sha256 is not 64 hex chars"
        );
        out.push_str(&format!(
            "    Pin {{ platform_key: {key:?}, bundle_name: {bundle:?}, sha256: {sha256:?} }},\n"
        ));
    }
    out.push_str("];\n");

    let dest = Path::new(&env::var("OUT_DIR").unwrap()).join("pins_gen.rs");
    fs::write(&dest, out).expect("write pins_gen.rs");
}
