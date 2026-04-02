# executor package layout

This package keeps App methods in the root package and extracts implementation details into reusable subpackages.

## Root package

- app.go: app bootstrap, queue consumer, task execution flow
- callback.go / node_auth.go: callback and node auth behaviors

## Subpackages

- contracts: queue/callback DTOs
- store: database models used by executor runtime
- runtimeexec: artifact unzip, checksum, command execution utilities

The compatibility wrapper layer was removed in phase 2; root code now imports these subpackages directly.
