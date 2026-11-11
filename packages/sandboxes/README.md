# Browser and computer sandbox sidecars

Two local Hostlier packages attach an agent-controlled sandbox to an existing v2 application:

- `sandbox-browser`: Chromium controlled through Playwright 1.65.0's remote WebSocket protocol.
- `sandbox-computer`: Xvfb/Openbox Linux desktop, xterm, visible Chromium, and an HTTP screenshot/mouse/keyboard API.

These are additive `project-contribute` packages. They require Hostlier's `sidecars` and `secrets` contribution fields; older CLIs fail on the unknown fields. They do not supply a primary builder or run provisioning code locally. Runtime images are built separately and transferred through Hostlier's normal image/archive deployment path. The four platform binaries for each package are published in `hostlier-packages.json`, with SHA-256 checksums. Runtime images are built separately.

## Install from the catalog

On the CLI machine:

```sh
catalog_version=4c8232f3763989ff4366453eca02d3b0a21901e3
hostlier package install sandbox-browser --version "$catalog_version"
hostlier package install sandbox-computer --version "$catalog_version"
```

These commands use Hostlier's default package Git catalog. Installation requires `--version`. The example pins the published sandbox commit; for updates, obtain the full commit from `hostlier package available` and pass it to both install commands. Older Git-catalog clients can also specify `--repository https://github.com/Usefused/hostlier-packages.git`. The repository remains private and uses your existing local Git credentials.

The installed CLI must understand additive `sidecars` and `secrets` project contributions; package installation alone does not upgrade Hostlier core. Use the compatible CLI/controller built from the sandbox implementation before deployment; this package publication does not release those core changes.

## Build and install from source

From this repository:

```sh
go build -o dist/hostlier-sandbox-browser ./cmd/hostlier-sandbox-browser
go build -o dist/hostlier-sandbox-computer ./cmd/hostlier-sandbox-computer
hostlier package install sandbox-browser --file dist/hostlier-sandbox-browser
hostlier package install sandbox-computer --file dist/hostlier-sandbox-computer
```

Build images for the receiving server's architecture. This example targets Linux ARM64; substitute `linux/amd64` for an x86 server. Both Dockerfiles pin the Playwright image and npm package to 1.65.0. The lockfile pins dependency integrity. Hostlier resolves the built image to immutable content during deployment.

```sh
docker build --platform linux/arm64 -t hostlier-sandbox-browser:local \
  -f packages/sandboxes/runtime/Dockerfile.browser packages/sandboxes/runtime
docker build --platform linux/arm64 -t hostlier-sandbox-computer:local \
  -f packages/sandboxes/runtime/Dockerfile.computer packages/sandboxes/runtime
```

Either push those images to your own registry and set `image`, or save them into your application's directory and set `archive`:

```sh
mkdir -p /path/to/agent/images
docker save -o /path/to/agent/images/browser.tar hostlier-sandbox-browser:local
docker save -o /path/to/agent/images/computer.tar hostlier-sandbox-computer:local
```

Install the updated Hostlier CLI. In the app directory, run `hostlier package add sandbox-browser` and/or `hostlier package add sandbox-computer`. Create stored secrets with `hostlier secret set browser-token` and `hostlier secret set computer-token`. Use independently generated 32-byte random hex values (64 characters); the service requires 32–256 URL-safe token characters. Enter values at the secret prompt, never in package settings.

```yaml
packages:
  sandbox-browser:
    archive: images/browser.tar
    tokenSecret: browser-token
  sandbox-computer:
    archive: images/computer.tar
    tokenSecret: computer-token
```

Remove any unused `image` or `archive` scaffold field. `hostlier plan` shows the combined proposal; `hostlier deploy --server production` builds/transfers artifacts and requests normal deployment review. Package composition uses the default `.hostlier`; `--file` selects an already prepared manifest and bypasses local package hooks.

| Setting | Browser default | Computer default |
| --- | --- | --- |
| `name` | `browser` | `computer` |
| `port` | `9222` | `9333` |
| `urlEnv` | `BROWSER_URL` | `COMPUTER_URL` |
| `tokenEnv` | `BROWSER_TOKEN` | `COMPUTER_TOKEN` |
| `cpu` | `"2"` | `"2"` |
| `memory` | `2Gi` | `2Gi` |

