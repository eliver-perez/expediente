import { test, expect, type Page } from '@playwright/test';
import { mkdtemp, copyFile, mkdir, utimes, rename, rm, realpath } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';

async function login(page: Page) {
  await page.goto('/login'); await page.getByLabel('Usuario', { exact: true }).fill('admin-e2e');
  await page.getByLabel('Contraseña', { exact: true }).fill('Browser-fixture-password-2026');
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Mi cuenta', exact: true })).toBeVisible();
}
async function mutate(page: Page, path: string, body: unknown) {
  const session = await (await page.request.get('/api/v1/auth/session')).json();
  return page.request.post(`/api/v1${path}`, { headers: { Origin: 'http://127.0.0.1:8099', 'X-CSRF-Token': session.csrf_token }, data: body });
}

test('biblioteca vinculada: raíz, OCR, búsqueda, texto retenido y vista lógica', async ({ page }) => {
  const directory = await realpath(await mkdtemp(join(tmpdir(), 'documental-h3-browser-')));
  const nested = join(directory, 'Planos'); await mkdir(nested);
  const native = join(nested, 'Plano-PR-008.pdf'); const scanned = join(nested, 'Escaneado.pdf');
  await copyFile(resolve('../testdata/documents/native.pdf'), native); await copyFile(resolve('../testdata/documents/scanned.pdf'), scanned);
  const past = new Date(Date.now() - 5000); await utimes(native, past, past); await utimes(scanned, past, past);
  try {
    await login(page); await page.getByRole('link', { name: 'Bibliotecas', exact: true }).click();
    await page.getByRole('button', { name: 'Nueva biblioteca' }).click(); await page.getByLabel('Nombre de la biblioteca').fill('Archivo de planos E2E');
    await page.getByLabel('Asignarme como gestor de esta biblioteca').check(); await page.getByRole('button', { name: 'Crear biblioteca', exact: true }).click();
    await expect(page.getByRole('heading', { name: 'Archivo de planos E2E', exact: true })).toBeVisible();
    await page.getByRole('tab', { name: 'Carpetas', exact: true }).click();
    await page.getByLabel('Ruta de la carpeta en el servidor').fill(directory); await page.getByRole('button', { name: 'Inspeccionar carpeta' }).click();
    await expect(page.getByText('Carpeta disponible para registrar e indexar.')).toBeVisible(); await page.getByRole('button', { name: 'Registrar e indexar' }).click();
    await expect(page.getByText('Raíz registrada. El escaneo está en cola.')).toBeVisible();
    const libraries = await (await page.request.get('/api/v1/libraries')).json(); const library = libraries.items.find((item: { name: string }) => item.name === 'Archivo de planos E2E');
    await expect.poll(async () => { const response = await mutate(page, '/search', { query: 'ESCANEADO', search_type: 'ocr', library_ids: [library.id] }); return (await response.json()).result_count; }, { timeout: 30_000 }).toBe(1);
    await page.getByLabel('Ruta de la carpeta en el servidor').fill(nested); await page.getByRole('button', { name: 'Inspeccionar carpeta' }).click();
    await expect(page.getByText('Esta subcarpeta ya pertenece a una raíz. Puedes crear una vista para consultarla por separado.')).toBeVisible();
    await page.getByLabel('Nombre de la vista').fill('Planos de consulta'); await page.getByRole('button', { name: 'Crear vista lógica' }).click(); await expect(page.getByText('Vista creada.')).toBeVisible();
    await page.getByRole('tab', { name: 'Documentos', exact: true }).click(); await expect(page.getByRole('button', { name: 'Planos', exact: true })).toBeVisible();
    await page.getByRole('button', { name: 'Planos', exact: true }).click(); await expect(page.getByRole('button', { name: 'Plano-PR-008.pdf', exact: true })).toBeVisible();
    await page.screenshot({ path: 'test-results/h3-explorer.png', fullPage: true });
    await page.getByRole('link', { name: 'Buscar documentos', exact: true }).click(); await page.getByLabel('Texto a buscar').fill('  "estructura metálica"  ');
    await page.getByRole('button', { name: 'Buscar documentos', exact: true }).click(); await expect(page.getByRole('heading', { name: '1 documento encontrado' })).toBeVisible();
    await page.getByRole('button', { name: 'Ver ficha', exact: true }).click(); await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByRole('link', { name: 'Descargar PDF' })).toBeVisible(); await expect(page.locator('.retained-text pre')).toContainText('estructura metálica');
    // Wait for Chrome's native PDF viewer, not just the surrounding dialog.
    await expect.poll(async () => {
      const viewer = page.frames().find(frame => frame.url().startsWith('chrome-extension://'));
      if (!viewer) return 0;
      return viewer.locator('pdf-viewer').evaluate(element => (element as HTMLElement & { loadProgress_: number }).loadProgress_).catch(() => 0);
    }).toBe(100);
    await page.screenshot({ path: 'test-results/h3-document.png', fullPage: true });
    await page.getByRole('button', { name: 'Cerrar ficha' }).click();
    await rename(native, native + '.removed');
    const roots = await (await page.request.get(`/api/v1/libraries/${library.id}/roots`)).json();
    await mutate(page, `/libraries/${library.id}/verify`, { root_id: roots.items[0].id });
    await expect.poll(async () => { const response = await mutate(page, '/search', { query: 'PR-008', library_ids: [library.id], filters: { availability: 'missing' } }); return (await response.json()).result_count; }, { timeout: 20_000 }).toBe(1);
    await page.getByRole('button', { name: 'Buscar documentos', exact: true }).click(); await page.getByRole('button', { name: 'Ver ficha', exact: true }).click();
    await expect(page.getByRole('heading', { name: 'Original no disponible' })).toBeVisible(); await expect(page.getByRole('link', { name: 'Descargar PDF' })).toHaveCount(0);
    await expect(page.locator('.retained-text pre')).toContainText('estructura metálica'); await page.getByRole('button', { name: 'Cerrar ficha' }).click();
    await page.setViewportSize({ width: 390, height: 844 }); expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
    await page.screenshot({ path: 'test-results/h3-search-mobile.png', fullPage: true });
  } finally { await rm(directory, { recursive: true, force: true }); }
});
