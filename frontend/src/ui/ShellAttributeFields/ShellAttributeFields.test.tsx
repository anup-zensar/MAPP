import { cleanup, screen, within } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import type { components } from '@/api/generated/schema';
import {
  activityForm,
  campaignForm,
  LUBRICANTS,
  MOBILITY,
  planForm,
} from '@/testing/fixtures/attribute-forms';
import { renderWithProviders } from '@/testing/render';
import { ShellAttributeFields } from './ShellAttributeFields';
import type { ShellAttributeFieldsProps } from './ShellAttributeFields';

type AttributeValues = components['schemas']['AttributeValues'];

interface ExampleProps extends Partial<Omit<ShellAttributeFieldsProps, 'values' | 'onChange'>> {
  start?: AttributeValues;
}

function Example({
  start = {},
  form = campaignForm(),
  businessUnit = [LUBRICANTS],
  ...props
}: ExampleProps) {
  const [values, setValues] = useState<AttributeValues>(start);
  return (
    <>
      <ShellAttributeFields
        form={form}
        businessUnit={businessUnit}
        values={values}
        onChange={setValues}
        {...props}
      />
      <p>values: {JSON.stringify(values)}</p>
    </>
  );
}

describe('ShellAttributeFields', () => {
  it('shows the attributes by section with their marking words and ⓘ help (FR-024)', () => {
    renderWithProviders(<Example />);

    expect(screen.getByRole('heading', { name: 'Shell attributes' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Lubricants' })).toBeInTheDocument();
    expect(screen.getByRole('group', { name: 'Cluster Needed to publish' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'About Cluster' })).toBeInTheDocument();
    expect(screen.getByRole('checkbox', { name: 'SMP' })).not.toBeChecked();
    expect(screen.getByRole('textbox', { name: 'Campaign code' })).toHaveAttribute(
      'id',
      'attributes.SA-CODE',
    );
    expect(screen.getByRole('textbox', { name: 'Notes' })).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: 'Target accounts' })).toBeInTheDocument();
  });

  it('locks a dependent until its parent is chosen, then offers only the linked options (FR-025)', async () => {
    const { user } = renderWithProviders(<Example />);

    expect(screen.getByText('Choose a cluster first.')).toBeInTheDocument();

    await user.click(screen.getByRole('checkbox', { name: 'B2B' }));

    const region = screen.getByRole('group', { name: 'Region Needed to publish' });
    expect(
      within(region)
        .getAllByRole('checkbox')
        .map((box) => box.getAttribute('id')),
    ).toHaveLength(2);
    expect(within(region).getByRole('checkbox', { name: 'B2B – EUROPE' })).toBeEnabled();
    expect(screen.queryByText('Choose a cluster first.')).not.toBeInTheDocument();
  });

  it('fills in the only fitting option and says so (design system §7, US3)', async () => {
    const { user } = renderWithProviders(<Example />);

    await user.click(screen.getByRole('checkbox', { name: 'Retail' }));

    expect(
      screen.getByText("Region set to 'RETAIL – CHINA', the only region for this cluster."),
    ).toBeInTheDocument();
    expect(screen.getByRole('checkbox', { name: 'RETAIL – CHINA' })).toBeChecked();
  });

  it('says when a chosen parent has no options set up (D-034)', async () => {
    const { user } = renderWithProviders(<Example />);

    await user.click(screen.getByRole('checkbox', { name: 'Fleet' }));

    expect(
      screen.getByText('No options set up for this cluster — ask an administrator.'),
    ).toBeInTheDocument();
  });

  it('says which values a save would clear when their parent changes (FR-025)', async () => {
    const stored = { 'SA-CLUSTER': ['SO-B2B'], 'SA-REGION': ['SO-EU'] };
    const { user } = renderWithProviders(<Example start={stored} previousValues={stored} />);

    await user.click(screen.getByRole('checkbox', { name: 'B2B' }));
    await user.click(screen.getByRole('checkbox', { name: 'Retail' }));

    expect(
      screen.getByText("Region 'B2B – EUROPE' will be cleared — it doesn't fit 'Retail'."),
    ).toBeInTheDocument();
  });

  it('hides attributes of other business units but keeps their values (FR-026, US3-3)', () => {
    renderWithProviders(
      <Example businessUnit={[MOBILITY]} start={{ 'SA-BRANDS': ['SO-HELIX'] }} />,
    );

    expect(screen.queryByRole('heading', { name: 'Lubricants' })).not.toBeInTheDocument();
    expect(screen.getByText('values: {"SA-BRANDS":["SO-HELIX"]}')).toBeInTheDocument();
  });

  it('says why a conditional section is shown (FR-023)', () => {
    renderWithProviders(<Example businessUnit={[LUBRICANTS, MOBILITY]} />);

    expect(
      screen.getByText("Shown because the plan's business unit includes Lubricants."),
    ).toBeInTheDocument();
  });

  it('says more attributes appear once the plan has a business unit, with a link (FR-023)', () => {
    renderWithProviders(<Example businessUnit={[]} planLink="/plans/P-1001" />);

    expect(
      screen.getByText("More Shell attributes appear once the plan's business unit is chosen.", {
        exact: false,
      }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Open the plan' })).toHaveAttribute(
      'href',
      '/plans/P-1001',
    );
  });

  it('on the plan form, its own business unit decides (FR-023)', async () => {
    const { user } = renderWithProviders(<Example form={planForm()} businessUnit={null} />);

    expect(screen.queryByRole('textbox', { name: 'Fleet size' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('checkbox', { name: 'Mobility' }));
    expect(screen.getByRole('textbox', { name: 'Fleet size' })).toBeInTheDocument();
  });

  it('keeps an inactive attribute’s value read-only, and hides it when empty (FR-027)', () => {
    renderWithProviders(<Example form={activityForm()} start={{ 'SA-RETIRED': 'FIT4-0042' }} />);

    expect(screen.getByText('FIT4-0042 (inactive)')).toBeInTheDocument();
    expect(
      screen.getByText("Retired field is inactive — its value is kept and can't be changed."),
    ).toBeInTheDocument();
    cleanup();

    renderWithProviders(<Example form={activityForm()} />);
    expect(screen.queryByText('Retired field')).not.toBeInTheDocument();
  });

  it('marks inactive options a record keeps (FR-027)', () => {
    renderWithProviders(<Example start={{ 'SA-CLUSTER': ['SO-B2B'], 'SA-REGION': ['SO-OLD'] }} />);

    expect(screen.getByRole('checkbox', { name: 'B2B – OLD (inactive)' })).toBeChecked();
    expect(
      screen.getByText("Inactive options stay on this record but can't be chosen again."),
    ).toBeInTheDocument();
  });

  it('edits a % split and a whole number in the API’s shapes (FR-027)', async () => {
    const { user } = renderWithProviders(<Example form={activityForm()} />);

    await user.click(screen.getByRole('checkbox', { name: 'Shell Helix' }));
    expect(
      screen.getByText('values: {"SA-SPLIT":[{"optionId":"SO-HELIX","share":"100.00"}]}'),
    ).toBeInTheDocument();
  });

  it('stores whole numbers as numbers and shows problems by field name', async () => {
    const { user } = renderWithProviders(
      <Example errors={{ 'attributes.SA-ACCOUNTS': 'Enter a whole number.' }} />,
    );

    await user.type(screen.getByRole('textbox', { name: 'Target accounts' }), '1200');

    expect(screen.getByText('values: {"SA-ACCOUNTS":1200}')).toBeInTheDocument();
    expect(screen.getByText('Enter a whole number.')).toBeInTheDocument();
  });
});
