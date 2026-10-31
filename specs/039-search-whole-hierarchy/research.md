# Research: Search the whole hierarchy

## R1 — Where the contents are read

**Decision**: a `Searcher` port on the plans service (`UseSearcher`), implemented in `internal/app` over the
stores' existing batched reads: campaigns of the plans, their initiatives, their activities, their line items.

**Rationale**: four statements whatever the number of plans (B5), no new SQL, no new store method, so no new
fakes or isolation tests; the reads take plan IDs that already come from the actor's scope.

**Alternatives considered**: one SQL query joining the four tables (a cross-area query and a new isolation test for
little gain at this volume); a `Search` function per level in the level registry (each level would re-read the
levels above it to find its plan).

## R2 — How text matches

**Decision**: name or ID contains the trimmed text, ignoring letter case — the rule of both searches today
(004 FR-014) — done in Go over the rows read.

**Rationale**: the same rule everywhere; All work matches in Go too (010).

## R3 — How the tree shows a line item

**Decision**: while a search is typed, the matching line items become rows under their activity (level
`lineItem`, name, ID, leading to `/line-items/{id}`); the tree's own filter keeps them because they match, and the
path opens when the answer arrives.

**Rationale**: the tree shows records where they sit; a separate results list would hide the path.

## R4 — What the Plans list says

**Decision**: `foundIn` on a plan listed only through its contents: the first match top-down (campaign,
initiative, activity, line item, then by name) and how many more. A plan whose own name or ID matches carries none.

**Rationale**: the person sees why a plan with an unrelated name is listed, in words.
