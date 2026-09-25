import { test, expect, type Page } from '@playwright/test';
import { mkdtemp, realpath, rm, appendFile, readFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';

async function login(page: Page, username = 'admin-e2e', password = 'Browser-fixture-password-2026') {
  await page.goto('/login'); await page.getByLabel('Usuario', { exact: true }).fill(username); await page.getByLabel('Contraseña', { exact: true }).fill(password); await page.getByRole('button', { name: 'Entrar', exact: true }).click(); await expect(page.getByRole('heading', { name: 'Mi cuenta', exact: true })).toBeVisible();
}
async function request(page: Page, method: string, path: string, data: unknown) {
  const session = await (await page.request.get('/api/v1/auth/session')).json();
  const response = await page.request.fetch(`/api/v1${path}`, { method, headers: { Origin: 'http://127.0.0.1:8099', 'X-CSRF-Token': session.csrf_token }, data });
  expect(response.ok(), await response.text()).toBeTruthy(); return response.status() === 204 ? null : response.json();
}

test('H5: confirmar, detectar modificación administrada y revisar con otra cuenta', async ({ page, browser }) => {
  test.setTimeout(90_000);
  const destination = await realpath(await mkdtemp(join(tmpdir(), 'aibid-h5-browser-')));
  const reviewerPage = await browser.newPage();
  try {
    await login(page); await page.getByRole('link', { name: 'Bibliotecas', exact: true }).click(); await page.getByRole('button', { name: 'Nueva biblioteca' }).click();
    await page.getByLabel('Nombre de la biblioteca').fill('Revisión H5'); await page.getByLabel('Modalidad de biblioteca').selectOption('hybrid'); await page.getByLabel('Asignarme como gestor de esta biblioteca').check(); await page.getByRole('button', { name: 'Crear biblioteca', exact: true }).click();
    await page.getByRole('tab', { name: 'Carpetas', exact: true }).click(); await page.getByLabel('Origen de la carpeta').selectOption('managed'); await page.getByLabel('Ruta de la carpeta en el servidor').fill(destination); await page.getByRole('button', { name: 'Inspeccionar carpeta' }).click(); await page.getByRole('button', { name: 'Registrar destino administrado' }).click(); await expect(page.getByText('Destino administrado registrado. Selecciónalo en Configuración.')).toBeVisible();
    await page.getByRole('tab', { name: 'Configuración', exact: true }).click(); await page.getByRole('combobox', { name: 'Carpeta administrada', exact: true }).selectOption({ label: destination }); await page.getByRole('button', { name: 'Guardar configuración', exact: true }).click(); await expect(page.getByText('Configuración guardada.')).toBeVisible();
    const all = await (await page.request.get('/api/v1/libraries')).json(); const library = all.items.find((item: { name: string }) => item.name === 'Revisión H5');
    // Catalog creation is covered by H4; these fixtures isolate the H5 workflow.
    const category = await request(page, 'POST', `/libraries/${library.id}/categories`, { name: 'Planos' });
    const kind = await request(page, 'POST', `/libraries/${library.id}/document-types`, { name: 'Plano', category_id: category.id, allows_multiple: true, requires_descriptive_title: false });
    await request(page, 'POST', `/libraries/${library.id}/cases`, { identifier: 'PR-H5' });
    const reviewer = await request(page, 'POST', '/users', { username: 'reviewer-browser-h5', display_name: 'Revisora H5', password: 'abcdef' });
    await request(page, 'PUT', `/libraries/${library.id}/members/${reviewer.id}`, { role_ids: ['library_reviewer'] });
    await page.getByRole('tab', { name: 'Cargas', exact: true }).click(); await page.getByLabel('Archivos PDF', { exact: true }).setInputFiles(resolve('../testdata/documents/native.pdf')); await page.getByRole('button', { name: 'Guardar borradores' }).click(); await page.getByRole('button', { name: 'Clasificar native.pdf', exact: true }).click();
    const dialog = page.getByRole('dialog'); await dialog.getByText('Clasificar o asociar a expediente', { exact: true }).click(); await dialog.getByLabel('Expediente de destino').selectOption({ label: 'PR-H5' }); await dialog.getByLabel('Categoría del documento').selectOption(category.id); await dialog.getByLabel('Tipo del documento').selectOption(kind.id); await dialog.getByRole('button', { name: 'Guardar clasificación' }).click(); await expect(dialog).toContainText('Expediente PR-H5');
    await dialog.getByRole('button', { name: 'Confirmar documento', exact: true }).click(); await expect(dialog.getByText('Guardado definitivo completado', { exact: true })).toBeVisible({ timeout: 25_000 }); await expect(dialog).toContainText('Aprobado'); await expect(dialog).toContainText('Ruta definitiva:'); await page.screenshot({ path: 'test-results/h5-finalized.png', fullPage: true });
    const documents = await (await page.request.get(`/api/v1/libraries/${library.id}/explorer`)).json(); const document = documents.items[0];
    expect(await readFile(document.original_path)).toEqual(await readFile(resolve('../testdata/documents/native.pdf')));
    await dialog.getByRole('button', { name: 'Cerrar ficha' }).click();
    await appendFile(document.original_path, '\n% external modification from browser fixture\n');
    await page.getByRole('button', { name: 'Verificar biblioteca ahora', exact: true }).click();
    await expect.poll(async () => (await (await page.request.get(`/api/v1/documents/${document.id}`)).json()).approval_status).toBe('needs_review');
    await page.getByRole('tab', { name: 'Configuración', exact: true }).click(); await page.getByLabel('Exigir revisión para documentos administrados').check(); await page.getByRole('button', { name: 'Guardar configuración', exact: true }).click(); await expect(page.getByText('Configuración guardada.')).toBeVisible();
    await page.getByRole('tab', { name: 'Documentos', exact: true }).click(); await page.getByRole('button', { name: 'Ver ficha', exact: true }).click(); await expect(dialog).toContainText('El archivo administrado cambió fuera de AIBID.'); await dialog.getByRole('combobox', { name: 'Revisor', exact: true }).selectOption({ label: 'Revisora H5' }); await dialog.getByRole('button', { name: 'Enviar a revisión', exact: true }).click(); await expect(dialog).toContainText('Pendiente de decisión'); await dialog.getByRole('button', { name: 'Cerrar ficha' }).click();
    await login(reviewerPage, 'reviewer-browser-h5', 'abcdef'); await reviewerPage.getByRole('link', { name: 'Bibliotecas', exact: true }).click(); await reviewerPage.getByRole('button', { name: /Revisión H5/ }).click(); await reviewerPage.getByRole('tab', { name: 'Revisiones', exact: true }).click(); await reviewerPage.getByRole('button', { name: 'Revisar documento', exact: true }).click();
    const reviewDialog = reviewerPage.getByRole('dialog'); await expect(reviewDialog).not.toContainText(destination); await reviewDialog.getByLabel('Motivo del rechazo').fill('Confirma la modificación del plano.'); await reviewDialog.getByRole('button', { name: 'Rechazar documento', exact: true }).click(); await expect(reviewDialog).toContainText('Rechazado'); await reviewDialog.getByRole('button', { name: 'Cerrar ficha' }).click();
    await page.getByRole('button', { name: 'Actualizar documentos', exact: true }).click(); await page.getByRole('button', { name: 'Ver ficha', exact: true }).click(); await expect(dialog).toContainText('Confirma la modificación del plano.'); await dialog.getByRole('button', { name: 'Enviar a revisión', exact: true }).click(); await expect(dialog).toContainText('Pendiente de decisión'); await dialog.getByRole('button', { name: 'Cerrar ficha' }).click();
    await reviewerPage.getByRole('button', { name: 'Actualizar pendientes', exact: true }).click(); await reviewerPage.getByRole('button', { name: 'Revisar documento', exact: true }).click(); await reviewDialog.getByRole('button', { name: 'Aprobar documento', exact: true }).click(); await expect(reviewDialog).toContainText('Aprobado'); await expect(reviewDialog).not.toContainText('El archivo administrado cambió fuera de AIBID.');
    await reviewDialog.getByRole('button', { name: 'Cerrar ficha' }).click();
    await page.getByRole('tab', { name: 'Cargas', exact: true }).click();
    await page.getByLabel('Archivos PDF', { exact: true }).setInputFiles({ name: 'Para revisión.pdf', mimeType: 'application/pdf', buffer: await readFile(resolve('../testdata/documents/native.pdf')) });
    await page.getByRole('button', { name: 'Guardar borradores' }).click(); await page.getByRole('button', { name: 'Clasificar Para revisión.pdf', exact: true }).click();
    await dialog.getByText('Clasificar o asociar a expediente', { exact: true }).click(); await dialog.getByLabel('Expediente de destino').selectOption({ label: 'PR-H5' }); await dialog.getByLabel('Categoría del documento').selectOption(category.id); await dialog.getByLabel('Tipo del documento').selectOption(kind.id); await dialog.getByRole('button', { name: 'Guardar clasificación' }).click(); await expect(dialog).toContainText('Expediente PR-H5');
    await dialog.getByRole('button', { name: 'Enviar a revisión', exact: true }).click(); await expect(dialog).toContainText('Pendiente de decisión'); await dialog.getByRole('button', { name: 'Cerrar ficha' }).click();
    await reviewerPage.getByRole('button', { name: 'Actualizar pendientes', exact: true }).click(); await reviewerPage.getByRole('button', { name: 'Revisar documento', exact: true }).click(); await expect(reviewDialog).toContainText('Temporal privado'); await expect(reviewDialog).not.toContainText(destination);
    await reviewDialog.getByRole('button', { name: 'Aprobar documento', exact: true }).click(); await expect(reviewDialog.getByText('Guardado definitivo completado', { exact: true })).toBeVisible({ timeout: 25_000 }); await expect(reviewDialog).toContainText('Aprobado'); await expect(reviewDialog).not.toContainText(destination);
    await reviewerPage.setViewportSize({ width: 390, height: 844 }); expect(await reviewerPage.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy(); await reviewerPage.screenshot({ path: 'test-results/h5-review-mobile.png', fullPage: true });
  } finally { await reviewerPage.close(); await rm(destination, { recursive: true, force: true }); }
});
