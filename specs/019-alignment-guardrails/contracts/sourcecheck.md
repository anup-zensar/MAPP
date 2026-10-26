# Contract: `sourcecheck`

## Command

```sh
go -C backend run ./cmd/sourcecheck [-root DIR] [-tighten] [-seed]
```

| Flag | Default | Meaning |
|---|---|---|
| `-root` | the Git repository's top level | The repository to check |
| `-tighten` | off | After checking, lower baseline counts to current values and drop entries now within limits; never raise or add |
| `-seed` | off | Write `quality/baseline.json` from the current tree; refuses if it exists |

## Files checked

Tracked and new, not ignored, files (`git ls-files --cached --others --exclude-standard`) ending in
`.go`, `.ts`, `.tsx`, `.css` or `.sql`, except generated files: a Go header
`// Code generated … DO NOT EDIT.`, the openapi-typescript header, or a path under
`frontend/src/api/schema.d.ts` or `backend/internal/platform/database/sqlcdb/`.

## Rules

- **Size**: non-blank lines. ≤ 300 passes; 301–400 passes only with an exception whose `lines` ≥
  the count; ≥ 401 fails unless the file is in the baseline at a count ≥ the current one.
- **Comments**: lines holding a comment, directives excluded (`//go:`, `//nolint`,
  `eslint-disable`, `eslint-enable`, `prettier-ignore`, `@ts-expect-error`, `/// <reference`,
  `-- +goose`, sqlc `-- name:`). Budget = ceil(5% × non-blank lines). Over budget fails unless the
  file is in the baseline at a count ≥ the current one.

## Output (stdout)

```text
frontend/src/features/plans/Big.tsx: 420 non-blank lines; limit 400
backend/internal/plans/new.go: 7 comment lines; budget 5 (5% of 100 lines)
backend/internal/allwork/allwork_source.go: 915 lines in baseline; can be tightened to 910
2 problems in 1,352 files; baseline: 126 entries (3 can be tightened)
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | No problems |
| 1 | At least one problem |
| 2 | Usage error, unreadable file, invalid JSON, or `-seed` with an existing baseline |

## File formats

`quality/baseline.json`:

```json
[
  { "path": "backend/internal/allwork/allwork_source.go", "metric": "lines", "count": 915 }
]
```

`quality/exceptions.json`:

```json
[
  { "path": "frontend/src/features/plans/PlanEditor.tsx", "lines": 340, "reason": "One form whose fields share validation state; splitting it would scatter that state." }
]
```

Both arrays are sorted by `path` (then `metric`), written with two-space indentation and a final
newline.
