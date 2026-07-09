---
name: pr-review-cycle
description: Process Copilot review comments on a PR - assess findings, fix on-point medium/high ones, push, let Copilot re-review (max 3 automated iterations), and merge on Gonzalo's approval. Use when asked to process a PR's review comments, run the review cycle, or check whether a PR is mergeable.
---

# PR Review Cycle

Automated review-response protocol for agent-authored PRs on nuchi. Copilot is
the default first reviewer; Gonzalo owns approval. Claude executes this cycle.

## Inputs

A PR number. Resolve the linked `[Backend Migration NN]` ticket from the PR
title/body before starting.

## Iteration tracking

Count prior automated iterations by scanning PR comments for the marker
`<!-- claude-review-iteration: N -->`. If N >= 3, do NOT process further
review comments automatically: comment that the automated cap is reached and
that Gonzalo re-triggers Copilot manually or reviews directly. Stop.

## Cycle (one iteration)

1. Fetch review state:
   - `gh pr view <n> --json reviews,reviewDecision,statusCheckRollup`
   - `gh api repos/{owner}/{repo}/pulls/<n>/comments` for inline comments.
   Only process comments newer than the last iteration marker.
2. Classify each unresolved Copilot comment by severity. Copilot sometimes
   tags severity in the comment body; when untagged, judge it yourself:
   - high: correctness, security, data loss, money math, ownership/auth gaps
   - medium: behavior divergence from fixtures/contract, error-shape
     mismatches, missing test coverage for acceptance criteria
   - low: style, naming, micro-optimizations, subjective preferences
3. Assess on-pointness against the ticket, spec, and
   `docs/specs/18-go-backend-replacement/api-parity-fixtures.md`. A comment is
   NOT on point if it contradicts the fixtures, the OpenAPI contract, or a
   documented intentional migration change.
4. Act:
   - high/medium and on point: fix in the PR branch.
   - high/medium but wrong: reply on the thread quoting the fixture/spec line
     that decides it. Do not change code.
   - low: reply briefly; fix only if trivial and riskless.
5. Verify: run the ticket's verification commands (minimum
   `cd backend && go test ./...` for Go changes, `bun run lint` for TS).
6. Push to the PR branch. Post a PR comment summarizing: findings addressed,
   findings rebutted (with reasons), verification output, and the marker
   `<!-- claude-review-iteration: N+1 -->`.
7. Copilot re-reviews automatically on push (repo ruleset). If it does not,
   re-request: `gh api -X POST repos/{owner}/{repo}/pulls/<n>/requested_reviewers -f "reviewers[]=copilot-pull-request-reviewer[bot]"`.

## Merge protocol

Merge ONLY when all of these hold:

1. Gonzalo has explicitly said to merge — in the session, or in a PR comment
   ("approved", "merge it", "LGTM"). GitHub review approval is NOT usable as
   the signal: Claude acts through Gonzalo's `gh` auth, so PRs are
   self-authored and GitHub forbids approving your own PR. Never infer
   approval from silence, resolved threads, or green CI.
2. CI checks are green (`backend`, `frontend`, `openapi`).
3. Review threads are resolved (ruleset-enforced), each one addressed or
   explicitly rebutted first.

Then:

- `gh pr merge <n> --merge` (merge commit, never squash-delete). NEVER pass
  `--delete-branch`; branches are retained.
- Write a descriptive merge commit subject/body: what shipped, ticket ref,
  verification evidence.
- Comment verification evidence on the ticket, close it if the PR body's
  `Closes #NN` did not, set its board status to Done
  (project 1, owner GonzaloSecades).
- Unblock the next queue ticket: remove `blocked`; add `agent:ready` only if
  `risk:low`.
- On master: run `graphify update .` and commit the refreshed artifacts.

## Never

- Never merge without Gonzalo's approval, even with green CI.
- Never exceed 3 automated iterations.
- Never delete the PR branch.
- Never silence a Copilot finding by resolving the thread without a reply.
