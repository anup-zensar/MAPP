# Feature Specification: QA Findings — Fixes from Exploratory QA of develop@85a3d495

**Feature Branch**: `feature/bug-fixes` (spec folder `specs/038-qa-findings`)

**Created**: 2026-10-29

**Status**: Draft

**Input**: Exploratory browser QA of `develop` at `85a3d495` (29 Oct 2026; four testers, Playwright and axe, each on
its own copy of the demo data) reported **54 unique findings** — 1 critical (C-1), 5 high (H-1 – H-5), 18 medium
(M-1 – M-18) and 30 low (L-1 – L-30) — and three gaps in its security checks (sign-in rate limit; security headers;
the same-origin check, which is L-14). Each finding was traced to the code before it was specified here
([research.md](research.md) gives file and line). The product owner decided four points on 2026-10-29: who sees
GSAP transactions that are not linked (H-4), approving while line items exceed an activity's planned cost (M-18),
security hardening (sign-in limit, headers, referring page) and the date format everywhere (L-18) — recorded as
D-359 – D-362. Smaller calls made while specifying are D-363, confirmed by the product owner on 2026-10-29. The
red asterisk on mandatory fields (User Story 18) is a product owner request of the same day (D-364).

**Amends**: [003](../003-people-and-teams/spec.md) FR-024 (clarified) · [009](../009-home-dashboard/spec.md) FR-001
(wording) · [011](../011-activity-log-and-recycle-bin/spec.md) FR-004 · [013](../013-bcpir-and-spa/spec.md) FR-017 ·
[014](../014-access-control/spec.md) Table B and the approve check order ·
[015](../015-financial-transactions/spec.md) FR-026, FR-028, Q-015c · [032](../032-purchase-requests/spec.md)
FR-037 · [036](../036-planning-dashboards-and-export/spec.md) FR-007 (restored; its data model and contract
corrected) · [037](../037-overspend-control/spec.md) FR-006 and FR-019. The amendment index at the end lists each one.

**References**: QA report "SP&A One QA Report" (54 findings, security checks table) and its detail notes
(findings-area2, findings-area4, reproduction scripts area1–area4) · constitution 4.6 VI (a failing test first for
every bug), VII (scope isolation, input at the trust boundary), F7 (content security policy) · design system
§3.1 (contrast on grey), §4 (after delete), §10 · domain model §4.3, §4.4 · decisions D-016, D-061, D-301,
D-322, D-346 – D-349, D-352.

Demo people (014 seed): Alex Morgan (Administrator), Priya Raman (Planner, Lubricants Europe), Daniel Okoro
(Approver), Carlos Mendes (Global planner), Mei Lin (Planner, Mobility China), Sara Novak (Viewer), and the
Requisitioner.

## Clarifications

### Session 2026-10-29

- Q: Who sees GSAP transactions that are not linked to any line item (H-4)? → A: Only people who may import GSAP
  extracts (Administrators by default) — the product owner confirmed the behaviour already built. Everyone else
  sees no Not linked tab, count or Home link. Because only those people can see what there is to link, "Link
  financial transactions" is granted only together with "Import GSAP extracts": the built-in Global planner and
  Planner roles lose it, and a role holding it alone is refused on save (D-359).
- Q: What may a person do with links they can see (H-5)? → A: Linking a not-linked transaction needs both
  permissions and a target line item the person can see; unlinking a manual link needs "Link financial
  transactions" in the plan of the linked line item. A transaction or line item the person cannot see answers
  exactly as a missing one, before any permission, version or state check (D-359).
- Q: Should approving be refused while an activity's line items exceed its planned cost (M-18)? → A: Both
  behaviours are offered: Settings › Overspend control decides between **Warn the approver** (default — today's
  behaviour, now with a prominent warning) and **Refuse approval** (D-360).
- Q: Which sign-in limit (security)? → A: 5 failed attempts for one e-mail within 15 minutes, or 20 from one
  client within 15 minutes, refuse further attempts for 15 minutes; success clears the count (D-361).
- Q: Does the date format apply to day headings and month labels (L-18)? → A: Yes, everywhere; 003 FR-024 wins
  over 011 FR-004's example (D-362).
- Q: What does a negative "Not yet scheduled" become (M-2)? → A: It is never negative; the excess of line items
  over planned cost gets its own row, so the equation of 013 FR-017 still holds (D-363).
- Q: What happens to typed values when the session ends (M-4)? → A: Drafts are not kept in the browser
  (constitution F7 allows only interface preferences there); the person is told on the sign-in page that the
  changes were not saved and returned to the same address after signing in (D-363).

## User Scenarios & Testing *(mandatory)*

Stories are grouped by area. P1 = the critical and high findings and security hardening; P2 = the medium
findings; P3 = the low findings.

### User Story 1 - A purchase-request change never frees money before it is approved (Priority: P1)

Priya's purchase request PR-1002 for 15,800.00 GBP on line item L-1003 (planned cost 15,800.00 GBP) is approved.
She lowers the line to 12,000.00 and chooses Submit changes. Until Daniel decides, the line item keeps
15,800.00 reserved, so a second request for 3,800.00 is refused; if Daniel declines (or Priya recalls), PR-1002
returns to 15,800.00 and the line item is never committed above its planned cost.

**Why this priority**: C-1 — money can be committed twice; the purchase request is what commits Shell's money.

**Independent Test**: Run QA's sequence (approve, lower and submit changes, raise a second request, decline):
the second request is refused naming the available amount 0.00 GBP; after the decline Available to request is
0.00, never negative.

**Acceptance Scenarios**:

1. **Given** an approved purchase request whose line reserves 15,800.00 on a line item planned at 15,800.00,
   **When** the line is lowered to 12,000.00 and the change submitted, **Then** the line item's Total requested
   stays 15,800.00 and Available to request stays 0.00 while the change is in review.
2. **Given** that change in review, **When** anyone raises or saves another purchase request line on the same
   line item for 3,800.00, **Then** it is refused with the existing "only <available> left to request" problem.
3. **Given** that change, **When** it is approved, **Then** 3,800.00 becomes available; **When** it is instead
   declined or recalled, **Then** the approved 15,800.00 stays reserved and nothing becomes negative.
4. **Given** a change that raises a line from 10,000.00 to 12,000.00, **When** it is submitted, **Then**
   12,000.00 is reserved (the larger amount) and refused if the line item cannot hold it.
