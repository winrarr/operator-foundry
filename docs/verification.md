# Verification

Verification is evidence about observable behavior, not a list of commands to run without interpretation.

| Command | Evidence | Does not prove |
| --- | --- | --- |
| `make format-check` | Go sources are formatted | The design is correct |
| `make generate` | Generated code and API reference can be produced | Generated output is committed |
| `make verify-generated` | Tracked generated output matches its sources | External API compatibility |
| `make test` | Unit, HTTP contract, and reconciliation tests pass | Live Kubernetes or external-system behavior |
| `make kind-e2e` | The default-CNI cluster installs the operator and exercises the golden lifecycle against the mock API | A real third-party API or every possible Kubernetes distribution |
| `KIND_CNI=cilium make kind-e2e` | Cilium/Hubble readiness and the example NetworkPolicy boundary work in addition to the golden lifecycle | Production network topology or policy design |
| `make helm-lint` and `make helm-template` | The chart is structurally valid and renders | A production deployment is safe |
| `make kustomize-build` | The installation bundle renders | Cluster admission and runtime behavior |
| `make docs-build` | The documentation site and links validate | Documentation is complete for a future operator |

The reference status contract uses Kubernetes conditions named `Ready`, `Reconciling`, and `Stalled`, with `status.observedGeneration`. The tests show the intended lifecycle classification without requiring a particular external kstatus library.

The reference E2E matrix is intentionally broader than the unit suite: initial dependency waiting, successful reconciliation, idempotency, updates, one-shot external failure and recovery, remote deletion and recreation, adoption, managed deletion, orphaning, and (in Cilium mode) an allow/deny NetworkPolicy boundary. When adding a pattern, extend the smallest test layer that can prove it and use Kind only for behavior that requires a real cluster or network.
