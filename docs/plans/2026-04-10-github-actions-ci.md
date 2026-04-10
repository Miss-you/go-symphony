# GitHub Actions CI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a runnable GitHub Actions CI workflow with separate build, lint, unit, and e2e jobs.

**Architecture:** Use one workflow file with four jobs. Reuse the repository `Makefile` for build and test commands, and use the official `golangci-lint` action for lint annotations and deterministic setup.

**Tech Stack:** GitHub Actions, Go, Make, golangci-lint

---

### Task 1: Add Workflow Skeleton

**Files:**
- Create: `.github/workflows/ci.yml`

**Step 1: Define triggers and defaults**

Add a single workflow that runs on:

- `push` to `main`
- `pull_request` targeting `main`

**Step 2: Add shared job structure**

Create four jobs:

- `build`
- `lint`
- `unit`
- `e2e`

Each job runs on `ubuntu-latest`.

### Task 2: Wire Commands To Jobs

**Files:**
- Modify: `.github/workflows/ci.yml`

**Step 1: Build job**

Use checkout + setup-go, then run:

```bash
make build
```

**Step 2: Lint job**

Use checkout + setup-go, then run `golangci/golangci-lint-action`.

**Step 3: Unit job**

Use checkout + setup-go, then run:

```bash
make test-unit
```

**Step 4: E2E job**

Use checkout + setup-go, then run:

```bash
make test-e2e
```

### Task 3: Verify The Workflow Definition

**Files:**
- Modify: `.github/workflows/ci.yml`

**Step 1: Validate YAML parses**

Run a local YAML parse check against `.github/workflows/ci.yml`.

**Step 2: Re-run the local commands**

Run:

```bash
make build
make lint
make test-unit
make test-e2e
```

Expected: all commands exit `0`.