5. **Given** a change that removes a line or adds one, **Then** a removed line keeps its approved amount reserved
   until the change is approved, and an added line reserves its amount from its first save.

---

### User Story 2 - Financial transactions follow the person's scope (Priority: P1)

Mei, a Planner for Mobility China, sees and changes only what her scope holds. Transactions not linked to any line
item are worked by Alex, who imports GSAP extracts; Mei sees no Not linked tab and no link actions she cannot use.

**Why this priority**: H-5 lets a write reach a record the person cannot see and lets them enumerate other teams'
line items; H-4 offers a permission that cannot be used.

**Independent Test**: As Mei, try to link FT-1004 (not linked), unlink a link of another market, and link to an
out-of-scope and to a non-existent line item: each answer equals the answer for a missing record. As Alex, link
FT-1004 to L-1007 and unlink it.

**Acceptance Scenarios**:

1. **Given** Mei, **When** she opens Financials or Home, **Then** there is no Not linked tab, count or "n
   transactions not linked" link, and opening FT-1004 by its address shows "This GSAP transaction no longer
   exists" as for a missing one.
2. **Given** Mei, **When** she asks to link FT-1004 (which she cannot see), **Then** the answer is identical to
   linking a transaction that does not exist, and nothing is saved.
3. **Given** Mei and a manual link to a line item of another market, **When** she asks to unlink it (with any
   version), **Then** the answer is identical to a missing transaction, and no person's name appears in it.
4. **Given** a person who holds both permissions with a scope that leaves out plan P-1004 (custom role), **When**
   they link a not-linked transaction to a line item of P-1004 or to an ID that does not exist, **Then** both
   answers read the same "No line item <ID> — check the ID" problem.
5. **Given** a transaction panel, **Then** Link to a line item and Unlink appear only when the server says the
   person may use them.
6. **Given** Settings › Roles, **When** an administrator saves a role with "Link financial transactions" but
   without "Import GSAP extracts", **Then** the save is refused: "Linking financial transactions needs Import
   GSAP extracts too — only people who import see transactions that are not linked."; the built-in Global
   planner and Planner roles no longer hold it.

---

### User Story 3 - The plan workbook checks a full-size file in seconds (Priority: P1)

Priya uploads a workbook of 5,000 activity rows (within the documented 5 MB and 5,000-row limits). Check file
answers in seconds, other people's saves are not held up while the file is read, and the problem list stays
readable even when a row has a very long value.

**Why this priority**: H-2 — the main import use case fails at its documented size and one legal upload holds
every other save for minutes.

**Independent Test**: Check files of 250, 1,000 and 5,000 valid rows and one with 1,500 bad rows of
4,000-character descriptions; measure duration and server work per row.

**Acceptance Scenarios**:

1. **Given** a valid 5,000-row workbook, **When** Priya chooses Check file, **Then** the result appears within the
   time of SC-002 and Import completes within the same bound.
2. **Given** a workbook with a 4,000-character value in a problem, **Then** the problem shows at most 200
   characters of it followed by "…".
3. **Given** a 5 MB workbook, **When** uploaded through the packaged web server, **Then** it is accepted (not
   refused as too large before reaching SP&A One).
4. **Given** any workbook, **Then** the check finds exactly the problems it found before this feature (same rows,
   same messages).

---

### User Story 4 - A dashboard link always opens a dashboard (Priority: P1)

Alex opens a saved link `/dashboards/executive?currency=JPY` after JPY was deactivated, or
`/dashboards/regional?team=T-9999` for a market that no longer exists. The dashboard opens with the default
currency or market, the filter bar shows what is used, and the address is corrected.

**Why this priority**: H-1 — saved and shared links break for good once a currency or market is deactivated.

**Independent Test**: Open both dashboards with unknown, deactivated and malformed currency and market values;
each shows the dashboard with its filter bar.

**Acceptance Scenarios**:

1. **Given** an unknown or inactive currency in the address, **When** a dashboard opens, **Then** it uses the
   person's display currency (or US dollars) and the address shows that currency.
2. **Given** an unknown or out-of-scope market on the Regional dashboard, **Then** it uses the person's default
   market as when no market is given.
3. **Given** the same address, **When** a .pptx or .pdf export is made, **Then** it uses the same fallback.

---

### User Story 5 - Deleting a purchase request returns with Undo (Priority: P1)

Priya deletes her purchase request from its page. She lands on the Purchase requests page with "Purchase request
'<name>' deleted" and Undo, as for every other record.

**Why this priority**: H-3 — the person is told the record they just deleted "no longer exists — someone may have
deleted it", and has no Undo.

**Independent Test**: Delete a purchase request from its page; check the landing page, the toast and Undo, and
that no request for the deleted record is made. Repeat for a plan and a line item (L-3).

**Acceptance Scenarios**:

1. **Given** a purchase request Priya may delete, **When** she confirms Delete on its page, **Then** she is on
   the Purchase requests page (or the line item she came from, when she has no access to that page) with the
   toast and Undo; Undo restores it.
2. **Given** a plan or line item deleted from its page, **Then** no request for the deleted record is sent after
   the delete.

---

### User Story 6 - Sign-in and every page are hardened (Priority: P1)

Before SP&A One leaves a developer's machine, guessing passwords is limited, pages cannot be framed or run
foreign scripts, change requests from other sites are refused even without an origin header, and unusual
characters never cause a server error.

**Why this priority**: The QA security table's two gaps and L-14; M-3 makes the unauthenticated sign-in answer
with a server error.

**Independent Test**: 6 wrong passwords for one e-mail; inspect the protections on the web pages and the server's
answers; send a change request with only a foreign referring page; send a NUL character in every text field and
record ID.

**Acceptance Scenarios**:

1. **Given** 5 failed sign-ins for one e-mail within 15 minutes, **When** a 6th attempt arrives (even with the
   right password), **Then** it is refused with "Too many sign-in attempts. Try again in 15 minutes." and the
   remaining wait, and the sign-in page shows that message.
2. **Given** that lock, **When** 15 minutes have passed since the 5th failure, **Then** the right password signs
   in at the first try and the count starts again.
3. **Given** 3 failures then a successful sign-in, **Then** the next failure counts from 1.
4. **Given** 20 failed attempts from one client within 15 minutes for any e-mails, **Then** further attempts from
   that client are refused in the same way for 15 minutes; other clients are not affected.
