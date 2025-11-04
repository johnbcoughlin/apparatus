import { test, expect } from './fixtures.js';

test.describe('Apparatus End-to-End Tests', () => {
  test('create run with params, metrics, and artifact, then verify via UI navigation', async ({ page, apparatusAPI }) => {
    // Step 1: Create a run with a unique name
    const timestamp = Date.now();
    const runName = `e2e-test-run-${timestamp}`;
    const runId = await apparatusAPI.createRun(runName);

    // Step 2: Log parameters
    await apparatusAPI.logParam(runId, 'learning_rate', 0.001, 'float');
    await apparatusAPI.logParam(runId, 'batch_size', 1000, 'int');

    // Step 3: Log metrics
    await apparatusAPI.logMetric(runId, 'accuracy', 0.92, { step: 10 });
    await apparatusAPI.logMetric(runId, 'loss', 0.15, { step: 10 });

    // Step 4: Upload an artifact
    const artifactContent = 'Model training completed successfully\nFinal accuracy: 0.92\n';
    await apparatusAPI.logArtifact(runId, 'results/training_log.txt', artifactContent, 'training_log.txt');

    // Step 5: Navigate to homepage
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Step 6: Find and click the run link on the homepage
    const runLink = page.getByRole('link', { name: runName });
    await expect(runLink).toBeVisible();
    await runLink.click();

    // Step 7: Verify we're on the run page and the overview tab loads automatically
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(`/runs/${runId}`);
    await expect(page.locator('text=Run: ' + runName)).toBeVisible();

    // Step 8: Wait for the overview tab content to load (it loads automatically via htmx)
    // The overview content loads into #tab-content div automatically
    await page.waitForSelector('text=Parameters', { timeout: 10000 });

    // Step 9: Verify parameters are displayed
    await expect(page.getByRole('cell').filter({hasText: 'learning_rate'})).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: '0.001'})).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: 'batch_size'})).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: '1000'})).toBeVisible();

    // Step 10: Verify metrics are displayed
    await expect(page.locator('text=Metrics')).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: 'accuracy'})).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: '0.92'})).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: 'loss'})).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: '0.15'})).toBeVisible();

    // Step 11: Navigate to artifacts tab by clicking the button
    await page.getByRole('tab', { name: 'Artifacts' }).click();
    await page.waitForLoadState('networkidle');

    // Step 12: Verify artifact is listed and click on it
    await page.waitForSelector('text=Artifacts');
    const artifactButton = page.locator('button', { hasText: 'training_log.txt' });
    await expect(artifactButton).toBeVisible();
    await artifactButton.click();

    // Step 13: Wait for artifact display to load
    await page.waitForLoadState('networkidle');
    // Verify artifact URI is displayed in the artifact-display div
    await expect(page.locator('#artifact-display')).toContainText('results/training_log.txt');

    // Step 14: Navigate back to Overview tab
    await page.getByRole('tab', { name: 'Overview' }).click();
    await page.waitForLoadState('networkidle');

    // Step 15: Verify we're back on overview and can see params
    await expect(page.locator('text=Parameters')).toBeVisible();
    await expect(page.locator('text=learning_rate')).toBeVisible();

    // Step 16: Navigate back to Artifacts tab
    await page.getByRole('tab', { name: 'Artifacts' }).click();
    await page.waitForLoadState('networkidle');

    // Step 17: Verify the artifact is still selected and its URI is still displayed
    // This tests that the selected artifact persists across tab navigation
    await expect(page.locator('#artifact-display')).toContainText('results/training_log.txt');
  });
});
