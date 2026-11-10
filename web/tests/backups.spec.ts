import { test, expect } from '@playwright/test';

test('Fase 2: generar, descargar e inspeccionar un respaldo básico', async ({ page }) => {
  await page.goto('/login');
  await page.getByLabel('Usuario', { exact: true }).fill('admin-e2e');
  await page.getByLabel('Contraseña', { exact: true }).fill('Browser-fixture-password-2026');
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await page.getByRole('link', { name: 'Respaldos y restauración' }).click();
  await expect(page.getByRole('heading', { name: 'Respaldos y restauración' })).toBeVisible();
  await page.getByRole('button', { name: 'Generar respaldo' }).click();
  const row = page.getByRole('row').filter({ hasText: 'Respaldo básico' }).first();
  await expect(row).toContainText('Completado', { timeout: 30_000 });
  const downloadPromise = page.waitForEvent('download');
  await row.getByRole('link', { name: 'Descargar' }).click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/\.aidexbak$/);
  const pathname = await download.path();
  expect(pathname).toBeTruthy();
  await page.getByLabel('Archivo .aidexbak').setInputFiles(pathname!);
  await page.getByRole('button', { name: 'Validar y consultar contenido' }).click();
  await expect(page.getByRole('heading', { name: 'Vista previa del respaldo' })).toBeVisible({ timeout: 30_000 });
  await expect(page.getByText('Las rutas vinculadas requieren comprobación en este equipo')).toBeVisible();
  await page.getByRole('button', { name: 'Descartar vista previa' }).click();
  await expect(page.getByRole('heading', { name: 'Vista previa del respaldo' })).toHaveCount(0);
});
