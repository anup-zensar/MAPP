import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { LockedNote } from './LockedNote';

describe('LockedNote', () => {
  it('says why something is locked, with an id a control can point to (design system §6.6)', () => {
    render(
      <>
        <input aria-label="Code" aria-describedby="code-lock" readOnly value="EUR" />
        <LockedNote id="code-lock">The code can't change once the currency is saved.</LockedNote>
      </>,
    );

    expect(screen.getByRole('textbox', { name: 'Code' })).toHaveAccessibleDescription(
      "The code can't change once the currency is saved.",
    );
  });
});
