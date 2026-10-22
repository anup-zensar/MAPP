# Shell UI patterns: research findings and what MAPP should change

| | |
|---|---|
| **For** | Product owner and the Shell client team |
| **Purpose** | Ground the MAPP UI/UX revamp in how Shell's own digital products look and behave |
| **Date of capture** | 20 Oct 2026 (live sites; values can change when Shell redeploys) |
| **Method** | Downloaded and read the HTML, CSS, JavaScript and font files of Shell's public sites and B2B portal; measured computed styles in a headless browser at 1440 × 900 (desktop) and 390 × 844 (phone); keyboard focus walk; colour-contrast maths; compared with MAPP's `docs/product/design-system.md` and `frontend/src/styles/tokens.css` |
| **Evidence images** | [`ux-research-shell-ui-patterns/`](ux-research-shell-ui-patterns/) (screenshots 01–09) |

---

## Summary

1. **Shell runs two design generations.** The current one, which Shell's code calls "Amidala", runs shell.com, shell.co.uk, Shell for Business, Fleet Solutions and the Shell Recharge pages. It is clean and white, with rounded corners, yellow action buttons and dark-grey text. The older one runs the **Shell Fleet Hub** portal for fleet managers. It has square corners, dark-blue buttons and upper-case labels. **MAPP should follow Amidala's look** and borrow Fleet Hub's *structure* (left navigation, page actions, data tables, activity log), because Fleet Hub is the Shell product closest to MAPP.
2. **Typography is simple and strict.** Headings use Shell's own typeface, **"Shell Heavy"**, made by URW Type Foundry in 2018. Licensing is proprietary; the font file's name table says "All rights reserved". All other text is **Arial** at 14 px with a 1.5 line height (12 px for small text). Six core text styles cover almost every word on nine pages. MAPP uses **15 different font sizes**, including half-pixel sizes such as 12.5 and 13.5 px.
3. **Closest free alternative to the Shell typeface: Jost.** It is on Google Fonts under the SIL Open Font License, and side by side it is almost the same shape (image 08). Second choice is Outfit. For body text, MAPP should use **Arial, as Shell does**: no download, figures already the same width for money columns, and it looks exactly like Shell's own pages.
4. **The colours that matter:**
   - **Shell Yellow `#FFC800`** with dark text `#404040`/`#4A4A4A` (5.7–6.7:1) for primary buttons and "you are here" bars.
   - **Shell Red `#DD1D21`** used sparingly.
   - **Shell Dark Blue `#003C88`** for Fleet Hub's primary buttons.
   - **Greys `#404040` → `#F5F5F5`** for everything else.

   MAPP's Apple blue `#0071E3` and its blue-to-violet gradient have no Shell equivalent.
5. **Everything sits on a 4 px grid.** Shell's spacing steps are 4, 8, 12, 16, 20, 24, 32 px and so on. Gaps between grid columns are 20 px, cards have 24 px padding, the page is at most **1300 px** wide, and there are **only two breakpoints (600 and 960 px)**. Buttons and inputs come in fixed heights of **32 / 40 / 48 px**, with 8 px corners. MAPP's misalignment has measurable causes:
   - **seven different control heights** (26–38 px) that sit next to each other in the same toolbars;
   - spacing values off the grid (5, 6, 7, 10, 14, 18, 26 px);
   - **11 media queries at 8 different widths**;
   - search boxes shaped like pills sitting beside inputs with 10 px corners.
6. **Wayfinding is a single Shell signature: a 4 px yellow bar.** It sits under the current main-menu item, above the current audience ("Customer / Business") and under the selected tab. Breadcrumbs (12 px) sit under the header and are repeated in the footer. On phones only the parent page is shown, with a back chevron.
7. **Accessibility: Shell does the basics; MAPP is already ahead in places.**
   - Shell has a skip link on every Amidala page, proper landmarks, ARIA tabs and menus, `aria-current`, and text contrast of 8.9:1.
   - It relies on the browser's default focus ring and has **no reduced-motion handling**.
   - Fleet Hub removes focus outlines and replaces them with a thin dashed line, and its landing page has no `main` landmark.
   - MAPP should keep its own stronger focus ring and reduced-motion rules, and fix one real failure: **input borders `#D2D2D7` are only 1.5:1 against white**. WCAG 1.4.11 asks for 3:1, and Shell's inputs meet it (about 3.1:1).
8. **Recommendation for the app shell** (§9.7):
   - a slim white top bar with Shell's tool icons (search, help, notifications, account);
   - a white left sidebar (240 px, or a 64 px icon rail) with 40 px items and the yellow "you are here" bar;
   - breadcrumb, then a 28 px page title, then at most one yellow primary button;
   - underline tabs;
   - full-width tables on a light-grey `#F5F5F5` page with white cards.
