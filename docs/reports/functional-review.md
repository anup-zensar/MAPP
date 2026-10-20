# NorthStar functional review — Plan module (features 001–011)

**Date:** 20 Oct 2026 · **Branch reviewed:** `plan-module` (commit `e4966aa`), plus `lane-011` for feature 011 ·
**Reviewer:** functional review agent · **Audience:** product owner and the Shell client team

## Summary

NorthStar's Plan module works from start to finish. A planner can build a plan down to line items, see what is
missing, fix it from the links and submit. An approver can approve from the Approvals page, and an
administrator can manage Settings, people and teams. Every business rule we tried held, for example dates
inside the initiative's dates, duplicate e-mails and not deactivating yourself. Forms show all errors at
once and protect unsaved changes. Exports work.

These are the most important problems for a production-ready, Shell-facing tool:

1. **The plan page breaks owner names one letter per line (high).** On a normal 1440 px desktop screen,
   the Owner column of the plan tree prints "P / ri / y / a / R / a / m / a / n". The plan page is the
   screen people use most, so this is the most visible quality defect.
2. **Deleted records cannot be brought back after the 10-second Undo (high).** The delete dialog says "It
   moves to the recycle bin; you can restore it", but the Recycle bin page says "Coming soon". The records
   are kept on the server, but the screens give no way back to them.
