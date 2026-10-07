import { test, expect } from '@playwright/test';

test('acceso local y LAN real, UUID en HTTP y vuelta a local', async ({ page }) => {
  // Exercise the same fallback used by browsers on an insecure LAN origin.
  await page.addInitScript(() => Object.defineProperty(Crypto.prototype, 'randomUUID', { value: undefined, configurable: true }));
  async function login(url: string) {
    await page.goto(url);
    await page.getByLabel('Usuario', { exact: true }).fill('admin-e2e');
    await page.getByLabel('Contraseña', { exact: true }).fill('Browser-fixture-password-2026');
    await page.getByRole('button', { name: 'Entrar', exact: true }).click();
    await expect(page.getByRole('link', { name: 'Ajustes', exact: true })).toBeVisible();
  }
  await login('/login');
  await page.getByRole('link', { name: 'Ajustes', exact: true }).click(); await page.getByRole('navigation', { name: 'Secciones de configuración' }).getByRole('link', { name: 'Acceso y red', exact: true }).click();
  await expect(page.getByText('Correcto · puerto en escucha', { exact: true })).toBeVisible();
  await page.getByLabel('Modo de acceso').selectOption('lan');
  await page.getByRole('button', { name: 'Guardar acceso', exact: true }).click();
  await expect(page.getByText('Acceso en red guardado.', { exact: false })).toBeVisible();
  const settings = await (await page.request.get('/api/v1/system/network')).json();
  expect(settings.mode).toBe('lan');
  expect(settings.listen_address).toBe('0.0.0.0:8099');
  expect(settings.interfaces.length).toBeGreaterThan(0);
  const address = settings.interfaces.find((item: { suggested: boolean }) => item.suggested) || settings.interfaces[0];
  try {
    await login(`${address.url}/login`);
    await page.getByRole('link', { name: 'Bibliotecas', exact: true }).click();
    await expect(page.getByRole('heading', { name: 'Bibliotecas', exact: true })).toBeVisible();
    await page.getByRole('button', { name: 'Nueva biblioteca', exact: true }).click();
    await expect(page.getByRole('heading', { name: 'Crear biblioteca', exact: true })).toBeVisible();
    await page.getByRole('link', { name: 'Licencia', exact: true }).click();
    await expect(page.getByRole('heading', { name: 'Licencia de AIBID', exact: true })).toBeVisible();
    await page.getByText('Identificación para soporte', { exact: true }).click();
    await expect(page.locator('.document-accordion[open]').getByText('Clave pública de instalación', { exact: true })).toBeVisible();
    await page.getByRole('link', { name: 'Ajustes', exact: true }).click(); await page.getByRole('navigation', { name: 'Secciones de configuración' }).getByRole('link', { name: 'Acceso y red', exact: true }).click();
    await page.setViewportSize({ width: 390, height: 844 });
    await expect(page.getByLabel('Modo de acceso')).toHaveValue('lan');
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
    await page.screenshot({ path: 'test-results/network-lan-mobile.png', fullPage: true });
    await page.getByLabel('Modo de acceso').selectOption('local');
    await page.getByRole('button', { name: 'Guardar acceso', exact: true }).click();
    await expect(page.getByText('Acceso local guardado.', { exact: false })).toBeVisible();
  } finally {
    await login('http://127.0.0.1:8099/login');
    await page.getByRole('link', { name: 'Ajustes', exact: true }).click(); await page.getByRole('navigation', { name: 'Secciones de configuración' }).getByRole('link', { name: 'Acceso y red', exact: true }).click();
    // Leave the shared browser fixture local even if a prior assertion fails.
    await expect(page.getByLabel('Modo de acceso')).toBeVisible();
    if (await page.getByLabel('Modo de acceso').inputValue() === 'lan') {
      await page.getByLabel('Modo de acceso').selectOption('local');
      await page.getByRole('button', { name: 'Guardar acceso', exact: true }).click();
    }
    await expect(page.getByLabel('Modo de acceso')).toHaveValue('local');
  }
});
