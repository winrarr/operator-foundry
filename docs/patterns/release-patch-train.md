# Optional release patch train

This is an optional release-maintenance pattern for operators that publish
immutable versioned artifacts and maintain `release/vX.Y` branches. It is not a
required part of every operator. Keep it when the repository has adopted a
release policy that benefits from automatically publishing dependency-only
patches; remove or defer it when that release model does not apply.

The reference implementation is intentionally inactive as a GitHub Actions
workflow. The copyable bundle is:

- `.github/workflow-patterns/release-branch-patch-train.yml`;
- `.github/workflow-patterns/publish-release.yml`;
- `hack/release_branch_patch_train.py`;
- `hack/required_checks.py`; and
- `hack/test_release_branch_patch_train.py`.

The detailed activation, branch-protection, and GitHub App procedure is in
[optional release automation](../contributing/releases.md).

## Release contract

The Harbor-derived example assumes a paired operator image and Helm chart:

- `vX.Y.Z` publishes an operator image and a chart from the same tagged
  commit;
- `Chart.yaml:appVersion` identifies the operator image version;
- `Chart.yaml:version` identifies the chart package version and may use a
  different version line; and
- `chart-vX.Y.Z` can publish a chart-only change while retaining the existing
  operator image through `appVersion`.

Adapt this contract if the new operator publishes a different artifact set.
The patch-train logic must not be enabled until the publication workflow and
the version/tag relationship agree with one another.

## Eligibility and decision order

The patch train follows this order. A skipped branch is a successful,
explainable outcome; it must not guess at release intent.

1. Fetch all remote branches and tags.
2. Process the manually selected branch, or the newest three valid release
   branches when no branch was selected. The supported count is configurable
   through `SUPPORTED_RELEASE_BRANCH_COUNT`.
3. Require the exact `release/vX.Y` or `release/X.Y` branch shape and an
   existing remote branch.
4. Find the newest merged stable `vX.Y.Z` operator tag for that release line.
   A branch without a stable operator tag is skipped.
5. If the branch head is already the tagged commit, enter recovery only when
   the tag message identifies an automated dependency patch release. This
   allows a missing publication workflow run to be resumed without creating a
   new version.
6. Inspect runtime changes since the operator tag and chart changes since the
   latest chart release tag. Documentation-only changes do not trigger a
   dependency patch release.
7. Skip when operator implementation paths changed outside the approved
   automatic trigger files. The reference trigger files are `go.mod`,
   `go.sum`, and `Dockerfile`.
8. Skip when chart files other than `Chart.yaml` changed without an explicit
   chart release. This prevents a dependency patch from silently publishing
   unreleased chart behavior.
9. Calculate the next operator patch and the next chart patch. Chart-only
   releases are included when determining the next chart version.
10. Accept `Chart.yaml` only when its metadata is either the previous release
    pair or the already prepared next pair. Unexpected metadata is skipped for
    human review.
11. In a live run, update both chart values in one commit, push the release
    branch, and wait for the required checks on the resulting commit.
12. Create one annotated immutable `vX.Y.Z` tag, or safely recognize the same
    tag already pointing at the expected commit. That tag starts publication.

The script exposes `CHART_PATH`, `CHART_NAME`, `REQUIRED_CHECKS`,
`AUTO_RELEASE_TRIGGER_PATHS`, `OPERATOR_RUNTIME_PATHS`, and
`PUBLISH_WORKFLOW` environment variables so the example can be adapted without
editing its decision logic. The defaults match this repository.

## Edge cases and recovery

| Situation | Required behavior |
| --- | --- |
| No release branches | Exit successfully and report that there is nothing to process. |
| Invalid or missing selected branch | Skip it with the expected branch format or missing-ref reason. |
| No stable operator tag on the line | Skip; a release line needs an explicit initial release first. |
| More than the supported maintenance window | Process only the newest configured number unless a branch was selected manually. |
| No operator or chart changes | Skip without changing metadata. |
| Dependency file plus operator code change | Skip and require explicit release intent. |
| Dependency file plus unreleased chart template/value change | Skip until the chart change has an explicit chart release. |
| Prior chart-only release | Advance the next chart version from that chart release, not from the last paired operator tag. |
| Stale or unexpected chart metadata | Skip and print both the observed and expected pairs. |
| Metadata already contains the calculated next pair | Do not create a second metadata commit. Continue safely toward the tag. |
| Required checks are pending or failed | Do not tag. Wait up to the configured timeout, then leave the branch for recovery. |
| Expected release tag already exists at the expected commit | Treat it as idempotent and do not move or recreate it. |
| Expected release tag exists at another commit | Stop and report the immutable-tag conflict. |
| A stale local tag has the wrong target | Delete only that local tag, then recreate it at the expected target; never rewrite a remote tag. |
| Automated tag exists but publication did not complete | Verify chart metadata and checks, then dispatch the publication workflow for the existing tag. |
| Existing publication lacks the expected chart asset | Do not overwrite it; stop for manual investigation. |
| A human-created tag is at the branch head without new commits | Do not assume it is an automated release or dispatch publication. |
| A later publication step fails | Keep the immutable tag, inspect the exact tag and published assets, and rerun publication for that tag. |

Dry-run mode exercises eligibility, version calculation, metadata-state checks,
and the intended tag/publication actions without committing, tagging, or
calling GitHub APIs. The local test suite uses temporary Git repositories to
cover the dependency update, prior chart-only release, prior paired release,
unreleased chart change, missing paired publication, and stale metadata cases:

```sh
python3 hack/test_release_branch_patch_train.py
```

Do not move an existing release tag to repair a failed publication. Recovery is
based on rerunning the publication workflow with the existing tag, not on
creating a replacement tag with the same version.

## Why this remains optional

The pattern adds direct branch and tag mutation, registry publication, release
credentials, and a versioning contract. Those are valuable for a maintained
operator with immutable releases, but they are not universal operator
requirements. A new project should evaluate the pattern explicitly during
kickoff, preserve the code if it is useful as a reference, and enable it only
after its artifact model, branch rules, required checks, and recovery path have
been configured.
