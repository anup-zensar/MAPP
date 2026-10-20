import { ArrowDown, ArrowUp, CircleAlert, Plus, Trash2 } from 'lucide-react';
import { Button, CheckboxField, ChoiceField, problemOf, TextField } from '@/components';
import type { FieldProblem } from '@/components';
import { PersonField } from '@/features/people';
import { emptyLevel, levelFieldName } from './levelDrafts';
import type { ApproverType, LevelDraft, LevelField, LevelMode } from './levelDrafts';
import { RuleConditionFields } from './RuleConditionFields';
import { useRuleOptions } from './workflowsApi';
import type { ApprovalRuleOptions } from './workflowsApi';
import styles from './LevelsEditor.module.css';

export interface LevelsEditorProps {
  levels: LevelDraft[];
  onChange: (levels: LevelDraft[]) => void;
  /** The last save's problems; a level's are keyed "levels.<n>.<field>" (n from 1). */
  problems: FieldProblem[];
}

const SOURCES: { value: ApproverType; label: string }[] = [
  { value: 'people', label: 'Named people' },
  { value: 'manager', label: "The owner's line manager" },
  { value: 'role', label: 'People with a role' },
];

const MODES: { value: LevelMode; label: string }[] = [
  { value: 'any', label: 'Any one approves' },
  { value: 'all', label: 'All must approve' },
];

const BACKUP_HINT = 'Takes over when an approval is escalated, or when nobody else can approve.';

/**
 * The levels of an approval workflow (008 FR-002, FR-003; 017 FR-001 – FR-006): one block per
 * level, in order — its name, who approves (named people, the owner's line manager or the people
 * with a role), whether any one or all of them must approve, when the level is needed at all or
 * approved automatically, and its backup approver — with Move up, Move down and Remove named per
 * level, and **Add level**. Each of the API's problems shows at its field.
 */
export function LevelsEditor({ levels, onChange, problems }: LevelsEditorProps) {
  // The roles, business units and activity types load once a level needs them.
  const needsOptions = levels.some(
    (level) => level.approverType === 'role' || level.onlyWhen || level.autoWhen,
  );
  const ruleOptions = useRuleOptions(needsOptions);
  const optionsReason = ruleOptions.isError
    ? "The choices couldn't be loaded. Try again in a moment."
    : ruleOptions.data
      ? ''
      : 'Loading the choices…';

  function change(index: number, update: Partial<LevelDraft>) {
    onChange(levels.map((level, i) => (i === index ? { ...level, ...update } : level)));
  }

  function move(index: number, by: -1 | 1) {
    const next = [...levels];
    [next[index], next[index + by]] = [next[index + by], next[index]];
    onChange(next);
  }

  return (
    <div className={styles.editor}>
      {needsOptions && ruleOptions.isError && (
        <div className={styles.loadFailed}>
          <p>{optionsReason}</p>
          <Button size="sm" onClick={() => void ruleOptions.refetch()}>
            Try again
          </Button>
        </div>
      )}
      <ol role="list" className={styles.levels}>
        {levels.map((level, index) => {
          const number = index + 1;
          return (
            <li key={level.key} className={styles.level}>
              <div className={styles.header}>
                <h3 className={styles.heading}>{`Level ${number}`}</h3>
                <div className={styles.tools}>
                  {index > 0 && (
                    <Button
                      variant="tertiary"
                      size="sm"
                      icon={<ArrowUp />}
                      aria-label={`Move level ${number} up`}
                      onClick={() => move(index, -1)}
                    />
                  )}
                  {index < levels.length - 1 && (
                    <Button
                      variant="tertiary"
                      size="sm"
                      icon={<ArrowDown />}
                      aria-label={`Move level ${number} down`}
                      onClick={() => move(index, 1)}
                    />
                  )}
                  {levels.length > 1 && (
                    <Button
                      variant="tertiary"
                      size="sm"
                      icon={<Trash2 />}
                      aria-label={`Remove level ${number}`}
                      onClick={() => onChange(levels.filter((_, i) => i !== index))}
                    />
                  )}
                </div>
              </div>
              <LevelFields
                level={level}
                number={number}
                problems={problems}
                options={ruleOptions.data}
                optionsReason={optionsReason}
                onChange={(update) => change(index, update)}
              />
            </li>
          );
        })}
      </ol>
      <div>
        <Button icon={<Plus />} onClick={() => onChange([...levels, emptyLevel()])}>
          Add level
        </Button>
      </div>
    </div>
  );
}

interface LevelFieldsProps {
  level: LevelDraft;
  number: number;
  problems: FieldProblem[];
  options: ApprovalRuleOptions | undefined;
  optionsReason: string;
  onChange: (update: Partial<LevelDraft>) => void;
}

