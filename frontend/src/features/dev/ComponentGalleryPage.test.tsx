import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { CurrentUser } from '@/app/useSession';
import { findMessages, renderWithProviders } from '@/test/render';
import { ComponentGalleryPage } from './ComponentGalleryPage';
import { testUser } from '@/test/users';

const PRIYA: CurrentUser = testUser();

/** One card per shared building block of the plan's table (FR-019, design system §5). */
const COMPONENT_CARDS = [
  'Button and LinkButton',
  'PageHeader',
  'SummaryCard',
  'LinkSummaryCard',
  'Card',
  'StatusChip',
  'CompletenessChip',
  'InfoTip',
  'Tooltip',
  'ActionsMenu',
  'Avatar',
  'Logo',
  'SidePanel',
  'ConfirmDialog',
  'Toast',
  'EmptyState',
  'Skeleton',
  'TextField and PasswordField',
  'FormErrorSummary',
  'ErrorState',
  'DemoBanner',
  // The Plan module's building blocks (build guide §4).
  'Breadcrumb',
  'SearchField, SegmentedFilter and FilterChips',
  'PillTabs and CountBadge',
  'LevelTag and ProgressBar',
  'Items below and summary detail',
  'LockedNote',
  'MoneyField and DecimalField',
  'TextAreaField, CheckboxField and suggestions',
  'ChoiceField',
  'SplitEditor',
  'DateRangeField',
  'PersonLabel and TeamPills',
  'PersonPicker and TeamPicker',
  'ShellAttributeFields and ShellAttributeValues',
  'TreeTable',
  'FormSection, FormFooter, FormDialog and FormPanel',
];

function renderGallery() {
  return renderWithProviders(<ComponentGalleryPage />, { user: PRIYA });
}

function card(title: string) {
  return screen.getByRole('region', { name: title });
}

