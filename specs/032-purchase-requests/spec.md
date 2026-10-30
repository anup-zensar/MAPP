# Feature Specification: Purchase Requests on MAPP's Side

**Feature Branch**: `032-purchase-requests` (no branch created: the product owner asked for no Git
branches or commits on 2026-10-27)

**Created**: 2026-10-27

**Status**: Draft (clarified with documented assumptions, see Clarifications)

**Input**: Product owner instruction of 2026-10-27: "Build Shell's purchase-request process on MAPP's side
after the SWBS code, as in Hive9 … I need to do the complete PO PR process seamlessly." Decision D-320
brings purchase requests into scope (constitution 4.4.0) as client specification 013 *minus* supporting
documents (013 FR-046/047), invoice approval (013 FR-058–FR-060), bulk creation (013 FR-002), the annual
and monthly PR rates (013 FR-017, replaced by today's rate, D-303) and e-mail (013 FR-039–FR-045,
replaced by routing inside MAPP to a requisitioner queue with a downloadable PR form).

**References**: client specification `context/specs/013-purchase-requests/spec.md` (FR-001–FR-071, PR
field inventory, status model, record operations, requisitioner attachment, list columns) · Hive9
functional analysis §C4 (Purchase Support), §C6 (approvals), §D2–§D7 (actors, the four stages, keys,
rules), §F2 (Purchase Request Configuration), §F4 (roles: PR rights separate from the PR grid; the TA
requisitioner is view-only) · gap analysis `docs/reports/planning-prpo-gap-analysis.md` §3.2, §6.1, §7 ·
decisions D-302, D-303, D-320–D-324 · ADR-0016 · features 007 (line items), 014 (access), 015
(financial transactions), 017 (approval rules), 030 (SWBS codes from the simulated GSAP), 031 (approval
of any record type), 033 (GSAP PR number and purchase order from the simulated GSAP).

**Words used here**: a *purchase request* (PR) is the request form raised in MAPP from line items — not
GSAP's requisition. A *PR line* is one claim of a PR on one line item. The *HX key* is "HX" followed by
the line item's number (line item L-1006 has HX1006). The *requisitioner* is the procurement person who
turns an approved PR into a GSAP PR. *Available to request* is a line item's planned cost minus what its PR
lines already reserve. Screens say "In review" for the status this document calls In-Review.

## Clarifications

### Session 2026-10-27

No product owner was available in the loop (brief §5). Each open point was resolved from the sources in the
order client spec > Hive9 live (FA) > workshop > guide > decision log, and is recorded here as a documented
assumption for the product owner to confirm.

- Q: Is the MoA approver a typed e-mail and alias (013 field inventory) or a person in MAPP? → A: A person
  in MAPP ("MoA approver", needed to submit), because the approval engine routes to "the person a record
  field names" (031, brief §8.2). The PR form shows that person's name and e-mail. The two text attributes
  MoA/Approver Email and MoA/Approver ID/Alias are not seeded.
- Q: May one PR draw on line items of several plans? → A: No. All lines of a PR belong to the plan of its
  first line; the PR is seen and controlled through that plan, like every plan-owned record.
- Q: Overall limit in local currency is a decimal custom attribute in 013; Shell attributes have no money
  type. → A: It is a standard PR line field (money in the line's currency, needed to submit, defaults to the
  line amount).
- Q: How are the China-specific attributes shown only for China operating units (013 FR-009)? → A: MAPP's
  attribute conditions work on the plan's Business unit only. OA vendor and Sub category (China specific)
  are seeded optional and always shown; Shell can mark them needed later.
- Q: What does In-Review mean? → A: In a first round, the PR is In-Review after a level has approved while
  further levels remain. A submitted modification of an Approved PR is In-Review from submission until it
  is decided.
- Q: Does an Approved PR keep a draft of unsubmitted changes? → A: No. Editing an Approved PR saves and
  submits the change in one step (Submit changes); editing a Submitted or In-Review PR saves and re-submits
  in one step (Save and re-submit). Only Created and Declined PRs are saved without an approval step.
