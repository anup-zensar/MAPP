# ADR-0008: Role-based access with team scopes

**Status:** Accepted · **Date:** 2026-10-20 · **Supersedes:** ADR-0005's "every signed-in user may do
everything" (its demo sign-in stays)

## Context
ADR-0005 postponed roles: every signed-in user could do everything. Shell's BRD makes access control an MVP
requirement: view-only versus edit by role and team, "configurable through access rules rather than code"
(BUS.ADM.020); global and local budget owners who see different figures (BUS.BUD.040); role-based approvals
with segregation of duties (BUS.WFL.010). The product owner decided on 2026-10-20 to build every MVP gap now
(D-315), while sign-in stays a demo until Shell's Microsoft Entra ID is available. About 210 API operations
in the packages of 002–017 must each be checked, and a forgotten one must not slip through. Feature 014
specifies the behaviour (`specs/014-access-control/`).

## Decision
- **Permissions are fixed in code; roles are data.** `internal/platform/authz` holds the catalogue of 25
  permissions (014 Table A); each is either scoped (applies to plans in scope) or applies everywhere. A role
  is a named set of permissions that administrators edit in Settings › Roles and access; five default roles
  are seeded (Administrator, Global planner, Planner, Approver, Viewer; 014 Table B). A person may hold
  several roles and gets their union. The Administrator role is locked to every permission and All markets,
  and the last active administrator cannot be removed (D-223, D-224, D-234).
- **Scopes by team; a market is a team.** Every role assignment has a scope: All markets, My teams (the
  person's live team memberships) or Chosen teams. A plan is in scope when one of its teams is; a plan
  without teams is in scope for its creator. A person sees a plan, and everything in it, when a role gives
  "View plans" for it or they own it; activity owners and named approvers also see their activities
  (D-225, D-226).
- **Enforced by the server, in one place per kind of rule** (D-228):
  1. the access middleware (`access.Authorize`, inside the `signedIn` wrap every package already receives)
     builds the person's Grants once per request and applies a **central route table**
     (`internal/access/routes.go`) with one rule per API operation — Public, Signed in, Need *p*, InPlan *p*,
     View or Custom (014 plan, Table C and its additions for 015–017, D-301);
  2. scoped checks run in the service method the handler calls (`authz.RequireIn`, `authz.RequireView`),
     for every record a change touches (source and destination of moves and copies);
  3. lists, totals and exports are filtered where plans are read;
  4. the activity log and the recycle bin go through one `history.Access` adapter (D-238).
- **Fail closed.** An operation without a rule is refused, and `app/access_routes_test.go` fails until the
  rule exists. Every HTTP route except the four public ones (health, demo people, sign-in, sign-out)
  passes the middleware. A context without Grants (seed, internal calls, service unit tests) is
  unrestricted.
- **403 `forbidden`.** Every refusal by permission or scope answers HTTP 403 with error code `forbidden` and
  a plain message that says whom to ask; a record outside the person's access answers 403 with a message
  that names nothing of it. 404 stays for records that do not exist; 409 `not_allowed` stays for rules about
  a record's state (D-227, D-299). Refusals are not written to the activity log; the technical log records
  `forbidden`, the operation pattern and the person's ID only (D-241).
- **The server decides, the web app shows.** Record answers carry `access` objects (`{available, reason}`
  per action) and `GET /api/me` carries the person's roles, permissions and scopes; the web app uses them
  only to hide or explain actions (D-229, D-230).
- ADR-0005's demo sign-in, the actor on every request and the activity log stay unchanged.

## Consequences
- Every operation added from now on gets its route rule in the same change; specs name the permission each
  action needs.
- Every existing package gains scoped checks and filtered lists. API tests sign in with a role
  (`apitest.GrantRole`); the end-to-end journeys sign in as the persona whose seeded role allows each step.
- Shell changes roles, their permissions and scopes without code. A new permission is a code change,
  because the list of possible actions is product behaviour.
- With Microsoft Entra ID, only the sign-in page and the middleware that produces the actor change; role
  assignments may then come from Entra app roles or groups, mapped to the same roles and scopes.
- Scoping by market attribute values (Business unit, Country/OU) can be added later as a fourth scope type
  without changing the permission checks.
- The web app is never the only line of defence: a hidden or disabled button grants or refuses nothing.
