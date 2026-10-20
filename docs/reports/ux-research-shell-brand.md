# Shell brand and design language: research for the NorthStar UI revamp

| | |
|---|---|
| **Date** | 20 October 2026 (all pages captured on this date) |
| **For** | Product owner and the Shell client team |
| **Question** | How do Shell's public websites look and behave, and what should an internal Shell planning tool (NorthStar) take from them? |
| **Scope** | Brand principles and tone, colour, logo (pecten) rules, imagery and icons, fonts, consumer vs business pages. A token set for NorthStar with checked contrast. |
| **Out of scope** | Detailed layout, navigation and component research. Other reports in this folder cover those. This report touches them only where they carry the brand. |

---

## Summary

1. **All Shell websites we studied run on one shared design system with named colour tokens.** shell.com, shell.co.uk, shell.us, shell.nl, the careers, newsroom and investor pages, and the business pages all load the same front-end library and the same theme file. Its colour tokens can be read straight from the code (§3.1). So "Shell's design expectations" for the web are concrete numbers, not guesses.
2. **Shell yellow on the live sites is `#FFC800` and Shell red is `#DD1D20`** (written in the code as `hsl(47 100% 50%)` and `hsl(359 77% 49%)`). The often-quoted values `#FBCE07` and `#DD1D21` are older or rounded versions. Shell's older Fleet Hub portal still uses `#FBCE07`. Official print and screen specifications are only in Shell's brand portal, which needs a Shell login (§3.5).
3. **Brand colour is used very sparingly.** Text is one dark grey (`#4A4A4A`) almost everywhere. Backgrounds are white and a light grey (`#F5F5F5`). A whole page paints only 8 to 10 distinct colours. On the homepage, 440 text elements are grey and about 24 are yellow, and all of those yellow ones sit on dark-grey buttons. **Red appears in no interface element on 15 of the 17 Shell web pages we measured.** The two exceptions are red buttons on the careers page and a few red accents on the brand-history page. Otherwise red lives only in the logo and the photography (§3.4).
4. **Yellow means "act here" and "you are here".** It fills the main button (with dark-grey text, never white), marks the current menu item with a 4 px bar, and fills some promotional panels. Yellow is never used for text on white, because it reaches only 1.55:1 contrast there (§3.4, §9).
5. **The pecten is a registered trade mark and cannot be used without Shell's written consent.** Shell runs a yearly consent process through its "Brand Central" portal and sends logo files only after an agreement is signed. Any artwork carrying the logo goes to `Askbrand@shell.com` for approval (§4). NorthStar must not show the pecten until Shell grants this.
6. **Headings use a proprietary Shell typeface and body text uses Arial.** The heading font file names itself "ShellHeavy" and carries the notice "Copyright 2018 by URW Type Foundry GmbH. All rights reserved". Shell's own body text stack is `Arial, Roboto, Helvetica, sans-serif`, which anyone may use (§5).
7. **Business pages look more restrained than consumer pages.** shell.co.uk (drivers) is full of yellow panels, app badges, red Ferrari imagery and "+" sparkles. shell.com (corporate) and the business pages are mostly white and grey with yellow only on buttons. Sub-brands such as Shell Recharge and Shell Energy swap in a teal accent while keeping yellow for actions (§8). **An internal enterprise tool should follow the corporate and business look, not the consumer one.**
8. **Shell publicly commits to WCAG 2.2 A and AA** (higher than NorthStar's current WCAG 2.1 target), 400 % zoom and dark mode that follows the system setting. Shell's own sites define a full dark palette, which we can reuse (§9).
9. **NorthStar today does not look like Shell.** It uses an Apple-style blue (`#0071e3`) for actions and a blue-to-purple gradient for the brand (`frontend/src/styles/tokens.css:30-41`). Neither appears anywhere on Shell's sites. §10 proposes a Shell-aligned token set with every pair checked against WCAG AA in light and dark mode. It also lists what must wait for official Shell assets.

**Recommendation in one line:** keep NorthStar white, calm and simple. Replace the blue and the gradient with **Shell yellow for the one main action and the current place**, Shell's grey ink for text, **Shell red only for danger and errors**, and Shell's secondary colours (blue, green, purple, teal) for data meaning. Leave the logo and the Shell typeface out until Shell supplies them.

---

## 1. Method and sources

**How we looked.** Each page was opened in a headless Chromium browser at desktop size (1440 × 900) and phone size (390 × 844), in light mode and, for two pages, in dark mode. For every visible element we read the computed styles: font, size, weight, colour, background, border, corner radius and shadow. We counted which colours and fonts are actually painted. We also read the design tokens directly from Shell's shared theme stylesheet and its JavaScript bundle, read the font file's name table, and read Shell's public documents on trade-mark use and accessibility. Contrast ratios use the WCAG 2.x formula.

**Why the screenshots are not in this repository.** Shell's website terms say its material, including the "Shell" mark and the Shell emblem, may not be copied or distributed without written consent ([terms of use](https://www.shell.com/terms-of-use.html), section on copyright and trade marks). So we cite each page by URL, and anyone can open it. The screenshots stayed in the researcher's temporary working folder, and the copy of the font file read for §5 was deleted after reading.

**Pages studied**

| Area | Page | Type |
|---|---|---|
| Corporate | [shell.com](https://www.shell.com/) · [Careers](https://www.shell.com/careers.html) · [Newsroom](https://www.shell.com/news-and-insights/newsroom.html) · [Investors](https://www.shell.com/investors.html) · [Our brand history](https://www.shell.com/who-we-are/our-history/our-brand-history.html) · [Terms of use](https://www.shell.com/terms-of-use.html) · [Accessibility](https://www.shell.com/accessibility.html) | Corporate |
| UK consumer | [shell.co.uk](https://www.shell.co.uk/) · [Shell GO+ Rewards](https://www.shell.co.uk/shell-go-plus.html) · [EV charging (Shell Recharge)](https://www.shell.co.uk/electric-vehicle-charging.html) · [Who we are (UK)](https://www.shell.co.uk/about-us/who-we-are.html) | Consumer |
| US | [shell.us](https://www.shell.us/) · [Shell Energy (US business)](https://www.shell.us/business/shell-energy.html), reached from shellenergy.com | Consumer + B2B |
| Business (B2B) | [shell.co.uk Business](https://www.shell.co.uk/business.html) · [Shell Fleet Solutions](https://www.shell.co.uk/business/shell-fleet-solutions.html) · [Business customers](https://www.shell.com/business-customers.html) · [Lubricants for business](https://www.shell.com/business-customers/lubricants-for-business.html) · [Shell Brand Licensing](https://www.shell.com/business-customers/shell-brand-licensing.html) · [Shell Fleet Hub](https://fleethub.shell.com/) (customer portal landing page) | B2B |
| Sub-brand | [shellrecharge.com](https://shellrecharge.com/), which now redirects to [shell.nl/elektrisch-opladen](https://www.shell.nl/elektrisch-opladen.html) | Consumer |
| Brand rules | [Shell TM Consent & Artwork Approval Process (Shell USA, Jan 2025, PDF)](https://www.shell.us/about-us/projects-and-locations/_jcr_content/par/textimage.stream/1713545199865/2a7400272d635be4d99f89c8d71deb1c80104f1c/shell-brand-approval-process-external.pdf) · [Brand Central](https://brandcentral.shell.com/) (sign-in only) | Official |

**Limits**

- **No public Shell brand book exists.** Shell's brand guidelines, logo files and official colour specifications sit behind the [Brand Central](https://brandcentral.shell.com/) sign-in. Web searches also return several PDFs titled "Shell Brand Guidelines" on unrelated domains. They are unverified and we did not use them.
- **Shell appears to have an internal design system for its own apps.** A search engine lists a Shell intranet page titled "Shell DS React Native Framework" ([realtime.shell.com](https://realtime.shell.com/RTATD/AllProjects/Devkit), sign-in required). The search summary describes it as a component framework whose components are brand compliant and accessibility compliant. We could not open the page itself. If Shell's digital team can share it, it outranks everything in this report (§10.7).
- **Shell Energy UK no longer exists as a Shell brand for households.** `shellenergy.co.uk` no longer resolves. Shell sold its UK and German home-energy business to Octopus Energy, completed on 1 December 2023 ([S&P Global](https://www.spglobal.com/commodity-insights/en/news-research/latest-news/electric-power/090123-shell-sells-uk-german-home-energy-businesses-to-octopus-energy), [EV Infrastructure News](https://www.evinfrastructurenews.com/public-charging/shell-energy-retail-becomes-octopus-energy-operations-2-as-acquisition-completes)). "Shell Energy" survives as a business brand (UK business menu, US business site).
- **The Shell App itself was not tested.** We studied the web pages that promote it (Shell GO+ and app pages). The app screens shown on those pages use the same language: white cards, yellow buttons and grey text.
- **The shared JavaScript and CSS files have content-hash names** (for example `app.min.ACSHASHc4fb…js`). They will change when Shell redeploys, so the values in §3 are a snapshot from 20 October 2026.

---

## 2. Brand principles and tone

**Purpose and values (Shell's own words).**

- Purpose: "to power progress together, by working with each other, our customers and our partners" ([shell.com home](https://www.shell.com/), "Our strategy" card). The US homepage headline is simply "Powering Progress Together" ([shell.us](https://www.shell.us/)).
- Core values: **respect, honesty, integrity** ([Who we are, Shell UK](https://www.shell.co.uk/about-us/who-we-are.html), "Our values").

**What the brand history says about the look** ([Our brand history](https://www.shell.com/who-we-are/our-history/our-brand-history.html)):

- The pecten (Latin for scallop) dates from 1900 as a trade mark: "the oldest of 22,000+ trademarks owned by Shell".
- Raymond Loewy designed today's pecten in 1971. "The 'Shell' name was removed from the centre of the symbol and placed below in a specially designed typeface."
- The current colours date from the early 1990s: "a warmer yellow and red to soften the feel of our brand and give it a broader appeal".
- A sound logo, the "Sound of Shell", was added in 2015.

**Tone seen on the sites.** No public tone-of-voice guide exists. The pages consistently show:

- **Short, plain, benefit-first headlines in sentence case**: "Shell keeps you moving", "Meeting growing LNG demand", "Fleet management made effortless."
- **Helpful questions as section titles**: "How can we help?" on the homepage and "Can we help?" in every footer.
- **Verb-first action labels**: "Find a Shell station", "Find an EV charge point", "Search for a job", "Contact Us", "Request a call back". The generic "Read more" is also very common.
- **Calm, factual corporate copy** on shell.com, investors and newsroom. Promotional copy and asterisked offer terms appear only on consumer pages.
- **Small inconsistencies**: "Read more" and "Read More" on the same page ([shell.co.uk](https://www.shell.co.uk/)), and all-capital page titles on the [terms](https://www.shell.com/terms-of-use.html) and error pages ("TERMS AND CONDITIONS", "ERROR PAGE") while every other page uses sentence case.

**What this means for NorthStar.** NorthStar's existing writing rules already match Shell's style: plain words, sentence case, verb buttons, friendly specific messages (`docs/product/design-system.md` §8). No change is needed beyond applying them consistently. Shell's own small slips show why a written rule helps.

---

## 3. Colour

### 3.1 The colour tokens every Shell site shares

Every Shell site we studied loads the same front-end library (Adobe Experience Manager pages with a shared library the code calls `amidala`). It sets these custom properties on the page root. We verified them as computed values on every page in the table in §1.

| Shell token | Written in Shell's code (HSL) | Hex | What Shell uses it for (observed) | Contrast on white |
|---|---|---|---|---|
| `--color-paper1` | `0, 0%, 100%` | `#FFFFFF` | Page background, buttons on yellow | – |
| `--color-paper2` | `0, 0%, 96%` | `#F5F5F5` | Cards, alternating section bands, utility bar | 1.09 : 1 |
| `--color-ink1` (= `ink2`) | `0, 0%, 29%` | `#4A4A4A` | All text, icons, links, the dark "inverse" button | 8.86 : 1 |
| `--color-brand1-bg` | `47, 100%, 50%` | **`#FFC800` (Shell yellow)** | Main buttons, current-menu bar, help band, promo panels, skip link | 1.55 : 1 |
| `--color-brand1-text` | = ink1 | `#4A4A4A` | Text on yellow | 5.70 : 1 on yellow |
| `--color-brand2-bg` | `359, 77%, 49%` | **`#DD1D20` (Shell red)** | Secondary buttons on the careers page only | 4.92 : 1 (white text on red) |
| `--color-brand2-text` | = paper1 | `#FFFFFF` | Text on red | – |
| `--color-brand3-bg` | `212, 49%, 39%` | `#336094` (blue) | Defined; not seen painted on the pages measured | 6.48 : 1 |
| `--color-brand4-bg` | `159, 100%, 26%` | `#008556` (green) | Defined; not seen painted | 4.68 : 1 |
| `--color-brand5-bg` | `306, 61%, 33%` | `#87217D` (purple) | Defined; not seen painted | 8.26 : 1 |
| `--color-system-bg` | `210, 78%, 16%` | `#092949` (navy) | Defined with white text (`--color-system-text`); the name suggests system messages; not seen painted | 14.74 : 1 |
| `--color-black-bg` | `0, 0%, 0%` | `#000000` | Defined; not seen painted as a panel | – |

Other values seen: dividers are ink at 20 % opacity (about `#DBDBDB`), the input border is ink at 60 % (about `#929292`, 3.11 : 1), and hero text panels are black at 60 % opacity over the photograph.

Source files: the theme stylesheet [`theme-base.…css`](https://www.shell.com/etc.clientlibs/amidala/clientlibs/theme-base.ACSHASHb291465fc8d3c7bcefde74d34044b698.css) holds the dark-mode values, and the light values are set by the shared [`app.min.…js`](https://www.shell.com/etc.clientlibs/amidala/clientlibs/app.min.ACSHASHc4fb4841e466a60a7ed3479e9efcaf79.js). Exact extract:

```css
--color-paper1: 0, 0%, 100%;   --color-paper2: 0, 0%, 96%;   --color-ink1: 0, 0%, 29%;
--color-brand1-bg: 47, 100%, 50%;   --color-brand1-text: var(--color-ink1);
--color-brand2-bg: 359, 77%, 49%;   --color-brand2-text: var(--color-paper1);
--color-brand3-bg: 212, 49%, 39%;   --color-brand4-bg: 159, 100%, 26%;   --color-brand5-bg: 306, 61%, 33%;
--color-system-bg: 210, 78%, 16%;   --color-system-text: 0, 0%, 100%;
```

### 3.2 Shell's dark mode

Shell's sites switch to dark mode automatically when the computer is set to dark (`@media (prefers-color-scheme: dark)` in the theme stylesheet). We confirmed this by rendering [shell.co.uk](https://www.shell.co.uk/) and [shell.co.uk/business](https://www.shell.co.uk/business.html) in dark mode.

| Token | Dark value (HSL) | Hex | Note |
|---|---|---|---|
| `paper1` | `212, 18%, 14%` | `#1D232A` | Page background: blue-tinted charcoal, not black |
| `paper2` | `213, 22%, 10%` | `#14191F` | Cards and bands |
| `ink1` | `0, 0%, 88%` | `#E0E0E0` | Text (12.0 : 1 on `#1D232A`) |
| `brand1-bg` | `45, 100%, 42%` | `#D6A100` | **Yellow is dimmed in dark mode**, and its text becomes the dark `paper1` (6.75 : 1) |
| `brand2-bg` | `15, 100%, 36%` | `#B82E00` | **Red shifts to a deep orange-red** in dark mode (white text 6.12 : 1) |
| `system-bg` | unchanged | `#092949` | |

The pecten keeps its full colours in dark mode.

### 3.3 Sub-brand and older-generation palettes

| Where | What changes | Evidence |
|---|---|---|
| **Shell Recharge** (EV charging) | A sub-brand theme file overrides only `brand3` to teal `hsl(187 100% 29%)` = `#008192`, plus two pale teal tints (`#C0E6EC`, `#ECF6F9`). Yellow, red, ink and paper stay the same. | [`theme-shell-recharge.…css`](https://www.shell.nl/etc.clientlibs/amidala/clientlibs/theme-shell-recharge.ACSHASH23a752d535925afba7aeea6dc8e3b8f7.css); painted on [UK EV charging](https://www.shell.co.uk/electric-vehicle-charging.html) |
| **Shell Energy** (US business) | Teal-blue `#0080A3` used for 42 text elements and 7 backgrounds; yellow kept for actions | [Shell Energy US](https://www.shell.us/business/shell-energy.html) |
| **Shell Fleet Hub** (B2B customer portal) | An older visual generation: yellow `#FBCE07`, dark blue `#003C88` for the main login button with white text, text `#404040`, square corners, upper-case labels, its own "Shell-Bold" web font | [fleethub.shell.com](https://fleethub.shell.com/) |

**Takeaway.** Shell's pattern is "**master brand fixed, one accent per sub-brand**". Yellow, ink and paper never change. A product or sub-brand adds one secondary colour. NorthStar could do the same: Shell yellow for actions, and Shell's secondary colours for meaning (§10).

### 3.4 How sparingly brand colours are used

We counted the computed colour of every visible element:

| Page | Text in ink `#4A4A4A` | Yellow (as text or fill) | Red in the interface | Distinct colours painted |
|---|---|---|---|---|
| [shell.com](https://www.shell.com/) | 440 elements | 24 yellow labels, all on dark-grey buttons in one "How can we help?" band; 2 yellow fills | **0** | 10 |
| [shell.co.uk](https://www.shell.co.uk/) | 546 | Several promo panels, main buttons, 64 px icon circles, "Sign in" | **0** (red only in a Ferrari photograph) | 8 |
| [shell.co.uk/business](https://www.shell.co.uk/business.html) | 416 | 3 (buttons, contact band) | **0** | 8 |
| [Newsroom](https://www.shell.com/news-and-insights/newsroom.html) | 675 | 3 ("Load more", two contact buttons) | **0** | – |
| [Careers](https://www.shell.com/careers.html) | 1,050 | 6 buttons | **5** red buttons ("Apply to the Programme", "Learn more") | 10 |
| [Terms](https://www.shell.com/terms-of-use.html) | 436 | **0** | **0** | – |
| [Our brand history](https://www.shell.com/who-we-are/our-history/our-brand-history.html) | 575 | – | 3 red text elements, 1 red fill | – |
| 11 further pages (shell.us, investors, business customers, lubricants, UK business, Fleet Solutions, EV charging, GO+, Who we are UK, Shell Energy US, Shell Recharge NL) | 320 to 813 each | Buttons and promo panels | **0** on every page | – |

**Rules Shell's sites follow in practice:**

- **Ink grey carries the page.** Headings, body, links, icons and dividers all use one grey, `#4A4A4A`, at different weights and sizes. Links are the same grey, bold or underlined, **never blue**.
- **Yellow is the action and location colour.** It fills the main button (48 px tall, 8 px corners, bold 14 px dark-grey text). It draws the 4 px bar under the current menu item and the current tab. Yellow is never used for text on white.
- **The inverse button is dark grey with yellow text.** It appears on yellow bands only, for example the homepage "Job search · Shell station locator · Shell Card" band.
- **Hover inverts the colours.** A yellow button turns dark grey (ink at 60 %) with yellow text (measured on shell.com).
- **Grey `#F5F5F5` replaces borders and shadows.** Cards are grey on white with 16 px corners, no border and no shadow. Only one hover shadow was found on the whole homepage.
- **Red is not an interface colour** on corporate, consumer or business pages. It appears in the pecten, in photographs of uniforms and forecourts, and as a secondary button colour on the careers page only.

### 3.5 Which hex values are "official"?

| Colour | Live CSS on Shell's sites (Oct 2026) | Commonly quoted elsewhere | Status |
|---|---|---|---|
| Shell yellow | `#FFC800` (`hsl(47,100%,50%)`) | `#FBCE07` (still used on [Fleet Hub](https://fleethub.shell.com/)); third-party sites also give `#FFD500` and `#FFCD00` | Use `#FFC800` provisionally, to match today's websites |
| Shell red | `#DD1D20` (`hsl(359,77%,49%)`) | `#DD1D21`; third-party sites also give `#DA291C`, `#ED1C24` | `#DD1D21` and `#DD1D20` are visually identical (1 unit of blue). Use `#DD1D21` provisionally |

Third-party "brand colour" sites disagree with each other (for example [brandpalettes.com](https://brandpalettes.com/shell-logo-colors/)), so we did not rely on them. **The official values must come from Shell's brand portal** (§10.7).

---

## 4. Logo (pecten): what Shell asks of partners

We will **not** copy logo files. Here is what Shell publishes about using its marks.

1. **Consent is mandatory and renewed every year.** "The Shell name and 'Pecten' logo are registered trademarks that require consent for use. Shell has a mandatory process for consent of trademark use. These processes must be completed annually for each organization." The steps are: request access to Shell's content manager through **Brand Central**, which the partner's **Shell Program Manager must approve**. Then fill in the Standard Trademark Consent Form and sign a DocuSign agreement. "Once countersigned by Shell, you will receive Shell logo files." Shell suggests asking for one year of consent and being ready to list "assets where the Shell logo will be used". ([Shell TM Consent & Artwork Approval Process, Shell USA, January 2025](https://www.shell.us/about-us/projects-and-locations/_jcr_content/par/textimage.stream/1713545199865/2a7400272d635be4d99f89c8d71deb1c80104f1c/shell-brand-approval-process-external.pdf), page 1.)
2. **Every piece of artwork with the logo needs approval.** "Shell has a mandatory process for approval of any artwork that includes the Shell logo. This process must be completed annually for each organization and each creative." Align the design with the Shell program manager first, then send it for final approval to **`Askbrand@shell.com`**, copying the program manager. (Same PDF, page 2.)
3. **The website terms forbid copying or altering the marks.** The site "contains trade marks, including the mark 'Shell' and the Shell emblem". Visitors "may not copy … distribute … alter or tamper with in any way or otherwise use any material … without our express written consent" ([terms of use](https://www.shell.com/terms-of-use.html)).
4. **Brand licensing is a commercial programme.** Shell offers brand licensing to selected business partners ([Shell Brand Licensing](https://www.shell.com/business-customers/shell-brand-licensing.html)). Licensees get the brand through a contract, not by default.
5. **Partner rules reported but not verified.** A 2018 Shell visual-identity document hosted by the Finnish fuel retailer ST1 ([PDF](https://content.st1.fi/sites/default/files/2018-06/b48eac65-ce9f-4447-8859-88ff4ffddc18.pdf)) reportedly says three things: use by partners, agencies and suppliers must be approved in advance by Shell Global Brand and Communications; poor reproduction "will weaken the visual strength of the Shell brand"; and no "TM" should be placed near the pecten. The file returned an error when we tried to open it, so treat these points as unconfirmed.

**How Shell itself places the pecten on its websites.** It shows the symbol only, without the "Shell" word, top-left in a 64 px square that links to the home page, in full colour on both the light and dark backgrounds. Nothing sits on top of it. The page name ("Shell Global", "shell.co.uk") is given as accessible text, not as a visible wordmark.

**For NorthStar:** show no pecten, no "Shell"-styled wordmark and no red-and-yellow shell shapes (including look-alike icons or striped red and yellow livery) until Shell gives consent and artwork approval. Keep a neutral NorthStar name mark in the logo slot. The word "Shell" may appear as plain text where it describes the business ("Shell attributes"), which is how NorthStar already uses it.

---

## 5. Fonts the sites load

| Font | Where | Proprietary? | Evidence |
|---|---|---|---|
| **"ShellHeavy"**, served under the alias `custom-font`, weight 700 | All headings (H1 to H3, card titles, promo headlines) on every shell.com / .co.uk / .us / .nl page | **Yes.** The font file's name table reads "Copyright (URW), Copyright 2018 by URW Type Foundry GmbH. All rights reserved", version 1.007, vendor URW. It is served from Shell's own servers only. | [`shell-typeface-la-heavy-221208.woff2`](https://www.shell.com/etc.clientlibs/amidala/clientlibs/theme-base/resources/fonts/shell-typeface-la-heavy-221208.woff2), declared in the theme stylesheet as `@font-face{font-family:'custom-font'; font-weight:700; …}` |
| **Arial**, with fallbacks `Roboto, Helvetica, sans-serif` | All body text, menus, buttons, inputs, footers | **No.** These are system fonts (Arial on Windows and Mac, Roboto on Android). | Shared token `--font-default: Arial, Roboto, Helvetica, sans-serif`; headings use `font-family: var(--font-custom), var(--font-default)` |
| "Shell-Bold" | Fleet Hub portal only | Yes (Shell font, own CDN) | [fleethub.shell.com](https://fleethub.shell.com/) |
| Roboto (Google Fonts) | Requested by embedded components on careers, EV charging and Fleet Hub | No (open licence) | Request to `fonts.gstatic.com/s/roboto/v48/…` |

**Only one weight of the Shell typeface (Heavy) is used on the web.** Body text is not set in a Shell font. The 1971 "specially designed typeface" for the Shell name ([brand history](https://www.shell.com/who-we-are/our-history/our-brand-history.html)) is a separate logotype and is not a web font.

**Type scale on the sites.** The shared library defines sizes of 11, 12, **14** (body), 16, 18, **20** (card titles, H3), **28** (section titles, H2), **32** (page titles, H1), 36, 40, 44, 60 and 68 px, all with line height 1.5. Most body text on every page is 14 px Arial regular; bold 14 px is used for buttons.

Typography is studied in more depth in its own report. Our conclusion for the brand: **NorthStar may use Shell's own body stack (Arial first) freely, but may not use the Shell typeface without a licence from Shell.**

---

## 6. Imagery and iconography

- **Photography shows real people at work and on the move.** Engineers and operators in PPE on LNG and offshore sites, drivers at the wheel, colleagues at desks, people charging EVs. The scenes are diverse, in natural light, with warm colour. Yellow and red appear naturally (hard hats, uniforms, forecourt canopies) rather than as overlays. Every image has descriptive alt text, for example "Digital visualisation of the Sparta offshore oil and gas platform showing a large floating structure at sea supported by four yellow columns…" ([shell.com](https://www.shell.com/)).
- **Photos sit in cards with rounded 16 px corners.** Hero text sits on a dark see-through panel (black at 60 %) over the photograph, with white text.
- **Icons are simple outline (line) icons in the ink colour.** They are 24 px in the header (fuel pump, EV charger, help, search), 12 px for chevrons, and arrows after "Read more". On consumer pages some icons sit in 64 px yellow circles.
- **Graphic devices appear on consumer and marketing pages only.** Red and white "+" sparkles on Shell GO+ panels ([GO+](https://www.shell.co.uk/shell-go-plus.html)); chevron-shaped photo crops and yellow halftone dots ([Fleet Solutions](https://www.shell.co.uk/business/shell-fleet-solutions.html)).
- **Corporate and legal pages carry little or no imagery** ([terms](https://www.shell.com/terms-of-use.html), [newsroom](https://www.shell.com/news-and-insights/newsroom.html) list cards with a standard forecourt image).

**For NorthStar:** working screens need no photography. One outline icon set at 20 to 24 px in the ink colour matches Shell's style. Shell photography, if wanted on the sign-in screen, must come from Shell's licensed image library (§10.7). The consumer graphic devices (sparkles, halftone dots, chevrons) do not belong in an enterprise tool.

---

## 7. Brand-carrying patterns on Shell's sites

These are the patterns that make a page feel "Shell". Full layout and component research is in the sibling reports.

| Pattern | Shell's sites (measured) |
|---|---|
| **Header** | 48 px grey utility bar (audience switch "Customer · Business · About us" on country sites, a language or site picker, and a yellow "Sign in" button 32 px tall with 6 px corners) above a 64 px white menu bar with the pecten at left and icon buttons at right. A hairline separates the header from the page. |
| **Current place** | 4 px yellow bar under the current menu item or tab. The text stays ink-coloured. |
| **Main button** | Yellow fill, ink text, Arial bold 14 px, 48 px tall, 12 px side padding, 8 px corners. Hover inverts to dark grey with yellow text. |
| **Other buttons** | Dark grey with yellow text (on yellow bands); red with white text (careers only); outlined or plain-text links with arrows ("Read more →"). |
| **Cards** | Grey `#F5F5F5`, 16 px corners, no border, no shadow. Image on top, 20 px Shell Heavy title, 14 px Arial text, link bottom-right. |
| **Inputs** | 48 px tall, 8 px corners, 1 px ink border at 60 %, 12 px padding, faint inner shadow ([newsroom search](https://www.shell.com/news-and-insights/newsroom.html)). |
| **Chips / tags** | Small grey tag with 4 px corners ("Media Relations"). |
| **Links in text** | Ink colour, underlined, often bold. |
| **Page grid** | 12 columns; content width 1300 px (menu 1400 px); three-card rows of 412 px with 20 px gaps; breakpoints at 600 px and 960 px; 4 px spacing unit; animations of 250 ms. |
| **Corners** | Only a few radii: 16 px (cards, panels), 8 px (buttons, inputs, skip link), 6 px (small button), 4 px (chips), 50 % (icon circles). No pill shapes. |
| **Accessibility features** | A "Skip to main content" link (yellow, appears on focus); one `main`, `header`, `nav` and `footer` landmark each; breadcrumbs; dark mode. **Focus uses the browser's default thin outline** (measured `outline: auto 1px`), which is weak. |
| **Footer** | White, "Page last updated" date, three link columns (Explore, Can we help?, Connect with us), legal links and cookie preferences. |

---

## 8. Consumer, corporate and business pages compared

| | Consumer: [shell.co.uk](https://www.shell.co.uk/), [GO+](https://www.shell.co.uk/shell-go-plus.html) | Corporate: [shell.com](https://www.shell.com/), [newsroom](https://www.shell.com/news-and-insights/newsroom.html), [investors](https://www.shell.com/investors.html) | Business: [UK business](https://www.shell.co.uk/business.html), [Fleet Solutions](https://www.shell.co.uk/business/shell-fleet-solutions.html), [Lubricants](https://www.shell.com/business-customers/lubricants-for-business.html) |
|---|---|---|---|
| Amount of yellow | High: large promo panels, icon circles, app-download blocks | Low: one button per hero, one help band | Low to medium: buttons, one headline panel, contact band |
| Red | In photographs (Ferrari partnership) | None in the interface | None in the interface |
| Imagery | Lifestyle, app mock-ups, offers with asterisks | Industrial sites, people at work, reports | Vehicles, fleets, industrial customers, case studies |
| Graphic devices | "+" sparkles | None | Chevrons, halftone dots (Fleet) |
| Content | Offers, rewards, "Sign in", chat button | News lists, reports, filings, dates | Problems and solutions, "Contact us", tools ("Find the right lubricant", "Shell Fuel Card Tools", "Shell APIs") |
| Menu | Products and services | Who we are · What we do · News · Business customers · Investors · Careers | Fleet Solutions · Lubricants · Oil and Gas · Sectors · Shell Energy |
| Sub-brand accent | – | – | Shell Energy teal-blue `#0080A3`; Recharge teal `#008192` |
| Customer portal | – | – | Fleet Hub: older style (square corners, dark-blue button, upper case) |

**Pattern:** the more serious the audience, the less brand colour. Corporate and business pages look like a calm white page with grey ink and yellow only where you act. That is the right model for an internal planning tool used all day by marketing and finance staff. **Fleet Hub, Shell's closest public example of a logged-in business tool, looks dated next to the current websites.** NorthStar should follow the current web system, not Fleet Hub.

---

## 9. Accessibility and contrast evidence

**Shell's public commitment** ([Accessibility, shell.com](https://www.shell.com/accessibility.html)):

- "following W3C's WCAG 2.2 A and AA standards and guidelines"
- "Zoom in up to 400% without text overlapping or losing clarity for most content"
- "If you would like to use dark mode when visiting Shell sites this can be done by setting this as your preference in your system or browser settings."
- The statement covers "our internal and external sites", which includes internal tools like NorthStar.

NorthStar's design system currently targets WCAG **2.1** AA (`docs/product/design-system.md` §9). **We recommend moving to WCAG 2.2 AA** to match Shell. WCAG 2.2 adds rules for the focused item not being hidden, minimum target size and dragging alternatives.

**Contrast of Shell's own colours** (WCAG 2.x ratios; AA needs 4.5 : 1 for normal text and 3 : 1 for large text, icons and control borders):

| Colour | on white | on `#F5F5F5` | on dark `#1D232A` | Verdict |
|---|---|---|---|---|
| Ink `#4A4A4A` | 8.86 | 8.13 | 1.79 | Text on light |
| Yellow `#FFC800` | **1.55** | 1.43 | 10.19 | **Fill only on light; fine as text or indicator on dark** |
| Ink on yellow | 5.70 | – | – | Text on yellow buttons passes |
| Red `#DD1D21` | 4.92 | **4.51** | 3.22 | White text on red passes; red text on grey is borderline |
| Blue `#336094` | 6.48 | 5.94 | 2.44 | Text on light |
| Green `#008557` | 4.67 | **4.29** | 3.39 | Text on white only; fails on grey |
| Purple `#87217D` | 8.26 | 7.57 | 1.92 | Text on light |
| Teal `#008192` | 4.61 | **4.23** | 3.44 | Text on white only |
| Navy `#092949` | 14.74 | 13.52 | 1.07 | Text, focus ring |

**Weak spots on Shell's sites that NorthStar should not copy:**

- The current-page bar is yellow on white (1.55 : 1). It is acceptable only because the menu text is still readable; it must never be the only sign of state.
- The input border (ink at 60 %, about 3.1 : 1) barely passes.
- The focus indicator is the browser's default 1 px outline.
- Body text at 14 px is small for long reading.

---

## 10. Recommendations for NorthStar

### 10.1 Direction

**"Shell corporate, not Shell consumer."** NorthStar stays white, calm and simple, as the product owner asked (design system §1). We change the accent from Apple blue to Shell's system:

- **Ink grey** for all text and icons.
- **White and `#F5F5F5` grey** for surfaces.
- **Shell yellow** for the one main action on a screen and for "you are here".
- **Shell red** only for danger, errors and money problems.
- **Shell's secondary colours** (blue, green, purple, teal) only where colour carries meaning: planning levels, statuses, teams, charts.
- **No gradients**, no photography in working screens, no logo until Shell approves it.

This replaces the visual part of decision D-005 (ZenReach look). ZenReach's layout and interaction patterns can stay. **This is a decision for the product owner to log** (§10.8).

### 10.2 Proposed tokens: light mode

Token names match `frontend/src/styles/tokens.css`, so the change is a value swap plus a few new tokens. All ratios are computed, not estimated.

| Token | Today | Proposed | Use | Contrast (AA check) |
|---|---|---|---|---|
| `--page-bg` | `#ffffff` | `#FFFFFF` | Page | – |
| `--surface` | `#ffffff` | `#FFFFFF` | Cards, dialogs | – |
| `--gray-bg` | `#f5f5f7` | **`#F5F5F5`** (Shell paper2) | Sidebar, table header, grey cards, bands | – |
| `--ink` | `#1d1d1f` | **`#404040`** | Main text and icons. Slightly darker than Shell's `#4A4A4A` for dense tables; Shell itself uses `#404040` on careers and Fleet Hub | 10.37 white · 9.51 grey ✔ |
| `--ink-2` | `#6e6e73` | **`#595959`** | Secondary text | 7.00 · 6.42 ✔ |
| `--muted` | `#86868b` (3.6, decoration only) | **`#6E6E6E`** | Hints, placeholders; **now also safe for small text** | 5.10 · 4.68 ✔ |
| `--hairline` | `#d2d2d7` | **`rgba(64,64,64,.2)`** (≈ `#D9D9D9`) | Dividers (decorative) | 1.41: decoration only |
| `--control-border` *(new)* | (hairline used) | **`#8C8C8C`** | Input, select and checkbox borders | 3.36 · 3.08 ✔ (≥ 3 : 1) |
| `--brand-yellow` *(new; replaces `--button-primary-bg` `#0071e3`)* | – | **`#FFC800`** | Main-button fill, current-place bar, focus fill of the skip link | 1.55: **fill only** |
| `--brand-yellow-hover` / `-pressed` *(new)* | `#0062c4` | **`#F2B800`** / **`#E0A800`** | Button hover and pressed | ink text 9.34 / 7.84 ✔ |
| `--on-yellow` *(new; replaces `--on-accent` for the main button)* | `#ffffff` | **`#404040`** (= ink) | Text and icons on yellow | 6.67 ✔ |
| `--brand-yellow-tint` *(new)* | `--banner-bg #fff4ce` | **`#FFF4CC`** | Selected menu row, selected table row, demo banner | ink 9.41 ✔ |
| `--danger-bg` | `#d03b3b` | **`#DD1D21`** (Shell red) | Delete buttons, error icons, over-budget bars | white text 4.92 ✔ |
| `--danger-bg-hover` *(new)* | – | **`#B8171B`** | Delete hover | white 6.61 ✔ |
| `--link` | `#0066cc` | **`#404040` + underline** | Shell links are ink, underlined, not blue. Record names in tables: ink, semi-bold, underline on hover and focus | 10.37 ✔ |
| `--blue` (focus, selection, eyebrows) | `#0071e3` | **retire**; focus becomes `--focus-ring` | – | – |
| `--focus-ring` *(new)* | `--blue` | **`#092949`** (Shell system navy), 2 px with 2 px white offset | Keyboard focus on every control | 14.74 white · 9.48 against yellow ✔ |
| `--brand-gradient`, `--brand-gradient-strong` | blue to purple | **retire** | Avatars become initials on `--gray-bg` with ink text, or white on a level colour; progress bars become solid colours | – |
| `--status-neutral` / `-text` | `#86868b` / `#636366` | `#6E6E6E` / **`#595959`** | Draft, Not started | text 7.00 · 6.42 · on own tint 5.88 ✔ |
| `--status-review` / `-text` | `#c77700` / `#8f5400` | `#C77700` / **`#8F5B00`** | Submitted, In review. **Amber, deliberately different from brand yellow** | dot 3.46 ✔ · text 5.73 · 5.26 · tint 4.84 ✔ |
| `--status-positive` / `-text` | `#0ca30c` / `#077307` | **`#008557`** (Shell green) / **`#00704A`** | Published, Approved | dot 4.67 ✔ · text 6.15 · 5.64 · tint 5.14 ✔ |
| `--status-active` / `-text` | `#0071e3` / `#0060c0` | **`#336094`** (Shell blue) / `#336094` | In progress, Running | 6.48 · 5.94 · tint 5.45 ✔ |
| `--status-done` / `-text` | `#5b6b7a` / `#4d5c6a` | `#5B6B7A` / `#4D5C6A` (unchanged) | Complete | 6.88 · 6.31 · tint 5.76 ✔ |
| `--status-problem` / `-text` | `#d03b3b` / `#b42c2c` | **`#DD1D21`** / **`#B8171B`** | Declined, Incomplete, negative money | dot 4.92 ✔ · text 6.61 · 6.06 · tint 5.36 ✔ |
| `--level-plan` | `#2a78d6` | **`#336094`** (Shell blue) | Level bar and tag | 6.48 ✔ |
| `--level-campaign` | `#4a3aa7` | **`#87217D`** (Shell purple) | | 8.26 ✔ |
| `--level-initiative` | `#1baf7a` | **`#008557`** (Shell green) | | 4.67 ✔ |
| `--level-activity` | `#e87ba4` | **`#008192`** (Shell Recharge teal) | | 4.61 ✔ |
| `--level-line-item` | `#86868b` | **`#6E6E6E`** | | 5.10 ✔ |
| `--overlay` | `rgba(0,0,0,.3)` | `rgba(0,0,0,.4)` | Behind dialogs (Shell uses black at 60 % behind hero text) | – |
| `--tooltip-bg` | `#1d1d1f` | **`#262626`** | Tooltip | white 15.13 ✔ |

"Tint" means the colour at 12 % over white, as used for status chips today. Every chip text token passes 4.5 : 1 on its own tint.

### 10.3 Proposed tokens: dark mode (built on Shell's own dark palette)

| Token | Proposed dark value | Source | Contrast |
|---|---|---|---|
| `--page-bg` | `#14191F` | Shell `paper2` dark | – |
| `--surface` | `#1D232A` | Shell `paper1` dark | – |
| `--gray-bg` | `#232A32` | Derived | ink 10.98 · ink-2 6.61 · muted 5.00 ✔ |
| `--ink` | `#E0E0E0` | Shell `ink1` dark | 13.38 page · 12.00 surface ✔ |
| `--ink-2` | `#A8B0B9` | Derived | 8.06 · 7.22 ✔ |
| `--muted` | `#8E99A4` | Derived | 6.09 · 5.46 ✔ |
| `--hairline` | `#333B44` | Derived | decoration |
| `--control-border` | `#6B7682` | Derived | 3.82 · 3.42 ✔ |
| `--brand-yellow` | `#D6A100` | Shell `brand1` dark (dimmed) | – |
| `--on-yellow` | `#1D232A` | Shell `brand1-text` dark | 6.75 ✔ |
| `--focus-ring` | `#FFC800` | Shell yellow | 11.37 · 10.19 ✔ |
| `--danger-bg` | `#B82E00` | Shell `brand2` dark (orange-red) | white 6.12 ✔ |
| `--brand-yellow-tint` | yellow at 16 % over surface (≈ `#413D23`) | Derived | ink 8.30 ✔ |
| Status text: positive / review / problem / active / done | `#4CC79A` / `#F0A640` / `#FF7B7E` / `#8FB4E3` / `#A9B8C6` | Derived | all ≥ 6.3 on surface and ≥ 4.8 on their own tint ✔ |
| Levels: plan / campaign / initiative / activity / line item | `#6F9BD1` / `#C96BBF` / `#3DBB8C` / `#3FB6C5` / `#8E99A4` | Lightened Shell secondaries | all ≥ 4.78 on surface ✔ |

Keep the current behaviour: follow the computer's setting first, then remember the user's choice (D-038). This is also what Shell's accessibility statement promises.

### 10.4 What to use the brand colours for, and what not

| Colour | Use it for | Never use it for |
|---|---|---|
| **Shell yellow** | The **one** main button per screen (ink text); the current-place marker (a 4 px bar on the sidebar item or tab, **together with** bold text and the yellow tint); the skip link; the demo banner tint; the focus ring in dark mode | Text or icons on white (1.55 : 1); thin lines or borders as the only signal; warnings (use amber, so "brand" and "warning" never look alike); large yellow areas behind tables or forms; chart series on white; more than one yellow button per screen |
| **Shell red** | Delete and other destructive buttons; error messages and icons; Declined and Incomplete statuses; over-budget and negative money | Decoration, headings, main actions or brand accents; red and yellow together in stripes, gradients or shell shapes (these imitate Shell's protected trade dress) |
| **Shell blue, green, purple, teal** | Meaning only: planning levels, statuses, team dots, charts (always next to a word) | Buttons, headings, backgrounds of whole areas |
| **Navy `#092949`** | Keyboard focus ring in light mode; optional dark tooltip or system banner | Large areas (it reads as "Fleet Hub-era" Shell blue) |
| **Ink and greys** | Everything else: text, icons, links (underlined), borders, secondary buttons (outlined in ink) | – |

### 10.5 Typography, shape and motion tokens

| Token | Proposed | Why |
|---|---|---|
| `--font` (body) | **`Arial, "Segoe UI", Roboto, Helvetica, sans-serif`** | Shell's own body stack (Arial first), free to use. "Segoe UI" covers Shell's Windows desktops if Arial is missing. Arial digits are tabular, which suits money columns. |
| `--font-display` *(new)* | `var(--font)` at weight 700 **for now** | Reserved for Shell's heading typeface once Shell licenses it (§10.7). Headings become bold Arial in the meantime. |
| Base size | **14 px** for interface text; **16 px** for help text and long reading | Matches Shell's 14 px body while keeping reading text comfortable |
| Heading sizes | Page title 28 px, section 20 px, card title 16 px, all bold, line height 1.3 to 1.5 | Shell's scale: 32 / 28 / 20 |
| Corners | Cards **16 px** (unchanged); buttons and inputs **8 px** (was 10); chips **4 px** (was 8); **no pill shapes** (`--radius-pill` used only for avatars and toggles) | Shell uses 16 / 8 / 4 and no pills |
| Control height | 40 px default, 32 px compact (tables), 48 px on touch | Shell's 48 px buttons suit marketing pages; a dense tool needs 40 / 32. All stay above WCAG 2.2's 24 px minimum target |
| Spacing | 4 px grid (unchanged) | Same as Shell |
| Motion | 200 to 250 ms; none with reduced motion | Shell uses 250 ms |
| Shadows | Keep only `--shadow-md` for floating layers (menus, dialogs). Cards get no shadow: grey fill or a 1 px border instead | Shell cards have no shadow |

### 10.6 Logo and naming

- Keep the logo slot top-left with a **neutral NorthStar name mark** in ink. No pecten, no yellow-and-red shell shape, no "Shell" in a Shell typeface.
- Use the browser tab title "NorthStar". Do not use the pecten as favicon.
- When Shell approves its use, place the pecten exactly as Shell's sites do (symbol only, full colour, top-left, linking home) and follow the clear-space and minimum-size rules Shell supplies.

### 10.7 What must wait for official Shell assets

| Item | Why it waits | Who to ask |
|---|---|---|
| **Pecten files and permission to show them in NorthStar** | Registered trade mark; yearly consent and artwork approval required | Shell program manager → Brand Central consent; artwork to `Askbrand@shell.com` ([Shell process PDF](https://www.shell.us/about-us/projects-and-locations/_jcr_content/par/textimage.stream/1713545199865/2a7400272d635be4d99f89c8d71deb1c80104f1c/shell-brand-approval-process-external.pdf)) |
| **Shell typeface (Heavy, plus any text weights) and a licence for an internal app** | Font copyright URW, all rights reserved | Shell brand team via Brand Central |
| **Official screen colour values** (confirm `#FFC800` vs `#FBCE07`, `#DD1D21`) and secondary palette | Only the live CSS is public; official specifications are behind sign-in | Shell brand team |
| **Shell's internal design system ("Shell DS")**: tokens, React components, icon set | It may be mandatory for internal Shell applications. If so, NorthStar should use its tokens instead of ours | Shell digital or IT contact for internal apps |
| **Icon library, illustration and photography** | Shell images are copyright; consumer graphic devices do not suit a tool | Shell brand team |
| **Rules for internal tool names and co-branding** (is "NorthStar" acceptable? any vendor credit?) | Not public | Shell program manager |
| **Shell's internal accessibility standard** for employee applications | Public statement says WCAG 2.2 AA; internal tools may have a checklist | Shell digital accessibility contact |
| **Tone-of-voice guide** | None is public; NorthStar's words rules already match the observed tone | Shell brand team (nice to have) |

Until these arrive, the token set in §10.2 and §10.3 is a **provisional, Shell-aligned** look that uses no protected asset.

### 10.8 Decisions to record (for the product owner)

1. **NorthStar adopts Shell's web colour system** (yellow action, ink text, red for danger only, Shell secondaries for meaning). This replaces ZenReach's blue and gradient. ZenReach layout and interaction patterns stay. This changes the visual part of D-005.
2. **Accessibility target moves from WCAG 2.1 AA to WCAG 2.2 AA**, matching [Shell's statement](https://www.shell.com/accessibility.html).
3. **No Shell logo or typeface until Shell grants consent.** A neutral NorthStar name mark is used; `--font-display` stays on Arial bold.
4. **Provisional brand values** are `#FFC800` (yellow) and `#DD1D21` (red), to be confirmed by Shell (new open question).
5. **Ask Shell whether its internal design system is mandatory** for NorthStar (new open question). If it is, it replaces items 1 and 4.

---

## Appendix A: Evidence notes

- **Token extraction.** Light-mode tokens come from the shared bundle `app.min.ACSHASHc4fb4841e466a60a7ed3479e9efcaf79.js`; dark-mode tokens and the `@font-face` rule come from `theme-base.ACSHASHb291465fc8d3c7bcefde74d34044b698.css`. Both were served by www.shell.com. We confirmed the same computed values on shell.co.uk, shell.us and shell.nl pages, with the sub-brand override noted in §3.3.
- **The shared library also defines** `--font-default: Arial, Roboto, Helvetica, sans-serif`, `--font-custom: custom-font`, `--page-width: 1300px`, `--nav-width: 1400px`, `--anim-time: 250ms`, `--font-scale: 1` (1.15 for Arabic, Japanese, Korean, Thai and Chinese), radius steps 3 / 4 / 6 / 8 / 12 / 16 / 24 px, and breakpoints below 600 px, 600 to 959 px and 960 px and above.
- **Colour counts in §3.4** count visible elements by computed `color` and `background-color` after the cookie banner was closed and the page was scrolled to load lazy content.
- **Hover and focus** were measured on shell.com, shell.co.uk, UK business, UK EV charging and careers, by hovering the first "Read more" button and pressing Tab six times.
- **Font metadata** (§5) was read from the font file's `name` table: family "ShellHeavy", version 1.007, manufacturer "URW Type Foundry GmbH", notice "Copyright (URW), Copyright 2018 by URW Type Foundry GmbH. All rights reserved." The local copy was deleted after reading.
- **Contrast ratios** were computed with the WCAG 2.x relative-luminance formula. Tints are the colour at 12 % (light) or 16 % (dark) alpha over the background.

## Appendix B: Sources

Shell (primary)

- Shell Global home: <https://www.shell.com/>
- Shell UK home: <https://www.shell.co.uk/>
- Shell USA home: <https://www.shell.us/>
- Careers: <https://www.shell.com/careers.html>
- Newsroom: <https://www.shell.com/news-and-insights/newsroom.html>
- Investors: <https://www.shell.com/investors.html>
- Our brand history: <https://www.shell.com/who-we-are/our-history/our-brand-history.html>
- Who we are (UK): <https://www.shell.co.uk/about-us/who-we-are.html>
- Terms of use (copyright and trade marks): <https://www.shell.com/terms-of-use.html>
- Accessibility statement: <https://www.shell.com/accessibility.html>
- Business customers: <https://www.shell.com/business-customers.html>
- Lubricants for business: <https://www.shell.com/business-customers/lubricants-for-business.html>
- Shell Brand Licensing: <https://www.shell.com/business-customers/shell-brand-licensing.html>
- UK Business: <https://www.shell.co.uk/business.html>
- Shell Fleet Solutions (UK): <https://www.shell.co.uk/business/shell-fleet-solutions.html>
- EV charging / Shell Recharge (UK): <https://www.shell.co.uk/electric-vehicle-charging.html>
- Shell GO+ Rewards (UK): <https://www.shell.co.uk/shell-go-plus.html>
- Shell Recharge (redirects to shell.nl): <https://shellrecharge.com/>
- Shell Energy US business (from shellenergy.com): <https://www.shell.us/business/shell-energy.html>
- Shell Fleet Hub: <https://fleethub.shell.com/>
- Brand Central (sign-in): <https://brandcentral.shell.com/>
- Shell TM Consent & Artwork Approval Process (Shell USA, January 2025): <https://www.shell.us/about-us/projects-and-locations/_jcr_content/par/textimage.stream/1713545199865/2a7400272d635be4d99f89c8d71deb1c80104f1c/shell-brand-approval-process-external.pdf>
- Shared theme CSS: <https://www.shell.com/etc.clientlibs/amidala/clientlibs/theme-base.ACSHASHb291465fc8d3c7bcefde74d34044b698.css>
- Shell Recharge theme CSS: <https://www.shell.nl/etc.clientlibs/amidala/clientlibs/theme-shell-recharge.ACSHASH23a752d535925afba7aeea6dc8e3b8f7.css>
- Shared front-end bundle: <https://www.shell.com/etc.clientlibs/amidala/clientlibs/app.min.ACSHASHc4fb4841e466a60a7ed3479e9efcaf79.js>
- Heading font file (read for metadata only): <https://www.shell.com/etc.clientlibs/amidala/clientlibs/theme-base/resources/fonts/shell-typeface-la-heavy-221208.woff2>
- Shell internal design-system page (sign-in, not opened): <https://realtime.shell.com/RTATD/AllProjects/Devkit>

Secondary (context only)

- Shell sells UK and German home energy to Octopus (S&P Global): <https://www.spglobal.com/commodity-insights/en/news-research/latest-news/electric-power/090123-shell-sells-uk-german-home-energy-businesses-to-octopus-energy>
- Shell Energy Retail becomes Octopus Energy Operations 2: <https://www.evinfrastructurenews.com/public-charging/shell-energy-retail-becomes-octopus-energy-operations-2-as-acquisition-completes>
- ST1-hosted Shell identity document (could not be opened; points in §4 item 5 unverified): <https://content.st1.fi/sites/default/files/2018-06/b48eac65-ce9f-4447-8859-88ff4ffddc18.pdf>
- Third-party colour listing, shown only as an example of conflicting values: <https://brandpalettes.com/shell-logo-colors/>

NorthStar files referred to

- `docs/product/design-system.md` §1, §3, §4, §8, §9 (current design language)
- `frontend/src/styles/tokens.css:9-142` (current token values)
- `docs/sdd/decision-log.md` D-005 (ZenReach look), D-038 (theme follows the computer, then the user's choice)