`image` or `archive` and `tokenSecret` are required. CPU choices are 1–8; memory choices are 2/3/4/6/8/16 GiB. Both sidecars are required for readiness. Names, ports and environment destinations may be changed to avoid conflicts with the application. Each browser task should create and close its own context. Each computer sidecar has one desktop shared by its callers.

## Playwright client

Use Playwright 1.65.x in the parent application (major/minor must match). This is `chromium.connect`, not `connectOverCDP`:

```js
import { chromium } from 'playwright';

const endpoint = new URL('/playwright', process.env.BROWSER_URL);
endpoint.protocol = 'ws:';
const browser = await chromium.connect(endpoint.toString(), {
  headers: { Authorization: `Bearer ${process.env.BROWSER_TOKEN}` },
});
try {
  const context = await browser.newContext();
  try {
    const page = await context.newPage();
    await page.goto('https://example.com');
    console.log(await page.title());
    await page.screenshot({ path: '/tmp/example.png' });
  } finally { await context.close(); }
} finally { await browser.close(); }
```

An agent with Playwright tools can use the same connection. The package does not register tools in an external agent automatically; configure its remote-browser connection and token from the supplied environment variables.

## Computer-use client

Every HTTP call requires `Authorization: Bearer <COMPUTER_TOKEN>`. Tokens in query strings are not accepted. Browser-origin requests are rejected. Use the server-to-server API from the parent:

```js
const headers = { Authorization: `Bearer ${process.env.COMPUTER_TOKEN}` };
const response = await fetch(`${process.env.COMPUTER_URL}/v1/screenshot`, { headers });
if (!response.ok) throw new Error('Screenshot failed');
const png = new Uint8Array(await response.arrayBuffer());

await fetch(`${process.env.COMPUTER_URL}/v1/actions`, {
  method: 'POST', headers: { ...headers, 'Content-Type': 'application/json' },
  body: JSON.stringify({ action: 'click', x: 320, y: 180, button: 'left' }),
});
```

| Method and path | Result / JSON input |
| --- | --- |
| `GET /health` | Readiness; returns 503 if a required runtime process stops |
| `GET /v1/info` | Display size (1280×800), supported actions and paths |
| `GET /v1/screenshot` | PNG of the current desktop |
| `POST /v1/actions` | One of the actions below |

```json
{"action":"move","x":320,"y":180}
{"action":"click","x":320,"y":180,"button":"left"}
{"action":"type","text":"Hello"}
{"action":"key","key":"ctrl+l"}
{"action":"key","key":"Return"}
{"action":"scroll","direction":"down","amount":3}
```

Coordinates are zero-based screen pixels. Buttons are `left`, `middle`, `right`; scroll directions are `up`, `down`, `left`, `right` with 1–20 steps. Keys use xdotool/X11 key names (`Return`, `Tab`, `Escape`, `ctrl+c`, `alt+Tab`). Text is limited to 16,384 characters; the JSON request is limited to 64 KiB. Actions and screenshots run in one queue for a consistent desktop view. The desktop intentionally allows terminal interaction within the container; no separate arbitrary host execution API is exposed.

## Isolation, state and operations

API listeners bind to loopback inside the application's network namespace. They have no public route or host port. The parent and its sidecars share that namespace and form one trust group; this is not a VM boundary or per-task network isolation. Outbound network access follows Hostlier's runtime policy.

Containers run as UID/GID 1000 with read-only roots, no privilege escalation, all capabilities dropped, no swap, a 512-process limit and a 1 GiB `/tmp` tmpfs. Chromium's sandbox is explicitly enabled; failure does not fall back to `--no-sandbox`. `/tmp` also holds browser shared-memory spillover, profiles, downloads and desktop home files. On Docker, contents disappear with the container's scratch mount. On k3s, memory `emptyDir` contents survive a container restart and disappear when the Pod is replaced or deleted. No host directories or Docker socket are mounted. Sandbox tokens are removed from GUI/browser child environments.

