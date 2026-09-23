# Lifecycle and ownership

External resources commonly need more than create and update. The reference patterns demonstrate:

- explicit creation versus adoption;
- stable external identity in status;
- optional deletion through a finalizer;
- releasing a deletion finalizer when a required connection or credential is confirmed missing;
- safe orphaning when Kubernetes deletion should not remove the external object;
- remote deletion and recreation;
- ownership checks before mutating an existing object.
- separate claims for relationships when one controller must own one edge rather than a parent's whole relationship list.

These are strong defaults to investigate, not mandatory features. A new operator should use them when the external system supports the semantics and the product goal benefits from them. If it does not, document the reason and preserve a design that can evolve if the requirement appears later.

When a `Delete` finalizer cannot reach the external system because its Kubernetes connection or credential Secret was deleted, retrying forever can block namespace cleanup. The reference reconciler releases the finalizer only for a confirmed `NotFound` dependency and logs that the external object may be orphaned. Transient API errors continue to hold the finalizer so cleanup can retry. Document the resulting administrative cleanup path for the external system.
