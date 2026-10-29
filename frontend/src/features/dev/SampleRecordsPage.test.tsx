import { screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  IDEAS,
  IDEAS_IN_BIN,
  listOf,
  mockSampleRecords,
  PLANS,
  renderSampleRecordsPage,
  renderSampleRecordsPageLoading,
  summaryValue,
  UNREACHABLE,
} from '@/testing/fixtures/dev';

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('SampleRecordsPage: the page', () => {
  it('says what it is for, offers New sample folder and names the tab (plan "Developer pages", FR-017)', async () => {
    mockSampleRecords();
    await renderSampleRecordsPage();

    expect(screen.getByText('Developer')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1, name: 'Sample records' })).toBeInTheDocument();
    expect(
      screen.getByText(
        'Try the shared create, edit, delete, Undo and activity-log behaviour. Not part of the Plan module.',
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'New sample folder' })).toBeInTheDocument();
    expect(document.title).toBe('Sample records · SP&A One');
  });

  it('counts the folders, their notes and what is in the recycle bin (plan "Developer pages")', async () => {
    mockSampleRecords({ bin: { body: listOf([IDEAS_IN_BIN]) } });
    await renderSampleRecordsPage();

    expect(summaryValue('Sample folders')).toHaveTextContent('2');
    expect(summaryValue('Sample notes')).toHaveTextContent('2');
    expect(summaryValue('In the recycle bin')).toHaveTextContent('1');
  });

  it('lists each folder with its ID, its notes and who changed it last, and when (plan "Developer pages")', async () => {
    mockSampleRecords();
    await renderSampleRecordsPage();

    const ideas = screen.getByRole('row', { name: /^Ideas/ });
    expect(within(ideas).getByRole('rowheader')).toHaveTextContent('Ideas SF-1001');
    expect(within(ideas).getByText('Agenda')).toBeInTheDocument();
    expect(within(ideas).getByRole('button', { name: "Delete note 'Agenda'" })).toBeInTheDocument();
    expect(
      within(ideas).getByRole('button', { name: "Delete note 'Minutes'" }),
    ).toBeInTheDocument();
    expect(within(ideas).getByText('Priya Raman, 12 Mar 2026, 09:00')).toBeInTheDocument();
    expect(within(ideas).getByRole('button', { name: 'Actions for Ideas' })).toBeInTheDocument();

    const plans = screen.getByRole('row', { name: /^Plans/ });
    expect(within(plans).getByText('Daniel Okoro, 12 Mar 2026, 14:32')).toBeInTheDocument();
    expect(within(plans).queryByRole('button', { name: /^Delete note/ })).not.toBeInTheDocument();
  });

  it('offers Edit, Add note and, after a separator, Delete for each folder (design system §6.1)', async () => {
    mockSampleRecords();
    const { user } = await renderSampleRecordsPage();

    await user.click(screen.getByRole('button', { name: 'Actions for Ideas' }));

    const items = (await screen.findAllByRole('menuitem')).map((item) => item.textContent);
    expect(items).toEqual(['Edit', 'Add note', 'Delete']);
  });

  it('invites to create the first sample folder when there are none (plan "User-visible messages")', async () => {
    mockSampleRecords({ folders: { body: listOf([]) } });
    const { user } = await renderSampleRecordsPage();

    const card = screen.getByRole('region', { name: 'Sample folders' });
    expect(
      within(card).getByText('No sample folders yet — create your first sample folder'),
    ).toBeInTheDocument();
    await user.click(within(card).getByRole('button', { name: 'New sample folder' }));

    expect(await screen.findByRole('dialog', { name: 'New sample folder' })).toBeInTheDocument();
  });

  it('shows a loading placeholder until the lists arrive (design system §5)', async () => {
    mockSampleRecords({ folders: () => new Promise<never>(() => {}) });
    renderSampleRecordsPageLoading();

    expect(await screen.findByRole('status')).toHaveTextContent('Loading…');
  });

  it('says when the API cannot be reached, and shows the page after Try again (FR-020)', async () => {
    let reads = 0;
    mockSampleRecords({
      folders: () =>
        ++reads === 1
          ? Promise.reject(new TypeError('Failed to fetch'))
          : { body: listOf([IDEAS, PLANS]) },
    });
    const { user } = renderSampleRecordsPageLoading();

    expect(await screen.findByRole('alert')).toHaveTextContent(UNREACHABLE);
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('row', { name: /^Ideas/ })).toBeInTheDocument();
  });

  it('leaves out Recently deleted and the activity log while they have nothing to show', async () => {
    mockSampleRecords();
    await renderSampleRecordsPage();

    expect(screen.queryByRole('region', { name: 'Recently deleted' })).not.toBeInTheDocument();
    expect(screen.queryByRole('region', { name: 'Activity log' })).not.toBeInTheDocument();
  });
});
