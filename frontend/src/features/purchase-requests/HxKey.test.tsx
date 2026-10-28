import { screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { HxKey } from './HxKey';
import { PurchaseRequestStatusChip } from './PurchaseRequestStatusChip';

afterEach(() => vi.restoreAllMocks());

describe('032 FR-020: HX key', () => {
  it('shows the key read-only, says where it comes from and copies it', async () => {
    mockApi([]);
    const { user } = renderWithProviders(<HxKey value="HX1006" />, { user: testUser() });

    expect(screen.getByText('HX1006')).toHaveAccessibleDescription(
      'Generated from the line-item ID',
    );
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Copy HX key HX1006' }));

    expect(await findMessages()).toHaveTextContent('HX key copied');
    expect(await navigator.clipboard.readText()).toBe('HX1006');
    await expectNoAxeViolations();
  });

  it('says how to copy it by hand when the browser refuses', async () => {
    mockApi([]);
    const { user } = renderWithProviders(<HxKey value="HX1006" compact />, { user: testUser() });
    vi.spyOn(navigator.clipboard, 'writeText').mockRejectedValue(new Error('denied'));

    await user.click(screen.getByRole('button', { name: 'Copy HX key HX1006' }));

    expect(await findMessages()).toHaveTextContent(
      "Couldn't copy the HX key — select it and copy it yourself.",
    );
  });
});

describe('032 FR-052: status chip', () => {
  it.each([
    ['created', false, 'Created'],
    ['submitted', false, 'Submitted'],
    ['inReview', false, 'In review'],
    ['declined', false, 'Declined'],
    ['approved', false, 'Approved'],
    ['closed', false, 'Closed'],
    ['approved', true, 'Approved · Change in review'],
  ] as const)('%s (change pending %s) reads "%s"', (status, changePending, word) => {
    mockApi([]);
    renderWithProviders(
      <PurchaseRequestStatusChip status={status} changePending={changePending} />,
      { user: testUser() },
    );

    expect(screen.getByText(word)).toBeInTheDocument();
  });
});
