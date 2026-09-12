# Rhino-bird 2026: Topic 2

Participant: Liu Debao (GitHub: `mingri31164`). Topic: Visual Sandbox Workbench.

The final code is tagged [`rhino-2026-final-2-v3`](https://github.com/mingri31164/WeKnora/tree/rhino-2026-final-2-v3), at commit `28d676539ca25fcff06aa18aa315eff14c092e43`. The feature PR is [Tencent/WeKnora#3146](https://github.com/Tencent/WeKnora/pull/3146). Official main `462999ec` is merged; the v1 and v2 tags remain unchanged.

The canonical `submission.yaml` is committed at the root of `feat/sandbox-workbench` after the final code tag, as required by the delivery guide. This branch mirrors that file and keeps public evidence. The tag must not be moved to either metadata commit.

## Run And Test

Use the project [README](README.md) for the application and dependencies. Enable the workbench and configure trusted Origins, Redis, and a dedicated sandbox according to [Sandbox Workbench](docs/sandbox-workbench.md). This document includes Docker/E2B setup, limits, security boundaries, and reproducible real-backend tests. The original presentation Skill is under [examples/skills/presentation-builder](examples/skills/presentation-builder/).

```bash
git checkout rhino-2026-final-2-v3
go test ./... -count=1
go vet ./...
python3 -B -I internal/sandbox/workbench_files_test.py
python3 -B -I internal/sandbox/terminal_runtime_probe_test.py
python3 -B -I internal/sandbox/terminal_runner_test.py
```

In `frontend/`, run `npm ci`, `npm test`, `npm run type-check`, and `npm run build`. Real-backend tests require dedicated `WORKBENCH_TEST_*` settings from the workbench document; a skipped backend is not a passing real test. Python file tests require a short physical temporary path with no symlink components.

## Validation And Limits

The final revision passed full Go tests (8,147 leaf tests, 33 environment skips), `go vet`, targeted race tests, incremental golangci-lint, 943 frontend tests, type check, build, and 55 Python tests. Docker/E2B-compatible integration tests passed 45 leaf cases in normal and race runs with no skips. The real API/WebSocket suite passed 24 cases. Concurrent two-tenant checks confirmed separate PID namespaces, private process/file visibility, rejected cross-tenant requests, and per-command audit records. See the [versioned report](submission/validation-report.md).

The v3 review fixed five confirmed gaps: access-token revocation, disguised Office archive validation, Mermaid text fallback, cursor-paginated audit history, and cumulative-output limit reasons. Existing Service/Repository and authentication contracts are reused; no new database table or Agent engine was added. [Task and architecture assessment](submission/task-review.md).

The final supplement enforces sampled aggregate command-tree RSS in addition to per-process address space. Sampling can overshoot and count shared pages repeatedly; this is not a container memory quota. The workbench terminates bounded commands, while the upstream reconnectable shell retains detach semantics.

Browser recordings and the earlier Agent-generated artifacts are separately versioned to `2a209307`. Their report retains driver failures with corrected passes, skips, an executor GPU/proc restriction, and an optional Agent import failure. The original CSV/XLSX outputs omit the requested synthetic-data notice; structure and numeric checks passed, not full prompt conformance.

A post-merge Agent run at `e0377f24` predates the v3 workbench fixes; it is not a replay of the final revision. Four files were generated and downloaded in 53.622 seconds. Its original strict checker failed on one-decimal percentage precision; independent semantic checks and five tamper counterexamples passed. The Agent recovered from a Worksheet API error. CSV lacks a synthetic notice, XLSX adds a footer, and explanatory formula text omits the factor of 100. [Raw outputs and integration details](submission/closeout-verification.md) retain these limitations.

Dependency audit reports 10 advisory package groups (7 high, 3 moderate). All affected lock entries are unchanged against both the previous branch head and official main. No zero-vulnerability or runtime exploitability claim is made.

Real E2B evidence uses a Kubernetes-compatible container backend, not E2B Cloud or native Cube. Cube workbench files remain disabled, the local compatible template catalog returns 404, and host-process execution is not exposed. No production capacity or MicroVM isolation claim is made.

Unconditional completion is not claimed. CPU/memory budgets apply to each submitted command tree, not the aggregate sandbox session or concurrent Agent/other-shell processes. Whole-sandbox quota acceptance remains unverified if required by the topic's session semantics. Final-revision browser acceptance and email sending also remain outstanding. Current upstream explicitly removed the local host-process backend.

This branch keeps a mirror of `submission.yaml`, this note, public reports, screenshots with provenance, reproducible acceptance scripts, and the synthetic Agent outputs. Credentials and private runtime logs are excluded. The email submission only needs the canonical root `submission.yaml` as an attachment.
