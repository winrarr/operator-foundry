#!/usr/bin/env bash
set -euo pipefail

KUBECTL=${KUBECTL:-kubectl}
KIND_CLUSTER=${KIND_CLUSTER:-operator-foundry}
KIND_CNI=${KIND_CNI:-default}
PROJECT_NAME=${PROJECT_NAME:-operator-foundry}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-operator-foundry-system}
E2E_TEST_NAMESPACE=${E2E_TEST_NAMESPACE:-operator-foundry-e2e}
MOCK_API_IMG=${MOCK_API_IMG:-ghcr.io/winrarr/operator-foundry-mock-api:dev}
CURL_TEST_IMAGE=${CURL_TEST_IMAGE:-curlimages/curl:8.22.0@sha256:58adaa4e8dca9c988bae2aba4ab3434a0bb2da16bbe3f92dec39ec7785166777}
context="kind-${KIND_CLUSTER}"
mock_api_url="http://127.0.0.1:18080"
port_forward_pid=""
port_forward_log="/tmp/operator-foundry-e2e-port-forward.$$.log"

kubectl_cmd() {
  "${KUBECTL}" --context="${context}" "$@"
}

cleanup() {
  if [[ -n "${port_forward_pid}" ]]; then
    kill "${port_forward_pid}" >/dev/null 2>&1 || true
    wait "${port_forward_pid}" >/dev/null 2>&1 || true
  fi
  rm -f "${port_forward_log}"
}
trap cleanup EXIT

fail_with_state() {
  echo "$1" >&2
  kubectl_cmd -n "${E2E_TEST_NAMESPACE}" get patternconnections,patternresources,patternmemberships,pods -o wide >&2 || true
  kubectl_cmd -n "${E2E_TEST_NAMESPACE}" get patternconnection,patternresource,patternmembership -o yaml >&2 || true
  if [[ -f "${port_forward_log}" ]]; then
    cat "${port_forward_log}" >&2
  fi
  exit 1
}

require_commands() {
  command -v curl >/dev/null || fail_with_state "curl is required for the mock external API checks"
  command -v jq >/dev/null || fail_with_state "jq is required for the mock external API checks"
}

wait_for_ready() {
  local resource_type=$1
  local resource_name=$2
  local namespace=$3
  local observed
  for _ in {1..90}; do
    observed="$(kubectl_cmd -n "${namespace}" get "${resource_type}/${resource_name}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || true)"
    if [[ "${observed}" == "True" ]]; then
      return 0
    fi
    sleep 1
  done
  fail_with_state "${resource_type}/${resource_name} did not become Ready"
}

wait_for_reason() {
  local resource_type=$1
  local resource_name=$2
  local namespace=$3
  local expected=$4
  local observed
  for _ in {1..90}; do
    observed="$(kubectl_cmd -n "${namespace}" get "${resource_type}/${resource_name}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].reason}' 2>/dev/null || true)"
    if [[ "${observed}" == "${expected}" ]]; then
      return 0
    fi
    sleep 1
  done
  fail_with_state "${resource_type}/${resource_name} did not report reason ${expected}; observed ${observed}"
}

wait_for_status_value() {
  local resource_type=$1
  local resource_name=$2
  local namespace=$3
  local jsonpath=$4
  local expected=$5
  local observed
  for _ in {1..90}; do
    observed="$(kubectl_cmd -n "${namespace}" get "${resource_type}/${resource_name}" -o jsonpath="${jsonpath}" 2>/dev/null || true)"
    if [[ "${observed}" == "${expected}" ]]; then
      return 0
    fi
    sleep 1
  done
  fail_with_state "${resource_type}/${resource_name} did not report ${jsonpath}=${expected}; observed ${observed}"
}

wait_for_absent() {
  local resource_type=$1
  local resource_name=$2
  local namespace=$3
  for _ in {1..90}; do
    if ! kubectl_cmd -n "${namespace}" get "${resource_type}/${resource_name}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  fail_with_state "${resource_type}/${resource_name} was not deleted"
}