5. **Given** the demo sign-in page, **When** a demo person is chosen, **Then** it counts exactly like typing
   their e-mail and the demo password.
6. **Given** any page of the web app or any server answer, **Then** it may not be shown inside a frame of another
   site, sends no full referring address to other sites, is not content-sniffed, and the page runs only its own
   scripts (the theme script by its fingerprint) with no browser policy violation in any journey.
7. **Given** a change request with no origin information and a referring page of another site, **Then** it is
   refused like a request from another site; with neither it is handled as today.
8. **Given** a NUL or other control character in any text field (sign-in included), **Then** the answer is a
   field problem "Remove the control characters from this text." naming the field; in a record ID it is "not
   found"; never a server error, and no record ID is used up.

---

### User Story 7 - Approving with line items above planned cost: warn or refuse (Priority: P2)

Daniel approves a purchase-request round while the activity's line items are 25,000.00 USD above its planned cost.
By default he sees, before approving, "Line items of activity '<name>' are 25,000.00 USD above its planned cost";
if Alex has set Settings › Overspend control to refuse, Approve is refused with that reason.

**Why this priority**: M-18 — undermines 037's purpose, but a workaround (declining) exists.

**Independent Test**: With the demo data, add line items above A-1003's planned cost; open the approval as
Daniel with each setting.

**Acceptance Scenarios**:

1. **Given** Settings › Overspend control, **Then** below "New budgets start with" it offers "When an activity's
   line items exceed its planned cost, approving it or its purchase requests": **Warn the approver** (default) and
   **Refuse approval**, changed by people who may change Settings, logged "Approval over planned cost: Warn →
   Refuse".
2. **Given** Warn and an open round (activity, purchase request or its change) whose activity's line items exceed
   its planned cost, **When** the approver opens it, **Then** the approval section and the Approve dialog show the
   warning per activity with the overage in the plan currency; approving succeeds and its activity-log entry adds
   "while line items were <overage> above planned cost".
3. **Given** Refuse, **Then** Approve is shown disabled with "Line items of activity '<name>' are <overage> above
   its planned cost — ask the owner to lower them, or decline."; an approval sent anyway is refused with that
   message and nothing changes; Decline and Recall stay available.
4. **Given** the line items are brought back within the planned cost, **Then** neither warning nor refusal shows.

---

### User Story 8 - Money figures add up (Priority: P2)

Every grid of money adds up, and when line items exceed planned cost the screen says why monthly totals do not
match planned cost.

**Why this priority**: M-1 and M-2 show figures that contradict each other; L-2 is the same rule unexplained.

**Independent Test**: A-1011 in USD and EUR display: every row and column adds up. A-1003 with line items above
planned cost: SP&A view, Budget & cost, dashboards and Budget by month.

**Acceptance Scenarios**:

1. **Given** the spread grid in any display currency, **Then** each row's months add up to its total, each
   column's cells to its column total, the column totals to the grand total, and each row total equals the line
   item's planned cost in that currency.
2. **Given** an activity whose line items exceed its planned cost by 15,000.00, **When** the SP&A view opens,
   **Then** "Not yet scheduled" is never negative and a row "Line items above planned cost" shows 15,000.00, and
   the quarters, Not yet scheduled and that row reconcile to the planned cost as stated beside them.
3. **Given** the same, **Then** All work Budget & cost, the dashboards' monthly chart and Budget by month say "1
   activity has line items above its planned cost, so its months add up to more than its planned cost."

---

### User Story 9 - Purchase-request screens say what will happen (Priority: P2)

**Why this priority**: M-7 – M-10: view-only people see dead buttons, the counter disagrees with the page, the
submission check cannot say who decides, and Submit changes starts a round silently.

**Independent Test**: Open a purchase request as Viewer, Requisitioner and owner; edit, count, submit and submit
changes.

**Acceptance Scenarios**:

1. **Given** a person who may change nothing on a purchase request, **Then** one View only note shows and no
   disabled change action is listed (D-352); a Requisitioner who may not edit is told her role does not include
   editing, not to "edit and choose Submit changes".
2. **Given** a needed-to-submit attribute empty (Invoice approver name), **Then** the edit form's counter and the
   page's "fields needed" agree.
3. **Given** a workflow whose level takes its approver from the MoA approver field, **When** Priya opens Submit,
   **Then** the dialog names that person for the level.
4. **Given** an approved purchase request edited, **When** Priya chooses Submit changes, **Then** a confirmation
   first says a change approval round starts, with which workflow and approvers, and that amounts stay reserved
   until it is approved; only Confirm saves.

---

### User Story 10 - Screens follow access and keep the person's work visible (Priority: P2)

**Why this priority**: M-6, M-12 and M-4 contradict the design system (actions hidden per access, all problems at
once, no surprise loss).

**Independent Test**: Sara (Viewer) on a plan, campaign and initiative; empty Save on New global budget and New
Shell attribute; end the session with a form open.

**Acceptance Scenarios**:

1. **Given** Sara, **Then** "Edit budget by month" is not offered on plans, campaigns or initiatives.
2. **Given** an empty New global budget form, **When** Save is chosen, **Then** Name, Amount and Owner problems
   show together; likewise every Shell attribute problem (options, help-text length, duplicate label) at once.
3. **Given** a form with unsaved changes when the session ends, **Then** the sign-in page says "Your session
   ended, so your changes in '<title>' weren't saved." and after signing in the person is back at the same
   address, fragment included (L-28).

---

### User Story 11 - Home, All work and exports agree (Priority: P2)

**Acceptance Scenarios**:

1. **Given** Priya owns 9 activities in FY2026 and FY2027 plans, **When** she follows Home's "Activities I own",
   **Then** All work opens on the years of those plans and lists 9.
2. **Given** a partly approved campaign, **When** All work is exported, **Then** its status reads "Partly
   approved", as on screen.

**Why this priority**: M-5, M-11 — figures that differ between pages break trust in reconciliations.

**Independent Test**: Compare Home's figure with the list it opens; compare export and grid statuses.

---

### User Story 12 - Imports say exactly what is wrong (Priority: P2)

**Acceptance Scenarios**:

1. **Given** a rate file with a row `=HYPERLINK("x"),USD,EUR,…`, **Then** that row is listed under Problems and the
   other rows are checked (M-13).
