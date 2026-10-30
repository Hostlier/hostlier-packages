# Validation

`make check` runs Go vet and race-enabled tests across the package executables, database/provider libraries, resource adapters, protocol SDK, and protected state/bundle helpers. `make pulumi-test` checks the isolated Pulumi module separately.

`make build` compiles each standard package independently; `make linux` cross-compiles Linux amd64 and arm64 artifacts. `make pulumi` builds the optional Pulumi executable separately.

Tests requiring Docker, Kubernetes, S3, provider accounts, Terraform, or Pulumi CLIs remain opt-in. A passing ordinary suite does not imply live provider provisioning has been validated. The controller's approval, private-output, encrypted-state and installed-CLI integration tests remain in the Hostlier core repository.

The extraction preserves the version-one private-pipe protocol. `SOURCE.json` records the original input hashes. No core module dependency, local core checkout replacement, or runtime state is required to build these packages.
