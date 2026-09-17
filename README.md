# Operator Foundry

Operator Foundry is a public reference repository and GitHub template for building Kubernetes operators with a strong engineering and documentation baseline.

It is intentionally opinionated about the parts that benefit from consistency: repository structure, Go and controller-runtime conventions, status reporting, testing, generated artifacts, documentation, local Kubernetes workflows, and agent-oriented project startup. It is deliberately not a universal operator framework. The reference patterns are starting points to evaluate against the new external system.

## Starting a new operator

Use GitHub's **Use this template** action to create a new repository, then give the first coding agent the operator's north-star goal. The agent should begin with [PROJECT_KICKOFF.md](PROJECT_KICKOFF.md), which guides user-story discovery, external-system research, pattern selection, first-slice planning, and implementation.

The example API and controllers in this repository are reference material. Keep them available while the new project becomes solid; adapt or remove them only when the new project's design makes that appropriate. Do not treat any example resource as a requirement.

## What this repository demonstrates

- kstatus-compatible `Ready`, `Reconciling`, and `Stalled` conditions;
- typed external API clients with explicit request and response contracts;
- connection and Secret-reference handling;
- adoption, orphaning, deletion, finalizers, and drift detection;
- dependency readiness and reconciliation retries;
- fake-client, HTTP contract, and in-cluster operational tests;
- generated CRDs, RBAC, API reference documentation, Helm, and Kustomize;
- strict documentation validation and GitHub Pages publication;
- a disposable mock external API and Kind golden-path E2E matrix.

The lifecycle patterns are intentionally demonstrated because they are useful for most external-system operators, but every new operator must validate their semantics and scope before adopting them.

The reference E2E is the confidence path for this repository. It installs the operator into Kind, drives the example resources against the disposable mock API, and checks the observable lifecycle. The optional Cilium mode additionally checks Hubble readiness and a metrics NetworkPolicy allow/deny boundary.

## Local development

```sh
make check
make build
make docs-build
make helm-lint
make helm-template
```

See [the documentation map](docs/index.md) for the durable project knowledge and [PROJECT_KICKOFF.md](PROJECT_KICKOFF.md) for the development workflow.

## Inspiration

The initial conventions were distilled from the Harbor Operator, Infisical Entity Operator, and OpenBao Entity Operator. Those projects remain useful references for concrete patterns, but external projects and this repository are evidence and inspiration—not authorities for a new operator's API or lifecycle semantics.

## License

Apache License 2.0. See [LICENSE](LICENSE).