2. **Given** an empty rate file, **Then** "This file is empty."; given several missing columns, **then** every
   missing column is named; a rate of 1000001 gets the "too large" message; a rate date before 1 Jan 2000 is a
   problem (L-21).
3. **Given** a GSAP extract separated by semicolons, **Then** it is read like a comma-separated one (L-13).

**Why this priority**: M-13 refuses a whole file for one row with a misleading message.

**Independent Test**: QA's rate files and a semicolon GSAP extract.

---

### User Story 13 - Layout and accessibility hold everywhere (Priority: P2)

**Acceptance Scenarios**:

1. **Given** a 120-character plan or campaign name without spaces, **Then** Plans list and the SP&A view fit the
   window at 390, 768 and 1440 px with no sideways page scroll (M-14, M-15).
2. **Given** the import problems table, **Then** it can be focused and scrolled by keyboard (M-16).
3. **Given** the split links, the Regional dashboard empty-state hint and the purchase-request form counter on a
   grey surface, **Then** their contrast is at least 4.5:1 (M-17).

**Why this priority**: WCAG 2.2 AA failures (1.4.3, 1.4.10, 2.1.1).

**Independent Test**: axe in light and dark mode on those screens; page width at 390 px.

---

### User Story 14 - Wording and small interactions are consistent (Priority: P3)

**Acceptance Scenarios** (one per finding; texts in FR-039 – FR-053):

1. Activity conflict: the message adds "Your changes are still here. Save again to replace their changes, or
   Cancel to see theirs." (L-1).
2. Custom spread and budget-by-month problems name their month in the error summary (L-4).
3. Activity menus say "Move to…" (L-6); bulk submit says "Submit 1 activity" (L-9).
4. Pasting "1000⇥2000⇥3000" into March fills March, April and May; "$2,500.50" is accepted as 2500.50 (L-7).
5. Global budgets: never "$1000T"; "26%" used; the can't-delete dialog has one Close (L-12).
6. Profile › Your access shows plain words for every permission (L-17).
7. The rounding note appears without moving the fields below (L-19); Enter saves Display preferences (L-20);
   the rate import's collapsed groups show a marker (L-22).
8. Restore texts say a rename once; the toast reads "<Level> '<name>' restored — View"; level tags use one style
   (L-23).
9. A missing purchase request's breadcrumb says Purchase requests (L-24); Home's getting-started steps suit the
   role (L-25); the development sample page offers Restore only to those who may (L-27); an empty dashboard
   purchase-request panel says so (L-29).

**Why this priority**: cosmetic or wording; each has a workaround.

**Independent Test**: Each scenario on the demo data.

---

### User Story 15 - The server answers precisely (Priority: P3)

**Acceptance Scenarios**:

1. A campaign date of 2026-02-30 gets "Enter a valid date." on that field; a body over the size limit is refused
   as "This request is larger than 1 MB." (L-5).
2. A name with a tab, a line break or a direction-override character is refused on that field (L-11).
3. A Viewer approving a round someone already decided is told their role does not include approving (L-15).
4. A setting never changed shows "Not changed since SP&A One was set up" (L-16).

**Why this priority**: low impact; consistency of 022's error rules.

**Independent Test**: QA's request probes.

---

### User Story 16 - The person's date format is used everywhere (Priority: P3)

**Acceptance Scenarios**:

1. **Given** Priya's date format YYYY-MM-DD, **Then** the Activity log heading reads "Tuesday 2026-10-27",
   History likewise, and "Accrued (2026-09)" (L-18).
2. **Given** each of the four formats, **Then** day headings and month labels follow FR-058's table; "Today" and
   "Yesterday" stay words.

**Why this priority**: preference ignored on a few screens; cosmetic.

**Independent Test**: switch the format and visit the Activity log, History, Financials, line item, budget by
month and dashboards.

---

### User Story 17 - Exports and demo data are coherent (Priority: P3)

**Acceptance Scenarios**:

1. The plan workbook's money columns are numbers (Excel SUM works); re-importing an unchanged export still
   reports nothing to change (L-10).
2. Every seeded record's "Created by … on …" matches its first History entry (L-8).
3. PR-1001 can be submitted after its activity is approved: FY2027 finance budget items exist (L-30).

**Why this priority**: demo and spreadsheet convenience.

**Independent Test**: Export, sum and re-import; reset the demo data and walk PR-1001.

---

### User Story 18 - Mandatory fields carry a red asterisk (Priority: P2)

A person filling in any form sees at a glance which fields must be filled to save: a red asterisk after the
label, the convention people know from other business applications. Product owner request of 2026-10-29
(D-364, constitution 4.7.0).

**Acceptance Scenarios**:

