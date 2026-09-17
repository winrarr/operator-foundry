# Supply-chain verification

The repository separates local security checks from release signing. This
keeps everyday development reproducible and keeps registry credentials and
OIDC permissions inside the workflows that actually need them.

## Responsibility boundaries

| Surface | Responsibility |
| --- | --- |
| `renovate.json5` | Discover dependencies, update pinned versions and digests, group changes, and restrict release-branch updates. |
| `Makefile` | Provide canonical local entry points for source scans, image scans, and SBOM generation. |
| `hack/` scripts | Hold non-trivial, deterministic parsing, release decisions, or verification logic that deserves independent tests. |
| GitHub Actions | Select triggers, parallelize jobs, cache tools and layers, grant minimum permissions, use OIDC, authenticate to registries, and publish artifacts. |
| `Dockerfile` | Define the image contents and pinned build/runtime bases. |
| Documentation | Explain what each result proves, how to verify it, and how a new operator should adapt the pattern. |

Do not put release credentials in Make targets or helper scripts. A local
Cosign command can be useful for a project that has its own signing policy, but
the reference release flow uses keyless signing in GitHub Actions so the
signature identity is tied to the workflow rather than a private key stored in
the repository.

## Renovate

`renovate.json5` is active for this repository and is also a copyable pattern.
It covers the native Go, Dockerfile, Helm-values, and GitHub Actions managers,
plus custom references in the Makefile and API-reference configuration.

The important release-train behavior is:

- the default branch and `release/vX.Y` branches are update targets;
- patch dependency updates and Docker digest refreshes may be automerged only
  after the repository checks pass;
- major and minor Go module updates are disabled on release branches;
- GitHub Actions, Helm development-image values, and custom tool-version
  updates are disabled on release branches; and
- Dockerfile runtime/build-image updates and patch-level Go module updates can
  still feed the optional patch train.

The release branch workflows run the same checks as `main`, and the patch-train
default gate uses the exact job names emitted by those workflows. If a new
operator changes job names or adds a release-critical check, update
`RELEASE_REQUIRED_CHECKS` rather than allowing the train to guess.

Renovate uses comments next to versions that are not discoverable by a native
manager. Keep those comments adjacent to the value they describe and verify
the configuration with Renovate's config validator after changing them.

## Local checks

The local entry points are intentionally explicit:

```sh
make vulnerability-scan
make docker-build IMG=local/operator-foundry:security
make image-vulnerability-scan IMG=local/operator-foundry:security
make image-sbom IMG=local/operator-foundry:security
```

The source scan combines `govulncheck` for Go code and Trivy filesystem
scanning for high and critical vulnerabilities and manifest misconfiguration.
The image scan checks high and critical vulnerabilities and ignores unfixed
findings. These thresholds are a starting policy for the reference repository,
not a claim that lower-severity or unfixed findings are harmless.

`image-sbom` writes an SPDX JSON SBOM to `dist/`, which is ignored local output.
The SBOM describes the exact image reference supplied through `IMG` or
`SBOM_SOURCE`; use a digest for release verification.

The repository has one path-scoped Trivy exception for the reference example's
intentional cluster-wide Secret read permission. Re-evaluate or remove that
exception when adapting the skeleton; it is not a general exemption for a new
operator.

The active security workflow runs source checks and builds, scans, and exports
an image SBOM. It is deliberately separate from `make check` because it needs
Docker, downloads vulnerability databases, and is slower than the source and
manifest verification suite.

## Release signing and attestations

The inactive release publication workflow contains the optional release path.
When activated and configured, it:

1. builds the release image and records the immutable digest;
2. generates an SPDX JSON SBOM for that digest;
3. signs the digest with keyless Cosign through GitHub OIDC;
4. creates a GitHub build-provenance attestation for the digest;
5. creates a signed SBOM attestation for the same digest; and
6. includes the SBOM in the GitHub Release assets.

The workflow grants `id-token: write` and `attestations: write` only because
those steps need them. Keep the publication workflow outside
`.github/workflows` until the registry, release environment, and branch policy
are configured.

A signature authenticates an artifact digest and its signing identity. A
provenance attestation is a signed statement about how that artifact was
built, including its source and workflow identity. An SBOM attestation is a
signed statement containing the component inventory. They complement a
vulnerability scan; none of them replaces the others.

For a published image, verify the Cosign signature against the expected
workflow identity and verify GitHub's provenance attestation against the
repository and signer workflow. The GitHub CLI supports verification directly:

```sh
cosign verify \
  --certificate-identity-regexp 'https://github.com/winrarr/operator-foundry/.github/workflows/.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  ghcr.io/winrarr/operator-foundry@sha256:<digest>

gh attestation verify \
  oci://ghcr.io/winrarr/operator-foundry@sha256:<digest> \
  --repo winrarr/operator-foundry \
  --signer-workflow winrarr/operator-foundry/.github/workflows/publish-release.yml
```

Use the exact image digest and signer identity for the adapted repository. Do
not verify only a mutable tag when making a deployment decision.
