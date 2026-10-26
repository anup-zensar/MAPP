import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { page, shellOption } from '@/testing/fixtures/shell-attributes';
import {
  optionQueries,
  PRINT,
  RADIO,
  renderOptionsCard,
  TV,
} from '@/testing/fixtures/shell-attribute-pages';
import { reply } from '@/testing/replies';

describe('OptionsCard (FR-018, FR-020, SC-007)', () => {
  it('lists the options with their own Used by and status', async () => {
    renderOptionsCard([
      reply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([TV, RADIO, PRINT]) },
        { attributeId: 'SA-1010' },
      ),
    ]);

    const tv = await screen.findByRole('row', { name: /TV/ });
    expect(within(tv).getByText('Used by 7 activities')).toBeInTheDocument();
    expect(within(tv).getByText('Active')).toBeInTheDocument();
    const print = screen.getByRole('row', { name: /Print/ });
    expect(within(print).getByText('Inactive')).toBeInTheDocument();
    expect(screen.queryByRole('columnheader', { name: 'Linked to' })).not.toBeInTheDocument();
    expect(screen.queryByText(/Showing/)).not.toBeInTheDocument();
  });

  it('finds an option among 1,217 by any part of its label, asking the API', async () => {
    const match = shellOption({ id: 'SO-2220', label: 'CN04000220', attributeId: 'SA-1010' });
    const firstPage = Array.from({ length: 3 }, (_, index) =>
      shellOption({
        id: `SO-${3000 + index}`,
        label: `CN0400${1000 + index}`,
        attributeId: 'SA-1010',
      }),
    );
    const { user, api } = renderOptionsCard([
      reply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        (request) =>
          new URL(request.url).searchParams.get('q') === '0220'
            ? { body: page([match]) }
            : { body: page(firstPage, 1217) },
        { attributeId: 'SA-1010' },
      ),
    ]);

    expect(
      await screen.findByText('Showing 3 of 1,217 — search to find the others.'),
    ).toBeInTheDocument();
    await user.type(screen.getByRole('searchbox', { name: 'Search options' }), '0220');

    expect(await screen.findByRole('rowheader', { name: 'CN04000220' })).toBeInTheDocument();
    expect(screen.getAllByRole('rowheader')).toHaveLength(1);
    const last = optionQueries(api).at(-1);
    expect(last?.get('q')).toBe('0220');
    expect(last?.get('status')).toBe('all');
    expect(last?.get('limit')).toBe('100');
  });

  it('shows 100 more options at a time with Show more', async () => {
    const first = Array.from({ length: 100 }, (_, index) =>
      shellOption({
        id: `SO-${3000 + index}`,
        label: `Option ${index + 1}`,
        attributeId: 'SA-1010',
      }),
    );
    const second = Array.from({ length: 2 }, (_, index) =>
      shellOption({
        id: `SO-${4000 + index}`,
        label: `Option ${index + 101}`,
        attributeId: 'SA-1010',
      }),
    );
    const { user, api } = renderOptionsCard([
      reply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        (request) =>
          new URL(request.url).searchParams.get('offset') === '100'
            ? { body: page(second, 1217) }
            : { body: page(first, 1217) },
        { attributeId: 'SA-1010' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Show more' }));

    expect(
      await screen.findByText('Showing 102 of 1,217 — search to find the others.'),
    ).toBeInTheDocument();
    expect(screen.getByRole('rowheader', { name: 'Option 102' })).toBeInTheDocument();
    expect(optionQueries(api).map((query) => query.get('offset'))).toEqual(['0', '100']);
    expect(optionQueries(api).map((query) => query.get('limit'))).toEqual(['100', '100']);
  });

  it('filters by status and clears the filters when nothing matches', async () => {
    const { user, api } = renderOptionsCard([
      reply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        (request) =>
          new URL(request.url).searchParams.get('status') === 'inactive'
            ? { body: page([]) }
            : { body: page([TV, RADIO, PRINT]) },
        { attributeId: 'SA-1010' },
      ),
    ]);

    await screen.findByRole('rowheader', { name: 'TV' });
    await user.click(screen.getByRole('button', { name: 'Inactive' }));
    await user.click(await screen.findByRole('button', { name: 'Clear filters' }));

    expect(await screen.findByRole('rowheader', { name: 'Radio' })).toBeInTheDocument();
    expect(optionQueries(api).map((query) => query.get('status'))).toEqual([
      'all',
      'inactive',
      'all',
    ]);
  });
});

describe('OptionsCard: options from a spreadsheet (FR-031)', () => {
  it('opens the import of the attribute', async () => {
    const { user } = renderOptionsCard([
      reply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([TV]) },
        { attributeId: 'SA-1010' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Import from a spreadsheet' }));

    expect(
      await screen.findByRole('dialog', { name: 'Import options into Media channel' }),
    ).toBeInTheDocument();
  });

  it('downloads the template of the attribute', async () => {
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      value: () => 'blob:template',
    });
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: () => {} });
    const names: string[] = [];
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
      this: HTMLAnchorElement,
    ) {
      names.push(this.download);
    });
    const { user } = renderOptionsCard([
      reply(
        'GET /api/v1/shell-attributes/{attributeId}/options',
        { body: page([TV]) },
        { attributeId: 'SA-1010' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Download template' }));

    expect(names).toEqual(['Media channel – options template.csv']);
    click.mockRestore();
    Reflect.deleteProperty(URL, 'createObjectURL');
    Reflect.deleteProperty(URL, 'revokeObjectURL');
  });
});
