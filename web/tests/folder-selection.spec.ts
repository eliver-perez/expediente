import {test, expect, type Page} from '@playwright/test';
import {mkdtemp, realpath, mkdir, rm} from 'node:fs/promises';
import {join} from 'node:path';
import {tmpdir} from 'node:os';
async function library(page: Page, name: string) {
  await page.goto('/login'); await page.getByLabel('Usuario', {exact:true}).fill('admin-e2e'); await page.getByLabel('Contraseña', {exact:true}).fill('Browser-fixture-password-2026');
  await page.getByRole('button', {name:'Entrar',exact:true}).click(); await expect(page.getByRole('heading',{name:'Mi cuenta',exact:true})).toBeVisible();
  await page.getByRole('link',{name:'Bibliotecas',exact:true}).click(); await page.getByRole('button',{name:'Nueva biblioteca',exact:true}).click();
  await page.getByLabel('Nombre de la biblioteca',{exact:true}).fill(name); await page.getByLabel('Modalidad de biblioteca').selectOption('hybrid');
  await page.getByRole('button',{name:'Crear biblioteca',exact:true}).click(); await page.getByRole('tab',{name:'Carpetas',exact:true}).click();
}
test('cliente remoto explora únicamente raíces habilitadas y selecciona subcarpetas del servidor',async({page})=>{
  const directory=await realpath(await mkdtemp(join(tmpdir(),'aidex-browse-'))); await mkdir(join(directory,'Expedientes'));
  try {
    await library(page,'Explorador remoto'); const path=page.getByLabel('Ruta de la carpeta en el servidor'); await path.fill(directory);
    await page.getByRole('button',{name:'Inspeccionar carpeta',exact:true}).click(); await page.getByRole('button',{name:'Registrar e indexar',exact:true}).click();
    await expect(page.locator('.root-row').filter({hasText:directory})).toBeVisible();
    // Real API request with the proxy header exercises the server's remote path.
    await page.route('**/api/v1/folder-selections',route=>route.continue({headers:{...route.request().headers(),'X-Forwarded-For':'192.0.2.1'}}));
    await page.getByRole('button',{name:'Seleccionar carpeta',exact:true}).click(); const dialog=page.getByRole('dialog');
    await expect(dialog.getByRole('heading',{name:'Seleccionar carpeta del servidor'})).toBeVisible();
    await dialog.getByRole('button',{name:directory.split('/').pop()!,exact:true}).click();
    await dialog.getByRole('button',{name:'Expedientes',exact:true}).click(); await dialog.getByRole('button',{name:'Seleccionar esta carpeta',exact:true}).click();
    await expect(path).toHaveValue(join(directory,'Expedientes')); await expect(page.getByText('Esta subcarpeta ya pertenece a una raíz.',{exact:false})).toBeVisible();
  } finally {await rm(directory,{recursive:true,force:true});}
});
test('respuesta del auxiliar simulado completa ruta e inspección; cancelar conserva alternativa manual',async({page})=>{
  const directory=await realpath(await mkdtemp(join(tmpdir(),'aidex-picked-')));
  try {
    await library(page,'Respuesta del selector'); await page.getByLabel('Origen de la carpeta').selectOption('managed');
    const ticket='a'.repeat(64); let cancelled=false;
    // Plumbing test only: no native dialog, protocol handler or UAC is invoked.
    await page.route('**/api/v1/folder-selections',route=>route.fulfill({status:201,json:{mode:'native',ticket,launch_url:'#selector-prueba'}}));
    await page.route(`**/api/v1/folder-selections/${ticket}`,route=>route.fulfill({json:{done:true,server_path:cancelled?'':directory,code:cancelled?'cancelled':''}}));
    await page.getByRole('button',{name:'Seleccionar carpeta',exact:true}).click();
    await expect(page.getByLabel('Ruta de la carpeta en el servidor')).toHaveValue(directory); await expect(page.getByRole('button',{name:'Registrar destino administrado',exact:true})).toBeVisible();
    cancelled=true; await page.getByRole('button',{name:'Seleccionar carpeta',exact:true}).click();
    const dialog=page.getByRole('dialog'); await expect(dialog).toContainText('Selección cancelada'); await dialog.getByRole('button',{name:'Cerrar ventana'}).click();
    await expect(page.getByLabel('Ruta de la carpeta en el servidor')).toBeEnabled();
  } finally {await rm(directory,{recursive:true,force:true});}
});
test('modal Windows ofrece autorización administrada, SVG y error visible sin bloquear comprobación',async({page})=>{
  await page.route('**/api/v1/about',async route=>{const response=await route.fetch();await route.fulfill({json:{...await response.json(),os:'windows'}});});
  await library(page,'Modal Windows'); await page.getByLabel('Origen de la carpeta').selectOption('managed');
  await page.getByLabel('Ruta de la carpeta en el servidor').fill('C:\\Users\\Ana\\Documents\\Archivo');
  await page.route('**/api/v1/libraries/*/root-plans',route=>route.fulfill({status:422,json:{error:{code:'ROOT_PERMISSION_DENIED',message:'Necesita autorización'}}}));
  await page.getByRole('button',{name:'Inspeccionar carpeta',exact:true}).click();const dialog=page.getByRole('dialog');
  await expect(dialog).toContainText('carpeta Archivo');await expect(dialog).toContainText('leer, guardar y administrar');await expect(dialog).not.toContainText('NT SERVICE');await expect(dialog).not.toContainText('C:\\Users');
  for(const name of ['Autorizar carpeta','Comprobar acceso']) {const svg=dialog.getByRole('button',{name,exact:true}).locator('svg');await expect(svg).toBeVisible();expect((await svg.boundingBox())?.width).toBe(16);}
  await page.route('**/api/v1/local-folder-authorizations',route=>route.fulfill({status:403,json:{error:{code:'LOCAL_AUTHORIZATION_REQUIRED',message:'Abre AIDEX desde el equipo servidor para autorizar.'}}}));
  await dialog.getByRole('button',{name:'Autorizar carpeta',exact:true}).click();await expect(dialog).toContainText('Abre AIDEX desde el equipo servidor');await expect(dialog.getByRole('button',{name:'Comprobar acceso',exact:true})).toBeEnabled();
  await page.screenshot({path:'../evidencias/seleccion-carpetas-20261103/modal-windows.png'});
});
