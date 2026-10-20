import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { Button } from '../Button/Button';
import { LevelTag } from '../LevelTag/LevelTag';
import { PageHeader } from './PageHeader';

describe('PageHeader', () => {
  it('shows the eyebrow, the page title as the one main heading and the description (FR-010)', () => {
    renderWithProviders(
      <PageHeader
        eyebrow="Settings"
        title="Models"
        description="Create yearly marketing plans, give them budgets and publish them when they're ready."
      />,
    );

    expect(screen.getByRole('heading', { level: 1, name: 'Models' })).toBeInTheDocument();
    expect(screen.getByText('Settings')).toBeInTheDocument();
    expect(
      screen.getByText(
        "Create yearly marketing plans, give them budgets and publish them when they're ready.",
      ),
    ).toBeInTheDocument();
  });

  it('does not repeat an eyebrow that is the title itself (Shell look)', () => {
    renderWithProviders(<PageHeader eyebrow="Plans" title="Plans" />);

    expect(screen.getAllByText('Plans')).toHaveLength(1);
  });

  it('adds the highlight phrase to the title (FR-010)', () => {
    renderWithProviders(<PageHeader eyebrow="Home" title="Welcome back," highlight="Priya." />);

    expect(
      screen.getByRole('heading', { level: 1, name: 'Welcome back, Priya.' }),
    ).toBeInTheDocument();
  });

  it('shows the page actions next to the title (FR-010)', () => {
    renderWithProviders(
      <PageHeader
        eyebrow="Developer"
        title="Sample records"
        actions={<Button variant="primary">New sample folder</Button>}
      />,
    );

    expect(screen.getByRole('button', { name: 'New sample folder' })).toBeInTheDocument();
  });

  it('shows the breadcrumb above the title instead of the eyebrow (design system §6.3)', () => {
    renderWithProviders(
      <PageHeader
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Models' }]}
        eyebrow="Settings"
        title="Models"
      />,
    );

    const trail = screen.getByRole('navigation', { name: 'Breadcrumb' });
    expect(within(trail).getByRole('link', { name: 'Settings' })).toHaveAttribute(
      'href',
      '/settings',
    );
    // The breadcrumb names the section already, so the eyebrow is not repeated.
    expect(screen.getAllByText('Settings')).toHaveLength(1);
  });

  it('shows a level tag and the record’s facts with the title (design system §6.3)', () => {
    renderWithProviders(
      <PageHeader
        tag={<LevelTag level="plan" />}
        title="FY2026 Lubricants"
        meta={<span>P-1001</span>}
      />,
    );

    expect(screen.getByText('PLAN')).toBeInTheDocument();
    expect(screen.getByText('P-1001')).toBeInTheDocument();
  });

  it('puts less-used actions in a More menu (design system §6.3)', async () => {
    const onDelete = vi.fn();
    const { user } = renderWithProviders(
      <PageHeader
        title="FY2026 Lubricants"
        actions={<Button variant="primary">Edit</Button>}
        moreActions={[{ label: 'Delete', tone: 'danger', onSelect: onDelete }]}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'More' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));

    expect(onDelete).toHaveBeenCalledTimes(1);
  });
});
