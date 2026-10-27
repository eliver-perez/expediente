import { test, expect, type Page } from '@playwright/test';

async function login(page: Page) {
  await page.goto('/login');
  await page.getByLabel('Usuario', { exact: true }).fill('admin-e2e');
  await page.getByLabel('Contraseña', { exact: true }).fill('Browser-fixture-password-2026');
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Mi cuenta', exact: true })).toBeVisible();
}

test('El CLUF pendiente persiste ante rutas directas y permite activar o rechazar', async ({ page }) => {
  await page.route('**/api/v1/auth/session', async route => {
    const response = await route.fetch();
    const session = await response.json();
    await route.fulfill({ response, json: { ...session, eula_accepted: false } });
  });
  await page.route('**/api/v1/eula', async route => {
    const response = await route.fetch();
    const contract = await response.json();
    await route.fulfill({ response, json: { ...contract, accepted: false } });
  });
  await page.goto('/login');
  await page.getByLabel('Usuario', { exact: true }).fill('admin-e2e');
  await page.getByLabel('Contraseña', { exact: true }).fill('Browser-fixture-password-2026');
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Contrato de licencia de AIDEX' })).toBeVisible();
  await expect(page.locator('.eula-text')).not.toContainText('## 18.');
  await page.goto('/libraries');
  await expect(page.getByRole('heading', { name: 'Contrato de licencia de AIDEX' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Bibliotecas', exact: true })).toHaveCount(0);
  await page.getByRole('link', { name: 'Administrar licencia' }).click();
  await expect(page.getByRole('heading', { name: 'Licencia de AIDEX' })).toBeVisible();
  await page.getByRole('link', { name: 'Leer contrato' }).click();
  await expect(page.getByRole('button', { name: 'Rechazar y cerrar sesión' })).toBeVisible();
  await page.getByRole('button', { name: 'Rechazar y cerrar sesión' }).click();
  await expect(page.getByRole('heading', { name: 'Iniciar sesión' })).toBeVisible();
});

test('Acerca de AIDEX muestra el CLUF completo sin aceptarlo al consultarlo', async ({ page }) => {
  await login(page);
  await page.getByRole('link', { name: 'Acerca de AIDEX' }).click();
  await expect(page.getByRole('heading', { name: 'Acerca de AIDEX' })).toBeVisible();
  await expect(page.getByText('Versión instalada')).toBeVisible();
  await expect(page.getByText('Referencia para soporte')).toBeVisible();
  const before = await (await page.request.get('/api/v1/eula')).json();
  await page.getByRole('button', { name: 'Ver contrato de licencia' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  await expect(dialog.getByRole('heading', { name: '18. Integridad, comunicaciones y consulta' })).toBeVisible();
  await expect(dialog).not.toContainText('## 18.');
  await expect(dialog).toContainText('Borrador pendiente de revisión jurídica');
  await page.keyboard.press('Escape');
  await expect(dialog).not.toBeVisible();
  const after = await (await page.request.get('/api/v1/eula')).json();
  expect(after.accepted).toBe(before.accepted);
  expect(after.accepted_at).toBe(before.accepted_at);
});

test('La navegación y el contrato siguen utilizables de 360 px a escritorio', async ({ page }) => {
  test.setTimeout(120000);
  await page.setViewportSize({ width: 360, height: 800 });
  await page.goto('/login');
  await expect(page.getByText('Organiza, consulta y da seguimiento a tus expedientes.')).toBeVisible();
  const loginBox = await page.getByRole('heading', { name: 'Iniciar sesión' }).boundingBox();
  expect(loginBox?.y || 999).toBeLessThan(400);
  await login(page);
  for (const width of [360, 390, 440, 768, 1024, 1440]) {
    await page.setViewportSize({ width, height: 800 });
    if (width <= 440) {
      const menu = page.getByRole('button', { name: 'Menú' });
      await expect(menu).toBeVisible();
      await menu.click();
      await expect(page.getByRole('link', { name: 'Bibliotecas', exact: true })).toBeVisible();
      await page.getByRole('link', { name: 'Acerca de AIDEX' }).click();
      await expect(menu).toHaveAttribute('aria-expanded', 'false');
    } else {
      await page.getByRole('link', { name: 'Acerca de AIDEX' }).click();
    }
    await expect(page.getByRole('heading', { name: 'Acerca de AIDEX' })).toBeVisible();
    await page.getByRole('button', { name: 'Ver contrato de licencia' }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog.getByRole('button', { name: 'Cerrar' }).first()).toBeInViewport();
    await dialog.getByRole('button', { name: 'Cerrar' }).first().click();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `overflow at ${width}px`).toBeTruthy();
  }
  await page.getByRole('link', { name: 'Bibliotecas', exact: true }).click();
  await page.getByRole('button', { name: 'Nueva biblioteca' }).click();
  await page.getByLabel('Nombre de la biblioteca').fill('Responsive ' + Date.now());
  await page.getByLabel('Modalidad de biblioteca').selectOption('hybrid');
  await expect(page.getByLabel('Asignarme como gestor de esta biblioteca')).toHaveCount(0);
  await page.getByRole('button', { name: 'Crear biblioteca', exact: true }).click();
  await expect(page.getByRole('tab', { name: 'Procesamiento', exact: true })).toBeVisible();
  for (const width of [360, 390, 440, 768, 1024, 1440]) {
    await page.setViewportSize({ width, height: 800 });
    for (const tab of ['Estadísticas', 'Expedientes', 'Procesamiento']) {
      await page.getByRole('tab', { name: tab, exact: true }).click();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${tab} overflow at ${width}px`).toBeTruthy();
    }
  }
});
