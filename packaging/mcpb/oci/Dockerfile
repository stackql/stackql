# syntax=docker/dockerfile:1
#
# StackQL MCP server image. The binary is extracted from the signed release
# zips by 'make oci-stage' into .oci-stage/linux/<arch>/ before building -
# this Dockerfile only copies, it does not download.
#
#   make oci VERSION=X.Y.Z        local single-arch build (amd64) for testing
#   make oci-push VERSION=X.Y.Z   multi-arch build + push (requires docker login)
#
# Run as an MCP stdio server:
#   docker run -i --rm stackql/stackql-mcp:X.Y.Z

FROM debian:bookworm-slim

ARG TARGETARCH
ARG VERSION=dev

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && useradd --create-home --uid 10001 stackql

# --chmod: the staged file's mode depends on the host that ran unzip (Windows
# stages it non-executable), so set it explicitly.
COPY --chmod=0755 .oci-stage/linux/${TARGETARCH}/stackql /usr/local/bin/stackql

# io.modelcontextprotocol.server.name is required by the Official MCP Registry
# to validate namespace ownership of oci packages.
LABEL io.modelcontextprotocol.server.name="io.github.stackql/stackql-mcp" \
      org.opencontainers.image.title="StackQL MCP Server" \
      org.opencontainers.image.description="SQL-native query and provisioning engine for cloud infrastructure, served over MCP." \
      org.opencontainers.image.url="https://stackql.io" \
      org.opencontainers.image.source="https://github.com/stackql/stackql" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.licenses="MIT"

USER stackql
WORKDIR /home/stackql

ENTRYPOINT ["/usr/local/bin/stackql"]
# approot and audit are pinned to writable locations even though the container
# home is writable - same cwd-independence rules as the .mcpb manifest.
CMD ["mcp", "--mcp.server.type=stdio", "--approot", "/home/stackql/.stackql", "--mcp.config", "{\"server\": {\"audit\": {\"disabled\": true}}}"]
