Branch protection suggestion — verifying snapshot backfill

To prevent accidental merging of the NOT NULL migration before backfill verification, we recommend enabling branch protection rules on the protected branch (e.g., `main`):

1. Settings → Branches → Add rule for the target branch.
2. Require status checks to pass before merging.
   - Add `Verify Snapshot Backfill` (workflow name) as a required status check.
3. Optionally require pull request reviews and restrict who can push to the branch.

When enabled, pull requests touching `backend/migrations/20260207_add_region_to_snapshots_and_preaggs.up.sql` will trigger the `Verify Snapshot Backfill` workflow and the PR cannot be merged until the workflow (including Trino verification when configured) passes.

If you'd like, I can open a small PR that adds this doc to the repo and include a checklist for repo admins to follow when enabling branch protection.