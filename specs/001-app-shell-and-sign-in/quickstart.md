# Quickstart: Application Shell and Demo Sign-in

**Feature**: `001-app-shell-and-sign-in` · **Plan**: [plan.md](plan.md)

How to run feature 001 on a laptop and check that it works. Details of the records are in
[data-model.md](data-model.md); the API is in [contracts/openapi.yaml](contracts/openapi.yaml)
(copied to `backend/api/openapi.yaml`, the source of truth). Every message quoted below comes from the
plan's *User-visible messages* table.

## Prerequisites

| Tool | Version | Check |
|---|---|---|
| Go | 1.27 | `go version` |
| Node.js | 24 LTS (npm 11) | `node --version` |
| Git | any recent | `git --version` |

Windows note: if Node.js was installed into `C:\Users\<you>\tools\node` after the terminal was opened,
open a new terminal (or add that folder to `PATH`).

## 1. Start the API (terminal 1)

```powershell
cd backend
go run ./cmd/seed          # first time only: loads the six demo people into backend/data
go run ./cmd/api           # API on http://127.0.0.1:8080
```

Expected: `MAPP API listening on http://127.0.0.1:8080 (environment: local, demo sign-in on)`.

The API reads the data folder only when it starts, so **stop the API (Ctrl+C) before running the seed**.
The seed refuses to run while the API is running, and refuses a folder that already has data; to start
again from the demo data use `go run ./cmd/seed -reset` (this deletes the current data).

## 2. Start the web app (terminal 2)

```powershell
cd frontend
npm install                # first time only
npm run dev                # web app on http://127.0.0.1:5173
```

Open <http://127.0.0.1:5173>. The demo password is **mapp**.

## 3. Automated checks

```powershell
cd backend
gofmt -l .                 # prints nothing when formatting is right
go vet ./...
go test ./...              # unit + contract tests

cd ../frontend
npm run check              # API types up to date, type check, lint, format, unit/component tests
npm run test:e2e           # browser tests (starts its own API on :18080 and web app on :14173)
```

All commands must finish without errors. The end-to-end tests never touch `backend/data`.

## 4. Manual validation (with the seed data)

| # | Story | Steps | Expected |
|---|---|---|---|
| 1 | US1 | Open <http://127.0.0.1:5173/plans> while signed out | Sign-in page with "Welcome back", the demo notice and password, and six demo people; tab title "Sign in · MAPP"; yellow demo banner at the top |
| 2 | US1 | Click **Priya Raman** | Plans "Coming soon" page (the address asked for); sidebar shows Priya's avatar and first name |
| 3 | US1 | Sign out, type ` PRIYA.RAMAN@shell.com ` and a wrong password | "Incorrect e-mail or password."; the e-mail stays in the field |
| 4 | US1 | Empty both fields and choose **Sign in** | "Enter your e-mail" and "Enter the password" shown together, with the summary at the top |
| 5 | US1 | Sign in, reload, then open a second tab on <http://127.0.0.1:5173> | Still signed in in both |
| 6 | US2 | Visit every sidebar item | Each opens its page (or "Coming soon"), is highlighted, and the tab title is "<Page> · MAPP" |
| 7 | US2 | Collapse the sidebar; reload | Icon rail with tooltips; still collapsed after reload and after signing in again as the same person |
| 8 | US2 | Switch to dark mode; reload; sign in as someone else | Whole app dark and remembered for the first person; the second person keeps their own setting |
| 9 | US2 | Make the window narrower than 768 px; open the menu | The sidebar slides in as a drawer; Escape, a click outside it, or choosing an item closes it |
| 10 | US2 | Open <http://127.0.0.1:5173/nowhere>, signed in and signed out | "Page not found" with **Go to Home** |
| 11 | US2 | Open the bell | "You're all caught up." |
| 12 | US3 | Open Home | "Welcome back, Priya." and the five getting-started steps in order, each with a link |
| 13 | US4 | Open the account button | Avatar, name, e-mail, job title, **My profile**, **Sign out** |
| 14 | US4 | Choose **Sign out**, then press the browser's Back button | "You have been signed out."; Back shows the sign-in page, not the previous page |
| 15 | US4 | Signed in on Home, delete the `mapp_session` cookie in the browser's developer tools, then click the theme switch | Sign-in page with "Please sign in again"; after signing in you are back on Home |
| 16 | US5 | Open <http://127.0.0.1:5173/dev/sample-records>; create a folder, rename it, add a note, delete the folder through **⋯ › Delete** | The activity-log card shows `created`, `changed` ("Name: old → new"), `created` (note, "in <folder>") and `deleted` entries with your name; the toast offers **Undo** |
| 17 | US5 | Choose **Undo** within 10 seconds | The folder and its note come back with the same IDs; a `restored` entry is logged |
| 18 | US5 | Delete a note, then delete its folder; in **Recently deleted** choose **Restore** on the note | "Restore '<folder name>' first" and nothing changes |
| 19 | FR-029 | Open the sample page as Priya in one browser and as Daniel in a private window; both edit the same folder; Daniel saves first | Priya's save says "Not saved — Daniel Okoro changed this sample folder at <time>" and keeps her typed name |
| 20 | Edge | Stop the API and reload the web app | "MAPP can't reach its server right now. Check that it is running and try again." with **Try again**; start the API, choose **Try again** → the page loads |
| 21 | FR-028 | Stop the API. Run `$env:DATA_DIR = Join-Path $env:TEMP 'mapp-empty'; go run ./cmd/api` and reload the sign-in page. Then follow the page's instruction: stop the API, run `go run ./cmd/seed`, start `go run ./cmd/api` again and reload. Finally stop the API, run `Remove-Item Env:DATA_DIR` and start `go run ./cmd/api` | First "No demo users found — load the demo data first" with the instruction; after following it, six demo people appear |
| 22 | SC-006 | Use only the keyboard: Tab to a demo person, Enter, Tab through the sidebar, open and close the account menu | Focus is always visible; everything works |
| 23 | Design | Open <http://127.0.0.1:5173/dev/components> in light and dark mode | Every shared building block of design system §5 follows design system v2 (Shell's design language) and is readable |

**SC-002** is checked in a moderated session, organised by the product owner, with at least 5 people who
have not used MAPP: each is asked to find every navigation item and to say what each section is for.
The results are recorded in `specs/001-app-shell-and-sign-in/checklists/usability.md`.

## 5. Developer pages and environments

`/dev/sample-records` and `/dev/components` (and the sample API) exist only when `APP_ENV` is `local`
(default) or `development`. Starting the API with `APP_ENV=production` turns off demo sign-in and the
developer pages: the sign-in page then says "Demo sign-in is turned off in this environment. Shell single
sign-on comes later.", and no earlier demo session is accepted.
