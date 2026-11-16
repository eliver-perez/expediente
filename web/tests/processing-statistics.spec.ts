import { expect, test, type Route } from '@playwright/test';

test('estadísticas persistentes, respuestas incompletas, errores y progreso independiente', async ({ page }) => {
  await page.goto('/login');
  await page.getByLabel('Usuario', { exact: true }).fill('admin-e2e');
  await page.getByLabel('Contraseña', { exact: true }).fill('Browser-fixture-password-2026');
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await page.getByRole('link', { name: 'Bibliotecas', exact: true }).click();
  await page.getByRole('button', { name: 'Nueva biblioteca' }).click();
  await page.getByLabel('Nombre de la biblioteca').fill('Estadísticas persistentes');
  await page.getByRole('button', { name: 'Crear biblioteca', exact: true }).click();
  await expect(page.getByRole('tab', { name: 'Procesamiento', exact: true })).toBeVisible();
  const libraries = await (await page.request.get('/api/v1/libraries')).json();
  const library = libraries.items.find((entry: { name: string }) => entry.name === 'Estadísticas persistentes');
  const actual = await page.request.get(`/api/v1/libraries/${library.id}/processing/stats`);
  expect(actual.status()).toBe(200);
  expect(await actual.json()).toMatchObject({ stats_available: true, stats: { total: 0, processed: 0 } });
  const activity = await page.request.get(`/api/v1/libraries/${library.id}/processing?view=activity`);
  expect(activity.status()).toBe(200);
  const activityTemplate = await activity.json();

  let mode: 'valid' | 'pending' | 'unavailable' | 'partial' | 'error' | 'network' = 'pending';
  let statsRequests = 0;
  let activityRequests = 0;
  const pending: Route[] = [];
  const counts = { total: 12, linked: 10, managed: 2, pending: 4, stored_only: 1, processed: 6, errors: 1, native: 4, ocr: 2 };
  let obtainedAt = new Date().toISOString();
  await page.route('**/api/v1/libraries/*/processing/stats', async route => {
    statsRequests++;
    if (mode === 'pending') { pending.push(route); return; }
    if (mode === 'network') { await route.abort('failed'); return; }
    if (mode === 'error') { await route.fulfill({ status: 503, json: { error: { code: 'UNAVAILABLE', message: 'Error temporal de estadísticas' } } }); return; }
    await route.fulfill({ json: mode === 'unavailable'
      ? { stats_available: false, stats: Object.fromEntries(Object.keys(counts).map(key => [key, 0])), stats_error: 'El cálculo superó el tiempo de espera.' }
      : { stats_available: true, stats: mode === 'partial' ? { total: 999 } : counts, stats_updated_at: obtainedAt } });
  });
  await page.route('**/api/v1/libraries/*/processing?view=activity', async route => {
    activityRequests++;
    await route.fulfill({ json: { ...activityTemplate, stats_available: false,
      scans: [{ id: 'scan-fixture', root_id: 'root-fixture', server_path: 'Carpeta de prueba', status: 'running', started_at: obtainedAt, updated_at: obtainedAt, completed_at: '', current_path: `avance-${activityRequests}.pdf`, directories: 1, pending_directories: 2, files: activityRequests, unchanged: 0, hashed: 1, excluded: 0, errors: 0, skipped_files: 0, paused: false }],
      active_jobs: [{ id: 'job-fixture', filename: 'Trabajo de prueba.pdf', job_type: 'extract', operation: 'native', total_units: 100, completed_units: activityRequests, started_at: obtainedAt, root_path: '', relative_path: '' }],
    } });
  });
  await page.clock.install();
  await page.getByRole('tab', { name: 'Procesamiento', exact: true }).click();
  const metrics = page.getByLabel('Estadísticas de procesamiento', { exact: true });
  const status = page.getByRole('status').filter({ hasText: /estadísticas|Última obtención/ });
  await expect(status).toHaveText('Cargando estadísticas…');
  await expect(metrics).toHaveCount(0);
  await expect(page.getByRole('heading', { name: 'Trabajo de prueba.pdf' })).toBeVisible();
  await expect.poll(() => pending.length).toBe(1);
  await pending.shift()!.fulfill({ json: { stats_available: false, stats_error: 'El cálculo superó el tiempo de espera.' } });
  await expect(status).toContainText('No hay estadísticas válidas todavía.');
  await expect(metrics).toHaveCount(0);
  mode = 'valid';
  await page.clock.runFor(10500);
  await expect(metrics.locator('.metric strong')).toHaveText(['12', '10', '2', '4', '1', '6', '1', '4', '2']);
  const block = await metrics.elementHandle();
  const total = await metrics.locator('strong').first().elementHandle();
  await expect(status.locator('time')).toHaveAttribute('datetime', obtainedAt);

  // Every failure retains all values and the original DOM nodes.
  for (const nextMode of ['unavailable', 'partial', 'error', 'network'] as const) {
    mode = nextMode;
    const before = statsRequests;
    await page.clock.runFor(10500);
    await expect.poll(() => statsRequests).toBeGreaterThan(before);
    await expect(status).toContainText('Datos desactualizados.');
    await expect(metrics.locator('.metric strong')).toHaveText(['12', '10', '2', '4', '1', '6', '1', '4', '2']);
    expect(await block!.evaluate(node => node.isConnected)).toBe(true);
    expect(await total!.evaluate(node => node.isConnected)).toBe(true);
    await expect(status.locator('time')).toHaveAttribute('datetime', obtainedAt);
  }

  mode = 'pending';
  const beforeActivity = activityRequests;
  await page.clock.runFor(10500);
  await expect.poll(() => pending.length).toBe(1);
  await expect(status).toContainText('Actualizando estadísticas…');
  await expect(metrics).toBeVisible();
  // Client timeout also preserves the snapshot while scans and jobs change.
  await page.clock.runFor(8500);
  await expect(status).toContainText('El servidor superó el tiempo de espera.');
  await expect.poll(() => activityRequests).toBeGreaterThan(beforeActivity);
  await expect(page.getByText(/Ruta actual o última: avance-/)).toContainText(`avance-${activityRequests}.pdf`);
  await expect(page.getByText(/% · \d+ de 100 unidades/)).toContainText(`${activityRequests} de 100 unidades`);
  await expect(metrics.locator('strong').first()).toHaveText('12');

  await page.getByRole('tab', { name: 'Carpetas', exact: true }).click();
  await page.getByRole('tab', { name: 'Procesamiento', exact: true }).click();
  await expect(metrics.locator('strong').first()).toHaveText('12');
  await expect(status.locator('time')).toHaveAttribute('datetime', obtainedAt);
  await page.clock.runFor(65000);
  await expect(status).toContainText('Datos desactualizados.');

  // A new complete response patches the values; it does not replace the block.
  const remounted = await metrics.elementHandle();
  const remountedTotal = await metrics.locator('strong').first().elementHandle();
  mode = 'valid'; counts.total = 15; counts.processed = 9; counts.native = 7;
  obtainedAt = new Date(Date.now() + 120000).toISOString();
  await page.clock.runFor(20000);
  await expect(metrics.locator('.metric strong')).toHaveText(['15', '10', '2', '4', '1', '9', '1', '7', '2']);
  await expect(status).not.toContainText('Datos desactualizados.');
  await expect(status.locator('time')).toHaveAttribute('datetime', obtainedAt);
  expect(await remounted!.evaluate(node => node.isConnected)).toBe(true);
  expect(await remountedTotal!.evaluate(node => node.isConnected)).toBe(true);

  // Another library must never inherit this library's snapshot.
  await page.getByRole('button', { name: 'Todas las bibliotecas' }).click();
  await page.getByRole('button', { name: 'Nueva biblioteca' }).click();
  await page.getByLabel('Nombre de la biblioteca').fill('Sin estadísticas anteriores');
  await page.getByRole('button', { name: 'Crear biblioteca', exact: true }).click();
  mode = 'pending';
  await page.getByRole('tab', { name: 'Procesamiento', exact: true }).click();
  await expect(status).toHaveText('Cargando estadísticas…');
  await expect(metrics).toHaveCount(0);
  await page.getByRole('button', { name: 'Todas las bibliotecas' }).click();
  await page.getByRole('button', { name: 'Abrir biblioteca Estadísticas persistentes', exact: true }).click();
  await page.getByRole('tab', { name: 'Procesamiento', exact: true }).click();
  await expect(metrics.locator('strong').first()).toHaveText('15');
});
