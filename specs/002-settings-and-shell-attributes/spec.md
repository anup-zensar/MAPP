# Feature Specification: Settings and Shell Attributes

**Feature Branch**: `002-settings-and-shell-attributes`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "Settings: models, activity types, Shell attributes and options, currencies and rates, finance budget items"

**References**: Domain model [§3 markings, §4.3 currency conversion, §4.5 goals, §5 rules R7–R12, §6 supporting records](../../docs/product/domain-model.md) · Field catalogue [§1–§5 Shell attribute rows = seed configuration](../../docs/product/field-catalogue.md) · Design system [§2 layout, §4 money, §5 components, §6 CRUD pattern, §7 forms, §10 usability targets](../../docs/product/design-system.md) · Demo data [Settings seed](../../docs/product/demo-users-and-data.md) · Decisions D-004, D-008, D-010, D-014, Q-003, Q-005 · Hive9 functional analysis E4.5 (attributes), E7.5.1 (models), E7.6 and F6 (currency) · ZenReach spec §7.3 (models)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Manage models and see their goal waterfall (Priority: P1)

The administrator opens **Settings** (sidebar MORE › Settings), a page of section cards — Models, Activity
types, Shell attributes, Currencies, Finance budget items, plus People and Teams (003) and Approval workflows
(008) — and opens **Models**. A model is the rulebook a plan uses (glossary): INQ→MQL rate, MQL→deal rate,
average deal size (USD) and allowed activity types. The form shows a **live worked example** of the waterfall
(domain model §4.5) for an example number of inquiries the user can change (not saved).

**Why this priority**: A plan cannot be saved without a model and every goal is calculated from it.

**Independent Test**: Create a model and check its example; change a seeded model's rate and check warning and log.

**Acceptance Scenarios**:

1. **Given** the Models list, **When** the user chooses **New model**, enters a name, rates 2 % and 5 %, deal
   size 10,000 USD, ticks two activity types and saves, **Then** the model is listed as Active, the toast
   "Model '<name>' created — View" appears, and new plans can choose it.
2. **Given** those values and the example 250,000 inquiries, **Then** the form reads "250,000 INQ → 5,000 MQL →
   250 deals → 2,500,000.00 USD revenue", and it updates as soon as any value changes.
3. **Given** a model, **When** the user selects it, **Then** the details show every field, the example, "Used by
   2 plans" with links and who last changed it; **Duplicate** opens a new form named "<name> (copy)".
4. **Given** a model used by plans, **When** the user changes a rate or the deal size and saves, **Then**
   NorthStar first says "Used by 2 plans (31 activities). Their calculated MQL and revenue goals will be
   recalculated; typed goals ('Set later stages myself') are kept." with **Save changes** / **Keep editing**,
   and after saving the log shows each old and new value.
5. **Given** an empty name, a rate of 0 % and no activity type, **When** the user saves, **Then** all three
   problems are shown at once and nothing is saved.
6. **Given** activities in the model's plans use "Event", **When** the user un-ticks "Event", **Then** NorthStar
   warns "5 activities use 'Event'. They keep it; new activities can't choose it." before saving.
7. **Given** a model in use, **When** the user chooses **Delete**, **Then** the dialog says "Used by 2 plans —
   set it inactive instead" and offers **Set inactive**; plans keep it as "Lubricants 1.0 (inactive)" and still
   calculate goals. An unused model moves to the recycle bin with **Undo**.

---

### User Story 2 - Define Shell attributes and their options (Priority: P1)

Shell attributes are the configurable fields of each level's form (field catalogue). The list has a level
filter (Plan · Campaign · Initiative · Activity · Line item) and shows, in display order, section, type,
marking, options, "Used by" and status. An attribute opens on a full page with its settings, its options
and a **Preview** of the level's form exactly as users will see it.

**Why this priority**: Shell's process lives in these fields, which Shell wants to maintain itself (functional
analysis F3, decision D-014).

**Independent Test**: Create "Media channel" with four options, preview it, inactivate an option, delete, undo.

**Acceptance Scenarios**:

1. **Given** the Activity level, **When** the user creates "Media channel" (single choice, four options,
   Needed to submit, section "Shell attributes"), **Then** it is listed last in that section and shows on
   activity forms and the preview with the words "Needed to submit".
