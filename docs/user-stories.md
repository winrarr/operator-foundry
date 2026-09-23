# User stories

These stories describe the people and workflows the foundation should support. They are intentionally broader than the first implementation slice.

## Current stories

- As a project owner, I want to start a new operator from a consistent repository foundation so repeated setup decisions are already made.
- As a coding agent, I want a single kickoff document that tells me how to understand the goal, research the external system, inspect comparable projects, choose patterns, and proceed independently.
- As a coding agent, I want concrete API, client, reconciler, testing, documentation, and workflow examples that I can adapt without treating them as universal requirements.
- As a maintainer, I want generated artifacts, local commands, CI, and documentation validation to form one coherent verification workflow.
- As an operator author, I want lifecycle, status, dependency, authentication, and ownership patterns available for deliberate reuse.
- As an operator author, I want an example that gives one Kubernetes resource ownership of one external relationship edge.
- As a platform operator, I want to restrict a manager and tenant authors to selected namespaces and tenant-safe resource kinds.
- As an operator author, I want fast unit, HTTP contract, and reconciliation tests plus a disposable Kind E2E path for behavior that requires a real cluster.
- As an operator author, I want to switch between Kind's default CNI and an optional Cilium/Hubble setup when documenting or testing NetworkPolicy behavior.
- As an operator author, I want to apply temporary default-deny rules and use Hubble's dropped flows to derive the smallest native and Cilium network policies that permit the operator's representative workflows.
- As a release maintainer, I want an optional dependency-only patch-train pattern that can be reviewed and enabled when the repository's release policy is ready.

## Future stories

- As an operator author, I want to compare several reference patterns for the same concern before choosing an API or reconciliation design.
- As a maintainer of several operators, I want improvements to the foundation to be easy to evaluate and selectively carry into existing repositories.
- As a project owner, I want the foundation to support both small operators and ambitious multi-resource operators without forcing irrelevant behavior into every project.
