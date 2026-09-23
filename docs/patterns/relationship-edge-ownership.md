# Relationship edge ownership

Some external resources contain relationships such as group membership, project assignment, or parent-child links. If separate Kubernetes objects or teams can manage different relationships, give each claim its own Kubernetes resource.

`PatternMembership` claims one edge between two `PatternResource` objects:

```yaml
apiVersion: patterns.operator-foundry.example/v1alpha1
kind: PatternMembership
metadata:
  name: application-readers
  namespace: platform
spec:
  connectionRef:
    name: external-api
  parentRef:
    name: application
  memberRef:
    name: readers
```

Both referenced resources must be Ready and use the same connection. The controller stores their stable external IDs in status and ensures exactly that edge. Deleting the `PatternMembership` removes its edge; it does not replace the parent's entire relationship list or delete either referenced resource.

The references are immutable. Delete and recreate the claim to change either endpoint. Set `driftDetectionInterval` when external actors can remove the edge without a Kubernetes event; the next observation restores it. If the connection or credential Secret disappears during deletion, the finalizer is released with a warning that the external edge may need administrative cleanup.

Use this pattern when the external API can mutate one relationship independently. If it only accepts full-list replacement, the controller must first define how it preserves edges owned by other actors before adopting that API.