1. **Given** any form with a field required to save (New plan's Name, New team's Name, a PR line's amount),
   **When** it opens, **Then** the label is followed by a red `*` in both themes, and the word tag "Required"
   is no longer shown.
2. **Given** a field "Needed to publish" or "Needed to submit", **When** the form opens, **Then** it keeps its
   word tag and has no asterisk (it is not needed to save).
3. **Given** a form with at least one asterisk, **When** it opens, **Then** a legend "* Required" appears once,
   above the first section or field.
4. **Given** a screen reader, **When** focus reaches a required field, **Then** it announces the label and
   "required" once (not "star"), and the control has `aria-required="true"`.
5. **Given** the dark theme, **When** axe runs on a form, **Then** no contrast violation is reported for the
   asterisk.

**Why this priority**: a visible, familiar marking for every form; no business rule changes.

**Independent Test**: Open New plan, New team and a purchase request's form in light and dark mode; run axe.

---

### Edge Cases

- **Change raises one line and lowers another** (C-1): each line reserves its own larger amount; the lowered
  line's difference is not available to the raised one until approval.
- **Approved change after a top-up** (033): the reservation becomes the approved values exactly as today.
- **Restore from the recycle bin of a purchase request with a pending change**: its room check uses the same
  larger amounts.
- **Rate changes during review**: the larger amount is compared after converting both with the current rate, as
  every reservation (D-303).
- **A transaction linked to a line item later moved to another plan**: unlink is checked against the plan of the
  line item it is linked to now.
- **An administrator with import but a scope smaller than all markets**: sees every not-linked transaction (they
  belong to no plan) and may link only to line items in scope.
- **Sign-in lock and server restart**: counts are kept in memory; a restart clears them (local phase, one
  process) — accepted and documented.
- **Behind the web server or the development proxy** the client is the address the proxy reports; a request that
  does not come through a trusted proxy is counted by its own address.
- **E-mail written differently** ("Priya.Raman@shell.com ") counts as the same e-mail; unknown e-mails count and
  lock exactly like known ones, so the answer does not reveal whether a person exists.
- **Approve with Warn when two activities exceed**: the warning lists both, each once.
- **Overspend approval setting changed while a round is open**: the setting at the moment of approving decides.
- **Dashboard address with a malformed value** (lower case, wrong pattern): falls back as before (036).
- **A workbook row that fails because an earlier row in the same file created a duplicate name**: still reported
  (the in-file duplicate rule is unchanged).
- **Home owned activities spanning more than 5 years** (plans run to current + 10): the link and the figure cover
  the 5 years from the earliest, and the card says "in <first>–<last>".
- **Month label in charts' axis ticks**: ticks keep month names (Jan … Dec) without a year, like the Calendar's
  column headings; full labels follow the format.
- **Paste with more values than months left**: the extra values are ignored and the summary says how many were
  left out.

## Requirements *(mandatory)*

### Functional Requirements

Each requirement names the QA finding(s) it closes and the earlier requirement it restores (R) or amends (A).

**Purchase-request reservation (C-1)**

- **FR-001**: While a change of an approved purchase request is in review, each line MUST reserve on its line item
  the larger of its last approved amount and its changed amount; a line the change adds reserves its changed
  amount; a line the change removes keeps its approved amount reserved until the change is approved. [C-1; A 032
  FR-037, R 032 FR-017]
- **FR-002**: Total requested and Available to request on line items and activities, the line-item choices when
  adding purchase-request lines, the room check on save, and the room check of a restore MUST all use FR-001's
  reservation. [C-1; R 032 FR-017, FR-018]
- **FR-003**: Approving a change MUST release what it lowers; declining or recalling it MUST restore the approved
  values without any line item's Total requested exceeding what was reserved during the review. [C-1; R 032
  FR-017, FR-037]

**Financial transactions (H-4, H-5)**

- **FR-004**: Transactions not linked to any line item — in lists, the Not linked tab and its count, Home's "n
  transactions not linked" and opening one — MUST be visible only to people who hold "Import GSAP extracts";
  everyone else MUST see no Not linked tab, count or link, and opening one MUST answer as a missing transaction.
  [H-4; A 015 FR-026, FR-028, Q-015c]
- **FR-005**: Linking a not-linked transaction MUST need "Import GSAP extracts" and "Link financial transactions"
  in the target line item's plan; unlinking a manual link MUST need "Link financial transactions" in the plan of
  the line item it is linked to and a transaction the person can see. Link and Unlink MUST be shown only when the
  server says the person may use them. [H-4, H-5; A 015 FR-026, FR-028]
- **FR-006**: "Link financial transactions" MUST be granted only together with "Import GSAP extracts": the
  built-in Global planner and Planner roles no longer hold it (014 Table B), saving a role that holds it alone
  MUST be refused with the message of US2 scenario 6, and a role that still holds it alone grants nothing.
  [H-4; A 014 Table B, 015 assumptions]
- **FR-007**: A transaction, or a target line item, outside the person's scope MUST behave exactly like a missing
  one — same answer, same message — checked before permission, version and state; a conflict or "linked by"
  message MUST NOT name a person for a record the person cannot see. [H-5; R 014 FR-008, 022]

**Plan workbook (H-2)**

- **FR-008**: Checking and importing a workbook within the documented limits (5 MB, 5,000 data rows) MUST finish
  within SC-002 and never fail because of its size; the work per row MUST NOT include reading reference data
  (plan context, types, people, workflows, attributes, siblings, budgets), which is read once per check. The
  problems found MUST be exactly those found before. [H-2; R 012 FR-028, 028 FR-004]
- **FR-009**: Reading and parsing an uploaded file MUST happen before other people's saves are held back.
  [H-2]
- **FR-010**: A problem's "value found" MUST be shortened to at most 200 characters followed by "…". [H-2]
- **FR-011**: The packaged web server MUST accept an upload of the documented maximum size. [H-2, found while
  tracing it]

**Dashboards (H-1)**

- **FR-012**: A dashboard address whose currency or market is unknown, inactive or outside the person's scope
  MUST show the dashboard with the default (display currency, else US dollars; the person's default market), the
  filter bar MUST show the value used and the address MUST be corrected to it; exports MUST fall back alike.
  [H-1; R 036 FR-007]

**Delete (H-3, L-3)**

- **FR-013**: Deleting a purchase request from its page MUST land on the Purchase requests page (or the line
  item it was raised from when the person cannot see that page) with "Purchase request '<name>' deleted" and Undo;
  deleting a purchase request, plan or line item from its page MUST NOT ask for the deleted record again.
  [H-3, L-3; R 032 FR-042, design system §4 after delete]

**Security hardening (security checks, L-14, M-3)**

- **FR-014**: Sign-in MUST refuse further attempts for 15 minutes when one e-mail (compared ignoring case and
  surrounding spaces) has 5 failed attempts within 15 minutes, or one client has 20; the refusal reads "Too many
  sign-in attempts. Try again in 15 minutes." with the remaining wait, also for the right password; attempts
  refused during the lock do not extend it; a successful sign-in clears the failures of that e-mail and that
  client; choosing a demo person counts as signing in with its e-mail; unknown e-mails count like known ones.
  [Security check "Sign-in brute force"; D-361]
- **FR-015**: Every web page and every server answer MUST forbid being framed by any site, send no referring
  address beyond the own origin, and forbid content sniffing; every web page MUST run under a content security
  policy that admits only the application's own scripts, styles, fonts, images and connections, admits the
  inline theme script only by its fingerprint, and forbids other inline scripts and evaluated code — enforced by
  the production preview and the packaged web server, reported only by the development server. [Security check
  "Security headers"; R constitution F7; D-361]
- **FR-016**: A request that changes something and carries no origin information MUST be refused like a request
  from another site when its referring page belongs to another site; one with neither is handled as today.
  [L-14; D-361]
- **FR-017**: A text value containing NUL, another control character (tab, line feed and carriage return
  allowed in multi-line texts only) or an unpaired surrogate MUST be refused with "Remove the control characters
  from this text." on that field before anything is saved or numbered; a record ID containing one MUST answer
  "not found"; neither may answer with a server error. [M-3, L-11 (surrogates); R 001 FR-020, 022]

**Approval over planned cost (M-18)**

- **FR-018**: Settings › Overspend control MUST offer "When an activity's line items exceed its planned cost,
  approving it or its purchase requests": **Warn the approver** (default) or **Refuse approval**, changed only by
  people who may change Settings, versioned, and logged with the old and new value. [M-18; A 037 FR-006; D-360]
- **FR-019**: With Warn, the approval section of an activity and of a purchase request, and the Approve dialog,
  MUST show for each activity of the round whose line items exceed its planned cost "Line items of activity
  '<name>' are <overage> above its planned cost" before approving, and the approval's activity-log entry MUST
  record the overage. [M-18; D-360]
- **FR-020**: With Refuse, approving any level of an activity round, a purchase-request round or a change round
  MUST be refused while one of its activities' line items exceed its planned cost, naming each activity and its
  overage; nothing changes; Approve is shown disabled with that reason; Decline and Recall stay allowed.
  [M-18; A 037 FR-019; D-360]
- **FR-021**: The overage MUST be the activity's negative unassigned planned cost (domain model §4.3) in the plan
  currency, worked out when approving. [M-18; R 037 FR-017]

**Money figures (M-1, M-2, L-2)**

- **FR-022**: In the spread grid, in any display currency, each row's months MUST add up to its total, each
  column's cells to its column total and the column totals to the grand total; each row total MUST equal the line
  item's planned cost converted once. [M-1; R domain model §4.4]
- **FR-023**: In the SP&A view, "Not yet scheduled" MUST be the sum, per activity, of planned cost not yet placed
  in a month, never below zero for any activity; activities whose line items exceed their planned cost MUST be
  shown as a row "Line items above planned cost" with the total excess, so that periods + Not yet scheduled −
  excess = planned cost. [M-2; A 013 FR-017; D-363]
- **FR-024**: All work Budget & cost, the dashboards' monthly chart and Budget by month of a plan, campaign or
  initiative MUST say "<n> activities have line items above their planned cost, so their months add up to more
  than their planned cost." whenever that is why the monthly total exceeds planned cost. [L-2; R 010 FR-023, 036
  FR-014]