- Q: How does the approved PR reach the requisitioner without e-mail? → A: Automatically on approval, MAPP
  records a routing with its recipients (the PR's Email group and PR recipients values, or "Requisitioner
  queue" when both are empty) and the PR form rows as routed; the PR appears in the "Waiting for GSAP" view
  of the Purchase requests page; the form downloads as a spreadsheet or CSV file. Routing inside MAPP cannot
  fail, so there are no retries and no "Routing failed" state.
- Q: Close-out reduces the PR "to the invoiced amount" — which invoices? → A: Invoices and credit notes in
  MAPP's GSAP data that refer to the line's PO number and PO line (feature 015 data; the PO comes from feature
  033). Until a line has a PO with invoices its invoiced amount is 0.00, so closing releases the whole line. A
  line is never raised by closing: its closed amount is the smaller of its amount and its invoiced amount.
- Q: How far does the Purchase requests page go (013 FR-061–FR-064)? → A: Grouped by vendor, PR, PR line and
  line item; search; filters on status, PR type, vendor, owner, created dates and "Waiting for GSAP"; sorting
  on a few columns; pages of 50. Column show, hide and reorder, and the filters on Email group, PR
  recipients, OA vendor, Sub category, Outline agreement and Contract holder are left for later.
- Q: Who may do what (FA F4)? → A: Four new permissions — raise and edit, delete, submit and recall, and
  see the Purchase requests page — all scoped to plans. The seeded Planner role (Hive9's Marketer) raises,
  deletes and submits PRs but does not see the page; the Global planner holds all four; a new seeded
  Requisitioner role sees plans and the page and may run the simulated GSAP (`finance.import`, used by 030
  and 033), for All markets; one fictitious requisitioner is seeded.
- Q: Which deletes are refused while a PR draws on a line item (013 FR-026)? → A: The line item and every
  record above it — activity, initiative, campaign, plan — and moving the campaign to another plan.
- Q: Is the SWBS code typed by Finance? → A: No. The gate uses the code the line item received from the
  simulated GSAP (feature 030).
- Q: May a PR line use another currency than its line item? → A: Yes (013 FR-017, edge cases). Every check
  converts into the line item's currency at today's rate (D-303) and shows both amounts.

## User Scenarios & Testing *(mandatory)*

All examples use the demo data: "FY2026 Lubricants Europe" (P-1001, Priya Raman) with "Billboard winter
wave" (A-1003, Approved, company code GB01, cost centre GB01000105) and its line item "Stand build" (L-1006,
9,200.00 EUR, Request SWBS code on); "Trade magazine ads" (A-1004, Submitted, NL02) with "Media booking"
(L-1005, 30,000.00 USD); "Social always-on" (A-1002, Approved, DE03) with "Speaker fees" (L-1004, 5,000.00 USD,
Request SWBS code off); seeded rates EUR 0.92 and GBP 0.79 per USD. Daniel Okoro approves (role Approver);
Lea Santos is the seeded requisitioner.

### User Story 1 - Raise a purchase request from a line item (Priority: P1)

A marketer opens a line item. Its **Purchase requests** section shows what is already requested and what is
still available, and offers **New purchase request**. The new PR opens with its header fields and one line
for that line item, already filled with everything still available. The marketer fills in the header and
saves. The PR is **Created** and the money is reserved on the line item at once, whatever the activity's
status.

**Why this priority**: every purchase Shell makes from a plan starts with this form; without it nothing can
be sent to the requisitioner, and no GSAP PR or PO can be traced back to the plan.

**Independent Test**: as Priya Raman, open "Stand build" (L-1006), choose New purchase request, fill in the
required fields and save. The PR is Created with one 9,200.00 EUR line showing HX1006; the line item shows
Total requested 9,200.00 EUR and Available to request 0.00 EUR.

**Acceptance Scenarios**:

1. **Given** "Stand build" with no PR lines, **When** Priya chooses New purchase request, **Then** a new PR
   opens with Name "PR for Stand build", Owner Priya Raman and one line for L-1006 with Currency EUR, Amount
   9,200.00 and Overall limit 9,200.00. The HX key HX1006 is shown read-only with "Generated from the
   line-item ID" and a copy button.
2. **Given** that new PR with Approval workflow, Approval workflow for modification and the line's Delivery
   date empty, **When** Priya saves, **Then** nothing is saved and all three problems are listed together,
   in a summary and next to each field.
3. **Given** a saved PR with a 6,000.00 EUR line on L-1006, **When** Priya creates a second PR from L-1006,
   **Then** its line is pre-filled with 3,200.00 EUR; saving it with 3,500.00 EUR is refused with "Only
   3,200.00 EUR of line item L-1006 is not yet requested."
4. **Given** "Media booking" (L-1005) on the Submitted activity "Trade magazine ads", **When** Priya creates
   and saves a PR on it, **Then** the PR is saved with status Created.
5. **Given** PR A with a 6,000.00 EUR line and PR B with a 3,200.00 EUR line on L-1006, **When** anyone who
   can see the plan opens L-1006, **Then** Total requested shows 9,200.00 EUR and Available to request 0.00
   EUR, and "Billboard winter wave" shows its Total requested, in the plan currency, on its activity page.
6. **Given** L-1005 with 4,000.00 USD still available, **When** Priya enters a line of 3,700.00 EUR,
   **Then** the save is refused because 3,700.00 EUR is 4,021.74 USD at today's rate; the message shows both
   amounts.
7. **Given** Purchase Support is switched off in Settings, **When** anyone opens a line item, **Then** New
   purchase request is not offered and existing PRs open read-only with "Purchase requests are switched off
   for your organisation."
8. **Given** Sara Novak (Viewer, no permission to raise PRs), **When** she opens L-1006, **Then** she sees
   the Purchase requests section without New purchase request, and MAPP refuses a PR she tries to save another way.

---

### User Story 2 - Submit the purchase request and get it approved (Priority: P1)

The marketer chooses **Submit**. MAPP accepts it only when every line's activity is approved, every line's
line item has received its SWBS code from the simulated GSAP, all lines charge one company code and every
field needed to submit is filled. The approval engine then runs the PR's approval workflow; the PR is
**Submitted**, **In review** while further levels remain, then **Approved** or **Declined**. Self-approval
is allowed. **Re-submit** starts a new round from Submitted, In review or Declined; **Recall** takes it back
to Created. Approvers find the PR in the Approvals inbox.

**Why this priority**: only an approved PR may become a GSAP PR. The gates stop requests GSAP would refuse.

**Independent Test**: with a Created PR on L-1006 after the simulated GSAP returned L-1006's SWBS code,
Priya submits; Daniel Okoro (the MoA approver) approves from the Approvals page. The PR is Approved and its
approval history shows the level, approver, date, outcome and comment.

**Acceptance Scenarios**:

1. **Given** a Created PR on L-1005, whose activity "Trade magazine ads" is Submitted, **When** Priya
   chooses Submit, **Then** it is refused with "Activity 'Trade magazine ads' must be approved before this
   purchase request can be submitted."
2. **Given** a Created PR on L-1006 whose SWBS status is "Ready to send to GSAP", **When** Priya submits,
   **Then** it is refused with a message naming L-1006 "Stand build" and its SWBS status.
3. **Given** a PR that meets both gates but has Short text, Vendor, MoA approver and the line's Overall
   limit empty, **When** Priya submits, **Then** one answer lists all four, and the PR stays Created.
4. **Given** a PR whose lines' activities have company codes GB01 and NL02 (one changed after the lines
   were added), **When** Priya submits, **Then** it is refused with "One purchase request cannot charge two
   company codes (GB01 and NL02)."
5. **Given** a PR that meets every gate, **When** Priya submits, **Then** it becomes Submitted; its
   Approval section shows the first level waiting and later levels waiting or not needed; the PR shows
   "Awaiting approval — saving changes will restart approval."
6. **Given** a Submitted PR, **When** Daniel declines it with "Wrong vendor", **Then** it becomes Declined,
   the open level shows Cancelled, the comment is in the history, the money stays reserved, and Priya can
   edit and Re-submit; she is notified.
7. **Given** a Submitted PR on a two-level workflow, **When** level 1 approves, **Then** the PR is In review;
   **When** level 2 approves, **Then** it is Approved and the history lists both levels.
8. **Given** a workflow whose level 2 applies only from 100,000.00 USD, **When** a 9,200.00 EUR PR is
   submitted, **Then** level 2 shows "Not needed" and approval at level 1 approves the PR.
9. **Given** a workflow whose only approver is the MoA approver and Priya names herself MoA approver,
   **When** Priya approves her own PR, **Then** the approval is accepted and recorded.
10. **Given** a Submitted PR, **When** Priya chooses Recall, **Then** the open level shows Cancelled with
    "Recalled" and the PR returns to Created.
11. **Given** a Submitted PR, **When** Priya changes the line's delivery date and chooses Save and
    re-submit, **Then** the open round ends as superseded, a new round starts at the first applicable level
    and the PR stays Submitted.
12. **Given** Daniel opens the Approvals page, **Then** the PR waiting for him is listed with its name,
    amount and owner; selecting it shows a preview with the PR's lines, total, approval chain and Approve
    and Decline.

