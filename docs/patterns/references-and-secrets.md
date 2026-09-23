# References and secrets

References should have a clear namespace and trust boundary. Same-namespace references are a useful safe default for namespaced resources; cross-namespace and cluster-scoped references need an explicit security and ownership rationale.

Read credentials only from Kubernetes Secrets, never place them in status or logs, and ensure tests and examples use non-sensitive values. If the operator writes credentials to a Secret, define ownership, replacement, rotation, and deletion behavior before implementing it.

## Authentication and credential lifecycle

The active `PatternConnection` uses a static bearer token from a same-namespace Secret. Treat that as one option, not a universal connection shape. Depending on the external system, a connection may instead use a projected workload identity, an exchange of a short-lived assertion for an API token, or another renewable credential flow.

For renewable authentication, define where login happens, how token expiry and renewal work, and which connection changes invalidate a cached client. A client cache should be keyed by the complete authentication and endpoint configuration. Static Secret credentials should be reread after rotation; renewable clients should refresh before expiry and reread their source credentials when logging in again. Never log returned tokens.

Test credential rotation, an expired token, failed login, and recovery. Watch referenced Secrets when that produces a useful prompt reconciliation, while retaining a bounded retry for failures that do not generate a Kubernetes event.

## External scope and tenancy

Some APIs use an organization, project, account, region, or namespace as request context. Put stable target scope on the connection when all dependent resources share it. Validate its syntax, decide whether changing it can redirect managed ownership, and attach it consistently to every request. A health check may need a different endpoint or permission from ordinary operations; connection readiness should prove access in the configured scope.

The external target scope is independent of Kubernetes watch scope. Keep the two boundaries explicit in the API, RBAC, and documentation.