2. **Given** a choice attribute with no options, or two options named "TV" and "tv", **When** the user saves,
   **Then** "Add at least one option" and "'TV' appears twice" are shown together.
3. **Given** an attribute, **When** the user views it, **Then** each option shows its own "Used by" count and the
   attribute shows "Used by 12 activities (1 in the recycle bin)".
4. **Given** an attribute in use, **When** the user renames it or an option, **Then** every record shows the new
   wording at once and the log keeps old and new labels; the type shows a lock and "Type can't change once
   records use this attribute — create a new attribute instead."
5. **Given** 7 Created activities without a value, **When** the marking changes from Optional to Needed to
   submit, **Then** NorthStar warns "7 activities have no value yet and will show as incomplete. Submitted and
   approved activities keep their status." before saving.
6. **Given** the list, **When** the user moves an attribute up or down or changes its section, **Then** the
   preview and every form of that level follow the new order.
7. **Given** an attribute or option in use, **When** the user chooses **Delete**, **Then** **Set inactive** is
   offered: forms stop offering it, values already on records stay visible read-only as "(inactive)" and never
   block publishing or submitting. An unused attribute moves to the recycle bin with **Undo**.

---

### User Story 3 - Dependent options and business-unit conditions (Priority: P1)

Some options depend on another attribute (Region by Cluster, R8; GL code by Spend type, R11) and some
attributes appear only when the plan's business unit includes Lubricants or Mobility (R9). The administrator
links each dependent option to its parent options and sets conditions; the preview shows the effect.

**Why this priority**: Without them the seed forms of field catalogue §2–§5 cannot work.

**Independent Test**: In the preview, pick a cluster and check the regions; switch the business unit.

**Acceptance Scenarios**:

1. **Given** Region depends on Cluster, **When** a form has cluster "Retail – China", **Then** only its regions
   are offered and, when exactly one fits, it is filled in: "Region set to 'RETAIL – CHINA', the only region for
   this cluster."
2. **Given** GL code "7250080" is linked to three spend types, **When** a line item has any of them, **Then**
   it is offered; saving a dependent attribute with unlinked options lists every unlinked option at once.
3. **Given** "Brands" is shown only when the business unit includes Lubricants, **When** the preview is a
   Mobility-only plan, **Then** Brands is hidden; as a Lubricants plan its section says "Shown because the
   plan's business unit includes Lubricants."
4. **Given** a record whose parent value is removed or whose section becomes hidden, **When** the user saves,
   **Then** dependent values to be cleared are listed first, and hidden values are kept and not counted missing.

---

### User Story 4 - Manage currencies and exchange rates (Priority: P1)

Each currency has an ISO code, a name, a symbol and a rate in units per 1 USD (decision D-010); USD is fixed
at 1. The form shows the rate both ways ("1 USD = 0.92 EUR · 1 EUR = 1.086957 USD").

**Why this priority**: Line items in local currency (007) and display currencies (003) need it.

**Independent Test**: Add JPY; change the EUR rate and check the warning, the new USD values and the log.

**Acceptance Scenarios**:

1. **Given** the Currencies list, **When** the user creates JPY "Japanese yen", symbol "¥", rate 150, **Then** it
   is listed as Active and offered for line items and as a display currency.
2. **Given** a currency, **When** the user views it, **Then** they see its rate both ways, "Used by 14 line
   items", "Display currency of 2 people" and who last changed the rate and when; if it is in use, **Delete**
   offers **Set inactive** instead, and an unused currency moves to the recycle bin with **Undo**.
3. **Given** 14 EUR line items in 6 activities, **When** the EUR rate changes from 0.92 to 0.95, **Then** before
   saving NorthStar says "14 line items in 6 activities use EUR. Their EUR amounts stay the same; their USD
   amounts will be recalculated (52,173.91 → 50,526.32 USD). Unassigned planned cost of these activities will
   change. No activity goes back for approval." and, once confirmed, logs the change.
4. **Given** USD, **Then** its rate and status show a lock and "USD is NorthStar's base currency; its rate is
   always 1."
5. **Given** code "EURO", a code already used and rate 0, **When** the user saves, **Then** all three problems
   are shown at once.