---

### User Story 3 - Route the approved PR to the requisitioner inside MAPP (Priority: P1)

When a PR is approved, MAPP routes it to the requisitioner at once: it records the routing with its
recipients and the PR form as routed, and sets each line's Sent date. The requisitioner finds it in the
**Waiting for GSAP** view of the Purchase requests page and downloads the **PR form** — one row per line
with everything needed to type the GSAP PR, the HX key first. The owner can **Resend PR form** at any
time; the routing history lists every routing.

**Why this priority**: this hand-off is the only way an approved request reaches GSAP, and the HX key on
the form is the only thread that brings GSAP's PR, PO and invoices back to the right line item. E-mail is
out of scope (D-320), so the hand-off happens inside MAPP.

**Independent Test**: approve a PR with Email group "DE_LUBES" and one line on L-1006. As Lea Santos, open
Purchase requests › Waiting for GSAP, open the PR and download its PR form; the row holds HX1006 and every
column of the form.

**Acceptance Scenarios**:

1. **Given** a PR with Email group "DE_LUBES" and one 9,200.00 EUR line on L-1006 (SWBS code received, GL
   code and cost centre GB01000105 on its activity), **When** the last approver approves it, **Then** in the
   same moment the PR's routing history shows "Original · routed to DE_LUBES" with the time and the
   approver, the line's Sent date is today, and the PR is listed under Waiting for GSAP.
2. **Given** that routing, **When** Lea downloads the PR form, **Then** the spreadsheet has one row with the
   PR ID, PR name, PR line ID, HX1006, the short text, the vendor and vendor code, GB01000105, L-1006's SWBS
   code, its GL code, 9,200.00, EUR, the delivery date, the MoA approver's name and e-mail, the routing type
   "Original" — and notes saying it was routed by MAPP, when, and that documents are not attached.
3. **Given** an approved PR with neither Email group nor PR recipients, **When** it is routed, **Then** the
   recipients read "Requisitioner queue".
4. **Given** a PR routed on 14 Oct 2026, **When** Priya chooses Resend PR form on 16 Oct 2026, **Then** a
   routing "Resend" with today's values is recorded, the history lists both, and Sent date stays 14 Oct 2026.
5. **Given** a Created PR, **When** anyone looks for the PR form, **Then** it is not offered: "The PR form is
   created when the purchase request is approved."
6. **Given** Lea Santos (Requisitioner), **When** she opens the PR, **Then** she can read everything and
   download the form, and sees no Edit, Submit, Close or Delete.

---

### User Story 4 - Manage the lines of a purchase request (Priority: P2)

In the PR's edit page the marketer adds lines with **Add line**, choosing from a searchable list of line
items of the same plan and company code that shows what each still has available; edits amounts, currency,
delivery date, overall limit and the line's Shell attributes; and removes lines from Created or Declined
PRs after a confirmation.

**Why this priority**: agencies often bill one PR across several line items; the single-line flow of story
1 already delivers value without this.

**Independent Test**: in a Created PR on L-1006, add a line for another GB01 line item, search the picker
for an NL02 line item and read why it is unavailable, change an amount and remove a line, checking the
confirmation text and the released balance.

**Acceptance Scenarios**:

1. **Given** a Created PR with a line on L-1006 (GB01), **When** Priya chooses Add line and picks a GB01
   line item, **Then** a second line appears, pre-filled with that line item's available amount and its HX
   key.
2. **Given** the same PR, **When** Priya searches the picker for "Media booking" (NL02), **Then** it is
   listed as unavailable with "One purchase request cannot charge two company codes (GB01 and NL02)."
3. **Given** a line item already on the PR, a line item with nothing left to request and a line item of
   another plan, **When** Priya searches for them, **Then** the first two are listed unavailable with
   "Already on this purchase request" and "Nothing left to request", and the third is not listed.
4. **Given** a line on L-1006 with 3,200.00 EUR available, **When** Priya enters 3,500.00 EUR and clears the
   delivery date and saves, **Then** nothing is saved and both problems are shown next to their fields.
5. **Given** a Created PR with a 3,200.00 EUR line on L-1006, **When** Priya removes it, **Then** a
   confirmation asks "Remove the line for line item L-1006 (3,200.00 EUR) from this purchase request? The
   3,200.00 EUR reserved on the line item will be released." After saving, the line is in the recycle bin
   and L-1006 has 3,200.00 EUR more available.
6. **Given** an Approved PR, **When** Priya looks at a line, **Then** Remove line is unavailable with "Lines
   of approved purchase requests cannot be removed — close the purchase request instead."
7. **Given** a PR with one line, **When** Priya tries to remove it, **Then** it is refused with "A purchase
   request needs at least one line — delete the purchase request instead."

---

### User Story 5 - Find, view, edit and delete purchase requests (Priority: P2)

People who follow PRs across plans — requisitioners, finance, business champions — open **Purchase
requests** from the sidebar. The list is grouped by vendor; each PR opens to its lines and their line items.
It offers search, filters and a Waiting for GSAP view. Opening a PR shows its details page with header,
lines, approval, routing history and history. Created and Declined PRs can be deleted to the recycle bin;
nothing that a PR draws on can be deleted.

**Why this priority**: this is the requisitioner's main page and the cross-plan list of commitments; single
PRs are still reachable from their line items without it.

**Independent Test**: with PRs for two vendors, open Purchase requests, find one PR by its HX key, open it,
edit it with two errors and see both, leave with unsaved changes and get the prompt; delete a Created PR and
check the confirmation; check that an approved PR offers no delete and says why.

**Acceptance Scenarios**:

1. **Given** PRs for vendors "Northlight Events GmbH" and "Brightside Media BV", **When** Lea opens Purchase
   requests, **Then** both vendors are listed with their number of PRs and total; each vendor lists its
   PRs, each PR its lines and each line its line item, with the columns of FR-044.
2. **Given** many PRs, **When** Lea searches "HX1006", a PO number, "Brightside" or a GSAP PR number,
   **Then** only matching PRs are listed; search also matches PR ID and name.
3. **Given** PRs of several types and statuses, **When** Lea filters PR type "Standard PR/PO" and status
   "Approved", **Then** only those remain; choosing "Not completed" under PR type lists PRs without a type.
4. **Given** the Waiting for GSAP view, **Then** it lists exactly the approved, routed PRs that GSAP has not
   answered yet.
5. **Given** a Created PR in its edit page, **When** Priya clears Name and types "abc" as Vendor email
   address and saves, **Then** nothing is saved and both problems are shown together.
