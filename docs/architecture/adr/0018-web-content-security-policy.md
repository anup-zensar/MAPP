# ADR-0018: The web app's content security policy, with inline style elements allowed

**Status:** Accepted · **Date:** 2026-10-29

## Context

Feature 038 FR-015 (D-355, research R7) puts every web page under a content security policy that admits only
the app's own scripts, styles, fonts, images and connections, and the inline theme script of `index.html` only
by its sha256. Constitution F7 forbids inline scripts and evaluated code; it does not forbid inline styles.

Checked on the built app (`vite build` + `vite preview`, Chromium) with `style-src 'self'`:

- Opening any Radix dialog or side panel adds a `<style>` element (react-remove-scroll-bar, used by Radix's
  scroll lock). Its text holds the scrollbar's width, so it differs from one computer to another and has no
  fixed hash. The browser blocked it (`style-src-elem`), and the page behind a dialog kept scrolling.
- Zod 4 probes `new Function("")` once per page to choose its fast path. The browser blocked it as intended,
  but reported a `script-src` violation.

## Decision

- The policy (one source: `frontend/security-headers.ts`, copied into `frontend/nginx.conf` and kept equal by
  `security-headers.test.ts`) is:
  `default-src 'self'; script-src 'self' 'sha256-<theme script>'; style-src 'self'; style-src-elem 'self'
'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'none';
base-uri 'self'; form-action 'self'; frame-ancestors 'none'`.
- `'unsafe-inline'` applies to `<style>` elements only (`style-src-elem`). Style attributes in markup stay
  refused (`style-src 'self'`); React's `style={}` sets styles through the CSSOM, which no policy blocks.
- `main.tsx` sets `z.config({ jitless: true })`, so Zod never probes for eval and nothing is reported.
- Enforced by `vite preview` and nginx; the Vite dev server sends the same policy as report-only, since its
  runtime injects its own inline preamble.

## Alternatives considered

- **A hash of Radix's `<style>`**: its text changes with the scrollbar width. Rejected.
- **A nonce**: needs a server that writes a fresh nonce into each `index.html`; nginx serves the file as is.
  Rejected.
- **`'unsafe-inline'` for all styles**: would also admit style attributes. Not needed. Rejected.
- **Patching Radix to skip the scroll lock**: the page behind a modal would scroll again, and the patch would
  need upkeep on every Radix update. Rejected.

## Consequences

- Injected CSS in a `<style>` element is not blocked. React escapes text, and no code writes HTML from data, so
  injecting one would first need a script injection, which the policy still blocks.
- A new library that needs eval or inline scripts breaks the end-to-end tests: `e2e/support/no-csp-violations.ts`
  fails any test on a `securitypolicyviolation`.
