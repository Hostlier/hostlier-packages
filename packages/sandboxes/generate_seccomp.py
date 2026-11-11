#!/usr/bin/env python3
"""Resolve the reviewed Playwright/Docker policy for cap-drop=ALL and Linux >=4.8.

Generated OCI profiles work with Docker and containerd. Never strip conditional
metadata without evaluating it: doing so would allow privileged host syscalls.
"""
import argparse
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent / "runtime"
ARCHITECTURES = {"amd64": "SCMP_ARCH_X86_64", "arm64": "SCMP_ARCH_AARCH64"}


def included(rule, arch):
    """Evaluate Docker filters for a container with no outer capabilities."""
    inc, exc = rule.get("includes", {}), rule.get("excludes", {})
    if set(inc) - {"caps", "arches", "minKernel"} or set(exc) - {"caps", "arches"}:
        raise ValueError("unreviewed seccomp filter")
    if inc.get("minKernel") not in (None, "4.8"):
        raise ValueError("unreviewed minimum kernel")
    return not (inc.get("caps") or
                (inc.get("arches") and arch not in inc["arches"]) or
                arch in exc.get("arches", []))


def generate(source, arch):
    """Freeze architecture and capability decisions, preserving argument checks."""
    if set(source) != {"defaultAction", "archMap", "syscalls"}:
        raise ValueError("unreviewed profile fields")
    if source["defaultAction"] != "SCMP_ACT_ERRNO":
        raise ValueError("profile must default deny")
    native = ARCHITECTURES[arch]
    mapping = next(entry for entry in source["archMap"] if entry["architecture"] == native)
    rules = []
    for rule in source["syscalls"]:
        if set(rule) - {"names", "action", "args", "comment", "includes", "excludes"}:
            raise ValueError("unreviewed syscall fields")
        if included(rule, arch):
            rules.append({key: value for key, value in rule.items()
                          if key in ("names", "action", "args") and value is not None})
    return {"defaultAction": source["defaultAction"],
            "architectures": [native, *mapping["subArchitectures"]], "syscalls": rules}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="fail if committed profiles differ")
    args = parser.parse_args()
    source = json.loads((ROOT / "seccomp.json").read_text())
    for arch in ARCHITECTURES:
        path = ROOT / ("seccomp-" + arch + ".json")
        expected = json.dumps(generate(source, arch), indent=2) + "\n"
        if args.check:
            if not path.exists() or path.read_text() != expected:
                raise SystemExit(str(path) + " needs regeneration")
        else:
            path.write_text(expected)


if __name__ == "__main__":
    main()
