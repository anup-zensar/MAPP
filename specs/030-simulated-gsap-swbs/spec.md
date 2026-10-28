# Feature Specification: SWBS Codes from a Simulated GSAP

**Feature Branch**: `030-simulated-gsap-swbs`

**Created**: 2026-10-27

**Status**: Draft

**Input**: Product owner instruction of 2026-10-27: mimic GSAP's creation of SWBS codes with a simulated
GSAP, so that the whole purchase-request to purchase-order process can be shown now (constitution 4.4.0,
D-321, ADR-0016). A line item of an approved activity asks GSAP for an SWBS code; a simulated GSAP answers
with one code per line item; the line item keeps the code, shows when it was requested and received, and
shows its HX key, which purchase requests and GSAP data need later.

**References**: Client spec 005 (line items) FR-014, FR-015 · client spec 015 (GSAP integration) FR-005 –
FR-016, "Line Item GSAP Status" · Hive9 functional analysis §B7 (the line item is the only door to GSAP),
§D3 stage 1.2, §D4 (FSSI 1523), §D5 (keys) · ADR-0009, ADR-0016 · decisions D-255 (superseded), D-304,
D-321 · domain model §2.4 · field catalogue §5 #13, #16 · features 007 (line items), 008 (approvals), 010
(All work), 014 (access), 015 (financial transactions).

## Clarifications

### Session 2026-10-27

No product owner was available during the session. Each point below was resolved from the sources in
this order: client spec, Hive9 functional analysis, decision log, ADRs. They are documented assumptions
for the product owner to confirm.

- Q: Who runs the simulated GSAP? → A: People whose role includes "Import GSAP extracts", the permission
  that already loads simulated GSAP data (ADR-0016). Everyone else sees statuses and codes but cannot run it.
- Q: Which line items does a run send? → A: Every line item that is Ready to send and that the person's
  "Import GSAP extracts" access reaches (that permission applies to All markets), or only the line items
  the person names. A named line item that the person cannot see answers "not found"; a named line item
  that is not Ready to send stops the run, which then sends nothing.
- Q: How quickly does the simulated GSAP answer? → A: At once. Real GSAP needs 24–48 hours (FA §D3); the
  simulation returns a code for every request in the same run, so the request and receipt times are the
  run's time and there is no "sent, waiting for GSAP" status. Rejections stay with live integration
  (ADR-0016).
- Q: What does a simulated code look like? → A: `SIM-<company code>-<sequence>`, the sequence four digits,
  counted per company code from 0001 across all runs and never reused (FA §D7: codes are sequential per
  company code). `0000` stands in for the company code when the activity has none. The company code is
  the activity's "Company code" Shell attribute.
- Q: What happens to a code when the activity leaves approval? → A: The code stays and the request
  cannot be switched off any more (client 005 FR-015, 015 FR-039, D-321). Moving the line item to
  another activity, or deleting and restoring it, keeps the code too. Duplicating a line item or copying
  an activity or plan never copies the code (client 005, "Activity copied").
- Q: Is a run with nothing to send kept? → A: Yes, as a run with no codes, logged as "Nothing was ready
  to send" (client 015 FR-007). The screen offers the action only when something is ready.
- Q: Where is the status shown? → A: Wherever line items show it today — the activity's Line items
  section, the line item page and All work — plus the code, its dates and its source on the line item
  page. The approval preview does not show the SWBS status today and is unchanged.
- Q: What does the demo data show? → A: One run of the demo data gave "Venue hire" (Social always-on,
  company code DE03) its code `SIM-DE03-0001`; "Stand build" (Billboard winter wave, GB01) stays Ready to
  send, so the demo's first run gives it `SIM-GB01-0001`. The demo GSAP transactions of those two line
  items carry those codes, so the line item and its GSAP data agree.

## User Scenarios & Testing *(mandatory)*

All examples use the demo data (docs/product/demo-users-and-data.md). Alex Morgan holds the Administrator
role, the only demo role with "Import GSAP extracts"; Priya Raman is a Planner for Lubricants Europe;
Daniel Okoro approves Lubricants activities. "Trade magazine ads" (A-1004, company code NL02) is Submitted
with one line item, "Media booking" (L-1005), whose request is on.

