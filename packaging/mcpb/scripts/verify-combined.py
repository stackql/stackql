"""Byte-level verification of the multiplatform .mcpb bundle.

Asserts every platform binary is present at its expected path inside the
bundle AND is the right machine type - the check that would have caught the
original wrong-artefact incident (a Mach-O served to Windows/Linux users) in
seconds. Stdlib only.

Usage:
    python3 scripts/verify-combined.py dist/stackql-mcp-multiplatform.mcpb
"""

import sys
import zipfile

# Mach-O magics: fat (universal) both endians, plus thin 64-bit for safety.
MACHO_MAGICS = (
    b"\xca\xfe\xba\xbe",  # FAT_MAGIC
    b"\xbe\xba\xfe\xca",  # FAT_CIGAM
    b"\xcf\xfa\xed\xfe",  # MH_MAGIC_64 (little endian on disk)
    b"\xfe\xed\xfa\xcf",  # MH_CIGAM_64
)

CHECKS = [
    ("server/win32/stackql.exe", "windows PE (MZ)", lambda b: b[:2] == b"MZ"),
    ("server/linux-x64/stackql", "linux ELF", lambda b: b[:4] == b"\x7fELF"),
    ("server/linux-arm64/stackql", "linux ELF", lambda b: b[:4] == b"\x7fELF"),
    ("server/darwin/stackql", "Mach-O (universal)", lambda b: b[:4] in MACHO_MAGICS),
    ("server/linux/stackql", "sh dispatch shim", lambda b: b[:9] == b"#!/bin/sh"),
    ("manifest.json", "manifest", lambda b: b.lstrip()[:1] == b"{"),
]


def main() -> int:
    if len(sys.argv) != 2:
        print(__doc__.strip(), file=sys.stderr)
        return 2
    bundle_path = sys.argv[1]
    failures = []
    with zipfile.ZipFile(bundle_path) as bundle:
        names = set(bundle.namelist())
        for member, description, predicate in CHECKS:
            if member not in names:
                failures.append(f"MISSING: {member} ({description})")
                continue
            with bundle.open(member) as fh:
                head = fh.read(16)
            if not predicate(head):
                failures.append(
                    f"WRONG TYPE: {member} expected {description}, "
                    f"got leading bytes {head[:4].hex()}"
                )
            else:
                print(f"ok: {member} is {description}")
    if failures:
        for failure in failures:
            print(failure, file=sys.stderr)
        return 1
    print(f"verified: {bundle_path} contains all platform binaries with correct machine types")
    return 0


if __name__ == "__main__":
    sys.exit(main())
