# CI configuration

`github-actions.example.yml` is an inactive GitHub Actions configuration for the package checks and independent Pulumi module. Local verification uses `make check`, `make build linux`, and `make pulumi-test pulumi`.

The GitHub login used for the initial repository push does not have the OAuth `workflow` scope. Consequently no Actions workflow is installed or triggered by this repository. A maintainer with workflow-write permission can copy this file to `.github/workflows/check.yml` to enable it.