### User Story 1 - Send the waiting SWBS requests to the simulated GSAP (Priority: P1)

A finance person opens **Financials**. A **Simulated GSAP** panel says how many line items are waiting to
be sent for an SWBS code. They choose **Send SWBS requests to simulated GSAP**, confirm, and every waiting
line item receives its code at once. The panel shows what came back and keeps the history of every run.

**Why this priority**: Without an SWBS code no purchase request can be raised and no purchase order
created (FA §D8). This is the first step of the purchase-request to purchase-order process the product
owner wants to demonstrate.

**Independent Test**: Sign in as Alex Morgan on the demo data, open Financials, send the waiting requests
and check that "Stand build" (L-1006) now shows `SIM-GB01-0001`.

**Acceptance Scenarios**:

1. **Given** the demo data, **When** Alex opens Financials, **Then** the Simulated GSAP panel says "1 line
   item is ready to send" and offers "Send SWBS requests to simulated GSAP"; its history lists the demo
   data's run, which gave Venue hire (L-1003) `SIM-DE03-0001`.
2. **Given** that panel, **When** Alex chooses the action, **Then** a confirmation reads "Send 1 SWBS
   request to simulated GSAP? The simulated GSAP returns one SWBS code per line item at once. Codes can't
   be changed or removed in MAPP." **When** he confirms, **Then** the panel says "Simulated GSAP returned
   1 SWBS code", lists "Stand build (L-1006) · SIM-GB01-0001", the ready count falls to 0, and the new run
   is first in the history with his name and the time.
3. **Given** Daniel Okoro has approved "Trade magazine ads", **When** Alex sends the waiting requests,
   **Then** "Media booking" (L-1005) receives `SIM-NL02-0001` together with Stand build's code; a later
   NL02 line item receives `SIM-NL02-0002`.
4. **Given** an approved activity without a company code whose line item is Ready to send, **When** a run
   sends it, **Then** its code is `SIM-0000-0001` (the next 0000 number).
5. **Given** nothing is ready to send, **When** Alex looks at the panel, **Then** it says "No line item is
   waiting for an SWBS code" and the send action is unavailable with that reason next to it.
6. **Given** a run, **When** anyone allowed to see it opens the activity log, **Then** exactly one entry
   records the run: who ran it, when, how many codes the simulated GSAP returned and which.
7. **Given** Priya (no "Import GSAP extracts"), **When** she opens Financials, **Then** she sees no
   Simulated GSAP panel, and a request to run it is refused as not allowed.

---

### User Story 2 - See where a line item's SWBS request stands (Priority: P1)

A marketer opens a line item and sees its SWBS request: the status in words and colour with a short
explanation, the code once it has arrived, when it was requested and received, and that the simulated
GSAP produced it. The same status shows on every row of the activity's Line items section and in All
work.

**Why this priority**: Marketers often do not know which SWBS code to use or why a purchase request cannot
be raised yet (FA §D7). The line item answers that without a call to finance.

**Independent Test**: Open "Venue hire" (L-1003), "Stand build" (L-1006), "Speaker fees" (L-1004) and
"Agency fee" (L-1001) and compare the status, explanation and code of each.

**Acceptance Scenarios**:

1. **Given** "Agency fee" (L-1001) of the Created activity "Launch film", **When** the marketer opens it,
   **Then** the status reads "Waiting for activity approval" with "The request is sent once the activity
   is approved." and the code reads "Not received yet".
2. **Given** "Speaker fees" (L-1004), whose request is off, **When** it is opened, **Then** the status reads
   "Not requested" with "'Request SWBS code from GSAP' is off, so no code will be requested."
3. **Given** "Stand build" (L-1006) of an approved activity, **When** it is opened, **Then** the status
   reads "Ready to send to GSAP" with "Sent with the next run of the simulated GSAP."
4. **Given** "Venue hire" (L-1003), **When** it is opened, **Then** the status reads "SWBS code received",
   the code `SIM-DE03-0001`, "Requested" and "Received" with the demo data's date and time, and the source
   "Simulated GSAP".
