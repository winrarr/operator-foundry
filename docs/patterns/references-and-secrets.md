# References and secrets

References should have a clear namespace and trust boundary. Same-namespace references are a useful safe default for namespaced resources; cross-namespace and cluster-scoped references need an explicit security and ownership rationale.

Read credentials only from Kubernetes Secrets, never place them in status or logs, and ensure tests and examples use non-sensitive values. If the operator writes credentials to a Secret, define ownership, replacement, rotation, and deletion behavior before implementing it.
