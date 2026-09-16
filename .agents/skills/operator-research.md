---
name: operator-research
description: Research an external system and comparable operator implementations before designing a Kubernetes operator.
---

# Operator research

Research is input to design, not a substitute for design.

Start with official documentation, API specifications, version notes, SDKs, and source code. Then inspect comparable operators and adjacent tools, including their tests and operational documentation. Search for identity, scope, authentication, idempotency, updates, deletion, retries, eventual consistency, rate limits, drift, ownership, and status behavior.

For every important finding, distinguish:

- observation: directly supported by a source;
- inference: a conclusion drawn from multiple observations;
- hypothesis: plausible but unverified;
- decision: accepted project behavior.

Write a dated research note with source links, scope, source quality, relevant repository paths or commits, unresolved questions, and the decisions it informs. Do not copy a comparable project's API or lifecycle semantics without checking the new external system.