6. **Given** unsaved changes, **When** Priya tries to leave, **Then** she is asked to save, discard or stay.
7. **Given** a Created PR "PR for Stand build" with two lines, on L-1006 and on "Gift boxes" (a line item
   Priya added to "Christmas trade gifting", also GB01), **When** Priya chooses Delete purchase request,
   **Then** the confirmation reads "Delete purchase request 'PR for Stand build'? Its 2 lines will be
   removed and the money reserved on line items L-1006 and <Gift boxes' ID> will be released. You can
   restore it from the recycle bin." After confirming, the PR is in the recycle bin with Undo and the money
   is released.
8. **Given** an Approved PR, **Then** Delete is unavailable with "Approved purchase requests cannot be
   deleted — close them instead."; a Submitted one with "Purchase requests awaiting approval cannot be
   deleted — recall them first."
9. **Given** a deleted PR whose line items' balance has since been taken by another PR, **When** someone
   restores it, **Then** the restore is refused naming each line item and its remaining available amount.
10. **Given** any PR, **When** the user points at GSAP PR number, PO number or Total invoiced, **Then** the
    field is read-only and explains "Filled by GSAP."
11. **Given** a PR drawing on L-1006, **When** Priya deletes L-1006, its activity, initiative, campaign or
    plan, or moves the campaign to another plan, **Then** it is refused naming the PR; lowering L-1006's
    planned cost below its Total requested is refused naming the PRs.
12. **Given** a person without the Purchase requests page permission, **Then** the sidebar has no Purchase
    requests entry and the page's address shows "You don't have access to this page".

---

### User Story 6 - Modify or close an approved purchase request (Priority: P2)

A marketer changes an Approved PR — amounts, delivery dates, header fields, or new lines if Settings allow —
and chooses **Submit changes**; the change goes through the PR's approval workflow for modification while the
approved version stays visible. Once the work is done, **Close purchase request** reduces the PR to what GSAP
has invoiced and returns the rest to the plan.

**Why this priority**: purchases change after approval, and unused money on a finished PR must return to the
plan; the raise–approve–route flow works without this.

**Independent Test**: on an Approved 9,200.00 EUR PR, raise the line to 9,500.00 EUR after giving the line
item room; submit the change, approve it, and check routing type "Modification". Then close the PR with no
invoices and check that 9,500.00 EUR is released.

**Acceptance Scenarios**:

1. **Given** an Approved PR with a 6,000.00 EUR line on L-1006 (3,200.00 EUR still available), **When**
   Priya changes the line to 8,000.00 EUR and chooses Submit changes, **Then** the PR is In review on its
   modification workflow, 8,000.00 EUR is reserved, and the Approved version (6,000.00 EUR) stays visible
   until the change is decided.
2. **Given** that change is approved, **Then** the PR is Approved with 8,000.00 EUR and is routed again as
   "Modification" with the changed line marked.
3. **Given** that change is declined with a comment, or recalled, **Then** the PR is Approved with 6,000.00
   EUR again, the reservation returns to 6,000.00 EUR and the comment is recorded.
4. **Given** the setting Lines on approved purchase requests is "Not allowed", **When** Priya looks for Add
   line on an Approved PR, **Then** it is unavailable with "Your organisation does not allow lines to be
   added to approved purchase requests."
5. **Given** the setting is "Allowed without approval", **When** Priya adds a line to an Approved PR and
   saves without other changes, **Then** the PR stays Approved with the new line, reserved, and is routed as
   "Modification".
6. **Given** an Approved PR with a 9,200.00 EUR line on which GSAP data shows 7,500.00 EUR invoiced, **When**
   Priya chooses Close purchase request, **Then** a confirmation lists per line the current amount
   9,200.00, invoiced 7,500.00 and to be released 1,700.00 EUR; after confirming, the line is 7,500.00 EUR,
   the PR is Closed, L-1006 has 1,700.00 EUR more available and the PR is routed as "Close".
7. **Given** an Approved PR with no invoices, **When** Priya closes it, **Then** every line becomes 0.00 and
   all its money is released.
8. **Given** a Closed PR, **Then** every field is read-only with "Closed purchase requests cannot be
   changed."; Resend PR form stays available.
9. **Given** a modified line amount below the line's total invoiced, **When** Priya submits the change,
   **Then** it is refused with the invoiced amount.

---

### User Story 7 - Configure purchase requests and who may use them (Priority: P2)

An administrator switches Purchase Support on or off and chooses whether lines may be added to approved
PRs, in **Settings › Purchase requests**. The PR and PR line attributes Shell uses are configuration on the
Shell attributes page, under two new levels. Roles carry the new purchase-request permissions; a
Requisitioner role and one demo requisitioner are seeded.

**Why this priority**: the seeded configuration lets the flow run; changing it is an administrator task.

**Independent Test**: as Alex Morgan, open Settings › Purchase requests, switch Purchase Support off, check a
line item, switch it on again; open Shell attributes › Purchase request and see Vendor, PR type and the
other seeded attributes.

**Acceptance Scenarios**:

1. **Given** Alex Morgan (Administrator), **When** he opens Settings › Purchase requests, **Then** he sees
   Purchase Support (on) and Lines on approved purchase requests (Allowed with modification approval), can
   change both and save; the change is in the activity log.
2. **Given** Priya (no `settings.manage`), **When** she opens Settings › Purchase requests, **Then** she sees
   the values read-only.
3. **Given** Settings › Shell attributes, **Then** the levels "Purchase request" and "PR line" list the
   seeded attributes of the field inventory below, with their types, options and needed-to-submit marks;
   the attributes MAPP reads (Vendor, Vendor code, Vendor email address, Short text, PR type, Email group,
   PR recipients) cannot change level or type or be deleted, and say why.
4. **Given** Settings › Roles and access, **Then** the four purchase-request permissions appear under
   "Purchase requests", and the Requisitioner role holds View plans, See the Purchase requests page and
   Import GSAP extracts.

---

### Edge Cases

- **SWBS code arrives after a refused submission**: once the simulated GSAP stores the code (030), the same
  PR can be submitted with no other change.
- **The activity leaves approval after the PR is approved** (its planned cost or dates change): the approved
  PR and its routing are unaffected; new submissions, re-submissions and changes of PRs on that activity's
  line items are refused, with the reason, until the activity is approved again.
- **Line item planned cost or currency changed below its Total requested**: refused, naming the PRs.
- **A company code changes on an activity after its line item is on a PR**: the PR stays saved; submitting
  is refused until all lines charge one company code again.
- **Two people claim the same balance or edit the same PR at once**: the later save is refused (version
  conflict, or "Only … is not yet requested"); entries stay visible; a reload is offered; nothing is
  overwritten.
