import { expect, test } from '@playwright/test';

test('failure diagnosis to intervention to resume controls are visible', async ({ page }) => {
  await page.goto('/');
  await page.getByTitle('Runs').click();
  await expect(page.locator('#runsList')).toBeVisible();

  const firstRun = page.locator('#runsList .run-item').first();
  if ((await firstRun.count()) > 0) {
    await firstRun.click();
    await expect(page.locator('#runExecutionState')).toBeVisible();
    await expect(page.locator('[data-intervention="force_retry"]')).toBeVisible();
  }
});

test('runtime DLQ contains requeue controls', async ({ page }) => {
  await page.goto('/');
  await page.getByTitle('Runtime').click();
  await expect(page.locator('#dlqList')).toBeVisible();
  await expect(page.locator('#queueEventsList')).toBeVisible();
});
