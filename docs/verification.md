# Verification

Verification is evidence about observable behavior, not a list of commands to run without interpretation.

| Command | Evidence | Does not prove |
| --- | --- | --- |
| `make format-check` | Go sources are formatted | The design is correct |
| `make generate` | Generated code and API reference can be produced | Generated output is committed |
| `make verify-generated` | Tracked generated output matches its sources | External API compatibility |
| `make test` | Unit, HTTP contract, and reconciliation tests pass | Live Kubernetes or external-system behavior |
| `make helm-lint` and `make helm-template` | The chart is structurally valid and renders | A production deployment is safe |
| `make kustomize-build` | The installation bundle renders | Cluster admission and runtime behavior |
| `make docs-build` | The documentation site and links validate | Documentation is complete for a future operator |

The reference status contract uses Kubernetes conditions named `Ready`, `Reconciling`, and `Stalled`, with `status.observedGeneration`. The tests show the intended lifecycle classification without requiring a particular external kstatus library.

When adding an operator resource, test the observable states that its contract supports: initial dependency waiting, successful reconciliation, invalid configuration, external failures and recovery, remote deletion, updates, idempotency, and deletion behavior when applicable.
