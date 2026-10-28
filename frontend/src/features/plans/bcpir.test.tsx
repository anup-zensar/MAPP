import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { plan } from '@/testing/fixtures/plans';
import { testUser } from '@/testing/fixtures/users';
import { renderWithProviders } from '@/testing/render';
import { BcpirChoice } from './BcpirChoice';
import { DEFAULT_FILTERS } from './list-preferences';
import { chipReset, filterChips } from './plan-list-chips';
import { PlanHeader } from './PlanHeader';

describe('013 FR-003: the BCPIR plans filter of the Plans list', () => {
  it('shows a chip while it is in use, and its removal includes every plan again', () => {
    expect(filterChips({ ...DEFAULT_FILTERS, bcpir: 'only' })).toContainEqual({
      key: 'bcpir',
      label: 'BCPIR plans: Only',
    });
    expect(filterChips(DEFAULT_FILTERS)).toEqual([]);
    expect(chipReset('bcpir')).toEqual({ bcpir: 'include' });
  });

  it('offers All plans, Only BCPIR plans and No BCPIR plans', async () => {
    const onChange = vi.fn();
    renderWithProviders(<BcpirChoice name="bcpir" value="include" onChange={onChange} />);

    expect(screen.getByRole('radio', { name: 'All plans' })).toBeChecked();
    await userEvent.click(screen.getByRole('radio', { name: 'Only BCPIR plans' }));

    expect(onChange).toHaveBeenCalledWith({ bcpir: 'only' });
  });
});

describe('013 FR-002, FR-014: the plan header', () => {
  it('marks a BCPIR plan and links to its SP&A view', () => {
    renderWithProviders(<PlanHeader plan={plan({ bcpir: true })} moreItems={[]} />, {
      user: testUser(),
    });

    expect(screen.getByRole('group', { name: 'Plan facts' })).toHaveTextContent('BCPIR');
    expect(screen.getByRole('link', { name: 'SP&A view' })).toHaveAttribute(
      'href',
      '/plans/P-1001/spa',
    );
  });

  it('shows no marker on another plan', () => {
    renderWithProviders(<PlanHeader plan={plan({ bcpir: false })} moreItems={[]} />, {
      user: testUser(),
    });

    expect(screen.getByRole('group', { name: 'Plan facts' })).not.toHaveTextContent('BCPIR');
  });
});
