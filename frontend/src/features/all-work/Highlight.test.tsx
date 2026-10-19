import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Highlight } from './Highlight';

function marks(text: string, search: string) {
  const { container } = render(
    <p>
      <Highlight text={text} search={search} />
    </p>,
  );
  return {
    text: container.textContent,
    marked: [...container.querySelectorAll('mark')].map((mark) => mark.textContent),
  };
}

describe('Highlight (FR-006)', () => {
  it('marks every place of the searched text, ignoring letter case', () => {
    expect(marks('Helix launch — HELIX social', ' helix ')).toEqual({
      text: 'Helix launch — HELIX social',
      marked: ['Helix', 'HELIX'],
    });
  });

  it('marks nothing without a search or a hit', () => {
    expect(marks('Fleet roadshow', '').marked).toEqual([]);
    expect(marks('Fleet roadshow', 'helix').marked).toEqual([]);
  });

  it('marks an ID', () => {
    expect(marks('A-1001', 'a-10').marked).toEqual(['A-10']);
  });
});
