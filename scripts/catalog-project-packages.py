#!/usr/bin/env python3
"""Build and verify the self-contained project package catalog."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
CATALOG = ROOT / "hostlier-packages.json"
TARGETS = [(os_name, arch) for os_name in ("darwin", "linux") for arch in ("amd64", "arm64")]


def build_package(name, source, command, stage):
    """Build four native executables before changing any published catalog paths."""
    artifacts = []
    for os_name, arch in TARGETS:
        path = Path("binaries") / f"hostlier-{name}-{os_name}-{arch}"
        output = stage / path
        output.parent.mkdir(exist_ok=True)
        subprocess.run(
            ["go", "build", "-mod=readonly", "-buildvcs=false", "-trimpath", "-ldflags=-s -w",
             "-o", str(output), command], cwd=source, check=True,
            env={**os.environ, "CGO_ENABLED": "0", "GOOS": os_name, "GOARCH": arch},
        )
        artifacts.append({"os": os_name, "arch": arch, "path": str(path),
                          "sha256": hashlib.sha256(output.read_bytes()).hexdigest()})
    return {"name": name, "artifacts": artifacts}


def verify(packages):
    """Refuse to release bytes that differ from the Git install contract."""
    for name in ("minion", "harnest", "sandbox-browser", "sandbox-computer"):
        entries = packages.get(name, {}).get("artifacts", [])
        if sorted((e["os"], e["arch"]) for e in entries) != sorted(TARGETS):
            raise SystemExit(f"incomplete platform catalog for {name}")
        for entry in entries:
            path = f'binaries/hostlier-{name}-{entry["os"]}-{entry["arch"]}'
            if entry.get("path") != path or hashlib.sha256((ROOT / path).read_bytes()).hexdigest() != entry["sha256"]:
                raise SystemExit(f"invalid catalog artifact: {path}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--hostlier", type=Path, help="Hostlier source checkout containing Minion")
    parser.add_argument("--harnest", type=Path, help="Harnest source checkout")
    parser.add_argument("--sandboxes", action="store_true", help="refresh both sandbox packages from this checkout")
    parser.add_argument("--verify", action="store_true", help="verify the checked-in catalog without rebuilding")
    args = parser.parse_args()
    catalog = json.loads(CATALOG.read_text())
    packages = {p["name"]: p for p in catalog["packages"]}
    if args.verify:
        verify(packages)
        return
    if bool(args.hostlier) != bool(args.harnest):
        parser.error("refresh external adapters with both --hostlier and --harnest")
    if not args.hostlier and not args.sandboxes:
        parser.error("refresh requires --hostlier and --harnest, or --sandboxes")
    builds = []
    if args.hostlier:
        builds.extend((
            ("minion", args.hostlier, "./packages/minion/cmd/hostlier-minion"),
            ("harnest", args.harnest, "./cmd/hostlier-harnest"),
        ))
    if args.sandboxes:
        subprocess.run(["python3", str(ROOT / "packages/sandboxes/generate_seccomp.py"), "--check"], check=True)
        builds.extend((name, ROOT, "./cmd/hostlier-" + name)
                      for name in ("sandbox-browser", "sandbox-computer"))
    with tempfile.TemporaryDirectory(dir=ROOT) as temp:
        stage = Path(temp)
        for name, source, command in builds:
            packages[name] = build_package(name, source.resolve(), command, stage)
        (ROOT / "binaries").mkdir(exist_ok=True)
        for output in (stage / "binaries").iterdir():
            output.replace(ROOT / "binaries" / output.name)
        catalog["packages"] = [packages[name] for name in sorted(packages)]
        manifest = stage / CATALOG.name
        manifest.write_text(json.dumps(catalog, indent=2) + "\n")
        manifest.replace(CATALOG)


if __name__ == "__main__":
    main()