**Purchase-request screens (M-7 – M-10)**

- **FR-025**: A purchase-request page for a person who may change nothing on it MUST show one View only note and
  no disabled change actions (D-352); a refusal for a person lacking the permission MUST give the permission
  reason before any status guidance. [M-7; R 014 FR-011]
- **FR-026**: The purchase-request form's needed-fields counter MUST count the same needed fields as the page's
  "fields needed", including attributes of the header and lines needed to submit. [M-8; R 032 FR-021]
- **FR-027**: The purchase-request submission check MUST name the approvers of each level, including levels whose
  approver comes from a field of the record (MoA approver). [M-9; R 031 record-field approvers]
- **FR-028**: Submit changes on an approved purchase request MUST first show a confirmation stating that a change
  approval round starts, with its workflow and first approvers, and that amounts stay reserved as FR-001 says
  until it is approved. [M-10; R design system "consequences first"]

**Access-consistent screens (M-6, M-12, M-4, L-28)**

- **FR-029**: "Edit budget by month" MUST be offered only to people who may edit the plan, campaign or
  initiative. [M-6; R 014 FR-011]
- **FR-030**: The New and Edit global budget and Shell attribute forms MUST report every problem at the first
  Save. [M-12; R design system "all problems at once"]
- **FR-031**: When the session ends while a form has unsaved changes, the sign-in page MUST say "Your session
  ended, so your changes in '<form title>' weren't saved."; after signing in the person MUST return to the same
  address including its fragment. [M-4, L-28; R 001 FR-006; D-363]

**Home, All work (M-5, M-11)**

- **FR-032**: Home's "Activities I own" MUST open All work on the plan years of those activities (at most 5,
  from the earliest), name those years on the card ("in 2026–2027"), and its figure MUST equal what that list shows. [M-5; R 009 FR-007, FR-017; D-363]
- **FR-033**: All work exports MUST use the status words of the screen ("Partly approved"). [M-11; R 010 FR-032,
  FR-033]

**Imports (M-13, L-21, L-13)**

- **FR-034**: The rate import MUST check a row containing stray quotes as a row (a problem if invalid) instead of
  refusing the file; MUST say "This file is empty." for an empty file; MUST name every missing column at once;
  MUST give the "too large" message for a rate above the maximum; and MUST refuse rate dates before 1 Jan 2000.
  [M-13, L-21; R 016 FR-006]
- **FR-035**: The GSAP extract import MUST read files separated by semicolons as it reads comma-separated ones.
  [L-13; A 015 import layout]

**Layout and accessibility (M-14 – M-17)**

- **FR-036**: Names without spaces MUST wrap inside their card or field, and the Plans list and the SP&A view MUST
  fit 390, 768 and 1440 px without sideways page scroll. [M-14, M-15; R design system layout, WCAG 1.4.10]
- **FR-037**: Every region that scrolls sideways MUST be reachable and scrollable by keyboard. [M-16; R WCAG
  2.1.1]
- **FR-038**: Text on grey surfaces MUST reach 4.5:1 (split links, Regional dashboard empty-state hint,
  purchase-request form counter). [M-17; R design system §3.1, WCAG 1.4.3]

**Wording and small interactions (P3)**

- **FR-039**: The activity form's conflict message MUST add the next step the other forms give. [L-1]
- **FR-040**: Error-summary entries for month amounts MUST name the month. [L-4]
- **FR-041**: Every Move menu item MUST read "Move to…". [L-6]
- **FR-042**: Pasting tab-separated values into a month MUST fill that and the following months; money inputs
  MUST accept a leading currency symbol or code and thousands separators. [L-7]
- **FR-043**: Counts in bulk dialogs MUST agree in number ("Submit 1 activity"). [L-9; R 012 FR-013]
- **FR-044**: Compact amounts MUST never round up to "1000T" (the full amount shows instead); Budget used in the
  allocations table MUST be a whole percent ("26%"); a dialog that only informs MUST have one Close. [L-12; R 014
  FR-032]
- **FR-045**: My profile › Your access MUST describe every permission in words of 014 Table A. [L-17]
- **FR-046**: A note that appears under a field (rounding) MUST NOT move the fields below it. [L-19]
- **FR-047**: Enter MUST save a dialog whose fields are all choice lists (Display preferences). [L-20; R design
  system "Enter submits"]