- **SWBS code changed by GSAP after routing**: the PR shows the line item's current code; the routing keeps
  the code that was routed; Resend PR form routes the new one.
- **Exchange rate changes after a PR line in another currency was saved**: the reservation follows today's
  rate; a line item may then show a negative Available to request, labelled "Requested above planned cost
  by …"; new lines are refused until there is room again.
- **Approval workflow made inactive while a PR is in it**: the open round finishes on its copy (017); new
  submissions need an active workflow of the right type.
- **Purchase Support switched off while PRs exist**: they stay readable; nobody can create, change, submit,
  approve, close, delete or restore a PR until it is switched on again; the Purchase requests page leaves
  the sidebar.
- **Owner leaves Shell (made inactive)**: the PR keeps its owner, shown "(inactive)"; anyone allowed to edit
  can change the owner of a PR that is not Closed.
- **A line item in the recycle bin**: cannot be added to a PR (only live line items are offered).
- **Empty list**: when no PR matches, the page says so and offers Clear filters; when there are no PRs at
  all, it explains that PRs are raised from a line item's Purchase requests section.
- **Purchase requests page with Purchase Support off**: the address shows "Purchase requests are switched
  off for your organisation."

## Requirements *(mandatory)*

Client requirement IDs are cited as "013 FR-0nn". Out of scope by D-320: 013 FR-002, FR-017 (rates),
FR-039–FR-047 as e-mail and documents, FR-056 (PR transactions list), FR-058–FR-060, and the GRSP and Actuals
amounts (no goods-receipt data exists).

### Functional Requirements

#### Creating a purchase request

- **FR-001**: Users MUST be able to create a PR from a line item's Purchase requests section (New purchase
  request) or from the Purchase requests page after picking a line item, whatever the activity's status
  (013 FR-001).
- **FR-002**: System MUST treat a line item as eligible for a new PR line only when it is live (not in the
  recycle bin), belongs to the PR's plan and has more than 0.00 available to request (013 FR-003).
- **FR-003**: System MUST pre-fill a new PR: Name "PR for <line item name>", Owner the acting person, and
  one line in the line item's currency for its whole available amount, with Overall limit equal to that
  amount; every pre-filled value stays editable (013 FR-004, FR-018).
- **FR-004**: System MUST honour the Purchase Request Configuration: Purchase Support on or off (seed on)
  and Lines on approved purchase requests (Not allowed · Allowed with modification approval (seed) · Allowed
  without approval). While Purchase Support is off, no PR can be created, changed, submitted, approved,
  closed, deleted or restored, the Purchase requests page is not offered, and PRs show "Purchase requests
  are switched off for your organisation." (013 FR-005, FR-051).

#### Header

- **FR-005**: A PR MUST have these standard fields: Name (needed to save, 1–120 characters), Description
  (0–2,000), Owner (needed to save; a person; defaults to the acting person), MoA approver (needed to
  submit; a person), Approval workflow (needed to save; an active workflow of type Purchase request) and
  Approval workflow for modification (needed to save; an active workflow of type Modified purchase
  request) (013 FR-006).
- **FR-006**: System MUST offer the PR and PR line custom attributes as Shell attributes at two new levels,
  "Purchase request" and "PR line", managed in Settings like every level and seeded as the PR Field
  Inventory below states; an attribute marked needed to submit blocks submitting, never saving (013 FR-007,
  FR-008, FR-015).
- **FR-007**: System MUST refuse a Vendor email address that is filled but not a well-formed e-mail address
  (013 FR-010).
- **FR-008**: System MUST show read-only on the header: Total amount (sum of the lines, in the reader's
  display currency), Company code and Cost centre(s) of the lines' activities (every distinct value), the
  HX keys of the lines, Total invoiced, and GSAP PR number and date (013 FR-011, FR-012, FR-028).
- **FR-009**: GSAP PR number, GSAP PR date, PO number, PO line, PO line status, PO date, PO value and Total
  invoiced MUST be read-only for every person, labelled "Filled by GSAP", and change only through GSAP's
  answer (feature 033) and GSAP data (feature 015) (013 FR-013).

#### Lines

- **FR-010**: Users MUST add a line only through Add line and a line-item picker that searches by name, ID
  and HX key, shows each line item's available amount, offers eligible line items (FR-002) whose activity
  has the PR's company code, and lists matching ineligible ones as unavailable with the reason (013 FR-014).
- **FR-011**: A line MUST have: Line item (name and ID) and HX key, read-only; Currency (needed to save; an
  active currency; defaults to the line item's); Amount (needed to save; above 0.00); Delivery date (needed
  to save; a calendar date); Overall limit (needed to submit; 0.00 or more; defaults to the amount);
  Description (0–2,000); its PR line attributes; and read-only SWBS code ("Filled by GSAP through the line
  item"), Sent date and the fields of FR-009 (013 FR-015).
- **FR-012**: System MUST refuse a line amount that, converted into the line item's currency at today's
  rate, is above the line item's planned cost minus every other PR line on it — PRs not in the recycle bin,
  Closed PRs at their closed amount — with a message giving the available amount and, for another
  currency, both amounts (013 FR-016, FR-017 per D-303). Example: 9,200.00 EUR planned with a 6,000.00 EUR
  line elsewhere accepts up to 3,200.00 EUR; 3,200.01 EUR is refused.
- **FR-013**: System MUST refuse lines whose activities have different filled company codes ("One purchase
  request cannot charge two company codes (GB01 and NL02).") and lines of another plan than the PR's (013
  FR-019).
- **FR-014**: System MUST allow a line item at most once on one PR, and on any number of PRs within its
  available amount (013 FR-020).
- **FR-015**: Users MUST be able to remove a line from a Created or Declined PR after a confirmation; the
  line goes to the recycle bin and its reservation is released at once; a PR keeps at least one line; on
  other PRs Remove line is unavailable with the reason (013 FR-021).
- **FR-016**: System MUST refuse a save based on an out-of-date copy of the PR (who changed it and when),
  keep the person's entries and offer a reload, and never overwrite silently (013 FR-022).

#### Reservation

- **FR-017**: System MUST reserve each line's amount on its line item from the first save, whatever the
  status, and release it only when a line is removed or lowered before approval, the PR is deleted, an
  approved modification lowers it, a declined or recalled modification restores a lower approved amount,
  or the PR is closed. Declining a PR MUST NOT release it (013 FR-023, FR-024).
- **FR-018**: System MUST show on a line item its Total requested (its reserved line amounts in its
  currency) and Available to request, and on an activity its Total requested (the sum of its line items, in
  the plan currency) (013 FR-025).
- **FR-019**: System MUST refuse to delete a line item, or a record above it (activity, initiative,
  campaign, plan), and to move a campaign to another plan, while a PR not in the recycle bin draws on a line
  item in it; and refuse to lower a line item's planned cost, or change its currency, so that its Total
  requested would be above its planned cost. Each refusal names the PRs (013 FR-026).

#### HX key

- **FR-020**: Every line MUST show its HX key — "HX" followed directly by the line item's number (L-1006 →
  HX1006) — read-only, copyable, labelled "Generated from the line-item ID"; the same key on every line of
  that line item; on the PR header and in the PR form; never typed by anyone (013 FR-027–FR-029).

