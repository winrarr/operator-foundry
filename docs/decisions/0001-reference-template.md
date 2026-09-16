# 0001. Use one repository as both reference and template

- Status: accepted
- Date: 2026-09-16

## Decision

Operator Foundry is a public GitHub repository enabled as a template. It contains a rich but generic reference implementation and pattern catalog. New operators begin from a template snapshot, while agents can also inspect the public repository for conventions and examples.

## Rationale

A clean template reduces repeated setup work, while a concrete reference repository gives agents useful local examples for API design, external clients, reconcilers, status, tests, documentation, and workflows. The kickoff process distinguishes reusable foundation from example-domain code and keeps the examples available while a new operator becomes solid.

## Consequences

- Example code must be clearly labeled and should avoid pretending to define universal APIs.
- The kickoff workflow must require deliberate review of inherited resources and patterns.
- New projects are independent snapshots; improvements to this repository are adopted selectively.
- A large generator or shared runtime dependency is not required for the first version.
