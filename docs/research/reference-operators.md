# Reference operator research — 2026-09-16

## Scope

This note records the repository evidence used to shape Operator Foundry. It compares local implementations of the Harbor Operator, Infisical Entity Operator, and OpenBao Entity Operator. The note is about reusable engineering patterns, not external-product compatibility.

## Sources

- [Harbor Operator](https://github.com/winrarr/harbor-operator) and its `AGENTS.md`
- [Infisical Entity Operator](https://github.com/winrarr/infisical-entity-operator) and its `AGENTS.md`
- [OpenBao Entity Operator](https://github.com/winrarr/openbao-entity-operator) and its `AGENTS.md`

The source repositories were inspected locally at `/home/rkth/Documents/dev/harbor-operator`, `/home/rkth/Documents/dev/infisical-entity-operator`, and `/home/rkth/Documents/dev/openbao-entity-operator`. They may contain uncommitted work. Current files are evidence of implementation patterns, not immutable baselines.

## Observations

- All three repositories use Go, controller-runtime, Kubebuilder-style API markers, generated CRDs and RBAC, Helm packaging, Make-based verification, and Kind-oriented live validation.
- All three treat API definitions, external client behavior, reconciliation, status, tests, documentation, and generated delivery assets as one resource contract.
- Harbor has the richest shared controller layer, including connection selection, dependency watches, drift detection, multi-tenancy, and focused repo-local skills.
- Infisical has an explicit status lifecycle test for `Ready`, `Reconciling`, and `Stalled`, plus conflict-safe status patching and research-driven API scope.
- OpenBao has a relatively small typed HTTP client and a clear separation between API types, controllers, client code, generated assets, and strict documentation validation.
- The repositories have evolved independently. Their Makefiles, Dockerfiles, linter configuration, docs tooling, and generated-artifact checks are similar but not identical.
- Harbor's release branch patch train is a tested, idempotent pattern that handles dependency eligibility, chart-only history, exact check waiting, immutable tags, missing publication recovery, and stale metadata; Operator Foundry carries it as an inactive optional reference.
- Infisical's E2E workflow uses a local BuildKit cache handoff and committed-artifact deployment path; Operator Foundry uses the same cache approach and now separates committed live deployment from generation-heavy development deployment.
- The kstatus convention is implemented through Kubernetes conditions and tests rather than a common runtime dependency.

## Inferences

The durable shared layer is repository and workflow structure plus a small set of status, dependency, testing, and client patterns. External resource semantics should remain project-owned. A rich reference implementation is more useful to a new coding agent than an empty scaffold, provided the example domain and reusable foundation are clearly distinguished.

## Decisions informed

- Keep Operator Foundry as a public reference repository and GitHub template.
- Include concrete example code and patterns without requiring every new operator to retain them.
- Make the kickoff workflow require explicit review of lifecycle and ownership patterns.
- Prefer copied, self-contained patterns over an initial shared Go runtime library or large code generator.
