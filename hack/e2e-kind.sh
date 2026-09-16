#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
KIND_CLUSTER=${KIND_CLUSTER:-operator-foundry}
KIND_CNI=${KIND_CNI:-default}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-operator-foundry-system}
E2E_TEST_NAMESPACE=${E2E_TEST_NAMESPACE:-operator-foundry-e2e}
context="kind-${KIND_CLUSTER}"

kubectl_cmd() {
  "${KUBECTL}" --context="${context}" "$@"
}

kubectl_cmd create namespace "${E2E_TEST_NAMESPACE}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
kubectl_cmd -n "${OPERATOR_NAMESPACE}" rollout status deployment -l app.kubernetes.io/instance=operator-foundry --timeout=5m
kubectl_cmd get crd patternconnections.patterns.operator-foundry.example >/dev/null
kubectl_cmd get crd patternresources.patterns.operator-foundry.example >/dev/null

cat <<'EOF' | kubectl_cmd -n "${E2E_TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternConnection
metadata:
  name: missing-credentials
spec:
  endpoint: https://api.example.invalid
  authSecretRef:
    name: missing-token
EOF

for _ in {1..60}; do
  condition="$(kubectl_cmd -n "${E2E_TEST_NAMESPACE}" get patternconnection/missing-credentials -o jsonpath='{.status.conditions[?(@.type=="Ready")].reason}' 2>/dev/null || true)"
  if [[ "${condition}" == "DependencyNotReady" || "${condition}" == "InvalidConfiguration" ]]; then
    break
  fi
  sleep 2
done

observed="$(kubectl_cmd -n "${E2E_TEST_NAMESPACE}" get patternconnection/missing-credentials -o jsonpath='{.status.conditions[?(@.type=="Ready")].reason}' 2>/dev/null || true)"
[[ "${observed}" == "DependencyNotReady" || "${observed}" == "InvalidConfiguration" ]] || {
  kubectl_cmd -n "${E2E_TEST_NAMESPACE}" get patternconnection/missing-credentials -o yaml >&2 || true
  echo "PatternConnection did not reconcile to an observable dependency/configuration state" >&2
  exit 1
}

if [[ "${KIND_CNI}" == cilium ]]; then
  kubectl_cmd -n kube-system rollout status daemonset/cilium --timeout=10m
  kubectl_cmd -n kube-system rollout status deployment/cilium-operator --timeout=10m
  kubectl_cmd -n kube-system rollout status deployment/hubble-relay --timeout=10m
  kubectl_cmd -n kube-system get service/hubble-relay >/dev/null
  kubectl_cmd apply -f config/network-policy/allow-metrics.yaml >/dev/null
fi

echo "Kind E2E passed with KIND_CNI=${KIND_CNI}"
