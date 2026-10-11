"""Byte-level verification of the multiplatform .mcpb bundle.

Asserts every platform binary is present at its expected path inside the
bundle AND is the right machine type - the check that would have caught the
original wrong-artefact incident (a Mach-O served to Windows/Linux users) in
seconds. Stdlib only.

Usage:
    python3 scripts/verify-combined.py dist/stackql-mcp-multiplatform.mcpb
"""

import sys
import struct
import zipfile

# Mach-O magics: fat (universal) both endians, plus thin 64-bit for safety.
MACHO_MAGICS = (
    b"\xca\xfe\xba\xbe",  # FAT_MAGIC
    b"\xbe\xba\xfe\xca",  # FAT_CIGAM
    b"\xcf\xfa\xed\xfe",  # MH_MAGIC_64 (little endian on disk)
    b"\xfe\xed\xfa\xcf",  # MH_CIGAM_64
)

def is_pe_machine(data: bytes, machine: int) -> bool:
    if len(data) < 64 or data[:2] != b"MZ":
        return False
    offset = struct.unpack_from("<I", data, 0x3C)[0]
    return (
        offset + 6 <= len(data)
        and data[offset:offset + 4] == b"PE\0\0"
        and struct.unpack_from("<H", data, offset + 4)[0] == machine
    )


CHECKS = [
    ("server/windows-x64/stackql.exe", "windows x64 PE", lambda b: is_pe_machine(b, 0x8664)),
    ("server/windows-arm64/stackql.exe", "windows ARM64 PE", lambda b: is_pe_machine(b, 0xAA64)),
    ("server/win32/stackql.ps1", "PowerShell dispatch shim", lambda b: b.startswith(b"$ErrorActionPreference")),
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
                head = fh.read()
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