3. **The owner of an activity can approve it themselves (high, needs a client decision).** The submitter
   sees a blue **Approve** button in the header of their own activity. This follows decision D-020 ("any
   signed-in user can approve"), and the dialog does warn, but Shell will expect separate people to submit
   and approve.
4. **"View history", the Activity log and Home's "Recent changes" all lead to "Coming soon" (medium).**
   Feature 011 is built on its own branch but not merged. Merging it means fixing **5 files by hand**, and
   one of them is in the restore logic, where a careless merge breaks the build or silently loses the
   approval rules for restoring.
5. **Words and numbers disagree between pages (medium):**
   - "In progress" means two different things.
   - Amounts are written as "$500.00" in one place and "500.00 USD" in another on the same page.
   - All work's "Unassigned planned cost" does not equal planned cost minus line items, and the screen does
     not explain why.
   - Settings sections use different filter pills and different confirmations for the same action.
6. **Phone screens scroll sideways (medium).** At 390 px width the plan page is 709 px too wide and the
   activity page 460 px too wide.
7. **The demo and the documents do not match what is built (medium).**
   - The demo data has an empty activity log and an empty recycle bin, but the demo document says both
     have content.
   - 153 task boxes in `tasks.md` for features 002–010 are still open, and many of them are for code that
    already exists.

**Counts:** 3 high · 12 medium · 10 low findings (25 in total). The list of spec requirements that are not
built is near the end; most of them are feature 011.

**How this was tested:**
- **Setup:** fresh demo data, loaded into a temporary folder. The API ran on 127.0.0.1:18102 and the web
  app on 127.0.0.1:14102, and both were stopped at the end.
- **Browser testing:** scripted Playwright journeys as Priya Raman (planner), Daniel Okoro (approver) and
  Alex Morgan (administrator), at 1440 × 900 and at 390 px width.
- **API testing:** direct calls with curl.
- **Feature 011:** a trial merge and a backend test run, both done in a temporary copy of the repository.
  The repository itself was not changed.

Screenshots named below are in
[`docs/reports/functional-review-evidence/`](functional-review-evidence/); where a screenshot is not
copied there, its name refers to the review's temporary run.

**Severity key:**
- **High:** blocks or misleads a core task, or is a business-control risk.
- **Medium:** makes the work slower, causes confusion or inconsistency, or leaves a dead end.
- **Low:** polish.

---

## 1. Planner journey — build a plan, check status and what is missing, submit

What we did (as Priya):
- New plan, then Add campaign, Add initiative, Add activity, and a line item.
- Tried **Submit for approval** while fields were missing.
- Followed the Ready-to-submit links to the edit form and filled everything in.
- Fixed the line item's finance coding, then submitted.

It worked, and the activity's planned cost rolled up correctly to the initiative, campaign and plan. It took
about 44 clicks and field entries from an empty plan to a submitted activity, and 21 of those went on the
"needed to submit" fields (see P-04).

### P-01 · High · The Owner column of the plan tree breaks names one letter per line

- **What happens:** on a desktop screen (1440 px wide), every campaign and initiative owner prints one or
  two letters per line ("P / ri / y / a / R / a / m / a / n"). Each row becomes about 150 px tall, and the
  tree of a normal plan fills three screens. The plan row prints the owner as plain text with no avatar,
  while the rows below use avatar and name, so one column has two styles. The "Budget used" value also
  wraps ("38 / %").
- **Steps:** sign in as Priya and open **Plans › FY2026 Lubricants Europe** (`/plans/P-1001`).
- **Evidence:** `priya-03-plans_P_1001.png`, `misc-03-after-delete.png`.
- **Cause:** `PersonLabel` sets `overflow-wrap: anywhere` on the name
  (`frontend/src/components/PersonLabel/PersonLabel.module.css:17`). In a table whose column widths follow
  their content, that lets the Owner column shrink to one character.
- **Files:**
  - `frontend/src/features/campaigns/campaignRows.tsx:175` (avatar and name for campaign and initiative rows)
  - `frontend/src/features/plans/tree/PlanTree.tsx:104` (plain name for the plan row)
  - `frontend/src/components/TreeTable/TreeTable.module.css`

### P-02 · High · Delete promises a recycle bin that cannot be opened

- **What happens:**
  - The delete dialog says "It moves to the recycle bin; you can restore it."
  - The toast offers **Undo** for about 10 seconds, and its screen-reader text says "Restore it from the
    recycle bin".
  - The **Recycle bin** page in the sidebar says "Coming soon — This part of NorthStar isn't built yet."
  - After the toast is gone, a planner cannot bring the record back. The API still holds it
    (`GET /api/recycle-bin` lists it as `BIN-1001`), but no screen reaches it.
- **Steps:** open any activity, choose **⋯ › Delete › Delete activity**, wait 12 seconds, then open
  **Recycle bin**.
- **Evidence:** `misc-03-after-delete.png`. The dialog and toast texts were recorded during the review run.
- **Files:**
  - `frontend/src/app/navigation.ts:66-78` (Recycle bin and Activity log are still "Coming soon" sections)
  - the 011 pages that would fix this are on branch `lane-011` (see section 6)

### P-03 · Medium · Dates on forms ignore the user's date format

- **What happens:**
  - The start and end date fields use the browser's own format. In an en-US browser they show
    "01/01/2026" and "12/31/2026".
  - Directly underneath, the hint says "Allowed: 01 Jan 2026 – 31 Dec 2026".
  - My profile lets people choose a date format (DD MMM YYYY by default), but that choice applies only to
    text, not to the fields people type into.
  - UK users with en-GB browsers will see dd/mm/yyyy, so the risk is mixed formats on one screen rather
    than wrong dates. This was already listed as a known item.
- **Steps:** open **Add activity** or **Edit activity**.
- **Evidence:** `planner-07-activity-form.png`.
- **Files:**
  - `frontend/src/components/DateRangeField/`
  - `specs/003-people-and-teams/spec.md` FR-024

### P-04 · Medium · Too many separate fields stand between a new activity and Submit

- **What happens:**
  - A new "Event" activity in a Lubricants plan starts with **15 fields needed**. Its first line item
    (created in one click with defaults) adds 3 more, so "Submitting needs 18 things".
  - Filling them took 16 field interactions in one edit, and 5 more on the line item.
  - The create form shows these fields, but marks them only with small grey "Needed to submit" text.
  - Nothing is copied down from the initiative or campaign (Country / OU, Company code, Brands, Sector …),
    so every activity repeats the same answers.
- **Suggestion:** this is a spec-level change for the product owner, not a bug:
  - copy values down from the parent or from a sibling activity ("Copy Shell attributes from…"), or
  - show a progress counter on the form ("6 of 15 needed fields filled").
- **Steps:** add an activity of type Event to any initiative of a Lubricants plan.
- **Evidence:** `planner-12-activity-edit-at-field.png`. The 15-item panel text was recorded during the
  review run.
- **Files:**
  - `frontend/src/features/activities/ActivityForm.tsx:293`
  - `docs/product/field-catalogue.md` §4

### P-05 · Medium · Activity rows are hidden when the plan page opens

- **What happens:**
  - The plan tree opens with plan and campaign rows expanded and initiative rows collapsed, so no activity
    is visible. The activity rows are the ones that carry Created / Submitted / Declined and the missing
    fields.
  - To see them, the planner chooses **Expand all** on every visit; the choice is not remembered.
  - Each campaign or initiative row also stacks up to four chips in the Status column. For example "In
    progress · 2 of 4 activities approved", "Running", "All filled in" and "2 items below need attention"
    all appear together, which is hard to scan.
- **This follows the spec:** 005 FR-020 says "plan and campaign rows start expanded"
  (`specs/005-campaigns-and-initiatives/spec.md:267`).
- **Suggestion:** remember the expand state, or open to activity level.
- **Evidence:** `priya-03-plans_P_1001.png`.
- **Files:** `frontend/src/components/TreeTable/`, `frontend/src/features/plans/tree/PlanTree.tsx`

### P-06 · Medium · A published plan can be empty, and can become incomplete later without notice

- **What happens:**
  - **Publish** on a brand-new plan with no campaigns succeeded at once, with no confirmation. The only
    feedback was the toast "Plan '…' published".
  - Adding a campaign afterwards with Cluster and Region empty is allowed. The plan stays **Published**
    and shows "1 item below needs attention".
  - Both behaviours match 004 FR-024, which checks only at the moment of publishing
    (`specs/004-plans/spec.md:275`). Shell may still expect publishing to be a deliberate step, and a
    published plan to stay complete.
- **Suggestion:** the product owner decides whether to:
  - confirm before publishing,
  - warn on publish when the plan has no campaigns, and
  - show a "Published · changes since" notice.
- **Evidence:** `planner-04-publish-empty-plan.png`; the plan-tree rows were recorded during the review run.

### P-07 · Medium · Amounts are written two ways on the same page

- **What happens:**
  - On the activity page, the header shows "Unassigned planned cost **-$500.00**". The Ready-to-submit
    panel and the Line items section of the same page show "**500.00 USD**" and "-500.00 USD".
  - With the display currency set to EUR, cards show "1.1M EUR" where in USD they show "$1.2M".
  - The difference comes from decision 007-P4, which writes amounts in the line-item area with the currency
    code. To a reader it looks like two different numbers.
- **Steps:** open `/activities/A-1008` (Grand opening Shanghai).
- **Evidence:** `priya-05-activities_A_1008.png`.
- **Files:**
  - `frontend/src/features/activities/ActivityHeader.tsx`
  - `frontend/src/features/line-items/`
  - `frontend/src/lib/format.ts`

### P-08 · Low · "Shell attributes" heading appears twice

- **What happens:** the activity page's card titled "Shell attributes" opens with a second heading "Shell
  attributes". The edit form does the same.
- **Evidence:** `priya-05-activities_A_1008.png`, `planner-12-activity-edit-at-field.png`.
- **Files:**
  - `frontend/src/features/activities/ActivitySections.tsx:96`
  - `frontend/src/features/activities/ActivityForm.tsx:293`
  - the attribute section name added by `frontend/src/components/ShellAttributeFields/ShellAttributeFields.tsx`

### P-09 · Low · Toasts from the previous page cover the Save bar of the next one

- **What happens:** after adding a campaign and an initiative, their two toasts were still on screen when
  the New activity form opened. They sit at the bottom centre, on top of the form's sticky Save / Save and
  add another / **Cancel** bar, and hide Cancel.
- **Evidence:** `planner-07-activity-form.png`.
- **Files:** `frontend/src/components/Toast/Toast.module.css:1-4`

### P-10 · Low · "Ready to submit ✓" still shows on an activity that is already submitted

- **What happens:** a Submitted activity still shows a "Ready to submit" panel with a green tick. It
  should read "Everything needed is filled in" or be hidden once the activity is submitted.
- **Evidence:** `misc-01-a1004-own-submitted.png`.
- **Files:** `frontend/src/features/activities/` (Ready to submit panel)

---

## 2. Approver journey — work through approvals

What we did (as Daniel Okoro):
- Found Priya's new activity on Home ("Waiting for my approval") and on **Approvals › Waiting for me**.
- Opened the preview and chose **Approve**. The dialog said "This is the last level — the activity becomes
  Approved", and Approve completed it.
- Checked **Decided by me**.

Priya then saw it as In progress on the activity page, in the plan tree and in the bell. Levels, the
"on behalf of" wording, decline-needs-a-comment and the notifications all behaved as the spec says.

### A-01 · High (client decision) · The submitter can approve their own activity, from a primary button

- **What happens:**
  - On her own Submitted activity, Priya sees a blue **Approve** button in the page header, and Approve /
    Decline in the Approval section.
  - The dialog warns "You own this activity. You are not an approver of level 1. Your decision will be
    recorded as made on behalf of Daniel Okoro", and **Approve** completes it.
  - This follows decision D-020 and 008 FR-017 ("Any signed-in user MUST be able to approve",
    `specs/008-activity-approvals/spec.md:236`).
  - For a Shell marketing budget this is a segregation-of-duties risk.
- **Suggestion:** the client confirms the rule. Until roles arrive, at least:
  - do not offer Approve as the primary action to the activity's owner, and
  - keep it behind the ⋯ menu with the warning.
- **Steps:** as Priya, open `/activities/A-1004` (Trade magazine ads, Submitted, waiting for Daniel).
- **Evidence:** `misc-01-a1004-own-submitted.png`.
- **Files:** `frontend/src/features/activities/ActivityHeader.tsx`, `frontend/src/features/approvals/`

### A-02 · Medium · "In progress" means two different things

- **What happens:** for plans, campaigns and initiatives, "In progress" is *approval progress*. It even
  shows when nothing has been submitted: a campaign whose only activity is Created shows "In progress · 0 of
  1 activity approved". For an activity, "In progress" means *approved and its start date has passed*.
  These pages mix the two:
  - **All work's** single Status column shows Helix launch "In progress" and Partner co-marketing fund "In
    progress".
  - **The plan page's** summary card reads "Activities approved 4 of 7 · In progress".
- **Spec:** the rule is in `docs/product/domain-model.md:63-72` (§2.2) and §2.3.
- **Suggestion:** rename the roll-up to "Partly approved" or "0 of 3 approved", or show only the count.
- **Steps:** open `/all-work`, then a new campaign with one Created activity.
- **Evidence:** All work grid text and plan-tree rows, recorded during the review run.

### A-03 · Low · "Approval needed" notices stay unread after the decision is made

- **What happens:** after Daniel approved A-1011, his bell still lists "Approval needed — Priya Raman
  submitted 'Review activity …'" as Unread (`GET /api/notifications`: 2 unread). An approver who works
  from the Approvals page keeps a bell count for work already done.
- **Files:** `backend/internal/approvals/rounds.go:159, 258` (where the notices are sent)

### A-04 · Low · Approving one activity takes three clicks and there is no bulk approve

- **What happens:** each approval takes three clicks (open the preview, Approve, confirm Approve). That is
  fine for a few activities, but a team leader at year start may face dozens.
- **Note:** bulk actions are feature 012, which is deferred. Mention this to the client as known scope.

---

## 3. Administrator journey — Settings, people and teams

What we did (as Alex Morgan):
- New person: empty save, then a duplicate e-mail, then a valid save.
- Opened Deactivate on Priya, who owns records, to see what it says.
- Created a new team.
- New approval workflow: empty save.
- Set the EUR currency inactive.
- Opened a Shell attribute's details page.
- Tried to deactivate himself through the API.

Validation messages are clear. Some examples:
- "2 problems need your attention · Enter a name. · Enter an e-mail address."
- "priya.raman@northstar.demo already belongs to Priya Raman."
- "You can't deactivate yourself — ask someone else to do it."
- The Deactivate dialog lists everything the person owns.

### S-01 · Medium · The same action behaves differently across Settings sections

- **Making something inactive:**
  - **Currencies and models:** ⋯ › **Set inactive** applies **at once**, with no confirmation and no Undo.
    EUR, used by 2 line items, went inactive in one click with only the toast "Currency 'Euro' set
    inactive".
  - **People:** **Deactivate** opens a confirmation that lists everything the person owns.
  - **Approval workflows:** they are confirmed too (008 FR-007).
  - The spec for Settings asks only for "Set inactive" (`specs/002-settings-and-shell-attributes/spec.md:207`),
    so this matches each spec, but users meet two behaviours for one idea.
- **Filter pills:**
  - Models, Activity types, Currencies and Finance budget items show **Active n · Inactive n · All n**,
    with Active first and counts.
  - People, Teams and Approval workflows show **All · Active · Inactive**, with All first and no counts.
- **Count lines:** "4 of 5", "6 people" and "3 teams".
- **Create / edit pattern:**
  - Side panel: campaigns, initiatives, people, teams, models, currencies, workflows.
  - Full page: plans, activities, line items, Shell attributes.
  - All of these follow their specs, but the UX revamp should choose one rule and explain it.
- **Evidence:** `admin-07-deactivate-eur.png`, `alex-10-settings_people_U_1002.png`.
- **Files:**
  - `frontend/src/features/settings/currencies/CurrenciesPage.tsx:162-166` (pills with counts)
  - `frontend/src/features/people/PeoplePage.tsx:164-165` (pills without counts)
  - `frontend/src/features/settings/currencies/CurrencyDetails.tsx`

### S-02 · Low · Labels in the person details panel

- **What happens:**
  - The section titled "Owns and approves" lists only what the person owns ("Owner of 2 plans, 2 campaigns,
    3 initiatives, 8 activities and 7 line items"). It says nothing about the approval workflows the person
    is a level approver in.
  - "People are never deleted, so the history keeps their name" is helpful, but the History it mentions
    does not exist yet (011).
- **Evidence:** `alex-10-settings_people_U_1002.png`.
- **Files:** `frontend/src/features/people/`

---

## 4. Across the module — consistency, navigation, totals, responsiveness, demo data

### X-01 · Medium · Phone screens scroll sideways

- **What happens:**
  - At 390 px width these pages scroll sideways: plan page **709 px**, activity page **460 px**, Settings ›
    People **40 px**.
  - The plan tree's Status, Budget and Planned cost columns go off screen.
  - The design system says tables "may scroll sideways inside their card", not the whole page
    (`docs/product/design-system.md` §2, "Width").
  - 006's own check asks for "no horizontal page scroll" at 390 px (`specs/006-activities/tasks.md:177`).
- **Evidence:** `misc-07-phone_plans_P_1001.png`.
- **Files:** `frontend/src/components/TreeTable/TreeTable.module.css:1-4, 54-56`. The `.scroll` wrapper
  exists, but the card that contains it does not stop the table from widening the page; the name cell also
  has a 260 px minimum width.

### X-02 · Medium · All work: the totals do not visibly add up

- **What happens:**
  - Unassigned planned cost rolls up **only positive balances** (rule in
    `specs/010-all-work-views/spec.md:209`), but the grid does not say so.
  - **FY2026 Mobility China:** Planned cost 190,000.00 − Line items 40,500.00 = 149,500.00, but Unassigned
    shows **150,000.00**.
  - **Station promotions:** 40,000.00 − 40,500.00 shows **0.00**.
  - A finance reviewer will read this as an arithmetic error.
- **Suggestion:** label the column "Unassigned (positive balances)", or add an ⓘ tip, or show the negative
  activities in the total row.
- **Evidence:** `GET /api/all-work?year=2026`, rows P-1002, C-1003, I-1005.

### X-03 · Medium · View history, the Activity log and Recent changes lead to "Coming soon"

- **What happens:**
  - **⋯ › View history** on activities opens `/activity-log?recordType=activity&recordId=…`, which shows
    "Coming soon". This was clicked. The plan, campaign, initiative and line-item menus link to the same
    page; the code was read, but those links were not clicked:
    - `frontend/src/features/plans/PlanPage.tsx:49`
    - `frontend/src/features/campaigns/RecordDetails.tsx:93`
    - `frontend/src/features/line-items/LineItemPage.tsx:103`
    - `frontend/src/features/activities/useActivityActions.tsx:35`
  - The **Activity log** sidebar item shows "Coming soon".
  - Home's **Recent changes** card shows "Coming soon". Its API `GET /api/home/recent-changes` answers 404.
  - These are expected until 011 is merged, but in a client demo they are dead ends reached from normal
    menus.
- **Files:**
  - `frontend/src/app/navigation.ts:66-89`
  - `frontend/src/features/home/RecentChangesCard.tsx`
  - `specs/009-home-dashboard/spec.md:237` (FR-015)

### X-04 · Medium · The demo data does not match its description

- **What happens:**
  - `docs/product/demo-users-and-data.md:43` says "The recycle bin contains one deleted initiative … and the
    activity log contains about two weeks of back-dated history".
  - A fresh seed loads **neither**: `GET /api/activity-log` returns `{"items":[],"total":0}`, and so does
    `GET /api/recycle-bin`.
  - The seed for this is 011 task T019, which is not built on either branch.
- **Files:** `backend/seed/` (no `history.go`), `docs/product/demo-users-and-data.md:43`

### X-05 · Medium · Task checkboxes do not show what is built

- **What happens:**
  - Features 002–010 have **153 open task boxes** (001 has 9 more), but much of that work exists and runs. Some examples:
    - 002 T005: `backend/internal/platform/money/money.go`
    - 004 T006: `backend/internal/plans/model.go`
    - 003 T033: `frontend/src/components/Combobox/`
    - 004 T036: `frontend/src/components/TreeTable/`
    - 007 T022: `backend/internal/lineitems/provided.go`
  - What is genuinely open is mostly the per-feature end-to-end files, the usability sessions (SC-002), and
    the "set to Done in feature-map" steps. `docs/sdd/feature-map.md` still says "In progress" for
    001–010 and "Draft" for 011.
  - Before the client signs off, the boxes should match the code, or the traceability evidence will be
    disputed.
- **Files:** `specs/00{2..9}-*/tasks.md`, `specs/010-all-work-views/tasks.md`, `docs/sdd/feature-map.md`

### X-06 · Low · The contract lists endpoints that answer 404

- **What happens:**
  - `backend/api/openapi.yaml` describes operations that the API does not serve yet, so the web app's
    generated types include them:
    - `GET /api/activity-log/summary` (line 4297)
    - `POST /api/recycle-bin/empty` (4331)
    - `/api/bulk-actions/*` (4413–4517)
    - `/api/plans/{planId}/imports*` (4542–4670)
    - `GET /api/home/recent-changes` (3993)
  - The contract test skips them through a `notYetServed` list
    (`backend/internal/app/app_test.go:186-197`).
- **Why it matters:** this is acceptable during development. It should be documented for anyone
  integrating with the API.

### X-07 · Low · Home: the same problem has different wording and layout

- **What happens:**
  - **Wording:** Needs your attention says "1 item below needs attention" and My activities says "1 line
    item needs attention" for the same activity (Launch film).
  - **Link:** "Show fields" appears after "2 items below need attention", where there are no fields to show.
  - **Layout:** in My activities, a row with problems puts the amount mid-row and the chips on a new line,
    while the other rows align amount and chip on the right.
- **Evidence:** `priya-01-home.png`.
- **Files:** `frontend/src/features/home/`

### X-08 · Low · "Nothing" is shown three ways

- **What happens:**
  - The plan page shows "Planned cost — · Nothing planned yet".
  - The Plans card shows "Planned cost —".
  - Home shows "Planned cost vs plan budget $0.00 of $500K".
  - All three are for FY2027 Global Brand, which has no activities.
- **Files:** `frontend/src/features/plans/PlanSummaryCards.tsx`, `frontend/src/features/home/`

### X-09 · Low · The sign-in page logs a 401 error in the browser console

- **What happens:** the session check on the sign-in page logs "Failed to load resource: 401". It is
  harmless, but it adds noise to monitoring once the app is hosted.
- **Files:** `frontend/src/app/SessionProvider.tsx`

---

## 5. Spec requirements not built (current `plan-module`)

| Feature | Requirement | State on `plan-module` | Note |
|---|---|---|---|
| 011 | All of it: Activity log page (US2), History sections (US3), Recycle bin page with restore, restore with parents, delete permanently, empty (US1, US4, US5) | "Coming soon" pages | Mostly built on `lane-011` (section 6) |
| 004 | FR-031 View history | Opens the 011 placeholder | Depends on 011 |
| 006 | FR-032 View history | Opens the 011 placeholder | Depends on 011 |
| 005, 007 | View history of campaign, initiative, line item | Links to the 011 placeholder (`RecordDetails.tsx:93`, `LineItemPage.tsx:103`) | Depends on 011 |
| 009 | FR-015, FR-016 Recent changes card, View activity log | "Coming soon"; `GET /api/home/recent-changes` returns 404; 009 T013 open | Depends on 011 |
| 001 / demo | Seeded log history and bin item (demo-users-and-data.md) | Not seeded | 011 T019 |
| 007 | FR-025 "From finance (coming later)" | Placeholder by design | Finance module is out of scope |
| 012 | Bulk actions and import | Contract only (404) | Deferred, P3 |
| 013 | BCPIR and SP&A views | Not started | Waiting on Q-001 |
| All | End-to-end files listed per feature (e.g. `plans.spec.ts`, `activities.spec.ts`, `approvals.spec.ts`, `all-work.spec.ts`) | Only `smoke`, `core-flow` and `module-rest` exist | Core journeys are covered; per-story and axe coverage per page is not |
| All | Success-criteria checks (moderated usability sessions SC-002, timed runs) | Not done | e.g. 001 T085, 004 T065, every TC99 |

Everything else we exercised behaved as its FR says. That covers: plans list and plan page, create, edit,
publish; campaign and initiative panels and row menus; activity create, edit, Ready to submit with links to
the field; line items, spread and unassigned balance; the submit, approve and decline dialogs and the bell;
Home cards and their links (009 FR-007); All work grid and CSV/XLSX export; Settings lists; people and
teams; profile display currency (003 FR-023).

---

## 6. Feature 011 — Activity log and recycle bin: status of branch `lane-011`

**The branch:**
- Two commits, both dated 19 Oct 2026:
  - `1b90d34` feat(011): Activity log and Recycle bin pages (web)
  - `29e603f` feat(011): activity log and recycle bin (API) — set aside, not merged
- It starts from `2cc4415` (the frontend foundation), so it is **46 commits behind `plan-module`**.
- It changes 61 files (+8,874 / −192).
- On its own, the branch's backend is green: `go vet ./...` and `go test ./...` pass, run in a temporary
  copy of the repository.
- Its `tasks.md` still has only T001 and T002 ticked.

### What is built on `lane-011`, task by task (`specs/011-activity-log-and-recycle-bin/tasks.md`)

| Task(s) | Status on `lane-011` | Evidence |
|---|---|---|
| T001, T002 contract and web types | Done (already on `plan-module`) | ticked |
| T003 test fixtures for the contract change | Partly | `samples/handler_test.go`, `ComingSoonPage.test.tsx` |
| T004 `parent_deleted_permanently` error | Done | `backend/internal/platform/errs/errs.go:38-40, 204` |
| T005 `historyIncludeInside` preference | **Not built** | in the contract and a web test only; no Go code in `people` |
| T006, T007 history model types, service refactor | Done | `history/model.go`, `history/service.go` |
| T008–T010 log query and `GET /api/activity-log`, summary | Done | `history/query.go`, `query_test.go`, `handler.go:34` |
| T011 record history with items inside | Done | `TestQueryLog_RecordWithItemsInside_FR012_FR013` |
| T012–T018 bin list, restore preview, restore with parents, delete permanently, empty | Done | `history/bin.go`, `bin_test.go`, `handler.go:36-38` |
| T019 demo history seed | **Not built** | no `backend/seed/history.go` |
| T020 navigation and routes | Done | `navigation.ts`, `router.tsx` |
| T021 error kind in the web app | Done (already in the foundation) | `frontend/src/api/errors.ts:34, 55` |
| T022 `formatWeekdayDate` | Done | `frontend/src/lib/format.ts` |
| T023 Home pieces "if 009 not built" | Not needed | 009 is on `plan-module` |
| T024–T030 Activity log page | Done | `frontend/src/features/activity-log/*` (filters are in `LogFilterBar.tsx`, not `LogFilters.tsx` as the task names it) |
| T031 History section component | Done | `HistorySection.tsx`, `showHistory.ts` |
| T032–T035 place the History section on the plan, campaign/initiative, activity and line-item pages | **Not built** | no 004–007 page uses `HistorySection` or `showHistory` |
| T036–T044 Recycle bin page, restore, restore with parents, delete permanently, empty | Done | `frontend/src/features/recycle-bin/*` |
| T045, T046 end-to-end `recycle-bin.spec.ts`, `activity-log.spec.ts` | **Not built** | `frontend/e2e/` has only `fixtures.ts`, `smoke.spec.ts` |
| T047 package comment and OpenAPI tag | Package comment done; tag not checked | `history/model.go:1-9` |
| T048 feature map status | Not done | `docs/sdd/feature-map.md` still "Draft" |
| T049 final full run | Not done | — |
| TC00 remove served routes from `notYetServed` | Done on the branch | conflicts on merge (below) |
| TC99 success criteria SC-003, SC-004, SC-007 | Not done | — |

**What remains after the merge:**
- T005, T019, T032–T035, T045–T049 and TC99.
- Ticking the boxes in `tasks.md`.
- Wiring Home's Recent changes (009 FR-015, task 009 T013, `GET /api/home/recent-changes`), which 011
  makes possible.

### What merging `lane-011` into `plan-module` conflicts with

We trial-merged in a temporary copy of the repository. **5 files conflict:**

1. **`backend/internal/history/service.go`: the restore logic. This is the risky one.** Since the branch
   point, `plan-module` added several things to restore:
   - `RestoreContext.DeletedAt` / `DeletedBy`;
   - the `AfterRestore` hook of each record kind. It is used by `backend/internal/lineitems/kinds.go:44`,
     and `backend/internal/approvals/hooks.go:106-112` relies on the delete time and actor it carries.
     Together they enforce 007 FR-033 and 008 FR-023, which decide what happens to approvals when
     something is restored;
   - `SnapshotsInBin`, used by `backend/internal/settings/usage.go:220`;
   - a `putSnapshots` that returns one record.

   `lane-011` instead moved restore into `restoreEntry`, with `putSnapshots` returning a list, no
   `AfterRestore` call, and no `DeletedAt` / `DeletedBy`. Its new Restore-with-parents in `bin.go` builds
   on that version. What happens with the simple choices:
   - **Taking `lane-011`'s file breaks the build:** "`s.history.SnapshotsInBin undefined`" in
     `settings/usage.go:220`, and in `history/service_test.go:921, 947`.
   - **Taking `plan-module`'s file** loses what `bin.go` relies on.

   The fix is a hand merge, keeping `AfterRestore`, `DeletedAt` / `DeletedBy` and `SnapshotsInBin` inside
   `restoreEntry`. Then re-run the 007 and 008 restore tests together with 011's `bin_test.go`.
2. **`backend/internal/app/app_test.go`:** the `notYetServed` list. `plan-module` lists the 011 routes as
   not served, while `lane-011` still lists the 010 routes. Resolve by removing both groups.
3. **`frontend/src/app/navigation.ts`:** the "Coming soon" sections conflict with `plan-module`'s Approvals
   badge and `alsoCurrentOn` changes.
4. **`frontend/src/app/router.tsx`:** the route list.
5. **`frontend/src/features/placeholders/ComingSoonPage.test.tsx`.**

**Also check after merging, even though they merge cleanly:**
- **Compile and test the web app.** The `lane-011` web code was never compiled or tested against
  `plan-module`'s newer components, for example the Home and SummaryCard additions of 009 and
  `recordLinks`. Run `npm run check`.
- **Point "View history" at the History section.** The ⋯ › View history items of 004–007 still go to
  `/activity-log?recordType=…`; T032–T035 should make them open the record's History section.
- **Run the full checks** of 011 T049: backend `go test ./...`, frontend `npm run check` and
  `npm run test:e2e`.

---

## Appendix — how to reproduce

- **Data:** in `backend/`, run `go run ./cmd/seed -data <temp folder> -reset`. Then start the API with
  `DATA_DIR=<temp folder> APP_PORT=18102 go run ./cmd/api`. In `frontend/`, start the web app with
  `NORTHSTAR_API_URL=http://127.0.0.1:18102 npx vite --host 127.0.0.1 --port 14102 --strictPort`.
- **People:**
  - Priya Raman (`priya.raman@northstar.demo`), planner
  - Daniel Okoro, approver for Lubricants
  - Alex Morgan, administrator and level-2 approver for China
  - The password is `northstar`.
- **Records created during the review:** plan P-1004 "Review plan …" with C-1005, I-1009, A-1011 and
  L-1009; the throwaway activity A-1012, deleted to the bin; team "Review team …". EUR was set inactive and
  then reactivated.
- **Review scripts:** Playwright crawl, planner, submit, admin and misc journeys. They were run from the
  review's temporary folder and are not part of the repository.
