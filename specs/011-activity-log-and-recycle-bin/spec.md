# Feature Specification: Activity Log and Recycle Bin

**Feature Branch**: `011-activity-log-and-recycle-bin`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "Activity log page and record history; recycle bin page with restore and permanent delete. The mechanism (writing entries, moving deletes to the bin, Undo) is in feature 001 — this feature adds the screens and the user-facing restore and purge behaviour."

**References**: Domain model [§1 delete rule, §2.3 and §2.5 activity status and re-approval, §5 R13–R14, §7 activity log and recycle bin entries](../../docs/product/domain-model.md) · Design system [§2 layout, §4 colours, §5 dialog and toast, §6.3 History section, §6.5 delete, §8 words, §10 usability targets](../../docs/product/design-system.md) · Feature 001 [User Story 5, FR-022–FR-027](../001-app-shell-and-sign-in/spec.md) · Demo data [recycle bin and log seed](../../docs/product/demo-users-and-data.md) · Decisions D-004, D-013, D-016, D-019 · Hive9 functional analysis §B8 (recycle bin, change history) · ZenReach spec §6.6–6.7 (audit log, recycle bin), §6.8.9–6.8.10 (their screens)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find and restore a deleted item (Priority: P1)

Under **MORE › Recycle bin** the user sees everything that was deleted, one row per delete: name, level
tag, ID, where it was ("FY2026 Lubricants Europe › Helix launch"), what came with it ("2 activities ·
5 line items"), who deleted it and when. A search box and type filters help to find an item. **Restore**
opens a confirmation that says where the item returns and what changes. After the user confirms, the
item is back with its original IDs.

**Why this priority**: The recycle bin is the safety net behind every delete. In Hive9, restoring takes
4–7 steps; NorthStar's target is at most 3 (design system §10).

**Independent Test**: Delete an initiative with activities, let the Undo message expire, restore it from
the recycle bin, and confirm that everything returns with the same IDs and that activities are Created.

**Acceptance Scenarios**:

1. **Given** the seed data, **When** the user opens the Recycle bin, **Then** the seeded initiative is
   listed with its activity count, who deleted it and when, under "1 item in the recycle bin".
2. **Given** a deleted, approved activity "Launch film", **When** the user chooses **Restore**, **Then**
   a dialog says "Restore activity 'Launch film'? It returns to FY2026 Lubricants Europe › Helix launch ›
   Digital & social with 3 line items. It comes back as Created and must be submitted for approval
   again." The button reads **Restore activity**.
3. **Given** that dialog, **When** the user confirms, **Then** the activity and its line items return
   with their IDs and values, plan totals include them again, the toast reads "Activity 'Launch film'
   restored — View", and the activity log records the restore.
4. **Given** an activity that was deleted together with its initiative, **When** the user searches for
   the activity's ID, **Then** the initiative's row is shown with "Contains A-1042 'Launch film'".
5. **Given** a new initiative named "Digital & social" now exists in the same campaign, **When** the
   deleted one is restored, **Then** the dialog warns, and the restored initiative is named "Digital &
   social (restored)".

---

### User Story 2 - See what changed across NorthStar (Priority: P1)

Under **MORE › Activity log** the user sees summary cards (**Changes today, Changes this week, People
active this week, Deletions this week**), filters, and entries grouped by day. Each entry shows an
avatar, "Priya Raman changed activity 'Launch film'", where the record sits, the time and the action as
a chip. Expanding an entry lists each changed field as old → new. The record name links to the record.

**Why this priority**: People must be able to trust the numbers, which means knowing who changed a
planned cost and when. Hive9's change history was not even in users' menus (functional analysis §B8).

**Independent Test**: Using the seeded two weeks of history plus a few new changes, filter by person,
record type, action, plan and dates, expand entries, and follow their links.

**Acceptance Scenarios**:

1. **Given** the seed history, **When** the page opens, **Then** entries appear newest first under "Today",
   "Yesterday", "Monday 12 Oct 2026" and so on, each group with its count.
2. **Given** an entry "Priya Raman changed activity 'Launch film'", **When** the user expands it,
   **Then** every changed field is listed, e.g. "Planned cost: 40,000.00 USD → 45,000.00 USD" and
   "Status: Approved → Created (needs approval again)".
3. **Given** the filters person "Daniel Okoro" and action "Status changed", **Then** only those entries
   remain, each filter shows as a chip, and the count reads "Showing N of M entries".
4. **Given** an entry about a record now in the recycle bin, **When** the user clicks its name, **Then**
   the Recycle bin opens with that item highlighted. A permanently deleted record's name is plain text
   followed by "(permanently deleted)".
5. **Given** more than 50 matching entries, **Then** the page says "Showing the last 50 of N", and
   **Load more** adds the next 50.

---

### User Story 3 - See a record's history (Priority: P1)

Every plan, campaign, initiative, activity and line item page has a **History** section, which **⋯ More
› View history** also opens. It shows the activity-log entries for that record. Containers and
activities can include the changes of the items inside them.

**Why this priority**: "Who changed this, and what was it before?" is asked on the record itself, for
example by an approver before approving an activity again.

**Independent Test**: Change an activity twice and one of its line items once, then open the activity's
History with and without **Include line items**.

**Acceptance Scenarios**:

1. **Given** an activity changed twice, **When** its History opens, **Then** both entries are shown,
   newest first, and each expands into its field changes.
2. **Given** **Include line items** is on, **Then** the line item's change appears too, naming the line
   item.
3. **Given** a campaign with **Include items inside** on, **Then** entries of its initiatives,
   activities and line items appear, including deleted ones. **Open in activity log** opens the log
   filtered to the campaign and the items inside it.

---

### User Story 4 - Restore an item whose parent was also deleted (Priority: P2)

When an item's parent is also in the recycle bin, its row says so, and Restore offers **Restore both**.
When the parent was deleted permanently, the row explains that the item can no longer be restored.

**Why this priority**: Restoring parents first (domain model R14) is safe, but Hive9 leaves users to
work out the order themselves.

**Independent Test**: Delete an activity, then its initiative, and restore the activity with **Restore
both**. Repeat, but delete the initiative permanently, and confirm the activity cannot be restored.

**Acceptance Scenarios**:

1. **Given** an activity and, deleted later, its initiative are both in the bin, **Then** the activity's
   row says "Its initiative 'Digital & social' is also in the recycle bin".
2. **Given** that row, **When** the user chooses **Restore**, **Then** the dialog explains this, lists
   what the initiative brings back with it, and offers **Restore both**. After confirming, both are back.
3. **Given** the initiative was deleted permanently, **When** the user chooses **Restore** on the
   activity, **Then** NorthStar explains "This activity can't be restored because its initiative
   'Digital & social' was permanently deleted." and offers **Delete permanently**. Nothing changes.

---

### User Story 5 - Delete permanently and empty the recycle bin (Priority: P2)

From a row's **⋯** menu the user can **Delete permanently**, and a header button offers **Empty recycle
bin**. Both are confirmed with counts and the warning "This can't be undone."

**Why this priority**: Users need to clear out mistakes and test data, but it is rarely urgent.

**Independent Test**: Permanently delete one entry, then empty the bin, checking the dialogs, the
result and the activity-log entries.

**Acceptance Scenarios**:

1. **Given** an initiative entry, **When** the user chooses **Delete permanently**, **Then** the dialog
   reads "Delete 'Digital & social' permanently? This also deletes 2 activities and 5 line items. This
   can't be undone." **Cancel** has focus, and the red button says **Delete permanently**.
2. **Given** another entry in the bin belongs inside that initiative, **Then** the dialog also warns "1
   other item in the recycle bin belongs inside it and can no longer be restored."
3. **Given** N entries, **When** the user empties the bin and confirms, **Then** the bin is empty and
   the activity log shows one entry, "emptied the recycle bin (N items)", listing them.

---

### Edge Cases

- Two people restore the same item: the second sees "'Launch film' is no longer in the recycle bin —
  Priya Raman restored it at 14:05", and the list refreshes.
- The parent has been moved since the delete: the item returns under the parent's current place, and
  the dialog shows the current path.
- Restored line items make the activity's unassigned balance negative: the restore goes ahead, and the
  activity shows the usual warning (decision D-016).
- A restored record breaks a rule that changed meanwhile (dates outside the parent's, inactive type or
  workflow, type no longer allowed by the model): the restore goes ahead, and the record's completeness
  indicator and warnings show what to fix.
- The name clashes again: " (restored 2)", " (restored 3)" and so on.
- An entry about a renamed record shows the name at the time and "now 'New name'".
- Empty pages: "Nothing in the recycle bin — when you delete a plan, campaign, initiative, activity or
  line item, it comes here so you can bring it back." and "No changes yet". When filters hide
  everything: "Nothing matches these filters" with **Clear filters**.

## Requirements *(mandatory)*

### Functional Requirements

**Activity log page**

- **FR-001**: NorthStar MUST provide an **Activity log** page under MORE in the sidebar, with the page
  header of design system §2 ("ACTIVITY LOG — Every change, *on the record.*").
- **FR-002**: The page MUST show four summary cards, each with an ⓘ tip:
  - **Changes today**: entries since midnight;
  - **Changes this week**: entries since Monday;
  - **People active this week**: the number of different people with an entry this week;
  - **Deletions this week**: delete, permanent-delete and empty-bin entries this week.

  The cards cover the whole log in the user's local time and do not follow the filters. Clicking a card
  applies the matching filters.
- **FR-003**: The page MUST offer these filters:
  - **Person** (who acted);
  - **Record type** (the types present in the log);
  - **Action**: Created · Changed · Status changed · Moved · Copied · Deleted · Restored · Permanently
    deleted;
  - **Plan**: entries about the plan and anything that is or was inside it;
  - **Record**: one record, optionally with the items inside it; this filter is set only from a
    record's History (FR-013);
  - **Date range**: Today · This week · Last 30 days · This year · Custom;
  - **Search** on record name, ID or person.

  Filters apply at once to the whole log, not only to the entries loaded. Each one shows as a removable
  chip, with **Clear all** and "Showing N of M entries".
- **FR-004**: Entries MUST be listed newest first and grouped by day in the user's local time ("Today",
  "Yesterday", then "Monday 12 Oct 2026"), each group header showing its count.
