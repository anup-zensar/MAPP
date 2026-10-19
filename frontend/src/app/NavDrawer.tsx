import { Menu, X } from 'lucide-react';
import { Dialog } from 'radix-ui';
import { useEffect, useState } from 'react';
import { useLocation } from 'react-router';
import { Button, Tooltip } from '@/components';
import styles from './NavDrawer.module.css';
import { Sidebar } from './Sidebar';

/** From this width on the sidebar is shown, so the drawer is not needed (plan "Layout"). */
const SIDEBAR_SHOWN = '(min-width: 768px)';

/**
 * "Open menu" on narrow screens and the drawer it opens (FR-011, US2 scenario 4): the whole
 * sidebar slides in from the left over a dimmed page. It closes on Escape, on Close menu, on a
 * click outside it, after choosing an item, and when the window becomes wide enough for the
 * sidebar.
 */
export function NavDrawer() {
  const location = useLocation();
  // The drawer belongs to the address it was opened on: any navigation — choosing an item, My
  // profile, Back — leaves it closed, without an effect to close it.
  const [openOn, setOpenOn] = useState<string | null>(null);
  const open = openOn === location.key;

  useEffect(() => {
    const sidebarShown = window.matchMedia(SIDEBAR_SHOWN);
    function closeWhenSidebarShows() {
      if (sidebarShown.matches) {
        setOpenOn(null);
      }
    }
    sidebarShown.addEventListener('change', closeWhenSidebarShows);
    return () => sidebarShown.removeEventListener('change', closeWhenSidebarShows);
  }, []);

  return (
    <Dialog.Root open={open} onOpenChange={(isOpen) => setOpenOn(isOpen ? location.key : null)}>
      <Tooltip content="Open menu">
        <Dialog.Trigger asChild>
          <Button variant="ghost" icon={<Menu />} aria-label="Open menu" />
        </Dialog.Trigger>
      </Tooltip>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.overlay} />
        {/* No description: tell Radix so, instead of pointing to a missing text. */}
        <Dialog.Content className={styles.drawer} aria-describedby={undefined}>
          <Dialog.Title className="visually-hidden">Menu</Dialog.Title>
          {/* No tooltip: inside a dialog it would swallow the first Escape (research R14). */}
          <Dialog.Close asChild>
            <Button variant="ghost" icon={<X />} aria-label="Close menu" className={styles.close} />
          </Dialog.Close>
          <Sidebar inDrawer />
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
