import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { useSearchParams } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import { ClearValuesDialog } from '@/ui/ShellAttributeFields/ClearValuesDialog';
import { valuesToClear } from '@/ui/ShellAttributeFields/rules';
import { useToast } from '@/ui/Toast/useToast';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { recordHref } from '@/ui/record-links';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { useFormatters } from '@/format/useFormatters';
import { focusOf } from '../schemas';
import { fetchLatestRecord, useUpdateRecord } from './api';
import type { Campaign, Initiative } from './api';
import { bodyOf, fieldId, RECORD_FIELDS, valuesOf } from './form-values';
import { RecordFormPanel } from './PanelFrame';
import { RecordForm } from './RecordForm';
import { overspendRefusal } from './overspend-check';
import { recordFormSchema } from './schemas';
import type { RecordFormValues } from './schemas';
import { CONFLICT_NEXT_STEP, EDIT_CAMPAIGN, EDIT_INITIATIVE, savedToast } from './texts';
import type { RecordLevel } from './texts';
import { useLeaveAfterSave } from './useLeaveAfterSave';

export interface RecordEditProps {
  level: RecordLevel;
  record: Campaign | Initiative;
  planId: string;
  attributeForm: AttributeForm;
  businessUnit: string[];
  currency: string;
}

export function RecordEdit({
  level,
  record,
  planId,
  attributeForm,
  businessUnit,
  currency,
}: RecordEditProps) {
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const update = useUpdateRecord(level, record.id);
  const { leaving, leave } = useLeaveAfterSave();
  const [opened] = useState(record);
  const start = valuesOf(opened);
  const form = useForm<RecordFormValues>({
    resolver: zodResolver(recordFormSchema),
    defaultValues: start,
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems } = useFormProblems(form, RECORD_FIELDS);
  const attributes = useWatch({ control: form.control, name: 'attributes' });
  const [version, setVersion] = useState(opened.version);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [askingToClear, setAskingToClear] = useState(false);
  const detailsHref = recordHref(level, record.id, planId);
  const cleared = valuesToClear(attributeForm, attributes, start.attributes);
  const focus = focusOf(searchParams);
  useFocusField(focus ? fieldId(level, focus) : null);

  const requestSave = form.handleSubmit((typed) => {
    const refused = overspendRefusal(typed, opened);
    if (refused) {
      setFailure({ message: refused });
      return;
    }
    if (valuesToClear(attributeForm, typed.attributes, start.attributes).length > 0) {
      setAskingToClear(true);
    } else {
      void save(typed);
    }
  }, onInvalid);

  async function save(values: RecordFormValues) {
    setSaving(true);
    try {
      const saved = await update.mutateAsync({ ...bodyOf(values, currency), version });
      show({ message: savedToast(level, saved.name) });
      leave(detailsHref);
    } catch (error) {
      setAskingToClear(false);
      await showFailure(error);
      setSaving(false);
    }
  }

  async function showFailure(error: unknown) {
    const apiError = error instanceof ApiError ? error : new ApiError('other');
    if (apiError.kind === 'validation') {
      showServerProblems(apiError.fields);
      setFailure(null);
      return;
    }
    showServerProblems([]);
    if (apiError.kind === 'conflict') {
      const latest = await fetchLatestRecord(queryClient, level, record.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
      setFailure({
        message: conflictMessage(apiError, level, dateFormat),
        nextStep: CONFLICT_NEXT_STEP,
      });
      return;
    }
    if (apiError.kind === 'overBudget') {
      setFailure({ message: apiError.message });
      return;
    }
    setFailure(null);
    show({ message: apiError.message });
  }

  const campaign = 'campaign' in opened ? opened.campaign : undefined;

  return (
    <RecordFormPanel
      title={level === 'campaign' ? EDIT_CAMPAIGN : EDIT_INITIATIVE}
      description={opened.name}
      isDirty={!leaving && form.formState.isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      fieldId={(field) => fieldId(level, field)}
      onSave={() => void requestSave()}
      onCancel={() => leave(detailsHref)}
      onClose={() => leave(`/plans/${planId}`)}
      dialogs={
        <ClearValuesDialog
          open={askingToClear && cleared.length > 0}
          onOpenChange={setAskingToClear}
          cleared={cleared}
          onConfirm={() => void save(form.getValues())}
          busy={saving}
        />
      }
    >
      <RecordForm
        level={level}
        control={form.control}
        problems={problems}
        attributeForm={attributeForm}
        businessUnit={businessUnit}
        previousAttributes={start.attributes}
        planId={planId}
        planYear={opened.plan.year}
        campaign={campaign}
        parentRemaining={opened.parentRemainingBudget}
        savedBudget={opened.budget}
        currency={currency}
        phased={!!opened.budgetPhasing && opened.budgetPhasing.method !== 'none'}
        overspendLock={
          opened.overspend.canChange.available ? '' : opened.overspend.canChange.reason
        }
        planned={opened.budgetAndCost.plannedCost}
      />
    </RecordFormPanel>
  );
}

function useFocusField(id: string | null) {
  useEffect(() => {
    if (!id) {
      return;
    }
    let tries = 0;
    const timer = window.setInterval(() => {
      const element = document.getElementById(id);
      tries += 1;
      if (element || tries > 40) {
        element?.focus();
        window.clearInterval(timer);
      }
    }, 50);
    return () => window.clearInterval(timer);
  }, [id]);
}
