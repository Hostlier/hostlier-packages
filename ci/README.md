# CI configuration

`.github/workflows/sandbox-runtimes.yml` is the active manual runtime publication workflow. It builds and exercises both sandboxes on native Linux AMD64 and ARM64 runners before publishing their GHCR platform images and multiarch indexes. Dispatch with a new `version` such as `1.65.0-1`; existing release tags are rejected. The `runtime-images` artifact records immutable index references for a reviewed package catalog update. The first publication of each GHCR package must be made public in its GitHub package settings and verified with an anonymous pull before shipping package defaults.

`github-actions.example.yml` remains an inactive example for broader package checks and the independent Pulumi module. Local verification uses `make check`, `make build linux`, and `make pulumi-test pulumi`.
