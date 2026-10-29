import type { Locator, Page } from '@playwright/test';

async function isFocused(target: Locator): Promise<boolean> {
  return target.evaluate((element) => element === document.activeElement, undefined, {
    timeout: 5_000,
  });
}

/** Presses Tab until `target` has the focus; fails when the keyboard never reaches it. */
export async function tabTo(page: Page, target: Locator): Promise<void> {
  for (let presses = 0; presses < 300; presses++) {
    if (await isFocused(target)) {
      return;
    }
    await page.keyboard.press('Tab');
  }
  throw new Error('The keyboard never reached the element');
}

/** Moves through an open menu with the arrow keys and chooses the item called `name`. */
export async function chooseMenuItem(page: Page, name: string): Promise<void> {
  const item = page.getByRole('menuitem', { name, exact: true });
  for (let presses = 0; presses < 20; presses++) {
    if (await isFocused(item)) {
      await page.keyboard.press('Enter');
      return;
    }
    await page.keyboard.press('ArrowDown');
  }
  throw new Error(`The menu item '${name}' never got the focus`);
}
