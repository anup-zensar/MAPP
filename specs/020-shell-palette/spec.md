# Feature Specification: Shell Palette

**Feature Branch**: `feature/020-shell-palette`

**Created**: 2026-10-26

**Status**: Draft

**Input**: User description: "Make the web app's colours match the Shell palette in constitution
Principle X exactly, in both light and dark themes, and update docs/product/design-system.md to
match. Keep today's token names where they map to a palette role; apply the contrast rules; where
the constitution marks a dark value TODO(DARK_TOKENS), keep today's dark value."

**References**: Constitution 3.1.0 — Principle X (palette, usage and contrast rules), Principle IX,
F6, Governance → Alignment · `docs/product/design-system.md` §3.1, §3.11, §4, §9

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Every screen shows Shell's exact colours (Priority: P1)

Priya opens the Plans list. The page is Shell's light grey, the cards are white, the text is Shell's
dark grey and **Create plan** is Shell's Primary Yellow with dark text; under the pointer and when
pressed it takes Shell's hover and press yellows. She deletes a campaign: the confirm button is
Secondary Red with white text. No colour on screen is a tint, shade or faded copy of a palette
colour.

**Why this priority**: the palette is non-negotiable; it is what makes MAPP part of Shell's estate.

**Independent Test**: inspect the computed colours of the page, a card, body text, secondary text,
a link, a control border, the primary button at rest, hovered and pressed, and a destructive button,
and compare them with the constitution's table.

**Acceptance Scenarios**:

1. **Given** the light theme, **When** any page is shown, **Then** the page is `#F5F5F5`, cards and
   panels `#FFFFFF`, text `#292929` or `#757575`, links `#4B78AA`, control borders `#757575`.
2. **Given** a primary button, **When** it is at rest, hovered and pressed, **Then** its fill is
   `#FFC600`, `#FFCB14` and `#EBB600`, and its label `#292929`.
3. **Given** a destructive confirm button, **When** it is shown and hovered, **Then** it is
   `#DD1D21` with white text; hovering underlines the label instead of changing the red.
4. **Given** a record with the status Declined or missing fields, **When** its chip is shown,
   **Then** the chip is a red fill with white text, not a red tint.
5. **Given** a disabled button, **When** it is shown, **Then** it uses the Disabled colours, not a
   faded copy of its normal colours.

---

### User Story 2 - Text stays readable wherever it sits (Priority: P1)

Sara scans a table: the header band is grey, and the row under her pointer turns Shell's hover grey.
Secondary words, links and red amounts on those greys switch to Text Strong, so nothing drops below
4.5:1. The page header on the grey page writes its breadcrumb, facts and description in Text Strong.
Her keyboard focus is always a Text Strong ring.

**Why this priority**: Shell's subtle grey, link blue and red are only 4.5:1 on white; on any grey
they fail WCAG 2.2 AA unless the rule is applied.

**Independent Test**: run axe colour checks on every page in both themes, with a row hovered and
menus open; check the focus ring colour and offset.

**Acceptance Scenarios**:

1. **Given** the page header on the grey page, **When** it is shown, **Then** breadcrumb links, facts
   and description are Text Strong.
2. **Given** a table row under the pointer or just created, **When** it turns the hover grey,
   **Then** its secondary text, links and red text are Text Strong.
3. **Given** a grey fill (table header, totals row, read-only box), **When** it holds secondary
   text or links, **Then** they are Text Strong.
4. **Given** keyboard focus on any control, **When** it is shown, **Then** the ring is Text Strong
   of the active theme, 2 px away from the control (2 px inside it where a scrolling box would cut
   it off).
5. **Given** the end-to-end tests, **When** axe checks their pages, **Then** it finds no colour
   contrast violation.

---

### User Story 3 - The dark theme uses Shell's dark values (Priority: P2)

Mei works in dark mode. The page is `#1C1C1C`, cards and panels `#292929`, menus and dialogs
`#343434`; text `#F5F5F5` and `#919191`, links `#54B8C8`. In menus and dialogs secondary text is Text
Strong. Yellow keeps its light values with dark text; red appears only as a fill.

**Why this priority**: dark mode is used daily, but the light theme carries most of the work.

**Independent Test**: switch to dark mode and repeat User Story 1's inspection with the dark column;
run axe colour checks with menus and dialogs open.

**Acceptance Scenarios**:

1. **Given** the dark theme, **When** a page, a card, a menu and a dialog are shown, **Then** they
   use Foundation Base, Surface and Raised exactly.
2. **Given** a menu or dialog in the dark theme, **When** it holds secondary text, **Then** that
   text is Text Strong.
3. **Given** a dark value marked TODO(DARK_TOKENS), **When** it is needed, **Then** today's dark value
   is used and listed in the pull request.

---

### User Story 4 - The design system says the same as the code (Priority: P3)

A designer reads `docs/product/design-system.md` and finds the same values, roles and contrast rules
as the constitution and `tokens.css`.

