import { test, expect } from '@grafana/plugin-e2e';

test.describe(
  'Explore query bookmarks',
  {
    tag: ['@various'],
  },
  () => {
    test.describe.configure({ mode: 'serial' });

    const bookmarkName = `E2E bookmark ${Date.now()}`;

    async function openBookmarksModal(page: import('@playwright/test').Page) {
      await page.getByRole('button', { name: 'Bookmarks' }).click();
      const modal = page.getByRole('dialog', { name: 'Query bookmarks' });
      await expect(modal).toBeVisible();
      return modal;
    }

    test('should show Bookmarks button in Explore', async ({ page, selectors, dashboardPage }) => {
      await page.goto('/explore');

      const exploreContainer = dashboardPage.getByGrafanaSelector(selectors.pages.Explore.General.container);
      await expect(exploreContainer).toBeVisible();
      await expect(page.getByRole('button', { name: 'Bookmarks' })).toBeVisible();
    });

    test('should save, list, and delete a bookmark', async ({ page, selectors, dashboardPage }) => {
      await page.goto('/explore');

      const exploreContainer = dashboardPage.getByGrafanaSelector(selectors.pages.Explore.General.container);
      await expect(exploreContainer).toBeVisible();

      const runButton = dashboardPage.getByGrafanaSelector(selectors.components.RefreshPicker.runButtonV2);
      await runButton.click();
      await page.waitForResponse((resp) => resp.url().includes('/api/ds/query') && resp.status() === 200);

      let modal = await openBookmarksModal(page);
      await modal.getByPlaceholder('e.g. CPU usage last 6 hours').fill(bookmarkName);

      const createResponse = page.waitForResponse(
        (resp) => resp.url().includes('/api/explore/bookmarks') && resp.request().method() === 'POST'
      );
      await modal.getByRole('button', { name: 'Save current' }).click();
      await createResponse;

      await expect(modal.getByText(bookmarkName)).toBeVisible({ timeout: 5_000 });

      await modal.getByRole('button', { name: 'Close' }).click();
      await expect(modal).not.toBeVisible();

      modal = await openBookmarksModal(page);
      const bookmarkCard = modal.locator(`[data-testid^="explore-bookmark-"]`).filter({ hasText: bookmarkName });
      await expect(bookmarkCard).toBeVisible({ timeout: 5_000 });

      await bookmarkCard.getByRole('button', { name: 'Delete bookmark' }).click();
      const confirmModal = page.getByRole('dialog', { name: 'Delete bookmark' });
      await expect(confirmModal).toBeVisible();

      const deleteResponse = page.waitForResponse(
        (resp) => resp.url().includes('/api/explore/bookmarks/') && resp.request().method() === 'DELETE'
      );
      await confirmModal.getByRole('button', { name: 'Delete' }).click();
      await deleteResponse;

      await expect(modal.getByText(bookmarkName)).not.toBeVisible();
    });

    test('should persist bookmarks after page reload', async ({ page, selectors, dashboardPage }) => {
      await page.goto('/explore');

      const exploreContainer = dashboardPage.getByGrafanaSelector(selectors.pages.Explore.General.container);
      await expect(exploreContainer).toBeVisible();

      const modal = await openBookmarksModal(page);
      await expect(modal.getByText(bookmarkName)).not.toBeVisible();

      await modal.getByRole('button', { name: 'Close' }).click();

      const runButton = dashboardPage.getByGrafanaSelector(selectors.components.RefreshPicker.runButtonV2);
      await runButton.click();
      await page.waitForResponse((resp) => resp.url().includes('/api/ds/query') && resp.status() === 200);

      const reloadModal = await openBookmarksModal(page);
      await reloadModal.getByPlaceholder('e.g. CPU usage last 6 hours').fill(bookmarkName);

      const createResponse = page.waitForResponse(
        (resp) => resp.url().includes('/api/explore/bookmarks') && resp.request().method() === 'POST'
      );
      await reloadModal.getByRole('button', { name: 'Save current' }).click();
      await createResponse;
      await expect(reloadModal.getByText(bookmarkName)).toBeVisible({ timeout: 5_000 });
      await reloadModal.getByRole('button', { name: 'Close' }).click();

      await page.reload();
      await expect(exploreContainer).toBeVisible();

      const persistedModal = await openBookmarksModal(page);
      await expect(persistedModal.getByText(bookmarkName)).toBeVisible({ timeout: 5_000 });
    });
  }
);