5. **Given** the Line items section of "Social always-on" and All work with its SWBS request column,
   **When** they are shown, **Then** Venue hire shows "SWBS code received" and Speaker fees "Not
   requested", in the same words and colours as the line item page; sorting All work by the column puts
   received after ready to send.

---

### User Story 3 - A received code stays with the line item (Priority: P1)

Once GSAP has given a line item its code, the code belongs to that line item for good. The activity can be
changed and approved again, the line item can be edited, moved, deleted and restored — the code stays, and
the request can no longer be switched off.

**Why this priority**: A purchase request and every GSAP document carry the code (FA §D5). Losing or
reissuing it would break the chain to GSAP.

**Independent Test**: Take Venue hire, send its activity back for approval, edit the line item and try to
switch the request off; check the code after each step.

**Acceptance Scenarios**:

1. **Given** Venue hire with its code, **When** its activity leaves approval (a change sends it back to
   Created, or it is declined), **Then** Venue hire still shows `SIM-DE03-0001` and "SWBS code received",
   and no run sends it again.
2. **Given** Venue hire, **When** someone edits it, **Then** "Request SWBS code from GSAP" is on and cannot
   be switched off, with "This line item already has its SWBS code from the simulated GSAP, so the request
   stays on." next to it; a save that switches it off anyway is refused with that message on the field.
3. **Given** Venue hire, **When** it is duplicated, or its activity or plan is copied, **Then** the copy
   has no code, its request is on, and its status follows its own activity.
4. **Given** Venue hire, **When** it is deleted and restored from the recycle bin, or moved to another
   activity, **Then** it keeps its code and its dates.
5. **Given** a line item Ready to send, **When** its request is switched off, or its activity is recalled
   or declined before a run, **Then** the next run does not send it.

---

### User Story 4 - Send one line item now from its page (Priority: P2)

A finance person working on one line item that is ready to send does not want to send everything else as
well. On the line item page they choose **Send to simulated GSAP now**, confirm, and that line item alone
receives its code.

**Why this priority**: It shortens the demonstration of one purchase request, but the panel of User Story 1
already covers the need.

**Independent Test**: As Alex, open Stand build, send it alone and check that Media booking, also ready,
was not sent.

**Acceptance Scenarios**:

1. **Given** Stand build is Ready to send and Alex may import GSAP extracts, **When** he opens it, **Then**
   the SWBS request card offers "Send to simulated GSAP now".
2. **Given** that action, **When** Alex confirms "Send Stand build to simulated GSAP now? It receives its
   SWBS code at once. Codes can't be changed or removed in MAPP.", **Then** the card shows
   `SIM-GB01-0001`, the status "SWBS code received" and the run time, and no other line item was sent.
3. **Given** Priya, or a line item that is not Ready to send, **When** the page is shown, **Then** the
   action is not offered.
4. **Given** someone else sent the line item meanwhile, **When** Alex confirms, **Then** nothing is sent,
   he reads that the line item is no longer ready to send, and the page shows its current status.

---

### User Story 5 - Find and copy the HX key (Priority: P2)

A requisitioner types the HX key into GSAP's purchase request so that GSAP data finds its way back to the
right line item (FA §D3 stage 2.1). The line item page shows the HX key, read-only, with a copy action.

**Why this priority**: Purchase requests (feature 032) and their PR form need it, and GSAP matching
already uses it (feature 015), but nothing breaks without it today.

**Independent Test**: Open Stand build and copy its HX key.

**Acceptance Scenarios**:

1. **Given** Stand build (L-1006), **When** it is opened, **Then** its page shows "HX key: HX1006",
   labelled "Generated from the line-item number", which cannot be edited.
2. **Given** that HX key, **When** the person chooses Copy, **Then** HX1006 is on the clipboard and the page
   confirms "HX key copied".

---

### Edge Cases

- A line item added to an approved activity is Ready to send at once.
- The request is switched off and on again before a run: Not requested, then Ready to send again.
- Two people run the simulated GSAP at the same moment: the runs happen one after the other, and the
  second does not send what the first already sent.
