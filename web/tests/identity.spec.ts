import { test, expect, type Page } from '@playwright/test';

async function login(page: Page, username = 'admin-e2e', password = 'Browser-fixture-password-2026') {
  await page.goto('/login');
  await page.getByLabel('Usuario', { exact: true }).fill(username);
  await page.getByLabel('Contraseña', { exact: true }).fill(password);
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Mi cuenta', exact: true })).toBeVisible();
}

test('otro navegador reemplaza la sesión y el sondeo muestra el aviso exacto', async ({ browser }) => {
  const first = await browser.newContext(); const second = await browser.newContext();
  const firstPage = await first.newPage(); const secondPage = await second.newPage();
  await login(firstPage);
  await login(secondPage);
  // No click or document mutation in the first browser: wait for its 30 s poll.
  await expect(firstPage.getByText('Tu sesión fue cerrada porque se inició sesión con tu cuenta desde otro equipo o navegador', { exact: true })).toBeVisible({ timeout: 40_000 });
  await expect(firstPage).toHaveURL(/\/login$/);
  await expect(secondPage.getByRole('heading', { name: 'Mi cuenta', exact: true })).toBeVisible();
  await first.close(); await second.close();
});

test('administrar usuarios, cambiar contraseña, ver auditoría y denegar privilegios', async ({ page, browser }) => {
  await login(page);
  await page.getByRole('link', { name: 'Usuarios', exact: true }).click();
  await page.getByRole('button', { name: 'Nuevo usuario' }).click();
  await page.getByLabel('Nombre visible', { exact: true }).fill('Usuario de prueba');
  await page.getByLabel('Nombre de usuario', { exact: true }).fill('personal-e2e');
  await page.getByLabel('Contraseña inicial').fill('abcdef');
  await page.getByRole('button', { name: 'Crear cuenta' }).click();
  await expect(page.getByText('Usuario creado. Puedes asignarle permisos desde Editar.')).toBeVisible();
  await page.getByRole('button', { name: 'Editar a personal-e2e' }).click();
  await page.getByLabel('Nombre visible', { exact: true }).fill('Nombre actualizado');
  await page.getByRole('button', { name: 'Guardar datos' }).click();
  await expect(page.getByRole('heading', { name: 'Editar a Nombre actualizado' })).toBeVisible();
  await page.getByRole('link', { name: 'Eventos', exact: true }).click();
  await expect(page.getByRole('cell', { name: 'Usuario creado', exact: true })).toBeVisible();
  const eventResponse = await page.request.get('/api/v1/audit-events?event_type=user.created');
  expect(eventResponse.ok()).toBeTruthy();

  const personalContext = await browser.newContext(); const personalPage = await personalContext.newPage();
  await login(personalPage, 'personal-e2e', 'abcdef');
  await expect(personalPage.getByRole('link', { name: 'Usuarios', exact: true })).toHaveCount(0);
  expect((await personalPage.request.get('/api/v1/users')).status()).toBe(403);
  await personalPage.getByLabel('Contraseña actual', { exact: true }).fill('abcdef');
  await personalPage.getByLabel('Nueva contraseña', { exact: true }).fill('ghijkl');
  await personalPage.getByLabel('Repetir nueva contraseña', { exact: true }).fill('ghijkl');
  await personalPage.getByRole('button', { name: 'Actualizar contraseña' }).click();
  await expect(personalPage.getByText('Contraseña actualizada. Inicia sesión con tu nueva contraseña.')).toBeVisible();
  await login(personalPage, 'personal-e2e', 'ghijkl');
  await personalContext.close();
});

test('login adaptable en móvil y error de acceso sin enumeración', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/login');
  await page.getByLabel('Usuario', { exact: true }).fill('unknown-e2e');
  await page.getByLabel('Contraseña', { exact: true }).fill('Wrong-fixture-password');
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await expect(page.getByRole('alert')).toHaveText('Usuario o contraseña incorrectos.');
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
  await page.screenshot({ path: 'test-results/login-mobile.png', fullPage: true });
});
