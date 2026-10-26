import { useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router';
import { useAttributeForm } from '@/api/settingsQueries';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problemOf';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { recordHref } from '@/lib/recordLinks';
import { usePageTitle } from '@/lib/usePageTitle';
import { planCurrencyOf } from '@/features/plans/planCurrency';
import { useCreateCampaign, usePlan, usePlanCampaigns } from './campaignApi';
import type { PlanCampaigns } from './campaignApi';
import { PanelFailure, PanelLoading, RecordFormPanel, StatePanel } from './PanelFrame';
import { RecordForm } from './RecordForm';
import { bodyOf, defaultAttributes, fieldId, sameValues } from './formValues';
import type { RecordFormValues } from './formValues';
import { createdToast, insidePlan, NEW_CAMPAIGN, openName, VIEW } from './texts';
import { failureOf, useLeaveAfterSave } from './useLeaveAfterSave';

/**
 * New campaign (US1, FR-001 – FR-005), a side panel over the plan page at
 * /plans/:planId/campaigns/new: owner and dates pre-filled from the plan and labelled, the
 * remaining plan budget following the typed budget, and the plan's Shell attributes.
 */
export function NewCampaignPanel() {
  const { planId = '' } = useParams();
  const location = useLocation();
  usePageTitle(NEW_CAMPAIGN);
  const campaigns = usePlanCampaigns(planId);
  const plan = usePlan(planId);
  const attributeForm = useAttributeForm('campaign');
  const closeTo = `/plans/${planId}`;

  const failed = [campaigns, plan, attributeForm].find((query) => query.isError);
  if (failed) {
    return (
      <StatePanel title={NEW_CAMPAIGN} closeTo={closeTo}>
        <PanelFailure error={failed.error} onRetry={() => void failed.refetch()} />
      </StatePanel>
    );
  }
  if (!campaigns.data || !plan.data || !attributeForm.data) {
    return (
      <StatePanel title={NEW_CAMPAIGN} closeTo={closeTo}>
        <PanelLoading />
      </StatePanel>
    );
  }
  return (
    // A new address (Save and add another) starts a fresh form.
    <NewCampaignForm
      key={location.key}
      planId={planId}
      data={campaigns.data}
      businessUnit={plan.data.businessUnit.map((option) => option.id)}
      attributeForm={attributeForm.data}
      currency={planCurrencyOf(plan.data)}
    />
  );
}

interface NewCampaignFormProps {
  planId: string;
  data: PlanCampaigns;
  businessUnit: string[];
  attributeForm: AttributeForm;
  /** The plan currency: the budget is typed and sent in it (016 FR-011). */
  currency: string;
}

function NewCampaignForm({
  planId,
  data,
  businessUnit,
  attributeForm,
  currency,
}: NewCampaignFormProps) {
  const navigate = useNavigate();
  const { show } = useToast();
  const create = useCreateCampaign(planId);
  const { leaving, leave } = useLeaveAfterSave();
  const year = data.plan.year;
  // FR-003: the plan's owner and year, labelled while unchanged; yes/no attributes No.
  const defaults = {
    ownerId: data.planOwner?.id ?? '',
    startDate: `${year}-01-01`,
    endDate: `${year}-12-31`,
  };
  const [start] = useState<RecordFormValues>(() => ({
    name: '',
    description: '',
    budget: '',
    ...defaults,
    attributes: defaultAttributes(attributeForm),
  }));
  const [values, setValues] = useState(start);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const planHref = `/plans/${planId}`;

  async function save(addAnother: boolean) {
    setSaving(true);
    try {
      const created = await create.mutateAsync(bodyOf(values, currency));
      show({
        message: createdToast('campaign', created.name),
        action: {
          label: VIEW,
          altText: openName(created.name),
          onClick: () => void navigate(recordHref('campaign', created.id, planId)),
        },
      });
      leave(addAnother ? `${planHref}/campaigns/new` : planHref, { highlight: created.id });
    } catch (error) {
      const shown = failureOf(error);
      setProblems(shown.problems);
      setFailure(shown.failure);
      setSaving(false);
    }
  }

  return (
    <RecordFormPanel
      title={NEW_CAMPAIGN}
      description={insidePlan(data.plan.name)}
      isDirty={!leaving && !sameValues(values, start)}
      saving={saving}
      problems={problems}
      failure={failure}
      fieldId={(field) => fieldId('campaign', field)}
      onSave={() => void save(false)}
      onSaveAndAddAnother={() => void save(true)}
      onCancel={() => leave(planHref)}
      onClose={() => leave(planHref)}
    >
      <RecordForm
        level="campaign"
        values={values}
        onChange={setValues}
        problems={problems}
        attributeForm={attributeForm}
        businessUnit={businessUnit}
        planId={planId}
        planYear={year}
        defaults={defaults}
        parentRemaining={data.totals.remainingPlanBudget}
        savedBudget={null}
        currency={currency}
      />
    </RecordFormPanel>
  );
}
