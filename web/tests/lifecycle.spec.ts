import { test, expect, type Page } from '@playwright/test';
import { mkdtemp, mkdir, writeFile, realpath, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

async function login(page: Page) { await page.goto('/login'); await page.getByLabel('Usuario', { exact: true }).fill('admin-e2e'); await page.getByLabel('Contraseña', { exact: true }).fill('Browser-fixture-password-2026'); await page.getByRole('button', { name: 'Entrar', exact: true }).click(); await expect(page.getByRole('heading', { name: 'Mi cuenta', exact: true })).toBeVisible(); }
async function create(page: Page, name: string) {
  await page.getByRole('link', { name: 'Bibliotecas', exact: true }).click(); await page.getByRole('button', { name: 'Nueva biblioteca' }).click(); await page.getByLabel('Nombre de la biblioteca').fill(name);  await page.getByRole('button', { name: 'Crear biblioteca', exact: true }).click(); await expect(page.getByRole('heading', { name, exact: true })).toBeVisible();
  return (await (await page.request.get('/api/v1/libraries')).json()).items.find((item: { name: string }) => item.name === name);
}

test('Fase 1: archivo, restauración, confirmación exacta, manifiesto y Toast fijo', async ({ page }) => {
  const errors: string[] = []; page.on('pageerror', e => errors.push(e.message));
  await login(page); const name = 'Biblioteca lifecycle 1'; const library = await create(page, name);
  await page.getByRole('tab', { name: 'Configuración', exact: true }).click();
  await page.getByRole('button', { name: 'Archivar biblioteca', exact: true }).click();
  const dialog = page.getByRole('dialog'); await expect(dialog).toContainText('Sus documentos permanecerán intactos');
  await dialog.getByRole('button', { name: 'Cancelar', exact: true }).focus(); await page.keyboard.press('Enter'); await expect(dialog).toHaveCount(0);
  await page.getByRole('button', { name: 'Archivar biblioteca', exact: true }).click(); await dialog.getByRole('button', { name: 'Archivar biblioteca', exact: true }).click();
  await expect(page.getByRole('status').filter({ hasText: 'La biblioteca se archivó correctamente.' })).toBeVisible();
  expect((await (await page.request.get('/api/v1/libraries')).json()).items.some((item: { id: string }) => item.id === library.id)).toBe(false);
  await page.getByRole('tab', { name: 'Bibliotecas archivadas' }).click(); const card = page.locator('article.library-card').filter({ has: page.getByRole('heading', { name, exact: true }) }); await expect(card).toBeVisible();
  await card.getByLabel(`Opciones de ${name}`).click(); await card.getByRole('button', { name: 'Restaurar biblioteca' }).click(); await dialog.getByRole('button', { name: 'Restaurar biblioteca' }).click();
  await expect(page.getByRole('status').filter({ hasText: 'La biblioteca se restauró correctamente.' })).toBeVisible();
  await page.getByRole('tab', { name: 'Bibliotecas activas' }).click(); await card.getByLabel(`Opciones de ${name}`).click(); await card.getByRole('button', { name: 'Eliminar biblioteca', exact: true }).click();
  await expect(dialog).toContainText('Tus documentos definitivos NO se borrarán'); await expect(dialog.getByRole('button', { name: 'Eliminar biblioteca definitivamente', exact: true })).toBeDisabled();
  await dialog.getByLabel('Escribe exactamente el nombre de la biblioteca').fill(name.toLowerCase()); await expect(dialog.getByRole('button', { name: 'Eliminar biblioteca definitivamente', exact: true })).toBeDisabled();
  await dialog.getByLabel('Escribe exactamente el nombre de la biblioteca').fill(name); await expect(dialog.getByRole('button', { name: 'Eliminar biblioteca definitivamente', exact: true })).toBeEnabled();
  const manifestURL = await dialog.getByRole('link', { name: 'Descargar manifiesto de documentos' }).getAttribute('href'); expect((await page.request.get(manifestURL!)).ok()).toBeTruthy();
  let confirmations = 0; page.on('request', r => { if (r.method() === 'POST' && r.url().endsWith('/confirm') && r.url().includes('/deletions/')) confirmations++; });
  await dialog.getByRole('button', { name: 'Eliminar biblioteca definitivamente', exact: true }).dblclick();
  await expect(dialog).toHaveCount(0); expect(confirmations).toBe(1);
  await expect(card).toHaveCount(0); await expect(page.getByRole('status').filter({ hasText: 'Eliminación pendiente.' })).toBeVisible();
  await page.getByText('Eliminaciones y limpieza', { exact: true }).click(); await expect(page.locator('.deletion-history')).toContainText(name);
  await expect.poll(async () => (await (await page.request.get('/api/v1/deletions')).json()).items.find((item: { entity_id: string }) => item.entity_id === library.id)?.state, { timeout: 30000 }).toMatch(/complete|cleanup_pending/);
  await page.evaluate(() => { document.body.style.minHeight = '3000px'; window.scrollTo(0, 2500); });
  await expect(page.locator('.toast-host')).toHaveCSS('position', 'fixed'); const box = await page.getByRole('status').filter({ hasText: 'Eliminación pendiente.' }).boundingBox(); expect(box!.y).toBeLessThan(400);
  await page.setViewportSize({ width: 390, height: 844 }); expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBeTruthy();
  await page.screenshot({ path: 'test-results/phase1-lifecycle-mobile.png', fullPage: false });
  expect(errors).toEqual([]);
});

test('Fase 1: carpetas, omisiones paginadas, filtros, errores, retiro y reanudación', async ({ page }) => {
  test.setTimeout(120000);
  const dir = await realpath(await mkdtemp(join(tmpdir(), 'aidex-lifecycle-'))); const root = join(dir, 'originales'); await mkdir(root);
  for (let n = 0; n < 115; n++) await writeFile(join(root, `documento_${String(n).padStart(3, '0')}_%.exe`), 'synthetic blocked data');
  try {
    await login(page); const name = 'Raíces lifecycle'; const library = await create(page, name);
    await page.getByRole('tab', { name: 'Carpetas', exact: true }).click(); await page.getByLabel('Ruta de la carpeta en el servidor').fill(join(dir, 'missing')); await page.getByRole('button', { name: 'Inspeccionar carpeta' }).click(); await expect(page.getByRole('alert')).toContainText('No se encontró');
    await page.getByLabel('Ruta de la carpeta en el servidor').fill(root); await page.getByRole('button', { name: 'Inspeccionar carpeta' }).click(); await page.getByRole('button', { name: 'Registrar e indexar' }).click();
    const rootID = (await (await page.request.get(`/api/v1/libraries/${library.id}/roots`)).json()).items.find((item: { server_path: string }) => item.server_path === root)?.id;
    expect(rootID).toBeTruthy();
    await expect.poll(async () => (await (await page.request.get(`/api/v1/libraries/${library.id}/file-skips?page=1&page_size=10&sort=path&direction=asc`)).json()).total, { timeout: 30000 }).toBe(115);
    await page.getByRole('tab', { name: 'Procesamiento', exact: true }).click(); const skips = page.locator('section.file-skips');
    await expect(skips.getByRole('table').getByRole('row')).toHaveCount(11); await expect(skips.getByLabel('Registros por página')).toHaveValue('10');
    await skips.getByRole('button', { name: 'Siguiente', exact: true }).click(); await expect(skips).toContainText('Página 2 de 12');
    await skips.getByLabel('Buscar archivo o ruta').fill('documento_114_%'); await skips.getByRole('button', { name: 'Buscar', exact: true }).click(); await expect(skips.getByRole('table').getByRole('row')).toHaveCount(2);
    await skips.getByRole('combobox', { name: 'Motivo de omisión' }).selectOption('FILE_FORMAT_DISABLED'); await expect(skips).toContainText('No hay omisiones que coincidan');
    await skips.getByRole('combobox', { name: 'Motivo de omisión' }).selectOption(''); await skips.getByRole('combobox', { name: 'Dirección', exact: true }).selectOption('asc'); await skips.locator('.compact-path summary').click(); await expect(skips.locator('.compact-path p')).toContainText('documento_114_%.exe');
    const history = page.locator('.job-history'); await history.getByLabel('Estado de tarea').selectOption('failed'); await expect(history).toContainText('No hay tareas que coincidan'); await history.getByLabel('Estado de tarea').selectOption(''); await history.getByRole('combobox', { name: 'Tipo de tarea', exact: true }).selectOption('scan'); await expect.poll(() => history.getByRole('table').getByRole('row').count()).toBeGreaterThan(1);
    await page.getByRole('tab', { name: 'Carpetas', exact: true }).click(); await page.getByText('Configurar carpeta', { exact: true }).click(); await page.getByRole('button', { name: 'Dejar de supervisar esta carpeta…' }).click(); await page.getByLabel('Confirmo que se detendrá').check(); await page.getByRole('button', { name: 'Confirmar retiro de carpeta', exact: true }).click(); await page.getByRole('dialog').getByRole('button', { name: 'Confirmar retiro de carpeta', exact: true }).click();
    await expect(page.getByRole('button', { name: 'Reanudar supervisión', exact: true })).toBeVisible(); await page.getByRole('button', { name: 'Reanudar supervisión', exact: true }).click(); await page.getByRole('dialog').getByRole('button', { name: 'Reanudar supervisión', exact: true }).click(); await expect(page.getByRole('status').filter({ hasText: 'La supervisión se reanudó correctamente.' })).toBeVisible();
    // Wait for the finite scan before destructive metadata operations.
    await expect.poll(async () => (await (await page.request.get(`/api/v1/libraries/${library.id}/jobs`)).json()).items.filter((job: { status: string }) => ['queued','running','retry_wait'].includes(job.status)).length).toBe(0);
    await page.getByRole('button', { name: 'Eliminar carpeta de AIDEX', exact: true }).click(); await page.getByRole('dialog').getByRole('button', { name: 'Eliminar registro de carpeta', exact: true }).click(); await expect(page.getByRole('status').filter({ hasText: 'Eliminación pendiente.' })).toBeVisible();
    await expect.poll(async () => (await (await page.request.get('/api/v1/deletions')).json()).items.find((item: { entity_id: string }) => item.entity_id === rootID)?.state, { timeout: 30000 }).toMatch(/complete|cleanup_pending/);
    expect(await readFile(join(root, 'documento_000_%.exe'), 'utf8')).toBe('synthetic blocked data');
    await page.reload(); await page.getByRole('button', { name: `Abrir biblioteca ${name}`, exact: true }).click(); await page.getByRole('tab', { name: 'Carpetas', exact: true }).click(); await expect(page.getByText('Añade la primera carpeta para comenzar a indexar.')).toBeVisible();
  } finally { await rm(dir, { recursive: true, force: true }); }
});
