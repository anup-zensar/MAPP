# Feature Specification: GSAP PR Number and Purchase Order from a Simulated GSAP

**Feature Branch**: `033-simulated-gsap-pr-po` (no branch created: the product owner asked for no Git
branches or commits on 2026-10-27)

**Created**: 2026-10-27

**Status**: Draft (clarified with documented assumptions, see Clarifications)

**Input**: Product owner instruction of 2026-10-27: "When the flow goes back to GSAP (GSAP PR → PO), mimic
GSAP too: a simulated GSAP creates the GSAP PR number and the purchase order for a PR routed to the
requisitioner. The flow ends at the PO." Decision D-321 and ADR-0016: for a purchase request routed to the
requisitioner, a simulated GSAP creates the GSAP PR number and the purchase order (PO number, one PO line per
PR line, keyed by the HX key); the PO lines are recorded as GSAP transactions (feature 015) so Committed
figures follow; everything it produces is labelled "Simulated GSAP"; running it needs "Import GSAP extracts"
(`finance.import`).

**References**: client 013 FR-013 (GSAP fields read-only, "Filled by GSAP"), FR-055 (GSAP PR number and date
on the header; PO number, PO line, PO line status, PO date and value on each line); client 015 FR-017–FR-019
(inbound content: GSAP PR number and date, PO number, line, status, date, amount and currency, keyed by the
HX key; where GSAP data lands); Hive9 functional analysis §D3 stages 2–3 (the requisitioner builds the GSAP
PR with the HX key; GSAP converts it into a PO; the PO returns matched on the HX key), §D5 (keys); features
015 (financial transactions, Committed), 032 (purchase requests; the routing; the GSAP port and fields of
brief §8.3); ADR-0009, ADR-0016.

**Words used here**: the *simulated GSAP* is MAPP's stand-in for GSAP until live integration (constitution
II). A *run* is one request to it for one purchase request. *Top-up* is a run after an approved change of an
already ordered PR. Screens label everything it produces "Simulated GSAP".

## Clarifications

### Session 2026-10-27

No product owner was available in the loop (brief §5); each point below is a documented assumption to
confirm.

- Q: What do the simulated numbers look like? → A: GSAP PR numbers "SIM-" + 10 digits from SIM-0010001001,
  PO numbers "SIM-" + 10 digits from SIM-4500001001, each in its own sequence; PO lines numbered 10, 20, 30…
  as GSAP does. The "SIM-" marker means nobody takes them for real GSAP numbers (ADR-0016); Shell's real
  formats replace them with live GSAP.
- Q: One PO per PR, or several? → A: One PO per PR, one PO line per PR line. Real GSAP may split by vendor
  or outline agreement; that stays with live integration.
- Q: What status and date does a PO line get? → A: "Released", dated the day of the run; MAPP has no release
  threshold (GSAP's, FA D7) and no "waiting for release" stage.
- Q: Who runs it, and when? → A: A person allowed to import GSAP extracts (`finance.import`) — the seeded
  Requisitioner role holds it — for one PR at a time, from the PR's page, while the PR is in the Waiting for
  GSAP view (032 FR-032). There is no scheduled or "all waiting" run.
- Q: What happens on a second run? → A: It is refused unless the PR changed since the last run (an approved
  change, or a line added without approval, made a line's amount or currency differ from its PO value or
  added a line). Then the same GSAP PR and PO numbers are kept, changed PO lines are updated and new PR lines
  get new PO lines (top-up).
- Q: Does closing a PR change its PO? → A: No. Closing reduces the PR's lines to what GSAP invoiced (032
  FR-041); the PO stays as GSAP made it.
- Q: Is there a PO copy link? → A: No; no document is created (supporting documents and PO PDFs are out of
  scope).
- Q: Which date is the posting date of the PO transaction? → A: The day of the run.

## User Scenarios & Testing *(mandatory)*

Examples use the demo data and a purchase request that the 032 journey approved: **PR-1002** "PR for Gift
boxes", one line PRL-1002 for 10,000.00 USD on line item "Gift boxes" (written L-1011 below, HX key
HX1011, its SWBS code from the simulated GSAP of 030), vendor "Northlight Events GmbH" (vendor code
V-100101), short text "Christmas gift boxes", routed Original, listed under Waiting for GSAP. Lea Santos is the
seeded requisitioner (`finance.import`); Priya Raman owns the PR (no `finance.import`).

### User Story 1 - The simulated GSAP creates the GSAP PR and the purchase order (Priority: P1)

The requisitioner opens a PR from Waiting for GSAP and chooses **Create GSAP PR and PO (simulated)**. MAPP
asks the simulated GSAP, which answers at once with a GSAP PR number and a purchase order with one line per
PR line. The PR shows them read-only, labelled "Simulated GSAP"; the PO lines become GSAP purchase-order
transactions, so the line item's Committed figure and the Financials page show them; the PR leaves the
Waiting for GSAP view.

