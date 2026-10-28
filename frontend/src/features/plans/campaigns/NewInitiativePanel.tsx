import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useLocation, useNavigate, useParams } from 'react-router';
import { useAttributeForm } from '@/features/settings';
import { allows } from '@/session/gate-action';
import { useToast } from '@/ui/Toast/useToast';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { recordHref } from '@/ui/record-links';
import { usePageTitle } from '@/ui/usePageTitle';
import { planCurrencyOf } from '../plan-currency';
import { useCampaign, useCreateInitiative, usePlan } from './api';
import type { Campaign } from './api';
import { bodyOf, defaultAttributes, fieldId, RECORD_FIELDS } from './form-values';
import { PanelFailure, PanelLoading, RecordFormPanel, StatePanel } from './PanelFrame';
import { RecordForm } from './RecordForm';
import { RefusedPanel } from './RefusedPanel';
import { recordFormSchema } from './schemas';
import type { RecordFormValues } from './schemas';
import { createdToast, insideCampaign, NEW_INITIATIVE, openName, VIEW } from './texts';
import { failureOf, useLeaveAfterSave } from './useLeaveAfterSave';

/**
 * New initiative (US2, FR-001 – FR-006), a side panel over the plan page at
 * /plans/:planId/campaigns/:campaignId/initiatives/new: the campaign fixed and read-only, owner
 * and dates from it, the dates kept inside the campaign's (R3) and the remaining campaign budget
 * following the typed budget.
 */
export function NewInitiativePanel() {
  const { planId = '', campaignId = '' } = useParams();
  const location = useLocation();
  usePageTitle(NEW_INITIATIVE);
  const parent = useCampaign(campaignId);
  const plan = usePlan(planId);
  const attributeForm = useAttributeForm('initiative');
  const closeTo = `/plans/${planId}`;

  const failed = [parent, plan, attributeForm].find((query) => query.isError);
  if (failed) {
    return (
      <StatePanel title={NEW_INITIATIVE} closeTo={closeTo}>
        <PanelFailure error={failed.error} onRetry={() => void failed.refetch()} />
      </StatePanel>
    );
  }
  if (!parent.data || !plan.data || !attributeForm.data) {
    return (
      <StatePanel title={NEW_INITIATIVE} closeTo={closeTo}>
        <PanelLoading />
      </StatePanel>
    );
  }
  const addInitiative = parent.data.access?.addChild;
  if (!allows(addInitiative)) {
    return <RefusedPanel title={NEW_INITIATIVE} closeTo={closeTo} availability={addInitiative} />;
  }
  return (
    <NewInitiativeForm
      key={location.key}
      planId={planId}
      campaign={parent.data}
      businessUnit={plan.data.businessUnit.map((option) => option.id)}
      attributeForm={attributeForm.data}
      currency={planCurrencyOf(plan.data)}
    />
  );
}

interface NewInitiativeFormProps {
  planId: string;
  campaign: Campaign;
  businessUnit: string[];
  attributeForm: AttributeForm;
  /** The plan currency: the budget is typed and sent in it (016 FR-011). */
  currency: string;
}

function NewInitiativeForm({
  planId,
  campaign,
  businessUnit,
  attributeForm,
  currency,
}: NewInitiativeFormProps) {
  const navigate = useNavigate();
  const { show } = useToast();
  const create = useCreateInitiative(campaign.id);
  const { leaving, leave } = useLeaveAfterSave();
  // FR-003: the campaign's owner and dates (empty when it has none), labelled while unchanged.
  const defaults = {
    ownerId: campaign.ownerId ?? '',
    startDate: campaign.startDate ?? '',
    endDate: campaign.endDate ?? '',
  };
  const form = useForm<RecordFormValues>({
    resolver: zodResolver(recordFormSchema),
    defaultValues: {
      name: '',
      description: '',
      budget: '',
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
        message: createdToast('initiative', created.name),
        action: {
          label: VIEW,
          altText: openName(created.name),
          onClick: () => void navigate(recordHref('initiative', created.id, planId)),
        },
      });
      leave(
        addAnother ? `${recordHref('campaign', campaign.id, planId)}/initiatives/new` : planHref,
        { highlight: created.id },
      );
    } catch (error) {
      const shown = failureOf(error);
      showServerProblems(shown.problems);
      setFailure(shown.failure);
      setSaving(false);
    }
  }

  return (
    <RecordFormPanel
      title={NEW_INITIATIVE}
      description={insideCampaign(campaign.name)}
      isDirty={!leaving && form.formState.isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      fieldId={(field) => fieldId('initiative', field)}
      onSave={() => void form.handleSubmit((values) => save(values, false), onInvalid)()}
      onSaveAndAddAnother={() =>
        void form.handleSubmit((values) => save(values, true), onInvalid)()
      }
      onCancel={() => leave(planHref)}
      onClose={() => leave(planHref)}
    >
      <RecordForm
        level="initiative"
        control={form.control}
        problems={problems}
        attributeForm={attributeForm}
        businessUnit={businessUnit}
        planId={planId}
        planYear={campaign.plan.year}
        campaign={campaign}
        defaults={defaults}
        parentRemaining={campaign.budgetAndCost.remainingForChildren}
        savedBudget={null}
        currency={currency}
      />
    </RecordFormPanel>
  );
}
