import type { Control } from 'react-hook-form';
import { PersonField } from '@/features/people';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import type { ChoiceOption } from '@/ui/ChoiceField/ChoiceField';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormField } from '@/ui/FormField/FormField';
import { ShellAttributeFields } from '@/ui/ShellAttributeFields/ShellAttributeFields';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import { problemAt, problemMap } from './form-problems';
import type { PurchaseRequestFormValues } from './form-schema';
import { ATTRIBUTES_TITLE, LABELS, MOA_HINT, MODIFICATION_HINT, WORKFLOW_HINT } from './form-text';
import styles from './PurchaseRequestForm.module.css';

const NAME_MAX = 120;
const DESCRIPTION_MAX = 2000;

export interface HeaderFieldsProps {
  control: Control<PurchaseRequestFormValues>;
  problems: FieldProblem[];
  workflows: ChoiceOption[];
  modificationWorkflows: ChoiceOption[];
  attributeForm: AttributeForm;
}

/** The PR's standard fields and its Shell attributes (032 FR-005, FR-006). */
export function HeaderFields({
  control,
  problems,
  workflows,
  modificationWorkflows,
  attributeForm,
}: HeaderFieldsProps) {
  const error = (field: string) => problemAt(problems, field);
  return (
    <div className={styles.fields}>
      <FormField
        control={control}
        name="name"
        render={(field) => (
          <TextField
            {...field}
            label={LABELS.name}
            marking="Required"
            maxLength={NAME_MAX}
            error={error('name')}
          />
        )}
      />
      <FormField
        control={control}
        name="description"
        render={(field) => (
          <TextAreaField
            {...field}
            label={LABELS.description}
            maxLength={DESCRIPTION_MAX}
            error={error('description')}
          />
        )}
      />
      <FormField
        control={control}
        name="ownerId"
        render={(field) => (
          <PersonField
            {...field}
            label={LABELS.owner}
            marking="Required"
            error={error('ownerId')}
          />
        )}
      />
      <FormField
        control={control}
        name="moaApproverId"
        render={(field) => (
          <PersonField
            {...field}
            label={LABELS.moaApprover}
            marking="Needed to submit"
            hint={MOA_HINT}
            error={error('moaApproverId')}
          />
        )}
      />
      <FormField
        control={control}
        name="approvalWorkflowId"
        render={(field) => (
          <ChoiceField
            {...field}
            label={LABELS.approvalWorkflow}
            marking="Required"
            hint={WORKFLOW_HINT}
            options={workflows}
            error={error('approvalWorkflowId')}
          />
        )}
      />
      <FormField
        control={control}
        name="modificationWorkflowId"
        render={(field) => (
          <ChoiceField
            {...field}
            label={LABELS.modificationWorkflow}
            marking="Required"
            hint={MODIFICATION_HINT}
            options={modificationWorkflows}
            error={error('modificationWorkflowId')}
          />
        )}
      />
      {attributeForm.sections.length > 0 && (
        <div className={styles.attributes}>
          <h3 className={styles.subheading}>{ATTRIBUTES_TITLE}</h3>
          <FormField
            control={control}
            name="attributes"
            render={(field) => (
              <ShellAttributeFields
                form={attributeForm}
                values={field.value}
                onChange={field.onChange}
                businessUnit={[]}
                errors={problemMap(problems)}
              />
            )}
          />
        </div>
      )}
    </div>
  );
}
