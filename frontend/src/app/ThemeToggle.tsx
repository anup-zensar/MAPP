import { Moon, Sun } from 'lucide-react';
import { Button, Tooltip } from '@/components';
import styles from './ThemeToggle.module.css';
import { useSession } from './useSession';

export interface ThemeToggleProps {
  /**
   * Writes the button's name next to the icon instead of showing a tooltip — in the menu drawer,
   * where a tooltip would swallow the first Escape (research R14).
   */
  withLabel?: boolean;
}

/**
 * The light/dark switch (FR-014): it offers the other mode — a moon while the app is light, a
 * sun while it is dark. Signed in, the choice is saved for the person; signed out, only in this
 * browser (research R16).
 */
export function ThemeToggle({ withLabel = false }: ThemeToggleProps) {
  const { theme, updatePreferences } = useSession();
  const offered = theme === 'dark' ? 'light' : 'dark';
  const name = offered === 'dark' ? 'Switch to dark mode' : 'Switch to light mode';

  const button = (
    <Button
      variant="ghost"
      icon={offered === 'dark' ? <Moon /> : <Sun />}
      aria-label={name}
      className={withLabel ? styles.labelled : undefined}
      onClick={() => updatePreferences({ theme: offered })}
    >
      {withLabel ? (offered === 'dark' ? 'Dark mode' : 'Light mode') : undefined}
    </Button>
  );

  return withLabel ? button : <Tooltip content={name}>{button}</Tooltip>;
}
