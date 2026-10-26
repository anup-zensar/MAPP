# ADR-0014: pnpm as the web app's package manager

**Status:** Accepted · **Date:** 2026-10-27 · **Supersedes:** npm with `package-lock.json` (constitution F1)

## Context
The product owner asked for pnpm instead of npm. The web app keeps the same dependencies and versions; only
the tool that installs them changes. The 24-hour release age (constitution VII) must stay enforced by the
package manager itself, as npm's `min-release-age=1` did.

## Decision
- **pnpm 12.15.0**, pinned by `packageManager` in `frontend/package.json` and run through Corepack, which
  ships with Node.js 24; `engines.pnpm` and `engineStrict` refuse other versions.
- **`frontend/pnpm-lock.yaml`** replaces `package-lock.json`. It was produced with `pnpm import`, so every
  resolved version is the one npm had installed.
- **`frontend/pnpm-workspace.yaml`** holds the settings that `.npmrc` held:
  - `minimumReleaseAge: 1440` (minutes): a version must have been published a day before it is installed;
  - `engineStrict: true`;
  - `allowBuilds`: install scripts of dependencies run only when listed as allowed. Today none is:
    `unrs-resolver`'s script only checks for a native binary that pnpm already installs, so it is denied.
- **CI and the web image** install with `pnpm install --frozen-lockfile`, and every command becomes
  `pnpm run …`; the dependency audit is `pnpm audit --prod --audit-level high`.

## Consequences
- pnpm's node_modules exposes only declared dependencies, so a package used without being listed in
  `package.json` fails at once instead of working by accident.
- Developers run `corepack enable` once (on Windows as administrator), or install pnpm for their user with
  its standalone installer; pnpm then switches itself to the pinned version.
- Installs are faster and share one content-addressed store per machine.
