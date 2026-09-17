# Product direction

Operator Foundry provides a strong starting point for building Kubernetes operators for external systems. It combines reusable implementation conventions, a reference implementation, operational workflows, documentation structure, and a project-kickoff process that helps coding agents make sound early decisions.

The repository should make a new operator easy to start while preserving room for ambitious full-product goals. The reference patterns are intentionally concrete so that an agent can copy techniques, compare alternatives, and adapt them to the semantics of the new external system.

The immediate confidence goal is that the reference examples work together as a complete operator: the controller, typed client, lifecycle behavior, installation assets, and documentation should be exercised by unit, contract, operational, and Kind tests. The golden path should remain small enough to understand and strong enough to expose broken conventions before they are copied into a new operator.
