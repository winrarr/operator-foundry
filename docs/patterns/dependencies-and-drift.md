# Dependencies and drift

Dependent resources should report dependency readiness through status, requeue at a bounded interval, and watch referenced resources when that is safe and useful. A dependency failure should not be confused with an invalid resource specification.

Periodic drift detection is useful when external changes are not otherwise observable. Its interval should be explicit, bounded, and disabled when polling would be unsafe or unnecessary. Remote deletion should be treated as an explicit state transition rather than an unexplained API error.
