import { useState } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problem-of';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { plural } from '@/format/text';
import { useFormatters } from '@/format/useFormatters';
import styles from '@/ui/settings-table.module.css';
import { CONFLICT_NEXT_STEP } from '../save-failure';
import { fetchFinanceBudget, useRenameFinanceBudget } from './api';
import type { FinanceBudget } from './api';

const NOT_FOUND = 'This finance budget no longer exists — someone may have deleted it.';

export interface RenameBudgetDialogProps {
  budget: FinanceBudget;
  onClose: () => void;
}

/**
 * Rename budget (FR-015): one new name for the whole finance budget — all its items move with it,
 * and the plans using it as their default show the new name.
 */
export function RenameBudgetDialog({ budget, onClose }: RenameBudgetDialogProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const rename = useRenameFinanceBudget();
  const [name, setName] = useState(budget.name);
  // The version the person started from; after a conflict, the one with the other change.
  const [version, setVersion] = useState(budget.version);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const plans = budget.usage.parts
    .filter((part) => part.kind === 'plan')
    .reduce((sum, part) => sum + part.count, 0);
  const itemCount = budget.items.length;

  async function submit() {
    if (name === budget.name) {
      onClose();
      return;
    }
    try {
      const saved = await rename.mutateAsync({ id: budget.id, name, version });
      show({ message: `Budget renamed to '${saved.name}'` });
      onClose();
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      setProblems(apiError.kind === 'validation' ? apiError.fields : []);
      switch (apiError.kind) {
        case 'validation':
        case 'signInRequired':
        case 'sessionExpired':
          setFailure(null);
          break;
        case 'conflict': {
          setFailure({
            message: conflictMessage(apiError, 'finance budget', dateFormat),
            nextStep: CONFLICT_NEXT_STEP,
          });
          // The typed name stays; Rename budget again replaces the other change on purpose.
          const latest = await fetchFinanceBudget(budget.year, budget.id).catch(() => undefined);
          if (latest) {
            setVersion(latest.version);
          }
          break;
        }
        case 'notFound':
          setFailure({ message: NOT_FOUND });
          break;
        default:
          setFailure({ message: apiError.message });
      }
    }
  }

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={`Rename budget '${budget.name}'`}
      submitLabel="Rename budget"
      busy={rename.isPending}
      onSubmit={() => void submit()}
    >
      {failure && <FailureMessage failure={failure} />}
      <TextField
        label="New name"
        name="name"
        marking="Required"
        maxLength={120}
        value={name}
        onChange={setName}
        error={problemOf(problems, 'name')}
      />
      <p className={styles.muted}>
        {`All ${plural(itemCount, 'item')} ${itemCount === 1 ? 'moves' : 'move'} with it.`}
        {plans > 0 &&
          ` ${plural(plans, 'plan')} ${plans === 1 ? 'uses it as its' : 'use it as their'} default finance budget and will show the new name.`}
      </p>
    </FormDialog>
  );
}
