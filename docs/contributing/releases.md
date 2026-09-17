# Optional release automation

Release automation is an optional capability in Operator Foundry. The
reference repository keeps the Harbor-derived workflows outside
`.github/workflows` so a new operator can study and adapt them without
accidentally gaining credentials or permission to mutate branches and tags.

Use this page with the [release patch-train pattern](../patterns/release-patch-train.md).
The pattern is appropriate when the operator publishes immutable image and
chart versions, maintains release branches, and wants dependency-only updates
to receive a tested patch release. It is not expected for every operator.

## Files to activate

Copy or adapt these files only after the release contract is accepted:

```text
.github/workflow-patterns/publish-release.yml
.github/workflow-patterns/release-branch-patch-train.yml
hack/release_branch_patch_train.py
hack/required_checks.py
hack/test_release_branch_patch_train.py
```

Copy the two YAML files into `.github/workflows/` to activate them. The Python
files can remain in `hack/`; they have no effect until the workflow is active.
Adapt the chart path, chart name, image and chart publication rules, release
check names, and any additional approved dependency files. Keep the workflow
outside `.github/workflows` if the pattern is only being retained as an
example.

The patch train expects all required checks to run on release branches. Add
`release/v*` to the relevant `push` and pull-request workflow triggers before
enabling it, or change the configured required-check list to the checks that
the repository actually runs. Confirm the exact names with `gh pr checks` or
the commit's Checks page; the reference defaults are the seven job names
emitted by the current CI workflows: documentation, lint, generated assets,
unit and contract tests, Kind E2E, source vulnerability scan, and image
vulnerability scan/SBOM.

## Branch protection

Protect both `main` and `release/v*` with repository rules or rulesets. Exact
settings vary with the team's review policy, but the release line must retain
these properties:

- normal human changes arrive through reviewed pull requests;
- the required verification checks are required before merging;
- force-push and branch deletion are disabled;
- the release branch cannot be modified by arbitrary repository actors; and
- the release automation identity is the only deliberate exception for its
  metadata commit and immutable tag creation.

The Harbor-derived patch train pushes its generated `Chart.yaml` commit
directly to the release branch and then waits for checks on that exact commit.
Therefore, a repository that requires pull requests for every write must add
the release GitHub App as a narrowly scoped bypass actor for the matching
`release/v*` rule, or use an equivalent ruleset exception for that app. Do not
disable branch protection globally and do not allow general users to bypass it.

Configure the required status checks using the exact check-run names emitted by
the active workflows. Required checks should cover documentation, lint,
generated-artifact verification, unit/contract tests, and the relevant live
E2E test. If the operator has additional release-critical checks, include them
and pass the same list through `REQUIRED_CHECKS`.

The `main` rule should also protect the workflow and release-pattern files.
The patch train dispatches publication from `main`, so changing those files
should follow the same review and required-check path as other release
automation.

## GitHub App and credentials

Create a dedicated GitHub App for release automation rather than using a
personal token. Install it only on the repositories that use the pattern. A
minimal permission set for the reference workflows is:

- Contents: read and write, for checking out, committing release metadata,
  and pushing immutable tags;
- Checks: read, for waiting on the exact release commit's checks; and
- Pull requests: read, if the repository's check or branch tooling needs pull
  request metadata.

The workflow's `GITHUB_TOKEN` also needs Actions write permission when the
patch-train recovery path dispatches `publish-release.yml`. Keep that
permission at the job level and remove it if the repository does not use the
dispatch recovery path. The publication workflow separately needs Contents
write and Packages write to create releases and publish image/chart artifacts.

Configure these repository or organization values without putting secrets in
the repository:

| Name | Kind | Purpose |
| --- | --- | --- |
| `RELEASE_PATCH_TRAIN_ENABLED` | Actions variable | Must be `true` before the active patch-train job can mutate anything. |
| `RELEASE_APP_CLIENT_ID` | Actions variable | Client ID of the dedicated GitHub App. |
| `RELEASE_REQUIRED_CHECKS` | Actions variable | Comma-separated exact check-run names; defaults to the reference five. |
| `RELEASE_APP_PRIVATE_KEY` | Actions secret | Private key for the installed GitHub App. |

The App must be allowed to push to the protected release branches under the
branch rule described above. The private key must be available only as the
Actions secret and must never be echoed, committed, placed in a test fixture,
or included in a workflow artifact.

Before enabling live mode, run the workflow manually with a selected release
branch and `dry_run: true`. Verify that it finds the intended stable tag,
calculates the expected chart and operator versions, recognizes chart-only
history, and skips mixed implementation/chart changes. Then verify the App can
read checks and push only to the intended protected release branch in a test
repository or controlled release line.

## Operating and recovering the train

The active workflow runs on approved dependency-file changes on release
branches, weekly as a recovery backstop, and manually for a selected branch.
Its concurrency group does not cancel an in-progress run because a branch or
tag may already have been changed.

A normal live run is:

1. dependency updates land on a supported release branch;
2. the train validates that no unapproved operator or chart behavior is mixed
   in;
3. it commits the next `Chart.yaml` operator/chart patch pair;
4. it waits for checks on that exact commit;
5. it creates the annotated operator tag; and
6. the publication workflow builds/pushes the image and chart and creates the
   GitHub Release.

If metadata was committed but the tag was not created, rerun the train after
the checks complete. If the tag was created but publication failed, rerun the
publication workflow for that existing tag. Do not move or recreate the tag.
Inspect the tag target, chart metadata, image digest/tag, chart package, and
GitHub Release assets before retrying.

Run the local patch-train tests whenever the script changes:

```sh
python3 hack/test_release_branch_patch_train.py
```

The tests are deliberately independent of GitHub credentials and do not touch
the real repository remote.
