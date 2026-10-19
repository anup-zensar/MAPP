import { screen, within } from '@testing-library/react';
import { Construction } from 'lucide-react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { SettingsPage } from './SettingsPage';
import type { SettingsSection } from './sections';

// Feature 002's five sections, then a section of a later feature that has no page yet.
vi.mock('./sections', async (importOriginal) => {
  const original = await importOriginal<typeof import('./sections')>();
  const later: SettingsSection = {
    title: 'Later section',
    description: 'A section of a later feature.',
    path: '/settings/later',
    icon: Construction,
    comingSoon: true,
  };
  return { ...original, SETTINGS_SECTIONS: [...original.SETTINGS_OWN_SECTIONS, later] };
});

const OVERVIEW = {
  models: 3,
  activityTypes: 12,
  shellAttributes: 41,
  currencies: 1,
  financeBudgetItems: 1_250,
};

function renderSettings() {
  return renderWithProviders(<SettingsPage />, { route: '/settings', user: testUser() });
}

function card(title: string): HTMLElement {
  const list = screen.getByRole('list', { name: 'Sections' });
  const item = within(list)
    .getAllByRole('listitem')
    .find((element) => within(element).queryByRole('heading', { name: title }));
  if (!item) {
    throw new Error(`No card "${title}"`);
  }
  return item;
}

describe('SettingsPage (FR-001)', () => {
  it('shows the page title, its promise and one card per section with its active count', async () => {
    mockApi({ 'GET /api/settings/overview': { body: OVERVIEW } });
    renderSettings();

    expect(screen.getByRole('heading', { level: 1, name: 'Settings' })).toBeInTheDocument();
    expect(
      screen.getByText(
        'Set up the lists and rules every plan uses. Changes here apply to everyone straight away.',
      ),
    ).toBeInTheDocument();
    expect(document.title).toBe('Settings · NorthStar');

    expect(await within(card('Models')).findByText('3 active')).toBeInTheDocument();
    expect(within(card('Activity types')).getByText('12 active')).toBeInTheDocument();
    expect(within(card('Shell attributes')).getByText('41 active')).toBeInTheDocument();
    expect(within(card('Currencies')).getByText('1 active')).toBeInTheDocument();
    expect(within(card('Finance budget items')).getByText('1,250 active')).toBeInTheDocument();
    expect(
      within(card('Models')).getByText(
        'The rulebooks plans use: conversion rates, average deal size and allowed activity types.',
      ),
    ).toBeInTheDocument();
  });

  it('opens each section from its card: the link is named by the section', async () => {
    mockApi({ 'GET /api/settings/overview': { body: OVERVIEW } });
    const { user, router } = renderSettings();

    expect(screen.getByRole('link', { name: 'Currencies' })).toHaveAttribute(
      'href',
      '/settings/currencies',
    );
    await user.click(screen.getByRole('link', { name: 'Models' }));

    expect(router.state.location.pathname).toBe('/settings/models');
  });

  it('shows a section without a page as Coming soon, not as a link', async () => {
    mockApi({ 'GET /api/settings/overview': { body: OVERVIEW } });
    renderSettings();

    const later = card('Later section');
    expect(within(later).getByText('Coming soon')).toBeInTheDocument();
    expect(within(later).queryByRole('link')).not.toBeInTheDocument();
    expect(await within(card('Models')).findByText('3 active')).toBeInTheDocument();
  });

  it('keeps the cards usable when the counts cannot be loaded, and tries again', async () => {
    let calls = 0;
    mockApi({
      'GET /api/settings/overview': () => {
        calls += 1;
        return calls === 1
          ? { status: 500, body: { code: 'internal_error', message: 'Boom', fields: [] } }
          : { body: OVERVIEW };
      },
    });
    const { user } = renderSettings();

    expect(await screen.findByRole('alert')).toHaveTextContent("Counts couldn't be loaded.");
    expect(screen.getByRole('link', { name: 'Models' })).toBeInTheDocument();
    expect(within(card('Models')).queryByText(/active$/)).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await within(card('Models')).findByText('3 active')).toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });
});
