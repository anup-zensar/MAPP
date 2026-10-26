# frontend — MAPP web app

The MAPP web app: React 19 + TypeScript (strict), built with Vite (ADR-0002). It talks to the Go API
only through a small client whose types are generated from the API contract. Styles are plain CSS
Modules on the design tokens of `src/ui/tokens.css`; how screens look is in
[docs/product/design-system.md](../docs/product/design-system.md).

## Commands

Run them in this folder (`frontend/`). Start the API first (see [backend/README.md](../backend/README.md)):
the development server forwards `/api` to <http://127.0.0.1:8080> (or to `MAPP_API_URL`).

| Command              | What it does                                                                             |
| -------------------- | ---------------------------------------------------------------------------------------- |
| `pnpm install`       | Installs the dependencies (first time, and after `package.json` changes)                 |
| `pnpm run dev`       | Development server on <http://127.0.0.1:5173>                                            |
| `pnpm run build`     | Type-checks the app code (not the tests) and builds it into `dist/`                      |
| `pnpm run preview`   | Serves that build on <http://127.0.0.1:4173>                                             |
| `pnpm run check`     | All checks: API types up to date, type check, lint, format, unit and component tests     |
| `pnpm test`          | Unit and component tests once (`pnpm run test:watch` keeps watching)                     |
| `pnpm run test:e2e`  | Browser tests (Playwright), see below                                                    |
| `pnpm run api:types` | Regenerates `src/api/generated/schema.d.ts` after a change to `backend/api/openapi.yaml` |
| `pnpm run format`    | Formats every file with Prettier (`pnpm run lint` runs ESLint alone)                     |

`pnpm run test:e2e` builds the app and starts its own API and web app on ports 18080 and 14173, with
fresh demo data in a temporary folder, so Go must be installed too. On a new computer, run
`pnpm exec playwright install chromium` once first.

Ports are strict: when one is busy, the server stops with a message instead of moving to another port.

## Folders

The layout of constitution F2 (feature 026). Every folder may import only the folders below it, and
`pnpm run lint` checks it.

```
src/
  main.tsx      mounts <App /> and nothing else
  app/          the frame: App.tsx (providers, fonts, tokens), routes.tsx (every page and its address),
                pages.tsx (each page, loaded with its feature's code), layouts, sidebar, top bar,
                account menu, notification bell, navigation.ts (sidebar items), settings-sections.ts
                (the cards of the Settings page), error pages
  features/     one folder per business area: sign-in, home, plans (with campaigns/ and phasing/),
                activities, line-items, approvals, approval-workflows, notifications, all-work,
                activity-log, recycle-bin, people (with access/ — roles and access), global-budgets,
                financials, settings (models, activity types, Shell attributes, currencies, finance
                budget items, field help), placeholders and dev. Each has one index.ts — the only way
                in for other code —, api.ts (TanStack Query hooks) and schemas.ts (Zod schemas)
  ui/           the design tokens (tokens.css, global.css) and the shared building blocks: one folder
                per component (Button/, PageHeader/, FormField/, …) and small helpers (cx, usePageTitle,
                useUndoableDelete, useUnsavedChanges, useFormProblems, record-links, theme). No index.ts:
                import `@/ui/Button/Button`
  api/          client.ts (api and call()), errors.ts (ApiError and its messages), download.ts, and
                generated/schema.d.ts — generated from the contract: never edit it by hand
  session/      the signed-in person: SessionProvider, useSession, RequireSignIn, RequirePermission,
                access.ts (permissions), safe-next.ts, api.ts
  format/       intl.ts (dates, times and money through Intl), decimal.ts (exact amounts, percentages
                and rates with BigInt), text.ts, useFormatters
  testing/      test set-up (setup.ts), the fake API (mock-api.ts, replies.ts), render helpers,
                dom.ts, axe.ts and fixtures/ (test data per feature)
e2e/            Playwright end-to-end tests and their helpers (fixtures.ts, planning.ts)
```

