# Feature Specification: Product Name SP&A One

**Feature Branch**: `feature/035-product-name-spa-one`

**Created**: 2026-10-28

**Status**: Draft

**Input**: User description: "New requirement from the client: change the app name to SP&A One. Make the
& Shell blue and One Shell yellow." Then: "replace the star logo as well — it does not fit the theme."

**References**: Constitution 4.4 — Principle IX (visual hierarchy, Jost for the product name), Principle X
(palette and yellow's use), F6 · `docs/product/design-system.md` §3.10, §5 (`Logo`), §6 (`PublicLayout`) ·
`docs/product/glossary.md`

## Clarifications

### Session 2026-10-28

- Q: Constitution X forbids yellow text on a light surface (Primary Yellow is 1.45:1 on the page) — how is
  "One" yellow? → A: a darker shade of Shell yellow in the light theme so it stands out: dark Shell gold
  `#AD8700` (Shell yellow's hue; 3.36:1 on Surface, 3.09:1 on Foundation Base). The dark theme uses Primary
  Yellow `#FFC600` (9.2:1 on Surface). Constitution X is amended for the product name only.
- Q: Which Shell blue is the "&"? → A: Shell dark blue `#003C88` in the light theme (10.5:1 on Surface); in
  the dark theme `#6F9BD1` (5.1:1 on Surface), the lighter pair the design system already uses for it.
- Q: The star mark does not fit the theme — what replaces it? → A: no symbol: the coloured name is the
  logo. Where there is no room for the name (the narrow-screen top bar) and as the favicon, a small tile
  shows a Primary Yellow "1" on Shell dark blue (6.7:1), the same in both themes.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - The product is called SP&A One wherever its name appears (Priority: P1)

Priya opens the app. The sign-in page shows the name **SP&A One** and asks her to "Sign in to
SP&A One" (tab "Sign in · SP&A One"). After signing in, the top bar shows the same name (on narrow screens the small "1" tile),
the browser tab reads "Home · SP&A One", and the footer reads "© SP&A One — demo data". Wherever a page,
hint or message names the product ("Everyone who can sign in to SP&A One.", "SP&A One can't reach its
server right now…"), it uses the new name. The old name "MAPP" never appears on screen.

**Why this priority**: this is the client's requirement; a mix of two names would confuse users and look
unfinished.

**Independent Test**: sign in and visit every page in the navigation, open the common dialogs and force the
"can't reach its server" message; read every visible text and the tab title. None says "MAPP"; each that
names the product says "SP&A One".

**Acceptance Scenarios**:

1. **Given** the sign-in page, **When** it is shown, **Then** it shows the name "SP&A One"
   and the heading "Sign in to SP&A One", and the browser tab reads "Sign in · SP&A One" ("SP&A One" while
   the app loads).
2. **Given** any signed-in page titled *X*, **When** it is shown, **Then** the browser tab reads
   "*X* · SP&A One".
3. **Given** the top bar at 960 px or wider, or the menu drawer, **When** it is shown, **Then** the home link
   shows the name "SP&A One" and no star; **given** the narrow-screen top bar, **then** the home link shows the "1" tile
   and is named "SP&A One home" for assistive technology.
4. **Given** the footer, **When** any signed-in page is shown, **Then** it reads "© SP&A One — demo data".
5. **Given** a page description, hint, empty state, lock reason or error message that names the product,
   **When** it is shown, **Then** it says "SP&A One".

---

### User Story 2 - The name is the logo, in Shell's colours (Priority: P1)

The star mark is gone: the name itself is the logo. "SP" and "A" are in the normal text colour, the "&" is
Shell blue and "One" is Shell yellow — a darker Shell gold on light backgrounds so it stands out, Primary
Yellow in the dark theme. Where there is no room for the name, and in the browser tab, a small Shell dark
blue tile with a yellow "1" stands for the product. The colours are decoration only: a screen reader announces the name
once as "SP&A One", and copying it gives the same text.

**Why this priority**: the colours are part of the client's requirement and the product's identity.

**Independent Test**: inspect the name in the top bar and on the sign-in page in both themes and compare the
colour of each part with the agreed values; narrow the window and inspect the top bar's tile and the
favicon; read the name with a screen reader and copy it.

**Acceptance Scenarios**:

1. **Given** the light theme, **When** the name is shown, **Then** "SP" and "A" are Text Strong `#292929`,
   "&" is Shell dark blue `#003C88` and "One" is dark Shell gold `#AD8700`, and no star or other symbol
   stands next to it.
2. **Given** the dark theme, **When** the name is shown, **Then** "SP" and "A" are Text Strong `#F5F5F5`,
   "&" is `#6F9BD1` and "One" is Primary Yellow `#FFC600`.
3. **Given** a window narrower than 960 px, **When** the top bar is shown, **Then** its home link shows a
   tile of Shell dark blue `#003C88` with a Primary Yellow "1", in both themes.
4. **Given** the browser tab, **When** any page is open, **Then** the favicon is the same "1" tile.
5. **Given** a screen reader, **When** it reaches the name, **Then** it reads "SP&A One" once, not letter
   by letter or in pieces.

---

### User Story 3 - Names the server writes use SP&A One (Priority: P2)

Sara downloads the All work grid: the file is "SP&A One All work – Grid – 2026 – 28 Oct 2026.xlsx". The PR
form downloads as "SP&A One PR form – PR-1001 – Original – 28 Oct 2026.xlsx" and says "Routed by SP&A One —
no supporting documents are attached". In the activity log, changes the product made by itself (demo data,
simulated GSAP runs, automatic reminders) are shown as made by "SP&A One". Server messages that name the
product ("SP&A One couldn't read this request…", "USD is SP&A One's base currency…") use the new name.

**Why this priority**: people see these names too, but less often than the screens themselves.

**Independent Test**: download the All work export, the import template and a PR form; open the activity
log after re-seeding; trigger the currency lock and an unreadable request. Every name says "SP&A One".

**Acceptance Scenarios**:

1. **Given** a downloaded export, template or PR form, **When** it is saved, **Then** its file name starts
   with "SP&A One" and any text in it that names the product says "SP&A One".
2. **Given** freshly seeded demo data, **When** the activity log shows a change the product made by itself,
   **Then** it is shown as made by "SP&A One".
3. **Given** a server message or default help text that names the product, **When** it is shown, **Then**
   it says "SP&A One".

### Edge Cases

- **Records written before the change**: a change the product made by itself keeps the name it was written
  with ("MAPP") until the demo data is seeded again; the activity log records what happened and is not
  rewritten.
- **"&" in file names**: allowed on Windows, macOS and in browsers' downloads, so file names keep it.
- **"&" in text for assistive technology and HTML**: it is shown and read as "&" (spoken "and"), never as
  an escape sequence.
- **Narrow screens**: below 960 px the top bar shows only the "1" tile, as it shows only the mark today; the
  home link is still named "SP&A One home". The menu drawer it opens shows the full name.
- **Spacing**: always "SP&A One" — no spaces around "&", one space before "One"; never "SPA One",
  "SP & A One" or "SP&A ONE".

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The product name shown to people MUST be "SP&A One" everywhere the product names itself: the
  wordmark next to the mark, the browser tab title (alone on the sign-in page, "*page* · SP&A One"
  elsewhere), the sign-in heading, the footer, the home link's accessible name, and every page description,
  hint, empty state, lock reason, help text and error message.
- **FR-002**: The text "MAPP" MUST NOT appear on any screen, in any browser tab title or accessible name,
  or in any downloaded file name.
- **FR-003**: In the wordmark, "SP" and "A" MUST be Text Strong, "&" MUST be Shell blue (`#003C88` light,
  `#6F9BD1` dark) and "One" MUST be Shell yellow (`#AD8700` light, `#FFC600` dark), each from a design
  token. These two colours MUST NOT be used for anything but the product name and its tile.
- **FR-004**: The wordmark MUST be exposed to assistive technology and to copy as the single text
  "SP&A One"; the coloured parts MUST NOT change how it is read.
- **FR-005**: The star mark MUST be removed everywhere: the name is the logo. Where the name does not fit
  (the top bar below 960 px) and as the favicon, a tile of Shell dark blue `#003C88` with a Primary Yellow
  "1" MUST stand for the product, the same in both themes; it is decoration, and the link around it is
  named "SP&A One home".
- **FR-006**: Names the server gives people MUST use "SP&A One": downloaded file names (All work export,
  import template, PR form), text inside those files that names the product, the name of the actor for
  changes the product makes by itself, and server messages and default help text that name the product.
- **FR-007**: Internal identifiers MUST NOT change: code and module names, the repository, database names,
  addresses, browser storage keys (people keep their saved theme choice), and operator log messages.
- **FR-008**: The design system and the product documents that state on-screen wording MUST show the new
  name and the wordmark's colours, so new screens follow them.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On every page reachable from the navigation, and on the sign-in page, 0 occurrences of "MAPP"
  are visible in the page or its tab title, in both themes.
- **SC-002**: 100% of the places that name the product (screens, tab titles, accessible names, downloaded
  file names, server messages) say "SP&A One".
- **SC-003**: The accessibility checks report no new violations on the sign-in page and the home page in
  either theme.
- **SC-004**: A screen reader announces the product name as one name, "SP&A One", in 100% of the places it
  appears.

## Assumptions

- "SP&A One" is the full product name; no expansion of "SP&A" is shown.
- Shell yellow is Primary Yellow `#FFC600` (constitution X).
- The project keeps "MAPP" as its internal and documentation codename (constitution, CLAUDE.md, ADRs, spec
  folders); only product documents that state on-screen wording change.
- The API contract's title and descriptions are developer-facing; only the example file names in it change,
  to match FR-006.
- Demo data is seeded again after the change, so the demo shows "SP&A One" for changes the product made.
- No Shell logo or trade dress is introduced (design system §3.10 is unchanged in that respect): the name
  uses blue and yellow, never red with yellow.
- The wordmark is exempt from text contrast (WCAG 2.2 SC 1.4.3, logotypes); the chosen colours still reach
  3:1 or more on the surfaces the name sits on (Surface in both themes).
- Constitution X is amended (4.4 → 4.5.0) to allow the product name's two colours; the amendment and a
  decision-log entry are part of this feature.
