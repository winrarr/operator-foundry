# Namespace scope and tenant access

The Helm chart can restrict a manager to selected Kubernetes namespaces. This reduces the namespaces from which it reads custom resources and Secrets and limits its namespaced RBAC grants to the same allowlist.

```yaml
watchNamespaces:
  - payments
  - inventory
tenantAuthorRole:
  create: true
```

With a non-empty `watchNamespaces`, the chart configures the manager cache with `--watch-namespaces` and creates a RoleBinding to the manager ClusterRole in each selected namespace. It omits the cluster-wide manager ClusterRoleBinding. Empty `watchNamespaces` preserves the default cluster-wide behavior.

Create each selected namespace before installing the chart; the chart does not create tenant namespaces.

The chart also creates an unbound `<release-name>-tenant-author` ClusterRole. A platform administrator can bind it in a tenant namespace:

```sh
kubectl create rolebinding payments-author \
  --clusterrole=operator-foundry-tenant-author \
  --serviceaccount=payments:operator-author \
  --namespace=payments
```

The role grants access to `PatternResource` and `PatternMembership`. It does not grant access to `PatternConnection` or Secrets, which should remain platform-owned when they hold endpoint and credential configuration. Tenant authors can reference a connection by name from their own namespace without reading or changing that connection.

For Kustomize installations, set the manager's `WATCH_NAMESPACES` environment value, remove the generated manager ClusterRoleBinding, and bind the manager ClusterRole in each selected namespace. Apply the included unbound `tenant-author-role` only through per-namespace RoleBindings.

This setup scopes cache and Kubernetes permissions. It does not provide a complete hostile-tenant boundary by itself. Review all effective RoleBindings and admission policies, and restrict which resource fields tenants may author when those fields can redirect external mutations.

Run the isolated namespace and RBAC scenario with:

```sh
make kind-scoped-e2e
KIND_CLUSTER=operator-foundry-scoped make kind-down
```

The local scenario checks that the manager can read resources and Secrets in its selected namespace, cannot read them in another namespace, and that tenant principals cannot read connections or Secrets or access another tenant's resources.
