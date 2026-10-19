import styles from './Switch.module.css';

export interface SwitchProps {
  /** The words on the switch; they are its name too. */
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}

/**
 * An on/off switch with its words beside it (the Unassigned planned cost switch, FR-012): a
 * button with role="switch", so screen readers say "on" or "off".
 */
export function Switch({ label, checked, onChange }: SwitchProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      className={styles.switch}
      onClick={() => onChange(!checked)}
    >
      <span className={styles.track} aria-hidden="true">
        <span className={styles.thumb} />
      </span>
      {label}
    </button>
  );
}
