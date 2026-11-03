"""Choose the build suffix once for native binaries and Docker tags."""
import argparse
import os
from pathlib import Path


def prerelease_suffix(branch: str, live_branch: str, ref_type: str = "branch") -> str:
    if ref_type != "branch":
        return "-beta"
    if branch == live_branch or branch.startswith("stable/"):
        return ""
    return "-rc" if branch == "main" else "-alpha"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("branch")
    parser.add_argument("live_branch")
    parser.add_argument("--ref-type", choices=("branch", "tag"), default=os.environ.get("GITHUB_REF_TYPE", "branch"))
    args = parser.parse_args()
    suffix = prerelease_suffix(args.branch, args.live_branch, args.ref_type)
    if os.environ.get("GITHUB_ENV"):
        with Path(os.environ["GITHUB_ENV"]).open("a", encoding="utf-8") as output:
            output.write(f"BUILDPRERELEASESUFFIX={suffix}\n")
    print(suffix)


if __name__ == "__main__":
    main()
