import { test, expect } from '@playwright/test';
import { resetDB } from '../support/seed';
import { loginAsAdmin } from '../support/auth';

/**
 * Mobile-only: confirms the regression for "tap a sidebar menu item →
 * sidebar auto-collapses on phone width" stays fixed.
 *
 * This file runs under the `mobile-chrome` project (see playwright.config.ts).
 *
 * Determinism notes (previous version was flaky):
 *   - We assert on antd's `ant-layout-sider-collapsed` class (component state)
 *     instead of racing element visibility against the open/close CSS
 *     transition.
 *   - We wait for the sider to be fully expanded (menu item visible) BEFORE
 *     clicking it, so the click can't land on a zero-width (collapsedWidth=0)
 *     menu item that sits under the content area.
 *   - The hamburger is located inside the header, not via an empty-text filter
 *     that matched every button on the page.
 */

test.describe('responsive', () => {
  test.beforeAll(async ({ baseURL }) => {
    await resetDB(baseURL!);
  });

  test('mobile sidebar collapses after menu tap', async ({ page }) => {
    await loginAsAdmin(page);

    const sider = page.locator('.ant-layout-sider');
    // On phone width (< lg = 992px) the Sider auto-collapses on mount via its
    // breakpoint, rendering with collapsedWidth=0. Wait for that to settle.
    await expect(sider).toHaveClass(/ant-layout-sider-collapsed/);

    // Open the drawer via the header hamburger button.
    await page.locator('.ant-layout-header button').first().click();
    await expect(sider).not.toHaveClass(/ant-layout-sider-collapsed/);

    // Tap a menu item (scoped to the sider's menu so we never match page text).
    const patientsItem = sider
      .locator('.ant-menu-item')
      .filter({ hasText: /Patients|病人|ผู้ป่วย/i });
    await expect(patientsItem).toBeVisible();
    await patientsItem.click();

    // It should both navigate and auto-collapse the sider on mobile.
    await expect(page).toHaveURL(/\/admin\/patients/);
    await expect(sider).toHaveClass(/ant-layout-sider-collapsed/);
  });
});
