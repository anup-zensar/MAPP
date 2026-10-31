# Feature Specification: Performance Data from a Simulated Feed

**Feature Branch**: `046-performance-data`

**Created**: 2026-10-30

**Status**: Draft

**Input**: Measure module issue #39 (lane B, wave B): "The record types behind every performance figure, built
like the simulated GSAP (ADR-0016, D-370): data sources with column mappings (Settings › Performance data),
performance facts (inquiries, MQLs, deals, revenue, web metrics) loaded by file upload or a simulated pull,
matched to activities on activity ID, media code, integration ID or external name, rows to review and fix,
media codes generated per activity, the external name set when an activity is approved, and *Is integrated*
on the activity."

**Why this feature exists**: every actual figure in the Measure module resolves to a performance fact matched
to an activity. [Measure metrics](../../docs/product/measure-metrics.md) §2 defines *Actual INQ / MQL / Deals
/ Revenue* as "performance facts matched to the activities in view, by the fact's date; **a fact matched to no
activity counts nowhere** and is listed to review". Matching is therefore not a convenience — an unmatched row
is revenue that silently disappears from every report. This feature owns the records, the matching and the
review queue that make the wave B reports (047, 048, 049, 050) answerable.

**Scope boundary**: client 020 covers eight stories. Only **US1–US6** belong to 046. Client 020 US7
(projected stage volumes and revenue) is feature **047**; client 020 US8 (attribution models crediting
revenue) is feature **050**. This spec defines the facts those features read; it never projects or attributes.

