# External clients

Keep the external client intentionally small and typed around the operations the reconciler needs. Do not generate a complete client for an external OpenAPI document unless the project has a demonstrated need for it.

The client should own request construction, authentication headers, timeouts, response decoding, status-code handling, and safe error messages. Controllers should own Kubernetes references, lifecycle policy, status, retry classification, and dependency behavior.

Contract tests should use an HTTP test server or the external system's supported test surface. They should assert methods, paths, request bodies, headers, response decoding, and error behavior.
