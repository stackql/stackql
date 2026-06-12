# Makefile - thin wrapper around scripts/package.sh
#
# Usage:
#   make VERSION=X.Y.Z              download release artefacts + build all bundles
#   make download VERSION=X.Y.Z     just fetch the release artefacts into bin/
#   make package VERSION=X.Y.Z      just build bundles from whatever is in bin/
#   make <target> VERSION=X.Y.Z     build a single target
#                                   (linux-x64|linux-arm64|windows-x64|darwin-universal)
#   make one TARGET=<t> VERSION=X.Y.Z  download + build a single target and
#                                      fail hard if its bundle is not produced.
#                                      Use this on a Mac to build only the
#                                      darwin-universal bundle.
#   make signed VERSION=X.Y.Z       build with MCPB_SELF_SIGN=true
#   make publish VERSION=X.Y.Z      upload all dist/*.mcpb (+ .sha256) to the
#                                   stackql/stackql release matching v<VERSION>,
#                                   with --clobber for idempotent re-runs.
#                                   Requires 'gh auth login' with contents:write
#                                   on stackql/stackql.
#   make server-json VERSION=X.Y.Z  render registry/server.json from the template,
#                                   pinning the four per-platform SHA-256s.
#                                   Requires all four dist/*.sha256 files (run
#                                   on a machine that has gathered all bundles).
#   make registry-publish VERSION=X.Y.Z
#                                   publish registry/server.json to the Official
#                                   MCP Registry using mcp-publisher. Requires
#                                   mcp-publisher installed and 'mcp-publisher
#                                   login github'.
#   make list                       show artefacts present in bin/
#   make clean                      wipe dist/
#   make clean-bin                  wipe downloaded artefacts from bin/

SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c

# VERSION defaults to the stackql_release pinned in release.yaml (leading v
# stripped), so plain 'make all' builds the pinned release. Override with
# make <target> VERSION=X.Y.Z as before.
VERSION ?= $(shell sed -n 's/^stackql_release:[[:space:]]*v\{0,1\}//p' release.yaml 2>/dev/null | tr -d ' \r')
BIN_DIR ?= bin
DIST_DIR ?= dist
# Invoked via 'bash' so a lost executable bit (easy when committing from
# Windows) cannot break the build.
PACKAGE := bash scripts/package.sh

RELEASE_BASE := https://github.com/stackql/stackql/releases/download
ASSETS := stackql_linux_amd64.zip \
          stackql_linux_arm64.zip \
          stackql_windows_amd64.zip \
          stackql_darwin_multiarch.pkg

.PHONY: all check-version check-target download package one signed sign publish \
        server-json registry-publish clean clean-bin \
        linux-x64 linux-arm64 windows-x64 darwin-universal \
        list help

all: download package

help:
	@echo "targets:"
	@echo "  make VERSION=X.Y.Z              download release artefacts + build all bundles"
	@echo "  make download VERSION=X.Y.Z     fetch release artefacts into $(BIN_DIR)/"
	@echo "  make package VERSION=X.Y.Z      build bundles from whatever is in $(BIN_DIR)/"
	@echo "  make <target> VERSION=X.Y.Z     build a single target"
	@echo "                                  (linux-x64|linux-arm64|windows-x64|darwin-universal)"
	@echo "  make one TARGET=<t> VERSION=X.Y.Z  download + build a single target,"
	@echo "                                  fail hard if the bundle is not produced"
	@echo "                                  (use on a Mac for darwin-universal)"
	@echo "  make signed VERSION=X.Y.Z       build with MCPB_SELF_SIGN=true"
	@echo "  make sign                       envelope-sign dist/*.mcpb in place and"
	@echo "                                  regenerate .sha256 (MCPB_SELF_SIGN=true"
	@echo "                                  or MCPB_SIGN_CERT + MCPB_SIGN_KEY)"
	@echo "  make publish VERSION=X.Y.Z      upload dist/* to the stackql/stackql"
	@echo "                                  release matching v<VERSION>"
	@echo "  make server-json VERSION=X.Y.Z  render registry/server.json (pins"
	@echo "                                  SHAs from dist/*.sha256)"
	@echo "  make registry-publish VERSION=X.Y.Z"
	@echo "                                  publish to the Official MCP Registry"
	@echo "                                  via mcp-publisher (renders first)"
	@echo "  make list                       show artefacts present in $(BIN_DIR)/"
	@echo "  make clean                      wipe $(DIST_DIR)/"
	@echo "  make clean-bin                  wipe downloaded artefacts from $(BIN_DIR)/"

