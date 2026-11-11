#!/usr/bin/env python3
"""Opt-in Docker smoke test for the built sidecar images. Never contacts a Hostlier server."""
import json
import os
from pathlib import Path
import secrets
import subprocess
import time
import uuid

ROOT = Path(__file__).resolve().parent


def docker(*args, input=None, env=None, check=True):
    return subprocess.run(["docker", *args], input=input, text=True, capture_output=True,
                          env=env, check=check, timeout=180)


def smoke(kind, containers):
    name = "hostlier-sandbox-smoke-" + uuid.uuid4().hex[:12]
    token = secrets.token_hex(32)
    port = "9222" if kind == "browser" else "9333"
    env = {**os.environ, "SANDBOX_TOKEN": token}
    architecture = docker("image", "inspect", "--format", "{{.Architecture}}", "hostlier-sandbox-" + kind + ":local").stdout.strip()
    if architecture not in ("amd64", "arm64"):
        raise RuntimeError("Unsupported sandbox image architecture")
    containers.append(name)
    try:
        docker("run", "--detach", "--name", name, "--network", "none" if len(containers) == 1 else "container:" + containers[0], "--read-only",
               "--user", "1000:1000", "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true",
               "--security-opt", "seccomp=" + str(ROOT / "runtime" / ("seccomp-" + architecture + ".json")),
               "--memory", "2g", "--memory-swap", "2g", "--cpus", "2", "--pids-limit", "512",
               "--tmpfs", "/tmp:rw,nosuid,nodev,size=1073741824,mode=1777",
               "-e", "SANDBOX_TOKEN", "-e", "SANDBOX_PORT=" + port,
               "hostlier-sandbox-" + kind + ":local", env=env)
        for _ in range(120):
            result = docker("exec", name, "node", "/opt/hostlier-sandbox/health.mjs", check=False)
            if result.returncode == 0:
                break
            state = json.loads(docker("inspect", name).stdout)[0]["State"]
            if not state["Running"]:
                logs = docker("logs", name, check=False)
                raise RuntimeError(kind + " exited before readiness: " + (logs.stdout + logs.stderr).replace(token, "[redacted]"))
            time.sleep(1)
        else:
            logs = docker("logs", name, check=False)
            raise RuntimeError(kind + " failed readiness: " + logs.stdout.replace(token, "[redacted]") + logs.stderr.replace(token, "[redacted]"))
        script = (ROOT / "runtime/test" / (kind + "-live.mjs")).read_text()
        # A separate parent-like client reaches each sidecar through the group's loopback.
        result = docker("run", "--rm", "-i", "--network", "container:" + containers[0],
                        "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true",
                        "-e", "SANDBOX_TOKEN", "-e", "SANDBOX_PORT=" + port,
                        "--entrypoint", "node", "hostlier-sandbox-browser:local", "--input-type=module",
                        input=script, env=env)
        print(result.stdout.strip())
        status = json.loads(docker("inspect", name).stdout)[0]
        assert not status["HostConfig"]["Privileged"]
        assert not status["HostConfig"].get("PortBindings")
    except Exception:
        # The caller removes both siblings, including one that failed to start.
        raise


if __name__ == "__main__":
    containers = []
    try:
        for kind in ("browser", "computer"):
            smoke(kind, containers)
        print("Both sidecars share one private network namespace; separate parent clients passed.")
    except subprocess.CalledProcessError as error:
        # Commands contain only environment names; never dump container env or private output.
        raise SystemExit("Docker smoke command failed (exit " + str(error.returncode) + ")") from None
    finally:
        for name in reversed(containers):
            docker("rm", "--force", name, check=False)
