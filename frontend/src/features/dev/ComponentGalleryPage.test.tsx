import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { CurrentUser } from '@/session/useSession';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { ComponentGalleryPage } from './ComponentGalleryPage';

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
  'Tabs and CountBadge',
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

describe('ComponentGalleryPage', () => {
  it('shows a card for every shared building block and names the tab (FR-017, FR-019)', () => {
    renderWithProviders(<ComponentGalleryPage />, { user: PRIYA });

    expect(
      screen.getByRole('heading', { level: 1, name: 'Shared building blocks' }),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Every shared component of design system §5 in all its variants.'),
    ).toBeInTheDocument();
    expect(document.title).toBe('Shared building blocks · SP&A One');
    for (const title of COMPONENT_CARDS) {
      expect(screen.getByRole('region', { name: title })).toBeInTheDocument();
    }
  });
});