#### Submission and approval

- **FR-021**: System MUST refuse Submit, Re-submit, Save and re-submit and Submit changes unless: the PR has
  a line; every line's activity is Approved (In progress and Complete included); every line's line item
  has received its SWBS code; all lines' activities have the same, filled company code; the workflow of the
  round about to start (approval, or approval for modification) is active and of the right type; and every field needed to submit (MoA approver, overall limits, attributes
  marked needed to submit) is filled. Every failure MUST be reported at once, naming the activities, line
  items and fields (013 FR-030, FR-007).
  *Amended by [037 FR-019](../037-overspend-control/spec.md) (D-349): also refused while any line's activity
  has line items exceeding its planned cost (negative unassigned planned cost), whatever its budgets'
  overspend setting, naming each such activity and the overage with the other problems.*
- **FR-022**: System MUST run a submission through the Approval workflow (approval rounds of type Purchase
  request) and Submit changes through the Approval workflow for modification (type Modified purchase
  request), using the approval engine of feature 031, and set the PR status exactly as the Status Model
  below states (013 FR-031, FR-032).
- **FR-023**: System MUST offer Submit on Created PRs and Re-submit on Submitted, In review (first round)
  and Declined PRs; a change saved to a Submitted or In review PR MUST be saved as Save and re-submit, which
  ends the open round as superseded and starts a new one at the first applicable level; the PR MUST show
  "Awaiting approval — saving changes will restart approval." before the person edits (013 FR-033).
- **FR-024**: The owner or the person who submitted MUST be able to Recall a PR being approved: a first
  round returns it to Created (open levels Cancelled, "Recalled"); a modification round returns it to
  Approved with its last approved values and reservation (013 FR-034).
- **FR-025**: Declining MUST need a comment; the owner MUST be able to edit a Declined PR and Re-submit it
  (013 FR-035).
- **FR-026**: The PR MUST show the approval history of every round, first approval and modification:
  level, approver, date, outcome and comment (013 FR-036).
- **FR-027**: System MUST accept self-approval when the PR's workflow makes the owner or the MoA approver
  an approver; the activities' rule "Nobody approves an activity they own or submitted" does not apply to
  PRs (013 FR-037; D-324).
- **FR-028**: System MUST record the MoA approver and pass them to the requisitioner on the PR form, and
  MUST NOT check MoA authority, legal entities, amount thresholds or PO release thresholds (013 FR-038).
- **FR-029**: Approvers MUST find PRs waiting for them on the Approvals page with a preview (header, lines,
  total, approval chain) and Approve and Decline there or on the PR page; the owner MUST be notified in
  MAPP when the PR is approved or declined (013 FR-071 in part; 031).

#### Routing to the requisitioner

- **FR-030**: When a PR is approved — first approval, approved modification, or a line added without
  approval — System MUST route it inside MAPP in the same step: a routing with its time, the person whose
  approval completed it, its type (Original, Modification, Resend or Close), its recipients (the PR's Email
  group and PR recipients values, or "Requisitioner queue" when both are empty) and the PR form rows as
  routed; each line's Sent date MUST be set at its first routing and never change (013 FR-039, FR-040,
  FR-042 per D-320).
- **FR-031**: Each routing MUST be downloadable as the PR form — a spreadsheet, or a CSV file — with one row
  per line and the columns of the Requisitioner Attachment table below, the lines changed by a
  modification marked (013 FR-041).
- **FR-032**: The Purchase requests page MUST offer the view Waiting for GSAP: Approved PRs that have been
  routed and that GSAP has not answered yet (no PO, or a line whose amount or currency differs from its PO
  value, or a line without a PO line) — the requisitioner queue (D-320).
- **FR-033**: Users MUST be able to Resend PR form on an Approved or Closed PR that was routed: a routing of
  type Resend with today's values; Sent dates do not change (013 FR-044).
- **FR-034**: Closing a PR MUST route it with type Close, giving the closed amounts (013 FR-045).
- **FR-035**: The PR MUST show its routing history: every routing with time, type, recipients, who, and the
  PR form download (013 FR-042).

#### Modification of approved purchase requests

- **FR-036**: Any change to an Approved PR MUST be a modification, saved and submitted in one step with
  Submit changes, passing the gates of FR-021; the PR becomes In review on its modification workflow and
  shows its last approved values as the Approved version until the change is decided (013 FR-048).
- **FR-037**: System MUST reserve the modified amounts while the change is In review; an approved change
  makes them the PR's values and routes it (Modification); a declined or recalled change restores the last
  approved values and reservation (013 FR-049).
  *Amended by [038 FR-001 – FR-003](../038-qa-findings/spec.md) (QA C-1): while the change is In review each line
  reserves the larger of its approved and its modified amount (a removed line keeps its approved amount), so a
  lowered amount is released only when the change is approved, as FR-017 says.*
- **FR-038**: System MUST refuse a modified line amount below the line's Total invoiced or above the limit
  of FR-012 (013 FR-050).
- **FR-039**: Add line on an Approved PR MUST follow Lines on approved purchase requests: Not allowed —
  unavailable with "Your organisation does not allow lines to be added to approved purchase requests.";
  Allowed with modification approval — part of Submit changes; Allowed without approval — a save that only
  adds lines keeps the PR Approved and routes it as Modification (013 FR-051).

#### Close and delete

- **FR-040**: System MUST refuse to delete a Submitted, In review, Approved or Closed PR, saying why (013
  FR-052).
- **FR-041**: System MUST offer Close purchase request on an Approved PR with no change In review; before
  confirming it shows per line the current amount, the invoiced amount and the amount to be released; on
  confirmation each line amount becomes the smaller of its amount and its Total invoiced and the PR becomes
  Closed. A Closed PR cannot be changed, modified, deleted or reopened, and still shows GSAP data (013
  FR-053).
- **FR-042**: Deleting a Created or Declined PR MUST, after the confirmation of the Record Operations table,
  move it with its lines to the recycle bin, release its reservations and offer Undo; a restore brings it
  back Created only when every line still fits its line item's available amount, otherwise it is refused
  naming each line item and its available amount (013 FR-054).