- **FR-005**: Each entry MUST show the person's avatar, a sentence "<person> <verb> <record type>
  '<name>'", the record's ID, its path at the time ("in FY2026 Lubricants Europe › Helix launch"), the
  time, and an action chip with word and colour (design system §4):
  - Created: green;
  - Changed, Moved and Copied: grey;
  - Status changed and Restored: blue;
  - Deleted and Permanently deleted: red.
- **FR-006**: The verb MUST name what happened: created, changed, published, unpublished, submitted,
  approved, declined, moved, copied, deleted, restored, permanently deleted, emptied the recycle bin.
  Actions added by later features use their own plain verb.
- **FR-007**: Expanding an entry MUST show its details, with values shown as on screen (money with
  currency, dates, people's names, option labels) and "Not filled in" for empty values:
  - each changed field as old → new, with long text shown in full;
  - for a status change, the comment or reason;
  - for a delete, what went with the record;
  - for a move, the old and new place;
  - for a copy, the source;
  - for a restore, the status changes it made.
- **FR-008**: The record name MUST link to the record. A record in the recycle bin opens the bin with
  that item highlighted. A permanently deleted record is plain text with "(permanently deleted)". A
  renamed record shows its name at the time and "now '<current name>'".
- **FR-009**: The page MUST show 50 entries at first, with "Showing the last 50 of N", and **Load more**
  MUST add 50 at a time.
- **FR-010**: Entries MUST be read-only: there is no way to edit or remove them.

**History on record pages**

- **FR-011**: Every plan, campaign, initiative, activity and line item page MUST have a **History**
  section, which **⋯ More › View history** opens (design system §6.3). It shows that record's entries
  as in FR-004–FR-008, newest first, 20 at a time with **Load more**.
- **FR-012**: Plans, campaigns and initiatives MUST offer **Include items inside**, and activities
  **Include line items**. These add the entries of every record that is or was below, each naming that
  record. The choice MUST be remembered for each user.
- **FR-013**: **Open in activity log** MUST open the Activity log with the **Record** filter set to the
  record, including the items inside it when they are included, shown as a removable chip.

**Recycle bin page**

- **FR-014**: NorthStar MUST provide a **Recycle bin** page under MORE, with a page header, a count line
  ("3 items in the recycle bin · Items stay here until you restore or delete them") and an **Empty
  recycle bin** danger button while the bin is not empty.
- **FR-015**: The page MUST list one row per recycle-bin entry, newest first. Each row shows:
  - the name, level tag and ID;
  - "Was in <path at the time of the delete>";
  - what came with it, counted by level;
  - who deleted it (avatar and name), and when ("2 days ago", with the full date and time on hover);
  - **Restore**, and **Delete permanently** in a **⋯** menu.

  Expanding a row MUST list its contents as a tree with names and IDs.
- **FR-016**: The page MUST offer search by name or ID that also searches the contents of each entry
  ("Contains A-1042 'Launch film'"). It MUST also offer pill filters (All · Plans · Campaigns ·
  Initiatives · Activities · Line items, plus any other type present) and "Showing N of M".
- **FR-017**: A row whose parent is missing MUST say why on the row:
  - "Its <parent level> '<name>' is also in the recycle bin"; or
  - "Can't be restored — its <parent level> '<name>' was permanently deleted".

**Restore**

- **FR-018**: **Restore** MUST ask for confirmation, stating where the item returns (the parent's current
  path), what comes with it, every status change (FR-019, FR-020), and every name change (FR-022). The
  buttons are **Cancel** and **Restore <level>**.
- **FR-019**: Restoring from this page MUST apply these status rules:
  - activities return as **Created**, whatever their status was (domain model R14);
  - plans return as **Draft**;
  - campaigns and initiatives recalculate their derived statuses;
  - line items follow their activity.

  Unfinished approval steps are not resumed. Earlier approval history and comments are kept, and the
  restore entry says "Restored from the recycle bin — submit for approval again".
- **FR-020**: Restoring line items into a Submitted or Approved activity MUST send that activity back
  to Created, because its line items change (domain model §2.5). The confirmation says so.
- **FR-021**: A restore MUST bring back every record of the entry with its original IDs (decision
  D-019) and values, apart from FR-019 to FR-022. The records MUST appear again in every view, search
  and total.
- **FR-022**: When a restored record's name clashes with a record under the same parent (or, for a
  plan, with any plan), it MUST get the suffix " (restored)", then " (restored 2)" and so on.
