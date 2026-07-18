#!/usr/bin/env python3
"""
Append an externally-produced PKCS#7/CMS signature to a .mcpb bundle in the
MCPB signature framing, and regenerate the bundle's .sha256.

This exists for signing keys that cannot be exported to PEM (hardware tokens,
HSMs): produce a detached DER SignedData over the unsigned bundle bytes with
any external tool (e.g. OpenSSL with a PKCS#11 engine), then frame and append
it here. The result is byte-compatible with 'mcpb sign' output:

  [zip bytes][MCPB_SIG_V1][4-byte LE sig length][DER PKCS#7][MCPB_SIG_END]

Typical flow with a hardware token:

  openssl cms -sign -binary -in dist/stackql-mcp-linux-x64.mcpb \
    -signer cert.pem -certfile chain.pem \
    -keyform engine -engine pkcs11 -inkey "pkcs11:type=private" \
    -outform DER -out sig.der
  python scripts/append-signature.py dist/stackql-mcp-linux-x64.mcpb sig.der

If the bundle already carries a signature block it is replaced (the external
signature must have been produced over the *unsigned* bytes; pass the bundle
through this script only after signing the stripped content - or use
--strip-only first to get the unsigned bytes to sign).

Usage:
  python scripts/append-signature.py <bundle.mcpb> <signature.der>
  python scripts/append-signature.py --strip-only <bundle.mcpb>
"""
from __future__ import annotations

import hashlib
import struct
import sys
from pathlib import Path

HEADER = b"MCPB_SIG_V1"
FOOTER = b"MCPB_SIG_END"


def strip_signature(content: bytes) -> bytes:
    """Return the bundle bytes without any existing signature block."""
    footer_idx = content.rfind(FOOTER)
    if footer_idx == -1:
        return content
    header_idx = content.rfind(HEADER, 0, footer_idx)
    if header_idx == -1:
        return content
    return content[:header_idx]


def write_sha256(bundle: Path) -> None:
    digest = hashlib.sha256(bundle.read_bytes()).hexdigest()
    sha_path = bundle.with_name(bundle.name + ".sha256")
    sha_path.write_text(f"{digest}  {bundle.name}\n")
    print(f"{digest}  {bundle.name}")


def main(argv: list[str]) -> int:
    if len(argv) == 3 and argv[1] == "--strip-only":
        bundle = Path(argv[2])
        stripped = strip_signature(bundle.read_bytes())
        bundle.write_bytes(stripped)
        write_sha256(bundle)
        print(f"stripped signature block (if any) from {bundle.name}")
        return 0

    if len(argv) != 3:
        print(__doc__, file=sys.stderr)
        return 2

    bundle, sig = Path(argv[1]), Path(argv[2])
    if not bundle.exists():
        print(f"error: bundle not found: {bundle}", file=sys.stderr)
        return 1
    if not sig.exists():
        print(f"error: signature not found: {sig}", file=sys.stderr)
        return 1

    sig_der = sig.read_bytes()
    if not sig_der.startswith(b"\x30"):
        print("error: signature does not look like DER (no ASN.1 SEQUENCE)", file=sys.stderr)
        return 1

    content = strip_signature(bundle.read_bytes())
    framed = content + HEADER + struct.pack("<I", len(sig_der)) + sig_der + FOOTER
    bundle.write_bytes(framed)
    print(f"appended {len(sig_der)}-byte signature to {bundle.name}")
    write_sha256(bundle)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
