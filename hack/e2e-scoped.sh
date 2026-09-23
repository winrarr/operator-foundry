#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
KIND_CLUSTER=${KIND_CLUSTER:-operator-foundry-scoped}
PROJECT_NAME=${PROJECT_NAME:-operator-foundry}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-operator-foundry-system}
SCOPED_NAMESPACE=${SCOPED_NAMESPACE:-operator-foundry-scope-a}
OUT_OF_SCOPE_NAMESPACE=${OUT_OF_SCOPE_NAMESPACE:-operator-foundry-scope-b}
context="kind-${KIND_CLUSTER}"
tenant_author_role="${PROJECT_NAME}-tenant-author"

kubectl_cmd() {
  "${KUBECTL}" --context="${context}" "$@"
}

fail() {
  echo "$1" >&2
  kubectl_cmd get namespaces >&2 || true
  kubectl_cmd -n "${OPERATOR_NAMESPACE}" get deployment,pods -o wide >&2 || true
  exit 1
}

assert_can() {
  local principal=$1 namespace=$2 verb=$3 resource=$4 observed
  observed="$(kubectl_cmd auth can-i --as="${principal}" -n "${namespace}" "${verb}" "${resource}" || true)"
  [[ "${observed}" == yes ]] || fail "${principal} cannot ${verb} ${resource} in ${namespace}"
}

assert_cannot() {
  local principal=$1 namespace=$2 verb=$3 resource=$4 observed
  observed="$(kubectl_cmd auth can-i --as="${principal}" -n "${namespace}" "${verb}" "${resource}" || true)"
  [[ "${observed}" != yes ]] || fail "${principal} unexpectedly can ${verb} ${resource} in ${namespace}"
}

ensure_namespace() {
  local namespace=$1
  kubectl_cmd create namespace "${namespace}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
}

ensure_tenant_author() {
  local namespace=$1 service_account=$2
  kubectl_cmd -n "${namespace}" create serviceaccount "${service_account}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
  kubectl_cmd -n "${namespace}" create rolebinding "${service_account}-author" \
    --clusterrole="${tenant_author_role}" \
    --serviceaccount="${namespace}:${service_account}" \
    --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
}

echo "Checking namespace-scoped manager and tenant permissions"
ensure_namespace "${SCOPED_NAMESPACE}"
ensure_namespace "${OUT_OF_SCOPE_NAMESPACE}"
kubectl_cmd -n "${OPERATOR_NAMESPACE}" rollout status "deployment/${PROJECT_NAME}" --timeout=5m

manager_service_account="$(kubectl_cmd -n "${OPERATOR_NAMESPACE}" get "deployment/${PROJECT_NAME}" -o jsonpath='{.spec.template.spec.serviceAccountName}')"
manager_principal="system:serviceaccount:${OPERATOR_NAMESPACE}:${manager_service_account}"
assert_can "${manager_principal}" "${SCOPED_NAMESPACE}" list patternresources
assert_can "${manager_principal}" "${SCOPED_NAMESPACE}" list patternconnections
assert_can "${manager_principal}" "${SCOPED_NAMESPACE}" get secrets
assert_cannot "${manager_principal}" "${OUT_OF_SCOPE_NAMESPACE}" list patternresources
assert_cannot "${manager_principal}" "${OUT_OF_SCOPE_NAMESPACE}" list patternconnections
assert_cannot "${manager_principal}" "${OUT_OF_SCOPE_NAMESPACE}" get secrets

if kubectl_cmd get clusterrolebinding "${PROJECT_NAME}-manager" >/dev/null 2>&1; then
  fail "scoped chart unexpectedly created the cluster-wide manager ClusterRoleBinding"
fi
kubectl_cmd -n "${SCOPED_NAMESPACE}" get rolebinding "${PROJECT_NAME}-manager" >/dev/null

ensure_tenant_author "${SCOPED_NAMESPACE}" tenant-author
ensure_tenant_author "${OUT_OF_SCOPE_NAMESPACE}" other-tenant-author
tenant_principal="system:serviceaccount:${SCOPED_NAMESPACE}:tenant-author"
other_tenant_principal="system:serviceaccount:${OUT_OF_SCOPE_NAMESPACE}:other-tenant-author"
assert_can "${tenant_principal}" "${SCOPED_NAMESPACE}" create patternresources
assert_can "${tenant_principal}" "${SCOPED_NAMESPACE}" create patternmemberships
assert_cannot "${tenant_principal}" "${SCOPED_NAMESPACE}" get patternconnections
assert_cannot "${tenant_principal}" "${SCOPED_NAMESPACE}" create patternconnections
assert_cannot "${tenant_principal}" "${SCOPED_NAMESPACE}" get secrets
assert_cannot "${tenant_principal}" "${OUT_OF_SCOPE_NAMESPACE}" get patternresources
assert_cannot "${tenant_principal}" "${OUT_OF_SCOPE_NAMESPACE}" create patternresources
assert_cannot "${other_tenant_principal}" "${SCOPED_NAMESPACE}" get patternresources

cat <<EOF | kubectl_cmd -n "${OUT_OF_SCOPE_NAMESPACE}" apply -f - >/dev/null
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternConnection
metadata:
  name: outside-manager-scope
spec:
  endpoint: https://external.example.test
  authSecretRef:
    name: missing-token
EOF
sleep 5
conditions="$(kubectl_cmd -n "${OUT_OF_SCOPE_NAMESPACE}" get patternconnection/outside-manager-scope -o jsonpath='{.status.conditions[*].type}' 2>/dev/null || true)"
[[ -z "${conditions}" ]] || fail "manager reconciled an out-of-scope PatternConnection: ${conditions}"

echo "Scoped namespace and tenant RBAC scenarios passed"
