import { test, expect, type Page } from '@playwright/test';
import { mkdtemp, realpath, rm, readFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { randomUUID } from 'node:crypto';

async function login(page:Page,username='admin-e2e',password='Browser-fixture-password-2026'){await page.goto('/login');await page.getByLabel('Usuario',{exact:true}).fill(username);await page.getByLabel('Contraseña',{exact:true}).fill(password);await page.getByRole('button',{name:'Entrar',exact:true}).click();await expect(page.getByRole('heading',{name:'Mi cuenta',exact:true})).toBeVisible();}
async function post(page:Page,path:string,body:unknown,method='POST',revision?:number){const session=await(await page.request.get('/api/v1/auth/session')).json();const response=await page.request.fetch(`/api/v1${path}`,{method,headers:{Origin:'http://127.0.0.1:8099','X-CSRF-Token':session.csrf_token,'Idempotency-Key':randomUUID(),...(revision?{'If-Match':`"${revision}"`}:{})},data:body});expect(response.ok(),await response.text()).toBeTruthy();return response.status()===204?null:response.json();}

test('incorporar sin clasificación, corregir y sustituir conservando versiones',async({page})=>{
  test.setTimeout(90_000);page.setDefaultTimeout(10_000);await page.setViewportSize({width:1920,height:1080});
  const destination=await realpath(await mkdtemp(join(tmpdir(),'aidex-corrections-browser-')));
  try{
    await login(page);await page.getByRole('link',{name:'Bibliotecas',exact:true}).click();await page.getByRole('button',{name:'Nueva biblioteca',exact:true}).click();await page.getByLabel('Nombre de la biblioteca',{exact:true}).fill('Correcciones 20261030');await page.getByLabel('Modalidad de biblioteca').selectOption('hybrid');await page.getByRole('button',{name:'Crear biblioteca',exact:true}).click();await expect(page.getByRole('heading',{name:'Correcciones 20261030',exact:true})).toBeVisible();
    const all=await(await page.request.get('/api/v1/libraries')).json();const library=all.items.find((item:{name:string})=>item.name==='Correcciones 20261030');
    const plan=await post(page,`/libraries/${library.id}/root-plans`,{server_path:destination,storage_source:'managed'});await post(page,`/libraries/${library.id}/roots`,{plan_id:plan.id,expected_configuration_revision:plan.expected_configuration_revision});
    await page.reload();await page.getByRole('button',{name:'Abrir biblioteca Correcciones 20261030',exact:true}).click();
    await page.getByRole('tab',{name:'Configuración',exact:true}).click();await expect(page.getByLabel('Exigir clasificación antes de incorporar documentos',{exact:true})).toBeChecked();await page.getByLabel('Exigir clasificación antes de incorporar documentos',{exact:true}).uncheck();await page.getByRole('combobox',{name:'Carpeta administrada',exact:true}).selectOption({label:destination});await page.getByRole('button',{name:'Guardar configuración',exact:true}).click();await expect(page.getByText('Configuración guardada.',{exact:true})).toBeVisible();
    await page.getByRole('tab',{name:'Cargas',exact:true}).click();await page.getByLabel('Archivos',{exact:true}).setInputFiles(resolve('../testdata/documents/native.pdf'));await page.getByRole('button',{name:'Guardar borradores',exact:true}).click();await page.getByRole('button',{name:'Clasificar native.pdf',exact:true}).click();
    const dialog=page.getByRole('dialog');await expect(dialog).toContainText('Borrador temporal privado');await dialog.getByRole('button',{name:'Confirmar documento',exact:true}).click();await expect(dialog.getByText('Guardado definitivo completado',{exact:true})).toBeVisible({timeout:25000});await expect(dialog).toContainText('Incorporado sin clasificar');
    const original=(await(await page.request.get(`/api/v1/libraries/${library.id}/explorer?classification_status=unclassified`)).json()).items[0];const bytes=await readFile(original.original_path);
    await dialog.getByText('Corregir clasificación',{exact:true}).click();await dialog.getByLabel('Título descriptivo',{exact:true}).fill('Título corregido');await dialog.getByLabel('Motivo de la corrección o reasignación',{exact:true}).fill('Corrección de captura');await dialog.getByRole('button',{name:'Guardar clasificación',exact:true}).click();await expect(dialog.getByRole('heading',{name:'Título corregido',exact:true})).toBeVisible();await expect(dialog).toContainText('Aprobado');expect(await readFile(original.original_path)).toEqual(bytes);
    await dialog.getByText('Sustituir documento',{exact:true}).click();await dialog.getByLabel('Archivo correcto',{exact:true}).setInputFiles({name:'Archivo correcto.pdf',mimeType:'application/pdf',buffer:bytes});await dialog.getByLabel('Motivo de sustitución',{exact:true}).fill('Se recibió otra versión del original');await dialog.getByRole('button',{name:'Cargar nueva versión',exact:true}).click();await expect(dialog.getByRole('button',{name:'Confirmar documento',exact:true})).toBeVisible();await dialog.getByRole('button',{name:'Confirmar documento',exact:true}).click();await expect(dialog.getByText('Guardado definitivo completado',{exact:true})).toBeVisible({timeout:25000});expect(await readFile(original.original_path)).toEqual(bytes);
    await dialog.getByText('Versiones conservadas',{exact:true}).click();await expect(dialog.getByRole('button',{name:'Consultar versión',exact:true})).toHaveCount(2);
    const versions=await(await page.request.get(`/api/v1/documents/${original.id}/versions`)).json();expect(versions.items).toHaveLength(2);
    await dialog.getByRole('button',{name:'Cerrar ficha',exact:true}).click();await page.getByRole('tab',{name:'Documentos',exact:true}).click();await page.getByRole('combobox',{name:'Clasificación',exact:true}).selectOption('unclassified');await expect(page.locator('.explorer-details tbody tr')).toHaveCount(1);await page.getByRole('combobox',{name:'Clasificación',exact:true}).selectOption('historical');await expect(page.locator('.explorer-details tbody tr')).toHaveCount(1);await expect(page.locator('.explorer-details')).toContainText('Versión histórica conservada');
  }finally{await rm(destination,{recursive:true,force:true});}
});


test('el Colaborador presenta explícitamente el borrador sin clasificar al Gestor',async({page,browser})=>{
  test.setTimeout(60000);page.setDefaultTimeout(10000);
  const destination=await realpath(await mkdtemp(join(tmpdir(),'aidex-private-present-')));const author=await browser.newPage();
  try{
    await login(page);
    const library=await post(page,'/libraries',{name:'Presentación privada',description:'Prueba sintética',mode:'hybrid'});
    const plan=await post(page,`/libraries/${library.id}/root-plans`,{server_path:destination,storage_source:'managed'});
    const root=await post(page,`/libraries/${library.id}/roots`,{plan_id:plan.id,expected_configuration_revision:plan.expected_configuration_revision});
    const current=await(await page.request.get(`/api/v1/libraries/${library.id}`)).json();
    await post(page,`/libraries/${library.id}`,{name:current.name,description:current.description,ocr_languages:current.ocr_languages,mode:current.mode,settings:{...current.settings,require_classification:false,default_managed_root_id:root.root_id}},'PATCH',current.revision);
    const user=await post(page,'/users',{username:'private-present-author',display_name:'Colaborador prueba',password:'abcdef'});
    await post(page,`/libraries/${library.id}/members/${user.id}`,{role_ids:['library_contributor']},'PUT');
    await login(author,'private-present-author','abcdef');await author.getByRole('link',{name:'Bibliotecas',exact:true}).click();await author.getByRole('button',{name:'Abrir biblioteca Presentación privada',exact:true}).click();await author.getByRole('tab',{name:'Cargas',exact:true}).click();await author.getByLabel('Archivos',{exact:true}).setInputFiles(resolve('../testdata/documents/native.pdf'));await author.getByRole('button',{name:'Guardar borradores',exact:true}).click();await author.getByRole('button',{name:'Clasificar native.pdf',exact:true}).click();
    const own=(await(await author.request.get(`/api/v1/libraries/${library.id}/explorer?classification_status=temporary`)).json()).items[0];
    expect((await page.request.get(`/api/v1/documents/${own.id}`)).status()).toBe(404);
    const dialog=author.getByRole('dialog');await expect(dialog.getByRole('button',{name:'Confirmar documento',exact:true})).toHaveCount(0);await dialog.getByRole('button',{name:'Solicitar incorporación',exact:true}).click();await expect(dialog).toContainText('El autor solicitó la incorporación.');
    const shared=await(await page.request.get(`/api/v1/documents/${own.id}`)).json();expect(shared.approval_status).toBe('draft');
    await page.getByRole('link',{name:'Bibliotecas',exact:true}).click();await page.getByRole('button',{name:'Abrir biblioteca Presentación privada',exact:true}).click();await page.getByRole('button',{name:'native.pdf',exact:true}).click();const manager=page.getByRole('dialog');await manager.getByRole('button',{name:'Confirmar documento',exact:true}).click();await expect(manager.getByText('Guardado definitivo completado',{exact:true})).toBeVisible({timeout:25000});await expect(manager).toContainText('Incorporado sin clasificar');
  }finally{await author.close();await rm(destination,{recursive:true,force:true});}
});
