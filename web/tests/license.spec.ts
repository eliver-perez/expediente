import { test, expect } from '@playwright/test';

test('licencias: simulador HTTPS, archivos sin conexión y recuperación de solo lectura', async ({ page }) => {
  await page.goto('/login'); await page.getByLabel('Usuario', { exact: true }).fill('admin-e2e');
  await page.getByLabel('Contraseña', { exact: true }).fill('Browser-fixture-password-2026');
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await page.getByRole('link', { name: 'Licencia', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Licencia de AIBID' })).toBeVisible();
  const online = page.getByRole('heading', { name: 'Activación y renovación en línea' }).locator('..');
  async function activate(key: string) {
    await page.getByLabel('Operación', { exact: true }).selectOption('activate');
    await page.getByLabel('Clave comercial', { exact: true }).fill(key);
    await page.getByRole('button', { name: 'Activación en línea', exact: true }).click();
  }
  async function deactivate() {
    await page.getByLabel('Operación', { exact: true }).selectOption('deactivate');
    await page.getByLabel('Confirmo la desactivación.', { exact: false }).check();
    await page.getByRole('button', { name: 'Desactivación en línea', exact: true }).click();
    await expect(page.getByText('El servidor confirmó la desactivación. Tus documentos se conservan.')).toBeVisible();
  }
  await activate('DEMO-PERPETUAL');
  await expect(page.getByRole('alert')).toContainText('No fue posible contactar de forma segura');
  const pending = await (await page.request.get('/api/v1/license')).json();
  expect(pending.pending_operations).toHaveLength(1);
  const requestID = pending.pending_operations[0].request_id;
  await page.reload();
  await expect(page.getByText('Hay una solicitud pendiente.', { exact: false })).toBeVisible();
  await expect(page.getByLabel('Retomar solicitud pendiente', { exact: true })).toHaveValue(requestID);
  await activate('DEMO-PERPETUAL'); await expect(page.getByTestId('license-state')).toHaveText('Activa');
  expect((await (await page.request.get('/api/v1/license')).json()).pending_operations).toHaveLength(0);
  await expect(page.getByLabel('Clave comercial', { exact: true })).toHaveValue('');
  await expect(page.getByText('Perpetua · uso sin vencimiento', { exact: true })).toBeVisible();
  await page.getByLabel('Operación', { exact: true }).selectOption('refresh');
  await page.getByRole('button', { name: 'Renovación en línea', exact: true }).click();
  await expect(page.getByText('La licencia se verificó y quedó guardada.')).toBeVisible();
  const download = page.waitForEvent('download'); await page.getByRole('button', { name: 'Solicitar renovación', exact: true }).click();
  expect((await download).suggestedFilename()).toMatch(/\.licreq$/);
  await expect(page.getByText('Solicitud descargada. Entrégala al proveedor para recibir el archivo .lic.')).toBeVisible();
  await page.getByLabel('Archivo de licencia (.lic)', { exact: true }).setInputFiles({ name: 'incorrecta.lic', mimeType: 'application/octet-stream', buffer: Buffer.from('wrong.signature.value') });
  await page.getByRole('button', { name: 'Importar licencia', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('contrato de licencia');
  await expect(page.getByTestId('license-state')).toHaveText('Activa');
  const archived = await (await page.request.get('/api/v1/license/artifacts')).json();
  const response = archived.items.find((item: { direction: string; action: string }) => item.direction === 'response' && item.action === 'refresh');
  const license = await (await page.request.get(`/api/v1/license/artifacts/${response.id}`)).body();
  await page.getByLabel('Archivo de licencia (.lic)', { exact: true }).setInputFiles({ name: 'renovacion.lic', mimeType: 'application/octet-stream', buffer: license });
  await page.getByRole('button', { name: 'Importar licencia', exact: true }).click();
  await expect(page.getByText('Licencia importada y verificada.')).toBeVisible();
  await deactivate();
  await activate('DEMO-EXPIRED'); await expect(page.getByTestId('license-state')).toHaveText('Vencida · solo lectura');
  const session = await (await page.request.get('/api/v1/auth/session')).json();
  const blocked = await page.request.post('/api/v1/libraries', { headers: { Origin: 'http://127.0.0.1:8099', 'X-CSRF-Token': session.csrf_token, 'Idempotency-Key': 'license-browser-read-only' }, data: { name: 'No debe crearse', mode: 'linked', manager_user_id: session.user.id } });
  expect(blocked.status()).toBe(403); expect((await blocked.json()).error.code).toBe('LICENSE_READ_ONLY');
  await expect(online).toBeVisible();
  await page.screenshot({ path: 'test-results/h6-license-read-only.png', fullPage: true });
  await deactivate(); await activate('DEMO-PERPETUAL'); await expect(page.getByTestId('license-state')).toHaveText('Activa');
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
  await page.screenshot({ path: 'test-results/h6-license-mobile.png', fullPage: true });
});
