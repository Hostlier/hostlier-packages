# Packages from a Git repository

Package catalogs can live in any Git repository reachable over HTTPS or SSH, including a self-hosted server. Use the CLI's `--repository` option:

```sh
hostlier package available --repository https://git.example.com/team/packages.git
hostlier package available --repository git@git.example.com:team/packages.git --version v1
hostlier package available --repository https://git.example.com/team/packages.git --server production
hostlier package install postgres --repository https://git.example.com/team/packages.git --version COMMIT_FROM_LIST --server production
```

`package list --repository URL` is an alias for discovery. Discovery defaults to the repository's HEAD, and its `version` output is the resolved commit. Installation requires `--version` with a tag, branch, or commit; it resolves that selection once and binds approval to the commit and binary checksum. Use the commit printed by discovery to install precisely that catalog even if a branch or tag moves later. `file:///absolute/path/repository` also supports local/offline mirrors.

Git must be installed on the CLI machine. Normal Git credential helpers, SSH configuration, and SSH agents provide repository access. HTTPS URLs cannot contain credentials; SSH URLs may contain a public username. GitHub URLs can reuse the CLI's existing GitHub login. Other Git hosts require no GitHub account, API, or release service.

The controller receives verified binary bytes and public provenance. It does not fetch Git sources or require Git on the server. Installation uses the existing exact approval flow. Browsing never runs repository scripts, invokes builds, checks out files, or initializes submodules. A shallow Git fetch may transfer tracked binary objects; HTTPS artifact downloads occur only during installation.

## Catalog format

Place a regular `hostlier-packages.json` file at the repository root:

```json
{
  "version": 1,
  "packages": [
    {
      "name": "postgres",
      "artifacts": [
        {
          "os": "linux",
          "arch": "arm64",
          "path": "binaries/hostlier-postgres-linux-arm64",
          "sha256": "REPLACE_WITH_64_LOWERCASE_HEX_CHARACTERS"
        },
        {
          "os": "linux",
          "arch": "amd64",
          "url": "https://downloads.example.com/v1/hostlier-postgres-linux-amd64",
          "sha256": "REPLACE_WITH_64_LOWERCASE_HEX_CHARACTERS"
        }
      ]
    }
  ]
}
```

Each artifact uses exactly one `path` or `url`. Paths refer to ordinary files in the same Git commit as the catalog; symlinks, submodules, traversal, and Git LFS pointer files are not executable artifacts. URLs must use HTTPS. Both forms require the exact binary's SHA-256. Supported targets are Linux and macOS (`darwin`), on `amd64` and `arm64`. Package names and platform entries must be unique. A package with no artifacts is omitted from the available list.

Binaries are limited to 128 MiB. Catalog files are limited to 1 MiB. These limits apply to reads and installation; the Git fetch itself can transfer additional repository objects. The CLI independently checks the actual executable format and platform before upload, and the controller checks them again before activation.

Git credentials are not forwarded to arbitrary artifact URLs. Generic HTTPS artifacts must be accessible without repository credentials; store private artifacts as tracked Git blobs when needed. For GitHub-backed catalogs only, exact `api.github.com/repos/OWNER/REPO/releases/assets/ID` URLs for that same repository can reuse the local GitHub token. Other repositories and artifact hosts do not receive it, including after redirects. The catalog format itself is independent of the artifact hosting provider.

Current Hostlier clients use the official Git catalog by default. On older clients with Git-catalog support, use `--repository https://github.com/Usefused/hostlier-packages.git` explicitly. For GitHub release asset discovery and installation, select `--github OWNER/REPO`.
