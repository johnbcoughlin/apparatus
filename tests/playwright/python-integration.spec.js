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
    await apparatusAPI.logMetric(runId, 'accuracy', 10, 0.92);
    await apparatusAPI.logMetric(runId, 'loss', 10, 0.15);

    // Step 4: Upload an artifact
    const artifactContent = 'Model training completed successfully\nFinal accuracy: 0.92\n';
    await apparatusAPI.logArtifact(runId, 'results/training_log.txt', artifactContent, 'training_log.txt');

    // Step 5: Navigate to homepage
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Step 6: Click on the Default experiment (runs are now under experiments)
    const experimentLink = page.getByRole('link', { name: 'Default' });
    await expect(experimentLink).toBeVisible();
    await experimentLink.click();
    await page.waitForLoadState('networkidle');

    // Step 7: Find and click the run link on the experiment page
    const runLink = page.getByRole('link', { name: runName });
    await expect(runLink).toBeVisible();
    await runLink.click();

    // Step 8: Verify we're on the run page and the overview tab loads automatically
    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(`/runs/${runId}`);
    await expect(page.locator('text=Run: ' + runName)).toBeVisible();

    // Step 9: Wait for the overview tab content to load (it loads automatically via htmx)
    // The overview content loads into #tab-content div automatically
    await page.waitForSelector('text=Parameters', { timeout: 10000 });

    // Step 10: Verify parameters are displayed
    await expect(page.getByRole('cell').filter({hasText: 'learning_rate'})).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: '0.001'})).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: 'batch_size'})).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: '1000'})).toBeVisible();

    // Step 11: Verify metrics are displayed
    await expect(page.locator('text=Metrics')).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: 'accuracy'})).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: '0.92'})).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: 'loss'})).toBeVisible();
    await expect(page.getByRole('cell').filter({hasText: '0.15'})).toBeVisible();

    // Step 12: Navigate to artifacts tab by clicking the button
    await page.getByRole('tab', { name: 'Artifacts' }).click();
    await page.waitForLoadState('networkidle');

    // Step 13: Verify artifact is listed and click on it
    await page.waitForSelector('text=Artifacts');
    const artifactButton = page.locator('button', { hasText: 'training_log.txt' });
    await expect(artifactButton).toBeVisible();
    await artifactButton.click();

    // Step 14: Wait for artifact display to load
    await page.waitForLoadState('networkidle');
    // Verify artifact URI is displayed in the artifact-display div
    await expect(page.locator('#artifact-display')).toContainText('results/training_log.txt');

    // Step 15: Navigate back to Overview tab
    await page.getByRole('tab', { name: 'Overview' }).click();
    await page.waitForLoadState('networkidle');

    // Step 16: Verify we're back on overview and can see params
    await expect(page.locator('text=Parameters')).toBeVisible();
    await expect(page.locator('text=learning_rate')).toBeVisible();

    // Step 17: Navigate back to Artifacts tab
    await page.getByRole('tab', { name: 'Artifacts' }).click();
    await page.waitForLoadState('networkidle');

    // Step 18: Verify the artifact is still selected and its URI is still displayed
    // This tests that the selected artifact persists across tab navigation
    await expect(page.locator('#artifact-display')).toContainText('results/training_log.txt');
  });

  test('notes persist after page reload', async ({ page, apparatusAPI }) => {
    // Step 1: Create a run
    const timestamp = Date.now();
    const runName = `notes-test-run-${timestamp}`;
    const runId = await apparatusAPI.createRun(runName);

    // Step 2: Navigate to the run page
    await page.goto(`/runs/${runId}`);
    await page.waitForLoadState('networkidle');

    // Step 3: Wait for the overview tab content to load
    await page.waitForSelector('#notes-form', { timeout: 10000 });

    // Step 4: Enter a note in the textarea
    const testNote = `Test note created at ${timestamp}`;
    const textarea = page.locator('#notes-form textarea');
    await textarea.fill(testNote);

    // Step 5: Click the Save button
    await page.locator('#notes-form button[type="submit"]').click();

    // Step 6: Wait for htmx to swap in the updated form
    await page.waitForLoadState('networkidle');

    // Step 7: Verify the note is still in the textarea after save
    await expect(textarea).toHaveValue(testNote);

    // Step 8: Reload the page
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Step 9: Wait for the overview tab content to load again
    await page.waitForSelector('#notes-form', { timeout: 10000 });

    // Step 10: Verify the note persisted after reload
    await expect(page.locator('#notes-form textarea')).toHaveValue(testNote);
  });

  test('nested runs expand/collapse and URL state is preserved on navigation', async ({ page, apparatusAPI }) => {
    // Step 1: Create a forest of nested runs (parent -> child -> grandchild)
    const timestamp = Date.now();

    // Create first parent with children and grandchildren
    const parent1Name = `parent-1-${timestamp}`;
    const parent1Id = await apparatusAPI.createRun(parent1Name);

    const child1aName = `child-1a-${timestamp}`;
    const child1aId = await apparatusAPI.createRun(child1aName, parent1Id);

    const grandchild1a1Name = `grandchild-1a1-${timestamp}`;
    await apparatusAPI.createRun(grandchild1a1Name, child1aId);

    const grandchild1a2Name = `grandchild-1a2-${timestamp}`;
    await apparatusAPI.createRun(grandchild1a2Name, child1aId);

    // Create second parent with a child (no grandchildren)
    const parent2Name = `parent-2-${timestamp}`;
    const parent2Id = await apparatusAPI.createRun(parent2Name);

    const child2aName = `child-2a-${timestamp}`;
    await apparatusAPI.createRun(child2aName, parent2Id);

    // Step 2: Navigate to the default experiment page
    await page.goto('/experiments/00000000-0000-0000-0000-000000000000');
    await page.waitForLoadState('networkidle');

    // Step 3: Verify parent runs are visible with children indicators
    const parent1Summary = page.locator('summary').filter({ hasText: parent1Name });
    await expect(parent1Summary).toBeVisible();
    await expect(parent1Summary.locator('text=(1 children)')).toBeVisible();

    // Step 4: Verify the details are initially collapsed (child runs not visible)
    await expect(page.getByText(child1aName)).not.toBeVisible();

    // Step 5: Click the parent's summary to expand
    await parent1Summary.click();
    await page.waitForLoadState('networkidle');

    // Step 6: Verify child is now visible and URL has open_l0 param
    await expect(page.getByText(child1aName)).toBeVisible();
    await expect(page).toHaveURL(new RegExp(`open_l0=${parent1Id}`));

    // Step 7: Expand the child to see grandchildren
    const child1aSummary = page.locator('summary').filter({ hasText: child1aName });
    await child1aSummary.click();
    await page.waitForLoadState('networkidle');

    // Step 8: Verify grandchildren are visible and URL has both open_l0 and open_l1
    await expect(page.getByText(grandchild1a1Name)).toBeVisible();
    await expect(page.getByText(grandchild1a2Name)).toBeVisible();
    await expect(page).toHaveURL(new RegExp(`open_l0=${parent1Id}`));
    await expect(page).toHaveURL(new RegExp(`open_l1=${child1aId}`));

    // Step 9: Click into a grandchild run to navigate away
    await page.getByRole('link', { name: grandchild1a1Name }).click();
    await page.waitForLoadState('networkidle');

    // Step 10: Verify we're on the run page
    await expect(page).toHaveURL(new RegExp('/runs/'));
    await expect(page.locator('text=Run: ' + grandchild1a1Name)).toBeVisible();

    // Step 11: Navigate back using browser back button
    await page.goBack();
    await page.waitForLoadState('networkidle');

    // Step 12: Verify the expansion state is preserved - both parent and child should still be expanded
    await expect(page.getByText(child1aName)).toBeVisible();
    await expect(page.getByText(grandchild1a1Name)).toBeVisible();
    await expect(page.getByText(grandchild1a2Name)).toBeVisible();

    // Step 13: Verify URL still has the open params
    await expect(page).toHaveURL(new RegExp(`open_l0=${parent1Id}`));
    await expect(page).toHaveURL(new RegExp(`open_l1=${child1aId}`));

    // Step 14: Collapse the child by clicking its summary again (re-query after navigation)
    const child1aSummaryAfterBack = page.locator('summary').filter({ hasText: child1aName });
    await child1aSummaryAfterBack.click();

    // Step 15: Wait for URL to change (htmx should remove open_l1 or set it empty)
    // The toggle sets open_l1 to empty when collapsing
    await page.waitForURL(url => !url.toString().includes(`open_l1=${child1aId}`), { timeout: 10000 });
    await page.waitForLoadState('networkidle');

    // Step 16: Verify grandchildren are no longer visible but child is still visible
    await expect(page.getByText(grandchild1a1Name)).not.toBeVisible();
    await expect(page.getByText(child1aName)).toBeVisible();

    // Step 17: Verify open_l0 is still present
    await expect(page).toHaveURL(new RegExp(`open_l0=${parent1Id}`));
  });
});
