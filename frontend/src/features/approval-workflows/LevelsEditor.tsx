import { ArrowDown, ArrowUp, Plus, Trash2 } from 'lucide-react';
import { Button } from '@/ui/Button/Button';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { problemOf } from '@/ui/FormPanel/problem-of';
import { TextField } from '@/ui/TextField/TextField';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { PersonField } from '@/features/people';
import { emptyLevel, levelFieldName, singlePerson } from './level-drafts';
import type { ApproverType, LevelDraft, LevelField, LevelMode } from './level-drafts';
import { RadioGroup } from './RadioGroup';
import { RuleConditionFields } from './RuleConditionFields';
import { useRuleOptions } from './api';
import type { ApprovalRuleOptions } from './api';
import type { ObjectType } from './schemas';
import styles from './LevelsEditor.module.css';

export interface LevelsEditorProps {
  levels: LevelDraft[];
  onChange: (levels: LevelDraft[]) => void;
  /** The last save's problems; a level's are keyed "levels.<n>.<field>" (n from 1). */
  problems: FieldProblem[];
  objectType: ObjectType;
}

const SOURCES: { value: ApproverType; label: string }[] = [
  { value: 'people', label: 'Named people' },
  { value: 'owner', label: 'Record owner' },
  { value: 'manager', label: "The owner's line manager" },
  { value: 'role', label: 'People with a role' },
  { value: 'recordField', label: 'Person named on the record' },
];

/** Activities have no field that names an approver, so their workflows don't offer one. */
function sourcesOf(objectType: ObjectType) {
  return objectType === 'activity'
    ? SOURCES.filter((source) => source.value !== 'recordField')
    : SOURCES;
}

const MODES: { value: LevelMode; label: string }[] = [
  { value: 'any', label: 'Any one approves' },
  { value: 'all', label: 'All must approve' },
];

const BACKUP_HINT = 'Takes over when an approval is escalated, or when nobody else can approve.';

export function LevelsEditor({ levels, onChange, problems, objectType }: LevelsEditorProps) {
  const needsOptions = levels.some(
    (level) =>
      level.approverType === 'role' ||
      level.approverType === 'recordField' ||
      level.onlyWhen ||
      level.autoWhen,
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
    const level = levels[index];
    const other = levels[index + by];
    if (level && other) {
      onChange(levels.map((each, i) => (i === index ? other : i === index + by ? level : each)));
    }
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
                objectType={objectType}
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
  objectType: ObjectType;
  options: ApprovalRuleOptions | undefined;
  optionsReason: string;
  onChange: (update: Partial<LevelDraft>) => void;
}

function LevelFields({
  level,
  number,
  problems,
  objectType,
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
  const recordFields = (options?.recordFields ?? [])
    .filter((recordField) => recordField.objectType === objectType)
    .map((recordField) => ({ id: recordField.id, label: recordField.label, active: true }));

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
        options={sourcesOf(objectType)}
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
      {level.approverType === 'recordField' && (
        <ChoiceField
          label="Field that names the approver"
          marking="Required"
          name={field('recordField')}
          options={recordFields}
          value={level.recordField}
          onChange={(recordField) => onChange({ recordField })}
          disabledReason={optionsReason || undefined}
          error={problem('recordField')}
        />
      )}
      {!singlePerson(level.approverType) && (
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
            objectType={objectType}
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
            objectType={objectType}
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
