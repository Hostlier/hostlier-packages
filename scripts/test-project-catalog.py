#!/usr/bin/env python3
"""Smoke-test real Git catalog discovery, installation and enrollment without deploying."""
import argparse
import json
from pathlib import Path
import subprocess
import tempfile


def run(cli, root, *args):
    result = subprocess.run([cli, *args, "--json"], cwd=root, text=True,
                            stdout=subprocess.PIPE, check=True)
    return json.loads(result.stdout)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--cli", required=True, type=Path)
    parser.add_argument("--repository", help="override the default official Git catalog")
    parser.add_argument("--version", required=True)
    args = parser.parse_args()
    cli = args.cli.resolve()
    source = ["--repository", args.repository] if args.repository else []
    selection = [*source, "--version", args.version]
    with tempfile.TemporaryDirectory(prefix="hostlier-catalog-") as tmp:
        root = Path(tmp).resolve()
        catalog = run(cli, root, "package", "available", *selection)
        selection = [*source, "--version", catalog["version"]]
        available = {item["name"] for item in catalog["packages"]}
        if not {"harnest", "minion", "postgres"} <= available:
            raise ValueError(f"incomplete package catalog: {available}")
        state = ["--state-dir", str(root / "state")]
        for name in ("harnest", "minion"):
            project = root / name
            project.mkdir()
            run(cli, project, "init")
            manifest = project / ".hostlier"
            before = manifest.read_bytes()
            missing = subprocess.run([cli, "package", "add", name, *state], cwd=project,
                                     text=True, capture_output=True)
            if missing.returncode == 0 or "not installed locally" not in missing.stderr or manifest.read_bytes() != before:
                raise ValueError(f"{name} could be added without installing its package")
            receipt = run(cli, project, "package", "install", name, *selection, *state)
            capability = run(cli, project, "package", "describe", name, *state)
            added = run(cli, project, "package", "add", name, *state)
            fields = capability["configSchema"]["properties"]
            generated = manifest.read_text()
            if not added["added"] or not fields or f"  {name}:\n" not in generated:
                raise ValueError(f"missing installed contract for {name}")
            if not all(f"    {key}:" in generated for key in fields):
                raise ValueError(f"incomplete {name} input scaffold")
            repeat = run(cli, project, "package", "add", name, *state)
            if repeat["added"] or manifest.read_text() != generated or set(project.iterdir()) != {manifest}:
                raise ValueError(f"package add changed unexpected files for {name}")
            print(f"{name}: {receipt['digest']}, fields: {', '.join(sorted(fields))}")


if __name__ == "__main__":
    main()
