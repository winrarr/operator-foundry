# Kind and network-policy validation

The default local Kind workflow uses Kind's bundled CNI for speed and broad
compatibility. CNI selection is not part of operator correctness: the
operator's installation and reconciliation checks should work the same way in
both modes.

Cilium is an optional local investigation harness. Its purpose here is to
provide Hubble so an operator author can observe denied traffic while building
the smallest useful set of network policies. Hubble observes and explains
flows; it does not generate a correct policy automatically.

## Start the investigation cluster

The local workflow needs Docker, Kind, kubectl, Helm, and the Hubble CLI. The
repository pins a project-local Hubble CLI; install it while checking the
cluster with `make kind-hubble-check`. Create the cluster and install the
operator from committed assets:

```sh
make kind-down || true
KIND_CNI=cilium make kind-deploy-e2e
KIND_CNI=cilium make kind-hubble-check
```

`kind-deploy-e2e` is useful for policy work because it builds the local images,
starts the operator, installs the committed CRDs and chart, and does not spend
time regenerating documentation or manifests. Keep the cluster while
iterating and delete it with `make kind-down` when finished.

## Observe dropped flows with Hubble

Start a flow stream in one terminal. `--port-forward` asks the Hubble CLI to
forward the relay port from the selected Kind cluster:

```sh
bin/hubble observe \
  --kube-context kind-operator-foundry \
  --port-forward \
  --namespace operator-foundry-system \
  --verdict DROPPED \
  --drop-reason-desc POLICY_DENIED \
  --print-policy-names \
  --follow
```

The repository-pinned CLI is installed by `make kind-hubble-check`.

Use `--since 30s` or `--last 100` for a bounded query instead of `--follow`
when investigating one reconciliation attempt. Add `--output json` when the
source and destination labels, ports, identities, or drop reason need to be
recorded precisely.

In another terminal, apply a strict temporary default-deny policy for the
operator namespace, then apply or update the custom resources that represent
the operator's real workflows. The exact policy scope belongs to the new
operator; this example shows the shape of a disposable all-direction probe:

```sh
kubectl --context kind-operator-foundry apply -f - <<'EOF'
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: temporary-default-deny
  namespace: operator-foundry-system
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
EOF
```

The first denied flows are expected. They are evidence to investigate, not
proof that every observed destination should be allowed. For each relevant
drop:

1. identify the source workload, destination workload or entity, protocol,
   port, direction, and drop reason in Hubble;
2. determine whether the traffic is required for Kubernetes API access, DNS,
   the external system, metrics, probes, leader election, or another
   operator-specific operation;
3. add the narrowest rule that expresses that requirement;
4. repeat the same reconciliation or resource lifecycle operation and query
   Hubble again.

Native `NetworkPolicy` is usually the clearest choice for namespace, pod, and
port boundaries. A `CiliumNetworkPolicy` is useful when the requirement needs
Cilium identities or entities, FQDN-aware egress, or other Cilium-specific
selectors. Keep the policy source and the reason for every exception beside
the operator's deployment configuration.

For example, an operator that needs Kubernetes API and cluster DNS access can
express those Cilium-specific destinations like this. Replace the endpoint
labels and external destination with the labels and names observed in the
target cluster; do not copy them as universal values:

```yaml
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: operator-egress
  namespace: operator-foundry-system
spec:
  endpointSelector:
    matchLabels:
      app.kubernetes.io/name: operator-foundry
      app.kubernetes.io/instance: operator-foundry
  egress:
  - toEntities:
    - kube-apiserver
    toPorts:
    - ports:
      - port: "6443"
        protocol: TCP
  - toEndpoints:
    - matchLabels:
        k8s:io.kubernetes.pod.namespace: kube-system
        k8s:k8s-app: kube-dns
    toPorts:
    - ports:
      - port: "53"
        protocol: UDP
      - port: "53"
        protocol: TCP
```

The API-server entity, DNS rule, internal destination rules, and external
system rules are independent candidates. Add only the rules required by the
observed workflows, and keep native and Cilium policies from overlapping in a
way that makes the effective policy difficult to explain.

Continue until the representative create, update, drift, dependency, and
deletion workflows produce no unexpected `POLICY_DENIED` flows. Treat that as
a candidate minimum policy set, then verify it again from a fresh cluster and
with the failure and recovery paths. Hubble's absence of a drop only proves
the observed scenario; it does not prove that an untested operation is
permitted.

## Reference boundary check

The repository also includes a small native-policy boundary check in the
golden Kind script. With the optional Cilium cluster, it allows manager
metrics and confirms that the health port remains denied:

```sh
KIND_CNI=cilium make kind-e2e
```

That command is a regression example, not a replacement for the iterative
Hubble investigation above. The Cilium path is intentionally local and is not
run by the GitHub Actions workflow; the operator must remain CNI-independent.

Always delete the cluster before switching between `default` and `cilium`; the
CNI is selected during cluster creation.
