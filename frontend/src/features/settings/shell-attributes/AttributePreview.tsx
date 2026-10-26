import { useState } from 'react';
import { ApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { CompletenessChip } from '@/ui/CompletenessChip/CompletenessChip';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { ShellAttributeFields } from '@/ui/ShellAttributeFields/ShellAttributeFields';
import { businessUnitOf, missingAttributes } from '@/ui/ShellAttributeFields/rules';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import type { AttributeForm, AttributeValues } from '@/ui/ShellAttributeFields/rules';
import { useAttributeForm } from '../api';
import { LEVEL_WORDS } from './attribute-words';
import { checkAttributeValues } from './api';
import type { AttributeCheckResult, ShellAttribute } from './api';
import styles from './AttributePreview.module.css';

/**
 * Preview — <Level> form (feature 002, FR-020, US3): the level's Shell attributes exactly as
 * people will see them, with this attribute highlighted. Below the plan level it can be shown for
 * any business unit, so conditions can be checked; dependent choices, auto-fill and clearing
 * behave as on the real forms. **Try saving** asks the API what a save would say. Nothing is
 * saved.
 */
export function AttributePreview({ attribute }: { attribute: ShellAttribute }) {
  const form = useAttributeForm(attribute.level);
  const title = `Preview — ${LEVEL_WORDS[attribute.level]} form`;

  return (
    <Card title={title}>
      {form.data ? (
        <PreviewForm form={form.data} attributeId={attribute.id} />
      ) : form.isError ? (
        <ErrorState error={form.error} onRetry={() => void form.refetch()} />
      ) : (
        <Skeleton lines={5} height="18px" />
      )}
    </Card>
  );
}

const NO_PROBLEMS: AttributeCheckResult['problems'] = [];

interface PreviewFormProps {
  form: AttributeForm;
  attributeId: string;
}

/** The preview's own values, which nobody saves, and the answer of the last Try saving. */
function PreviewForm({ form, attributeId }: PreviewFormProps) {
  const isPlan = form.level === 'plan';
  const unitOptions = form.businessUnitOptions.filter((option) => option.active);
  // Below the plan: a plan whose business unit includes every business unit, at first.
  const [units, setUnits] = useState<string[]>(() => unitOptions.map((option) => option.id));
  const [values, setValues] = useState<AttributeValues>({});
  const [result, setResult] = useState<AttributeCheckResult | null>(null);
  const [checkFailure, setCheckFailure] = useState<string | null>(null);
  const [checking, setChecking] = useState(false);

  const businessUnit = isPlan ? null : units;
  // The fields still needed: as the API answered for these values, or worked out while typing.
  const missing =
    result?.missing ?? missingAttributes(form, values, businessUnitOf(form, values, businessUnit));
  const problems = result?.problems ?? NO_PROBLEMS;
  const errors = Object.fromEntries(problems.map((problem) => [problem.field, problem.message]));

  /** Any change makes the last answer out of date. */
  function forget() {
    setResult(null);
    setCheckFailure(null);
  }

  async function trySaving() {
    if (checking) {
      return;
    }
    setChecking(true);
    setCheckFailure(null);
    try {
      setResult(
        await checkAttributeValues(form.level, {
          values,
          ...(isPlan ? {} : { businessUnitOptionIds: units }),
        }),
      );
    } catch (error) {
      const failure = error instanceof ApiError ? error : new ApiError('other');
      // An ended session is left to the sign-in page, which says so.
      if (failure.kind !== 'signInRequired' && failure.kind !== 'sessionExpired') {
        setCheckFailure(failure.message);
      }
    } finally {
      setChecking(false);
    }
  }

  return (
    <div className={styles.preview}>
      <p className={styles.note}>Behaves like the real form. Nothing you type here is saved.</p>
      {!isPlan && unitOptions.length > 0 && (
        <ChoiceField
          label="Preview as a plan whose business unit includes:"
          name="preview-business-unit"
          multiple
          options={unitOptions.map((option) => ({
            id: option.id,
            label: option.label,
            active: option.active,
          }))}
          value={units}
          onChange={(next) => {
            setUnits(next);
            forget();
          }}
        />
      )}
      <div className={styles.completeness}>
        <CompletenessChip
          missing={missing.map((field) => ({ id: field.field, label: field.label }))}
          onSelectField={(id) => document.getElementById(id)?.focus()}
        />
      </div>
      <ShellAttributeFields
        form={form}
        values={values}
        onChange={(next) => {
          setValues(next);
          forget();
        }}
        businessUnit={businessUnit}
        errors={errors}
        highlightAttributeId={attributeId}
      />
      <div className={styles.trySaving}>
        <Button aria-busy={checking || undefined} onClick={() => void trySaving()}>
          Try saving
        </Button>
        <p className={styles.note}>Shows what saving a real record would say. Nothing is saved.</p>
      </div>
      {checkFailure && (
        <p role="alert" className={styles.failure}>
          {checkFailure}
        </p>
      )}
      {result && <CheckAnswer result={result} />}
    </div>
  );
}

/** What saving would say: every problem at once, or that it would work; values it would clear. */
function CheckAnswer({ result }: { result: AttributeCheckResult }) {
  return (
    <div className={styles.answer}>
      {result.problems.length > 0 ? (
        <FormErrorSummary errors={result.problems} />
      ) : (
        <p role="status" className={styles.works}>
          Saving would work.
        </p>
      )}
      {result.cleared.length > 0 && (
        <div>
          <p>These values would be cleared:</p>
          <ul className={styles.cleared}>
            {result.cleared.map((item) => (
              <li key={item.attributeId}>{`${item.label}: ${item.optionLabels.join(', ')}`}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
