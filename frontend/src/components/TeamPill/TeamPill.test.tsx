import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { TeamPill, TeamPills } from './TeamPill';

const EUROPE = { id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true } as const;
const CHINA = { id: 'T-1002', name: 'Mobility China', colour: 'teal', active: false } as const;

describe('TeamPill', () => {
  it('writes the name, with the colour as a decorative dot (D-025)', () => {
    renderWithProviders(<TeamPill team={EUROPE} />);

    expect(screen.getByText('Lubricants Europe')).toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('writes " (inactive)" out and links when asked', () => {
    renderWithProviders(<TeamPills teams={[EUROPE, CHINA]} linked />);

    expect(screen.getByRole('link', { name: 'Lubricants Europe' })).toHaveAttribute(
      'href',
      '/settings/teams/T-1001',
    );
    expect(screen.getByRole('link', { name: 'Mobility China (inactive)' })).toBeInTheDocument();
    expect(screen.getAllByRole('listitem')).toHaveLength(2);
  });
});