---

### User Story 5 - Maintain activity types and finance budget items (Priority: P2)

An **activity type** has a name and an optional default planned cost (USD) offered when a user picks it (R7);
the models allowing it can be ticked from the type or from the model. A **finance budget item** belongs to a
budget year and a budget name ("2026 Lubricants Budget"); the items sharing both form a **finance budget**.
Plans choose one budget as their default and each line item links to one item (R12). The list is labelled "A
simple list until the Finance module arrives" (domain model §6).

**Why this priority**: The seed covers the demo; administrators need these when Shell's lists change.

**Independent Test**: Create "Webinar" and a 2027 budget item, rename the budget, delete a used and an unused one.

**Acceptance Scenarios**:

1. **Given** the Activity types list, **When** the user creates "Webinar", default planned cost 5,000 USD,
   allowed in "Lubricants 1.0", **Then** it is listed as Active and appears in that model's allowed types.
2. **Given** a type in use, **When** the user views and edits it, **Then** they see its models and "Used by 4
   activities", and a changed default cost says "Applies when someone picks this type from now on."
3. **Given** the finance budget items list (year filter on the current year, items grouped by budget), **When**
   the user creates "Europe – Retail" in "2026 Lubricants Budget", code "LUB-EU-RET", **Then** it appears there.
4. **Given** an item, **When** the user views it, **Then** they see "Linked to 6 line items" and the plans using
   its budget as default; after editing its name or code, the linked line items show the new values, and
   **Rename budget** renames all its items at once, with those plans showing the new budget name.
5. **Given** a type or item in use, **When** the user chooses **Delete**, **Then** **Set inactive** is offered;
   an unused one moves to the recycle bin with **Undo**.

---

### User Story 6 - Import attribute options from a spreadsheet (Priority: P3)

Shell has long lists (1,217 cost centres). The administrator downloads a template, fills one row per option
(plus the parent option for a dependent attribute), uploads it and reviews the result before anything is added.

**Why this priority**: Typing long lists is slow, but the demo needs only short sample lists.

**Independent Test**: Import 1,217 cost centres, three of them faulty, into Cost centre.

**Acceptance Scenarios**:

1. **Given** an uploaded file, **Then** a review lists rows to add, rows already present (skipped) and rows with
   problems (empty, too long, duplicate in the file, parent option not found), each with its row number.
2. **Given** the review, **When** the user chooses **Add 1,214 options**, **Then** they are added, nothing is
   removed or renamed and one log entry records the import; **Cancel** adds nothing.

---

### Edge Cases

- A setting used only by records in the recycle bin counts as used, so a restore never meets a missing setting.
- A choice with 1,217 options is searched by any part of a label ("0220" finds "CN04000220").
- Two people edit the same setting: the second save is refused with "Not saved — Alex Morgan changed this
  model at 10:42" and the typed values stay on screen (shared behaviour of feature 001, FR-029). Nothing
  is overwritten silently.
- Moving a dependent attribute above its parent is refused: "Region must stay below Cluster."
- A rate change that makes line items exceed an activity's planned cost shows the red balance of domain model
  §4.3; approved activities stay approved.

## Requirements *(mandatory)*

### Functional Requirements

**Settings page and shared behaviour**

- **FR-001**: Settings MUST show one card per section (name, one-line explanation, count of active records),
  open each section as its own page with a breadcrumb, and state "Changes here apply to everyone straight away."
- **FR-002**: Every Settings record MUST follow the standard CRUD pattern (design system §6): search, an Active /
  Inactive / All filter, details in a side panel (Shell attributes: a full page, FR-020), **Edit**, all problems
  at once, unsaved-change protection, and its status as **Active** (green) or **Inactive** (grey) chips.
- **FR-003**: Every record MUST show where it is used ("Used by …"), counting records in the recycle bin.
- **FR-004**: A record in use MUST NOT be deleted: **Delete** stays visible, says where it is used and offers
  **Set inactive**. Inactive records MUST NOT be offered for new choices; records already using them keep them,
  labelled "(inactive)" (domain model §6); **Reactivate** reverses this. An unused record MUST be deletable to
  the recycle bin with **Undo** (feature 001).