The seccomp JSON derives from the [Playwright v1.65.0 profile](https://github.com/microsoft/playwright/blob/v1.65.0/utils/docker/seccomp_profile.json), used under [Playwright's Apache-2.0 license](https://github.com/microsoft/playwright/blob/v1.65.0/LICENSE). It retains default-deny semantics and adds an explicit `chroot` allow rule: Chromium calls it inside its unprivileged user namespace, but Docker otherwise filters the upstream rule out when the outer container drops every capability. No container capabilities are added. The modified profile and its upstream license are included in the runtime directory. Hostlier includes its exact bytes in deployment review. Host AppArmor/user-namespace restrictions can still block Chromium; such hosts need an administrator-prepared compatible policy. Both Docker and k3s use generated OCI profiles selected from the receiving server's `amd64` or `arm64` architecture. `generate_seccomp.py` evaluates the source profile's architecture and capability conditions for `cap-drop=ALL` on Linux ≥4.8; privileged syscall rules are excluded before Docker-only metadata is removed. Unsupported policy metadata fails generation. No policy is silently disabled.

Each sidecar follows its parent's lifecycle, replicas and rollback. A required sidecar failing readiness fails the deployment. There is no independent session scheduling, retention, TTL or live desktop video service in this version. The package does not install Tailscale.

## k3s deployment

Use the same package settings, runtime images and Playwright client on Hostlier's managed single-node k3s backend. No `runtime` package setting is required. Build images for the server CPU; package composition automatically chooses the matching seccomp profile. Registry images and Docker image archives use Hostlier's existing containerd preparation path.

The node must satisfy Hostlier's managed security contract before deployment:

- A protected `k3s-security.json` in the controller state directory declares `{"version":1,"podPidsLimit":512}` (or a stricter limit).
- The kubelet uses matching `pod-max-pids`, `fail-swap-on=true` and `root-dir=/var/lib/kubelet`; the host has no active swap. Hostlier checks live settings and requires exactly one local node.
- The Pod limit must be no larger than any secured component's `pidsLimit`. It is shared by the parent and both sidecars, so 512 is a group budget on k3s. Smaller pre-existing node limits remain in force.
- The host permits Chromium's unprivileged user namespaces under its AppArmor policy. Chromium still fails startup if this is blocked.

Hostlier's reviewed Docker-to-k3s migration prepares this contract for existing secured applications. For a new managed k3s installation, the administrator can provision the root-owned mode-0600 policy file before controller initialization; the installer incorporates its kubelet flags. An existing k3s node without the contract needs administrator maintenance to configure the matching node policy and restart k3s and the controller. Merely creating the policy file is insufficient. These packages never change node-wide settings themselves.

Hostlier writes the exact reviewed profile into `/var/lib/kubelet/seccomp` under a content-derived filename and references it through Kubernetes `Localhost` seccomp. Each sandbox runs as a native restartable sidecar with authenticated exec startup/readiness probes. Each gets its own 1 GiB memory `emptyDir`; pinned setup containers apply UID/GID 1000 and mode 1777 before either sidecar starts. Application containers keep all capabilities dropped. Tokens remain in scoped Kubernetes Secret references; no sandbox Service or host port is created.

Native sidecars require Kubernetes 1.29+ with `SidecarContainers` enabled; use the k3s version pinned by your Hostlier release. See the official [sidecar lifecycle](https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/) and [Localhost seccomp](https://kubernetes.io/docs/reference/node/seccomp/) documentation.

## Verification

```sh
python3 packages/sandboxes/generate_seccomp.py --check
python3 -m unittest discover -s packages/sandboxes -p 'test_*.py'
go test ./packages/sandboxes/... ./cmd/hostlier-sandbox-browser ./cmd/hostlier-sandbox-computer
cd packages/sandboxes/runtime
npm test
```

`python3 packages/sandboxes/smoke.py` runs opt-in Docker checks after building both images. It creates only uniquely named disposable containers, uses random in-memory tokens, applies the generated security profile, verifies Playwright context isolation and screenshot/desktop controls, and removes its containers afterward. No host port is published.

From the Hostlier repository, exercise both real package hooks against its Kubernetes renderer (set the path to the built binaries):

```sh
HOSTLIER_SANDBOX_PACKAGE_DIR=/path/to/hostlier-packages/dist \
  go test ./internal/runtime -run TestSandboxPackagesK3s -count=1
```

This checks both architectures, native sidecar ordering, required probes, private scratch mounts, exact seccomp references and scoped credentials. Optional `HOSTLIER_SANDBOX_K3S_EXPORT=/tmp/sandbox-fixtures` exports native-CPU manifests and profiles for an isolated test cluster. Exported credentials are synthetic test values: never deploy these fixtures to a real application. See `VALIDATION.md` for observed live results and limits.
