import { test, expect } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import { resolve } from 'node:path';

const evidence = resolve('../evidencias/catalogos-etapa1');
async function clearToasts(page: import('@playwright/test').Page) {
  while (await page.getByRole('button', { name: 'Cerrar notificación' }).count()) await page.getByRole('button', { name: 'Cerrar notificación' }).first().click();
}

test('Etapa 1: catálogos, plantilla versionada, tarjetas y diseño responsive', async ({ page }) => {
  test.setTimeout(120_000);
  mkdirSync(evidence, { recursive: true });
  await page.setViewportSize({ width: 1920, height: 1080 });
  await page.goto('/login');
  await page.getByLabel('Usuario', { exact: true }).fill('admin-e2e');
  await page.getByLabel('Contraseña', { exact: true }).fill('Browser-fixture-password-2026');
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await page.getByRole('link', { name: 'Bibliotecas', exact: true }).click();
  await page.getByRole('button', { name: 'Nueva biblioteca' }).click();
  await page.getByLabel('Nombre de la biblioteca').fill('Biblioteca visual de prueba');
  await page.getByLabel('Modalidad de biblioteca').selectOption('managed');
  await page.getByLabel('Descripción de la biblioteca (opcional)').fill('Un espacio documental para comprobar la composición completa de las tarjetas.');
  await page.getByRole('button', { name: 'Crear biblioteca', exact: true }).click();
  await page.getByRole('button', { name: 'Todas las bibliotecas' }).click();
  const card = page.getByRole('article').filter({ hasText: 'Biblioteca visual de prueba' });
  await expect(card).toBeVisible();
  await expect(card.getByRole('button', { name: 'Abrir biblioteca Biblioteca visual de prueba' })).toHaveCSS('min-height', '220px');
  await clearToasts(page); await page.screenshot({ path: `${evidence}/bibliotecas-escritorio.png` });
  await card.getByRole('button', { name: 'Abrir biblioteca Biblioteca visual de prueba' }).hover({ position: { x: 120, y: 190 } });
  await expect(card.getByRole('button', { name: 'Abrir biblioteca Biblioteca visual de prueba' })).toHaveCSS('background-color', 'rgb(237, 246, 240)');
  await page.screenshot({ path: `${evidence}/bibliotecas-hover-escritorio.png` });
  await card.getByLabel('Opciones de Biblioteca visual de prueba').click();
  await expect(card.getByText('Archivar biblioteca')).toBeVisible();
  await card.getByLabel('Opciones de Biblioteca visual de prueba').click();
  await card.getByRole('button', { name: 'Abrir biblioteca Biblioteca visual de prueba' }).click({ position: { x: 120, y: 190 } });
  await page.getByRole('tab', { name: 'Catálogos' }).click();

  await page.getByLabel('Nombre de categoría').fill('Planos');
  await page.getByRole('button', { name: 'Crear categoría' }).click();
  await expect(page.getByLabel('Nombre de categoría')).toHaveValue('');
  await page.getByLabel('Nombre de categoría').fill('Contratos');
  await page.getByRole('button', { name: 'Crear categoría' }).click();
  await page.getByLabel('Buscar categoría').fill('Plano');
  await expect(page.getByRole('button', { name: 'Editar categoría Planos' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Editar categoría Contratos' })).toHaveCount(0);
  await page.getByLabel('Buscar categoría').fill('');
  await page.getByRole('button', { name: 'Editar categoría Planos' }).click();
  await expect(page.getByRole('heading', { name: 'Editar categoría' })).toBeVisible();
  await page.getByLabel('Nombre de categoría').fill('Planos y croquis');
  await page.getByRole('button', { name: 'Guardar categoría' }).click();
  await expect(page.getByLabel('Nombre de categoría')).toHaveValue('');
  await expect(page.getByRole('heading', { name: 'Nueva categoría' })).toBeVisible();
  await page.getByLabel('Nombre de categoría').fill('   ');
  await page.getByRole('button', { name: 'Crear categoría' }).click();
  await expect(page.getByRole('alert')).toContainText('Nombre o catálogo inválido');
  await expect(page.getByLabel('Nombre de categoría')).toHaveValue('   ');
  await page.getByLabel('Nombre de categoría').fill('Facturas');
  await page.getByRole('button', { name: 'Crear categoría' }).click();
  await page.evaluate(async () => {
    const token = (await (await fetch('/api/v1/auth/session')).json()).csrf_token as string;
    const library = (await (await fetch('/api/v1/libraries')).json()).items[0].id as string;
    for (let index = 1; index <= 51; index++) {
      const response = await fetch(`/api/v1/libraries/${library}/categories`, { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': token }, body: JSON.stringify({ name: `Categoría adicional ${String(index).padStart(2, '0')}` }) });
      if (!response.ok) throw new Error(`No se pudo crear la categoría ${index}: ${response.status}`);
    }
  });
  await page.getByRole('tab', { name: 'Documentos', exact: true }).click();
  await page.getByRole('tab', { name: 'Catálogos', exact: true }).click();
  await expect(page.locator('.catalog-table-card').first().getByText('54 registradas')).toBeVisible();
  await expect(page.getByLabel('Categoría del tipo').locator('option')).toHaveCount(55);
  await expect(page.locator('.catalog-scroll').first()).toHaveCSS('overflow-y', 'auto');
  await clearToasts(page); await page.screenshot({ path: `${evidence}/categorias-escritorio.png` });

  await page.getByLabel('Categoría del tipo').selectOption({ label: 'Planos y croquis' });
  await page.getByLabel('Nombre del tipo').fill('Plano general');
  await page.getByLabel('Admite múltiples archivos').check();
  await page.getByLabel('Exige título descriptivo (por ejemplo, Varios)').check();
  await page.getByRole('button', { name: 'Crear tipo' }).click();
  await expect(page.getByLabel('Nombre del tipo')).toHaveValue('');
  await expect(page.getByLabel('Admite múltiples archivos')).not.toBeChecked();
  await page.getByLabel('Categoría del tipo').selectOption({ label: 'Contratos' });
  await page.getByLabel('Nombre del tipo').fill('Contrato firmado');
  await page.getByRole('button', { name: 'Crear tipo' }).click();
  await page.getByLabel('Buscar tipo', { exact: true }).first().fill('Plano');
  await expect(page.getByRole('button', { name: 'Editar tipo Plano general' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Editar tipo Contrato firmado' })).toHaveCount(0);
  await page.getByLabel('Buscar tipo', { exact: true }).first().fill('');
  await page.getByRole('button', { name: 'Editar tipo Plano general' }).click();
  await expect(page.getByLabel('Admite múltiples archivos')).toBeChecked();
  await expect(page.getByLabel('Exige título descriptivo (por ejemplo, Varios)')).toBeChecked();
  await page.getByLabel('Nombre del tipo').fill('Plano ejecutivo');
  await page.getByRole('button', { name: 'Guardar tipo' }).click();
  await expect(page.getByLabel('Nombre del tipo')).toHaveValue('');
  await expect(page.getByRole('heading', { name: 'Nuevo tipo documental' })).toBeVisible();
  await page.getByLabel('Categoría del tipo').selectOption({ label: 'Facturas' });
  await page.getByLabel('Nombre del tipo').fill('   ');
  await page.getByRole('button', { name: 'Crear tipo' }).click();
  await expect(page.getByRole('alert')).toContainText('Nombre o catálogo inválido');
  await expect(page.getByLabel('Nombre del tipo')).toHaveValue('   ');
  await page.getByLabel('Nombre del tipo').fill('Factura única');
  await page.getByRole('button', { name: 'Crear tipo' }).click();
  await clearToasts(page); await page.locator('.catalog-section-grid').nth(1).scrollIntoViewIfNeeded(); await page.screenshot({ path: `${evidence}/tipos-escritorio.png` });

  await page.getByLabel('Nombre de plantilla').fill('Expediente de obra');
  await page.getByRole('button', { name: 'Agregar Plano ejecutivo' }).click();
  await page.getByRole('button', { name: 'Agregar Contrato firmado' }).click();
  await expect(page.locator('.template-summary-card.teal strong')).toHaveText('2');
  await page.locator('.template-requirement-row').filter({ hasText: 'Plano ejecutivo' }).getByLabel('Cantidad requerida').fill('3');
  await page.locator('.template-requirement-row').filter({ hasText: 'Contrato firmado' }).getByLabel('Obligatorio').uncheck();
  await expect(page.locator('.template-summary-card.blue strong')).toHaveText('4');
  await expect(page.locator('.template-summary-card.green strong')).toHaveText('1');
  await expect(page.locator('.template-summary-card.amber strong')).toHaveText('1');
  await page.getByRole('button', { name: 'Quitar Contrato firmado de la plantilla' }).click();
  await expect(page.locator('.template-summary-card.teal strong')).toHaveText('1');
  await page.getByRole('button', { name: 'Agregar Contrato firmado' }).click();
  await page.locator('.template-requirement-row').filter({ hasText: 'Contrato firmado' }).getByLabel('Obligatorio').uncheck();
  await page.getByLabel('Nombre de plantilla').fill('   ');
  await page.getByRole('button', { name: 'Crear plantilla', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('Nombre inválido');
  await expect(page.getByLabel('Nombre de plantilla')).toHaveValue('   ');
  await expect(page.locator('.template-summary-card.teal strong')).toHaveText('2');
  await page.getByLabel('Nombre de plantilla').fill('Expediente de obra');
  await clearToasts(page);
  await page.locator('.template-form').scrollIntoViewIfNeeded();
  await page.screenshot({ path: `${evidence}/plantilla-escritorio.png` });
  for (const [name, width, height] of [['tablet', 1024, 768], ['movil', 390, 844]] as const) {
    await page.setViewportSize({ width, height });
    await page.locator('.template-builder').evaluate(element => element.scrollIntoView({ block: 'start' }));
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
    await page.screenshot({ path: `${evidence}/plantilla-${name}.png` });
  }
  await page.setViewportSize({ width: 1920, height: 1080 });
  await page.getByRole('button', { name: 'Crear plantilla', exact: true }).click();
  await expect(page.getByLabel('Nombre de plantilla')).toHaveValue('');
  await expect(page.locator('.template-summary-card.teal strong')).toHaveText('0');
  await page.getByLabel('Buscar plantilla').fill('obra');
  await page.locator('.template-accordion > summary').filter({ hasText: 'Expediente de obra' }).click();
  await page.locator('.template-version > summary').filter({ hasText: 'Versión 1' }).click();
  await expect(page.locator('.template-version').filter({ hasText: 'Versión 1' })).toContainText('Plano ejecutivo');
  await expect(page.locator('.template-version').filter({ hasText: 'Versión 1' })).toContainText('Contrato firmado');
  await page.getByRole('button', { name: 'Crear nueva versión de Expediente de obra' }).click();
  await expect(page.getByRole('heading', { name: 'Nueva versión de Expediente de obra' })).toBeVisible();
  await page.getByRole('button', { name: 'Quitar Contrato firmado de la plantilla' }).click();
  await page.getByRole('button', { name: 'Guardar nueva versión' }).click();
  await expect(page.getByRole('heading', { name: 'Crear plantilla' })).toBeVisible();
  const templateCard = page.locator('.template-accordion').filter({ hasText: 'Expediente de obra' });
  if (await templateCard.getAttribute('open') === null) await templateCard.locator(':scope > summary').click();
  const firstVersion = page.locator('.template-version').filter({ hasText: 'Versión 1' });
  if (await firstVersion.getAttribute('open') === null) await firstVersion.locator(':scope > summary').click();
  await expect(firstVersion.getByText('Contrato firmado')).toBeVisible();
  const secondVersion = page.locator('.template-version').filter({ hasText: 'Versión 2' });
  if (await secondVersion.getAttribute('open') === null) await secondVersion.locator(':scope > summary').click();
  await expect(secondVersion).not.toContainText('Contrato firmado');
  await clearToasts(page); await page.locator('.template-list').scrollIntoViewIfNeeded();
  await page.screenshot({ path: `${evidence}/versiones-escritorio.png` });

  for (const [name, width, height] of [['tablet', 1024, 768], ['movil', 390, 844]] as const) {
    await page.setViewportSize({ width, height });
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
    await page.getByRole('button', { name: 'Todas las bibliotecas' }).click();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
    await page.screenshot({ path: `${evidence}/bibliotecas-${name}.png` });
    if (name === 'tablet') {
      await page.getByRole('button', { name: 'Abrir biblioteca Biblioteca visual de prueba' }).click();
      await page.getByRole('tab', { name: 'Catálogos' }).click();
    }
  }
});
