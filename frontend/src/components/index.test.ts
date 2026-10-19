import { describe, expect, it } from 'vitest';
import * as components from './index';

describe('the shared building blocks', () => {
  it('are all available from @/components (FR-019)', () => {
    expect(Object.keys(components).sort()).toEqual([
      'ActionsMenu',
      'Avatar',
      'Button',
      'Card',
      'CompletenessChip',
      'ConfirmDialog',
      'DemoBanner',
      'EmptyState',
      'ErrorState',
      'FormErrorSummary',
      'InfoTip',
      'LinkButton',
      'Logo',
      'PageHeader',
      'PasswordField',
      'SidePanel',
      'Skeleton',
      'StatusChip',
      'SummaryCard',
      'TextField',
      'ToastProvider',
      'Tooltip',
      'useToast',
    ]);
  });
});
