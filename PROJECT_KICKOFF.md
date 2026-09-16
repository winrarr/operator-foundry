# Project kickoff

This repository is a reference-rich starting point for a new Kubernetes operator. The first user message supplies the operator's north-star goal. This file explains how to turn that goal into a well-founded project and then continue toward the full outcome.

## 1. Orient yourself

Read the repository README, inspect the source layout, and run the smallest useful local checks. Treat the existing API, controllers, clients, tests, documentation, and generated output as examples of conventions and patterns. They are not automatically requirements for the new operator.

Before changing public API shape, ownership, deletion behavior, authentication, or installation behavior, understand the relevant source files and record the reasoning in the appropriate durable document.

## 2. Capture the goal and user stories

Start from the user's full ambition. Do not narrow the project prematurely. Create or update the product brief with:

- the problem the operator should solve;
- who uses it and how;
- the desired end-to-end workflows;
- current user stories;
- future user stories that are already visible from the goal;
- the goal for the first useful implementation slice.

User stories describe desired behavior and help the project retain its direction. The first slice is a sequencing decision, not a definition of project completion.

Make reasonable assumptions when they are low risk and record them. Ask the user only when an unresolved choice materially changes the product, public API, security boundary, or operational risk.

## 3. Research before designing

Research the external system and comparable implementations before committing to the Kubernetes API.

Prefer evidence in this order:

1. official product documentation and API specifications;
2. official SDKs, source code, and compatibility notes;
3. tests and operational documentation from established projects;
4. other operators and adjacent tools with similar goals.

For the external system, investigate its identity model, scopes and tenancy, authentication, create/read/update/delete behavior, idempotency, eventual consistency, pagination, rate limits, error responses, version compatibility, drift behavior, and safe deletion semantics.

For comparable projects, inspect implementation and tests rather than copying their README claims. Look for how they model references, ownership, adoption, finalizers, status, retries, secrets, dependency watches, upgrades, and live validation.

Record dated research notes with links, repository commits or paths where applicable, source quality, observations, inferences, unresolved questions, and the decisions the evidence informs. Use comparable projects as inspiration, never as authority. Similar external products may have materially different semantics.

## 4. Evaluate the patterns in this repository

Review every relevant pattern before deciding what the new operator needs. Do not silently omit a pattern merely because the first resource does not use it. Adoption, orphaning, deletion, finalizers, and drift detection should be treated as likely requirements for external resources and need an explicit evaluation.

At minimum, consider:

- connection and authentication resources;
- same-namespace and cross-namespace reference boundaries;
- creation and adoption policy;
- deletion policy, finalizers, orphaning, and dependency loss;
- drift detection and external recreation;
- dependency readiness, watch mapping, and retry intervals;
- stable external identity in status;
- Secret inputs and operator-managed Secret outputs;
- multi-tenancy and cluster-scoped resources;
- relationships, claims, and ownership of individual edges;
- API versioning, generated assets, documentation, and installation surfaces.

For each pattern, decide whether it is used, deferred, or not applicable, and capture the reason when the choice is not self-evident. If a pattern is deferred, keep the design extensible and reflect the future need in the user stories or backlog rather than pretending the question does not exist.

## 5. Design the operator

Create or update the architecture document before implementing complex behavior. Define the resource model, scopes, references, status contract, reconciliation state transitions, external identity mapping, ownership boundaries, and failure behavior.

Use the simplest API that can support the full goal. Prefer Kubernetes references and stable status fields over raw external IDs where the relationship can be represented safely through Kubernetes objects.

Record consequential choices in decision records. Keep unsettled alternatives and staged work in a design document until a choice is accepted.

## 6. Implement a vertical slice

Implement the first useful slice through every affected layer:

1. API types, validation, defaults, status, print columns, and registration;
2. typed external-client interfaces and HTTP or SDK contract tests;
3. reconciliation, dependency handling, retries, status, drift, and lifecycle behavior;
4. fake-client reconciliation tests for observable transitions and edge cases;
5. samples, resource guides, architecture updates, and operational notes;
6. generated CRDs, RBAC, chart assets, and API reference documentation.

Keep the existing reference examples available while the new operator becomes solid. Reuse their structure and tested techniques, then reshape or remove example-domain code only when the new project's design requires it.

## 7. Verify and continue

Run the canonical checks early and often:

```sh
make check
make verify-generated
make build
```

Use a disposable Kind or equivalent live environment when real Kubernetes or external-system behavior remains material and unproven. Keep live credentials, kubeconfigs, tokens, and local state out of source, tests, samples, and documentation.

At each meaningful handoff, report the user stories addressed, decisions made, research evidence used, patterns adopted or deferred, automated checks run, live behavior exercised, and important assumptions that remain unverified. Continue toward the broader goal after the first slice instead of treating the first slice as completion.
