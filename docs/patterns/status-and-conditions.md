# Status and conditions

The reference implementation uses three conditions:

- `Ready=True` means the observed external state matches the desired state;
- `Reconciling=True` means progress is possible but the controller is waiting or retrying;
- `Stalled=True` means user action or a design correction is needed before progress can continue.

All conditions carry `observedGeneration`. Condition helpers preserve transition times when only the message changes and avoid unnecessary status writes.

Use this contract when it makes generic Kubernetes status tooling useful. Add resource-specific conditions only when they communicate information that cannot be expressed by the common lifecycle conditions.
