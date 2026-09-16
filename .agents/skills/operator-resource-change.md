---
name: operator-resource-change
description: Implement or change an external-system custom resource end to end.
---

# External resource change

Treat a resource as one contract spanning the Kubernetes API, external client, reconciliation, ownership, status, tests, documentation, samples, generated output, and installation assets.

## Establish the contract

Read `AGENTS.md`, `PROJECT_KICKOFF.md`, the relevant product and architecture documents, and the closest reference pattern. Research the external operation in official documentation or an API specification. Determine identity, scope, references, authentication, create/update/read behavior, adoption, deletion, drift, retries, and observable status before editing.

Treat adoption, finalizers, deletion policy, orphaning, and drift detection as likely requirements. Do not include them mechanically, but do not omit them silently. Record the reason when a pattern is deferred or not applicable.

## Implement the vertical slice

1. Define API fields, validation, defaults, status, print columns, and registration.
2. Add the smallest typed client surface and contract tests for changed external operations.
3. Reconcile dependencies, lifecycle, ownership, create/update, drift, and status in a legible order.
4. Test observable success, failure, dependency, idempotency, remote-deletion, and lifecycle states that the resource supports.
5. Update guides, user stories, samples, architecture, and operational documentation.
6. Regenerate CRDs, RBAC, deepcopy code, chart assets, and API reference documentation.

Never hand-edit generated output. Preserve unrelated work in a dirty checkout.

## Verify

Run focused tests while iterating, then run `make check` and `make verify-generated`. Use an isolated Kind or equivalent environment when real Kubernetes or external-system behavior remains material and unproven. Never use production credentials in tests or documentation.
