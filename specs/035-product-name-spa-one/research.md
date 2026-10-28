# Research: Product Name SP&A One

## R1 — Constitution amendment 4.5.0

- **Decision**: MINOR amendment, agreed by the product owner on 2026-10-28:
  - **X** gains a "Product name" rule: the name "SP&A One" is the logo. "&" is Shell dark blue (`#003C88`
    light, `#6F9BD1` dark), "One" is Shell yellow (`#AD8700` light — Primary Yellow darkened for light
    surfaces — and Primary Yellow `#FFC600` dark). Where the name does not fit, and as the favicon, a Shell
    dark blue tile carries a Primary Yellow "1". These are the only text in yellow or a yellow shade and
    the only use of these values; they never mean "act here".
  - **VIII** adds 035 to the features exempt from bringing a whole touched file within the comment budget,
    except the files it redesigns (`Logo`, `TopBar`, `Sidebar`, `PublicLayout`, `AppLayout`,
    `usePageTitle`, `SignInPage`, `tokens.css`).
  - **Opening**: MAPP is shown to people as **SP&A One**; "MAPP" stays the project's name in code and
    documents.
- **Rationale**: X forbids yellow text and shades of palette colours, and the client asks for both; an
  amendment is the governed way (Governance → Amendments). WCAG 2.2 SC 1.4.3 exempts logotypes, and the
  chosen values still reach 3:1.
- **Alternatives**: Primary Yellow text (1.58:1 on white, too faint — rejected by the product owner);
  "One" as dark text on a yellow fill (complies, but not yellow letters); a dark band behind the name
  (heavier look).

## R2 — Colours and tokens

| Token | Light | Dark | Use | Contrast (Surface) |
|---|---|---|---|---|
| `--brand-name-amp` | `#003C88` | `#6F9BD1` | "&" | 10.49 · 5.05 |
| `--brand-name-one` | `#AD8700` | `#FFC600` | "One" | 3.36 (3.09 on Foundation Base) · 9.23 |
| `--brand-tile` | `#003C88` | `#003C88` | the "1" tile | — (decoration) |
| `--on-brand-tile` | `#FFC600` | `#FFC600` | the "1" | 6.65 on the tile |

- `#AD8700` keeps Primary Yellow's hue (46.6°) at lightness 34 %, the lightest value of that hue with
  more than 3:1 on Surface. `#6F9BD1` is the dark-theme partner of `#003C88` the design system already
  uses for the Plan level. `--brand-mark` (only the star tile used it) is retired.
- **Alternatives**: gold-orange `#C28100` (close to the warning amber `#C77700`); Primary Yellow — Press
  `#EBB600` (1.87:1); palette Link blue for "&" (teal in the dark theme).

## R3 — How the name is read

- **Decision**: `Logo` renders `<span role="img" aria-label="SP&A One">` around the coloured `<span>`s.
  Assistive technology hears one name; selecting and copying gives "SP&A One". The link around it in the
  sidebar drawer and sign-in header takes its name from it ("SP&A One"); the top bar's link keeps
  `aria-label="SP&A One home"` because below 960 px it holds only the tile.
- **Rationale**: some screen readers (VoiceOver in Safari) split inline elements outside a link into
  separate items; `role="img"` with a label is the established text-logo pattern and passes axe.
- **Alternatives**: plain text in spans (risk of "S P", "and", "A One"); visually hidden duplicate text
  (copies twice).

## R4 — Which texts change

- **Decision**: every string a person reads that names the product: web-app titles, headings, footer,
  descriptions, hints, errors and fixtures; API messages (`unreadableMessage`, `usdLockedMessage`,
  `roleLock`, the `.xls` refusal), default field help (`plan.name`), the PR form's "Routed by …" line,
  and downloaded file names — `SP&A One All work – …`, `SP&A One PR form – …`, the GSAP extract sample
  `spa-one-gsap-extract-sample.csv` (lower case like today's `mapp-gsap-extract-sample.csv`). The web
  app's fallback export name becomes `SP&A One All work.<format>`.
- **Unchanged**: Go module, package and identifier names; `mapp.` browser-storage keys (people keep their
  theme and filters); database names; operator logs (`slog` "MAPP API listening") and the seed command's
  console messages, which developers read; code comments (MAPP stays the codename); the API's `info.title`.
- "&" in file names: legal on Windows, macOS and Linux; the quoted `filename="…"` carries it unchanged,
  and the PR form already adds `filename*` for its "–".

## R5 — The System actor

- **Decision**: `actor.System.Name` becomes "SP&A One"; the web-app fixtures follow. Records store the
  actor's name when written, so a re-seed (`go run ./cmd/seed`) shows the new name everywhere in the demo;
  history written earlier is not rewritten (it records what happened).
- **Alternative**: a migration renaming stored "MAPP" actor names — rejected: rewrites history for a
  cosmetic change, and constitution VI forbids testing migrations on a database anyway.

## R6 — Narrow screens and the tile

- **Decision**: the top bar shows `Logo` at ≥ 960 px and `LogoTile` below (CSS media query in
  `TopBar.module.css`); the menu drawer shows `Logo`; the favicon is the tile as SVG with the "1" as a path
  (no font in a favicon). The component gallery shows both.
- **Rationale**: the design system already shows only the mark below 960 px; the drawer has room for the
  name.

## R7 — Tests

- Unit (Vitest): `Logo` is read as "SP&A One" and draws no symbol; `LogoTile` is hidden from assistive
  technology; `TopBar` link named "SP&A One home"; `PublicLayout` banner names the product; sign-in
  heading; footer; `usePageTitle`. Existing tests change only their expected strings.
- Go: existing handler and service tests change their expected strings and file names; no new Go test
  (no new rule).
- End to end: `frontend/e2e/product-name.spec.ts` — US1 (sign-in heading and tab title, home tab title,
  footer, no "MAPP" on the pages of the navigation) and US2 (computed colours of "&" and "One" in the light
  and dark theme, the tile below 960 px, axe on sign-in and home).

## R8 — Documents

- Design system: §2 top bar, §3.1 brand tokens, §3.10 Logo and brand assets, §5 `Logo` and `LogoTile`,
  §6 `PublicLayout` ("Sign in to SP&A One"), change log 2.x.
- Glossary: a "SP&A One" entry (the product's name; MAPP is the project's codename).
- `demo-users-and-data.md`: changes made by the product itself are shown as made by "SP&A One".
- Decision log D-332; feature map row 035; root `CLAUDE.md` line on the on-screen name.
