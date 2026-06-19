@AGENTS.md

# Fork Maintenance (uniiverse/kargo)

This repository is **uniiverse/kargo**, a fork of upstream **akuity/kargo**
(git remote `upstream`). We do **not** intend to submit our changes upstream.

## Branch model

| Branch | Role |
|--------|------|
| `main` | Pristine mirror of `upstream/main`. **Fast-forward only — never commit custom work here.** Keeps a clean diff against upstream. |
| `universe` | Integration branch holding all custom changes. Based on the latest upstream **release tag** (started at `v1.10.7`), not `upstream/main`. |
| `feat/*` | Short-lived branches off `universe` for individual changes. |

## Syncing with upstream

1. `git fetch upstream --prune`
2. Fast-forward `main`: `git checkout main && git merge --ff-only upstream/main`
3. Bump `universe` to the new release tag by rebasing it forward, re-resolving
   conflicts once. Build and test before deploying from `universe`.

## Rules

- **Base `universe` on release tags, not `upstream/main`.** Kargo is a deployed
  control plane — track stable releases and bump deliberately.
- **Never cherry-pick in-flight upstream PRs onto custom branches.** They later
  land in upstream via different SHAs and collide on every subsequent sync. Wait
  for the release that includes a fix; if urgently needed, cherry-pick it but
  drop it at the next release bump.
- Custom value-add currently carried: the `string-replacer` promotion step
  (on `universe`). The project-labels UI feature and "watch recovery when k8s
  API disconnects" are parked on `feat/project-labels-ui` for later re-landing.
