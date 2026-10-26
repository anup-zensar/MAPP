import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import { ClearValuesDialog } from '@/ui/ShellAttributeFields/ClearValuesDialog';
import { valuesToClear } from '@/ui/ShellAttributeFields/rules';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import { fetchLatestRecord, useUpdateRecord } from './api';
import type { Campaign, Initiative } from './api';
import { bodyOf, fieldId, sameValues, valuesOf } from './form-values';
import { RecordFormPanel } from './PanelFrame';
import { RecordForm } from './RecordForm';
import { CONFLICT_NEXT_STEP, EDIT_CAMPAIGN, EDIT_INITIATIVE, savedToast } from './texts';
import type { RecordLevel } from './texts';
import { useLeaveAfterSave } from './useLeaveAfterSave';

export interface RecordEditProps {
  level: RecordLevel;
  record: Campaign | Initiative;
  planId: string;
  attributeForm: AttributeForm;
  businessUnit: string[];
  /** The plan currency: the budget is typed and sent in it (016 FR-011). */
  currency: string;
}

/**
 * A campaign's or initiative's edit form (US3, FR-018): it copies the record and its version once
 * (001's edit-form rule), lists the values a change will clear before saving, shows every problem
 * at once, keeps the typed values after a conflict and takes the new version, and asks before
 * typed changes are lost. `?focus=<field>` puts the focus on that field (completeness links).
 */
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
  // The record as the form opened it: later refreshes never change the form.
  const [opened] = useState(record);
  const [start] = useState(() => valuesOf(opened));
  const [values, setValues] = useState(start);
  const [version, setVersion] = useState(opened.version);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const [askingToClear, setAskingToClear] = useState(false);
  const detailsHref = recordHref(level, record.id, planId);
  const cleared = valuesToClear(attributeForm, values.attributes, start.attributes);
  const focus = searchParams.get('focus');
  useFocusField(focus ? fieldId(level, focus) : null);

  function requestSave() {
    if (cleared.length > 0) {
      setAskingToClear(true);
    } else {
      void save();
    }
  }

  async function save() {
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
      setProblems(apiError.fields);
      setFailure(null);
      return;
    }
    setProblems([]);
    if (apiError.kind === 'conflict') {
      // The typed values stay; the form takes the version with the other person's change, so
      // Save again replaces it on purpose.
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
    // Anything else: the message as a toast, the typed values kept (plan "States").
    setFailure(null);
    show({ message: apiError.message });
  }

  const campaign = 'campaign' in opened ? opened.campaign : undefined;

  return (
    <RecordFormPanel
      title={level === 'campaign' ? EDIT_CAMPAIGN : EDIT_INITIATIVE}
      description={opened.name}
      isDirty={!leaving && !sameValues(values, start)}
      saving={saving}
      problems={problems}
      failure={failure}
      fieldId={(field) => fieldId(level, field)}
      onSave={requestSave}
      onCancel={() => leave(detailsHref)}
      onClose={() => leave(`/plans/${planId}`)}
      dialogs={
        <ClearValuesDialog
          open={askingToClear && cleared.length > 0}
          onOpenChange={setAskingToClear}
          cleared={cleared}
          onConfirm={() => void save()}
          busy={saving}
        />
      }
    >
      <RecordForm
        level={level}
        values={values}
        onChange={setValues}
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
      />
    </RecordFormPanel>
  );
}

/**
 * Puts the focus on the field with this input id once it is on screen (a person field appears
 * when the people are loaded), after the panel has placed its own first focus.
 */
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