admin_curl() {
  curl --fail --silent --show-error --retry 15 --retry-delay 1 --retry-connrefused "$@"
}

stats_create_count() {
  admin_curl "${mock_api_url}/admin/stats" | jq -r '.create'
}

stats_get_count() {
  admin_curl "${mock_api_url}/admin/stats" | jq -r '.get'
}

wait_for_mock_api() {
  local status
  for _ in {1..60}; do
    status="$(curl --silent --show-error --max-time 3 --output /dev/null --write-out '%{http_code}' "${mock_api_url}/health" 2>/dev/null || true)"
    if [[ "${status}" == "204" ]]; then
      return 0
    fi
    sleep 1
  done
  fail_with_state "mock API port-forward did not return the expected health response"
}

membership_exists() {
  admin_curl --get \
    --data-urlencode "parentID=$1" \
    --data-urlencode "memberID=$2" \
    "${mock_api_url}/admin/memberships" >/dev/null
}

wait_for_get_count_after() {
  local previous=$1
  local observed
  for _ in {1..60}; do
    observed="$(stats_get_count)"
    if (( observed > previous )); then
      return 0
    fi
    sleep 1
  done
  fail_with_state "mock API did not observe the expected reconciliation read"
}

require_commands
kubectl_cmd create namespace "${E2E_TEST_NAMESPACE}" --dry-run=client -o yaml | kubectl_cmd apply -f - >/dev/null
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" delete patternresources --all --ignore-not-found --wait=true --timeout=2m >/dev/null
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" delete patternconnections --all --ignore-not-found --wait=true --timeout=2m >/dev/null
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" delete secret example-token --ignore-not-found >/dev/null
kubectl_cmd -n "${OPERATOR_NAMESPACE}" rollout status deployment -l "app.kubernetes.io/instance=${PROJECT_NAME}" --timeout=5m
kubectl_cmd get crd patternconnections.patterns.operator-foundry.example >/dev/null
kubectl_cmd get crd patternresources.patterns.operator-foundry.example >/dev/null

sed -e "s#ghcr.io/winrarr/operator-foundry-mock-api:dev#${MOCK_API_IMG}#g" -e "s#operator-foundry-e2e#${E2E_TEST_NAMESPACE}#g" config/e2e/mock-external-api.yaml | kubectl_cmd apply -f - >/dev/null
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" rollout restart deployment/mock-external-api
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" rollout status deployment/mock-external-api --timeout=5m

mock_api_pod="$(kubectl_cmd -n "${E2E_TEST_NAMESPACE}" get pod -l app.kubernetes.io/name=mock-external-api --sort-by=.metadata.creationTimestamp -o name | tail -n 1)"
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" wait --for=condition=Ready "${mock_api_pod}" --timeout=2m
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" port-forward "${mock_api_pod}" 18080:8080 >"${port_forward_log}" 2>&1 &
port_forward_pid=$!
wait_for_mock_api

cat <<EOF | kubectl_cmd -n "${E2E_TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternConnection
metadata:
  name: missing-credentials
spec:
  endpoint: http://mock-external-api.${E2E_TEST_NAMESPACE}.svc.cluster.local:8080
  authSecretRef:
    name: missing-token
EOF
wait_for_reason patternconnection missing-credentials "${E2E_TEST_NAMESPACE}" DependencyNotReady

cat <<EOF | kubectl_cmd -n "${E2E_TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: v1
kind: Secret
metadata:
  name: example-token
type: Opaque
stringData:
  token: test-token
---
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternConnection
metadata:
  name: example
spec:
  endpoint: http://mock-external-api.${E2E_TEST_NAMESPACE}.svc.cluster.local:8080
  authSecretRef:
    name: example-token
EOF
wait_for_ready patternconnection example "${E2E_TEST_NAMESPACE}"
wait_for_status_value patternconnection example "${E2E_TEST_NAMESPACE}" '{.status.observedGeneration}' 1

cat <<'EOF' | kubectl_cmd -n "${E2E_TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternResource
metadata:
  name: example
  annotations:
    e2e.operator-foundry.example/reconcile: initial
