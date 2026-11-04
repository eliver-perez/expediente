import {test,expect,type Page} from '@playwright/test';
import {mkdtemp,realpath,rm} from 'node:fs/promises';
import {join} from 'node:path';
import {tmpdir} from 'node:os';

async function library(page:Page,name:string){
 await page.goto('/login');await page.getByLabel('Usuario',{exact:true}).fill('admin-e2e');await page.getByLabel('Contraseña',{exact:true}).fill('Browser-fixture-password-2026');await page.getByRole('button',{name:'Entrar',exact:true}).click();await expect(page.getByRole('heading',{name:'Mi cuenta',exact:true})).toBeVisible();
 await page.getByRole('link',{name:'Bibliotecas',exact:true}).click();await page.getByRole('button',{name:'Nueva biblioteca',exact:true}).click();await page.getByLabel('Nombre de la biblioteca',{exact:true}).fill(name);await page.getByLabel('Modalidad de biblioteca').selectOption('hybrid');await page.getByRole('button',{name:'Crear biblioteca',exact:true}).click();await page.getByRole('tab',{name:'Carpetas',exact:true}).click();
}

test('destino inicial, cambio desde tarjetas y limpieza de inspecciones',async({page})=>{
 page.setDefaultTimeout(10000);
 const one=await realpath(await mkdtemp(join(tmpdir(),'aidex-destino-uno-')));const two=await realpath(await mkdtemp(join(tmpdir(),'aidex-destino-dos-')));
 try{
  await library(page,'Destinos desde tarjetas');const path=page.getByLabel('Ruta de la carpeta en el servidor');const source=page.getByLabel('Origen de la carpeta');await source.selectOption('managed');
  await path.fill(one);await page.getByRole('button',{name:'Inspeccionar carpeta',exact:true}).click();await expect(page.getByText('Carpeta disponible como destino administrado.',{exact:true})).toBeVisible();
  await path.fill(two);await expect(page.getByText('Carpeta disponible como destino administrado.',{exact:true})).toHaveCount(0);await expect(page.getByRole('button',{name:'Registrar destino administrado',exact:true})).toHaveCount(0);
  await path.fill(one);await page.getByRole('button',{name:'Inspeccionar carpeta',exact:true}).click();await expect(page.getByRole('button',{name:'Registrar destino administrado',exact:true})).toBeVisible();await source.selectOption('linked');await expect(page.getByRole('button',{name:'Registrar destino administrado',exact:true})).toHaveCount(0);
  await source.selectOption('managed');await page.getByRole('button',{name:'Inspeccionar carpeta',exact:true}).click();await page.getByRole('button',{name:'Registrar destino administrado',exact:true}).click();await expect(page.locator('.root-row').filter({hasText:one})).toContainText('Destino predeterminado');
  await path.fill(two);await page.getByRole('button',{name:'Inspeccionar carpeta',exact:true}).click();await page.getByRole('button',{name:'Registrar destino administrado',exact:true}).click();const card=page.locator('.root-row').filter({hasText:two});await card.getByRole('button',{name:'Establecer como destino',exact:true}).click();await expect(card).toContainText('Destino predeterminado');await expect(page.getByText('Destino predeterminado',{exact:true})).toHaveCount(1);
  await card.getByText('Configurar carpeta',{exact:true}).click();await card.getByLabel('Verificación habilitada').uncheck();await card.getByRole('button',{name:'Guardar verificación',exact:true}).click();await expect(page.getByText('Destino predeterminado',{exact:true})).toHaveCount(0);await expect(page.getByText('Selecciona un destino administrado activo antes de incorporar nuevos documentos.',{exact:false})).toBeVisible();
 }finally{await rm(one,{recursive:true,force:true});await rm(two,{recursive:true,force:true});}
});