- **FR-023**: When the parent is in the recycle bin, the confirmation MUST offer **Restore both**, or
  **Restore all N** when several levels are missing. It lists the parent entries and everything they
  bring back, and restores them, parents first, in one step.
- **FR-024**: When the parent was deleted permanently, **Restore** MUST stay visible, and choosing it
  MUST explain why the item cannot be restored and offer **Delete permanently**. Nothing changes.
- **FR-025**: A restore MUST NOT be blocked by any rule other than FR-023 and FR-024. Problems in a
  restored record appear through its completeness indicator and warnings.
- **FR-026**: After a restore, the toast MUST read "<Level> '<name>' restored — View". The entry leaves
  the bin, and the activity log records the restore with the actor and the status changes.
- **FR-027**: **Undo** in the delete toast (feature 001, FR-026) MUST still restore exactly what was
  deleted, statuses included, because nothing can have changed in those seconds. FR-019 and FR-020
  apply only to restores from this page.

**Delete permanently and empty the bin**

- **FR-028**: **Delete permanently** MUST ask for confirmation, stating:
  - the name and what is inside it, counted;
  - "This can't be undone";
  - how many other entries in the bin belong inside it and can then no longer be restored.

  **Cancel** has focus, and the red button reads **Delete permanently**.
- **FR-029**: **Empty recycle bin** MUST ask for confirmation, stating the number of entries and what
  they contain, and "This can't be undone". The red button reads **Empty recycle bin**.
