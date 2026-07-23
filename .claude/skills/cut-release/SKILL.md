---
name: cut-release
description: >-
  Cut (publish) a release of meshery-operator by publishing the current Release
  Drafter draft with the gh CLI. Use this whenever the user asks to "cut a
  release", "publish the release", "ship a release", "release meshery-operator",
  "release the operator", "do a release", "release the latest merge", or wants
  the recently merged PR to go out. Handles waiting for the Release Drafter
  workflow to fold the just-merged PR into the draft notes, then flipping the
  draft to published so the multi-platform Docker build and the downstream CRD/
  chart sync workflows fire. The version tag is already set by Release Drafter
  and auto-increments after each release, so this skill never creates or bumps a
  tag.
---

# Cut a meshery-operator release

Publishing a release here is intentionally a one-action step: **flip the existing
Release Drafter draft from draft to published.** Everything downstream is automated
by GitHub Actions.

## How releasing works in this repo

- `.github/workflows/release-drafter.yml` runs on **every push to `master`**. As PRs
  merge, it maintains a single **draft** GitHub Release whose tag auto-increments (the
  patch version bumps after each publish). There is always exactly one draft waiting to
  be published. This repo publishes **immutable** releases (assets cannot be added after
  publish), so the drafter also attaches the rendered CRD bundles to the *draft* on every
  master push - that is the canonical asset path, and it is why you must never hand-create
  the release.
- Publishing that draft (the release `published` event) triggers:
  - `.github/workflows/multi-platform.yml` - builds the multi-architecture operator
    container images and pushes them to the registry, tagged with the released version.
  - `.github/workflows/sync-downstream.yml` - renders the CRD bundles, syncs the CRDs +
    chart versions into `meshery/meshery`'s Helm charts, and publishes the operator-versioned
    chart to `oci://ghcr.io/meshery/charts`.

So the tag is already chosen and the notes are already drafted. Your job is only to make
sure the just-merged PR is reflected in the draft, then publish it. **Do not create a tag,
do not bump a version, do not write release notes by hand** - Release Drafter owns all of that.

## Steps

### 1. Confirm the Release Drafter run for the latest master commit has finished

The draft only includes a PR after the drafter workflow run for that merge commit completes.
If you publish too early, the just-merged PR is missing from the notes.

```bash
# Latest master commit that should be in the release
git fetch origin master --quiet && git rev-parse origin/master

# Most recent Release Drafter runs (headSha should match origin/master, status "completed")
gh run list --workflow="release-drafter.yml" --branch master --limit 3 \
  --json databaseId,status,conclusion,headSha,createdAt
```

If the run for the current `origin/master` SHA is still `in_progress` (or absent because the
push just landed), wait for it before publishing:

```bash
gh run watch <databaseId> --exit-status
```

A `conclusion` of `success` for the run whose `headSha` matches `origin/master` means the draft
is up to date (and its CRD bundles are attached). If the latest run **failed**, stop and surface
that - do not publish stale notes or a draft missing its CRD assets.

### 2. Identify and inspect the draft

```bash
# Tag of the current draft (there is normally exactly one)
gh release list --limit 25 --json tagName,isDraft --jq '.[] | select(.isDraft) | .tagName'

# Read the draft notes and confirm the recently merged PR appears in them
gh release view <draftTag> --json tagName,name,isDraft,body --jq \
  '{tag: .tagName, name: .name, isDraft: .isDraft, body: .body}'
```

Verify the merged PR the user is releasing shows up under one of the category headings. If it
does not, the drafter run from step 1 has not yet indexed it - re-check step 1 rather than
publishing without it.

If more than one draft is ever returned, stop and ask the user which to publish rather than
guessing - publishing the wrong one ships an unintended version.

### 3. Publish the draft

Flipping `--draft=false` publishes it and fires the `published` event that drives the
multi-platform image build and the downstream CRD/chart sync. Mark it `--latest` so it carries
the "Latest" badge.

```bash
gh release edit <draftTag> --draft=false --latest
```

### 4. Confirm it published

```bash
gh release view <draftTag> --json tagName,isDraft,isLatest,publishedAt --jq \
  '{tag: .tagName, isDraft: .isDraft, isLatest: .isLatest, publishedAt: .publishedAt}'
```

`isDraft: false` and a non-null `publishedAt` confirm the release is out. Report the published
version to the user and note that the multi-platform Docker build and the downstream CRD +
Helm-chart sync workflows have been triggered by the release event.

## What to watch for

- **Don't publish ahead of the drafter run.** The most common failure is publishing before the
  Release Drafter workflow has folded the merged PR into the notes, shipping a release whose notes
  omit the very change being released. Because releases here are immutable, the drafter also has to
  have attached the CRD bundles to the draft first. Step 1 exists to prevent exactly this.
- **Never hand-author the tag or notes.** If you find yourself computing a version number or
  writing changelog entries, something is wrong - Release Drafter already did both.
- **One draft only.** If `gh release list` shows zero drafts, no drafter run has occurred since the
  last release; if it shows more than one, ask the user which to publish.
