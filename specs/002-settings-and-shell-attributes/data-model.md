# Data Model: Settings and Shell Attributes

**Feature**: `002-settings-and-shell-attributes` · **Plan**: [plan.md](plan.md) · **Contract**:
[contracts/openapi.yaml](contracts/openapi.yaml)

The Settings records this feature stores, their fields, rules, relations and life cycles, the money types
every feature uses, and the Shell attribute values that features 004–007 keep on their records. Records are
**relational in shape** (one table per record type, ids and foreign keys, no nested ownership — ADR-0004).
A table is one file `backend/data/<table>.json` (an object whose keys are record ids). JSON field names are
the names used in files and in the API. The conventions of
[001 data-model](../001-app-shell-and-sign-in/data-model.md) apply: UTC timestamps, text lengths in Unicode
characters after trimming, `ActorRef`, `RecordRef`, and the bookkeeping `record.Meta` (`version`,
`createdAt`, `createdBy`, `updatedAt`, `updatedBy`) embedded in every record below and never listed as a
changed field.

Common rules of every Settings record (spec FR-002 – FR-008):

| Rule | Detail |
|---|---|
| Status | `active` boolean, shown as **Active** (green chip) or **Inactive** (grey chip). New records are active. |
| Inactive | Not offered for new choices. Records already using it keep it, labelled "(inactive)"; values keep converting and calculating. **Reactivate** reverses it. |
| In use | A record is *in use* when anything listed in its "Used by" row (§10) refers to it — records in the recycle bin included (spec edge case). A record in use is never deleted (`409 in_use`); **Set inactive** is offered instead. |
| Delete | An unused record moves to the recycle bin (001's `history.MoveToBin`) with **Undo**; restore follows §11. |
| Unique names | Compared ignoring letter case and surrounding spaces (`strings.EqualFold` after `TrimSpace`). The message names the clash. |
| Versions | Every update sends the `version` it started from; an out-of-date save answers `409 version_conflict` naming who changed it and when (001 FR-029). A save that changes nothing writes no version and no log entry. |
| Log | Every create, change (each field old → new), inactivation, reactivation, delete and restore writes one activity-log entry identifying the record by type and name (§12). |
| IDs | Internal ids from 001's `ids.Generator` with the prefixes below; shown in URLs and stored in log entries, never shown on screens as readable IDs (spec Assumptions). Currencies use their ISO code as id. |

New readable-ID counter prefixes (001 data-model §4): `MOD` model, `AT` activity type, `SA` Shell
attribute, `SO` Shell attribute option, `FB` finance budget, `FBI` finance budget item.

---

## 1. Money and exact decimals — package `internal/platform/money` (not stored on its own)

The one implementation of domain model §4 used by every feature (decisions D-010, D-024, D-027). No
binary floating point anywhere (constitution VI).

| Type | JSON | Meaning and rules |
|---|---|---|
| `Amount` | string `"1234.50"` | Exact amount in hundredths (`int64` cents inside). Parsing accepts `^-?[0-9]{1,15}(\.[0-9]{1,2})?$` (so at most 999,999,999,999,999.99); writing always has two decimals. Every currency, JPY included, uses two decimals (domain model §4.4 "rounded to the cent"). |
| `Money` | `{"amount": "1234.50", "currency": "USD"}` | An `Amount` with an ISO currency code (`^[A-Z]{3}$`). The API shape of every amount in NorthStar. |
| `Rate` | string `"0.92"` | Exchange rate = units of a currency per 1 USD (D-010), exact to six decimals (`int64` millionths inside). Parsing accepts `^[0-9]{1,7}(\.[0-9]{1,6})?$`, above 0 and at most 1,000,000. Writing drops trailing zeros (`"0.92"`, `"150"`, `"1.086957"`). `One` = 1. |
| `Percent` | string `"2.00"` | A percentage with two decimals (`int64` hundredths of a percent inside): model rates and % split shares. Parsing accepts `^[0-9]{1,3}(\.[0-9]{1,2})?$`; writing always has two decimals. `Hundred` = 100.00. |
| `Rates` | — | `map[string]Rate`: the current rate of every currency, active or not (D-027: one current rate). |

Rounding is **half-up** — for negative values half away from zero — and every intermediate product uses
`math/big`, so nothing overflows.

| Function | Rule | Source |
|---|---|---|
| `ToUSD(local Amount, r Rate) Amount` | `local ÷ r`, rounded half-up to the cent. 1,000.00 EUR at 0.92 → 1,086.96 USD | Domain model §4.3 |
| `FromUSD(usd Amount, r Rate) Amount` | `usd × r`, rounded half-up to the cent (display currency, feature 003) | Domain model §4 |
| `(Rate) Inverse() Rate` | `1 ÷ r` rounded half-up to six decimals: 0.92 → 1.086957 | Spec US4 |
| `(Amount) SplitEvenly(n) []Amount` | Each part rounded **down** to the cent, the remainder in the last part: 10,000.00 / 3 → 3,333.33, 3,333.33, 3,333.34 | Domain model §4.4 (E) |
| `ConvertSpread(parts []Amount, r Rate) []Amount` | Each part converted with `ToUSD`; the last part gets `ToUSD(sum) − sum(others)`, so the months add up to the line item's USD planned cost | D-024 |
| `PercentSplitEvenly(n) []Percent` | 100.00 % split, rounded down to the hundredth, remainder last: 3 → 33.33, 33.33, 33.34 | R10 |
| `(Percent) OfWhole(n int64) int64` | `n × p ÷ 100`, rounded half-up to a whole number: 250,000 × 2.00 % → 5,000; 25 × 2.00 % = 0.5 → 1 | Domain model §4.5 |
| `(Amount) MulInt(n int64) (Amount, error)` | Exact product; `ErrOutOfRange` beyond the `Amount` limit | Revenue = deals × deal size |
| `Sum`, `Add`, `Sub`, `Neg`, `Cmp`, `Sign`, `IsZero` | Exact; `Sum` of nothing is 0.00 | — |
| `(Rates) ToUSD(m Money) (Amount, error)` | `ErrUnknownCurrency` when the code has no rate; USD returns the amount unchanged | Domain model §4.3 |
| `(Amount) Format()` / `(Money) Format()` / `(Rate) Format()` / `(Percent) Format()` | `1,234.50` / `1,234.50 EUR` / `0.92` / `2.00 %` — the texts of activity-log entries | 001 data-model §5 |

Totals of converted amounts are **sums of the rounded per-line-item values** (what users see on each row).

---

## 2. Model — table `models`

The rulebook a plan uses (glossary): conversion rates, average deal size, allowed activity types.

| Field | Label on screen | Type | Required to save | Validation (message — plan *User-visible messages*) |
|---|---|---|---|---|
| `id` | — | string `MOD-1001` | system | never changes |
| `name` | Name | text | yes | trimmed; 1–80 characters; unique among models (FR-005, FR-009) |
| `inqToMqlRate` | INQ→MQL rate | `Percent` | yes | above 0.00 and at most 100.00, at most two decimals (FR-009) |
| `mqlToDealRate` | MQL→deal rate | `Percent` | yes | same as above |
| `averageDealSize` | Average deal size | `Money` (USD) | yes | currency `USD`; above 0.00; at most 100,000,000.00 |
| `activityTypeIds` | Allowed activity types | list of activity-type ids | yes (≥ 1) | each an existing activity type; a type **newly added** must be active; no duplicates; stored in the order of the activity-type list (by name) |
| `active` | Status | boolean | system | default `true` |
| *bookkeeping* | | `record.Meta` | | |

Derived (API only): `activityTypes` (`SettingRef` list: id, name, active), `usage` (§10), the worked
example is computed by the web app from the same formula (§2.1).

Relations: `activityTypeIds` → `activity_types.id` (many-to-many, stored on the model — the model is the
rulebook, domain model §6). Plans (004) refer to a model by `modelId`.

### 2.1 Goal waterfall (domain model §4.5, FR-010, SC-006)

`settings.Waterfall(inq int64, m Model) (Goals, error)` with `Goals{INQ, MQL, Deals int64; Revenue
money.Amount}`:

```
MQL     = inqToMqlRate.OfWhole(INQ)        (half-up to a whole number)
Deals   = mqlToDealRate.OfWhole(MQL)       (half-up to a whole number)
Revenue = averageDealSize.MulInt(Deals)    (exact; ErrOutOfRange when too large)
```

| Model | INQ | MQL | Deals | Revenue (USD) |
|---|---|---|---|---|
| Lubricants 1.0 (2.00 %, 5.00 %, 10,000.00) | 250,000 | 5,000 | 250 | 2,500,000.00 |
| Mobility 1.0 (3.00 %, 4.00 %, 5,000.00) | 250,000 | 7,500 | 300 | 1,500,000.00 |
| Lubricants 1.0 | 25 | 1 (0.5 rounds up) | 0 (0.05) | 0.00 |
| Lubricants 1.0 | 0 | 0 | 0 | 0.00 |

The web app's `lib/waterfall.ts` implements the same formula with `BigInt`; both are tested against this
table. Working a goal **backwards** (plan goal stage MQL or Revenue) is feature 004's rule.

### 2.2 Life cycle

```
            create                    Set inactive                 
 (none) ───────────► Active ◄───────────────────────► Inactive
                       │            Reactivate           │
                       └──── Delete (only unused) ───────┴──► recycle bin ──Undo / Restore──► back (same status)
```

---

## 3. Activity type — table `activity_types`

| Field | Label | Type | Required to save | Validation |
|---|---|---|---|---|
| `id` | — | string `AT-1001` | system | |
| `name` | Name | text | yes | trimmed; 1–120 characters; unique among activity types (FR-005, FR-013) |
| `defaultPlannedCost` | Default planned cost | `Money` (USD) or `null` | no | currency `USD`; 0.00 or more (FR-013); at most 999,999,999,999,999.99 |
| `active` | Status | boolean | system | |
| *bookkeeping* | | | | |

Derived (API only): `models` — the models whose `activityTypeIds` contain this type (`SettingRef` list),
`usage` (§10).

"Allowed in models" is edited from the type (body field `modelIds`) and from the model
(`activityTypeIds`); both write the model records (§12). A model must keep at least one allowed type, so
removing a type from a model where it is the only one is refused. The default planned cost applies only
when a type is picked afterwards (R7; feature 006 uses it).

---

## 4. Shell attribute — table `shell_attributes`

A configurable field of one level's form (field catalogue, D-014).

| Field | Label | Type | Required to save | Validation |
|---|---|---|---|---|
| `id` | — | string `SA-1001` | system | |
| `label` | Label | text | yes | trimmed; 1–80 characters; unique among the attributes **of the same level** (FR-016) |
| `helpText` | Help text | long text | no | at most 300 characters; shown as an ⓘ tip on forms (FR-016) |
| `level` | Level | enum `plan` · `campaign` · `initiative` · `activity` · `lineItem` | yes | locked once any record has a value (FR-021) |
| `type` | Type | enum `singleChoice` · `multipleChoice` · `multipleChoiceSplit` · `yesNo` · `text` · `longText` · `number` | yes | `multipleChoiceSplit` only at `activity` and `lineItem` (FR-017, D-029); locked once any record has a value (FR-021) |
| `marking` | Marking | enum `optional` · `neededToPublish` · `neededToSubmit` | yes | `neededToPublish` only at plan, campaign, initiative; `neededToSubmit` only at activity, line item; `yesNo` always `optional` (FR-017) |
| `section` | Section | text | yes (default "Shell attributes") | trimmed; 1–60 characters; empty → "Shell attributes" (FR-019) |
| `position` | Place on the form | integer ≥ 1 | system | see §4.2 |
| `parentAttributeId` | Depends on | attribute id or `""` | no | only for choice types; the parent is a `singleChoice` or `multipleChoice` attribute of the **same level placed before it** (FR-022); the Business unit attribute never depends on anything |
| `condition` | Shown when the plan's business unit includes | list of Business unit option ids | no (empty = always shown) | each an option of the Business unit attribute; not allowed on the Business unit attribute itself (FR-023) |
| `role` | — | enum `""` · `businessUnit` | system | `businessUnit` marks the plan-level Business unit attribute (field catalogue §1 #12), created by the seed; exactly one may exist |
| `active` | Status | boolean | system | |
| *bookkeeping* | | | | |

Derived (API only): `parentAttribute` and condition option labels (`SettingRef`s), `optionCount`,
`activeOptionCount`, `usage` (§10), `locks` (§4.3).

Labels of the enums on screen:

| Value | On screen |
|---|---|
| `plan` · `campaign` · `initiative` · `activity` · `lineItem` | Plan · Campaign · Initiative · Activity · Line item |
| `singleChoice` · `multipleChoice` · `multipleChoiceSplit` · `yesNo` · `text` · `longText` · `number` | Single choice · Multiple choice · Multiple choice with % split · Yes/no · Text · Long text · Number |
| `optional` · `neededToPublish` · `neededToSubmit` | Optional · Needed to publish · Needed to submit |

Choice types are `singleChoice`, `multipleChoice`, `multipleChoiceSplit`: they have options, at least one
(FR-018).

### 4.1 Rules that make an attribute "locked"

| Lock (`locks[].field`) | When | Reason shown |
|---|---|---|
| `type`, `level` | any record (live or in the recycle bin) has a value (FR-021) | "Type can't change once records use this attribute — create a new attribute instead." / "Level can't change once records use this attribute — create a new attribute instead." |
| `type`, `level`, `section`, `position`, `active`, `delete`, `condition`, `parentAttributeId` | `role = businessUnit` (FR-023) | "Business unit can't be deleted, set inactive, moved or given another type, because other attributes are shown or hidden by it." |
| `active` (set inactive) | an **active** attribute depends on it | "Set '<dependent label>' inactive first — it depends on '<label>'." |

Before any record has a value, changing the type from a choice type to a type without options removes the
attribute's options (warned first, FR-021; Proposed decision in the plan). Changing between choice types
keeps them.

### 4.2 Order and sections (FR-019, FR-024)

- The **display order** of a level: sections ordered by their lowest `position`; inside a section,
  attributes by `position`, then by `id`. Forms, the preview and the Settings list all use it.
- A new attribute gets `position` = (highest position of its section) + 1, or (highest of the level) + 1
  when its section is new, so it is listed **last in its section** (US2-1). Changing the section keeps the
  rule: the attribute goes last in its new section; nothing else moves.
- **Move up / down** swaps the attribute's position with its neighbour **in the same section**; at the
  edge of its section it is refused ("'<label>' is already first in its section." / "… last …").
- **Move section up / down** puts the whole section before the previous (after the next) section and
  renumbers the level's positions 1…n in the new order.
- After any move, every dependent attribute must still come after its parent ("Region must stay below
  Cluster.") and the Business unit attribute must stay first on the plan form; otherwise the move is
  refused and nothing changes.
- Each attribute whose position changed gets a new version and one `changed` log entry (field
  `position`).

### 4.3 Conditions (R9, FR-023, FR-026)

An attribute with a non-empty `condition` is **shown** on a record only when the plan's Business unit value
includes **any** of the condition's options; otherwise it is **hidden**: not shown, its value kept, never
counted as missing. A section is *conditional* when **every** attribute in it has a condition; a shown
conditional section says why ("Shown because the plan's business unit includes Lubricants."); a single
conditional attribute in an unconditional section says it under the field. A plan without a business unit
shows no conditional attribute and the note "More Shell attributes appear once the plan's business unit is
chosen."

---

## 5. Shell attribute option — table `shell_attribute_options`

| Field | Label | Type | Required to save | Validation |
|---|---|---|---|---|
| `id` | — | string `SO-1001` | system | |
| `attributeId` | — | attribute id | system | the attribute it belongs to; never changes |
| `label` | Label | text | yes | trimmed; 1–120 characters; unique within its attribute (FR-018) — "'TV' appears twice." in one request, "'TV' is already an option of Media channel." against stored options |
| `position` | Place in the list | integer ≥ 1 | system | new options go last; Move up / down swaps with the neighbour |
| `parentOptionIds` | Linked to | list of option ids of the parent attribute | for options of a dependent attribute (≥ 1) | each an option of the attribute's parent; a link **newly added** must be active (FR-022) |
| `active` | Status | boolean | system | |
| *bookkeeping* | | | | |

At most **5,000 options** per attribute (active and inactive). Search matches any part of the label
ignoring letter case ("0220" finds "CN04000220", SC-007). Derived (API only): `parentOptions`
(`SettingRef` list), `usage` (§10).

---

## 6. Currency — table `currencies`

| Field | Label | Type | Required to save | Validation |
|---|---|---|---|---|
| `code` | Code | string, also the id | yes | exactly three letters A–Z, stored in upper case; unique; **locked once saved** (FR-028) |
| `name` | Name | text | yes | trimmed; 1–80 characters; unique among currencies |
| `symbol` | Symbol | text | no | at most 5 characters |
| `rate` | Rate (units per 1 USD) | `Rate` | yes | above 0, at most six decimals, at most 1,000,000 (FR-028) |
| `rateChangedAt` | Rate last changed | timestamp | system | set on create and whenever `rate` changes |
| `rateChangedBy` | — | ActorRef | system | |
| `active` | Status | boolean | system | |
| *bookkeeping* | | | | |

Derived (API only): `inverseRate` (`Rate.Inverse()`), `usage` (§10), `locks`.

**USD** always exists, is active and has rate 1: its `rate`, `active` and delete are locked with "USD is
NorthStar's base currency; its rate is always 1." (FR-028). Its name and symbol may change.

A rate change applies at once everywhere (FR-029): features 003 and 007 never store converted values; they
convert when they read, with `settings.(*Service).Rates` (Proposed decision "Settings-derived values are
computed on read"). No activity changes status (domain model §2.5).

---

## 7. Finance budget — table `finance_budgets`

A finance budget is the group of items with the same budget year and budget name (spec Key Entities). It
is stored as its own small record so that **Rename budget** changes one record and plans that chose it as
their default (feature 004, by id) follow at once (FR-015).

| Field | Label | Type | Rules |
|---|---|---|---|
| `id` | — | string `FB-1001` | |
| `year` | Budget year | integer | never changes |
| `name` | Budget name | text | trimmed; 1–120 characters; unique within its year |
| *bookkeeping* | | | |

- Created automatically when an item is saved with a budget name not yet used that year (ignoring case);
  an item saved with an existing name joins that budget.
- Never deleted on its own in this phase. A budget without items is not listed and not offered for new
  choices; a plan that already uses it keeps it.
- Derived (API only): `items`, `activeItemCount`, `usage` (plans using it as their default, §10).

## 8. Finance budget item — table `finance_budget_items`

| Field | Label | Type | Required to save | Validation |
|---|---|---|---|---|
| `id` | — | string `FBI-1001` | system | |
| `budgetId` | Budget name | finance budget id | yes | the request sends `year` + `budgetName`; the service finds or creates the budget (§7) |
| `name` | Item name | text | yes | trimmed; 1–120 characters; unique within its budget (FR-014) |
| `code` | Code | text | no | trimmed; at most 40 characters; unique within the budget **year** ignoring case (FR-014) |
| `active` | Status | boolean | system | |
| *bookkeeping* | | | | |

- The budget **year** (from the budget) is chosen on create from current year − 1 to current year + 5
  (the clock's UTC year) and is locked afterwards: "The year can't change once the item is saved — create a
  new item for another year." Changing the budget name moves the item to that budget of the same year.
- Derived (API only): `year`, `budgetName`, `usage` (§10).
- Line items (007) link to one item by `financeBudgetItemId` (R12).

---

## 9. Shell attribute values on planning records (features 004–007)

Plans, campaigns, initiatives, activities and line items keep their Shell attribute values **inside their
own record**, in the field `attributes` (Proposed decision "Attribute values live on the record"):

```json
"attributes": {
  "SA-1003": ["SO-1010", "SO-1011"],
  "SA-1004": ["SO-1020"],
  "SA-1013": [{"optionId": "SO-1050", "share": "60.00"}, {"optionId": "SO-1051", "share": "40.00"}],
  "SA-1005": true,
  "SA-1020": "FIT4-0042",
  "SA-1031": "Long text…",
  "SA-1040": "SO-1101"
}
```

Go type `settings.AttributeValues` = `map[string]json.RawMessage` (attribute id → value). API schema
`AttributeValues`. The value's JSON shape follows the attribute's type:

| Type | Value | Empty (= no value, key removed) | Rules when saving |
|---|---|---|---|
| `singleChoice` | option id (string) | `""`, `null` | an option of this attribute |
| `multipleChoice` | list of distinct option ids | `[]`, `null` | options of this attribute, each once |
| `multipleChoiceSplit` | list of `{optionId, share}` (`share` a `Percent` above 0) | `[]`, `null` | each option once; any total may be **saved**; the total must be exactly 100.00 % to count as filled in (R10) |
| `yesNo` | boolean | `null` | always optional; missing reads as **No** |
| `text` | string | `""`, `null` | at most 120 characters |
| `longText` | string | `""`, `null` | at most 2,000 characters |
| `number` | whole number | `null` | from −999,999,999,999 to 999,999,999,999 |

`settings.(*Service).CheckAttributes(ctx, level, AttributeInput{Values, Previous, BusinessUnit})`
(the one implementation of FR-024 – FR-027 for every form, and of the preview's **Try saving**) returns:

| Part | Content | Effect |
|---|---|---|
| `Problems` | field `attributes.<id>` with a message: wrong shape, unknown attribute or option, an attribute of another level, text too long, number out of range, share not above 0 or with more than two decimals, an option twice; an **inactive** attribute whose value changed; an **inactive** option or a dependent option **not linked** to a selected parent option — only when **newly chosen** (not in `Previous`) | blocks saving |
| `Cleared` | dependent values whose parent value **changed in this save** so that they no longer fit (each `{attributeId, label, optionIds, optionLabels}`) | removed from the saved values; the form listed them before saving (FR-025) |
| `Missing` | `MissingField{field, label, reason}` for every **needed** attribute (marking of the level) that is active, shown (condition met), empty — or a split not totalling 100.00 % (reason "Shares add up to 90.00 % — they must add up to 100.00 %.") | blocks publishing (plan, campaign, initiative) or submitting (activity, line item) |
| `Values` | the values to store: trimmed, empty values removed, cleared values removed | — |

Never missing: yes/no attributes; inactive attributes; hidden attributes (FR-026); a dependent attribute
whose selected parent options have **no active linked options** (D-034 — the form says "No options set up
for this cluster — ask an administrator"); a needed attribute whose only value is an inactive option
(FR-027: never blocks).

---

## 10. Where a record is used ("Used by", FR-003)

Computed on every read from (a) the Settings records themselves, live and in the recycle bin, and (b) the
**references** that features 003–007 report through `settings.ReferenceSource` (plan "Dependencies"):
one `settings.Reference` per plan, campaign, initiative, activity, line item and person, live
(`InBin = false`) or in the recycle bin (`InBin = true`). Until those features exist nothing outside
Settings refers to Settings records.

| Record | Counts as used by | Usage parts (in this order) |
|---|---|---|
| Model | plans with `ModelID` | "Used by 2 plans" |
| Activity type | activities with `ActivityTypeID`; models whose `activityTypeIds` contain it | "Used by 4 activities" · "Allowed in 2 models" |
| Shell attribute | records of its level with a non-empty value; attributes whose `parentAttributeId` is it; (Business unit) attributes with a condition | "Used by 12 activities" · "Region depends on it" / "3 attributes depend on it" · "Condition of 5 attributes" |
| Option | records whose value contains it; options linked to it (`parentOptionIds`); attributes whose condition contains it | "Used by 3 activities" · "Linked to 4 Region options" · "Condition of 5 attributes" |
| Currency | line items with `Currency`; people whose display currency it is | "Used by 14 line items" · "Display currency of 2 people" |
| Finance budget item | line items with `FinanceBudgetItemID` | "Linked to 6 line items" |
| Finance budget | plans with `FinanceBudgetID` (shown, never blocks: budgets are not deleted) | "Default finance budget of 1 plan" |

- Each part counts records **in the recycle bin** too and says how many: "Used by 12 activities (1 in the
  recycle bin)". Settings records in the recycle bin count the same way (a model in the bin keeps its
  activity types in use).
- `summary` joins the parts with " · "; an unused record reads "Not used yet".
- `records` (models and finance budgets only) lists the live plans using it, sorted by name, for links
  (at most 50).
- **Impact statements** (`…/impact`, plan "Impact before saving") count **live** records only: records in
  the recycle bin are recalculated when restored, because derived values are computed on read.

## 11. Recycle bin: kinds, restore rules

| Kind (`history.Kind.Name`) | Singular / plural | Parent kind | Snapshot name | Restore (`PrepareRestore`) |
|---|---|---|---|---|
| `model` | model / models | — | name | name taken: **Undo** refused (`409 name_taken`, 001 message); **Restore** renames to "<name> (restored)", "(restored 2)", … with the note |
| `activityType` | activity type / activity types | — | name | same name rule |
| `shellAttribute` | Shell attribute / Shell attributes | — | label | label taken **in its level**: same name rule; keeps its position (ties sort by id) |
| `shellAttributeOption` | option / options | `shellAttribute` | label | label taken in its attribute: same name rule; the attribute no longer has options (its type changed): refused `409 cannot_restore` "Can't restore — '<attribute>' no longer has options." |
| `currency` | currency / currencies | — | code | code taken: **Undo** `409 name_taken` and **Restore** `409 cannot_restore` "Can't restore — there is now another currency with the code 'JPY'." |
| `financeBudget` | finance budget / finance budgets | — | name | never moved to the bin in this phase (registered as the parent of items) |
| `financeBudgetItem` | finance budget item / finance budget items | `financeBudget` | name | name taken in its budget: same name rule; code taken in its year: **Undo** `409 name_taken`; **Restore** removes the code with the note "Code '<code>' removed because another item now uses it." |

Deleting an attribute takes its options with it: one bin entry "Shell attribute 'Media channel', with 4
options". Restored records keep their ids, links and status. Because a record used by anything — also by
something in the recycle bin — cannot be deleted, a restore never meets a missing setting (spec edge
case).

## 12. Activity-log entries

`recordType` / `recordTypeLabel` / `recordId` / `recordName` as in §11. Field labels and texts come from
each kind's `Describe`:

| Kind | Field → label: text |
|---|---|
| model | `name` → Name · `inqToMqlRate` → INQ→MQL rate: "2.00 %" · `mqlToDealRate` → MQL→deal rate · `averageDealSize` → Average deal size: "10,000.00 USD" · `activityTypeIds` → Allowed activity types: names joined with ", " · `active` → Status: "Active" / "Inactive" |
| activityType | `name` → Name · `defaultPlannedCost` → Default planned cost: "25,000.00 USD" · `active` → Status |
| shellAttribute | `label` → Label · `helpText` → Help text · `level` → Level · `type` → Type · `marking` → Marking (words of §4) · `section` → Section · `position` → Place on the form · `parentAttributeId` → Depends on: parent label · `condition` → Shown when the business unit includes: option labels, or "Always" · `active` → Status |
| shellAttributeOption | `label` → Label · `parentOptionIds` → Linked to: parent option labels · `position` → Place in the list · `active` → Status |
| currency | `name` → Name · `symbol` → Symbol · `rate` → Rate (units per 1 USD): "0.92" · `active` → Status |
| financeBudget | `name` → Budget name |
| financeBudgetItem | `name` → Item name · `code` → Code · `budgetId` → Budget: budget name · `active` → Status |
| planning kinds (004–007) | `attributes.<attributeId>` → the attribute's label: option labels joined with ", ", splits "China 60.00 %, India 40.00 %", "Yes"/"No", the text, the number — via `settings.(*Service).DescribeAttribute` |

| Person's action | Entries |
|---|---|
| Create a model / type / currency / finance budget item | 1 × `created` (an item that creates its budget: 1 × `created` for the budget, then 1 × `created` for the item) |
| Create a Shell attribute with 4 options | 1 × `created` on the attribute, details "with 4 options" |
| Add one option | 1 × `created` on the option (path: the attribute) |
| Edit (each changed field old → new) | 1 × `changed` per record that changed. Editing a type's allowed models also writes 1 × `changed` on each model whose list changed (field `activityTypeIds`); creating a type allowed in models does the same. Linking options while making an attribute dependent writes 1 × `changed` per option whose links changed |
| Model rates or deal size changed while in use | the `changed` entry's details: "Used by 2 plans (31 activities): calculated goals recalculated." |
| Exchange rate changed | the `changed` entry's details: "14 line items in 6 activities recalculated: 52,173.91 → 50,526.32 USD." (or empty when none) |
| Type changed so its options were removed | the `changed` entry's details: "Removed its 4 options: TV, Radio, Print, Online." |
| Set inactive / Reactivate | 1 × `statusChanged`, verb `deactivated` / `reactivated`, change `active` Active ↔ Inactive |
| Move an attribute or option, move a section | 1 × `changed` (field `position`) per record whose place changed |
| Import options | 1 × `changed` on the attribute, verb `imported options into`, details "Added 1,214 options from 'cost-centres.xlsx'; skipped 3 rows." |
| Rename budget | 1 × `changed` on the finance budget (field `name`) |
| Delete / Undo / Restore | as 001 (`deleted`, `restored`) through `history` |

## 13. Seed (decision D-014, FR-032, demo-users-and-data.md "Settings seed")

Created by `backend/seed/seed.go` (`addSettings`) through the settings repository and the ID generator,
by `actor.System`, in this order (so ids are stable): currencies, activity types, models, finance budgets
and items, Shell attributes with their options. Option lists are **fictitious samples**.

| Record | Seed |
|---|---|
| Currencies (id = code) | USD "US dollar" "$" 1 · EUR "Euro" "€" 0.92 · GBP "Pound sterling" "£" 0.79 · CNY "Chinese yuan" "¥" 7.10 · INR "Indian rupee" "₹" 83.50 |
| Activity types `AT-1001`…`AT-1006` | Agency fees – Creative · Event (default 25,000.00 USD) · Digital media (10,000.00 USD) · Content production · Sponsorship (50,000.00 USD) · MOBILITY – Long-term contract |
| Models | `MOD-1001` **Lubricants 1.0** 2.00 % · 5.00 % · 10,000.00 USD · the first five types · `MOD-1002` **Mobility 1.0** 3.00 % · 4.00 % · 5,000.00 USD · all six types |
| Finance budgets (2026) | `FB-1001` 2026 Lubricants Budget: Europe – Digital `LUB-EU-DIG` · Europe – Events `LUB-EU-EVT` · Europe – Brand `LUB-EU-BRD` · `FB-1002` 2026 Mobility Budget: China – Retail `MOB-CN-RET` · China – Fleet `MOB-CN-FLT` · Delhi `MOB-IN-DEL` (items `FBI-1001`…`FBI-1006`) |

Shell attributes (`SA-1001`… in this order; section "Shell attributes" unless stated; positions 1…n per
level in this order):

| Level | Label | Type | Marking | Depends on / condition | Options (fictitious) |
|---|---|---|---|---|---|
| Plan | Business unit (`role` businessUnit) | multiple choice | Needed to publish | — | Mobility · Lubricants |
| Plan | Strategic Marketing Plan (SMP) | yes/no | Optional | shown when BU includes Mobility | — |
| Campaign | Cluster | multiple choice | Needed to publish | — | Lubricants – Europe · Lubricants – Asia Pacific · Retail – China · Fleet – China · Retail – India |
| Campaign | Region | multiple choice | Needed to publish | depends on Cluster | LUBES – UK & IRELAND, LUBES – BENELUX, LUBES – DACH (→ Lubricants – Europe) · LUBES – SOUTH EAST ASIA (→ Lubricants – Asia Pacific) · RETAIL – CHINA (→ Retail – China) · FLEET – NORTH CHINA, FLEET – SOUTH CHINA (→ Fleet – China). Retail – India has none (shows D-034) |
| Campaign | Insight session completed · Workshop completed · Global GM sign-off completed | yes/no | Optional | — | — |
| Initiative | Growth ambition | multiple choice | Optional | — | LUBRICANTS – Premium growth · LUBRICANTS – B2B expansion · MOBILITY – Integrated offer · MOBILITY – Convenience retail |
| Initiative | Function | single choice | Optional | — | Marketing · Sales · Customer service · Digital |
| Initiative | Focus area | single choice | Optional | — | Brand · Customer acquisition · Loyalty · Product launch |
| Initiative | FIT4 initiative | yes/no | Optional | — | — |
| Activity | Channel / audience | multiple choice | Needed to submit | — | External · Internal · Partners |
| Activity | Country / OU | multiple choice with % split | Needed to submit | — | China · India · United Kingdom · Netherlands · Germany · Singapore |
| Activity | Company code | single choice | Needed to submit | — | GB01 · NL02 · CN04 · IN05 · DE03 · SG06 |
| Activity | Cost centre | single choice | Needed to submit | — | CN04000220 · CN04000231 · GB01000105 · GB01000118 · NL02000340 · IN05000412 · DE03000150 · SG06000077 |
| Activity | Global campaign | multiple choice | Needed to submit | — | Go Well integrated campaign · Helix Ultra global platform · Shell Recharge launch |
| Activity | Growth ambition | multiple choice | Needed to submit | — | the four initiative values |
| Activity | PIR | single choice | Needed to submit | — | Yes · No |
| Activity | FIT4 initiative | single choice | Needed to submit | — | FIT4 – Cost · FIT4 – Growth · Not FIT4 |
| Activity | FIT4 ID | text | Needed to submit | — | — |
| Activity | Zero-based cost hierarchy | multiple choice | Optional | — | Media · Agency · Events · Production |
| Activity (section "Lubricants attributes") | Brands | multiple choice with % split | Needed to submit | shown when BU includes Lubricants | Helix · Rimula · Spirax · Tellus |
| Activity ("Lubricants attributes") | Global key accounts | multiple choice | Needed to submit | Lubricants | Account A – Automotive OEM · Account B – Mining · Account C – Logistics |
| Activity ("Lubricants attributes") | Sector | multiple choice | Needed to submit | Lubricants | Automotive · Mining · Construction · Agriculture |
| Activity ("Lubricants attributes") | Sub-sector | multiple choice | Needed to submit | Lubricants | Passenger cars · Heavy duty · Off-highway |
| Activity ("Lubricants attributes") | Product launch | single choice | Needed to submit | Lubricants | Yes – new product · Yes – relaunch · No |
| Activity ("Mobility attributes") | Function (Retail) | single choice | Optional | Mobility | Marketing · Operations · Category management |
| Activity ("Mobility attributes") | Focus area (Retail) | single choice | Optional | Mobility | Fuel · Convenience · Car wash |
| Activity ("Mobility attributes") | Function (Mobility) | single choice | Optional | Mobility | Marketing · Fleet sales · E-mobility |
| Activity ("Mobility attributes") | Focus area (Mobility) | single choice | Optional | Mobility | Fleet solutions · EV charging · Loyalty |
| Activity ("Mobility attributes") | Innovation | long text | Optional | Mobility | — |
| Line item (section "Finance coding") | Spend type | single choice | Needed to submit | — | Digital – Platform development · Digital – Paid media · BTL – Event · ATL – TV · Agency – Creative |
| Line item ("Finance coding") | GL code | single choice | Needed to submit | depends on Spend type | 7250080 – Advertising digital (→ Digital – Platform development, Digital – Paid media, Agency – Creative) · 7250010 – Advertising TV (→ ATL – TV) · 7250040 – Events and sponsorship (→ BTL – Event) · 7250060 – Agency fees (→ Agency – Creative) · 7250090 – Digital platforms (→ Digital – Platform development) |
| Line item ("Finance coding") | Spend category | single choice | Optional | — | B2B customer specific · B2C mass market · Internal |
| Line item ("Finance coding") | Contracted spend | single choice | Optional | — | Yes – long term · Yes – short term · No |

Help texts: Business unit "Lubricants or Mobility. Decides which extra Shell attributes the plan's
activities show."; Region "Choose a cluster first — each region belongs to a cluster."; Country / OU
"Split the activity's cost over countries; the shares must add up to 100 %."; Cost centre "Type part of the
code to find it."; GL code "Choose a spend type first — each GL code belongs to one or more spend types.";
the others none. Totals: 35 attributes, 110 options. `addSettings` returns the ids by name in a
`seed.SettingsSeed` value (plan "Dependencies"), which `Load` passes to the planning seed functions that
features 004–007 add, so they refer to settings by name.
