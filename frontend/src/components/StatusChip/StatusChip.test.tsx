import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { StatusChip } from './StatusChip';

describe('StatusChip', () => {
  it.each([
    ['neutral', 'Draft'],
    ['review', 'Submitted'],
    ['positive', 'Approved'],
    ['active', 'In progress'],
    ['done', 'Complete'],
    ['problem', 'Declined'],
  ] as const)(
    'always shows the status as a word, also in the %s colour (FR-019, constitution III)',
    (tone, word) => {
      render(<StatusChip tone={tone}>{word}</StatusChip>);

      expect(screen.getByText(word)).toBeVisible();
    },
  );
});