- **FR-030**: Permanently deleted records MUST be removed for good, and their IDs are never reused
  (feature 001, FR-027). Activity-log entries about them MUST be kept. The delete MUST be logged as
  "permanently deleted", and emptying the bin as one "emptied the recycle bin (N items)" entry with the
  list; both carry the **Permanently deleted** chip.
- **FR-031**: Nothing MUST leave the recycle bin automatically in this phase. Items stay until a user
  restores them or deletes them permanently.
- **FR-032**: When an entry was already restored or deleted by someone else, any action on it MUST say
  who did it and when, and refresh the list.

### Key Entities *(include if feature involves data)*

- **Activity-log entry** (domain model §7, written by feature 001): actor, time, record type, ID, name
  and path at the time, the plan it belongs to, action, changed fields (old → new), and any comment or
  reason. Read-only.
- **Recycle-bin entry** (domain model §7, written by feature 001): the deleted record with everything
  below it, who deleted it, when, and its parent. This feature derives its **restore state**:
  Restorable · Parent in the recycle bin · Parent permanently deleted.
- **History preference**: each user's **Include items inside** choice.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user restores a deleted item in 1 step (**Undo**) or, from the recycle bin, in at most 3
  (open Recycle bin, Restore, confirm), as design system §10 requires.
- **SC-002**: A user finds a deleted item by name or ID on the recycle bin page in at most 2 steps, even
  when it was deleted together with its parent.
