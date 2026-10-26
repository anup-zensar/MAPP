import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { PersonLabel } from './PersonLabel';

const PRIYA = { name: 'Priya Raman', initials: 'PR', active: true, jobTitle: 'Marketer' };

describe('PersonLabel', () => {
  it('writes " (you)" and " (inactive)" out, never by colour alone (SC-003)', () => {
    render(<PersonLabel person={{ ...PRIYA, active: false }} you />);

    expect(screen.getByText('Priya Raman (you) (inactive)')).toBeInTheDocument();
  });
});
