import { test, expect } from '@playwright/test';
import { mkdtemp, copyFile, mkdir, utimes, rm, realpath } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';

test('seguimiento automático, duplicados, auditoría y configuración', async ({ page }) => {
  const directory = await realpath(await mkdtemp(join(tmpdir(), 'aibid-processing-')));
  await mkdir(join(directory, '$RECYCLE.BIN'));
  for (const name of ['Original.pdf', 'Copia.pdf', '$RECYCLE.BIN/Omitido.pdf']) {
    await copyFile(resolve('../testdata/documents/native.pdf'), join(directory, name)); const past = new Date(Date.now() - 5000); await utimes(join(directory, name), past, past);
  }
  try {
    await page.goto('/login'); await page.getByLabel('Usuario', { exact: true }).fill('admin-e2e'); await page.getByLabel('Contraseña', { exact: true }).fill('Browser-fixture-password-2026'); await page.getByRole('button', { name: 'Entrar', exact: true }).click();
    await page.getByRole('link', { name: 'Bibliotecas', exact: true }).click(); await page.getByRole('button', { name: 'Nueva biblioteca' }).click(); await page.getByLabel('Nombre de la biblioteca').fill('Seguimiento E2E'); await page.getByLabel('Asignarme como gestor').check(); await page.getByRole('button', { name: 'Crear biblioteca', exact: true }).click();
    await page.getByRole('tab', { name: 'Carpetas', exact: true }).click(); await page.getByLabel('Ruta de la carpeta en el servidor').fill(directory); await page.getByRole('button', { name: 'Inspeccionar carpeta' }).click(); await page.getByRole('button', { name: 'Registrar e indexar' }).click();
    await page.getByRole('tab', { name: 'Procesamiento', exact: true }).click();
    await expect(page.locator('.metric').filter({ hasText: 'Procesados' }).locator('strong')).toHaveText('2', { timeout: 30000 });
    await expect(page.getByText('1 carpetas del sistema omitidas', { exact: false })).toBeVisible();
    await expect(page.locator('.metric').filter({ hasText: 'Archivos registrados' }).locator('strong')).toHaveText('2');
    await page.screenshot({ path: 'test-results/processing-progress.png', fullPage: true });
    await page.getByRole('tab', { name: 'Duplicados', exact: true }).click();
    await expect(page.getByRole('heading', { name: /^\d+ archivos con el mismo contenido$/ })).toBeVisible();
    await expect(page.locator('.duplicate-file').first()).toBeVisible();
    while (await page.getByRole('button', { name: 'Copia.pdf', exact: true }).count() === 0) {
      const paging = page.getByRole('navigation', { name: 'Archivos', exact: true });
      const before = await paging.textContent();
      await paging.getByRole('button', { name: 'Siguiente', exact: true }).click();
      await expect(paging).not.toHaveText(before || '');
      await expect(page.getByText('Consultando archivos…')).toHaveCount(0);
    }
    await page.getByRole('button', { name: 'Copia.pdf', exact: true }).click(); await expect(page.getByRole('dialog')).toBeVisible();
    await page.getByText('Huella y trazabilidad', { exact: true }).click();
    await page.getByRole('button', { name: 'Consultar historial', exact: true }).click();
    const history = page.getByRole('table', { name: 'Historial del documento' });
    await expect(history.getByRole('cell', { name: 'Documento detectado', exact: true })).toBeVisible();
    await expect(history.getByRole('cell', { name: 'Sistema', exact: true })).toBeVisible();
    await history.getByRole('row').filter({ hasText: 'Documento detectado' }).getByRole('button', { name: 'Ver detalles' }).click();
    await expect(page.getByRole('heading', { name: 'Documento detectado', exact: true })).toBeVisible();
    await page.getByRole('button', { name: 'Cerrar detalle' }).click();
    await page.getByRole('button', { name: 'Cerrar ficha' }).click();
    await page.getByRole('tab', { name: 'Auditoría', exact: true }).click(); await expect(page.getByRole('columnheader', { name: 'Descripción', exact: true })).toBeVisible();
    await page.getByRole('combobox', { name: 'Tipo de evento', exact: true }).selectOption('document.discovered'); await page.getByRole('button', { name: 'Aplicar filtros' }).click(); await expect(page.getByRole('cell', { name: 'Documento detectado', exact: true })).toHaveCount(2);
    await page.getByRole('button', { name: 'Ver detalles', exact: true }).first().click(); await expect(page.getByRole('dialog')).toContainText('Sistema'); await page.screenshot({ path: 'test-results/audit-details.png', fullPage: true }); await page.getByRole('button', { name: 'Cerrar detalle' }).click();
    await page.getByRole('textbox', { name: 'Buscar', exact: true }).fill('no-existe-e2e'); await page.getByRole('button', { name: 'Aplicar filtros' }).click(); await expect(page.getByText('No hay eventos que coincidan con estos filtros.')).toBeVisible(); await page.getByRole('button', { name: 'Restablecer' }).click();
    await page.getByRole('tab', { name: 'Configuración', exact: true }).click(); await expect(page.getByRole('group', { name: 'Configuración general' })).toBeVisible(); await expect(page.getByRole('group', { name: 'Almacenamiento', exact: true })).toHaveCount(0);
    await page.getByRole('combobox', { name: 'Modalidad', exact: true }).selectOption('hybrid'); await expect(page.getByRole('group', { name: 'Almacenamiento', exact: true })).toBeVisible(); await expect(page.getByRole('group', { name: 'Identificadores y expedientes' })).toBeVisible();
    await page.setViewportSize({ width: 390, height: 844 }); expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
  } finally { await rm(directory, { recursive: true, force: true }); }
});