- **SC-003**: 100 % of restores bring back every record of the entry with the same IDs and values,
  except for the status and name changes stated in the confirmation.
- **SC-004**: 100 % of restore confirmations name the destination, the contents and every status change
  before the user confirms.
- **SC-005**: From an activity's page, a user answers "Who last changed the planned cost, and what was it
  before?" in at most 3 steps.
- **SC-006**: With 50,000 entries in the log, the Activity log page shows its first entries, and the
  result of each filter change, in under 2 seconds.
- **SC-007**: Nothing is permanently deleted without a confirmation, and 100 % of restores, permanent
  deletes and bin empties appear in the activity log with the person who did them.
- **SC-008**: Both pages and the History section pass an automated WCAG 2.2 AA check with no failures
  and can be used with the keyboard alone.

## Assumptions

- Feature 001 writes every entry and recycle-bin entry. Each log entry also stores the record's path
  and plan at the time, readable values (labels and names, not codes) and any comment, so the log reads
  correctly after renames and deletes.
- Restored plans come back as Draft. This extends domain model R14, which covers only activities, in the
  spirit of R13 (copies start as Draft): anything once agreed must be agreed again.
- A record cannot be restored into a different parent. The user restores it and then uses **Move**
  (features 005–006).
- A bulk delete (feature 012) creates one recycle-bin entry per selected record, so each can be
  restored on its own.
- Settings, people and team records that their features send to the bin are listed with their type and
  restored with the name rule of FR-022. Configuration records are normally set inactive instead
  (domain model §6).
- Activity-log entries are kept for the whole phase (feature 001, FR-023). There is no export of the
  log, no tamper-evidence, and no notification on restore in this phase.
- No permissions (decision D-004): every user can restore, delete permanently and empty the bin. When
  roles arrive, emptying the bin is expected to be limited to administrators.
- Sign-in, sign-out and saved views (feature 010) are not business records and do not appear in the log.
