import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { nextOf } from '@/testing/dom';
import { renderWithProviders } from '@/testing/render';
import { launchFilm, MODEL } from '@/testing/fixtures/activities';
import { activityAttributeForm } from '@/testing/fixtures/activity-forms';
import { ActivityAttributes, ActivityDetails, ActivityGoals } from './ActivitySections';

function value(label: string): HTMLElement {
  return nextOf(screen.getByText(label, { selector: 'dt' }));
}

describe('ActivityDetails (feature 006, FR-013, FR-020)', () => {
  it('shows every value, "—" when empty, and says when the activity name is used', () => {
    renderWithProviders(
      <ActivityDetails
        activity={launchFilm({ owner: null, description: '', startDate: null, endDate: null })}
      />,
    );

    expect(screen.getByRole('region', { name: 'Details' })).toBeVisible();
    expect(value('Activity name')).toHaveTextContent('Launch film');
    expect(value('External name')).toHaveTextContent('— (the activity name is used)');
    expect(value('Campaign')).toHaveTextContent('Helix launch');
    expect(value('Initiative')).toHaveTextContent('Digital & social');
    expect(value('Owner')).toHaveTextContent('—');
    expect(value('Start date')).toHaveTextContent('—');
    expect(value('End date')).toHaveTextContent('—');
    expect(value('Planned cost')).toHaveTextContent('120,000.00 USD');
    expect(value('Approval workflow')).toHaveTextContent('Lubricants EU approval');
    expect(value('Description')).toHaveTextContent('—');
  });

  it('labels an inactive type, a type the model no longer allows, an inactive workflow and an ongoing activity', () => {
    renderWithProviders(
      <ActivityDetails
        activity={launchFilm({
          activityType: { id: 'AT-1001', name: 'Event', active: false, allowed: false },
          approvalWorkflow: { id: 'WF-1001', name: 'Old approval', active: false },
          ongoing: true,
          endDate: null,
        })}
      />,
    );

    expect(value('Activity type')).toHaveTextContent(
      "Event (inactive)Not allowed by the plan's model",
    );
    expect(value('Approval workflow')).toHaveTextContent('Old approval (inactive)');
    expect(value('End date')).toHaveTextContent('Ongoing (no end date)');
  });
});

describe('ActivityAttributes and ActivityGoals (feature 006, FR-014, FR-015, FR-018)', () => {
  it("shows the attribute values with the plan's business-unit group and why it is shown", () => {
    renderWithProviders(
      <ActivityAttributes
        activity={launchFilm({ attributes: { 'SA-BRANDS': ['SO-HELIX'], 'SA-FLEET': 'FC-2207' } })}
        form={activityAttributeForm()}
        businessUnits={['SO-BU-LUB']}
        error={null}
        onRetry={() => {}}
      />,
    );

    expect(screen.getByRole('region', { name: 'Shell attributes' })).toBeVisible();
    expect(value('Brands')).toHaveTextContent('Shell Helix');
    expect(value('Cluster')).toHaveTextContent('—');
    expect(
      screen.getByText("Shown because the plan's business unit includes Lubricants."),
    ).toBeVisible();
    expect(screen.queryByText('Fleet card')).not.toBeInTheDocument();
  });

  it('shows calculated goals with the formula, typed goals as Typed, and "—" when empty', async () => {
    const { user } = renderWithProviders(
      <ActivityGoals
        activity={launchFilm({
          goals: {
            inq: 250000,
            setLaterStagesMyself: false,
            mql: 5000,
            deals: 250,
            revenue: { amount: '2500000.00', currency: 'USD' },
            calculated: true,
          },
        })}
        model={MODEL}
      />,
    );

    expect(value('INQ')).toHaveTextContent('250,000');
    expect(value('MQL')).toHaveTextContent('5,000 Calculated');
    expect(value('Revenue')).toHaveTextContent('2,500,000.00 USD Calculated');
    await user.click(screen.getByRole('button', { name: 'About MQL calculation' }));
    expect(await screen.findByText(/MQL = INQ × 2.00 %/)).toBeVisible();
  });

  it('labels typed goals and shows "—" for empty ones', () => {
    renderWithProviders(
      <ActivityGoals
        activity={launchFilm({
          goals: {
            inq: null,
            setLaterStagesMyself: true,
            mql: 6000,
            deals: null,
            revenue: null,
            calculated: false,
          },
        })}
      />,
    );

    expect(value('INQ')).toHaveTextContent('—');
    expect(value('MQL')).toHaveTextContent('6,000 Typed');
    expect(value('Revenue')).toHaveTextContent('—');
  });
});
