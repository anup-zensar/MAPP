import { useId } from 'react';
import type { PlanningDashboard } from './api';
import type { DashboardFilters as Filters, DashboardKind, PlanType } from './dashboard-links';
import { PLAN_TYPES } from './dashboard-text';
import styles from './Dashboards.module.css';

interface DashboardFiltersProps {
  kind: DashboardKind;
  dashboard: PlanningDashboard;
  onChange: (patch: Partial<Filters>) => void;
}

export function DashboardFilters({ kind, dashboard, onChange }: DashboardFiltersProps) {
  const id = useId();
  const { filters, choices } = dashboard;
  return (
    <div className={styles.filters} role="group" aria-label="Filters">
      <label className={styles.field} htmlFor={`${id}-year`}>
        Year
        <select
          id={`${id}-year`}
          className={styles.select}
          value={filters.year}
          onChange={(event) => onChange({ year: Number(event.target.value), plan: '' })}
        >
          {choices.years.map((year) => (
            <option key={year} value={year}>
              {year}
            </option>
          ))}
        </select>
      </label>
      {kind === 'regional' && (
        <label className={styles.field} htmlFor={`${id}-team`}>
          Market
          <select
            id={`${id}-team`}
            className={styles.select}
            value={filters.teamId ?? ''}
            onChange={(event) => onChange({ team: event.target.value, plan: '' })}
          >
            {choices.teams.map((team) => (
              <option key={team.id} value={team.id}>
                {team.active ? team.name : `${team.name} (inactive)`}
              </option>
            ))}
          </select>
        </label>
      )}
      <label className={styles.field} htmlFor={`${id}-type`}>
        Plan type
        <select
          id={`${id}-type`}
          className={styles.select}
          value={filters.planType ?? ''}
          onChange={(event) => onChange({ planType: planTypeOf(event.target.value), plan: '' })}
        >
          <option value="">All plan types</option>
          {PLAN_TYPES.map((type) => (
            <option key={type.value} value={type.value}>
              {type.label}
            </option>
          ))}
        </select>
      </label>
      <label className={styles.field} htmlFor={`${id}-currency`}>
        Currency
        <select
          id={`${id}-currency`}
          className={styles.select}
          value={filters.currency}
          onChange={(event) => onChange({ currency: event.target.value })}
        >
          {choices.currencies.map((currency) => (
            <option key={currency.code} value={currency.code}>
              {currency.code} · {currency.name}
            </option>
          ))}
        </select>
      </label>
    </div>
  );
}

function planTypeOf(value: string): PlanType | '' {
  return PLAN_TYPES.find((type) => type.value === value)?.value ?? '';
}