- **FR-048**: Collapsed groups in the rate import review MUST show an open/closed marker. [L-22]
- **FR-049**: Restore texts MUST state a rename or status reset once; the restore toast MUST read "<Level>
  '<name>' restored — View" (View when the record has a page); level tags in the bin MUST use one style.
  [L-23; R 011 FR-026]
- **FR-050**: A purchase-request page, also when the request is missing, MUST start its breadcrumb with Purchase
  requests when the person may see that page. [L-24]
- **FR-051**: Home's getting-started steps MUST offer only what the person's role can do, and a person without a
  team who may not change teams MUST be told to ask an administrator. [L-25; A 001 FR-018 wording]
- **FR-052**: The development sample-records page MUST offer Restore only to people who may restore. [L-27]
- **FR-053**: The dashboards' Purchase requests panel with none MUST say "No purchase requests on these plans
  yet." instead of a table of zeros. [L-29; R 036 FR-028]

**Precise answers (P3)**

- **FR-054**: An impossible date on campaigns and initiatives MUST be refused on its field with "Enter a valid
  date."; a request larger than its limit MUST be refused as too large, saying the limit in words. [L-5 (a),
  (c); R 022]
- **FR-055**: Single-line names (plans, campaigns, initiatives, activities, line items, global budgets,
  purchase requests, Settings lists, teams) MUST refuse tabs, line breaks and direction-control characters with
  "Names can't contain tabs, line breaks or text-direction characters.". [L-11]
- **FR-056**: A person who holds neither "Approve or decline" permission MUST be refused for permission before
  being told that the round has moved on. [L-15; A 014 approve check order; D-363]
- **FR-057**: A Settings page whose setting was never changed MUST say "Not changed since SP&A One was set up"
  instead of a last-changed line (Overspend control, Purchase requests). [L-16]

**Dates (L-18)**

- **FR-058**: The person's date format (003 FR-024) MUST be used for day headings of the Activity log and History
  and for every month label with a year, as follows; "Today" and "Yesterday" stay words. [L-18; A 011 FR-004;
  D-362]

  | Format | Day heading | Month label |
  |---|---|---|
  | DD MMM YYYY (default) | Tuesday 27 Oct 2026 | Sep 2026 |
  | DD/MM/YYYY | Tuesday 27/10/2026 | 09/2026 |
  | MM/DD/YYYY | Tuesday 10/27/2026 | 09/2026 |
  | YYYY-MM-DD | Tuesday 2026-10-27 | 2026-09 |

**Exports and demo data (P3)**

- **FR-059**: The plan workbook MUST write money as numbers; an unchanged export re-imported MUST still report
  nothing to change. [L-10; R 012 FR-027]
- **FR-060**: Every seeded record's created line MUST match its first History entry. [L-8]
- **FR-061**: The demo data MUST include FY2027 finance budget items for the FY2027 line items, so PR-1001's
  activity can be approved and PR-1001 submitted. [L-30]

**Records**

- **FR-062**: The decision log (D-359 – D-363), the feature map and the amended specs (index below) MUST state
  these rules. [all]

**Mandatory-field marking** (product owner request, D-364)

- **FR-063**: Every field required to save (`FieldShell` marking "Required") MUST show a `*` after its label in
  Secondary Red, bold and at least 19 px (WCAG large text, so the 3:1 bar holds in the dark theme with Shell's
  exact red), with no word tag. The asterisk MUST be hidden from screen readers; the control MUST carry
  `aria-required="true"` and its accessible description "required" once. Fields "Needed to publish" and "Needed
  to submit" MUST keep their word tags and show no asterisk. [PO request]
- **FR-064**: A form, dialog or side panel with at least one asterisk MUST show the legend "* Required" once,
  before its first field, with the asterisk styled as in FR-063. [PO request]

### Findings that need no change of behaviour

- **L-26** (Home "—" and missing "HOME" eyebrow): intended. The design system (§2 page header: "No eyebrow… Home is
  the only exception: its title greets"; §4.4, X-08: "—" when nothing is planned) replaced 009 FR-001's eyebrow,
  and a test pins the "—". Only 009 FR-001's text is corrected (FR-062).
- **L-5 (b)** (bulk "change field" refused): not a bug — QA sent `field`/`value` text; the documented shape is a
  key and a typed value, which the screen sends and which works. L-5 (a) and (c) are fixed by FR-054.
- **L-2** is the intended rule of 010 FR-023; only its explanation is missing (FR-024).

### Key Entities

- **Overspend control (Settings)**: gains **Approval over planned cost**: Warn the approver (default) or Refuse
  approval; versioned and logged with the existing setting.
- **Reservation of a purchase-request line**: during a change in review, the larger of the approved and the
  changed amount; never stored separately (worked out from the line and the approved version).
- **Sign-in attempts**: per e-mail and per client, the failures of the last 15 minutes and a lock end time; held
  by the running server only.
- **Activity overage**: an activity's line items above its planned cost; shown in approvals, never stored.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Through any sequence of purchase-request changes, declines, recalls, approvals and new requests, no
  line item's Total requested exceeds what was approved or reserved: QA's C-1 sequence ends with Available to
  request ≥ 0.00 and the second request refused.
- **SC-002**: On the developers' reference machine, checking a 1,000-row workbook takes at most 5 seconds and a
  5,000-row one at most 20 seconds, without error; the server work of a check grows by at most 4 storage
  statements per row on top of at most 150 per check (today about 18 per row).
- **SC-003**: 100 % of dashboard addresses with unknown, inactive, out-of-scope or malformed filter values show a
  dashboard with its filter bar; 0 show an error page.
- **SC-004**: After 5 failed sign-ins for one e-mail within 15 minutes, 100 % of further attempts in the next 15
  minutes are refused (also the right password), and the right password succeeds at the first attempt afterwards.
- **SC-005**: Every web page and server answer carries the four protections of FR-015; the full end-to-end suite
  shows 0 content-security-policy violations.
- **SC-006**: 0 server errors across QA's malformed-input probes (NUL, control characters and unpaired surrogates
  in every text field and record ID, oversized bodies).
- **SC-007**: For every link and unlink probe of H-5, the answer for an out-of-scope record is byte-for-byte the
  answer for a missing one.
