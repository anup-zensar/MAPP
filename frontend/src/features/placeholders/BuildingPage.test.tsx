import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { BuildingPage, BuildingPanel, BuildingSection } from './BuildingPage';

describe('BuildingPage, BuildingPanel and BuildingSection', () => {
  it('shows the page’s title, description and breadcrumb, and that it is being built', () => {
    renderWithProviders(
      <BuildingPage
        eyebrow="Settings"
        title="Models"
        description="The rulebooks plans use."
        breadcrumb={[{ label: 'Settings', to: '/settings' }, { label: 'Models' }]}
      >
        <BuildingSection title="Line items" />
      </BuildingPage>,
    );

    expect(screen.getByRole('heading', { level: 1, name: 'Models' })).toBeInTheDocument();
    expect(screen.getByText('The rulebooks plans use.')).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toBeInTheDocument();
    expect(screen.getByText('This page is being built.')).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'Line items' })).toHaveTextContent(
      'This section is being built.',
    );
    expect(document.title).toBe('Models · MAPP');
  });

  it('names the tab differently when asked', () => {
    renderWithProviders(
      <BuildingPage eyebrow="My profile" title="Your profile" tabTitle="My profile" />,
    );

    expect(document.title).toBe('My profile · MAPP');
  });

  it('is a side panel that goes back to the page under it when closed', async () => {
    const { user, router } = renderWithProviders(
      <BuildingPanel title="New campaign" closeTo="/plans/P-1001" closeLabel="Back to the plan" />,
      { route: '/plans/P-1001/campaigns/new' },
    );

    expect(screen.getByRole('dialog', { name: 'New campaign' })).toHaveTextContent(
      'This page is being built.',
    );
    await user.click(screen.getByRole('button', { name: 'Back to the plan' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
  });
});