**References**: Client 020 US1–US6 and FR-001 – FR-040
([Shell's Measure requirements](../../docs/reference/shell-measure-requirements.md) §"Client 020") ·
Hive9 functional analysis §E4.1, §E7.2–E7.4, §G7 ·
[Measure metrics](../../docs/product/measure-metrics.md) §2 (*Actual INQ / MQL / Deals / Revenue*,
*Is integrated*) · [Measure module build guide](../../docs/sdd/measure-module-build.md) §2 (the 046 row),
§4 (ownership), §7 (words) · Feature 015 (import, matching and review pattern) · Feature 030 (external keys,
simulated GSAP) · ADR-0016 (simulated GSAP) and the ADR-0020 this feature writes (simulated performance feed) ·
Decisions D-367 (permissions `performance.import`, `performance.link`) and D-370 (performance data only from a
simulated feed behind an adapter) · Constitution 4.8.0 non-negotiable 13 (live performance connectors are out
of scope).

## Clarifications

### Session 2026-10-30

No product owner was available in the loop. Each point is resolved from the sources (client specs > functional
analysis > decision log > domain model) and recorded for the product owner to confirm. The integrator moves
these into the decision log at the end of wave B.

- **046-C1** — Q: What does "an activity that counts as approved" mean, given facts are gated on it?
  → A: Statuses **Approved, In progress and Complete** only; never Created, Submitted or Declined, and the
  rule applies again after a re-approval or a restore from the recycle bin (client 020 FR-025). This single
  definition governs the gate on the Performance section (FR-024), the holding of facts (FR-012) and the
  generation of the external name (FR-031).

- **046-C2** — Q: Four matching keys are listed; is the order meaningful, and what happens when two keys
  disagree? → A: The order is **strict and total**: activity ID, then media code, then integration ID, then
  external name (client 020 FR-011). The first key present in the row decides the match. When two keys are
  present and resolve to **different** activities, the row does not guess: it is left unmatched with the
  reason "keys disagree" and both candidates named. Ambiguous (one key, several activities) and unknown (no
  key resolves) rows are likewise unmatched with their own stated reason.

- **046-C3** — Q: A row can end up in several states; what are they and must they reconcile? → A: Five
  outcomes, and they **must add up to rows read** (client 020 FR-014): *matched* (bound to one approved
  activity, counted everywhere), *held* (matched to an activity not yet counting as approved — excluded from
  every figure, released automatically when it is approved, FR-012), *unattributed* (valid row deliberately
  carrying no activity key), *rejected* (failed validation), *unmatched* (no confident match). Rejected and
  unmatched rows go to the review queue; held facts do not, because nothing is wrong with them.

- **046-C4** — Q: What makes an upload idempotent? → A: The pair **(data source, source reference)** is the
  fact's identity (client 020 FR-015). Re-uploading the same file changes nothing; a later row with the same
  pair replaces the earlier fact and the replacement is audited with both versions, so an undo (FR-017) can
  restore what was replaced.

- **046-C5** — Q: Live connectors are out of scope (constitution non-negotiable 13, D-370), so what is the
  "simulated pull"? → A: An adapter in the shape of the simulated GSAP (ADR-0016): it produces rows on demand
  for a source whose delivery route says so, and those rows then travel **the same validation, matching and
  review path as an uploaded file** (client 020 FR-008: "any future automated feed passes through the same
  rules"). No network call, no scheduler, no external credential. ADR-0020 records this.

- **046-C6** — Q: How is personal data handled? → A: A file carrying personal-data columns is refused **in
  full** — not row by row — and nothing from it is stored (client 020 FR-010). Journey rows carry only a
  pseudonymous reference. This is a whole-file gate before any row is read.

- **046-C7** — Q: What happens to revenue in a currency with no rate? → A: The fact is stored **exactly** in
  its own currency and never converted at a guessed rate (constitution non-negotiable 9). It is marked, left
  out of converted totals, and the count of excluded facts is shown beside the total so a reader never sees a
  silently short figure; once a rate exists it is included (client 020 FR-018).

- **046-C8** — Q: Which year does a fact count in when its activity spans two? → A: The fiscal year of the
  **fact's own date**, never the activity's (client 020 FR-019). A multi-year activity therefore contributes
  to each year it produced facts in.

- **046-C9** — Q: May a data source be deleted once it holds facts? → A: No. Delete is offered only while the
  source holds no facts; otherwise **Deactivate** is offered instead, which keeps every fact, refuses new
  uploads and updates *Is integrated* at once, and can be reversed (client 020 FR-002, FR-004). Deleting a
  source that holds facts would silently restate history.

- **046-C10** — Q: Does copying an activity carry its performance identity? → A: No. A copy takes **no**
  external name, aliases, integration IDs, media codes or facts (client 020 FR-036); its external name reads
  "Generated on approval" like any new activity. Otherwise a copy would steal its source's incoming facts.

- **046-C11** — Q: Which web metrics does a source supply? Client 020 FR-026 requires "web metrics by month"
  but never names them. → A: **Sessions, visits and engagement touches** — the three the functional analysis
  attributes to the web-analytics connector category (§E7.2), which it says drive "early-stage waterfall
  volumes; attribution touchpoints". They are counts by month, never money, and they are reported on the
  activity rather than against a funnel stage, because no stage is defined for them.

- **046-C12** — Q: Where does the media-code configuration live — which activity types require codes, and
  which attributes those codes carry? → A: In **Settings › Performance data**, this feature's own area, as a
  list of the activity types that require codes plus the attribute catalogue. It does **not** become a field
  on the activity type record: activity types belong to feature 002's `settings`, and the build guide (§4.1)
  grants 046 exclusive ownership of only one file outside its own folders — the Performance slot in
  `ActivityPage.tsx`. Keeping the configuration in 046's area respects that boundary and matches the
  functional analysis, which describes media codes as their own management area (§E7.3, "Media Code
  Management"). If Shell later wants the flag on the activity type itself, that is a change for 002's lane.

- **046-C13** — Q: Who starts a simulated pull, and when? Client 020 FR-008 is silent. → A: An **explicit
  action on the data source**, by someone holding `performance.import` — never a schedule and never
  automatically. This follows the simulated GSAP precedent exactly (feature 033: the requisitioner opens the
  record and chooses "Create GSAP PR and PO (simulated)", a single operation behind a permission), which
  ADR-0016 set and ADR-0020 restates for this feed. A scheduler would be a live connector in all but name,
  which constitution non-negotiable 13 puts out of scope.


- **046-C14** — Q: Who owns the activity's external name? → A: **Feature 006 already does.**
  `activities.external_name` exists today as a free-text field a person edits, with its own permission
  ("Edit External Name"), a 120-character limit and a default of the activity name (field catalogue row 2).
  Client 020 FR-030 – FR-036 would make it generated on approval from the model's rule, behind
  `performance.link` and with the rule's own length — a change to feature 006's field, its permission and its
  package, none of which this feature owns (build guide §4.1 grants 046 exactly one file outside its folders,
  the Performance slot in `ActivityPage.tsx`).

  So 046 **reads** `activities.external_name` as a matching key and never writes it, and owns only the
  **aliases**: a superseded name is kept so rows already in flight still match. Generation, regeneration,
  "Generated on approval" and hand-set suppression are requested of feature 006's lane in this feature's
  pull request. Until that lands, the field keeps today's behaviour, and matching works on the current name
  plus any aliases.


- **046-C15** — Q: Must the imports list and the review queue filter by the actor's scope, as plan.md's
  Queries table says? → A: **No, and the table was wrong.** Neither is plan-owned data. An import records a
  file, its uploader and its counts; a **rejected or unmatched row belongs to no activity at all** — that is
  what makes it unmatched — so there is no plan to derive a scope from, and a scope filter would hide every
  unmatched row from everyone. Both are administrator work, gated by `performance.import` (FR-043).

  **But one field of a review row is plan-owned**: `candidates`, the activities named when keys disagree
  (FR-019). Naming an activity the reader cannot see would disclose its existence, which constitution
  non-negotiable 7 forbids. So the queue is permission-gated, and the candidate list each reader sees is
  filtered to the activities within their scope; a row whose candidates are all outside it still appears,
  with its reason, and names none of them. That satisfies FR-019 without turning the queue into a directory
  of records the reader may not know about.


- **046-C16** — Q: Who may undo a load? → A: **An administrator, not everyone who may load.** Client 020
  FR-017 says "Administrators can undo a whole upload"; this feature's own FR-017 copied that word but the
  permissions table then assigned undo to `performance.import`, the same permission as uploading. That is an
  internal contradiction, and the looser half would let anyone who can load a file silently unwind someone
  else's. Undo therefore needs `performance.import` **and** `settings.manage`.

- **046-C17** — Q: What states can a load end in? → A: **Completed, failed or undone.** The schema and the
  domain type carried only completed and undone, but client 020 FR-062 requires audit records for "uploads
  (completed, failed, undone)" and FR-063 lists "an upload failed" as a notification event. A load that falls
  over partway must be recorded as failed rather than vanishing or masquerading as completed.

- **046-C18** — Q: The counts must be reported "by stage or metric" (client 020 FR-014) — does the summary
  need that breakdown? → A: **Yes, and it was dropped.** FR-022 as first written kept only the five totals.
  A loader who sees "120 unmatched" and cannot see that all 120 are MQL rows has no way to find the cause.
  The five totals still reconcile against rows read; the breakdown sits under them.

- **046-C19** — Q: Does "unattributed" mean the same as "unmatched", as client 020 FR-057 suggests? → A:
  **No — FR-057 is a different context.** It governs attribution reporting in feature 050 ("Unmatched touches
  report as Unattributed"). For a load, the client's own US2 and FR-014 list them as two of five separate
  outcomes, so the distinction this spec draws in 046-C3 is the client's own. Recorded because the two words
  look interchangeable and a reader of FR-057 alone would merge them.


## User Scenarios & Testing *(mandatory)*

### User Story 1 - Set up a data source and its column mappings (Priority: P1)

An administrator opens Settings › Performance data and describes where a feed comes from: its name, owner,
category, delivery route, cadence and the models it supplies. Then, for that source, they say what each column
of its file means — this column is the matching key, that one is the fact date, that one is the stage, that
one is revenue with its currency. They can find, view, edit and delete sources and mappings, and download a
blank file template built from the mappings.

**Why this priority**: nothing else in the feature can exist first. A fact cannot be read without a mapping,
and cannot be attributed without a source. It is also the only story that stands alone as useful: Shell can
configure and review its feeds before a single row is loaded.

**Independent Test**: create a source "Marketing automation" with mappings for reference, date, stage and
revenue; download its template; edit a mapping; try to delete the source (allowed, it holds no facts). No
upload is needed.

**Acceptance Scenarios**:

1. **Given** no data sources exist, **When** an administrator creates one with a name, owner, category,
   delivery route, cadence and a model, **Then** it is saved, listed, and shows a freshness status of
   *Inactive* because it has received nothing yet.
2. **Given** a source exists, **When** its mappings are set so that one column is the matching key, one the
   fact date and one revenue with a currency, **Then** a blank file template can be downloaded carrying
   exactly those columns.
3. **Given** a model already draws its *MQL* stage from an active source, **When** a second active source is
   mapped to supply *MQL* for the same model, **Then** the mapping is refused and the message names the
   source that already supplies it.
4. **Given** a source that holds no facts, **When** an administrator deletes it, **Then** it is removed; and
   **Given** a source that holds facts, **When** they open the same menu, **Then** Delete is not offered and
   Deactivate is offered in its place with a note saying why.
5. **Given** a source is deactivated, **When** an upload is attempted for it, **Then** the upload is refused
   and every activity that depended only on that source stops counting as integrated at once.
6. **Given** a mapping is edited, **When** an earlier upload is viewed, **Then** that upload's results are
   unchanged — mappings affect future uploads only.

---

### User Story 2 - Upload performance data and match it to activities (Priority: P1)

A marketer uploads a CSV or spreadsheet for a chosen source. Every row is checked, every problem is reported at
once, and each valid row is matched to exactly one activity. The upload ends with a summary that reconciles:
rows read, and how many were matched, held, unattributed, rejected and unmatched. An administrator can undo the
whole upload.

**Why this priority**: this is the feature's reason to exist — it is where an actual figure is created. Every
wave B report reads what this story produces.

**Independent Test**: on the demo data, upload a file whose rows exercise all five outcomes against a
configured source; check the five counts add up to rows read; re-upload the identical file and confirm nothing
changes; undo the upload and confirm the figures return to what they were.

**Acceptance Scenarios**:

1. **Given** a source with mappings, **When** a file is uploaded whose rows carry a known activity ID, a date
   and a stage, **Then** each row becomes a fact matched to that activity and the summary counts them as
   matched.
2. **Given** a file with several faults in one row — an empty required column, a future date and a negative
   count — **When** it is uploaded, **Then** all three problems are reported together for that row, not one
   at a time.
3. **Given** a row whose activity ID and media code resolve to **different** activities, **When** it is
   uploaded, **Then** it is left unmatched with the reason "keys disagree", both candidate activities are
   named, and it appears in the review queue.
4. **Given** a row matching an activity that is still *Submitted*, **When** it is uploaded, **Then** the fact
   is **held**, counts in no figure anywhere, and is released automatically the moment that activity is
   approved.
5. **Given** an upload has completed, **When** the summary is read, **Then** matched + held + unattributed +
   rejected + unmatched equals rows read.
6. **Given** a file has already been uploaded, **When** the identical file is uploaded again, **Then** no
   figure changes; and **When** a later file repeats a source reference with a new value, **Then** it
   replaces the earlier fact and the replacement is recorded with both versions.
7. **Given** a file carrying a personal-data column, **When** it is uploaded, **Then** the whole file is
   refused, the column is named, and no row from it is stored.
8. **Given** an upload is undone by an administrator, **When** the confirmation is shown, **Then** it states
   the counts and what will be recalculated; after confirming, replaced versions are restored and the upload
   remains listed as *Undone*.

---

### User Story 3 - See performance on an approved activity (Priority: P1)

An activity owner opens an activity and finds a Performance section: per stage the goal, actual to date,
projected for the plan year and the conversion; revenue with the name of the model behind it; web metrics by
month; and how fresh each contributing source is. Below it, a Data sources area shows the external name and
aliases, the media-code count, the two flags, and per source the integration ID, fact count and last fact
date. Before the activity is approved, the section says so and shows no figures.

**Why this priority**: it is where the data becomes visible to the people who own the work, and it is the only
P1 story a non-administrator uses daily. It also proves the approval gate, which protects every report.

**Independent Test**: on the demo data, open a *Submitted* activity and confirm the gate message and no
figures; approve it and confirm its held facts appear with goal, actual and conversion per stage.

**Acceptance Scenarios**:

1. **Given** an activity is *Submitted*, **When** its Performance section is opened, **Then** it reads
   "Approve the activity to see performance data" and shows no figures at all.
2. **Given** an activity is *Approved* and has facts, **When** its Performance section is opened, **Then**
   each stage shows goal, actual to date, projected for the plan year and conversion, and revenue names the
   model it came from.
3. **Given** an activity is *Approved* but its model has no source for a stage, **When** that stage is
   viewed, **Then** it states that reason and never shows a sample or placeholder figure.
4. **Given** an approved activity with facts, **When** its facts are listed, **Then** each shows date, source,
   stage or metric, value, currency, the key it was matched on and the upload it came from, and the list can
   be searched and exported.
5. **Given** an approved activity whose only source has not delivered within its cadence, **When** the
   Performance section is opened, **Then** the source is shown as *Stale* with the date its data runs through.
6. **Given** an activity is restored from the recycle bin and re-approved, **When** its Performance section is
   opened, **Then** its held facts are released and counted exactly as they would be for any approved activity.

---

### User Story 4 - External names and integration IDs (Priority: P1)

An activity already carries an external name, so outside systems have a stable handle to send data against.
When someone changes it, MAPP keeps the previous name as an alias, so rows already in flight still find the
activity. Separately, one integration ID per data source can be recorded against an activity. Both the name
and the integration ID are matching keys.

**Why this priority**: two of the four matching keys resolve here. Without the alias, every rename silently
breaks attribution for data already on its way; without integration IDs, a source that knows the activity by
its own identifier cannot match at all.

**Independent Test**: record an integration ID for a source and confirm a row carrying it matches; rename an
activity's external name and confirm a row carrying the **old** name still matches through the alias.

**Acceptance Scenarios**:

1. **Given** an activity with an external name, **When** a row carrying that name is loaded, **Then** it
   matches that activity and records that it matched on the external name.
2. **Given** an activity's external name is changed, **When** a row carrying the **previous** name is
   loaded, **Then** it still matches that activity through the alias.
3. **Given** an alias that would repeat another activity's name or alias, **When** it would be kept,
   **Then** it is refused with the reason, so a key never resolves to two activities.
4. **Given** an activity already has an integration ID for a source, **When** a second is added for the same
   source, **Then** it is refused.
5. **Given** an integration ID is removed, **When** the confirmation is shown, **Then** it states the effect
   on matching, and the change is recorded in the activity log.
6. **Given** an activity with aliases, integration IDs, media codes and facts, **When** it is copied,
   **Then** the copy carries none of them.
7. **Given** feature 006 still owns how the external name is set, **When** this feature is used, **Then** it
   never writes that field — generation on approval is requested of 006's lane (046-C14).

---

### User Story 5 - Review and fix rows that could not be loaded or matched (Priority: P2)

Every rejected and unmatched row lands in a review queue. Someone searches it by key, source, reason, upload,
date or status, opens a row to see its history, corrects it and re-checks it, or dismisses it with a required
comment. A dismissed row can be reopened.

**Why this priority**: it turns silent data loss into visible, fixable work — the point `measure-metrics` §2
makes about facts that count nowhere. It is P2 only because the P1 stories must exist before there is anything
to review.

**Independent Test**: upload a file containing an unknown key and a rejected row; find both in the queue;
correct one and confirm it resolves; dismiss the other with a comment and reopen it.

**Acceptance Scenarios**:

1. **Given** an upload produced rejected and unmatched rows, **When** the review queue is opened, **Then**
   both appear with their reason, and no row can be created by hand.
2. **Given** an unmatched row whose key was a typo, **When** it is corrected and re-checked, **Then** it
   matches an activity and its status becomes *Resolved*.
3. **Given** an open row, **When** a further upload arrives from the same source, **Then** open rows and held
   facts are re-checked automatically, and any that now match become *Resolved*.
4. **Given** a row that will never match, **When** it is dismissed, **Then** a comment is required and stored
   with it; **When** it is later reopened, **Then** its history shows both the dismissal and the reopening.
5. **Given** a row was corrected by hand, **When** a later upload carries the same source and reference,
   **Then** the uploaded value wins and replaces the correction.

---

### User Story 6 - Media codes for campaign links (Priority: P2)

For activities whose type requires them, MAPP generates media codes — unique numbers used in campaign links so
that clicks and inquiries come back attributable. A code is generated with its required attributes, can be
copied on its own or as a tagged link, edited, deactivated or deleted while unused.

**Why this priority**: it supplies the second matching key and is what makes web-sourced facts attributable;
it is P2 because activity ID, integration ID and external name already give working matching without it.

**Independent Test**: on an activity whose type requires media codes, generate one with its required
attributes, copy the tagged link, then try to delete it after a fact has used it and confirm only Deactivate
is offered.

**Acceptance Scenarios**:

1. **Given** an activity whose type does not require media codes, **When** it is opened, **Then** no Media
   codes area is shown; and media codes never attach to a line item or any level other than an activity.
2. **Given** an activity whose type requires them, **When** a code is generated without a required attribute,
   **Then** every missing attribute is reported at once and no code is created.
3. **Given** a code exists, **When** it is copied, **Then** the code alone and a tagged link are both
   available to copy.
4. **Given** a code has never been used by a fact, **When** it is deleted, **Then** it is removed; **Given**
   it has been used, **Then** Delete is not offered and Deactivate is, and its number is never reused.
5. **Given** a code is generated at any activity status, **When** it is created, **Then** it succeeds —
   media codes do not wait for approval, unlike the external name.

---

### Edge Cases

- A row matches an activity that was moved to the recycle bin: the fact is **held** until the activity is
  restored and approved; if the activity is permanently deleted, the row goes to the review queue
  (client 020 FR-013).
- Two rows in the same file carry the same source reference: the second is rejected as a repeated reference
  rather than silently overwriting its neighbour within one upload.
- A file is uploaded for a source whose mappings were changed after an earlier upload: the new mappings apply
  to this upload only; the earlier upload's results stand.
- A revenue row arrives in a currency that has no rate: it is stored exactly, marked, and excluded from
  converted totals with its count shown; adding the rate later includes it without re-uploading.
- An activity is approved, accumulates facts, is declined back and re-approved: its facts are never discarded,
  and are excluded only while it does not count as approved.
- An upload is undone after some of its facts were replaced by a later upload: the undo restores the versions
  it replaced and does not remove the later upload's facts.
- A source is deactivated while rows from it sit open in the review queue: the rows remain reviewable, but
  re-checking them cannot match through that source.
- A fact's date falls outside every plan year in view: it still exists and is listed, and counts in no report
  whose period excludes it.

## Requirements *(mandatory)*

### Functional Requirements

**Data sources** *(client 020 FR-002 – FR-005)*

- **FR-001**: Users MUST be able to create, find, view and edit a data source carrying a name, description,
  owner, category, status, delivery route, cadence and the models it supplies.
- **FR-002**: The system MUST refuse to delete a data source that holds facts, and MUST offer Deactivate in
  its place, stating why Delete is unavailable (046-C9).
- **FR-003**: Within one model, each stage and revenue MUST be supplied by at most one **active** source; a
  conflicting mapping or assignment MUST be refused with the conflicting source named.
- **FR-004**: Deactivating a source MUST keep its facts, refuse further uploads, and update *Is integrated* on
  every affected activity at once; a deactivated source MUST be reactivatable.
- **FR-005**: Each source MUST show its freshness: last successful upload, the date its data runs through, its
  cadence, and a status of *Current*, *Stale*, *Failed* or *Inactive*.

**Column mappings** *(client 020 FR-006 – FR-007)*

- **FR-006**: Users MUST be able to create, find, view, edit and delete mappings, each saying that one file
  column is a matching key, a source reference, a fact date, a stage, revenue with its currency, a web metric
  (**sessions**, **visits** or **engagement touches**, 046-C11) or a journey element.
- **FR-007**: A mapping change MUST affect future uploads only and MUST never restate a completed upload.
- **FR-008**: A blank file template MUST be downloadable from a source's mappings, carrying exactly the
  columns those mappings define.

**Loading data** *(client 020 FR-008 – FR-010, FR-015 – FR-019)*

- **FR-009**: Users with `performance.import` MUST be able to upload a CSV or spreadsheet file for a chosen
  source.
- **FR-010**: A simulated pull MUST produce rows for a source whose delivery route says so, and those rows
  MUST pass through the same validation, matching and review path as an uploaded file (046-C5). A pull MUST be started only by an explicit
  action on the data source by someone holding `performance.import` (046-C13). No live connector, scheduler
  or external credential is part of this feature.
- **FR-011**: Every row MUST be validated with **all** of its problems reported together: empty required
  column, invalid or future date, negative or fractional count, invalid amount or unconfigured currency,
  unknown stage, missing or repeated reference.
- **FR-012**: A file carrying personal-data columns MUST be refused in full before any row is stored, naming
  the offending column; journey rows MUST carry only a pseudonymous reference (046-C6).
- **FR-013**: Loading MUST be idempotent on the pair (data source, source reference): an identical file MUST
  change nothing, and a repeated reference MUST replace the earlier fact with both versions audited (046-C4).
- **FR-014**: A revenue fact MUST be stored exactly in its own currency; without a rate it MUST be marked,
  excluded from converted totals with the excluded count shown, and included once a rate exists (046-C7).
- **FR-015**: A fact MUST count in the fiscal year of its own fact date, including for an activity spanning
  more than one year (046-C8).
- **FR-016**: A load MUST end in one of three recorded states — **completed, failed or undone** (client 020
  FR-062, 046-C17). Uploads MUST be findable by file, source, uploader, status and date and viewable with their
  summary; an upload MUST NOT be editable, and the reason MUST be stated where an edit would be expected.
- **FR-017**: An administrator MUST be able to undo a whole upload after a confirmation stating the counts and
  what will be recalculated; replaced versions MUST be restored and the upload MUST remain listed as *Undone*.

**Matching** *(client 020 FR-011 – FR-014)*

- **FR-018**: Each fact MUST match **exactly one** activity, resolved through activity ID, then media code,
  then integration ID, then external name, in that strict order (046-C2).
- **FR-019**: A row whose keys resolve to different activities MUST be left unmatched with the reason "keys
  disagree" and both candidates named; ambiguous and unknown keys MUST each leave the row unmatched with their
  own stated reason. The system MUST NOT guess between candidates.
- **FR-020**: A fact matched to an activity that does not count as approved MUST be **held**: excluded from
  every figure and released automatically when that activity is approved (046-C1).
- **FR-021**: A fact matched to a binned activity MUST be held until restore and approval; a fact matched to a
  permanently deleted activity MUST go to the review queue.
- **FR-022**: Every load MUST yield a summary of rows read and rows matched, held, unattributed, rejected and
  unmatched, **and those five broken down by stage or web metric** (client 020 FR-014), and the five totals
  MUST add up to rows read (046-C3).

**Rows to review** *(client 020 FR-020 – FR-022)*

- **FR-023**: The review queue MUST hold every rejected and unmatched row, MUST NOT allow a row to be created
  by hand, and MUST be findable by key, source, reason, upload, date and status.
- **FR-024**: A row MUST be viewable with its history, correctable and re-checkable, and dismissable with a
  **required** comment; a dismissed row MUST be reopenable.
- **FR-025**: Open rows and held facts MUST be re-checkable on demand and MUST be re-checked automatically at
  each load from the same source; a row that matches MUST become *Resolved*.
- **FR-026**: A later load carrying the same source and reference MUST replace a corrected row — the source
  wins over the correction.

**Performance on an activity** *(client 020 FR-023 – FR-029)*

- **FR-027**: An activity that does not count as approved MUST show "Approve the activity to see performance
  data" and no figures.
- **FR-028**: An activity counts as approved only while **Approved, In progress or Complete** — never while
  Created, Submitted or Declined, and the rule applies again after re-approval or restore (046-C1).
- **FR-029**: For an approved activity the Performance area MUST show, per stage, goal, actual to date,
  projected for the plan year and conversion; revenue as goal, attributed actual with its model named, and
  projected; web metrics by month; and freshness per contributing source.
- **FR-030**: An activity's facts MUST be listable with date, source, stage or metric, value, currency, the
  key matched on and the originating upload, and MUST be searchable and exportable.
- **FR-031**: The Data sources area MUST show external name and aliases, media-code count, *Is integrated* and
  *Goal entered*, and per source the integration ID, fact count and last fact date, with a notice when the
  model has no source.
- **FR-032**: Every empty performance view MUST state its reason — not approved, no source, no goal, no data
  yet, stale or failed — and MUST never show a sample figure.

**External names and integration IDs** *(client 020 FR-030 – FR-036)*

- **FR-033**: Matching MUST use the activity's existing external name together with its aliases. This
  feature reads that field and never writes it (046-C14).
- **FR-034**: When an activity's external name changes, the superseded name MUST be kept as a matching
  alias, so rows already in flight still match.
- **FR-035**: An alias MUST be unique across external names and aliases, so a key never resolves to two
  activities; a clash MUST be refused with the reason.
- **FR-036**: Generating the name from the model's rule, "Generated on approval", "Regenerate on edit",
  hand-set suppression and the rule's length — client 020 FR-030 – FR-036 — MUST be requested of feature
  006's lane, because `activities.external_name` and its package are not this feature's to change (046-C14).
  Until that change lands the field keeps feature 006's behaviour.
- **FR-037**: An activity MUST carry at most one integration ID per source, unique within that source,
  removable after a confirmation stating the effect on matching, with every change audited.
- **FR-038**: Copying an activity MUST copy no external name, alias, integration ID, media code or fact
  (046-C10).

**Media codes** *(client 020 FR-037 – FR-040)*

- **FR-039**: Whether an activity type requires media codes, and which code attributes exist and are required,
  MUST come from configuration held in Settings › Performance data — this feature's own area — as a list of
  the activity types that require codes and the attribute catalogue those codes carry (046-C12).
- **FR-040**: The Media codes area MUST appear only on activities whose type requires them, and a code MUST
  never attach to a line item or any level other than an activity.
- **FR-041**: A code MUST be generatable at any activity status with all required attributes, reporting every
  missing attribute at once; codes MUST be findable, viewable, editable in attributes and landing page only,
  deletable while unused and deactivatable otherwise.
- **FR-042**: Codes MUST be system-generated numbers, unique and never reused, from a configurable starting
  number so that new codes cannot collide with Hive9's.

**Permissions and scope** *(D-367, constitution non-negotiable 7)*

- **FR-043**: Loading data MUST require `performance.import`; setting external names, aliases, integration IDs
  and media codes MUST require `performance.link`. **Undoing a load is narrower than loading it**: client 020
  FR-017 gives it to administrators alone, so it MUST additionally require `settings.manage` (046-C16).
  Every operation MUST have a route rule and fail closed.
- **FR-044**: Every read of performance data MUST be filtered by the signed-in actor's scope: a fact, source,
  upload, review row or media code outside it MUST answer as a missing one, never as a refusal.

**Records and traceability** *(constitution, every new record type)*

- **FR-045**: Each new record type MUST carry its version and created/updated metadata, MUST refuse a stale
  update, MUST write one activity-log entry per change, and MUST delete to the recycle bin where deletion is
  offered.

### Key Entities

- **Data source**: where a feed comes from — name, description, owner, category, status, delivery route,
  cadence, the models it supplies, and its freshness. Holds mappings and facts.
- **Column mapping**: what one column of a source's file means — matching key, source reference, fact date,
  stage, revenue with currency, web metric or journey element. Belongs to one source; affects future loads only.
- **Performance import**: one upload or simulated pull — its file, source, uploader, date, status, and the
  summary reconciling rows read against matched, held, unattributed, rejected and unmatched. May be undone.
- **Performance fact**: one measured value — date, source, stage or web metric, value, currency, source
  reference, the activity it matched and the key it matched on, and whether it is held. Identified by
  (source, source reference).
- **Row to review**: a rejected or unmatched row kept for a person to fix — its raw values, reason, source,
  upload, status (*Open*, *Resolved*, *Dismissed*) and history, with a required comment on dismissal.
- **Media code**: a unique, never-reused generated number on an activity, with its configured attributes and
  landing page; usable as a matching key and copyable as a tagged link.
- **External name and alias**: the activity's stable outside handle, generated on first approval from the
  model's rule or set by hand; superseded names persist as aliases so rows in flight still match.
- **Integration ID**: an outside system's own identifier for an activity, at most one per source and unique
  within it.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every row of a loaded file ends in exactly one of five outcomes, and the five counts add up to
  rows read in 100% of loads — no row is ever unaccounted for.
- **SC-002**: No performance figure anywhere in MAPP includes a fact matched to an activity that does not
  count as approved, and no approved activity's fact is missing from its figures.
- **SC-003**: Every unmatched or rejected row is findable in the review queue with a stated reason — zero rows
  are discarded silently.
- **SC-004**: Loading the identical file twice changes no figure, and undoing a load returns every figure to
  its value before that load.
- **SC-005**: An administrator can configure a new data source with its mappings and load a first file without
  help from a developer or a database change.
- **SC-006**: A row carrying two keys that disagree is never attributed to a guessed activity — it is always
  reported for review.
- **SC-007**: No file containing personal-data columns results in a single stored row.
- **SC-008**: Every empty performance view states which of the six reasons applies; no view shows a sample or
  placeholder figure.
- **SC-009**: Loading and matching a file at the feature-028 benchmark volumes completes within the time
  recorded in `plan.md`, and matching one fact stays within a constant number of queries regardless of file
  size.
- **SC-010**: A revenue amount is never shown converted at a rate that does not exist; totals that exclude
  facts always show how many were excluded.

## Assumptions

- **Activity statuses and approval** come from features 008 and 031 as they stand; this feature adds no status
  and changes no approval rule — it only reads which statuses count as approved (046-C1).
- **Models, activity types and the external-name rule** live in Settings (feature 002/006) and are read here.
  Feature 047 adds velocity and activity categories to the same models; 046 does not.
- **The projected figures** shown per stage in FR-029 are computed by feature 047. Until 047 lands, those
  columns are absent rather than zero, and the goal/actual/conversion columns stand on their own.
- **Attributed revenue** names its model per FR-029; the attribution models themselves are feature 050. Until
  050 lands, revenue is credited to the activity its fact matched.
- **Display currency and period filters** come from feature 039's shared frame; this feature stores facts and
  does not own conversion or period logic.
- **The import, matching and review pattern** of feature 015 is reused rather than reinvented, and the
  simulated feed follows the adapter shape of the simulated GSAP (ADR-0016, 046-C5).
- **Scope** comes only from the signed-in actor, as for every other plan-owned record (constitution
  non-negotiable 7).
- **Dependency outside this lane**: the wave B demo facts belong in `backend/seed/measure.go`, which lane D
  creates and feature 039 wires into `seed.go`. Until that file exists, 046 cannot seed its demo facts; this
  is a sequencing risk for lanes A and C, who were told to expect those facts early, and is raised in this
  feature's pull request rather than worked around.
- **The benchmark volumes** are feature 028's and are recorded in `plan.md`, not here.

## Dependencies

- Feature 006 (Settings: models, activity types) — the external-name rule, media-code configuration and the
  stage vocabulary.
- Feature 008 (and 031) — approval statuses, which gate every fact.
- Feature 015 — the import, matching and review pattern this feature follows.
- Feature 030 — external keys and the simulated-adapter precedent (ADR-0016).
- Feature 039 — the Measure foundation; **not** a blocker for 046, which has no report route and lives in its
  own package. 046 may start before 039 exists.
- Lane D's `backend/seed/measure.go` — required before 046's demo facts can be seeded (see Assumptions).

## Out of Scope

- Projected stage volumes and revenue (client 020 US7) — feature **047**.
- Attribution models crediting revenue across journeys (client 020 US8) — feature **050**.
- Live performance connectors of any kind (constitution non-negotiable 13, D-370): data arrives only by file
  upload or the simulated feed behind its adapter.
- The reports that read these facts — the Summary (048), Revenue and funnel (049) and ROI (050) reports.
- Any change to approval rules, to the shared report frame, or to the wave contract outside this feature's own
  operations.