**Why this priority**: it closes the product owner's PR→PO journey ("the flow ends at the PO") without live
GSAP, and proves that GSAP's answer lands where marketers look.

**Independent Test**: as Lea, open PR-1002 and run it; check the GSAP PR number, the PO line, L-1011's
Committed figure and the Financials page; run it again and read why it is refused.

**Acceptance Scenarios**:

1. **Given** PR-1002 Approved, routed and waiting for GSAP, **When** Lea chooses Create GSAP PR and PO
   (simulated), **Then** a confirmation says "The simulated GSAP will create a GSAP PR and a purchase order
   with 1 line (10,000.00 USD) for PR-1002." and offers Create and Cancel.
2. **Given** she confirms, **Then** the PR shows GSAP PR number SIM-0010001001 and today's date with the
   source "Simulated GSAP", and line PRL-1002 shows PO number SIM-4500001001, PO line 10, PO line status
   Released, PO date today and PO value 10,000.00 USD, each read-only with "Filled by GSAP".
3. **Given** that run, **Then** L-1011's Financials section shows Committed 10,000.00 USD (Not yet committed
   0.00), the activity and the plan include it, and the Financials page lists "PO SIM-4500001001 / 10" linked
   to L-1011 by its HX key, from the import "Simulated GSAP purchase order".
4. **Given** that run, **When** Lea opens Purchase requests › Waiting for GSAP, **Then** PR-1002 is no longer
   listed; searching "SIM-4500001001" or "SIM-0010001001" finds it.
5. **Given** PR-1002 already ordered and unchanged, **When** Lea looks for the action, **Then** it is
   unavailable with "The simulated GSAP already created the PO for this purchase request. Run it again after
   an approved change of its amounts or lines."
6. **Given** Priya (owner, no `finance.import`), **When** she opens PR-1002, **Then** the action is not shown,
   and a request she makes another way is refused with "Importing GSAP extracts isn't part of your role …".
7. **Given** a Created, Submitted, In review, Declined or Closed PR, or an Approved PR not routed yet, **When**
   Lea opens it, **Then** the action is unavailable with the reason ("The simulated GSAP processes approved
   purchase requests that were routed to the requisitioner.").
8. **Given** the run, **Then** the PR's History shows one entry "Simulated GSAP created GSAP PR SIM-0010001001
   and PO SIM-4500001001 (1 line)" by Lea at the time of the run, and the Financials import history shows the
   import of 1 purchase-order line.

---

### User Story 2 - Top-up after an approved change (Priority: P2)

After an ordered PR's change is approved — an amount raised, or a line added — the PR is waiting for GSAP
again. A new run keeps the GSAP PR and PO numbers, updates the changed PO lines and adds PO lines for new PR
lines; Committed follows.

**Why this priority**: Shell's process tops up a PO after a modification (FA D3, slide 11); the first order
already delivers the P1 journey.

**Independent Test**: raise PR-1002's line from 10,000.00 to 12,000.00 USD after giving L-1011 room, approve
the change, run the simulated GSAP again and check the PO line and Committed.

**Acceptance Scenarios**:

1. **Given** PR-1002 ordered at 10,000.00 USD and a change to 12,000.00 USD approved (routed Modification),
   **Then** PR-1002 is listed under Waiting for GSAP again.
2. **When** Lea runs the simulated GSAP, **Then** the confirmation says "The simulated GSAP will update PO
   SIM-4500001001: 1 line changed, 0 lines added (top-up)."; after confirming, GSAP PR number and PO number are
   unchanged, PO line 10 shows PO value 12,000.00 USD and today's date, and L-1011's Committed is 12,000.00 USD
   — the earlier PO transaction is updated, not repeated.
3. **Given** a change that added a line on another GB01 line item was approved, **When** Lea runs it, **Then**
   the new line gets PO line 20 under the same PO number, and its line item's Committed includes it.
4. **Given** a change In review, **When** Lea looks for the action, **Then** it is unavailable with "Wait until
   the change of this purchase request is approved or declined."

---

### Edge Cases

- **Purchase Support switched off**: the action is unavailable and refused with "Purchase requests are
  switched off for your organisation."
- **PR outside the person's access**: not found, like a missing PR.
- **A line item's SWBS code changed after routing**: the PO line carries the line item's current code.
- **A PR line in another currency than its line item**: its PO line and its transaction are in the PR line's
  currency; Committed converts at today's rate (015).
- **Invoices later arrive** (simulated GSAP extract, 015) referring to PO SIM-4500001001 line 10: they count in
  the PR line's Total invoiced (032 FR-043) and so in close-out (032 FR-041).
- **The PR is closed after its PO**: the PO and its transactions stay; the PR leaves Waiting for GSAP for good.
- **Two requisitioners run the same PR at once**: the second sees it already ordered (refused as unchanged).
- **The run fails part-way** (for example a currency removed from Settings): nothing of it is kept; the
  message names the problem.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Users holding "Import GSAP extracts" who can see a purchase request MUST be offered **Create GSAP
  PR and PO (simulated)** on its page when Purchase Support is on, the PR is Approved with no change In review,
  it was routed to the requisitioner, and it is waiting for GSAP (032 FR-032); otherwise the action MUST be
  unavailable with the reason, or hidden for people without the permission.
- **FR-002**: The simulated GSAP MUST give a PR, at its first run, a GSAP PR number "SIM-" followed by ten
  digits, in one sequence starting at SIM-0010001001, and the run's date as GSAP PR date; later runs keep
  both.
- **FR-003**: The simulated GSAP MUST create one purchase order per PR, numbered "SIM-" followed by ten digits
  in its own sequence starting at SIM-4500001001, with one PO line per PR line, numbered 10, 20, 30… in line
  order; each PO line has status "Released", the run's date and a value equal to its PR line's amount in the
  PR line's currency. A PR line keeps its PO line on later runs; new PR lines continue after the highest PO
  line.
- **FR-004**: MAPP MUST store GSAP's answer on the PR (GSAP PR number, date, source) and on each line (PO
  number, PO line, PO line status, PO date, PO value), read-only for everyone, shown with "Filled by GSAP"
  and the source "Simulated GSAP" (013 FR-013, FR-055).