#### GSAP data on the purchase request

- **FR-043**: The PR MUST show GSAP data: GSAP PR number, date and source on the header; PO number, PO line,
  PO line status, PO date and PO value on each line (empty until GSAP answers, feature 033); Total invoiced
  per line and per PR — invoices minus credit notes in MAPP's GSAP data that refer to the line's PO number
  and PO line, converted at today's rate (013 FR-055, FR-057 in part).

#### Purchase requests page

- **FR-044**: A Purchase requests page MUST list PRs grouped by vendor (each vendor with its number of PRs
  and total), each PR expandable to its lines and each line showing its line item, with the columns Name,
  ID, Level, Status, Amount, Owner, Created, Delivery date, Sent date, PO number, PO line, PO line status,
  Invoiced and Approval workflow (013 FR-061, Purchase Requests List minus GRSP).
- **FR-045**: Users MUST be able to search by PR ID, name, vendor, HX key, GSAP PR number and PO number, and
  filter by Status, PR type and Vendor (each with "Not completed"), Owner, Created date range and Waiting for
  GSAP; active filters are visible and removable one by one; filters, search, sort and page live in the
  address so Back keeps them (013 FR-062).
- **FR-046**: Users MUST be able to sort by name, ID, status, amount and created date within the vendor
  groups, page through 50 PRs at a time and open the PR or line item behind a row (013 FR-063).
- **FR-047**: Amounts on the page MUST be in the reader's display currency with the currency code; line
  amounts in their own currency (013 FR-064).
- **FR-048**: The page MUST render loading, empty (no PRs, and no match with Clear filters), error, no
  permission and success states.

#### Access

- **FR-049**: System MUST add four permissions, all applying to plans in the role's scope, in a new group
  "Purchase requests": Raise and edit purchase requests; Delete purchase requests; Submit purchase
  requests for approval and recall them; See the Purchase requests page. Seeded roles: Administrator all;
  Global planner all four; Planner the first three; Approver and Viewer none; and a new Requisitioner
  role — View plans, See the Purchase requests page, Import GSAP extracts — proposed for All markets (FA
  F4).
- **FR-050**: A person MUST see a PR when they see its plan, own it, are its MoA approver, are named as an
  approver in one of its rounds, or may see the Purchase requests page for its plan; any other PR answers
  "not found" like a missing one. Approving and declining follow the approval engine's rules (approvals
  permission and being a named approver of the waiting level).
- **FR-051**: The demo data MUST include one fictitious requisitioner with the Requisitioner role, the
  Purchase Request Configuration, the PR attributes, two PR workflows (approval and modification, approver
  the MoA approver) and one Created PR, all fictitious.

#### Working with records

- **FR-052**: PRs and PR lines MUST follow the standard record pattern: a named create action, a searchable
  list, a details page with Edit, delete to the recycle bin with Undo, a status chip (word and colour) and a
  completeness chip ("3 things needed to submit") that links to what is missing (013 FR-065).
- **FR-053**: Saving MUST check the whole PR, header and lines, and show every problem at once, in a summary
  and next to each field (013 FR-066).
- **FR-054**: Leaving a PR with unsaved changes MUST ask to save, discard or stay (013 FR-067).
- **FR-055**: Delete, remove line and close MUST ask for confirmation naming the record and stating what
  will be removed and released; cancelling changes nothing (013 FR-068).
- **FR-056**: Every unavailable action and read-only field MUST say why, in the words of the Record
  Operations table, the Field Inventory and the Status Model (013 FR-069).

#### Audit and notifications

- **FR-057**: System MUST write an activity-log entry for: PR create, change (each changed field), line
  add and remove, submit, re-submit, recall, approve, decline, modification submit, approve and decline,
  routing, resend, close, delete, restore and changes of the Purchase Request Configuration; the PR page has
  a History section (013 FR-070 minus documents and invoices).
- **FR-058**: The owner MUST be notified in MAPP when the PR is approved or declined, and the notification
  MUST open the PR (013 FR-071 in part).

### Record Operations

| Record | Create | List | Details | Edit | Delete / remove (confirmation) | Unavailable and why |
|---|---|---|---|---|---|---|
| Purchase request | **New purchase request** on a line item's Purchase requests section, or on the Purchase requests page after picking a line item | Purchase requests page (FR-044–FR-048); a line item's Purchase requests section | Header, Lines, Approval, Routing history, History | Created, Declined: Save. Submitted, In review: Save and re-submit. Approved: Submit changes. Closed: none | **Delete purchase request** (Created, Declined): "Delete purchase request '<name>'? Its <n> lines will be removed and the money reserved on line items <IDs> will be released. You can restore it from the recycle bin." | "Approved purchase requests cannot be deleted — close them instead." · "Purchase requests awaiting approval cannot be deleted — recall them first." · "Closed purchase requests cannot be changed." · "Purchase requests are switched off for your organisation." |
| PR line | **Add line** in the PR's edit page, through the picker (FR-010) | Lines of the PR | Line item, HX key, amounts, delivery date, attributes, GSAP PO data, invoiced | With its PR (Approved: through a modification) | **Remove line** (Created, Declined): "Remove the line for line item <ID> (<amount>) from this purchase request? The <amount> reserved on the line item will be released." | "Lines of approved purchase requests cannot be removed — close the purchase request instead." · "Your organisation does not allow lines to be added to approved purchase requests." · "A purchase request needs at least one line — delete the purchase request instead." · GSAP fields: "Filled by GSAP" · HX key: "Generated from the line-item ID" · SWBS code: "Filled by GSAP through the line item" |
| Routing | Automatic on approval; **Resend PR form** (Approved, Closed) | Routing history of the PR; Waiting for GSAP view | Time, type, recipients, who, PR form download | Never changed | Never deleted | Before approval: "The PR form is created when the purchase request is approved." |
| Purchase Request Configuration | Seeded | Settings card "Purchase requests" | Purchase Support, Lines on approved purchase requests | Settings permission | Not deletable | Without the permission: read-only, "Changing Settings isn't part of your role …" (014) |

### PR Field Inventory (seed configuration)

Header standard fields: FR-005. Line standard fields: FR-011. Shell attributes seeded at level **Purchase
request** (in this order, section "Purchase request"); MAPP reads the ones with a role:

| Attribute | Type | Needed to submit | Seed options / notes | Role |
|---|---|---|---|---|
| Material or service | Multiple choice | Yes | Material · Service | — |
| PR type | Multiple choice | Yes | Standard PR/PO · Manual PR/PO · FI Payment | PR type |
| Short text | Text | Yes | Sent to the requisitioner | Short text |
| Requestor user ID | Text | No | — | — |
| Behalf of requestor | Text | No | — | — |
| Invoice approver name | Text | Yes | — | — |
| Vendor | Text | Yes | Groups the Purchase requests page | Vendor |
| Vendor code | Text | No | On the PR form | Vendor code |
| Vendor contact name | Text | No | — | — |
| Vendor email address | Text | No | Must be a well-formed e-mail (FR-007) | Vendor email |
| Vendor address | Long text | No | — | — |
| Contract holder | Text | No | — | — |
| Outline agreement | Text | No | SAP outline agreement number | — |
| CP lead | Text | No | — | — |
| Email group | Single choice | No | AT_RETAIL · BE_LUBES · DE_LUBES · FR_LUBES · G_GB_RETAIL · CN_MKT_MO_LUBES (Shell's functional mailbox names, FA E3.7.1.1) | Email group |
| PR recipients | Multiple choice | No | Fictitious: lea.santos@shell.com · pr.desk.manila@shell.com · pr.desk.krakow@shell.com | PR recipients |
| OA vendor (China specific) | Multiple choice | No | Fictitious sample vendors | — |
| Sub category (China specific) | Multiple choice | No | Anti-counterfeit · B2B Sensor · Branded item · Cinema · Creative · Digital marketing | — |

Shell attributes seeded at level **PR line** (section "PR line"): Material group (text) · Remarks /
comments (long text) · Material / service code (text); none needed to submit.

### Status Model

Statuses: **Created · Submitted · In review · Declined · Approved · Closed**. Rounds come from the approval
engine (031): type Purchase request for submissions, type Modified purchase request for changes.

| PR situation | Engine outcome or action | Resulting status |
|---|---|---|
| — | First save | Created |
| Created or Declined | Submit / Re-submit, gates met; a round starts | Submitted |
| Submitted or In review (first round) | Save and re-submit or Re-submit; the open round ends superseded, a new round starts | Submitted |
| Submitted | A level approves and further levels remain | In review |
| Submitted or In review (first round) | Every applicable level approved | Approved, routed (Original) |
| Submitted or In review (first round) | Declined (comment) | Declined (money stays reserved) |
| Submitted or In review (first round) | Recall | Created |
| Approved | Submit changes; a modification round starts | In review (change pending) |
| In review (change pending) | Re-submit of the change; open round superseded, new round | In review (change pending) |
| In review (change pending) | Every applicable level approved | Approved, routed (Modification) |
| In review (change pending) | Declined (comment) or Recall | Approved, last approved values restored |
| Approved (setting Allowed without approval) | Save that only adds lines | Approved, routed (Modification) |
| Approved, no change pending | Close purchase request | Closed, routed (Close) |
| Created or Declined | Delete | Recycle bin; restore returns it Created |

### Requisitioner Attachment (PR form columns, in order)

PR ID · PR name · PR line ID · HX key · Short text · Vendor (name and code) · Cost centre (of the line's
activity) · SWBS code (the line item's) · GL code (the line item's) · Amount · Currency · Delivery date ·
MoA approver (name and e-mail) · Routing type · Changed (Yes for lines changed by an approved modification).
Before the rows, notes: the PR, the routing type and time, the recipients, "Routed by MAPP — no supporting
documents are attached".

### Key Entities

- **Purchase request**: the request form raised in MAPP; header fields, Shell attribute values, owner, MoA
  approver, the two workflows, status, whether a change is In review, the last approved values, the plan
  and company code of its lines, GSAP PR number, date and source; belongs to one plan.
- **PR line**: one claim on one line item: currency, amount, delivery date, overall limit, description,
  attribute values, Sent date and GSAP PO fields; its amount is reserved on the line item.
- **HX key**: "HX" + line item number, derived, never stored or typed.
- **Approved version**: the PR's values at its last approval, kept while a change is In review.
- **Routing**: one hand-off of the PR to the requisitioner inside MAPP: time, who, type, recipients and the
  PR form rows as routed; never changed.
- **Purchase Request Configuration**: Purchase Support on or off; Lines on approved purchase requests.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In usability testing, at least 90% of marketers raise and save a complete single-line PR from
  a line item in under 3 minutes without help.
- **SC-002**: 100% of PR lines show an HX key equal to "HX" + their line item's number, and nobody can enter
  or change one.
- **SC-003**: Across the table-driven acceptance cases — boundaries, other currencies, concurrent saves — the
  reserved amount on a line item never exceeds its planned cost at the moment of the save (0 breaches).
- **SC-004**: Every refused save or submission lists all missing or invalid fields and failed gates in one
  answer (100% of tested cases).
- **SC-005**: 100% of approved PRs and approved modifications are in the Waiting for GSAP view, with a
  downloadable PR form, as soon as the final approval is saved.
- **SC-006**: In user acceptance testing, requisitioners can type the GSAP PR from the PR form alone, with no
  follow-up question to the marketer, for at least 95% of approved PRs.
- **SC-007**: The Purchase requests page shows its first page within 2 seconds at the 95th percentile with
  10,000 PR lines, and applies a search or filter within 2 seconds.
- **SC-008**: Every PR status change and every routing can be traced to the person and time, in the
  activity log and the routing history (100% of tested events).
- **SC-009**: In moderated tests with at least 8 marketers and 3 requisitioners new to MAPP, at least 90%
  complete on the first attempt: create a PR; add and remove a line; find a PR with search and filters;
  edit and save it; delete a Created PR — and everyone who meets an unavailable action can say why.

## Assumptions

- The documented assumptions of the Clarifications section, for the product owner to confirm.
- **Dependencies**: feature 030 stores the SWBS code on the line item and offers its facts (brief §8.1);
  feature 031 approves subjects of type Purchase request and Modified purchase request, with owner and
  record-field approvers and Re-submit from Submitted (brief §8.2); feature 034's D-322 lets line items be
  added to an approved activity without restarting its approval (useful for the demo, not required).
- **Feature 033** answers routed PRs with the GSAP PR number and the purchase order; until it is built, the
  GSAP fields stay empty and the Waiting for GSAP view keeps every routed PR.
- **No e-mail**: routing, the requisitioner queue and the downloadable PR form replace 013's e-mails, retries
  and Accounts Payable mailbox (D-320).
- **Out of scope**: supporting documents, invoice approval and the PR transactions list (a PR line's
  invoices are seen on its line item's Financials section, 015), bulk creation from a selection, monthly
  and annual PR rates, GRSP and Actuals amounts, per-person column choices on the page.
- **Demo data** is fictitious (constitution); Shell's Email group names are configuration (D-325), not people.
