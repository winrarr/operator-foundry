# Optional release patch train

The release patch-train workflow is kept as an inactive reference under `.github/workflow-patterns/`. Copy and adapt it only after the operator has an established release-branch policy and the repository has configured the required GitHub App credentials and protected checks.

The pattern is intended for dependency-only changes on supported `release/vX.Y` branches. It should:

1. prove that only approved runtime dependency files changed;
2. calculate the next patch version without moving an existing tag;
3. update release metadata in one commit;
4. wait for the required checks on that exact commit;
5. create the annotated operator tag that drives publication;
6. leave clear recovery output when a later step fails.

Do not enable a patch train by copying the workflow without reviewing versioning, chart publication, required checks, branch protection, token permissions, and recovery behavior for the new repository.
