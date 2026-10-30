# Hostlier packages

Optional, independently built executables for Hostlier database, provider, and infrastructure operations. This repository is private for now and contains the source needed to inspect, test, and build each package without checking out the Hostlier controller or CLI.

Hostlier core lives in [Usefused/hosting-sdk](https://github.com/Usefused/hosting-sdk/tree/main/hostlier). The CLI installs package binaries on the server; the controller calls them through versioned JSON over private stdin/stdout pipes. The package source and its implementation dependencies stay outside both core executables.

## Packages

| Executable | Implementation | Current scope |
| --- | --- | --- |
| `hostlier-postgres` | [Native engines](packages/engines), [runtime adapters](internal/resources) | Local PostgreSQL instances; scoped databases on existing servers |
| `hostlier-redis` | [Native engines](packages/engines), [runtime adapters](internal/resources) | Local Redis instances; ACL-scoped access on existing servers |
| `hostlier-neo4j` | [Runtime adapters](internal/resources) | Local Neo4j resource operations |
| `hostlier-aiven` | [Provider adapters](packages/providers) | Direct Aiven PostgreSQL/Valkey provisioning; cloud end-to-end validation remains pending |
| `hostlier-railway` | [Provider adapters](packages/providers) | Existing-connection workflows; direct managed creation is not implemented |
| `hostlier-terraform` | [Terraform adapter](packages/terraform) | Explicitly prepared and pinned Terraform bundles |
| `hostlier-pulumi` | [Pulumi adapter](packages/pulumi) | Explicitly prepared and pinned Pulumi bundles; separate Go module |
| `hostlier-package` | [Legacy configuration previews](packages/README.md) | Offline schema previews; does not provision infrastructure |

The experimental [Usefused adapter](packages/usefused) is retained as prototype source and is not a supported published integration.

## Build and test

Use Go 1.26 or newer, with access to the public Go dependency proxy on the first build.

```sh
make build       # native packages in dist/
make linux       # Linux amd64 and arm64 packages
make check       # vet and race tests; no cloud credentials required
make pulumi      # optional, separately isolated Pulumi dependencies
make pulumi-test
```

Live Docker, Kubernetes, provider, and infrastructure-tool tests require explicit opt-in and their documented fixtures. Ordinary checks do not provision external infrastructure. See [validation](docs/validation.md).

Build for the server architecture, then install with a separately installed Hostlier CLI:

```sh
sudo hostlier package install postgres --file ./dist/hostlier-postgres-linux-arm64
sudo hostlier package describe postgres
sudo hostlier package list
```

Installed executables live at `/var/lib/hostlier/packages/<name>`. Use `--state-dir DIR` for another controller state directory. With no explicit `--server` or `--socket`, installation is server-local. The controller pins the executable digest during planning and rejects replacement before an approved apply.

## Examples and sample workflows

[Examples](examples/README.md) are editable starting points for building an app and composing package operations. They are not a hosted product catalog or an installation registry. Copy a workflow, change its app image, domain and resource settings, and review it with `hostlier plan` before deployment.

Hostlier's product-definition syntax can name inputs and compose steps; examples using it remain sample workflows at this stage. A managed template experience and registry can be designed separately.

## Repository layout

- `cmd/`: one entry point per package executable.
- `packages/`: engine, provider, and infrastructure-tool implementations.
- `sdk/`: standalone package authoring helpers and version-one wire contracts.
- `internal/resources/`: Docker and Kubernetes database runtime implementations.
- `internal/resourcepackage/`: bounded resource protocol server.
- `internal/testexec/`: local package execution fixture used by optional Terraform tests.
- `examples/`: sample workflows with placeholders and stored-secret references.
- `docs/`: package behavior, authoring, and validation notes.

The SDK has no import or module replacement pointing at the core repository. Pulumi's nested module replaces this repository's root module with `../..`, entirely within this checkout.

## Source provenance and compatibility

[SOURCE.json](SOURCE.json) records the initial extraction's source commit and SHA-256 hashes of the original selected inputs. Import paths were rewritten and the package-only SDK was isolated afterward. Repository history, credentials, local state, and generated executables were not imported.

This repository is the home for future package development. Existing core-checkout copies remain as compatibility fixtures during migration; core integration tests still live with the controller and CLI. Changes to the wire contracts must retain compatibility with Hostlier's version-one protocol or explicitly introduce a new supported version. [Protocol notes](docs/package-authoring.md) describe this boundary.

## Continuous integration

Local verification commands are available through the Makefile. An [inactive GitHub Actions example](ci/README.md) is included; enabling it requires a GitHub credential with permission to publish workflows.

## CLI installation from GitHub

With a Hostlier CLI that supports GitHub packages, select a published release:

```sh
hostlier package install postgres --version v0.1.0 --server production
hostlier package list --server production
```

The CLI selects the connected server's OS and architecture, downloads the release asset locally, checks SHA-256 and executable format, uploads inert chunks, and requests exact hardware approval before installation. GitHub credentials remain on the CLI machine; authenticate using `gh auth login`, `GH_TOKEN`, or `GITHUB_TOKEN` while this repository is private. The controller requires neither Go, Git, nor GitHub access. Install to a host directly through its saved profile; fleet `--on` routing is not supported for package installation.

The local Go package-authoring SDK can be added to an existing project:

```sh
go mod init example.com/my-package
hostlier sdk install --version v0.1.0
```

This resolves the repository tag to a commit and adds `github.com/Usefused/hostlier-packages/sdk/packagekit` through Go modules. It is the standalone package-authoring SDK, distinct from the broader hosting-sdk project. Go and Git are needed on the development machine. Go authentication and private-module settings are scoped to the install command; subsequent direct Go commands need your normal private-module setup.

## Release artifacts

`make release` builds standard packages for Linux and macOS, amd64 and arm64, in `dist/releases/`, with `SHA256SUMS`. Asset names are `hostlier-NAME-OS-ARCH`, as required by the CLI. Set `RELEASE_OS=linux`, `RELEASE_ARCH=arm64`, or `WITH_PULUMI=1` to adjust the build. Pulumi remains an optional separate-module build; standard releases do not include it unless explicitly selected.

Publish assets built from the release tag together with `SHA256SUMS`. Release automation is not active; the inactive Actions example does not publish artifacts. Keep this repository private until explicitly approved for public access.

## Provider-independent Git catalog

The root `hostlier-packages.json` makes package discovery available through ordinary Git transport:

```sh
hostlier package available --repository https://github.com/Usefused/hostlier-packages.git
hostlier package install postgres --repository https://github.com/Usefused/hostlier-packages.git --version COMMIT_FROM_LIST --server production
```

Use a CLI with Git-catalog support. Discovery returns the resolved source commit as `version`; copy it into the install command. The original `v0.1.0` tag predates this catalog, so select a current catalog commit instead. The repository remains private, and its own release asset API URLs reuse the local GitHub login.

The catalog format does not require GitHub: each package has named platform artifacts with a SHA-256 and either a tracked-file `path` or an HTTPS `url`. A mirror can host the Git repository on GitLab, Bitbucket, or a self-hosted server and choose its own artifact locations. Generic external HTTPS artifacts do not inherit Git repository credentials; tracked Git blobs support private or offline distribution. See the core [Git catalog guide](docs/git-package-catalog.md) for the schema. Update the catalog in a new commit when publishing replacement binary artifacts; existing commit-based selections retain their previous checksums.
