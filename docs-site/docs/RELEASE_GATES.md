# Release Gates

This document defines the minimum quality gates before merging to the main branch and releasing to production.

## Required CI Jobs

All jobs in `.github/workflows/ci.yml` must pass:

- `python-lint`
- `frontend-build`
- `frontend-typecheck`
- `controlnode-quality`
- `executionnode-smoke`

## Branch Protection (Recommended)

For `main`:

- Require pull request before merging.
- Require status checks to pass before merging:
  - `Python Lint (Ruff)`
  - `Frontend Build`
  - `Frontend Type Check`
  - `ControlNode Test And Deploy Check`
  - `ExecutionNode Smoke Check`
- Require branch to be up to date before merging.
- Include administrators in branch protection.
- Disable force pushes and branch deletion.

## Pre-Release Checklist

- Environment:
  - Production `.env` values are set and validated.
  - `DJANGO_DEBUG=False`.
  - Strong `DJANGO_SECRET_KEY`.
  - `ARANEAE_CALLBACK_SHARED_SECRET` and `ARANEAE_NODE_API_TOKEN` are set.
- Backend:
  - `python manage.py check --deploy` passes.
  - `python manage.py test` passes.
- Frontend:
  - `npm run build` passes.
- Runtime:
  - Compose services are healthy (`healthcheck` green).
  - Basic smoke path works: login, node registration, task callback.

## Rollback Baseline

- Keep previous image tags available.
- Keep previous `.env` and compose bundle versioned.
- Ensure DB backup exists before release.
