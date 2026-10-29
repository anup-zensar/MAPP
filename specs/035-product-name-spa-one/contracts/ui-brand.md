# UI Contract: the product name and its tile

## `Logo` (`frontend/src/ui/Logo/Logo.tsx`)

```tsx
<Logo size?: 'md' | 'lg' />
```

- Renders only the name — no symbol: `SP` `&` `A` ` ` `One` in `--font-heading`, weight
  `--weight-heading`; `md` = `--text-lg` (sidebar drawer, top bar ≥ 960 px, sign-in header), `lg` =
  `--text-xl`.
- Colours: "SP", "A" → `--ink`; "&" → `--brand-name-amp`; "One" → `--brand-name-one`.
- Accessibility: one element `role="img"` with `aria-label="SP&A One"`; a link around it is named
  "SP&A One" unless it has its own `aria-label`.
- The `withWordmark` prop is removed: the name is always shown.

## `LogoTile` (`frontend/src/ui/Logo/LogoTile.tsx`)

```tsx
<LogoTile />
```

- A 28 × 28 tile, radius `--radius-control`, fill `--brand-tile`, with a "1" path in `--on-brand-tile`;
  the same in both themes; identical artwork to `frontend/public/favicon.svg`.
- Decoration: `aria-hidden="true"`; the link around it carries the name ("SP&A One home").

## Where each appears

| Place | ≥ 960 px | < 960 px | Link name |
|---|---|---|---|
| Top bar (signed in) | `Logo` | `LogoTile` | `aria-label="SP&A One home"` |
| Menu drawer (sidebar in a drawer) | `Logo` | `Logo` | `aria-label="SP&A One home"` |
| Public header (sign-in, not found, full-page states) | `Logo` | `Logo` | — (not a link) |
| Browser tab | favicon = tile | favicon = tile | title "SP&A One" or "*page* · SP&A One" |
| Footer | "© SP&A One — demo data" | same | — |

## Tokens

See [data-model.md](../data-model.md#design-tokens-frontendsrcuitokenscss). `--brand-mark` is removed;
nothing else used it.
