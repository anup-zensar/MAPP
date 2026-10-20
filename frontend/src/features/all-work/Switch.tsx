import styles from './Switch.module.css';

export interface SwitchProps {
  /** The words on the switch; they are its name too. */
  label: string;
  /**
   * The words shown below 600 px, where the toolbar has one row: the start of `label`, so the
   * visible words stay part of the name (WCAG 2.5.3).
   */
  shortLabel?: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}

/**
 * An on/off switch with its words beside it (the Unassigned planned cost switch, FR-012): a
 * button with role="switch", so screen readers say "on" or "off".
 */
export function Switch({ label, shortLabel, checked, onChange }: SwitchProps) {
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
      {shortLabel && label.startsWith(shortLabel) ? (
        <span>
          {shortLabel}{' '}
          <span className={styles.rest}>{label.slice(shortLabel.length).trimStart()}</span>
        </span>
      ) : (
        label
      )}
    </button>
  );
}
