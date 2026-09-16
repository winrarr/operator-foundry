#!/usr/bin/env bash
set -euo pipefail

chart_dir="charts/operator-foundry"
mkdir -p "${chart_dir}/crds"
rm -f "${chart_dir}/crds"/*.yaml
cp config/crd/bases/*.yaml "${chart_dir}/crds/"

{
  cat <<'EOF'
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ include "operator-foundry.fullname" . }}-manager
  labels:
{{ include "operator-foundry.labels" . | nindent 4 }}
EOF
  sed -n '/^rules:$/,$p' config/rbac/role.yaml
} > "${chart_dir}/templates/clusterrole.yaml"