- A line item in the recycle bin, alone or with its activity, is never sent; restored, it is sent by the
  next run if it is still Ready to send.
- The activity's company code, the line item's planned cost or its finance coding change after the code
  arrived: the code stays (GSAP owns it) and nothing is sent again.
- A line item moved to an activity that is not approved keeps its code and shows "SWBS code received".
- The send action is pressed twice: one run is made; the second press finds nothing ready or is blocked
  while the first is under way.
- A run fails half-way: nothing of it is saved — no line item gets a code and no run is kept.

## Requirements *(mandatory)*

### Functional Requirements

#### The line item's SWBS request

- **FR-001**: System MUST show each line item's SWBS request status, worked out in this order: **SWBS code
  received** when the line item has a code; **Not requested** when "Request SWBS code from GSAP" is off;
  **Ready to send to GSAP** when its activity is Approved (also while it shows In progress or Complete);
  otherwise **Waiting for activity approval** (domain model §2.4).
- **FR-002**: System MUST keep, for each line item, the SWBS code GSAP returned and the times the request
  was sent and the code received. Nobody can type, change or remove them; only a run of the simulated
  GSAP (FR-006) sets them.
- **FR-003**: System MUST keep a received code and its times when the activity leaves approval, when the
  line item is edited or moved, and when it is deleted and restored. A line item with a code is never sent
  again.
- **FR-004**: System MUST keep "Request SWBS code from GSAP" on once a code is received: a save that
  switches it off is refused with "This line item already has its SWBS code from the simulated GSAP, so the
  request stays on.", and the form shows the switch locked with that sentence next to it.
- **FR-005**: System MUST NOT copy the SWBS code or its times when a line item is duplicated or copied with
  its activity or plan; the copy keeps the request flag and its status follows its own activity.

#### The simulated GSAP run

- **FR-006**: Users with "Import GSAP extracts" MUST be able to run the simulated GSAP: it sends every line
  item that is Ready to send and within the person's reach for that permission — or only the line items
  the person names — and returns one SWBS code for each at once. Each line item sent stores its code with
  the run's time as both request and receipt time.
- **FR-007**: The simulated GSAP MUST give codes of the form `SIM-<company code>-<nnnn>`: the company code
  is the activity's "Company code" Shell attribute, `0000` when the activity has none; `nnnn` counts per
  company code from 0001 across all runs and is never reused. Line items of one run are numbered in the
  order of their IDs.
- **FR-008**: When the person names line items, System MUST answer "not found" for a named line item that
  does not exist or that the person cannot see, and MUST refuse the run, sending nothing, when a named line
  item is not Ready to send, naming it.
- **FR-009**: System MUST keep every run as a record with its ID, time, who ran it (MAPP for the demo
  data), its source ("Simulated GSAP", or "Demo data (simulated GSAP)" for the demo data), the number of
  line items sent and, for each, its ID, name, company code and code. A run that found nothing to send is
  kept with no codes.
- **FR-010**: System MUST write exactly one activity-log entry per run, naming the simulated GSAP and
  listing the codes it returned, or "Nothing was ready to send". The entry is visible to people with
  "Import GSAP extracts" and to people who see the whole activity log (as GSAP imports, D-304).
- **FR-011**: System MUST let users with "Import GSAP extracts" see the run history, newest first, with how
  many line items are Ready to send now.
- **FR-012**: System MUST refuse a run, and the run history, to people without "Import GSAP extracts"; it
  does not offer them the actions.

#### Screens

- **FR-013**: The Financials page MUST show people with "Import GSAP extracts" a **Simulated GSAP** panel
  with the number of line items ready to send, the action **Send SWBS requests to simulated GSAP**, and the
  run history (time, who, source, codes). The action is not the page's yellow primary action. With nothing
  ready, it is unavailable and says "No line item is waiting for an SWBS code". It asks for confirmation
  first, naming how many requests it sends and that codes cannot be changed or removed in MAPP, and shows
  the result in the panel.
