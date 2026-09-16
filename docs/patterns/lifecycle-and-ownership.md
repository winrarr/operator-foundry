# Lifecycle and ownership

External resources commonly need more than create and update. The reference patterns demonstrate:

- explicit creation versus adoption;
- stable external identity in status;
- optional deletion through a finalizer;
- safe orphaning when Kubernetes deletion should not remove the external object;
- remote deletion and recreation;
- ownership checks before mutating an existing object.

These are strong defaults to investigate, not mandatory features. A new operator should use them when the external system supports the semantics and the product goal benefits from them. If it does not, document the reason and preserve a design that can evolve if the requirement appears later.
