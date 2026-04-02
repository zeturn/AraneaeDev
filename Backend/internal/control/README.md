# control package layout

This package keeps App methods in the root package to preserve Go method locality, and moves reusable concerns into focused subpackages.

## Root package

- app.go: app bootstrap, lifecycle, rabbit/grpc wiring
- routes.go / routes_schedule.go / routes_runs.go / routes_compat.go: HTTP handlers
- auth.go / node_auth.go / scheduler.go / schedule_chain.go: service flows

## Subpackages

- contracts: transport DTOs used by control routes/queue
- security/password: password hashing and verification helpers
- infra/netx: network listener adapter

The compatibility wrapper layer was removed in phase 2; root code now depends on these subpackages directly.
