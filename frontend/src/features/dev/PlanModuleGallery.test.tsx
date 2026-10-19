import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { PlanModuleGallery } from './PlanModuleGallery';

function renderCards() {
  return renderWithProviders(<PlanModuleGallery />, { user: testUser() });
}

describe('PlanModuleGallery (component gallery, build guide §4)', () => {
  it('shows the plan tree with its levels and a totals row', () => {
    renderCards();

    const tree = screen.getByRole('treegrid', { name: 'Plan tree' });
    expect(within(tree).getByText('Digital & social')).toBeInTheDocument();
    expect(within(tree).getByText('Total · 1 campaign')).toBeInTheDocument();
  });

  it('keeps an inactive team on the team picker and offers the owner from the chosen teams', async () => {
    const { user } = renderCards();

    const pickers = screen.getByRole('region', { name: 'PersonPicker and TeamPicker' });
    expect(within(pickers).getByText('Fleet Americas (inactive)')).toBeInTheDocument();
    await user.click(screen.getByRole('combobox', { name: 'Owner' }));
    const owners = screen.getByRole('listbox', { name: 'Owner' });
    expect(
      within(owners)
        .getAllByRole('option')
        .map((option) => option.textContent),
    ).toEqual([
      'PRPriya Raman (you)Marketer · Lubricants Europe',
      'AMAlex MorganHead of marketing · Lubricants Europe',
    ]);
  });

  it('asks before clearing a region that no longer fits its cluster', async () => {
    const { user } = renderCards();

    const card = screen.getByRole('region', {
      name: 'ShellAttributeFields and ShellAttributeValues',
    });
    await user.click(within(card).getByRole('checkbox', { name: 'B2B' }));
    await user.click(within(card).getByRole('checkbox', { name: 'Retail' }));
    await user.click(within(card).getByRole('button', { name: 'Try saving' }));

    expect(
      await screen.findByRole('alertdialog', { name: 'Clear values that no longer fit?' }),
    ).toHaveTextContent('Region: B2B – EUROPE');
  });

  it('opens the form dialog with its first field focused', async () => {
    const { user } = renderCards();

    await user.click(screen.getByRole('button', { name: 'Open form dialog' }));

    expect(screen.getByRole('textbox', { name: 'Name of the copy' })).toHaveFocus();
  });
});
