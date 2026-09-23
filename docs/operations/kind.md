# Local Kind workflow

The default local cluster uses Kind's bundled CNI:

```sh
make kind-e2e
make kind-down
```

The optional namespace-scope and tenant-RBAC scenario uses its own cluster:

```sh
make kind-scoped-e2e
KIND_CLUSTER=operator-foundry-scoped make kind-down
```

To exercise Cilium and Hubble locally, recreate the cluster with the optional
Cilium path:

```sh
KIND_CNI=cilium make kind-deploy-e2e
KIND_CNI=cilium make kind-hubble-check
make kind-down
```

The Cilium configuration disables Kind's default CNI, installs the pinned
Cilium chart, and enables Hubble Relay. A cluster must be deleted before
switching between CNI modes because the CNI is selected during cluster
creation. See [Kind and network-policy validation](../patterns/kind-and-network-policy.md)
for the Hubble dropped-flow loop used to discover policy requirements.

The E2E script builds and loads both the operator and a disposable mock external
API. It proves cluster creation, CRD installation, Helm installation,
deployment rollout, dependency handling, create/update/adopt/recreate
behavior, failure recovery, Delete and Orphan semantics, and an observable
reconciliation state, including independent relationship-edge cleanup. Add operator-specific live behavior there only when
fake-client or HTTP contract tests cannot prove it. The GitHub Actions workflow
uses the default CNI; Cilium is a local diagnostic path because the CNI should
not affect the operator.