/** The fields of one level, in the order of 017 plan "Settings › Approval workflows". */
function LevelFields({
  level,
  number,
  problems,
  options,
  optionsReason,
  onChange,
}: LevelFieldsProps) {
  const field = (name: LevelField) => levelFieldName(number, name);
  const problem = (name: LevelField) => problemOf(problems, field(name));
  const roles = (options?.roles ?? []).map((role) => ({
    id: role.id,
    label: role.name,
    active: true,
  }));

  return (
    <>
      <TextField
        label="Level name"
        name={field('name')}
        hint={`Shown as 'Level ${number}' when empty`}
        maxLength={60}
        value={level.name}
        onChange={(name) => onChange({ name })}
        error={problem('name')}
      />
      <RadioGroup
        legend="Who approves"
        name={field('approverType')}
        options={SOURCES}
        value={level.approverType}
        onChange={(approverType) => onChange({ approverType })}
        error={problem('approverType')}
      />
      {level.approverType === 'people' && (
        <PersonField
          multiple
          label="Approvers"
          marking="Required"
          name={field('approverIds')}
          value={level.approverIds}
          onChange={(approverIds) => onChange({ approverIds })}
          error={problem('approverIds')}
        />
      )}
      {level.approverType === 'role' && (
        <ChoiceField
          label="Role"
          marking="Required"
          hint="Only roles that approve activities are offered."
          name={field('roleId')}
          options={roles}
          value={level.roleId}
          onChange={(roleId) => onChange({ roleId })}
          disabledReason={optionsReason || undefined}
          error={problem('roleId')}
        />
      )}
      {level.approverType !== 'manager' && (
        <RadioGroup
          legend="When there are several approvers"
          name={field('mode')}
          options={MODES}
          value={level.mode}
          onChange={(mode) => onChange({ mode })}
          error={problem('mode')}
        />
      )}
      <div className={styles.rule}>
        <CheckboxField
          label="Only needed when"
          name={field('appliesWhen')}
          checked={level.onlyWhen}
          onChange={(onlyWhen) => onChange({ onlyWhen })}
          error={problem('appliesWhen')}
        />
        {level.onlyWhen && (
          <RuleConditionFields
            label={`Level ${number}: only needed when`}
            idPrefix={field('appliesWhen')}
            value={level.appliesWhen}
            onChange={(appliesWhen) => onChange({ appliesWhen })}
            options={options}
            optionsReason={optionsReason}
          />
        )}
      </div>
      <div className={styles.rule}>
        <CheckboxField
          label="Approve automatically when"
          name={field('autoApproveWhen')}
          checked={level.autoWhen}
          onChange={(autoWhen) => onChange({ autoWhen })}
          error={problem('autoApproveWhen')}
        />
        {level.autoWhen && (
          <RuleConditionFields
            label={`Level ${number}: approve automatically when`}
            idPrefix={field('autoApproveWhen')}
            value={level.autoApproveWhen}
            onChange={(autoApproveWhen) => onChange({ autoApproveWhen })}
            options={options}
            optionsReason={optionsReason}
          />
        )}
      </div>
      <PersonField
        label="Backup approver"
        hint={BACKUP_HINT}
        name={field('backupApproverId')}
        value={level.backupApproverId}
        onChange={(backupApproverId) => onChange({ backupApproverId })}
        error={problem('backupApproverId')}
      />
    </>
  );
}

interface RadioGroupProps<T extends string> {
  legend: string;
  /** The group's name and id, so the form's problem summary can bring the focus here. */
  name: string;
  options: { value: T; label: string }[];
  value: T;
  onChange: (value: T) => void;
  error?: string;
}

/** A small set of choices shown at once, named by its legend; its problem under it. */
function RadioGroup<T extends string>({
  legend,
  name,
  options,
  value,
  onChange,
  error,
}: RadioGroupProps<T>) {
  const errorId = `${name}-error`;
  return (
    <fieldset
      id={name}
      tabIndex={-1}
      className={styles.radioGroup}
      aria-describedby={error ? errorId : undefined}
    >
      <legend className={styles.legend}>{legend}</legend>
      <div className={styles.radios}>
        {options.map((option) => (
          <label key={option.value} className={styles.radio}>
            <input
              type="radio"
              name={name}
              value={option.value}
              checked={value === option.value}
              onChange={() => onChange(option.value)}
            />
            {option.label}
          </label>
        ))}
      </div>
      {error && (
        <p id={errorId} className={styles.problem}>
          <CircleAlert aria-hidden="true" />
          {error}
        </p>
      )}
    </fieldset>
  );
}
