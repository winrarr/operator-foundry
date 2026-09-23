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

## Follow-up OpenBao commit review — 2026-09-23

The OpenBao Entity Operator history before Operator Foundry commit `2f6184e` (2026-09-17 15:09:41 +02:00) was reviewed as a source of engineering patterns. All 13 commits in that interval were inspected:

- `ef45c29` initial foundation;
- `5b90940` entity aliases;
- `3ec3c66` groups and membership;
- `2234435` cleanup dependency handling;
- `e6d87e4` external OpenBao namespace targeting;
- `06ce449` Helm packaging and release workflow;
- `063a6cc` Kubernetes Auth;
- `3f61bbf` ACL policies;
- `a59e2fc` comprehensive guides;
- `9e1dbe1` narrower live verification scope;
- `d267916` AppRole and tenancy roadmap;
- `02b0db1` namespace-scoped operator deployment; and
- `0c10d61` platform-owned tenant boundary enforcement.

### Adopted patterns

- A `Delete` finalizer must not hold Kubernetes deletion forever when a required connection or credential Secret is confirmed missing. Operator Foundry releases that finalizer with a warning that external state may be orphaned; transient external errors still retry.
- A manager can accept an optional namespace allowlist and bind its namespaced permissions only in those namespaces. Tenant author permissions are a separate unbound role that excludes platform-owned connections and Secrets. This scopes access but does not claim hostile multi-tenancy by itself.
- A relationship claim can own one external edge independently. Operator Foundry uses `PatternMembership` and does not replace a parent's whole relationship list when one claim changes.
- Authentication design should account for static credential rotation, renewable workload identities, short-lived tokens, client cache invalidation, and health checks within external tenant scope. The example keeps one simple bearer-token implementation while documenting the alternatives.

### Not adopted as generic API

OpenBao entity aliases, groups, policies, AppRole fields, Kubernetes Auth settings, and its external namespace property depend on OpenBao semantics. They informed the generic relationship, authentication, and scope patterns above, but their resource shape and token lifecycle remain examples to research rather than defaults for future operators.
