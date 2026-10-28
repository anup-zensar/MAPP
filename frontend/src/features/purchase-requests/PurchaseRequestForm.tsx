import { zodResolver } from '@hookform/resolvers/zod';
import { CircleAlert, RotateCw } from 'lucide-react';
import { useEffect, useEffectEvent, useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { ApprovalAlert } from '@/features/approvals';
import type { BreadcrumbItem } from '@/ui/Breadcrumb/Breadcrumb';
import { Button } from '@/ui/Button/Button';
import type { ChoiceOption } from '@/ui/ChoiceField/ChoiceField';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormFooter } from '@/ui/FormFooter/FormFooter';
import { FormCard, FormLayout } from '@/ui/FormRail/FormRail';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import { useUnsavedChanges } from '@/ui/useUnsavedChanges';
import type { LinesOnApproved, PurchaseRequest, PurchaseRequestInput } from './api';
import { clientProblems } from './form-problems';
import { purchaseRequestFormSchema } from './form-schema';
import type { PurchaseRequestFormValues } from './form-schema';
import { onlyAddsLines, saveModeOf, toInput } from './form-state';
import {
  CANCEL,
  HEADER_SECTION,
  LINES_DESCRIPTION,
  LINES_SECTION,
  RELOAD,
  SAVING,
} from './form-text';
import { railSections } from './form-progress';
import { HeaderFields } from './HeaderFields';
import type { LineItemPickerProps } from './LineItemPicker';
import { LineRowsEditor } from './LineRowsEditor';
import { usePurchaseRequestSave } from './usePurchaseRequestSave';
import type { SaveFailure } from './usePurchaseRequestSave';
import styles from './PurchaseRequestForm.module.css';

export interface PurchaseRequestFormOptions {
  attributeForm: AttributeForm;
  lineAttributeForm: AttributeForm;
  workflows: ChoiceOption[];
  modificationWorkflows: ChoiceOption[];
  currencies: ChoiceOption[];
  linesOnApproved?: LinesOnApproved;
}

export interface PurchaseRequestFormProps {
  title: string;
  breadcrumb: BreadcrumbItem[];
  initial: PurchaseRequestFormValues;
  /** The PR as stored (edit); null on New purchase request. */
  stored: PurchaseRequest | null;
  options: PurchaseRequestFormOptions;
  optionsQuery: LineItemPickerProps['query'];
  /** A field to focus on open (?field=, from the completeness chip). */
  focusField?: string | null;
  /** Throws ApiError; the edit page adds the version it last read. */
  onSave: (input: PurchaseRequestInput) => Promise<PurchaseRequest>;
  onSaved: (saved: PurchaseRequest) => void;
  /** Loads the stored PR again after a conflict, keeping what was typed (032 FR-016). */
  onReload?: () => void;
  onCancel: () => void;
}

/** New and edit purchase request (032 FR-003, FR-005 – FR-016, FR-053, FR-054): one form. */
export function PurchaseRequestForm(props: PurchaseRequestFormProps) {
  const { stored, options } = props;
  const [start] = useState(props.initial);
  const form = useForm<PurchaseRequestFormValues>({
    resolver: zodResolver(purchaseRequestFormSchema),
    defaultValues: start,
    ...CHECK_ON_SAVE,
  });
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const values = useWatch({ control: form.control, compute: (all) => all });
  const mode = saveModeOf(stored, onlyAddsLines(start, values), options.linesOnApproved);
  const save = usePurchaseRequestSave({ onSave: props.onSave, setProblems });
  const isDirty =
    JSON.stringify(toInput(values, 'save')) !== JSON.stringify(toInput(start, 'save'));
  const { dialog } = useUnsavedChanges(isDirty && save.saved === null);
  const { saved } = save;
  const { focusField } = props;
  const leave = useEffectEvent((done: PurchaseRequest) => props.onSaved(done));

  // Declared after the guard: by the time it runs, the guard has let go of the page.
  useEffect(() => {
    if (saved) {
      leave(saved);
    }
  }, [saved]);

  useEffect(() => {
    const field = focusField ? document.getElementById(focusField) : null;
    field?.scrollIntoView({ block: 'center' });
    field?.focus();
  }, [focusField]);

  const submit = form.handleSubmit(
    (valid) => {
      setProblems([]);
      void save.save(toInput(valid, mode.then));
    },
    (errors) => setProblems(clientProblems(errors)),
  );

  return (
    <div className={styles.page}>
      <PageHeader breadcrumb={props.breadcrumb} title={props.title} />
      <form noValidate aria-busy={save.saving} onSubmit={(event) => void submit(event)}>
        <FormLayout sections={railSections(values)}>
          {stored?.notice && (
            <ApprovalAlert tone="warning">
              <p>{stored.notice}</p>
            </ApprovalAlert>
          )}
          {save.failure && <Failure failure={save.failure} onReload={props.onReload} />}
          <FormErrorSummary errors={problems} />
          <FormCard id="pr-header" title={HEADER_SECTION}>
            <HeaderFields
              control={form.control}
              problems={problems}
              workflows={options.workflows}
              modificationWorkflows={options.modificationWorkflows}
              attributeForm={options.attributeForm}
            />
          </FormCard>
          <FormCard id="pr-lines" title={LINES_SECTION} description={LINES_DESCRIPTION}>
            <LineRowsEditor
              control={form.control}
              problems={problems}
              currencies={options.currencies}
              attributeForm={options.lineAttributeForm}
              stored={stored}
              optionsQuery={props.optionsQuery}
            />
          </FormCard>
        </FormLayout>
        <FormFooter>
          <Button type="submit" variant="primary" aria-busy={save.saving || undefined}>
            {save.saving ? SAVING : mode.label}
          </Button>
          <Button onClick={props.onCancel}>{CANCEL}</Button>
        </FormFooter>
      </form>
      {dialog}
    </div>
  );
}

function Failure({ failure, onReload }: { failure: SaveFailure; onReload?: () => void }) {
  return (
    <div role="alert" className={styles.failure}>
      <CircleAlert aria-hidden="true" />
      <div className={styles.failureText}>
        <p className={styles.failureMessage}>{failure.message}</p>
        {failure.details.length > 0 && (
          <ul className={styles.details}>
            {failure.details.map((detail) => (
              <li key={detail}>{detail}</li>
            ))}
          </ul>
        )}
        {failure.nextStep && <p>{failure.nextStep}</p>}
        {failure.conflict && onReload && (
          <div>
            <Button size="sm" icon={<RotateCw />} onClick={onReload}>
              {RELOAD}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