9. **Most of this is a token change.** A drop-in proposal for `tokens.css` is in §9.2. The suggested rollout (§9.10) is: tokens, then one height for every control, then the app shell, then tables and tabs, then pages. **Four client decisions are needed first** (§10):
   - use of the Shell font and Pecten logo;
   - yellow or dark blue as the primary-button colour;
   - whether the page background may be light grey;
   - a decision that replaces the visual part of D-005 (MAPP's version 1 Apple-inspired design language).

---

## 1. What was studied

| # | Product | URL | What it is | How it was read |
|---|---|---|---|---|
| 1 | Shell UK home | https://www.shell.co.uk/ | Consumer site, Amidala platform | HTML, page model JSON, theme CSS, app JS; computed styles; screenshots |
| 2 | Shell global | https://www.shell.com/ | Corporate site, same platform | Same |
| 3 | Shell UK for Business | https://www.shell.co.uk/business.html | B2B landing page | Same (image 01) |
| 4 | Shell Fleet Hub (info page) | https://www.shell.co.uk/business/shell-fleet-solutions/fleet-management-services/shell-fleet-hub.html | Product page for the fleet portal | Same (images 04, 05) |
| 5 | Shell Card | https://www.shell.co.uk/business/shell-fleet-solutions/shell-card.html | Fuel-card product page (the old `shell-fuel-card.html` link redirects here) | Same |
| 6 | Shell Recharge | https://www.shell.co.uk/electric-vehicle-charging.html (where `shellrecharge.com/en-gb` redirects) | EV charging pages; uses a separate Recharge colour theme | Same; theme CSS `theme-shell-recharge.css` |
| 7 | Shell App / Recharge app page | https://www.shell.co.uk/electric-vehicle-charging/shell-recharge-app.html | App marketing page | Same |
| 8 | Site search | https://www.shell.co.uk/search.html?q=fuel%20card | Search form, results, pagination | Same (image 02) |
| 9 | Shell global business customers | https://www.shell.com/business-customers.html | B2B hub | Same |
| 10 | **Shell Fleet Hub portal** | https://fleethub.shell.com/ | **Logged-in B2B portal for fleet managers** (only its public landing page can be seen) | HTML, 1 MB component CSS (`5997.css`), font files, computed styles (image 06) |
| 11 | Shell Fleet App | https://play.google.com/store/apps/details?id=com.shell.sitibv.shellfleetapp | Mobile fleet-management app (Google Play listing and screenshots) | Listing text and screenshots (image 07) |
| 12 | Shell Fleet App product page | https://www.shell.com/business-customers/shell-fleet-solutions/digital-and-payment-solutions/digital-solutions/shell-fleet-app.html | Feature list | Page model JSON |

Code assets that the findings quote (hashes valid on 20 Oct 2026):

- Amidala app script: `https://www.shell.co.uk/etc.clientlibs/amidala/clientlibs/app.min.ACSHASHc4fb4841e466a60a7ed3479e9efcaf79.js`. It holds all design tokens, the utility classes and every component.
- Amidala theme CSS: `.../theme-base.ACSHASHb291465fc8d3c7bcefde74d34044b698.css` and `.../theme-shell-recharge.ACSHASH23a752d535925afba7aeea6dc8e3b8f7.css`.
- Heading font file: `.../theme-base/resources/fonts/shell-typeface-la-heavy-221208.woff2`.
- Fleet Hub CSS: `https://sfh-neu-mfe-marketing-cdnendpoint-a6e0dzcscqdac7ha.a02.azurefd.net/5997.css`.

**Limits of this research**

- Logged-in screens (Fleet Hub after sign-in, Shell Card Online) could not be seen. Fleet Hub's *component stylesheet* still lists its in-app components: activity logs, notifications panel, page actions, dynamic and master-detail tables, editable sections, date pickers, CSV export, user management and card-ordering steps. These rules give real sizes and colours.
- `www.shellenergy.co.uk` did not resolve from our network. `www.shellenergy.com` now redirects to https://www.shell.us/business/shell-energy.html, which is on the same Amidala platform.
- `lubematch.shell.com` was blocked by the corporate proxy (certificate error 515).
- The Apple App Store pages for GB returned "page not found". A search result shows the stand-alone Shell Recharge iOS app has been retired and its features moved into the Shell App ([App Store listing](https://apps.apple.com/be/app/shell-recharge/id549121633)).
- No public Shell design-system site or package was found (searches and the npm registry). If Shell has an internal design system for staff tools, the client should share it (§10).

---

## 2. Brand colours as Shell actually implements them

Amidala sets its colours as HSL custom properties in the app script and theme CSS. The hex values below are converted from those.

| Shell token | HSL in code | Hex | Used for | Contrast on white |
|---|---|---|---|---|
| `--color-brand1-bg` (Shell Yellow) | 47, 100%, 50% | `#FFC800` | Primary buttons, skip link, "you are here" bars, Sign in button | 1.55:1 (a fill, never text) |
| `--color-brand1-text` | = ink1 | `#4A4A4A` | Text on yellow | 5.7:1 on yellow |
| `--color-brand2-bg` (Shell Red) | 359, 77%, 49% | `#DD1D20` | Accents (sparingly) | 4.9:1 |
| `--color-brand3-bg` | 212, 49%, 39% | `#336094` | Secondary blue sections | 6.5:1 |
| `--color-brand4-bg` | 159, 100%, 26% | `#008556` | Green sections | about 4.8:1 |
| `--color-brand5-bg` | 306, 61%, 33% | `#87217D` | Purple sections | high |
| `--color-system-bg` | 210, 78%, 16% | `#092949` | System messages (navy, white text) | 15:1 |
| `--color-ink1` | 0, 0%, 29% | `#4A4A4A` | **All text: headings, body and links** | 8.9:1 |
| `--color-paper1` / `paper2` | 100% / 96% | `#FFFFFF` / `#F5F5F5` | Page / cards and alternate sections | — |
| Dark mode: paper1 / paper2 / ink1 / yellow | 212,18%,14% / 213,22%,10% / 0,0%,88% / 45,100%,42% | `#1D232A` / `#14191F` / `#E0E0E0` / `#D6A100` | Shell's own `prefers-color-scheme: dark` values | ink on paper1 12:1 |

The Recharge theme overrides only three colours: a teal `#008192` and two pale teals (`theme-shell-recharge.css`). **Sub-brands are a palette swap, not a different design.**

Fleet Hub (older generation) uses the classic Shell web palette. These are the most frequent colours in `5997.css`:

| Colour | Hex | Count in CSS | Role in Fleet Hub | On white |
|---|---|---|---|---|
| Shell Dark Grey | `#404040` | 205 | Body text | 10.4:1 |
| Shell Yellow | `#FBCE07` | 106 | Brand blocks, icon buttons | 1.5:1 (fill only) |
| Shell Red | `#DD1D21` | 118 | Errors, accents | 4.9:1 |
| Shell Dark Blue | `#003C88` | 132 | **Primary buttons** (white text 10.5:1) | 10.5:1 |
| Teal | `#2A7A8F` / `#009EB4` | 191 / 60 | Link hover, focus outline | 4.9:1 / 3.2:1 |
| Light blue | `#89CFDC` | 137 | Focus on dark blocks | 1.75:1 |
| Dark green / light green | `#008443` / `#BED50F` | 31 / 39 | Status, charts | 4.8:1 / 1.7:1 |
| Purple / orange | `#641964` / `#EB8705` | 15 / 14 | Charts, warnings | 11.3:1 / 2.6:1 |
| Greys | `#595959`, `#717171`, `#A6A6A6`, `#D9D9D9`, `#F7F7F7` | many | Secondary text, placeholders, dividers, modal background | 7.0, 4.9, 2.4, 1.4, — |

What this means for MAPP:

- Yellow is a **fill colour with dark text**. It is never text, and never a focus ring on white, because it is 1.5:1.
- Orange `#EB8705`, light green and light blue fail as text colours. They can only be fills or decoration.
- Text hierarchy on Shell sites comes from **typeface and size**, not colour: headings and body text share the same grey.

---

## 3. Typography

### 3.1 Typefaces

| Where | Family | Weights in use | Evidence |
|---|---|---|---|
| Amidala headings | **Shell Heavy**: `font-family: 'custom-font'` loaded from `shell-typeface-la-heavy-221208.woff2` | One file, used as bold (700) | Font name table: *"Copyright 2018 by URW Type Foundry GmbH. All rights reserved"*, family **ShellHeavy**, version 1.007 |
| Amidala body, buttons, inputs | **Arial** (`--font-default: Arial, Roboto, Helvetica, sans-serif`) | 400, 700 | App script `:root` block |
| Fleet Hub headings and buttons | **Shell Bold** (`'Shell-Bold', 'Arial', sans-serif`) | 700 | `5997.css` (418 uses) |
| Fleet Hub other text | **Shell Light / Book / Medium**, plus italics, then Arial | 300, 400, 500 | `@font-face` rules in `5997.css`; Book file: family "Shell", style "Book", © 2018 URW |
| Fleet Hub body | **Arial** 16 px, `#404040` | 400 | Computed style |

So the Shell typeface family is **Shell Light, Book, Medium, Bold and Heavy**, made for Shell by URW. Licensing is proprietary. MAPP **must not copy these files from Shell's servers**. If Shell's brand team supplies them for internal tools (Fleet Hub already uses them), MAPP can switch in one token (§10, question 1).

Language-aware sizing: Amidala multiplies every font size by `--font-scale: 1.15` for Arabic, Japanese, Korean, Thai and Chinese (`:root:lang(ar|ja|ko|th|zh)` in the theme CSS).

### 3.2 Type scale measured on nine Amidala pages

| Style | Family | Size / line height | Weight | Letter spacing | Colour | Where |
|---|---|---|---|---|---|---|
| H1 (page title) | Shell Heavy | **32 / 48 px** | 700 | normal | `#4A4A4A` | Every page |
| H2 (section) | Shell Heavy | **28 / 42 px** | 700 | normal | `#4A4A4A` | Section titles |
| H3 (card title) | Shell Heavy | **20 / 30 px** | 700 | normal | `#4A4A4A` | Cards, results, footer columns |
| H4 (accordion) | Shell Heavy | 18 / 27 px | 700 | normal | `#4A4A4A` | Accordion summaries |
| Body | Arial | **14 / 21 px** | 400 | normal | `#4A4A4A` | All running text, buttons, nav |
| Strong / button | Arial | 14 / 21 px | 700 | normal | `#4A4A4A` | Buttons, CTA links |
| Small | Arial | **12 / 18 px** | 400 | normal | `#4A4A4A` | Breadcrumbs, footer links, legal |
| Lead (rare) | Arial | 20 / 30 px | 400 | normal | | Intro paragraph on product pages |

The scale as defined in the code (app script utility table): `text-xs 11, sm 12, base 14, lg 16, xl 18, 2xl 20, 3xl 28, 4xl 32, 5xl 36, 6xl 40, 7xl 44, 8xl 60, 9xl 68` px, all with line height 1.5. The heading component maps h1→32, h2→28, h3→20, h4→18, h5→16 and h6→14, always in Shell Heavy, bold, in the heading colour.

On every page measured, **six styles account for nearly all text**: Arial 12/400, 14/400 and 14/700, and Shell Heavy 20, 28 and 32. There is no negative letter spacing, no upper-case eyebrow labels and no light weights.

Fleet Hub's older scale is in `rem` on a 16 px root: h1 2.66 rem (42.6 px), h2 2.0 rem (32 px), buttons 0.75 rem (12 px) **upper case**, table headers 0.67 rem (10.7 px) upper case. These small upper-case labels are part of what makes Fleet Hub look dated. **Do not copy them.**

### 3.3 Closest open-licence alternatives

Each candidate was measured against Shell Book: x-height to cap-height ratio and average lowercase width. All candidates were then rendered beside Shell Heavy and Shell Book (image 08: [`08-font-comparison.png`](ux-research-shell-ui-patterns/08-font-comparison.png)). All candidates are from the Google Fonts repository, `ofl/` folder, so they are under the SIL Open Font License 1.1 and free to self-host in a commercial product.

| Candidate | x / cap ratio (Shell Book 0.62, Shell Heavy 0.65) | Avg lowercase width (Shell Book 0.48) | Visual match | Verdict |
|---|---|---|---|---|
| **Jost** ([Google Fonts](https://fonts.google.com/specimen/Jost), [source](https://github.com/google/fonts/tree/main/ofl/jost)) | 0.66 | 0.47 | Same geometric skeleton as Shell: round `o e c`, straight-legged `y`, similar `G`, `a`, `g`; Jost 600–700 is almost indistinguishable from Shell Heavy at heading sizes | **Use for headings** (600 and 700) |
| Outfit ([Google Fonts](https://fonts.google.com/specimen/Outfit)) | 0.68 | 0.50 | Geometric and close, slightly rounder and tighter | Second choice |
| League Spartan | 0.62 | 0.46 | Metrics close, but the shapes are heavier and more "display" | Not for UI |
| Figtree, Albert Sans, Urbanist | 0.71 | 0.51 | Friendly geometric, but the x-height is larger than Shell's | Acceptable, less Shell-like |
| Inter, DM Sans, Manrope | 0.75 | 0.53–0.54 | Neutral UI fonts, not Shell-like | Only if the team wants a UI-optimised body font |
| **Arial** (system) | — | — | **What Shell uses for body text on both platforms**; digits are the same width by default, so money columns line up | **Use for body/UI text** |

Recommended pairing: **Jost** (self-hosted woff2, weights 600 and 700, Latin subset; available on npm as `@fontsource/jost`) for headings and big numbers, and **Arial** for everything else. This is the same pairing Shell uses (Shell Heavy plus Arial).

Jost is not recommended for body text. Its small x-height (0.46) makes 12–14 px table text harder to read than Arial during a working day.

---

## 4. Layout structure

| Aspect | Amidala (current Shell) | Fleet Hub (older Shell B2B) | Evidence |
|---|---|---|---|
| Page width | `--page-width: 1300px`; header nav `max-w-page`; `--nav-width: 1400px` | Grid max-width 912 px (≥1008), **1200 px (≥1296)**, 1488 px (≥1824) | App script `:root`; `.o-grid` in `5997.css` |
| Side gutters | 24 px (`px-6`); 1300 px content, so 1252 px inside | 12 px on phones, 24 px from 624 px | Measured breadcrumb width 1252 px |
| Grid | 12 columns (`lg:grid-cols-12`), column gap **20 px** (`gap-x-5`) | Flex grid | PageHeader, PromoFull and CallToAction components |
| Spacing unit | **4 px** (`lr=4`); steps 0–12, 14, 16, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56, 60, 64, 72, 80, 96, 128 (× 4 px) | Mostly 4/8/12/16/24/48 px | App script spacing table |
| Card padding and inner rhythm | `Box` = **24 px padding, 20 px between children** (`p-6 space-y-5`) | 24 px panel padding | `function O` (Box) |
| Gap between cards | 20 px | 16–24 px | |
| Breakpoints | **sm < 600, md 600–959, lg ≥ 960** (only these) | 624, 1008, 1296, 1824 | `me=600, Be=960` in the app script |
| Radius scale | none 0, sm 3, **default 4**, md 6, **lg 8**, xl 12, **2xl 16**, full 50% | Mostly 0, 3, 4 px (square look) | `ot=[…]` in the app script |
| Alignment | Everything left-aligned to the same 1300 px column: logo, nav, breadcrumb, title, cards and footer share one left edge (images 01, 02, 05) | Same | Screenshots |
| Section rhythm | Full-width bands alternate white and `#F5F5F5`; white cards sit on grey bands and grey cards on white bands | Yellow hero band, then white | Images 01, 04, 05 |
| Shadows | Cards: none at rest; on hover the card lifts 8 px with shadow `0 8px 12px 2px rgba(0,0,0,.2)`. Menus `0 2px 8px rgba(0,0,0,.2)`. Dialogs `0 2px 16px rgba(0,0,0,.2)` | Panels `0 2px 4px rgba(0,0,0,.13)`; modal `0 2px 4px rgba(0,0,0,.13), 0 5px 10px rgba(0,0,0,.2)` | `Card` (He), `shadow` utilities |
| Motion | `--anim-time: 250ms`; expand (fade + move 8 px), toast (slide up 48 px), fade | 300 ms transitions | App script keyframes |

---

## 5. Component patterns

All Amidala specs below come from the component source in the app script, confirmed by computed styles. Fleet Hub specs come from `5997.css`.

| Component | Shell spec (Amidala unless noted) | Notes for MAPP |
|---|---|---|
| **Primary button** | Fill `#FFC800`, text `#4A4A4A` **bold 14 px**, 1 px border in the fill colour, **radius 8 px** (6 px at the smallest size), padding 0 12 px, gap 8 px, icon at left. **Heights: xs 32, sm 40, md 48 (default), lg 64 px.** Hover **inverts** (dark fill, yellow text). Disabled: greyscale and not clickable. Icon-only: square, with `aria-label` and `title`. Measured examples: "Sign in" 105 × 32, "Find out more" 119 × 48, chat 89 × 40 | One fixed set of heights; one radius |
| **Dark CTA button** | Fill `#4A4A4A`, text `#FFC800` bold, radius 8, 48 px ("Explore Shell Fleet Hub", "Job search"); contrast 5.7:1 | A secondary emphasis Shell uses inside yellow bands |
| **Plain button** | Transparent, ink text; hover inverts (ink fill, white text) | Tertiary or icon buttons |
| Fleet Hub buttons | Dark blue `#003C88` fill, white Shell Bold **12 px upper case**, **square corners**, 47–48 px; outlined: 1 px `#404040` | Shows dark blue is also a Shell primary colour |
| **Links** | Inside text: **ink colour, underlined**. Stand-alone CTA links: bold, underline on hover. Tags: 12 px, ink at 10% background, radius 3, hover inverts. External links get an icon | Shell does not use blue links on its public site |
| **Cards** | Link cards: `#F5F5F5` on white (or white on grey bands), **radius 16**, no border, no shadow at rest, image at top, 24 px padding, H3 20 px, "Read more →" at bottom right; the whole card is the link; hover lift 8 px + shadow | MAPP summary and plan cards can follow this |
| **Text input** | **48 px** high, 12 px padding, **radius 8**, 1 px border **ink at 60%** (about `#929292`, **3.1:1**), white background; disabled: ink at 10% background; text **16 px on phones and tablets** (stops iOS zooming) and 14 px on desktop; native checkbox and radio | Fixes MAPP's 1.5:1 borders |
| Textarea | Radius 8, border ink at 20%, 16 px padding, 144 px high, resizes vertically only | |
| Fleet Hub inputs | Underline style; on hover a teal underline; focus a 1 px dashed teal outline; boxed select 56 px with a 1 px `#A6A6A6` border | Older style, do not copy |
| **Search** | `role="search"` form; 48 px search input with placeholder; square 48 px yellow submit button with icon (`aria-label "Submit"`); then the text "53 results found"; results show H3 link (Shell Heavy 20 px, underlined), breadcrumb path and snippet with the matched words in bold (image 02) | MAPP's search boxes should say what they search |
| **Tables** | Wrapper scrolls sideways; 1 px ink-at-20% grid lines; **header bold on ink at 5%**; cell padding 8 px; row hover ink at 5%; outer corners 8 px; left-aligned, top-aligned | Fleet Hub: header Shell Bold 10.7 px upper case; **under 624 px rows become stacked cards** with shadow and 16 px gaps; master-detail and dynamic tables |
| **Tabs** | `role="tablist"` / `tab` / `tabpanel`, `aria-selected`, `aria-controls`; tab height **48 px**, 14 px text; selected tab = **4 px yellow bottom bar**; container has a 1 px bottom border (ink at 20%); when tabs overflow they scroll sideways with fade-out arrows; the compact variant becomes a dropdown on small screens; tab state is kept in the URL hash | Replace MAPP's pill tabs for page sections |
| **Main navigation bar** | 64 px high, Pecten logo 64 px box, items 14 px; selected item = **4 px yellow bottom border**; items with children open a dropdown (288 px wide, radius 8, shadow, expand animation) whose first item is "Go to: *Section*"; Escape closes; `aria-expanded`, `aria-controls`, `aria-current="page"`; tool icons on the right (station finder, help, search) as 48 px icon buttons | |
| **Utility (top) bar** | Grey `#F5F5F5` strip with "Customer / Business / About us" (audience switch, 12–14 px, selected = 4 px yellow **top** bar) and a yellow **Sign in** button (32 px) at the right | Same idea as a module switcher |
| **Vertical menus** | Selection = **4 px yellow bar on the end edge**; items 32 px (md) or 48 px (lg); hover ink at 10%; radius 4; nested levels indent by 16 px | Gives the sidebar pattern |
| **Breadcrumb** | 12 px, chevron separators, last item `aria-current="page"`, schema.org BreadcrumbList; hidden when there is only Home; **on phones and tablets only the parent is shown, with a back chevron**; repeated in large size in the footer | |
| **Modal (dialog)** | Native `<dialog>` with `showModal()`; centred: **512 px wide, radius 12, padding 24**, `shadow-lg`; backdrop black at 50% **with 4 px blur**; max size: screen minus 12 px; Escape closes; layouts "center", "full" (gallery) and "toast" (bottom sheet sliding up 48 px) | Fleet Hub modal: 888 px wide, `#F7F7F7` background, 24 px padding, max 80% of screen height |
| **Notification / toast** | Same `<dialog>` in toast layout at the bottom; title H3 + text + links; dismiss icon button; shown once per message, suppressed for 100 days, remembered only if functional cookies are allowed | |
| **System message** | `Message`: full-width, 24 × 16 px padding, 12 px text, **navy `#092949` with white text** | Use for MAPP's environment banner |
| **Accordion** | Native `<details>/<summary>`; radius 16; 1 px border ink at 20%; summary padding 24 × 12 px; Shell Heavy 18 px title; chevron rotates; content fades in | |
| Chips / badges / tags | Shell has no status-chip component; tags are 12 px rectangles (radius 3) on ink at 10% | MAPP's status chips stay, reshaped (§9.6) |
| **Pagination** | Centred; current page bold; window of ±4 pages; "Previous" / "Next" text hidden on small screens; disabled ends shown at 50% opacity | |
| **Back to top** | Appears after scrolling; fixed 24 px from the bottom corner; 40 px, radius 8, shadow; `aria-label` and `title` | |
| **Skip link** | First focusable element; yellow, bold, padding 16 px, radius 8; slides in at the top left when focused (image 09) | MAPP already has one |
| Cookie banner | Bottom panel; "Manage cookies" link plus two **equal** yellow buttons, "Reject optional cookies" and "Accept optional cookies" | Not needed in MAPP |
| Dates | `toLocaleDateString(lang, {day, month:'short', year})`, which gives "21 Jul 2026" in en-GB ("Page last updated: 21 Jul 2026") | **Already matches MAPP's D-216 format** |

---

## 6. Navigation and user-flow patterns

### 6.1 Amidala (public sites)

1. **Header, top to bottom** (images 01, 02):
   - a grey utility bar (audience switch plus Sign in);
   - a white 64 px main bar: Pecten, up to 5–6 sections with dropdowns, tool icons on the right;
   - a 1 px divider;
   - the breadcrumb;
   - the page.

   Everything aligns to the same 1300 px column.
2. **"You are here" is always shown the same way**: a 4 px yellow bar on the selected audience, the selected section and the selected tab, plus `aria-current`.
3. **Dropdown menus keep the parent page reachable**: the first entry is "Go to: Electric vehicle charging".
4. **Phone** (image 03):
   - the utility bar stays;
   - the main bar becomes logo, tool icons and a burger button (`aria-expanded`, `aria-haspopup`; Escape closes and returns focus);
   - the menu opens as a panel under the header, not a side drawer;
   - no page scrolled sideways at 390 px on any page tested.
5. **Pages follow a predictable order**: breadcrumb, title (32 px) and intro, then content bands, then "How can we help?", then "You might also be interested in" (three cards), then the footer.
6. **Footer** (image 05):
   - the breadcrumb again;
   - "Page last updated: 21 Jul 2026";
   - four link columns with 20 px Shell Heavy headings and 12 px links;
   - social links;
   - a legal row with Accessibility, Terms, Privacy and Cookie policy.
7. **Search** is reached from the magnifier icon in every header. The results page keeps the query in the box and shows the count, and each result shows where it lives (breadcrumb).
8. **Help** is always one click away: a "?" icon in the header, a floating "Contact Us" tab on B2B pages and a chat button.

### 6.2 Fleet Hub (the Shell product closest to MAPP)

- **Sign in first.** The public landing page is a yellow hero with one dark-blue full-width "Login in to Shell Fleet Hub" button and two outlined secondary actions (image 06).
- The **in-app navigation is a vertical primary nav**. Each item is 48 px high, with an icon at the left, `#595959` text and 1 px `#D9D9D9` dividers (`.c-primary-nav__item`, `.c-primary-nav__link`). There is a separate notifications panel.
- **In-app components show its workflow**: dashboard widgets (cards, invoices), **page actions** (stacked on phones, in a row from 1008 px), editable sections, tabular data panels (white, radius 4, light shadow, 24 px header), dynamic and master-detail tables, **activity logs**, confirm modals, multi-step "order cards" flows, CSV export options and user management. These are the same building blocks MAPP needs.
- The marketing image of the logged-in dashboard shows a **greeting ("Good morning Victor")**, then cards for *Card activity*, *Cards*, *Invoices* and *Vehicles*, each with counts and a "See all" link. This is the same idea as MAPP's Home summary cards.

### 6.3 Shell Fleet App (mobile, image 07)

- White screens with Shell Red marketing headlines.
- A **full-width yellow primary button** ("Login", "View Spends") and an outlined secondary button ("Create Account").
- Underline tabs (Statements / Invoices / Cards), a bottom tab bar with four icons, and ring charts in red and yellow for budget used.
- The listing has a 2.3★ rating from 1.27k reviews. Shell's mobile fleet experience is not a usability benchmark.

---

## 7. Accessibility and responsiveness

| Check | Amidala (9 pages) | Fleet Hub landing | MAPP today |
|---|---|---|---|
| Skip link | Yes, "Skip to main content", on every page | **None** | Yes (`global.css:75-90`) |
| Landmarks | `header`, `nav`, `main`, `footer` on every page; `role="search"` on the search page | **No `main`, no `nav`** | Yes |
| One H1 per page | Yes | Yes | Yes |
| Images without `alt` | 1 of 11 on the shell.co.uk home page, 0 elsewhere | 0 | — |
| Unnamed icon buttons | 0 | **2** | Icon buttons have names (design system §9) |
| Body text contrast | `#4A4A4A` on white **8.9:1**; on `#F5F5F5` 8.1:1 | `#404040` 10.4:1 | `--ink` 16:1, `--ink-2` 5.1:1 |
| Button text contrast | Ink on yellow 5.7:1 | White on dark blue 10.5:1 | White on `#0071E3` 4.7:1 |
| **Input border (WCAG 1.4.11, 3:1)** | Ink at 60%, **about 3.1:1, passes** | `#A6A6A6`, 2.4:1, fails | **`#D2D2D7`, 1.5:1, fails** (`TextField.module.css:37`, `inputs.module.css:9`, `Combobox.module.css:10`, `SearchField.module.css:22`) |
| Focus indicator | **Browser default** (no custom `:focus-visible` rules; the dialog removes its outline) | **Outlines removed** (`:focus { outline: none }`), replaced by a 1 px dashed teal line, which is weak | **2 px blue ring with 2 px offset** (`global.css:56-59`), better than Shell |
| Reduced motion | **Not handled** (no `prefers-reduced-motion` in the app script; cards lift on hover) | Not handled | **Handled** (`global.css:98-106`) |
| ARIA patterns | Tabs (tablist, tab, tabpanel, aria-selected, aria-controls); burger and dropdowns (aria-expanded, aria-controls, aria-haspopup, Escape returns focus); `aria-current` on breadcrumb and nav; `aria-label` + `title` on icon buttons | Partial | Present |
| Right-to-left and languages | Logical properties throughout (`start`/`end`, `ps`/`pe`), chevrons flip in RTL; 1.15× text for CJK, Arabic and Thai | — | English only (fine for now) |
| Responsive | 2 breakpoints; nothing scrolls sideways at 390 px; touch text 16 px; burger menu | 4 breakpoints; tables become stacked cards on phones | 11 media queries at 8 widths; tables scroll inside cards |

**What to copy:** landmarks, skip link, ARIA tab and menu patterns, "Go to: parent" in menus, 16 px input text on touch devices, 3:1 control borders, a breadcrumb that collapses to the parent on phones, and stacked-card tables on phones for inbox-style lists.

**Where MAPP should stay better than Shell:** a visible custom focus ring, reduced-motion support, an error summary with links, and status never shown by colour alone.

---

## 8. MAPP today compared with Shell

| Aspect | MAPP now | Shell | Gap |
|---|---|---|---|
| Design reference | MAPP's version 1 Apple-inspired design language (`design-system.md:3-6`, decision D-005) | Shell Amidala | **Visual language changes**; the CRUD rules in design system §6–§8 stay |
| Typeface | System stack `-apple-system, BlinkMacSystemFont, "Segoe UI", Inter…` (`tokens.css:101`), which shows as Segoe UI on Shell's Windows laptops | Shell Heavy headings + Arial text | Change |
| Font sizes in use | **15 distinct sizes** across 155 CSS files: 10.5, 11, 11.5, 12, 12.5, 13, 13.5, 14, 15, 16, 17, 22, 26, 27, 28 px (e.g. `Sidebar.module.css:107` 10.5 px, `SummaryCard.module.css:20` 27 px, `ConfirmDialog.module.css:29` 17 px) | 6 core styles (12, 14, 20, 28, 32 + bold) | **Main cause of the "unpolished" feel** |
| Weights | 600 used 162 times; Arial and Jost-at-text-size have no real 600 | 400 / 700 | Change to 400 / 700 |
| Letter spacing | −0.01 em on all body text (`global.css:19`), −0.02 em on titles, +0.04 em upper-case eyebrow | Normal everywhere | Remove |
| Colours | Apple greys `#1D1D1F` / `#6E6E73`; Apple blue `#0071E3` for buttons, focus and eyebrows; blue-to-violet gradient for logo and avatars (`tokens.css:19-41`) | Shell greys `#404040` / `#4A4A4A`, yellow `#FFC800`, red, dark blue `#003C88`; no gradients | Change |
| Control heights | **Seven heights in one product**: 26 (`Combobox.module.css:35`), 28 (`FilterBar.module.css:24`), 30 (`Button.module.css:47`, `SegmentedFilter.module.css:11`), 32 (`PillTabs.module.css:21`), 34 (`ExportMenu.module.css:16`), 36 (`Button.module.css:41`, `SearchField.module.css:20`), 38 (`TextField.module.css:35`) | 32 / 40 / 48 / 64 | **Second cause of misalignment**: a 36 px button next to a 38 px field next to a 30 px filter |
| Corner shapes | Cards 16, controls 10, chips 8, pills 999, plus 3, 4, 6, 8, 9, 12, 20 px used ad hoc; **search box is a pill** while inputs have 10 px corners | 4 / 8 / 12 / 16 | Consolidate |
| Spacing | Design system says "4 px grid" (`design-system.md:91`), but the code uses off-grid values: 6 px (128×), 10 px (91×), 14 px (30×), 18 px (17×), 5, 7, 26 px | 4 px steps | Snap to the grid |
| Breakpoints | 11 media queries at 480, 560, 639/640, 720, 767/768, 1024, 1280 px | 600 / 960 | Consolidate to 3 |
| Page width | 1120 px max, 20 px padding (`AppLayout.module.css:44-49`) | 1300 px, 24 px padding | Widen; tables benefit most |
| Page header | Blue upper-case eyebrow, 22/26 px title with a grey "highlight" phrase, 13.5 px subtitle (`PageHeader.module.css:15-43`) | Breadcrumb, then 32 px Shell Heavy title, then 14–20 px intro | Change |
| Sidebar | Grey `#F5F5F7`, 244 px, 13.5 px items, 12 px-radius pills, hover nudges 2 px right, blue gradient for the selected item with a 3 px blue bar (`Sidebar.module.css:43-101`) | White surfaces; selection = 4 px yellow bar; items 32/48 px | Restyle |
| Tabs and filters | Pill-shaped toggles (`PillTabs`, `SegmentedFilter`) | Underline tabs with a 4 px yellow bar | Change |
| Cards | White, faint border, small shadow, 20 px padding, 15 px title (`Card.module.css:1-45`) | No shadow, 24 px padding, 20 px title, grey and white contrast | Change |
| Input borders | `#D2D2D7` (1.5:1) | 3:1 | **Accessibility fix** |
| Focus ring | 2 px `#0071E3`, 2 px offset | Browser default | Keep the pattern, recolour |
| Environment banner | Pale yellow `#FFF4CE` (`tokens.css:58`) | — | Would clash with brand yellow; use Shell's navy system message |
| Dark mode | Yes (Apple-like values) | Yes (Shell values) | Swap to Shell's dark values |
| Date format | `DD MMM YYYY` | "21 Jul 2026" | **Already aligned** |
| Status chips | Word + colour on a 12% tint, radius 8 | (no equivalent) | Keep the idea; radius 4; Shell hues |

---

## 9. Recommendations

### 9.1 Principles

- **Keep** everything in design system §1 and §6–§10: one purpose per screen, the standard CRUD pattern, plain words, never greyed out without a reason, the error summary, Undo and the recycle bin. These are about behaviour and remain correct.
- **Replace** the visual language (§2–§5 of the design system) with Shell's Amidala language. Record this as a new decision that supersedes the *visual* part of D-005. The *functional* part stays: Hive9 screens are still not copied.
- **One of everything:**
  - one ink colour;
  - one accent (yellow) for "act here" and "you are here";
  - one link and focus colour (Shell Dark Blue);
  - one height per control size;
  - one corner radius per object type;
  - one spacing scale.

### 9.2 Proposed tokens (drop-in values for `frontend/src/styles/tokens.css`)

Existing token names are kept where possible, so components change only where noted. New tokens are marked `/* new */`.

```css
:root {
  color-scheme: light;

  /* Surfaces: grey canvas, white cards (Shell paper1 / paper2) */
  --page-bg: #f5f5f5;            /* was #ffffff: app canvas behind cards (see 10, question 3) */
  --surface: #ffffff;            /* cards, dialogs, menus, top bar, sidebar */
  --gray-bg: #f5f5f5;            /* table header, subtle fills, hovered rows */

  /* Text: Shell greys */
  --ink: #404040;                /* Shell Dark Grey, 10.4:1 on white */
  --ink-2: #595959;              /* Shell Mid Grey, 7.0:1 on white, 6.4:1 on #f5f5f5 */
  --muted: #757575;              /* placeholders and icons only, 4.6:1 */

  /* Lines */
  --control-border: #8c8c8c;     /* new: inputs, selects, checkboxes, 3.4:1 (WCAG 1.4.11) */
  --hairline: #d9d9d9;           /* Shell Pale Grey: dividers, table outer border */
  --row-divider: #e8e8e8;        /* new: lines between table rows */
  --card-border: transparent;    /* cards are separated by the grey canvas, not borders */
  --chip: rgba(64, 64, 64, 0.08);
  --hover: rgba(64, 64, 64, 0.08);   /* Shell hover is ink at 10% */

  /* Brand and interaction */
  --accent: #ffc800;             /* new: Shell Yellow (shell.com value); fills only, never text */
  --on-accent-dark: #404040;     /* new: text on yellow, 6.7:1 */
  --blue: #003c88;               /* Shell Dark Blue: focus ring, selection, checked controls */
  --link: #003c88;               /* 10.5:1 */
  --button-primary-bg: #ffc800;  /* was #0071e3 */
  --button-primary-text: #404040;/* new: Button.module.css .primary uses this instead of --on-accent */
  --button-primary-bg-hover: #404040;  /* Shell hover inverts: dark fill... */
  --button-primary-text-hover: #ffc800;/* new: ...yellow text, 6.7:1 */
  --danger-bg: #dd1d21;          /* Shell Red, white text 4.9:1 */
  --on-accent: #ffffff;          /* text on dark blue, red and navy */
  --focus-ring: #003c88;         /* new */
  --brand-gradient: none;        /* retire; logo and avatars use solid colours */
  --brand-gradient-strong: #003c88;

  /* Status: Shell hues; -text shades are 4.5:1 or more on the 12% tint */
  --status-neutral: #7f7f7f;   --status-neutral-text: #595959;
  --status-review: #eb8705;    --status-review-text: #8f5400;
  --status-positive: #008443;  --status-positive-text: #006b38;
  --status-active: #003c88;    --status-active-text: #003c88;
  --status-done: #404040;      --status-done-text: #404040;
  --status-problem: #dd1d21;   --status-problem-text: #b42c2c;

  /* Environment banner: Shell system message (navy) instead of yellow */
  --banner-bg: #092949;
  --banner-text: #ffffff;

  /* Planning levels: Shell secondary palette; never yellow (yellow means "selected") */
  --level-plan: #003c88;
  --level-campaign: #641964;
  --level-initiative: #008443;
  --level-activity: #009eb4;
  --level-line-item: #7f7f7f;

  /* Team colours (decorative dots, name always written) */
  --team-blue: #003c88;   --team-violet: #641964; --team-teal: #009eb4; --team-magenta: #a3195b;
  --team-orange: #eb8705; --team-green: #008443;  --team-slate: #595959; --team-gold: #c79400;

  /* Layers */
  --overlay: rgba(0, 0, 0, 0.5);           /* Shell dialog backdrop */
  --tooltip-bg: #404040;
  --tooltip-text: #ffffff;

  /* Depth: Shell uses no shadow on cards at rest */
  --shadow-sm: none;                                   /* cards */
  --shadow-md: 0 2px 8px rgba(0, 0, 0, 0.16);          /* menus, popovers, toasts (Shell "shadow") */
  --shadow-lg: 0 2px 16px rgba(0, 0, 0, 0.2);          /* dialogs, side panels (Shell "shadow-lg") */
  --shadow-lift: 0 8px 12px 2px rgba(0, 0, 0, 0.16);   /* new: hover on clickable cards only */

  /* Shape */
  --radius-xs: 4px;              /* new: chips, tags, tooltips, menu items, checkboxes */
  --radius-chip: 4px;            /* was 8px */
  --radius-control: 8px;         /* was 10px: buttons, inputs, selects, search, menus, table outline */
  --radius-dialog: 12px;         /* new */
  --radius-card: 16px;           /* unchanged (Shell 2xl) */
  --radius-pill: 999px;          /* avatars, count badges, toggle switch only */

  /* Control heights: one scale for every control */
  --control-sm: 32px;            /* new: inside tables, card toolbars, chips */
  --control-md: 40px;            /* new: default for buttons, inputs, selects, search, filters */
  --control-lg: 48px;            /* new: sign-in, empty-state call to action, touch layouts */

  /* Spacing: Shell's 4 px scale (names = multiples of 4) */
  --space-1: 4px;  --space-2: 8px;  --space-3: 12px; --space-4: 16px; --space-5: 20px;
  --space-6: 24px; --space-8: 32px; --space-10: 40px; --space-12: 48px; --space-16: 64px;

  /* Layout */
  --content-max: 1300px;         /* was 1120px: Shell --page-width */
  --form-max: 720px;             /* new: one-column forms and reading text */
  --gutter: 24px;                /* new: page side padding from 600px */
  --gutter-sm: 16px;             /* new: below 600px */
  --topbar: 56px;                /* new */
  --sidebar: 240px;              /* was 244px */
  --sidebar-collapsed: 64px;     /* was 72px */

  /* Type */
  --font: Arial, Helvetica, 'Liberation Sans', sans-serif;        /* body and UI, as Shell */
  --font-heading: 'Jost', Arial, Helvetica, sans-serif;           /* new: headings and big numbers */
  --text-xs: 12px;  --leading-xs: 18px;    /* new */
  --text-sm: 14px;  --leading-sm: 20px;    /* new */
  --text-md: 16px;  --leading-md: 24px;    /* new */
  --text-lg: 18px;  --leading-lg: 24px;    /* new */
  --text-xl: 20px;  --leading-xl: 28px;    /* new */
  --text-2xl: 28px; --leading-2xl: 36px;   /* new */
  --text-3xl: 32px; --leading-3xl: 40px;   /* new */

  /* Motion */
  --ease-out: cubic-bezier(0.22, 1, 0.36, 1);
  --dur: 0.25s;                  /* Shell --anim-time */
}

:root[data-theme='dark'] {
  color-scheme: dark;
  /* Shell's own dark values (theme-base.css) */
  --page-bg: #14191f;   --surface: #1d232a;   --gray-bg: #252c34;
  --ink: #e0e0e0;       --ink-2: #b3b8bd;     --muted: #8f969d;
  --control-border: #8f969d; --hairline: #3a434d; --row-divider: #2c343c;
  --chip: rgba(255, 255, 255, 0.09); --hover: rgba(255, 255, 255, 0.07);
  --accent: #d6a100;    --on-accent-dark: #1d232a;   /* Shell dark yellow; dark text 6.8:1 */
  --button-primary-bg: #d6a100; --button-primary-text: #1d232a;
  --button-primary-bg-hover: #ffc800; --button-primary-text-hover: #14191f;
  --blue: #8cb4e6;      --link: #8cb4e6;      --focus-ring: #ffc800;
  --banner-bg: #092949; --banner-text: #ffffff;
  --overlay: rgba(0, 0, 0, 0.6);
  --shadow-md: 0 2px 8px rgba(0, 0, 0, 0.5);
  --shadow-lg: 0 2px 16px rgba(0, 0, 0, 0.6);
}
```

Component edits these tokens require:

- `Button.module.css` `.primary` uses `--button-primary-text` and the hover pair.
- `Logo` and `Avatar` drop the gradient.
- `global.css` sets `letter-spacing: normal` and `font-size: var(--text-sm)`, adds `@font-face` for Jost 600/700, and uses `outline-color: var(--focus-ring)`.

Re-check the dark-mode values with the existing contrast tests (feature 001 research R17).

### 9.3 Type scale (replaces every ad-hoc size)

| Token | Size / line height | Family and weight | Use | Replaces today's |
|---|---|---|---|---|
| `--text-xs` | 12 / 18 | Arial 400; 700 for table headers and chips | Breadcrumbs, help text, captions, table headers, chips, timestamps, footer | 10.5, 11, 11.5, 12, 12.5 |
| `--text-sm` | **14 / 20** | Arial 400; **700** for labels, buttons and emphasis | **Default**: body, table cells, form labels and inputs, buttons, nav items, menu items, errors | 13, 13.5, 14 |
| `--text-md` | 16 / 24 | Arial 400; Jost 600 for card titles | Lead text, card titles, side-panel body on touch, input text on phones | 15, 16, 17 |
| `--text-lg` | 18 / 24 | Jost 600 | Dialog and side-panel titles, form section headings | 17 |
| `--text-xl` | 20 / 28 | Jost 700 | Card and section headings (H2 inside a page), empty-state titles | — |
| `--text-2xl` | 28 / 36 | Jost 700 | **Page title (H1)**, summary-card values | 22, 26, 27, 28 |
| `--text-3xl` | 32 / 40 | Jost 700 | Home greeting only | — |

Rules:

- Sentence case everywhere.
- No upper-case labels and no letter spacing.
- Headings use the same ink as body text.
- Numbers use `font-variant-numeric: tabular-nums` (Arial's digits are already equal width).
- Headings use a 1.25–1.3 line height, not Shell's 1.5. Shell's public pages can afford 48 px lines for a 32 px title; a working app needs compact headers.

### 9.4 Spacing scale and rules

Only `--space-*` values are allowed. The single exception is 1–2 px for borders and focus offsets.

| Situation | Value |
|---|---|
| Label to control | 8 |
| Control to help or error text | 4 |
| Between form fields | 20 |
| Between form sections | 32 |
| Card padding | 24 (20 on phones) |
| Card header to body | 16 |
| Between children inside a card | 20 (Shell `space-y-5`) |
| Gap between cards in a grid | 20 (Shell `gap-5`) |
| Page header: top bar to breadcrumb | 16 |
| Breadcrumb to title | 8 |
| Title to description | 4 |
| Header to content | 24 |
| Toolbar items | 8 |
| Table cell padding | 8 vertical × 12 horizontal (row height 40); compact 4 × 8 (32) |
| Icon to text in buttons and menu items | 8 |

Mapping from today's off-grid values: 6 → 8 (or 4 for tight pairs), 10 → 8 or 12, 14 → 12 or 16, 18 → 16 or 20, 26 → 24.

### 9.5 Grid, page widths and breakpoints

- **Content column:** fluid up to `--content-max: 1300px`, centred in the area right of the sidebar, with 24 px side padding (16 px below 600 px). At 1440 × 900 with the sidebar open, a list page gets about 1150 px of table width, up from today's 1080 px.
- **12-column grid with 20 px gaps** for dashboards:
  - summary cards: 3 or 4 per row, each spanning 3 or 4 columns;
  - details pages: an 8 + 4 split, main content and a side summary.
- **Forms and reading text:** one column, at most 720 px, aligned to the *left edge* of the content column, not centred. This keeps every page's left edge the same, as Shell does.
- **Breakpoints (three only):**
  - **< 600** phone: sidebar becomes a drawer, cards stack, tables become cards for inbox-type lists;
  - **600–959** tablet: icon rail;
  - **≥ 960** desktop: full sidebar, which can be collapsed;
  - one extra **≥ 1280** rule is allowed for the details-page side summary.

  Replace all 11 current media queries with these.
- **Alignment rule:** logo, breadcrumb, page title, toolbar, table and footer all start on the same left edge, and page actions end on the same right edge. Every control in a toolbar uses the same height, so baselines line up.

### 9.6 Component specs

| Component | Size | Radius | Border | Background / text | Shadow | States |
|---|---|---|---|---|---|---|
| **Primary button** | 40 (md) / 32 (sm) / 48 (lg); padding 0 16 (12 with icon); icon 20; text 14 bold | 8 | 1 px same as fill | `#FFC800` / `#404040` | none | Hover: `#404040` fill, `#FFC800` text. Pressed: `#2b2b2b`. Focus: 2 px `#003C88` ring, 2 px offset. Disabled: avoid (use `aria-disabled` plus an explanation, per design system §6.6); if truly disabled, `#E8E8E8` / `#757575`. **At most one per view.** |
| **Secondary button** | Same sizes | 8 | 1 px `#404040` | White / `#404040` | none | Hover `--hover` fill; focus ring |
| **Tertiary (ghost) button** | Same sizes | 8 | none | Transparent / `#404040` | none | Hover `--hover` |
| **Danger button** | Same sizes | 8 | none | `#DD1D21` / white | none | Hover `#B8171B`. Only inside confirmation dialogs; elsewhere use a secondary button with `--status-problem-text` text |
| **Icon button** | Square 40 / 32; icon 20 / 16 | 8 | none | Transparent / `#404040` | none | `aria-label` + tooltip (Shell gives `title` + `aria-label`) |
| **Text input, select, combobox, date, money** | 40 high (48 on touch), padding 0 12, text 14 (16 below 960 px) | 8 | 1 px `--control-border` (3.4:1) | White / ink; placeholder `--muted` | none | Hover border `#404040`; focus border `#003C88` + ring; invalid border `#DD1D21` + message; read-only `#F5F5F5` with no border |
| **Search field** | 40; magnifier icon left; clear button right | **8** (not a pill) | As input | As input | none | Placeholder says what it searches; result count beside it ("12 of 40") |
| **Checkbox / radio** | 20 box, 40 click area | 4 (checkbox), round (radio) | 1.5 px `--control-border` | Checked: `#003C88` fill, white tick | — | Focus ring |
| **Toggle switch** | 40 × 24 | pill | — | Off `#8C8C8C`, on `#003C88` | — | Label on the right |
| **Card / panel** | Padding 24; title Jost 600 16–20 | 16 | none (on the grey canvas) | White | none | Clickable cards: `--shadow-lift` on hover, no movement when reduced motion is set |
| **Summary (KPI) card** | Min height 104; padding 20 | 16 | none | White; label 12 bold `#595959`; value Jost 700 28 `#404040`, tabular | none | Whole card is a link where useful; "View" link at the bottom right (Shell "Read more →") |
| **Table / tree table** | Header 40 (12 bold `#404040` on `#F5F5F5`, bottom 1 px `#D9D9D9`); rows 40 (compact 32); cell padding 8 × 12; text 14 | Outer 8 | Outer 1 px `#D9D9D9`; rows 1 px `#E8E8E8`, no vertical lines | White | none | Row hover `--hover`; selected row: `#FFF8D6` fill + 4 px yellow start bar; sticky header; money right-aligned, tabular; totals row bold with a 2 px top border `#404040`; level bars stay 4 px in level colours |
| **Tabs (page sections)** | 48 high; text 14 (bold when selected); 24 between tabs | — | Container bottom 1 px `#D9D9D9`; selected **4 px `#FFC800` bottom bar** | Transparent | — | `role="tablist"`, arrow keys; tabs overflow by scrolling sideways; state kept in the URL |
| **Segmented filter** (status: All / Draft / Published) | 32 high; joined buttons; text 14 | 8 outer | 1 px `--control-border` | Selected `#404040` / white; others white / ink | — | Count in brackets ("Draft (2)") |
| **Filter chips** (applied filters) | 32 high; text 12 bold; × button 20 | 4 | none | `--chip` / ink | — | Remove on ×; "Clear filters" link |
| **Status chip** | 24 high; padding 0 8; text 12 bold; optional 8 px dot | 4 | none | 12% tint / `-text` shade | — | Word + colour (unchanged rule) |
| **Level tag** | 20 high; text 12 bold | 4 | 1 px level colour | White / ink | — | — |
| **Breadcrumb** | Text 12; chevron 12; 8 between | — | — | Links `#404040` (underline on hover); current item ink, not a link | — | Below 600 px: "‹ Parent" only; `aria-current="page"` |
| **Page header** | Breadcrumb, then title Jost 700 28/36, then description 14 `#595959` (max 720) | — | — | — | — | Actions at the right, aligned to the title: ⋯ More, secondary, primary (rightmost) |
| **Side panel (drawer)** | 480 (short records) / 560; header 64 (title Jost 600 18 + close 32); body padding 24; sticky footer 72 with buttons on the right | 0 (16 on the inner edge allowed) | Left 1 px `#D9D9D9` | White | `--shadow-lg` | Escape closes; focus trapped; backdrop `--overlay` |
| **Dialog** | 512 (confirmations, as Shell) / 720 (short forms); padding 24; title Jost 700 20 | 12 | none | White | `--shadow-lg` | Native `<dialog>` semantics; backdrop black at 50% (optional 4 px blur); Cancel focused for destructive actions |
| **Toast** | Width 360–480; padding 12 × 16; text 14; bottom centre, 24 from the bottom | 8 | 4 px start bar in the status colour | White / ink | `--shadow-md` | Slides up 48 px (Shell "toast" motion; none with reduced motion); action link bold |
| **Inline alert** | Padding 12 × 16; icon 20; title 14 bold; text 14 | 8 | 4 px start bar in the status colour | Status at 8% tint / ink | none | `role="status"` or `role="alert"` |
| **Environment banner** | 32 high; text 12 | 0 | — | Navy `#092949` / white | — | As Shell's system message |
| **Tooltip** | Padding 4 × 8; text 12; max width 280 | 4 | — | `#404040` / white | `--shadow-md` | Hover and focus; Escape hides |
| **Menu (⋯ actions, dropdowns)** | Width 220–288; items 40 high, 14 text, icon 16 | 8 (items 4) | — | White | `--shadow-md` | Hover `--hover`; delete item last, separated, red text; Escape returns focus; Shell's expand animation |
| **Avatar** | 32 (top bar) / 24 (tables); initials 12 bold | pill | — | `#003C88` / white (10.5:1) | — | — |
| **Empty state** | Icon 48 `#595959`; title Jost 700 20; text 14; primary button 40 | — | — | — | — | Same wording rules as today |
| **Skip link** | Padding 16; text 14 bold | 8 | — | `#FFC800` / `#404040` | — | As Shell (image 09) |
| **Focus ring (all controls)** | 2 px solid `--focus-ring`, 2 px offset | Follows the control | — | — | — | `#003C88` in light mode (10.5:1 on white, 7:1 on yellow); `#FFC800` in dark mode |

### 9.7 Navigation layout for an app marketers use all day

```
┌──────────────────────────────────────────────────────────────────────────────────────────┐
│ [Pecten] MAPP   │ 🔍 Search plans, campaigns, activities by name or ID │  ?  🔔3  (AB) │  ← top bar 56 px, white, 1 px #D9D9D9 bottom
├────────────────┬─────────────────────────────────────────────────────────────────────────┤
│ ▌⌂ Home        │  Plans › FY2026 Lubricants                         (breadcrumb, 12 px)   │
│  ▤ Plans       │  Helix launch                                [⋯ More] [Edit] [Submit]   │  ← title 28 px Jost; one yellow button
│  ✓ Approvals 4 │  Campaign · Draft · 3 fields needed           (chips 24 px)             │
│  ▦ All work    │  ────────────────────────────────────────────────────────────────────── │
│                │  Overview   Initiatives (3)   Activities (12)   History                  │  ← underline tabs, 4 px yellow bar
│  Admin         │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐     │
│  ⚙ Settings    │  │ Planned cost │ │ Committed    │ │ Unassigned   │ │ Activities   │     │  ← white KPI cards, radius 16
│  ↺ Recycle bin │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘     │     on #F5F5F5 canvas, 20 px gaps
│  ☰ Activity log│  ┌──────────────────────────────────────────────────────────────────┐   │
│                │  │ table: 40 px rows, sticky header, money right-aligned            │   │
│  « Collapse    │  └──────────────────────────────────────────────────────────────────┘   │
└────────────────┴─────────────────────────────────────────────────────────────────────────┘
   sidebar 240 px white; items 40 px; ▌ = 4 px yellow bar + bold text on the current section
```

**Top bar (56 px, white).** It carries the global tools, which today sit in the sidebar footer. This mirrors Shell's header, where the tool icons and Sign in are at the top right.

- Left: the Pecten mark (subject to brand approval, §10) and the product name "MAPP" in Jost 600 18 px. This mirrors Shell's `Navigation` title next to the logo.
- Centre-left: one **global search**, 40 px high and 360–480 px wide. It searches by name or ID across all levels, which serves the usability target "find an activity ≤ 2 steps".
- Right: Help (?), Notifications (bell with count), and the account avatar menu (name, role, theme, sign out). All are 40 px icon buttons with names.

**Left sidebar (240 px, white, 1 px `#D9D9D9` right border).**

- Items are 40 px high, Arial 14, with a 20 px icon, in `#404040`.
- Hover fills with `--hover`. The 2 px "nudge" animation is removed.
- **Current section: a 4 px yellow bar at the start edge, bold text and a `#F5F5F5` fill**, with `aria-current="page"`. This is Shell's vertical-menu selection; Shell puts the bar on the end edge in dropdowns, but the start edge reads better in a sidebar.
- Group labels such as "Admin" are 12 px bold `#595959` in sentence case (no upper case).
- Count badges ("Approvals 4") are pills, 12 px bold, in `#404040` with white text.
- Collapse to a **64 px icon rail** (tooltips on hover) by choice, or automatically between 600 and 959 px. Below 600 px it becomes a drawer with an overlay.

**Content area.**

- `#F5F5F5` canvas, 24 px padding, at most 1300 px wide.
- Breadcrumb on every page below the top level.
- Page header with at most one yellow primary button, at the far right, then secondary buttons, then ⋯ More.
- **Sticky** table headers and form footers, so the next action stays visible on long pages.

**Sections inside a record** use underline tabs. Pill toggles are kept only as segmented filters.

**Footer.** One 12 px row, not Shell's marketing footer: "MAPP · version · Accessibility · Privacy · Help". For example: "Data as of 20 Oct 2026 14:05" (the app equivalent of Shell's "Page last updated").

**Why this layout:** marketers work in MAPP for hours, moving between lists, records and approvals.

- A persistent left sidebar keeps every module one click away. This is how Shell's own fleet portal works.
- The top bar gives search and notifications a fixed place, as on every Shell site.
- The yellow "you are here" bar is the single most recognisable Shell navigation cue.

### 9.8 Dark mode

Keep dark mode (decision D-038), but use Shell's own dark values (§9.2). Yellow becomes `#D6A100` with dark text, and the focus ring turns yellow on dark backgrounds. Shell supports dark mode only through the computer's setting. MAPP's per-person switch can stay.

### 9.9 Accessibility rules to adopt or keep

1. Control borders at 3:1 or more (`--control-border`). This is the one current failure.
2. Never use yellow as text or as a focus ring on light backgrounds. Yellow is a fill with dark text.
3. Keep the 2 px focus ring with 2 px offset, in `#003C88` (light) and `#FFC800` (dark).
4. Keep reduced-motion support. Clickable-card lift and toast slides are switched off by it.
5. Copy Shell's patterns:
   - "Go to: *parent*" as the first item of any dropdown that has children;
   - breadcrumb collapsing to the parent on phones;
   - 16 px input text on touch screens;
   - `title` + `aria-label` on icon buttons;
   - native `<dialog>` and `<details>` where they fit.
6. Touch targets: 40 px controls meet WCAG 2.5.8 (24 px minimum), and 48 px is available for touch layouts.

### 9.10 Rollout order (lowest risk first)

1. **Tokens and fonts** (§9.2, §9.3): swap the values, add Jost, remove the gradient. Almost every screen changes at once, with few code edits.
2. **One control height**: Button, TextField, inputs, Combobox, SearchField, PillTabs, SegmentedFilter, FilterBar, ExportMenu and ColumnChooser all move to 32/40/48. This fixes most of the misalignment.
3. **Snap spacing and sizes**: replace the 15 font sizes and the off-grid spacing with tokens. A stylelint rule that rejects raw `px` font sizes and spacing outside the scale keeps it that way.
4. **App shell**: top bar, white sidebar with the yellow bar, page header with breadcrumb, footer row, navy environment banner.
5. **Tabs, tables, cards and dialogs** to the specs in §9.6.
6. **Page-by-page pass** (Home, Plans, plan details, All work, Approvals, Settings), checked at 1440 × 900, 1280 × 720 and 390 × 844.

Checks at each step:

- screenshot comparison of the main screens;
- automated accessibility scan (axe), including non-text contrast;
- a keyboard walk through the standard CRUD flow;
- the existing unit and e2e smoke tests.

Update `docs/product/design-system.md` §2–§5 and §9 in the same change as the tokens, so the specification and the code never disagree.

---

## 10. Questions for the client before building

1. **Shell typeface and Pecten logo.** May MAPP, as an internal Shell tool, use the Shell font files (Shell Book, Bold and Heavy; URW) and the Pecten? If yes, Jost is replaced by one token. If not, Jost (OFL) and a neutral MAPP wordmark are used. Either way, MAPP must not download the files from shell.com.
2. **Primary-button colour.** Shell's current sites use **yellow with dark text**; Fleet Hub uses **Shell Dark Blue with white text**. This report recommends yellow, for consistency with today's Shell brand. Please confirm, or name the colour Shell's digital brand team prefers for staff tools.
3. **Page background.** Shell's look puts white cards on light-grey bands (`#F5F5F5`). MAPP's constitution principle VI says "white". Cards, dialogs, top bar and sidebar all stay white; only the canvas behind them turns light grey. Is that acceptable? The alternative is a white canvas with 1 px `#D9D9D9` card borders.
4. **Shell design system for staff tools.** Does Shell have an internal design system or brand portal for staff and B2B apps (tokens, icons, components)? None is public. If one exists, it should override any value in this report.
5. **Decision log.** Approve a new decision that replaces the *visual* part of D-005 (the version 1 Apple-inspired look) with "Shell Amidala visual language". The functional rule stays: Hive9 screens are not copied.

---

## Appendix A — evidence images

| File | What it shows |
|---|---|
| [01-shell-uk-business-header.jpg](ux-research-shell-ui-patterns/01-shell-uk-business-header.jpg) | Utility bar with yellow "Business" bar, main nav, tool icons, breadcrumb, Shell Heavy titles, yellow button, grey cards with "Read more →" |
| [02-shell-uk-search.jpg](ux-research-shell-ui-patterns/02-shell-uk-search.jpg) | Search form (48 px input, radius 8, square yellow submit), result count, results with breadcrumb paths |
| [03-shell-uk-home-mobile.jpg](ux-research-shell-ui-patterns/03-shell-uk-home-mobile.jpg) | Phone header (utility bar, icons, burger), stacked layout, underlined links with icons |
| [04-shell-fleet-hub-info-page.jpg](ux-research-shell-ui-patterns/04-shell-fleet-hub-info-page.jpg) | B2B product page: breadcrumb, H1, yellow quote cards, accordion, yellow CTA band with dark button |
| [05-shell-footer.jpg](ux-research-shell-ui-patterns/05-shell-footer.jpg) | White cards on a grey band; footer with breadcrumb, "Page last updated", link columns, legal row |
| [06-shell-fleet-hub-portal-landing.jpg](ux-research-shell-ui-patterns/06-shell-fleet-hub-portal-landing.jpg) | Older generation: dark-blue full-width login button, outlined secondary buttons, square corners |
| [07-shell-fleet-app-store-listing.jpg](ux-research-shell-ui-patterns/07-shell-fleet-app-store-listing.jpg) | Shell Fleet App screens: yellow primary button, underline tabs, bottom nav, ring charts |
| [08-font-comparison.png](ux-research-shell-ui-patterns/08-font-comparison.png) | Shell Heavy and Shell Book compared with Jost, Outfit, League Spartan, Figtree, Arial and Inter |
| [09-shell-skip-link-focus.jpg](ux-research-shell-ui-patterns/09-shell-skip-link-focus.jpg) | Skip link on focus with the browser's default focus outline |

## Appendix B — how the measurements were taken

- Pages were loaded in headless Chromium (Playwright) at 1440 × 900 and 390 × 844, locale en-GB. Optional cookies were rejected before measuring. Computed styles were read from the elements themselves, including open shadow roots. A keyboard focus walk recorded the outline of each focused element.
- Font facts come from the font files' `name` and `OS/2` tables (fontTools). Candidate fonts were measured the same way and rendered with Pillow.
- Contrast uses the WCAG 2.x relative-luminance formula.
- MAPP figures come from searching all 155 CSS files under `frontend/src` (font sizes, weights, radii, padding, gaps, heights and media queries) on 20 Oct 2026.
- No personal data was collected; no sign-in was attempted.
