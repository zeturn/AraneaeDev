# Araneae Permission Matrix

This matrix reflects the current Araneae access model enforced by API route guards and per-resource ownership checks.

## Core Permission Labels

- `araneae.read`: read app resources.
- `araneae.write`: create or update resources.
- `araneae.admin`: tenant-level app administration.

## Role Mapping

- `admin`: `araneae.admin araneae.write araneae.read`
- `operator`: `araneae.write araneae.read`
- `viewer`: `araneae.read`

## Feature Tags

- Workspaces:
  - Read: `araneae.read`
  - Create/Update/Delete: `araneae.write` + owner/team access checks
- Projects:
  - Read: `araneae.read`
  - Create/Update/Delete/Upload: `araneae.write` + owner/workspace access checks
- Versions:
  - Read: `araneae.read`
  - Update/Delete: `araneae.write` + project ownership checks
- Tasks and Runs:
  - Read: `araneae.read`
  - Create/Update/Delete/Trigger: `araneae.write` + ownership checks
- Schedules:
  - Read: `araneae.read`
  - Create/Update/Delete/Enable/Disable/Trigger: `araneae.write` + ownership checks
- Users:
  - Read: `araneae.read`
  - Non-privileged users are restricted to their own user profile
- Nodes:
  - Read/Create/Update/Delete/Register/Install/Capabilities: admin-only role gate + `araneae.read`/`araneae.write`
- Teams (Araneae API surface):
  - Read path is delegated to BasaltPass S2S Team API
  - Write path is disabled in Araneae (`501 Not Implemented`)

## BasaltPass Integration Notes

- `BASALTPASS_ADMIN_EMAILS`: comma-separated emails that are force-mapped to local `admin` role during OAuth login.
- `BASALTPASS_S2S_SCOPES`: scopes used by Araneae client-credentials call to BasaltPass S2S APIs.
- Team data source: BasaltPass `/api/v1/s2s/users/:id/teams` and `/api/v1/s2s/teams/:id`.
