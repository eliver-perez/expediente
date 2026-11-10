import { test, expect, type Page } from '@playwright/test';

async function login(page: Page) {
  await page.goto('/login');
  await page.getByLabel('Usuario', { exact: true }).fill('admin-e2e');
  await page.getByLabel('Contraseña', { exact: true }).fill('Browser-fixture-password-2026');
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Mi cuenta', exact: true })).toBeVisible();
}
const attempt = (id: string, outcome = 'success') => ({ id, outcome, attempted_identifier: id,
  occurred_at: '2026-10-01T12:00:00Z', observed_ip: '127.0.0.1', user_agent: 'QA' });

test('QA: una página tardía no mezcla resultados después de cambiar filtros', async ({ page }) => {
  await login(page);
  let release!: () => void;
  const held = new Promise<void>(resolve => { release = resolve; });
  let requested!: () => void;
  const pending = new Promise<void>(resolve => { requested = resolve; });
  await page.route('**/api/v1/authentication-attempts?*', async route => {
    const query = new URL(route.request().url()).searchParams;
    if (query.has('cursor')) {
      requested(); await held;
      await route.fulfill({ json: { items: [attempt('PAGINA-ANTERIOR')], next_cursor: null } }).catch(() => {});
    } else {
      await route.fulfill({ json: query.has('outcome')
        ? { items: [attempt('SOLO-FALLIDOS', 'failed')], next_cursor: null }
        : { items: [attempt('TODOS-INICIAL')], next_cursor: 'previous-scope' } });
    }
  });
  try {
    await page.goto('/admin/access');
    await page.getByRole('tab', { name: 'Intentos de acceso' }).click();
    await expect(page.getByRole('cell', { name: 'TODOS-INICIAL', exact: true })).toBeVisible();
    await page.getByRole('button', { name: 'Mostrar más' }).click(); await pending;
    await page.getByRole('combobox', { name: 'Resultado', exact: true }).selectOption('failed');
    await page.getByRole('button', { name: 'Filtrar', exact: true }).click();
    await expect(page.getByRole('cell', { name: 'SOLO-FALLIDOS', exact: true })).toBeVisible();
    release();
    // Let a late network response settle before checking the filtered table.
    await page.waitForTimeout(200);
    await expect(page.getByRole('cell', { name: 'PAGINA-ANTERIOR', exact: true })).toHaveCount(0);
    await expect(page.getByRole('cell', { name: 'TODOS-INICIAL', exact: true })).toHaveCount(0);
    await expect(page.getByRole('cell', { name: 'SOLO-FALLIDOS', exact: true })).toHaveCount(1);
  } finally { release(); }
});

test('QA: una solicitud sin respuesta deja de cargar y permite recuperar el listado', async ({ page }) => {
  await login(page); await page.clock.install();
  let release!: () => void;
  const held = new Promise<void>(resolve => { release = resolve; });
  let requested!: () => void;
  const pending = new Promise<void>(resolve => { requested = resolve; });
  await page.route('**/api/v1/authentication-attempts?*', async route => {
    requested(); await held;
    await route.fulfill({ json: { items: [], next_cursor: null } }).catch(() => {});
  });
  try {
    await page.goto('/admin/access'); await page.getByRole('tab', { name: 'Intentos de acceso' }).click(); await pending;
    await page.clock.fastForward(46_000);
    await expect(page.getByRole('alert')).toContainText('tardó demasiado', { timeout: 2000 });
    await expect(page.getByText('Cargando…', { exact: true })).toHaveCount(0);
    release(); await page.unroute('**/api/v1/authentication-attempts?*');
    await page.getByRole('combobox', { name: 'Resultado', exact: true }).selectOption('rate_limited');
    await page.getByRole('button', { name: 'Filtrar', exact: true }).click();
    await expect(page.getByText('No hay registros para mostrar.', { exact: true })).toBeVisible();
    await expect(page.getByRole('alert')).toHaveCount(0);
  } finally { release(); }
});

test('QA: una respuesta de error ajena al contrato muestra un mensaje útil', async ({ page }) => {
  await login(page);
  await page.route('**/api/v1/authentication-attempts?*', route => route.fulfill({ status: 502, json: { error: null } }));
  await page.goto('/admin/access'); await page.getByRole('tab', { name: 'Intentos de acceso' }).click();
  await expect(page.getByRole('alert')).toHaveText('El servidor no pudo completar la operación.');
});