spec:
  connectionRef:
    name: example
  value: desired
  creationPolicy: Create
  deletionPolicy: Orphan
  driftDetectionInterval: 5s
EOF
wait_for_ready patternresource example "${E2E_TEST_NAMESPACE}"
wait_for_status_value patternresource example "${E2E_TEST_NAMESPACE}" '{.status.observedValue}' desired
example_create_count="$(stats_create_count)"
[[ "${example_create_count}" == 1 ]] || fail_with_state "expected one external create for example, got ${example_create_count}"

example_get_count="$(stats_get_count)"
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" annotate patternresource/example e2e.operator-foundry.example/reconcile=second --overwrite >/dev/null
wait_for_get_count_after "${example_get_count}"
[[ "$(stats_create_count)" == "${example_create_count}" ]] || fail_with_state "idempotent reconciliation issued an unexpected external create"

kubectl_cmd -n "${E2E_TEST_NAMESPACE}" patch patternresource/example --type=merge -p '{"spec":{"value":"updated"}}' >/dev/null
wait_for_status_value patternresource example "${E2E_TEST_NAMESPACE}" '{.status.observedValue}' updated
example_value="$(admin_curl "${mock_api_url}/admin/resources/example" | jq -r '.value')"
[[ "${example_value}" == updated ]] || fail_with_state "external update did not persist; got ${example_value}"

admin_curl -X POST "${mock_api_url}/admin/fail-next" -H 'Content-Type: application/json' -d '{"count":10,"status":503}' >/dev/null
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" annotate patternresource/example e2e.operator-foundry.example/reconcile=transient-failure --overwrite >/dev/null
wait_for_reason patternresource example "${E2E_TEST_NAMESPACE}" ExternalRequestFailed
admin_curl -X POST "${mock_api_url}/admin/clear-failures" >/dev/null
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" annotate patternresource/example e2e.operator-foundry.example/reconcile=recovery --overwrite >/dev/null
wait_for_ready patternresource example "${E2E_TEST_NAMESPACE}"

admin_curl -X DELETE "${mock_api_url}/admin/resources/example" >/dev/null
example_create_count_before_recreation="$(stats_create_count)"
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" annotate patternresource/example e2e.operator-foundry.example/reconcile=remote-deletion --overwrite >/dev/null
for _ in {1..90}; do
  if (( $(stats_create_count) > example_create_count_before_recreation )); then
    break
  fi
  sleep 1
done
[[ "$(stats_create_count)" -gt "${example_create_count_before_recreation}" ]] || fail_with_state "remote deletion was not recreated"
wait_for_status_value patternresource example "${E2E_TEST_NAMESPACE}" '{.status.observedValue}' updated

admin_curl -X POST "${mock_api_url}/admin/resources/adopted" -H 'Content-Type: application/json' -d '{"id":"adopted-1","value":"adopted"}' >/dev/null
adoption_create_count="$(stats_create_count)"
cat <<'EOF' | kubectl_cmd -n "${E2E_TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternResource
metadata:
  name: adopted
spec:
  connectionRef:
    name: example
  value: adopted
  creationPolicy: Adopt
  deletionPolicy: Orphan
EOF
wait_for_ready patternresource adopted "${E2E_TEST_NAMESPACE}"
wait_for_status_value patternresource adopted "${E2E_TEST_NAMESPACE}" '{.status.id}' adopted-1
[[ "$(stats_create_count)" == "${adoption_create_count}" ]] || fail_with_state "adoption issued an unexpected external create"

cat <<'EOF' | kubectl_cmd -n "${E2E_TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternResource
metadata:
  name: managed-delete
spec:
  connectionRef:
    name: example
  value: managed
  creationPolicy: Create
  deletionPolicy: Delete
EOF
wait_for_ready patternresource managed-delete "${E2E_TEST_NAMESPACE}"
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" delete patternresource/managed-delete --wait=false >/dev/null
wait_for_absent patternresource managed-delete "${E2E_TEST_NAMESPACE}"
if admin_curl "${mock_api_url}/admin/resources/managed-delete" >/dev/null 2>&1; then
  fail_with_state "Delete policy left the external resource behind"
