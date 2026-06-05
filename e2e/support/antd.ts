import { expect } from '@playwright/test';
import type { Page, Locator } from '@playwright/test';

/**
 * Helpers for driving antd v5 widgets (Select / DatePicker / TimePicker) from
 * Playwright. antd assigns the Form.Item `name` as the control's DOM `id`, so
 * fields are targetable as `#date`, `#startTime`, etc. without extra test-ids.
 */

/**
 * Open an antd showSearch Select (by clicking `opener`), type to filter, then
 * commit the highlighted match with Enter.
 *
 * We type + Enter rather than clicking the option element because clicking an
 * option mid dropdown-animation is flaky (the element is "not stable" and the
 * dropdown can close out from under the click). Enter selects whatever antd
 * has highlighted (the first match), which is deterministic.
 */
export async function pickOption(page: Page, opener: Locator, searchText: string) {
  await opener.click();
  await page.keyboard.type(searchText);
  // Click the visible option. We target `.ant-select-item-option` (the rendered
  // clickable row) rather than role=option, because antd also emits a hidden
  // a11y listbox whose role=option nodes are not visible/clickable.
  await page.locator('.ant-select-item-option', { hasText: searchText }).first().click();
}

/**
 * Type a value into an antd DatePicker / TimePicker input and confirm it.
 *
 * We deliberately avoid pressing Enter to confirm a TimePicker: Enter in a
 * field inside an antd Form submits the whole Form, which would create the
 * schedule prematurely. Instead we click the TimePicker panel's OK button,
 * which commits the value and closes the panel without submitting. DatePicker
 * (date-only) has no OK button, so we fall back to Enter — and the date field
 * is only ever filled while the form is still incomplete, so that Enter is a
 * harmless no-op submit.
 */
export async function fillPicker(input: Locator, value: string) {
  const page = input.page();
  const openDropdowns = page.locator('.ant-picker-dropdown:not(.ant-picker-dropdown-hidden)');
  // Make sure no other picker dropdown is still open/animating before we start,
  // otherwise the OK button below could target the wrong (stale) panel and the
  // value we type here would never get committed to form state.
  await expect(openDropdowns).toHaveCount(0);

  // force: pickers sit deep in a scrollable modal body and a just-closed
  // dropdown can briefly intercept the click; the input is the right target.
  await input.click({ force: true });
  await input.fill(value);

  const okButton = openDropdowns.locator('.ant-picker-ok button').last();
  if (await okButton.isVisible().catch(() => false)) {
    // force: the TimePicker columns scroll-animate to the typed value, so the
    // OK button is briefly "not stable"; it's the right element regardless.
    await okButton.click({ force: true });
  } else {
    await input.press('Enter');
  }
  // Wait for this picker's dropdown to fully close so the next fillPicker starts
  // from a clean slate (see the count==0 guard above).
  await expect(openDropdowns).toHaveCount(0);
}
