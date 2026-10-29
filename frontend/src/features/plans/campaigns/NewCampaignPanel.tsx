import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useLocation, useNavigate, useParams } from 'react-router';
import { useOverspendConfiguration } from '@/features/overspend';
import type { OverspendMode } from '@/features/overspend';
import { useAttributeForm } from '@/features/settings';
import { useToast } from '@/ui/Toast/useToast';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { recordHref } from '@/ui/record-links';
import { usePageTitle } from '@/ui/usePageTitle';
import { planCurrencyOf } from '../plan-currency';
import { useCreateCampaign, usePlan, usePlanCampaigns } from './api';
import type { PlanCampaigns } from './api';
import { PanelFailure, PanelLoading, RecordFormPanel, StatePanel } from './PanelFrame';
import { RecordForm } from './RecordForm';
import { bodyOf, defaultAttributes, fieldId, RECORD_FIELDS } from './form-values';
import { recordFormSchema } from './schemas';
import type { RecordFormValues } from './schemas';
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
  const overspend = useOverspendConfiguration();
  const closeTo = `/plans/${planId}`;

  const failed = [campaigns, plan, attributeForm, overspend].find((query) => query.isError);
  if (failed) {
    return (
      <StatePanel title={NEW_CAMPAIGN} closeTo={closeTo}>
        <PanelFailure error={failed.error} onRetry={() => void failed.refetch()} />
      </StatePanel>
    );
  }
  if (!campaigns.data || !plan.data || !attributeForm.data || !overspend.data) {
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
      overspend={overspend.data.defaultMode}
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
  overspend: OverspendMode;
}

function NewCampaignForm({
  planId,
  data,
  businessUnit,
  attributeForm,
  currency,
  overspend,
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
  const form = useForm<RecordFormValues>({
    resolver: zodResolver(recordFormSchema),
    defaultValues: {
      name: '',
      description: '',
      budget: '',
      overspend,
      ...defaults,
      attributes: defaultAttributes(attributeForm),
    },
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems } = useFormProblems(form, RECORD_FIELDS);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const planHref = `/plans/${planId}`;

  async function save(values: RecordFormValues, addAnother: boolean) {
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
      showServerProblems(shown.problems);
      setFailure(shown.failure);
      setSaving(false);
    }
  }

  return (
    <RecordFormPanel
      title={NEW_CAMPAIGN}
      description={insidePlan(data.plan.name)}
      isDirty={!leaving && form.formState.isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      fieldId={(field) => fieldId('campaign', field)}
      onSave={() => void form.handleSubmit((values) => save(values, false), onInvalid)()}
      onSaveAndAddAnother={() =>
        void form.handleSubmit((values) => save(values, true), onInvalid)()
      }
      onCancel={() => leave(planHref)}
      onClose={() => leave(planHref)}
    >
      <RecordForm
        level="campaign"
        control={form.control}
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