- **FR-005**: Names MUST be unique within their list, ignoring letter case and surrounding spaces, and the
  message MUST name the clash ("An activity type called 'Event' already exists.").
- **FR-006**: Every create, change (each field old → new), inactivation, reactivation, delete and restore MUST
  be written to the activity log (feature 001), identifying the record by type and name.
- **FR-007**: Changes MUST apply to every user's next form or calculation without a release; a save that would
  overwrite someone else's newer change MUST be refused, naming who changed it and when.
- **FR-008**: Every signed-in user MUST be able to change Settings (D-004). Shell's rule — only administrators
  maintain Settings; only permitted finance roles edit exchange rates — is enforced when roles arrive.

**Models**

- **FR-009**: A model MUST have a name (≤ 80 characters), an INQ→MQL and an MQL→deal rate (each above 0 % and
  at most 100 %, two decimals; above 0 because plan goals are worked backwards from them), an average deal
  size in USD above 0 and at least one allowed activity type. **Duplicate** opens an unsaved "<name> (copy)".
- **FR-010**: The model form and details MUST show the worked example of domain model §4.5 for an editable
  number of inquiries (default 250,000), rounded exactly as goals are.
- **FR-011**: Saving new rates or deal size for a model in use MUST first state how many plans and activities
  use it and that calculated goals will be recalculated (typed goals are kept); the values then apply everywhere.
- **FR-012**: Removing an allowed activity type MUST first state how many activities in the model's plans use
  it; they keep their type and only new choices are limited (R7).

**Activity types and finance budget items**

- **FR-013**: An activity type MUST have a name (≤ 120 characters) and MAY have a default planned cost in USD
  (0 or more), applied only when the type is picked afterwards (R7). Allowed models MUST be editable from the
  type and from the model.
- **FR-014**: A finance budget item MUST have a budget year (current year − 1 to + 5), a budget name and an item
  name (≤ 120 characters each; item names unique within their budget) and MAY have a code (≤ 40, unique within
  the year). Budget names already used that year MUST be offered while typing.
- **FR-015**: The finance budget item list MUST filter by year and group items by budget; **Rename budget** MUST
  rename all its items at once, and plans using that budget as default MUST follow.

**Shell attributes**

- **FR-016**: An attribute MUST have a label (≤ 80 characters, unique within its level), a level, a type from
  domain model §6 and a marking, and MAY have help text (≤ 300 characters) shown as an ⓘ tip on forms.
- **FR-017**: The marking MUST be **Optional** or **Needed to publish** at plan, campaign and initiative level,
  and **Optional** or **Needed to submit** at activity and line-item level (domain model §3). Yes/no attributes
  are always optional and default to No; a needed yes-or-no answer uses a single choice (as PIR does).
  "Multiple choice with % split" MUST be offered only at activity and line-item level, which carry cost.
- **FR-018**: Choice attributes MUST have at least one option. Option labels (≤ 120 characters) MUST be unique
  within the attribute; options have a display order and can be renamed, set inactive, reactivated, and
  removed while unused. Up to 5,000 options MUST stay searchable in Settings and on forms.
- **FR-019**: Each attribute MUST belong to a named section (default "Shell attributes") and have a display order
  in its level; sections appear on forms in the order of their first attribute.
- **FR-020**: The attribute page MUST show the settings, the options with their own "Used by" counts, and a
  **Preview** of the level's form that behaves like the real form and saves nothing.
- **FR-021**: Type and level MUST be locked, with the reason, once any record has a value; before that, changing
  the type of an attribute with options MUST warn that its options will be removed. Changing a marking MUST
  first say how many records become incomplete, and changes no status.
- **FR-022**: A choice attribute MAY depend on a choice attribute of the same level placed before it; each of
  its options MUST be linked to one or more parent options before saving.