test('macOS ofrece autorización administrada y descarta respuestas de una ruta anterior',async({page})=>{
 page.setDefaultTimeout(10000);await page.setViewportSize({width:1920,height:1080});await library(page,'Autorización administrada');await page.getByLabel('Origen de la carpeta').selectOption('managed');const path=page.getByLabel('Ruta de la carpeta en el servidor');await path.fill('/Users/Prueba/Biblioteca');
 let diagnostic='ROOT_PERMISSION_DENIED';
 await page.route('**/api/v1/libraries/*/root-plans',route=>route.fulfill({status:422,contentType:'application/json',body:JSON.stringify({error:{code:diagnostic,message:'El destino necesita permisos de escritura.'}})}));
 await page.getByRole('button',{name:'Inspeccionar carpeta',exact:true}).click();const dialog=page.getByRole('dialog');await expect(dialog.getByRole('button',{name:'Autorizar carpeta',exact:true})).toBeVisible();await expect(dialog).toContainText('leer, guardar y administrar');await expect(dialog.getByRole('button',{name:'Abrir ajustes de privacidad',exact:true})).toBeVisible();await expect(dialog.getByRole('button',{name:'Comprobar acceso',exact:true})).toBeVisible();await expect(dialog).toContainText('AIDEX necesita permiso para acceder a la carpeta Biblioteca.');await expect(dialog).not.toContainText('/Users/Prueba');await expect(dialog.locator('.folder-privacy-help')).toHaveCount(0);
 for(const name of ['Autorizar carpeta','Abrir ajustes de privacidad','Comprobar acceso']){
  const button=dialog.getByRole('button',{name,exact:true});const svg=button.locator('svg');await expect(svg).toBeVisible();await expect(svg.locator('path')).not.toHaveCount(0);const box=await svg.boundingBox();expect(box?.width).toBe(16);expect(box?.height).toBe(16);const appearance=await svg.evaluate(el=>{const style=getComputedStyle(el);return {color:style.stroke,display:style.display}});expect(appearance.color).not.toBe('none');expect(appearance.display).not.toBe('none');
 }
 await page.screenshot({path:'../evidencias/macos-autorizacion-20261030/modal-administrada.png'});
 diagnostic='ROOT_PRIVACY_DENIED';await dialog.getByRole('button',{name:'Comprobar acceso',exact:true}).click();await expect(dialog.locator('.folder-privacy-help')).toContainText('aidex-server');await page.screenshot({path:'../evidencias/macos-autorizacion-20261030/modal-privacidad.png'});
 await dialog.getByRole('button',{name:'Cerrar autorización',exact:true}).click();await page.getByLabel('Origen de la carpeta').selectOption('linked');diagnostic='ROOT_PERMISSION_DENIED';await page.getByRole('button',{name:'Inspeccionar carpeta',exact:true}).click();await expect(dialog).toContainText('sin modificarlos');await page.screenshot({path:'../evidencias/macos-autorizacion-20261030/modal-vinculada.png'});await dialog.getByRole('button',{name:'Cerrar autorización',exact:true}).click();
 await page.unroute('**/api/v1/libraries/*/root-plans');
 let release:()=>void=()=>{};const pending=new Promise<void>(resolve=>release=resolve);let entered:()=>void=()=>{};const received=new Promise<void>(resolve=>entered=resolve);
 await page.route('**/api/v1/libraries/*/root-plans',async route=>{entered();await pending;await route.fulfill({status:201,contentType:'application/json',body:JSON.stringify({id:'stale',server_path:'/Users/Prueba/Biblioteca',storage_source:'managed',relation:'distinct',related_root_ids:[],expected_configuration_revision:1})});});
 await page.getByRole('button',{name:'Inspeccionar carpeta',exact:true}).click();await received;await path.fill('/Users/Prueba/Otra carpeta');release();await expect(page.getByRole('button',{name:'Inspeccionar carpeta',exact:true})).toBeEnabled();await expect(page.getByRole('button',{name:'Registrar destino administrado',exact:true})).toHaveCount(0);await expect(dialog).not.toBeVisible();
});
