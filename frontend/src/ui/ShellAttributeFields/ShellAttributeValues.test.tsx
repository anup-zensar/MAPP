import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { activityForm, campaignForm, LUBRICANTS, MOBILITY } from '@/test/attributeForms';
import { renderWithProviders } from '@/test/render';
import { ShellAttributeValues } from './ShellAttributeValues';

/** The value next to a label in the definition list. */
function valueOf(label: string) {
  return screen.getByText(label, { selector: 'dt' }).nextElementSibling?.textContent;
}

describe('ShellAttributeValues', () => {
  it('shows each shown attribute’s value, "—" when empty and No for a missing yes/no (FR-024)', () => {
    renderWithProviders(
      <ShellAttributeValues
        form={campaignForm()}
        businessUnit={[LUBRICANTS]}
        values={{
          'SA-CLUSTER': ['SO-RETAIL', 'SO-B2B'],
          'SA-REGION': ['SO-OLD'],
          'SA-ACCOUNTS': 1200,
          'SA-BRANDS': ['SO-HELIX'],
        }}
      />,
    );

    expect(valueOf('Cluster')).toBe('Retail, B2B');
    expect(valueOf('Region')).toBe('B2B – OLD (inactive)');
    expect(valueOf('SMP')).toBe('No');
    expect(valueOf('Campaign code')).toBe('—');
    expect(valueOf('Target accounts')).toBe('1,200');
    expect(valueOf('Brands')).toBe('Shell Helix');
    expect(
      screen.getByText("Shown because the plan's business unit includes Lubricants."),
    ).toBeInTheDocument();
  });

  it('leaves out attributes hidden for the business unit (FR-026)', () => {
    renderWithProviders(
      <ShellAttributeValues
        form={campaignForm()}
        businessUnit={[MOBILITY]}
        values={{ 'SA-BRANDS': ['SO-HELIX'] }}
      />,
    );

    expect(screen.queryByText('Brands')).not.toBeInTheDocument();
  });

  it('writes splits with their shares and marks inactive attributes (FR-027)', () => {
    renderWithProviders(
      <ShellAttributeValues
        form={activityForm()}
        businessUnit={[]}
        values={{
          'SA-SPLIT': [
            { optionId: 'SO-HELIX', share: '60.00' },
            { optionId: 'SO-RIMULA', share: '40' },
          ],
          'SA-RETIRED': 'FIT4-0042',
        }}
      />,
    );

    expect(valueOf('Brand split')).toBe('Shell Helix 60.00 %, Shell Rimula 40.00 %');
    expect(valueOf('Retired field')).toBe('FIT4-0042 (inactive)');
  });
});