- **FR-023**: An attribute MAY be shown only when the plan's business unit includes any of chosen options of the
  plan-level **Business unit** attribute (field catalogue §1 #12). That attribute MUST NOT be deleted, set
  inactive, moved or retyped, because conditions depend on it; the lock MUST say why.

**How Shell attributes behave on forms (features 004–007 and the preview)**

- **FR-024**: Forms MUST show each active attribute of the level in its section and order, with its marking as a
  word and its help tip; choice lists with more than 10 options are searchable (design system §5).
- **FR-025**: A dependent attribute MUST stay visible but disabled with a hint until its parent has a value, then
  offer only active options linked to a selected parent option; when exactly one fits it MUST be filled in and
  say so; removing a parent value MUST list the values to be cleared before saving.
- **FR-026**: Attributes whose condition is not met MUST be hidden, keep their values and never count as
  missing; a conditional section MUST say why it is shown; a plan without a business unit MUST say more
  attributes appear once it is chosen.
- **FR-027**: % split attributes MUST use the split editor of design system §5 and rule R10. Inactive attributes
  and options already on a record MUST show read-only with "(inactive)" and never block publishing or submitting.

**Currencies and exchange rates**

- **FR-028**: A currency MUST have a three-letter code (unique, locked once saved), a name, an optional symbol
  and a rate of units per 1 USD (above 0, up to six decimals), also shown inverted. USD MUST always exist, be
  active and have rate 1; its locks MUST say why.
- **FR-029**: Conversions MUST always use the current rate: a new rate MUST recalculate the USD amount of every
  line item in that currency (domain model §4.3) and every display conversion.
- **FR-030**: Before saving a rate change NorthStar MUST state the number of line items and activities affected
  and their total USD before and after; no activity is sent back for approval.

**Importing options and seed configuration**

- **FR-031**: Users MUST be able to download a template and import one attribute's options from a spreadsheet
  file (Excel or CSV, up to 5,000 rows), see every row's outcome at once and add only new valid options; an
  import never removes, renames or reactivates options.
- **FR-032**: The demo data MUST include the Settings seed of demo-users-and-data.md and the Shell attributes of
  field catalogue §1–§5 with their levels, types, markings, conditions, dependencies (Region → Cluster,
  GL code → Spend type) and 3–8 fictitious options each (decision D-014).

### Key Entities *(include if feature involves data)*

- **Model**: name, two conversion rates, average deal size (USD), allowed activity types, active.
- **Activity type**: name, optional default planned cost (USD), allowed in models, active.
- **Shell attribute**: label, help text, level, type, marking, section, display order, optional parent
  attribute, optional business-unit condition, active; its **options** have a label, order, active flag and,
  for dependent attributes, linked parent options.
- **Currency**: code, name, symbol, rate per 1 USD, active.
- **Finance budget item**: budget year, budget name, item name, code, active; a *finance budget* is the group
  of items with the same year and budget name.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: At least 90 % of first-time administrators create, edit and set inactive a model, an activity
  type, a currency and a finance budget item without help, each within 2 minutes (design system §10).
- **SC-002**: At least 90 % of first-time administrators create a dependent choice attribute with five options
  and check it in the preview within 5 minutes without help.
- **SC-003**: 100 % of Settings form problems are shown in a single save attempt.
- **SC-004**: 100 % of attempts to delete a setting in use are prevented, showing where it is used and the
  inactive alternative.
- **SC-005**: Before 100 % of changes that alter calculated values or completeness (rates, deal size, markings,
  exchange rates) the user sees the number of records affected, and each change is in the activity log.
- **SC-006**: The worked example and recalculated goals match domain model §4.5 for every seeded model
  (Lubricants 1.0: 250,000 INQ → 5,000 MQL → 250 deals → 2,500,000 USD).
- **SC-007**: A user finds one cost centre among 1,217 options in at most 2 steps, in Settings and on a form.
- **SC-008**: A changed setting is visible on the next form any user opens, with no release.
- **SC-009**: Every Settings screen passes an automated WCAG 2.2 AA check and works with the keyboard alone,
  including reordering attributes.

## Assumptions

- The Settings page frame belongs to this feature; 003 adds People and Teams, 008 adds Approval workflows.
- Settings records have no readable IDs (domain model §1 defines them for planning records only).
- One current rate per currency; Hive9's yearly and monthly rates are out of this phase. Always using the
  current rate is a decision to confirm with Shell finance.
- A % split holds one share per value; Hive9's separate performance split serves Measure (out of scope).
- Option lists are fictitious until Shell supplies real lists (Q-003); markings follow the field catalogue
  until Q-005 is answered.
- Finance budget items carry no amounts; real finance budgets arrive with the Finance module.
