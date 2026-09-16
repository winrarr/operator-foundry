# Local Kind workflow

The default local cluster uses Kind's bundled CNI:

```sh
make kind-e2e
make kind-down
```

To exercise Cilium and Hubble, recreate the cluster with the optional Cilium path:

```sh
KIND_CNI=cilium make kind-e2e
KIND_CNI=cilium make kind-hubble-check
make kind-down
```

The Cilium configuration disables Kind's default CNI, installs the pinned Cilium chart, and enables Hubble Relay. A cluster must be deleted before switching between CNI modes because the CNI is selected during cluster creation.

The E2E script proves cluster creation, CRD installation, image loading, Helm installation, deployment rollout, and an observable reconciliation state. Add operator-specific live behavior there only when fake-client or HTTP contract tests cannot prove it. For NetworkPolicy work, pair the policy with traffic assertions and Hubble observations; component readiness alone is not a policy test.
