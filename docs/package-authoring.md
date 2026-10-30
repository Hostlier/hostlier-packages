# Authoring a Hostlier package

A package is an executable that handles a small, versioned protocol. The controller owns user authorization, secrets, dependency ordering, operation progress and encrypted workflow state. Packages implement the reviewed work.

## Workflow contract

Use [`sdk/packagekit`](../sdk/packagekit) to declare typed configuration, input, state and output codecs. `Define` and `Describe` expose package capabilities; `Serve` handles one bounded version-one request through stdin/stdout. Request and response types are in [`sdk/workflow`](../sdk/workflow).

A command normally accepts `describe` and `serve`. Native database packages also implement the separate `resource` protocol using [`sdk/resourceprotocol`](../sdk/resourceprotocol) and [`internal/resourcepackage`](../internal/resourcepackage).

- `plan`: validate intent and prepare a bounded review with an opaque execution payload.
- `apply`: execute only the reviewed request and return state plus declared outputs.
- `inspect` and `reconcile`: observe uncertain outcomes according to the package's supported lifecycle contract; they must not blindly repeat a mutation.

Inputs and predecessor outputs are passed over the private request pipe. Credentials must not appear in command arguments, stdout diagnostics, source files or workflow examples. Private outputs remain separate from application-visible bindings. Export only declared outputs.

See the [engine packages](../packages/engines) for database behavior and the [infrastructure adapter](../packages/iac) for managed state and reconciliation. The reusable [contract tests](../sdk/packagekit/contracttest) exercise protocol handling; each package also needs behavior tests for its actual provider or runtime.

## Compatibility

Both Hostlier core and these packages currently implement protocol version 1. Source-level Go imports do not cross repositories. Keep JSON field names, required fields, actions, output classification and bounds compatible. A protocol-breaking change must be coordinated with a controller version that explicitly supports it.

Hostlier installs an executable under a selected package name and pins its SHA-256 digest during review. Changes to installed executable bytes require a fresh plan. The core integration tests validate installation, approval, execution and recovery across actual processes; they stay in the Hostlier repository.