check-version:
	@if [ -z "$(VERSION)" ]; then \
	  echo "error: VERSION is required (e.g. make VERSION=0.10.500)," >&2; \
	  echo "       or set stackql_release in release.yaml" >&2; exit 2; \
	fi

download: check-version
	@command -v curl >/dev/null 2>&1 || { echo "error: curl is required for 'make download'" >&2; exit 2; }
	@mkdir -p $(BIN_DIR)
	@echo "downloading stackql v$(VERSION) release artefacts -> $(BIN_DIR)/"
	@for asset in $(ASSETS); do \
	  url="$(RELEASE_BASE)/v$(VERSION)/$$asset"; \
	  dest="$(BIN_DIR)/$$asset"; \
	  echo "  $$asset"; \
	  curl -fsSL --retry 3 -o "$$dest" "$$url" || { \
	    echo "  error: failed to download $$url" >&2; \
	    rm -f "$$dest"; \
	    exit 1; \
	  }; \
	done
	@echo "done."

check-target:
	@case "$(TARGET)" in \
	  linux-x64|linux-arm64|windows-x64|darwin-universal) ;; \
	  "" ) echo "error: TARGET is required (linux-x64|linux-arm64|windows-x64|darwin-universal)" >&2; exit 2 ;; \
	  *  ) echo "error: unknown TARGET '$(TARGET)' (linux-x64|linux-arm64|windows-x64|darwin-universal)" >&2; exit 2 ;; \
	esac

# Download just one platform's artefact and build only that bundle. Designed
# for the two-machine release flow: run 'make all' on your laptop for the
# Linux/Windows bundles, then 'make one TARGET=darwin-universal' on a Mac
# (e.g. MacInCloud) for the notarised .pkg slice.
one: check-version check-target
	@command -v curl >/dev/null 2>&1 || { echo "error: curl required" >&2; exit 2; }
	@mkdir -p $(BIN_DIR)
	@case "$(TARGET)" in \
	  linux-x64)        asset=stackql_linux_amd64.zip ;; \
	  linux-arm64)      asset=stackql_linux_arm64.zip ;; \
	  windows-x64)      asset=stackql_windows_amd64.zip ;; \
	  darwin-universal) asset=stackql_darwin_multiarch.pkg ;; \
	esac; \
	url="$(RELEASE_BASE)/v$(VERSION)/$$asset"; \
	dest="$(BIN_DIR)/$$asset"; \
	echo "downloading $$asset for target $(TARGET)"; \
	curl -fsSL --retry 3 -o "$$dest" "$$url" || { echo "  error: download failed: $$url" >&2; rm -f "$$dest"; exit 1; }
	$(PACKAGE) --version $(VERSION)
	@out="$(DIST_DIR)/stackql-mcp-$(TARGET).mcpb"; \
	if [ ! -f "$$out" ]; then \
	  echo "error: expected bundle not produced: $$out" >&2; exit 1; \
	fi; \
	echo "produced $$out"

package: check-version
	$(PACKAGE) --version $(VERSION)

signed: check-version
	MCPB_SELF_SIGN=true $(PACKAGE) --version $(VERSION)

