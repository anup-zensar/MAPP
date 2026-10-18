# ADR-0005: Demo sign-in now, Entra ID and roles later

**Status:** Accepted · **Date:** 2026-10-18

## Context
The product owner asked to postpone authentication and role-based access control, while keeping a single
sign-in page like ZenReach's. Shell's target is Microsoft Entra ID with app roles.

## Decision
A demo sign-in page: users choose a seeded demo person (or type its e-mail and the shared demo password).
The API issues a session token and resolves the **actor** on every request. Every signed-in user may do
everything. Services always receive the actor and record it in the activity log. Demo sign-in is enabled
only in local and development environments.

## Consequences
- Real sign-in replaces only the sign-in page and the middleware that produces the actor.
- Permission checks are added later in services, at one well-defined point per operation.
- The demo password is shown on the sign-in page; this is acceptable only because the data is fictitious.