- **FR-005**: MAPP MUST record every PO line of a run as a GSAP purchase-order transaction through the rules
  of feature 015 — document number the PO number, line the PO line, posting date the run's date, amount and
  currency the PO value, vendor code and name and description from the PR (Vendor code, Vendor, Short text),
  the line item's SWBS code, tracking key the line's HX key — in one import per run named "Simulated GSAP
  purchase order" with the PR's ID, so the line item, activity, initiative, campaign and plan Committed
  figures, Not yet committed and the Financials page include it, linked to the line item by its HX key.
- **FR-006**: MAPP MUST refuse a run of a PR whose PO already matches all its lines, with "The simulated GSAP
  already created the PO for this purchase request. Run it again after an approved change of its amounts or
  lines."
- **FR-007**: A run of an ordered PR after an approved change (top-up) MUST keep the GSAP PR and PO numbers,
  update each PO line whose PR line's amount or currency changed (value, currency and date; its transaction
  updated, never duplicated) and add PO lines for new PR lines.
- **FR-008**: Each run MUST write one activity-log entry on the PR naming the simulated GSAP, the GSAP PR and
  PO numbers and the lines created or updated, by the person who ran it, besides the financials import's own
  entry.
- **FR-009**: A run MUST change everything or nothing: the PR, its lines, the transactions and the import.
- **FR-010**: A run MUST be refused, with the reason, for a PR that is not Approved, has a change In review,
  was never routed, or while Purchase Support is off; a PR the person cannot see MUST answer not found and a
  person without "Import GSAP extracts" MUST be refused with the permission's message.
- **FR-011**: The Financials import history and the transaction details MUST name the source "Simulated GSAP
  purchase order"; the PR's GSAP data and its lines' PO fields MUST say "Simulated GSAP".
- **FR-012**: The seeded requisitioner MUST be able to complete the journey from a PR waiting for GSAP to its
  PO and Committed figures, and an end-to-end journey MUST cover the whole flow from a new activity to the PO
  (with 030 and 032).

### Key Entities

- **GSAP answer** (stored by 032): GSAP PR number, date and source on the PR; PO number, PO line, status, date
  and value on each line.
- **Simulated PO transaction** (feature 015's transaction): one per PO line, of type purchase order, keyed by
  PO number and line, linked to the line item by the HX key.
- **Simulated import**: one per run, source "Simulated GSAP", named "Simulated GSAP purchase order", with the
  PR's ID as its file name.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of PO lines the simulated GSAP creates show in their line item's Committed figure, and in
  the activity's and plan's, as soon as the run is saved.
- **SC-002**: Running the simulated GSAP again — refused or as a top-up — never duplicates a transaction (0
  duplicates across the tested cases).
- **SC-003**: A requisitioner gets from the Waiting for GSAP view to a PR with its PO in at most three actions
  (open, create, confirm).
- **SC-004**: 100% of GSAP PR and PO numbers the simulation creates start with "SIM-" and are unique.

## Assumptions

- The documented assumptions of the Clarifications section, for the product owner to confirm.
- **Dependencies**: feature 032 (PRs, routing, the Waiting for GSAP rule, the GSAP port and storage of brief
  §8.3), feature 015 (transactions and Committed), feature 030 (SWBS codes and the HX key facts).
- **Live GSAP later**: a PGSI 941 reader replaces the simulation behind the same port; the PR's fields, the
  screens and the transactions do not change (ADR-0016). The simulation does not reproduce GSAP's approvals,
  release thresholds, rejections or vendor and outline-agreement rules.
- **Out of scope**: PO copy links and documents, goods receipts and service entries, invoice approval,
  scheduled or batch runs.