- **FR-014**: The line item page MUST show an **SWBS request** card: the status chip with its explanation,
  the SWBS code ("Not received yet" without one), when it was requested and received, the source
  "Simulated GSAP", and the HX key (FR-016).
- **FR-015**: When the line item is Ready to send and the person has "Import GSAP extracts", the card MUST
  offer **Send to simulated GSAP now**, which, after confirmation, runs the simulated GSAP for that line
  item alone (FR-006, FR-008) and then shows the new code. When the run is refused because the line item is
  no longer ready, the card says so and shows the current status.
- **FR-016**: The line item page MUST show the line item's **HX key** — "HX" followed by the digits of the
  line item's ID (L-1006 → HX1006) — read-only, labelled "Generated from the line-item number", with a
  **Copy** action that puts it on the clipboard and confirms "HX key copied".
- **FR-017**: The activity's Line items section, the line item page and All work's SWBS request column MUST
  show the status in the same words and colours, the received status included; All work sorts it after
  Ready to send.
- **FR-018**: The line item form MUST show the read-only SWBS code once received, and otherwise "Filled by
  the simulated GSAP once the activity is approved and the request is sent"; the help of the request
  switch explains that the simulated GSAP stands in for GSAP.
- **FR-019**: The line item page's Financials section MUST no longer show SWBS codes taken from GSAP
  transactions: the SWBS request card shows the line item's own code (D-321 supersedes D-255).
- **FR-020**: Everything the simulated GSAP produced MUST be labelled "Simulated GSAP" on screen and in the
  activity log.

#### Demo data

- **FR-021**: The demo data MUST contain one run of the simulated GSAP, made by MAPP at the demo data's
  time through the same rules as any run, that gave "Venue hire" (L-1003) the code `SIM-DE03-0001`;
  "Stand build" (L-1006) stays Ready to send. The demo GSAP transactions of these two line items carry the
  codes `SIM-DE03-0001` and `SIM-GB01-0001`. Every other demo record stays as it is.

### Key Entities *(include if feature involves data)*

- **Line item's SWBS request**: part of a line item — the request flag (feature 007), the derived status,
  the SWBS code, when it was requested, when it was received, and the HX key worked out from the line
  item's ID.
- **SWBS run**: one run of the simulated GSAP — ID, time, who ran it, source, and the line items it sent
  with their company codes and codes. Runs are only ever added.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A finance person sends every waiting SWBS request and reads the returned codes in under one
  minute and three actions from the Financials page (open the panel's action, confirm, read the result).
- **SC-002**: In 100% of runs, every line item that was Ready to send in the person's reach receives
  exactly one code, and no code is ever given twice.
- **SC-003**: In 100% of tests where an activity leaves approval, or a line item is edited, moved, deleted
  and restored after its code arrived, the code and its times are unchanged.
- **SC-004**: No screen or request lets a person type, change or remove an SWBS code; tests cover every
  screen that shows one.
- **SC-005**: A first-time user can tell from the line item page, without help, whether its SWBS code has
  arrived and, if not, what it is waiting for; at least 90% of participants in a moderated test do so on
  the first attempt.
- **SC-006**: The HX key is copied in one action from the line item page.

## Assumptions

- The simulated GSAP stands in for GSAP until live integration (ADR-0016); live GSAP replaces it by
  implementing the same adapter, with requests sent in the daily FSSI 1523 file and codes arriving 24–48
  hours later. Statuses for "sent, waiting for GSAP", rejections, re-requests, exceptions and alerts
  (client 015 FR-008 – FR-016, FR-035 – FR-049) come with live integration and are not built now.
- The simulated GSAP accepts every request: it does not check the mandatory content of the SWBS request
  file (client 015 "Record Content"), because GSAP's checks are unknown until its layouts arrive (FA §D9).
- The company code is read from the activity's Shell attribute labelled "Company code"; the option's label
  is the code (GB01, NL02, …).
- "Import GSAP extracts" applies to All markets (feature 014, Table A), so a run reaches every plan.
- Purchase requests (feature 032) read the code and the HX key from the line item; they are not part of
  this feature.
