# Contributing

Keep source code, tests, documentation, generated assets, and workflows
synchronized. Use the repository's canonical Make targets and preserve the
distinction between source files and generated output.

When adding a reusable workflow, put durable instructions in a focused repo-local
skill under `.agents/skills/` and keep `AGENTS.md` as a pointer to the kickoff
document.

Release branches, publication, and the patch train are optional. See
[optional release automation](releases.md) before enabling that pattern.

The local security checks and optional release signing/attestation pattern are
documented in [supply-chain verification](../patterns/supply-chain.md).
