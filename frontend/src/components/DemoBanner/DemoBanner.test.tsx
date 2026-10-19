import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DemoBanner } from './DemoBanner';

describe('DemoBanner', () => {
  it('says that sign-in and data are for demonstration only (FR-016)', () => {
    render(<DemoBanner />);

    expect(
      screen.getByText('Demo environment — sign-in and data are for demonstration only'),
    ).toBeVisible();
  });
});
