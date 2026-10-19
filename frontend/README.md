# frontend — NorthStar web app

The NorthStar web app: React 19 + TypeScript (strict), built with Vite (ADR-0002). It talks to the Go API
only through a small client whose types are generated from the API contract. Styles are plain CSS
Modules on the design tokens of `src/styles/tokens.css`; how screens look is in
[docs/product/design-system.md](../docs/product/design-system.md).

## Commands

Run them in this folder (`frontend/`). Start the API first (see [backend/README.md](../backend/README.md)):
the development server forwards `/api` to <http://127.0.0.1:8080> (or to `NORTHSTAR_API_URL`).

| Command             | What it does                                                                         |
| ------------------- | ------------------------------------------------------------------------------------ |
| `npm install`       | Installs the dependencies (first time, and after `package.json` changes)             |
| `npm run dev`       | Development server on <http://127.0.0.1:5173>                                        |
| `npm run build`     | Type-checks the app code (not the tests) and builds it into `dist/`                  |
| `npm run preview`   | Serves that build on <http://127.0.0.1:4173>                                         |
| `npm run check`     | All checks: API types up to date, type check, lint, format, unit and component tests |
| `npm test`          | Unit and component tests once (`npm run test:watch` keeps watching)                  |
| `npm run test:e2e`  | Browser tests (Playwright), see below                                                |
| `npm run api:types` | Regenerates `src/api/schema.d.ts` after a change to `backend/api/openapi.yaml`       |
| `npm run format`    | Formats every file with Prettier (`npm run lint` runs ESLint alone)                  |

`npm run test:e2e` builds the app and starts its own API and web app on ports 18080 and 14173, with
fresh demo data in a temporary folder, so Go must be installed too. On a new computer, run
`npx playwright install chromium` once first.

Ports are strict: when one is busy, the server stops with a message instead of moving to another port.

## Folders

```
src/
  app/          the frame: router.tsx (every page and its address), navigation.ts (sidebar items),
                layouts, sidebar, account menu, session (SessionProvider, useSession) and theme
  components/   shared building blocks (Button, PageHeader, Card, SidePanel, ConfirmDialog, Toast, …):
                one folder per component, all exported from components/index.ts
  features/     the pages of each area: sign-in/, home/, placeholders/ (Coming soon, Page not found,
                and BuildingPage — the stub of a page still being built), dev/ (the developer pages),
                and the Plan module's settings/, people/, plans/, campaigns/, activities/, line-items/,
                approvals/, approval-workflows/, notifications/ (002–008), home/ (009, the Home
                dashboard) and all-work/ (010)
  api/          client.ts (api and call()), errors.ts (ApiError and its messages), settingsQueries.ts
                (Settings read hooks) and schema.d.ts — generated from the contract: never edit it by hand
  lib/          small helpers: format.ts (dates, date ranges, relative times and money), decimal.ts
                (typed amounts, percentages, rates — exact, with BigInt), text.ts, waterfall.ts,
                recordLinks.ts (the address of every planning record), useFormatters (dates and money
                for the signed-in person), download.ts (files the API sends: All work's export),
                safeNext.ts, usePageTitle, useUndoableDelete, useUnsavedChanges, cx
  styles/       tokens.css (every colour and size, light and dark) and global.css
  test/         test helpers: setup.ts, mockApi.ts, render.tsx, users.ts (testUser()) and
                attributeForms.ts (Shell attribute forms)
e2e/            Playwright end-to-end tests (*.spec.ts) and their helpers (fixtures.ts)
```

Import from `src` with the `@/` alias: `import { Button, PageHeader } from '@/components';`. Tests sit
next to the code they test (`Name.test.tsx`).

## How to add a page

1. Create the page in `src/features/<area>/`, for example `PlansPage.tsx` with `PlansPage.module.css` and
   `PlansPage.test.tsx`. Build it from the shared components of `@/components`, and name the browser
   tab with `usePageTitle('Plans')` ("Plans · NorthStar").
2. Add its route in `src/app/router.tsx`. A page for signed-in people goes inside the `RequireSignIn`
   group and needs `handle: { title: 'Plans' }`: the tab shows that title while the session is checked.
3. Add or change its sidebar item in `src/app/navigation.ts`. When a section is built, it gets its real
   page and leaves `COMING_SOON_SECTIONS`.
4. Ask the API with TanStack Query, `api` and `call()`, as `src/features/dev/sampleApi.ts` does:
   `useQuery({ queryKey: ['sample-folders'], queryFn: () => call(api.GET('/api/sample-folders')) })`.
   `call()` returns the data or throws an `ApiError` with a plain-language message.
5. Need a new shared component? Only with a reason in the feature's plan: its own folder in
   `src/components/`, a test, and an export in `index.ts`.

## Testing tips

Component and page tests use Vitest, Testing Library and jsdom, always in UTC.

- **Fake API.** `mockApi({ 'GET /api/me': { body: priya }, 'POST /api/auth/sign-out': { status: 204 } })`
  answers each `"METHOD /path"`; a function as the answer covers special cases (a slow answer, a network
  failure). A request without an answer **fails the test** — except the app frame's background
  requests (`FRAME_ANSWERS`: the Approvals count and the notifications, answered "nothing waiting"
  unless the test answers them). To see what the code sent, keep what `mockApi` returns:
  `api.requests`, or `api.bodiesOf('POST /api/auth/sign-in')` for one endpoint.
- **People.** `testUser(overrides)` (`src/test/users.ts`) is a complete signed-in person (Priya Raman);
  `src/test/attributeForms.ts` has Shell attribute forms for form tests.
- **Rendering** (`src/test/render.tsx`). Both helpers return `{ router, queryClient, user }`:
  `router.state.location` is the current address, `user` clicks and types.
  - `renderWithProviders(ui, { route, user, path })` shows one page with the app's providers. `user` is the
    signed-in person (`null`, the default: nobody); `ui` is shown at `path` (every address unless given);
    other addresses show the stand-ins "Sign-in page" and "Another page".
  - `renderRoutes(routes, { route, user })` renders the app's real routes. Without `user`, the app asks
    `GET /api/me` as on a real page load, so mock it.
  - `renderApp(path, answers)` (`src/test/app.tsx`) opens the whole app at an address as `testUser()`,
    with `answers` for the page's requests; `expectCurrentSection('Plans')` checks the sidebar. Each
    feature folder's `routes.test.tsx` uses them; it stands in for other features' parts with
    `vi.mock`, so their changes never break it.
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