Imports across top-level folders use the `@/` alias; inside one folder or one feature they are
relative. A feature reaches another feature only through `@/features/<name>`. Components never call
the API client or `fetch`: they use the hooks of an `api.ts` module. Tests sit next to the code they test
(`Name.test.tsx`).

## Forms

Every form uses React Hook Form with a Zod schema in its feature's `schemas.ts`, and shows its
problems on Save, all at once (`src/features/sign-in/SignInForm.tsx` is the shortest example):

```tsx
const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues, ...CHECK_ON_SAVE });
const { problems, onInvalid, showServerProblems } = useFormProblems(form, FIELDS);
// <FormErrorSummary errors={problems} />
// <FormField control={form.control} name="email" render={(field) => <TextField label="E-mail" {...field} />} />
```

A 400 from the API goes to `showServerProblems(error.fields)`; `form.formState.isDirty` feeds
`useUnsavedChanges`. URL parameters and stored preferences are parsed with Zod too, falling back to
their default, and each such parser has a `fast-check` property test.

## Roles in the web app (014)

The API decides what a person may do; the web app only shows it. The sidebar and the Settings cards show
the items a person's roles open (`NavItem.permission`, `SettingsSection.permission`), `RequirePermission`
guards whole pages (Settings › Roles and access needs `access.manage`), and `NoAccess` and `ViewOnlyNote`
explain a refusal or a view-only page. Record answers carry `access` (what the reader may do with that
record); pages use it, and `useAccess().has(permission)`, to hide or explain actions. Sign in as the demo
person with the role you want to see (see the root [README](../README.md#demo-people-and-their-roles)).

## How to add a page

1. Create the page in `src/features/<area>/`, for example `PlansPage.tsx` with `PlansPage.module.css` and
   `PlansPage.test.tsx`. Build it from the shared components of `src/ui/`, name the browser tab with
   `usePageTitle('Plans')` ("Plans · MAPP"), and export it from the feature's `index.ts`.
2. Declare it in `src/app/pages.tsx` (`lazy(() => plans().then((feature) => ({ default: feature.PlansPage })))`)
   and give it its address in `src/app/routes.tsx`. A page for signed-in people goes inside the
   `RequireSignIn` group and needs `handle: { title: 'Plans' }`: the tab shows that title while the
   session is checked. Its code is then loaded with its feature, the first time a page of it opens.
3. Add or change its sidebar item in `src/app/navigation.ts`.
4. Ask the API with TanStack Query in the feature's `api.ts`, with `api` and `call()`:
   `useQuery({ queryKey: ['sample-folders'], queryFn: () => call(api.GET('/api/v1/sample-folders')) })`.
   `call()` returns the data or throws an `ApiError` with a plain-language message. Components use
   these hooks and never the client itself.
5. Need a new shared component? Only with a reason in the feature's plan: its own folder in `src/ui/`,
   with a test.

## Testing tips

Component and page tests use Vitest, Testing Library and jsdom, always in UTC.

- **Fake API.** Tests answer requests through Mock Service Worker: `mockApi([reply('GET /api/v1/me', { body: priya }), reply('POST /api/v1/auth/sign-out', { status: 204 })])`.
  `reply` names an operation of the contract (`'GET /api/v1/plans/{planId}'`, with `{ planId: 'P-1004' }`
  to answer one plan only), and its body must fit the contract, so a contract change that breaks a
  test fails the type check. A function as the answer covers special cases (answers that change, a
  slow answer, a network failure); `apiError(code, message)` builds an error body. A request without
  an answer **fails the test** — except the app frame's background requests (`FRAME_ANSWERS`). To see
  what the code sent, keep what `mockApi` returns: `api.requests`, or `api.bodiesOf('POST /api/v1/auth/sign-in')`.
  Tests never replace the project's own modules with `vi.mock`.
- **Test data.** `src/testing/fixtures/` holds the test data of each feature; `testUser(overrides)`
  (`fixtures/users.ts`) is a complete signed-in person (Priya Raman). `src/testing/dom.ts` finds
  elements without type assertions (`closestTo`, `parentOf`, `nextOf`, `nth`).
- **Rendering** (`src/testing/render.tsx`). Both helpers return `{ router, queryClient, user }`:
  `router.state.location` is the current address, `user` clicks and types.
  - `renderWithProviders(ui, { route, user, path })` shows one page with the app's providers. `user` is the
    signed-in person (`null`, the default: nobody); `ui` is shown at `path` (every address unless given);
    other addresses show the stand-ins "Sign-in page" and "Another page".
  - `renderRoutes(routes, { route, user })` renders the app's real routes. Without `user`, the app asks
    `GET /api/v1/me` as on a real page load, so answer it.
  - `renderApp(path, answers)` (`src/testing/app.tsx`) opens the whole app at an address as `testUser()`,
    with `answers` for the page's requests; `expectCurrentSection('Plans')` checks the sidebar. Pages load
    lazily, so wait for them with `findBy…`.
- **Toast buttons.** `await findMessages()` before clicking a button in a message (Try again, Undo): the
  messages area takes clicks only a moment after the text appears.
- **After a navigation, wait in two steps**: first for the address,
  `await waitFor(() => expect(router.state.location.pathname).toBe('/sign-in'))`, then for the new
  page's content with a `findBy…` query.
- **Fake timers** must keep ticking, because Testing Library waits in real time. Create your own `user`
  for them (not the one the render helpers return):

  ```ts
  vi.useFakeTimers({ shouldAdvanceTime: true });
  const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
  ```

- **Radix `asChild`**: never give its child a `className` function (such as `NavLink`'s), because Radix
  joins class names as text. Use a plain class name and style states with attributes, for example
  `.item[aria-current='page']`.
- **Lint errors, not warnings**: the React Compiler rules of `eslint-plugin-react-hooks` v7 (no
  `setState` directly in an effect, no reading refs while rendering, pure components, …) and
  `react-refresh/only-export-components` (a file with components exports only components). That is why
  a React context and its hook live in a `.ts` file (`useSession.ts`, `useToast.ts`) and the provider in
  a `.tsx` file.

## Look and feel

MAPP looks like a Shell enterprise tool (product owner decisions of 20 October 2026; design system
§2–§5): a white 56 px top bar and a white 240 px sidebar on a light grey page (`#F5F5F5`), white cards,
Shell yellow (`#FFC600`, dark text) for the one primary button of a page or dialog and for the "you are
here" bar, red only for danger and errors, Jost for headings and Arial for text. Every colour, size and
space is a token in `src/ui/tokens.css` (light and dark); components use `var(--token)` and never
raw colours, font sizes or off-grid spacing:

- type: `--text-xs` 12 px … `--text-3xl` 32 px (14 px base), headings in `--font-heading`;
- spacing: `--space-1` (4 px) … `--space-16` (64 px), the 4 px grid; 1–2 px only for borders;
- controls: `--control-sm` 32 px (toolbars, tables), `--control-md` 40 px (default), `--control-lg` 48 px;
- radii: 4 / 8 / 12 / 16 px (`--radius-chip`, `--radius-control` and `--radius-card`, `--radius-dialog`);
- layout: `--content-max` 1300 px, breakpoints 600 and 960 px (written as numbers in media queries).

A box that scrolls sideways (a wide table) must be `position: relative`, so the visually hidden labels
inside it cannot widen the whole page.

## Third-party notices

- **Jost** (headings), from `@fontsource/jost` 5.3.0: Copyright 2020 The Jost Project Authors
  (<https://github.com/indestructible-type/Jost>), licensed under the SIL Open Font License, Version 1.1
  (OFL-1.1). The full licence text ships with the package (`node_modules/@fontsource/jost/LICENSE`) and is
  at <https://openfontlicense.org>. Only the Latin 600 and 700 weights are bundled, by `src/app/App.tsx`.
- No Shell logo (pecten) or Shell typeface is used: both need Shell's written consent.