describe('ComponentGalleryPage', () => {
  it('says what it shows and names the tab (plan "Developer pages", FR-017)', () => {
    renderGallery();

    expect(
      screen.getByRole('heading', { level: 1, name: 'Shared building blocks' }),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Every shared component of design system §5 in all its variants.'),
    ).toBeInTheDocument();
    expect(document.title).toBe('Shared building blocks · NorthStar');
  });

  it.each(COMPONENT_CARDS)('has a card for %s (FR-019)', (title) => {
    renderGallery();

    expect(card(title)).toBeInTheDocument();
  });

  it('shows every button variant and size, with an icon, icon only and as a link (FR-019)', () => {
    renderGallery();

    const buttons = card('Button and LinkButton');
    for (const name of ['Primary', 'Secondary', 'Danger', 'Ghost', 'Small primary', 'Small']) {
      expect(within(buttons).getByRole('button', { name })).toBeInTheDocument();
    }
    expect(within(buttons).getByRole('button', { name: 'With an icon' })).toBeInTheDocument();
    expect(within(buttons).getByRole('button', { name: 'Copy plan' })).toBeInTheDocument();
    expect(within(buttons).getByRole('link', { name: 'Go to Home' })).toHaveAttribute('href', '/');
  });

  it('shows the six status tones, each with its word (FR-019, design system §4)', () => {
    renderGallery();

    const chips = card('StatusChip');
    for (const word of ['Draft', 'Submitted', 'Approved', 'In progress', 'Complete', 'Declined']) {
      expect(within(chips).getByText(word)).toBeInTheDocument();
    }
  });

  it('shows both completeness states, and the missing fields lead to their fields (FR-019)', async () => {
    const { user } = renderGallery();

    const chips = card('CompletenessChip');
    expect(within(chips).getByText('All filled in')).toBeInTheDocument();
    await user.click(within(chips).getByRole('button', { name: '2 fields needed' }));
    await user.click(await screen.findByRole('button', { name: 'Campaign name' }));

    await waitFor(() => expect(screen.getByLabelText('Campaign name')).toHaveFocus());
  });

  it('shows four summary cards, each explained by an info tip, one with a shortened value (FR-019)', () => {
    renderGallery();

    const summaries = card('SummaryCard');
    expect(within(summaries).getAllByRole('button', { name: /^About / })).toHaveLength(4);
    expect(
      within(summaries).getByRole('button', { name: '$1.25M ($1,250,000.00)' }),
    ).toHaveTextContent('$1.25M');
  });

  it('shows the link variant of the summary card: one link per card, named with its value (009 D-197)', () => {
    renderGallery();

    const cards = card('LinkSummaryCard');
    expect(
      within(cards).getByRole('link', { name: "My teams' plans: 3 — open the Plans list" }),
    ).toHaveAttribute('href', '/plans?scope=myTeams');
    expect(
      within(cards).getByRole('link', {
        name: 'Planned cost: $1.25M ($1,250,000.00) — go to the top of this page',
      }),
    ).toHaveAttribute('href', '#main');
  });

  it('shows the avatar sizes next to the names they stand for (FR-019)', () => {
    renderGallery();

    const avatars = card('Avatar');
    for (const name of ['Priya Raman', 'Daniel Okoro', 'Mei Lin']) {
      expect(within(avatars).getByText(name)).toBeInTheDocument();
    }
  });

  it('opens the actions menu with Delete last (FR-019)', async () => {
    const { user } = renderGallery();

    await user.click(screen.getByRole('button', { name: 'Actions for Helix launch' }));

    const items = (await screen.findAllByRole('menuitem')).map((item) => item.textContent);
    expect(items).toEqual(['Edit', 'Delete']);
  });

  it('opens the side panel and closes it with Cancel (FR-019)', async () => {
    const { user } = renderGallery();

    await user.click(screen.getByRole('button', { name: 'Open side panel' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit campaign' });
    await user.click(within(panel).getByRole('button', { name: 'Cancel' }));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('opens the confirm dialog with Cancel focused (FR-019)', async () => {
    const { user } = renderGallery();

    await user.click(screen.getByRole('button', { name: 'Open confirm dialog' }));

    const question = await screen.findByRole('alertdialog', {
      name: "Delete campaign 'Helix launch'?",
    });
    expect(within(question).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    expect(within(question).getByRole('button', { name: 'Delete campaign' })).toBeInTheDocument();
  });

  it('shows a message, and a message with Undo that says when Undo was chosen (FR-019, FR-026)', async () => {
    const { user } = renderGallery();

    await user.click(screen.getByRole('button', { name: 'Show message' }));
    const messages = await findMessages();
    expect(messages).toHaveTextContent("Campaign 'Helix launch' created");

    await user.click(screen.getByRole('button', { name: 'Show message with Undo' }));
    await waitFor(() =>
      expect(messages).toHaveTextContent("Campaign 'Helix launch' moved to the recycle bin"),
    );
    await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));

    await waitFor(() => expect(messages).toHaveTextContent("Campaign 'Helix launch' restored"));
  });

  it('shows the fields with a marking, a hint and an error, and the error summary on request (FR-019, FR-020)', async () => {
    const { user } = renderGallery();

    expect(screen.getByLabelText('Campaign name')).toHaveAccessibleDescription(
      'Required Enter a name.',
    );
    expect(screen.getByLabelText('Budget owner')).toHaveAccessibleDescription(
      'Needed to publish The person who answers for the budget.',
    );
    expect(screen.getByLabelText('Password')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Show the error summary' }));

    const summary = within(card('FormErrorSummary')).getByRole('alert');
    expect(summary).toHaveTextContent('2 problems need your attention');
    expect(summary.parentElement).toHaveFocus();
  });

  it('shows the error state for a server that cannot be reached and for any other failure (FR-020)', () => {
    renderGallery();

    const errors = card('ErrorState');
    expect(errors).toHaveTextContent(
      "NorthStar can't reach its server right now. Check that it is running and try again.",
    );
    expect(errors).toHaveTextContent('Something went wrong. Please try again.');
    expect(within(errors).getAllByRole('button', { name: 'Try again' })).toHaveLength(2);
  });

  it('shows the empty state, the loading placeholder and the demo banner (FR-016, FR-019)', () => {
    renderGallery();

    expect(card('EmptyState')).toHaveTextContent('No plans yet — create your first plan');
    expect(within(card('Skeleton')).getAllByRole('status')[0]).toHaveTextContent('Loading…');
    expect(card('DemoBanner')).toHaveTextContent(
      'Demo environment — sign-in and data are for demonstration only',
    );
  });
});
