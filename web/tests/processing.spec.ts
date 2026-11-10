import { test, expect } from '@playwright/test';
import { mkdtemp, copyFile, mkdir, utimes, rm, realpath } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { mkdirSync } from 'node:fs';

const manual = resolve('../dist/manual-capturas-20261028');
async function capture(page: import('@playwright/test').Page, name: string) {
  await page.getByRole('button', { name: 'Cerrar notificación' }).evaluateAll(buttons => buttons.forEach(button => (button as HTMLButtonElement).click()));
  await page.screenshot({ path: join(manual, name), animations: 'disabled' });
}

test('seguimiento automático, duplicados, auditoría y configuración', async ({ page }) => {
  test.setTimeout(120_000);
  mkdirSync(manual, { recursive: true });
  await page.setViewportSize({ width: 1920, height: 1080 });
  const directory = await realpath(await mkdtemp(join(tmpdir(), 'aidex-processing-')));
  await mkdir(join(directory, '$RECYCLE.BIN'));
  for (const name of ['Original.pdf', 'Copia.pdf', '$RECYCLE.BIN/Omitido.pdf']) {
    await copyFile(resolve('../testdata/documents/native.pdf'), join(directory, name)); const past = new Date(Date.now() - 5000); await utimes(join(directory, name), past, past);
  }
  try {
    await page.goto('/login'); await page.getByLabel('Usuario', { exact: true }).fill('admin-e2e'); await page.getByLabel('Contraseña', { exact: true }).fill('Browser-fixture-password-2026'); await page.getByRole('button', { name: 'Entrar', exact: true }).click();
    await page.getByRole('link', { name: 'Bibliotecas', exact: true }).click(); await page.getByRole('button', { name: 'Nueva biblioteca' }).click(); await page.getByLabel('Nombre de la biblioteca').fill('Seguimiento E2E');  await page.getByRole('button', { name: 'Crear biblioteca', exact: true }).click();
    await page.getByRole('tab', { name: 'Carpetas', exact: true }).click(); await page.getByLabel('Ruta de la carpeta en el servidor').fill(directory); await page.getByRole('button', { name: 'Inspeccionar carpeta' }).click(); await page.getByRole('button', { name: 'Registrar e indexar' }).click();
    await page.getByRole('tab', { name: 'Procesamiento', exact: true }).click();
    await expect(page.locator('.metric').filter({ hasText: 'Procesados' }).locator('strong')).toHaveText('2', { timeout: 30000 });
    await expect(page.getByText('1 carpetas del sistema omitidas', { exact: false })).toBeVisible();
    await expect(page.locator('.metric').filter({ hasText: 'Archivos registrados' }).locator('strong')).toHaveText('2');
    await capture(page, 'procesamiento-biblioteca.png');
    await page.screenshot({ path: 'test-results/processing-progress.png', fullPage: true });
    await page.getByRole('tab', { name: 'Duplicados', exact: true }).click();
    await expect(page.getByRole('heading', { name: /^\d+ archivos con el mismo contenido$/ })).toBeVisible();
    await expect(page.locator('.duplicate-file').first()).toBeVisible();
    await capture(page, 'duplicados-grupos.png');
    while (await page.getByRole('button', { name: 'Copia.pdf', exact: true }).count() === 0) {
      const paging = page.getByRole('navigation', { name: 'Archivos', exact: true });
      const before = await paging.textContent();
      await paging.getByRole('button', { name: 'Siguiente', exact: true }).click();
      await expect(paging).not.toHaveText(before || '');
      await expect(page.getByText('Consultando archivos…')).toHaveCount(0);
    }
    await page.getByRole('button', { name: 'Copia.pdf', exact: true }).click(); await expect(page.getByRole('dialog')).toBeVisible();
    await capture(page, 'ficha-documental-cierre-x.png');
    await page.getByText('Huella y trazabilidad', { exact: true }).click();
    await page.getByRole('button', { name: 'Consultar historial', exact: true }).click();
    const history = page.getByRole('table', { name: 'Historial del documento' });
    await expect(history.getByRole('cell', { name: 'Documento detectado', exact: true })).toBeVisible();
    await expect(history.getByRole('cell', { name: 'Sistema', exact: true })).toBeVisible();
    await capture(page, 'huella-historial-documento.png');
    await history.getByRole('row').filter({ hasText: 'Documento detectado' }).getByRole('button', { name: 'Ver detalles del evento' }).click();
    await expect(page.getByRole('heading', { name: 'Documento detectado', exact: true })).toBeVisible();
    await capture(page, 'huella-detalle-evento.png');
    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog')).toHaveCount(1);
    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog')).toHaveCount(0);
    await page.getByRole('tab', { name: 'Auditoría', exact: true }).click(); await expect(page.getByRole('columnheader', { name: 'Descripción', exact: true })).toBeVisible();
    await page.getByRole('combobox', { name: 'Tipo de evento', exact: true }).selectOption('document.discovered'); await page.getByRole('button', { name: 'Aplicar filtros' }).click(); await expect(page.getByRole('cell', { name: 'Documento detectado', exact: true })).toHaveCount(2);
    await capture(page, 'auditoria-biblioteca.png');
    await page.getByRole('button', { name: 'Ver detalles del evento', exact: true }).first().click(); await expect(page.getByRole('dialog')).toContainText('Sistema'); await page.screenshot({ path: 'test-results/audit-details.png', fullPage: true }); await page.getByRole('button', { name: 'Cerrar detalle del evento' }).click();
    await page.getByRole('textbox', { name: 'Buscar', exact: true }).fill('no-existe-e2e'); await page.getByRole('button', { name: 'Aplicar filtros' }).click(); await expect(page.getByText('No hay eventos que coincidan con estos filtros.')).toBeVisible(); await page.getByRole('button', { name: 'Restablecer' }).click();
    await page.getByRole('tab', { name: 'Configuración', exact: true }).click(); await expect(page.getByRole('group', { name: 'Configuración general' })).toBeVisible(); await expect(page.getByRole('group', { name: 'Almacenamiento', exact: true })).toHaveCount(0);
    await page.getByRole('combobox', { name: 'Modalidad', exact: true }).selectOption('hybrid'); await expect(page.getByRole('group', { name: 'Almacenamiento', exact: true })).toBeVisible(); await expect(page.getByRole('group', { name: 'Identificadores y expedientes' })).toBeVisible();
    await page.getByRole('link', { name: 'Buscar documentos', exact: true }).click();
    await page.getByLabel('Texto a buscar').fill('Copia.pdf');
    await page.getByRole('button', { name: 'Buscar documentos', exact: true }).click();
    await expect(page.getByText('1 documento encontrado')).toBeVisible();
    await capture(page, 'busqueda-resultados-visualizar.png');
    await page.getByRole('link', { name: 'Eventos', exact: true }).click();
    await expect(page.getByRole('heading', { name: 'Auditoría', exact: true })).toBeVisible();
    await capture(page, 'auditoria-global.png');
    await page.getByRole('combobox', { name: 'Tipo de evento', exact: true }).selectOption('search.executed');
    await page.getByRole('button', { name: 'Aplicar filtros' }).click();
    await expect(page.getByRole('button', { name: 'Ver detalles del evento' }).first()).toBeVisible();
    await page.getByRole('button', { name: 'Ver detalles del evento' }).first().click();
    await expect(page.getByRole('dialog')).toContainText('Resultados encontrados');
    await capture(page, 'auditoria-busqueda-exacta.png');
    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog')).toHaveCount(0);
    await page.getByRole('link', { name: 'Usuarios', exact: true }).click();
    await page.getByRole('button', { name: /Editar a admin-e2e/ }).click();
    await capture(page, 'usuarios-perfiles-auditores.png');
    await page.getByRole('link', { name: 'Bibliotecas', exact: true }).click();
    await page.getByRole('button', { name: 'Abrir biblioteca Seguimiento E2E' }).click();
    await page.getByRole('tab', { name: 'Documentos', exact: true }).click();
    await capture(page, 'documentos-explorador-visualizar.png');
    await page.getByRole('button', { name: 'Todas las bibliotecas' }).click();
    await page.getByRole('button', { name: 'Nueva biblioteca' }).click();
    await page.getByLabel('Nombre de la biblioteca').fill('Manual administrada');
    await page.getByLabel('Modalidad de biblioteca').selectOption('managed');
    await page.getByRole('button', { name: 'Crear biblioteca', exact: true }).click();
    await page.getByRole('tab', { name: 'Catálogos', exact: true }).click();
    await page.getByLabel('Nombre de categoría').fill('Contratos');
    await page.getByRole('button', { name: 'Crear categoría' }).click();
    await page.getByLabel('Categoría del tipo').selectOption({ label: 'Contratos' });
    await page.getByLabel('Nombre del tipo').fill('Contrato firmado');
    await page.getByRole('button', { name: 'Crear tipo' }).click();
    await page.getByLabel('Nombre de plantilla').fill('Expediente de prueba');
    await page.getByRole('button', { name: 'Agregar Contrato firmado' }).click();
    await expect(page.getByRole('button', { name: 'Remover Contrato firmado de la plantilla' })).toBeVisible();
    await page.locator('.template-form').scrollIntoViewIfNeeded();
    await capture(page, 'plantilla-requisitos-remover.png');
    await page.getByRole('tab', { name: 'Cargas', exact: true }).click();
    await page.getByLabel('Archivos', { exact: true }).setInputFiles([resolve('../testdata/documents/native.pdf'), resolve('../testdata/documents/native.pdf')]);
    await expect(page.getByRole('button', { name: 'Remover native.pdf de la selección' })).toHaveCount(2);
    await capture(page, 'carga-multiple-remover.png');
    await page.getByRole('button', { name: 'Remover native.pdf de la selección' }).first().click();
    await expect(page.getByRole('button', { name: 'Remover native.pdf de la selección' })).toHaveCount(1);
    await page.setViewportSize({ width: 390, height: 844 }); expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
  } finally { await rm(directory, { recursive: true, force: true }); }
});
