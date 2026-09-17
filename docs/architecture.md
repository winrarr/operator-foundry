# Architecture

Operator Foundry has two related purposes:

1. provide a usable repository baseline for a new operator;
2. provide concrete reference patterns for agents and maintainers.

The active Go code demonstrates a small external-system operator shape:

```text
Kubernetes Secret → PatternConnection → PatternResource reconciler → typed external client → mock external API in E2E
```

The example API and controllers are deliberately generic. They demonstrate how the layers fit together, but they do not define the API of future operators.

## Boundaries

- `api/patterns/v1alpha1` contains the reference custom resources and their Kubernetes contract.
- `internal/controller` contains condition, dependency, lifecycle, and reconciliation patterns.
- `internal/exampleclient` contains a deliberately small typed HTTP client surface.
- `internal/mockexternalapi` and `cmd/mockexternalapi` contain the disposable external system used only by live tests.
- `config/` and `charts/` contain installation and generated delivery assets.
- `docs/patterns/` explains when the reference patterns are useful and what must be reconsidered.
- `PROJECT_KICKOFF.md` is the source of truth for how a new operator should use this repository.

Generated files are derived from API types, controller markers, and documentation configuration. Change their sources and regenerate; do not hand-edit generated CRDs, RBAC, deepcopy code, or the API reference.

The mock API has a separate administrative surface for deterministic E2E setup, remote deletion, and one-shot failure injection. It is not part of the operator's production surface and should not be used as an external API design authority.

## Design principles

- Treat the external system's semantics as authoritative.
- Keep Kubernetes API design, client behavior, reconciliation, status, tests, and documentation as one contract.
- Prefer explicit ownership and deletion decisions over accidental external mutations.
- Make errors observable through status without leaking credentials.
- Keep the reference examples available while a new project is being designed and implemented.