- **SC-008**: The spread grid of every demo activity adds up exactly by row and column in USD, GBP and EUR display.
- **SC-009**: axe reports 0 violations on the screens of M-16 and M-17 in light and dark mode; Plans list and SP&A
  view have no sideways page scroll at 390 px with 120-character names.
- **SC-010**: With Refuse, 0 approvals succeed while line items exceed planned cost; an administrator changes the
  setting in at most 3 steps (open Settings › Overspend control, choose, save) — design system §10.
- **SC-011**: After deleting a purchase request, the person reaches Undo in 1 step from where they land — design
  system §10.
- **SC-012**: Re-running QA's reproduction scripts, none of the 53 findings fixed here reproduces, and L-26 and
  L-5 (b) behave as described under "Findings that need no change of behaviour".
- **SC-013**: 100 % of fields required to save show the red asterisk and 0 show the "Required" word tag; axe
  reports 0 violations on New plan, New team and a purchase request's form in light and dark mode.

## Assumptions

- The sign-in limit is kept in the running server's memory: SP&A One runs as one process in this local phase; a
  shared store comes with hosting (later).
- The client of a sign-in is the address reported by the trusted local proxy (development server or packaged web
  server); no other proxy is trusted.
- The development server's policy is report-only because its tooling injects scripts during development; the
  constitution (F7) requires enforcement of the production build, which the preview and the packaged web server
  do.
- QA's measurements were taken on a developer laptop; SC-002 uses the same class of machine, measured with the
  repository's benchmark tool.
- Not tested by QA and out of scope here: the 12-hour session expiry, a screen-reader pass, PR top-up after an
  increased change, overspend Refuse through the allocation dialogs.
- `/auth/demo-people` exposing the demo password is by design (demo sign-in, D-004) and leaves with single sign-on.

## Traceability — every QA finding

| Finding | Severity | FR(s) | Note |
|---|---|---|---|
| C-1 | Critical | FR-001 – FR-003 | |
| H-1 | High | FR-012 | 036's data model and contract said such a link is refused; corrected |
| H-2 | High | FR-008 – FR-011 | FR-011: packaged web server limit found while tracing |
| H-3 | High | FR-013 | |
| H-4 | High | FR-004 – FR-006 | Product decision D-359 |
| H-5 | High | FR-005, FR-007 | |
| M-1 | Medium | FR-022 | |
| M-2 | Medium | FR-023 | Was pinned by a test; rule amended (D-363) |
| M-3 | Medium | FR-017 | Story 6 (P1: unauthenticated sign-in) |
| M-4 | Medium | FR-031 | Drafts not stored (F7); notice instead (D-363) |
| M-5 | Medium | FR-032 | |
| M-6 | Medium | FR-029 | |
| M-7 | Medium | FR-025 | |
| M-8 | Medium | FR-026 | |
| M-9 | Medium | FR-027 | |
| M-10 | Medium | FR-028 | |
| M-11 | Medium | FR-033 | |
| M-12 | Medium | FR-030 | |
| M-13 | Medium | FR-034 | |
| M-14 | Medium | FR-036 | |
| M-15 | Medium | FR-036 | |
| M-16 | Medium | FR-037 | |
| M-17 | Medium | FR-038 | |
| M-18 | Medium | FR-018 – FR-021 | Product decision D-360 |
| L-1 | Low | FR-039 | |
| L-2 | Low | FR-024 | Rule intended (010 FR-023); explanation added |
| L-3 | Low | FR-013 | Same family as H-3 |
| L-4 | Low | FR-040 | |
| L-5 | Low | FR-054 | (b) is not a bug — see above |
| L-6 | Low | FR-041 | |
| L-7 | Low | FR-042 | |
| L-8 | Low | FR-060 | |
| L-9 | Low | FR-043 | |
| L-10 | Low | FR-059 | |
| L-11 | Low | FR-055, FR-017 | Surrogates through FR-017 |
| L-12 | Low | FR-044 | |
| L-13 | Low | FR-035 | 015's plan said comma only; amended |
| L-14 | Low | FR-016 | Security decision D-361 |
| L-15 | Low | FR-056 | Documented check order changed (D-363) |
| L-16 | Low | FR-057 | |
| L-17 | Low | FR-045 | |
| L-18 | Low | FR-058 | Product decision D-362 |
| L-19 | Low | FR-046 | |
| L-20 | Low | FR-047 | |
| L-21 | Low | FR-034 | |
| L-22 | Low | FR-048 | |
| L-23 | Low | FR-049 | |
| L-24 | Low | FR-050 | |
| L-25 | Low | FR-051 | Wording from 001 FR-018; gated by role |
| L-26 | Low | FR-062 (doc only) | Intended — no behaviour change |
| L-27 | Low | FR-052 | |
| L-28 | Low | FR-031 | |
| L-29 | Low | FR-053 | |
| L-30 | Low | FR-061 | |
| Security: sign-in brute force | Gap | FR-014 | D-361 |
| Security: headers | Gap | FR-015 | D-361 |
| Product owner request: red asterisk on mandatory fields | Request | FR-063, FR-064 | D-364, constitution 4.7.0 |

54 findings: 53 change behaviour (L-2 only gains an explanation; part (b) of L-5 is not a bug); L-26 needs only a
correction of 009's text.

## Amendment index

| Feature | Requirement | Change |
|---|---|---|
| 003 | FR-024 | "Every date shown" includes day headings and month labels (038 FR-058) |
| 009 | FR-001 | No eyebrow on Home (design system); text corrected (L-26) |
| 011 | FR-004 | Day headings follow the date format: "Monday 12 Oct 2026" is the default format's example (038 FR-058) |
| 013 | FR-017 | "Not yet scheduled" never negative; "Line items above planned cost" row (038 FR-023) |
| 014 | Table B; approve check order | Global planner and Planner lose "Link financial transactions"; permission before "round moved on" (038 FR-006, FR-056) |
| 015 | FR-026, FR-028, Q-015c, assumptions, plan | Not linked visible to importers only; link and unlink rules; semicolon files (038 FR-004 – FR-007, FR-035) |
| 032 | FR-037 | Reserve the larger of approved and changed amounts during review (038 FR-001 – FR-003) |
| 036 | FR-007 (data model, contract) | Unknown currency or market falls back instead of being refused (038 FR-012) |
| 037 | FR-006, FR-019 | Second Overspend control setting; approval warned or refused (038 FR-018 – FR-021) |
