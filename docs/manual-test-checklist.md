# Manual test checklist

A hands-on pass over the Plan module. Automated tests cover rules, contracts and flows; this list is for what
they cannot judge: look, wording and feel. Demo data: `demo-users-and-data.md`.

## Setup

```bash
cd backend && go run ./cmd/seed
cd backend && go run ./cmd/api
cd frontend && pnpm run dev
```

All demo users share the password `mapp`.

| Person | Role | Sees |
|---|---|---|
| Priya Raman | Planner | Lubricants Europe |
| Mei Lin | Planner | Mobility China |
| Daniel Okoro | Planner and Approver | Lubricants Europe |
| Carlos Mendes | Global planner, approves Mobility level 1 | everything |
| Alex Morgan | Administrator | everything |
| Sara Novak | Viewer | both markets, read-only |
| Lea Santos | Requisitioner | Purchase requests |

Tick each box as you go. Note anything odd with the page, the person you were signed in as, and what you did.

## 1. Sign-in and app shell

- [ ] Signed out, opening the app sends you to sign-in, and after sign-in you return to the page you wanted
- [ ] A wrong password is refused with a clear message; the e-mail works in any case and with spaces
- [ ] The product is called **SP&A One** everywhere on screen; never MAPP, Program, Tactic or Hive9 terms
- [ ] Every sidebar item opens a page with its own heading and browser tab title
- [ ] The sidebar collapses and expands; the narrow window shows the "1" tile; the phone width shows a drawer
- [ ] Light and dark theme: text readable, yellow is Shell yellow, red only for errors and danger
- [ ] Home cards and sections load and every link goes to the right page (compare Priya with Daniel)
- [ ] Sign out returns to sign-in and Back does not show the page you left
- [ ] Tab reaches everything in a sensible order; Enter submits a dialog; Escape closes it without saving
- [ ] Each page has one most prominent title and at most one yellow primary button; nothing is greyed out without a reason

## 2. Settings (sign in as Alex)

- [ ] Edit a model, an attribute option and a currency rate; changes show where they are used
- [ ] Add an activity type and a finance budget item; both appear in the forms that use them
- [ ] Import options from a file: problems are listed all at once before anything is saved
- [ ] Roles and access: give Sara a Planner role on Mobility China; she may edit there at once
- [ ] The only administrator cannot remove their own Administrator role, and the reason is shown
- [ ] Sign in as Priya: Settings is missing or read-only, not broken

## 3. People and teams

- [ ] The people list shows role, team and line manager for each demo person
- [ ] Add a person: they get the Viewer role; every validation error shows at once
- [ ] Change a line manager and a team; the plan form's owner list follows the chosen teams
- [ ] Delete a person who owns nothing; Undo brings them back

## 4. Plans (Priya, then Mei)

- [ ] Priya sees only Lubricants plans; a Mobility plan address looks like "not found"
- [ ] Create a plan: required fields, completeness chip, status chip, all errors at once
- [ ] The owner can only be a member of the chosen teams
- [ ] Edit a plan; a second tab editing the same plan reports a version conflict, not silent overwrite
- [ ] Copy FY2026 Lubricants Europe into the next year: everything inside is copied and the copy opens
- [ ] Move an activity to another initiative; the roll-ups follow at source and destination
- [ ] Bulk: tick activities, change the owner, delete them, then Undo
- [ ] Import a plan skeleton from Excel: every problem is listed, then Undo the import
- [ ] Export a plan and import it back unchanged: nothing changes
- [ ] Plans with a read-only budget (global allocation) show why the field is locked

## 5. Campaigns and initiatives

- [ ] Create, edit and delete a campaign under FY2026 Lubricants Europe; Delete goes to the recycle bin with Undo
- [ ] Initiatives show remaining budget; FY2027 Global Brand shows the over-allocated initiative as negative
- [ ] Status and completeness chips are on every record; Mobility China shows its deliberate gaps
- [ ] Back from a record keeps the list's filters

## 6. Activities and line items

- [ ] Build an activity down to a line item in EUR; the cost rolls up in the reader's currency
- [ ] A GBP activity shows USD beside it; phase the budget by month
- [ ] Split a EUR line item off into a new activity; figures stay consistent
- [ ] Line items of an approved activity can be added and deleted without restarting approval
- [ ] Spend type and GL code are coded on a line item; invalid combinations are refused with the reason
- [ ] Money always shows currency and exact decimals; no rounding surprises
- [ ] Over-budget activity: with overspend control set to Refuse, saving is refused and the reason is shown first

## 7. Approvals

- [ ] Priya completes and submits an activity; the form says what happens next before she submits
- [ ] Daniel approves; for a two-level workflow Alex approves the next level
- [ ] A declined activity shows who declined it and why to the planner
- [ ] Nobody approves an activity they own: Daniel's own activity goes to the backup, Alex
- [ ] Approvals monitoring: Trade magazine ads is overdue; Run reminders and escalations escalates it to Alex
- [ ] Fleet card launch waits at level 2
- [ ] Settings: a rule that approves automatically below 10,000 USD approves a small activity on submit
- [ ] Delegation DLG-1001 (Daniel to Carlos) is shown as planned and changes nothing today

## 8. Planning views, dashboards and downloads

- [ ] All work lists everything the signed-in person may see and no more
- [ ] Executive dashboard adds up every market the person may see (compare Carlos with Priya)
- [ ] Financials: find a PO from the sidebar, open it, follow its link to the line item
- [ ] A line item shows its GSAP money without a click; the plan, campaign and initiative add up the same money
- [ ] Import a generated GSAP extract: it is checked first, and then changes the figures
- [ ] Import GSAP rates: a future rate waits for its date
- [ ] Download a plan as PowerPoint and as PDF; both open and have every page
- [ ] The SP&A view of a plan traces its figures to activities, periods and a purchase request

## 9. Activity log, recycle bin and purchase requests

- [ ] Activity log shows who changed what and when; recent changes appear as you make them
- [ ] Recycle bin: restore the deleted initiative by its activity's ID in three steps
- [ ] An approved activity restored from the bin comes back as Created
- [ ] An activity whose initiative is also in the bin: **Restore both** brings both back
- [ ] Delete permanently and Empty recycle bin ask first, and both appear in the log
- [ ] Lea sees Purchase requests; PR-1001 waits as Created on line item L-1010
- [ ] Raise, submit and approve a purchase request; it is routed to the requisitioner
- [ ] An approved activity leads through a PR to a purchase order, and Committed figures update
- [ ] The simulated SWBS request shows its code and HX key

## Always check

- [ ] Delete moves to the recycle bin with Undo; nothing is lost silently
- [ ] Wrong or missing input: all errors show at once, none are hidden
- [ ] A person who may not do something sees why, or does not see the control at all
- [ ] Browser window at phone width: nothing overlaps and nothing scrolls sideways