fi

cat <<'EOF' | kubectl_cmd -n "${E2E_TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternResource
metadata:
  name: orphan
spec:
  connectionRef:
    name: example
  value: retained
  creationPolicy: Create
  deletionPolicy: Orphan
EOF
wait_for_ready patternresource orphan "${E2E_TEST_NAMESPACE}"
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" delete patternresource/orphan --wait=false >/dev/null
wait_for_absent patternresource orphan "${E2E_TEST_NAMESPACE}"
orphan_value="$(admin_curl "${mock_api_url}/admin/resources/orphan" | jq -r '.value')"
[[ "${orphan_value}" == retained ]] || fail_with_state "Orphan policy did not retain the external resource"

cat <<'EOF' | kubectl_cmd -n "${E2E_TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternResource
metadata:
  name: membership-parent
spec:
  connectionRef:
    name: example
  value: parent
  creationPolicy: Create
  deletionPolicy: Orphan
---
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternResource
metadata:
  name: membership-member
spec:
  connectionRef:
    name: example
  value: member
  creationPolicy: Create
  deletionPolicy: Orphan
---
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternResource
metadata:
  name: membership-other-member
spec:
  connectionRef:
    name: example
  value: other-member
  creationPolicy: Create
  deletionPolicy: Orphan
EOF
wait_for_ready patternresource membership-parent "${E2E_TEST_NAMESPACE}"
wait_for_ready patternresource membership-member "${E2E_TEST_NAMESPACE}"
wait_for_ready patternresource membership-other-member "${E2E_TEST_NAMESPACE}"
parent_id="$(kubectl_cmd -n "${E2E_TEST_NAMESPACE}" get patternresource/membership-parent -o jsonpath='{.status.id}')"
member_id="$(kubectl_cmd -n "${E2E_TEST_NAMESPACE}" get patternresource/membership-member -o jsonpath='{.status.id}')"
other_member_id="$(kubectl_cmd -n "${E2E_TEST_NAMESPACE}" get patternresource/membership-other-member -o jsonpath='{.status.id}')"
cat <<'EOF' | kubectl_cmd -n "${E2E_TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternMembership
metadata:
  name: managed-edge
spec:
  connectionRef:
    name: example
  parentRef:
    name: membership-parent
  memberRef:
    name: membership-member
---
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternMembership
metadata:
  name: independent-edge
spec:
  connectionRef:
    name: example
  parentRef:
    name: membership-parent
  memberRef:
    name: membership-other-member
EOF
wait_for_ready patternmembership managed-edge "${E2E_TEST_NAMESPACE}"
wait_for_ready patternmembership independent-edge "${E2E_TEST_NAMESPACE}"
membership_exists "${parent_id}" "${member_id}" || fail_with_state "managed relationship edge was not created"
membership_exists "${parent_id}" "${other_member_id}" || fail_with_state "independent relationship edge was not created"
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" delete patternmembership/managed-edge --wait=false >/dev/null
wait_for_absent patternmembership managed-edge "${E2E_TEST_NAMESPACE}"
if ! membership_exists "${parent_id}" "${other_member_id}"; then
  fail_with_state "deleting one membership claim removed an unrelated edge"
fi
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" delete patternmembership/independent-edge --wait=false >/dev/null
wait_for_absent patternmembership independent-edge "${E2E_TEST_NAMESPACE}"

kubectl_cmd -n "${E2E_TEST_NAMESPACE}" create secret generic temporary-token --from-literal=token=test-token >/dev/null
cat <<EOF | kubectl_cmd -n "${E2E_TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternConnection
metadata:
  name: temporary
spec:
  endpoint: http://mock-external-api.${E2E_TEST_NAMESPACE}.svc.cluster.local:8080
  authSecretRef:
    name: temporary-token
---
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternResource
metadata:
  name: dependency-lost
spec:
  connectionRef:
    name: temporary
  value: orphaned-on-dependency-loss
  creationPolicy: Create
  deletionPolicy: Delete