**Why this priority**: the document is binding; a stale value would be copied into new screens.

**Independent Test**: compare §3.1 with `tokens.css` and constitution X line by line.

**Acceptance Scenarios**:

1. **Given** design system §3.1, §3.11 and §4, **When** compared with `tokens.css`, **Then** every
   token, value and rule matches.

### Edge Cases

- A highlighted destructive menu item would be red text on the hover grey: it becomes a red fill
  with white text.
- A problem chip inside a hovered row keeps its own red fill and white text.
- Links in sentences on a grey fill are Text Strong and stay underlined, so they never rely on colour.
- Visited links: the token exists, but in-app links are not coloured as visited (in a single-page
  app almost every link would be).
- Forced-colours mode keeps today's borders.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The brand tokens MUST carry Primary Yellow `#FFC600`, its hover `#FFCB14` and press
  `#EBB600`, and Secondary Red `#DD1D21`, in both themes; text and icons on yellow MUST be `#292929`.
- **FR-002**: Every theme token of constitution X MUST exist with its light and dark value, under
  today's token name where one fills the role: `--page-bg` (Foundation Base), `--surface`,
  `--raised` (new), `--ink` (Text Strong), `--ink-2` and `--muted` (Text Subtle), `--link`,
  `--link-visited`, `--link-hover`, `--link-pressed` (new), `--hairline`, `--row-divider` and
  `--card-border` (Border Subtle), `--control-border` (Border Medium), `--control-border-hover`
  (Border Strong), `--hover` (Hover Background), `--chip` (Press Background), `--disabled-bg`,
  `--disabled-text`, `--disabled-border` (new).
- **FR-003**: No palette colour MAY be tinted, shaded or faded: `--brand-yellow-tint` and
  `--danger-bg-hover` are retired, red is never mixed into a tint, and disabled controls use the
  Disabled tokens instead of opacity.
- **FR-004**: Colours outside the palette MUST be used only for statuses and feedback, planning
  levels, team dots and charts, and MUST NOT resemble Primary Yellow or Secondary Red; the roles
  that used navy or Shell dark blue (focus ring, logo, avatars, banner, tooltips, selection) MUST use
  palette tokens.
- **FR-005**: In the light theme, body-size Text Subtle, links and red text MUST appear only on
  Surface or Raised (red also on Foundation Base); on Foundation Base, Hover Background and Press
  Background they MUST be Text Strong.
- **FR-006**: Menus, popovers, dialogs and toasts MUST be Raised; in the dark theme Text Subtle on
  Raised MUST be Text Strong. In the dark theme red MUST appear only as a fill with white text or
  icon; dark error text keeps today's value until TODO(DARK_TOKENS) is resolved.
- **FR-007**: Link Hover MUST colour only large text and icon-only links; a hovered body-size link
  keeps the Link colour and shows hover by its underline.
- **FR-008**: Focus indicators MUST be Text Strong of the active theme with a 2 px offset (a 2 px
  inset where a scrolling container would clip the ring).
- **FR-009**: Red MUST mark only danger and errors, and yellow only the primary action, the
  current-place bar and the skip link; red controls show hover by underlining, not by a darker red.
- **FR-010**: `docs/product/design-system.md` MUST state the same tokens, values and rules.
- **FR-011**: All existing quality gates and end-to-end tests, axe included, MUST pass.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every token in `tokens.css` that fills a palette role carries the constitution's value
  in both themes (checked line by line in review).
- **SC-002**: axe finds 0 colour-contrast violations on the end-to-end pages (light theme) and on a
  crawl of every page in both themes with a row hovered and menus open.
- **SC-003**: 0 tints, shades or opacity changes of yellow, red or the theme greys remain in
  `frontend/src`.
- **SC-004**: Design system §3.1, §3.11 and §4 list the same values as `tokens.css`.

## Assumptions

- Side panels and the menu drawer are panels (Surface); menus, popovers, dialogs and toasts are
  Raised.
- `--gray-bg` (table headers, fills inside cards) takes Foundation Base; neutral chips, tracks and
  loading shapes take Press Background.
- `--blue` stays as an alias of Link for checked check boxes, radios and switches; progress bars,
  unread dots and chart bars use the information colour.
- The pin star of plans (018) fills with Text Strong instead of yellow, which constitution X reserves
  for "act here" and "you are here".
- Shadows and the dialog scrim stay black at an opacity: they are elevation, not palette roles.
- Status families other than problem keep their colours and tints (outside the palette, allowed).
  Team "gold" becomes a dark gold so it no longer resembles Primary Yellow.
- Dark values marked TODO(DARK_TOKENS) keep today's values: Link Hover `#8CB4E6`, Disabled Text
  `#B0B8C1`, error red `#FF5C5F` (icons, lines) and `#FF7B7E` (text).
- No new tests: axe in the end-to-end tests checks contrast, and no existing test asserted old
  colour values.
- Out of scope: moving files (026), pruning comments (021), backend, API contract and `src/api/`
  (022).
