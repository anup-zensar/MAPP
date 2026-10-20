import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectCurrentSection, renderApp } from '@/test/app';
import { FT_1003, transactionOf, transactionPage } from './testData';

const LIST = { 'GET /api/financials/transactions': { body: transactionPage() } };

describe('Financials routes (015 FR-020, FR-022)', () => {
  it('opens /financials: in the sidebar after All work, current, with its tab title', async () => {
    renderApp('/financials', LIST);

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Financials' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Financials · NorthStar');
    expectCurrentSection('Financials');
    const labels = within(screen.getByRole('navigation', { name: 'Main' }))
      .getAllByRole('link')
      .map((link) => link.textContent?.trim());
    expect(labels.indexOf('Financials')).toBe(labels.indexOf('All work') + 1);
  });

  it('opens a transaction at its own address over the list', async () => {
    renderApp('/financials/transactions/FT-1003', {
      ...LIST,
      'GET /api/financials/transactions/FT-1003': { body: transactionOf(FT_1003) },
    });

    expect(await screen.findByRole('dialog', { name: 'PO 4500001003 / 10' })).toBeInTheDocument();
    expect(document.title).toBe('Financials · NorthStar');
  });
});