EOF
wait_for_ready patternconnection temporary "${E2E_TEST_NAMESPACE}"
wait_for_ready patternresource dependency-lost "${E2E_TEST_NAMESPACE}"
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" delete patternconnection/temporary --wait=true >/dev/null
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" delete patternresource/dependency-lost --wait=false >/dev/null
wait_for_absent patternresource dependency-lost "${E2E_TEST_NAMESPACE}"
orphaned_value="$(admin_curl "${mock_api_url}/admin/resources/dependency-lost" | jq -r '.value')"
[[ "${orphaned_value}" == orphaned-on-dependency-loss ]] || fail_with_state "dependency-loss deletion did not expose the external orphan"
admin_curl -X DELETE "${mock_api_url}/admin/resources/dependency-lost" >/dev/null
kubectl_cmd -n "${E2E_TEST_NAMESPACE}" delete secret temporary-token --ignore-not-found >/dev/null

if [[ "${KIND_CNI}" == cilium ]]; then
  kubectl_cmd -n kube-system rollout status daemonset/cilium --timeout=10m
  kubectl_cmd -n kube-system rollout status deployment/cilium-operator --timeout=10m
  kubectl_cmd -n kube-system rollout status deployment/hubble-relay --timeout=10m
  kubectl_cmd -n kube-system get service/hubble-relay >/dev/null
  kubectl_cmd apply -f config/network-policy/default-deny-ingress.yaml >/dev/null
  kubectl_cmd apply -f config/network-policy/allow-metrics.yaml >/dev/null
  kubectl_cmd -n "${E2E_TEST_NAMESPACE}" delete pod/metrics-client --ignore-not-found --wait=true >/dev/null
  cat <<EOF | kubectl_cmd -n "${E2E_TEST_NAMESPACE}" apply -f - >/dev/null
apiVersion: v1
kind: ServiceAccount
metadata:
  name: metrics-client
  namespace: ${E2E_TEST_NAMESPACE}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ${PROJECT_NAME}-e2e-metrics-reader
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ${PROJECT_NAME}-metrics-reader
subjects:
- kind: ServiceAccount
  name: metrics-client
  namespace: ${E2E_TEST_NAMESPACE}
---
apiVersion: v1
kind: Pod
metadata:
  name: metrics-client
  namespace: ${E2E_TEST_NAMESPACE}
spec:
  restartPolicy: Never
  serviceAccountName: metrics-client
  containers:
  - name: metrics-client
    image: ${CURL_TEST_IMAGE}
    imagePullPolicy: IfNotPresent
    command: [sleep, "300"]
EOF
  kubectl_cmd -n "${E2E_TEST_NAMESPACE}" wait --for=condition=Ready pod/metrics-client --timeout=2m
  manager_pod_ip="$(kubectl_cmd -n "${OPERATOR_NAMESPACE}" get pod -l "app.kubernetes.io/instance=${PROJECT_NAME}" -o jsonpath='{.items[0].status.podIP}')"
  metrics_token="$(kubectl_cmd -n "${E2E_TEST_NAMESPACE}" exec metrics-client -- cat /var/run/secrets/kubernetes.io/serviceaccount/token)"
  kubectl_cmd -n "${E2E_TEST_NAMESPACE}" exec metrics-client -- curl --fail --silent --show-error --insecure --header "Authorization: Bearer ${metrics_token}" --max-time 5 "https://${manager_pod_ip}:8443/metrics" >/dev/null
  if kubectl_cmd -n "${E2E_TEST_NAMESPACE}" exec metrics-client -- curl --fail --silent --show-error --max-time 5 "http://${manager_pod_ip}:8081/healthz" >/dev/null 2>&1; then
    fail_with_state "default-deny NetworkPolicy unexpectedly allowed health-port ingress"
  fi
  kubectl_cmd -n "${E2E_TEST_NAMESPACE}" delete pod/metrics-client --ignore-not-found --wait=true >/dev/null
fi

echo "Kind E2E passed: install, dependency, create, idempotency, update, failure recovery, remote recreation, adoption, delete/orphan, edge ownership, dependency-loss cleanup, and CNI-specific checks"
