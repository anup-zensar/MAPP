import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { allWorkOptions, gridResult, standard2026 } from '@/testing/fixtures/all-work';
import { nth } from '@/testing/dom';
import { renderWithProviders } from '@/testing/render';
import type { ViewState } from './all-work-types';
import { RowName } from './RowName';
import { fromAddress } from './schemas';
import { chipsOf, removeChip } from './view-chips';
import { toQueryParams } from './view-state';

const onlyBCPIR: ViewState = {
  ...standard2026(),
  filters: { ...standard2026().filters, bcpir: 'only' },
};

describe('013 FR-003: the BCPIR plans filter of All work', () => {
  it('asks the API for BCPIR plans only, and leaves Include out of the query', () => {
    expect(toQueryParams(onlyBCPIR).bcpir).toBe('only');
    expect(toQueryParams(standard2026()).bcpir).toBeUndefined();
  });

  it('shows a chip that removes the filter', () => {
    const chips = chipsOf(onlyBCPIR, allWorkOptions());
    expect(chips.map((chip) => chip.label)).toEqual(['BCPIR plans: Only']);
    expect(removeChip(onlyBCPIR, nth(chips, 0).key).filters.bcpir).toBe('include');
  });

  it('opens from an address, ignoring a value it does not know', () => {
    const base = standard2026();
    expect(fromAddress(new URLSearchParams('bcpir=exclude'), base).filters.bcpir).toBe('exclude');
    expect(fromAddress(new URLSearchParams('bcpir=maybe'), base).filters.bcpir).toBeUndefined();
  });
});

describe('013 FR-002: the BCPIR marker of plan rows', () => {
  const renderName = (bcpir: boolean) =>
    renderWithProviders(
      <RowName
        visible={{ row: { ...nth(gridResult().rows, 0), bcpir }, expanded: false, childCount: 0 }}
        search=""
        showStatus={false}
        onToggle={() => undefined}
        onOpen={() => undefined}
      />,
    );

  it('marks a BCPIR plan', () => {
    renderName(true);
    expect(screen.getByText('BCPIR')).toBeInTheDocument();
  });

  it('leaves other plans unmarked', () => {
    renderName(false);
    expect(screen.queryByText('BCPIR')).not.toBeInTheDocument();
  });
});
