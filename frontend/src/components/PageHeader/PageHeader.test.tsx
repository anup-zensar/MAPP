import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Button } from '../Button/Button';
import { PageHeader } from './PageHeader';

describe('PageHeader', () => {
  it('shows the eyebrow, the page title as the one main heading and the description (FR-010)', () => {
    render(
      <PageHeader
        eyebrow="Plans"
        title="Plans"
        description="Create yearly marketing plans, give them budgets and publish them when they're ready."
      />,
    );

    expect(screen.getByRole('heading', { level: 1, name: 'Plans' })).toBeInTheDocument();
    expect(screen.getAllByText('Plans')).toHaveLength(2);
    expect(
      screen.getByText(
        "Create yearly marketing plans, give them budgets and publish them when they're ready.",
      ),
    ).toBeInTheDocument();
  });

  it('adds the highlight phrase to the title (FR-010)', () => {
    render(<PageHeader eyebrow="Home" title="Welcome back," highlight="Priya." />);

    expect(
      screen.getByRole('heading', { level: 1, name: 'Welcome back, Priya.' }),
    ).toBeInTheDocument();
  });

  it('shows the page actions next to the title (FR-010)', () => {
    render(
      <PageHeader
        eyebrow="Developer"
        title="Sample records"
        actions={<Button variant="primary">New sample folder</Button>}
      />,
    );

    expect(screen.getByRole('button', { name: 'New sample folder' })).toBeInTheDocument();
  });
});