# Envelope-sign whatever is already in dist/ and regenerate the .sha256
# files (the signature is appended to the bundle, so checksums must be
# recomputed). Same env contract as package.sh: MCPB_SELF_SIGN=true, or
# MCPB_SIGN_CERT + MCPB_SIGN_KEY (+ optional MCPB_SIGN_INTERMEDIATES).
# No-ops with a notice when no signing material is configured, so CI can
# call it unconditionally before 'make publish'.
sign:
	bash scripts/sign.sh

# Render registry/server.json from the template, pinning the four per-platform
# SHA-256s read from dist/*.sha256. Fails hard if any sha file is missing -
# run this on a machine that has gathered all four bundles (typically the
# workstation, after the Mac slice has been published and downloaded back, or
# after copying the darwin sha file across).
server-json: check-version
	bash scripts/render-server-json.sh --version $(VERSION)

# Publish the rendered server.json to the Official MCP Registry.
# Requires:
#   - mcp-publisher CLI on PATH (https://github.com/modelcontextprotocol/registry/releases)
#   - 'mcp-publisher login github' completed once (browser flow)
#   - GitHub user authorised on the 'stackql' org (for io.github.stackql/* namespace)
registry-publish: check-version server-json
	@command -v mcp-publisher >/dev/null 2>&1 || { \
	  echo "error: mcp-publisher not on PATH" >&2; \
	  echo "  install from https://github.com/modelcontextprotocol/registry/releases/latest" >&2; \
	  exit 2; \
	}
	cd registry && mcp-publisher publish

# Upload everything that landed in dist/ to the matching tag on stackql/stackql.
# Idempotent via --clobber, so running this from two machines (one with the
# darwin bundle, one with the rest) is safe in either order.
publish: check-version
	@command -v gh >/dev/null 2>&1 || { echo "error: gh CLI required (https://cli.github.com)" >&2; exit 2; }
	@shopt -s nullglob; \
	  files=( $(DIST_DIR)/stackql-mcp-*.mcpb $(DIST_DIR)/stackql-mcp-*.mcpb.sha256 ); \
	  if [ $${#files[@]} -eq 0 ]; then \
	    echo "error: nothing in $(DIST_DIR)/ to publish" >&2; exit 1; \
	  fi; \
	  echo "publishing $${#files[@]} file(s) to stackql/stackql release v$(VERSION):"; \
	  for f in "$${files[@]}"; do echo "  $$f"; done; \
	  gh release upload "v$(VERSION)" "$${files[@]}" --clobber --repo stackql/stackql

# Single-target builds: temporarily hide the other artefacts so package.sh
# skips them. Uses a sentinel dir under bin/ so nothing leaves the tree.
# Restores the moved artefacts even if package.sh fails.
define build_one
	@mkdir -p $(BIN_DIR)/.hidden
	@find $(BIN_DIR) -maxdepth 1 -mindepth 1 \
	  ! -name '.hidden' ! -name '.gitignore' ! -name 'README.md' \
	  $(1) -exec mv {} $(BIN_DIR)/.hidden/ \;
	-$(PACKAGE) --version $(VERSION)
	@mv $(BIN_DIR)/.hidden/* $(BIN_DIR)/ 2>/dev/null || true
	@rmdir $(BIN_DIR)/.hidden 2>/dev/null || true
endef

linux-x64: check-version
	$(call build_one, ! -name 'stackql_linux_amd64.zip')

linux-arm64: check-version
	$(call build_one, ! -name 'stackql_linux_arm64.zip')

windows-x64: check-version
	$(call build_one, ! -name 'stackql_windows_amd64.zip')

darwin-universal: check-version
	$(call build_one, ! -name 'stackql_darwin*.pkg')

list:
	@ls -1 $(BIN_DIR) 2>/dev/null | grep -v -E '^(\.gitignore|README\.md)$$' || echo "(empty)"

clean:
	bash scripts/clean.sh

clean-bin:
	@rm -f $(addprefix $(BIN_DIR)/,$(ASSETS))
	@echo "cleaned downloaded artefacts from $(BIN_DIR)/"
