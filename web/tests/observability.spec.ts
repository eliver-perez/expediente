import { test, expect, type Page } from '@playwright/test';
import { mkdtemp, writeFile, utimes, realpath, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
async function request(page:Page,method:string,path:string,data:unknown,revision?:number){
 const session=await(await page.request.get('/api/v1/auth/session')).json();
 const response=await page.request.fetch(`/api/v1${path}`,{method,headers:{Origin:'http://127.0.0.1:8099','X-CSRF-Token':session.csrf_token,...(revision?{'If-Match':`"${revision}"`}:{})},data});
 expect(response.ok(),await response.text()).toBeTruthy();return response.status()===204?null:response.json();
}
test('fase 7: indicadores, error real, diagnóstico, revisión, retención y pantallas adaptables',async({page})=>{
 test.setTimeout(120_000);
 const directory=await realpath(await mkdtemp(join(tmpdir(),'aibid-phase7-')));
 const pdf=join(directory,'privado-fase7.pdf');await writeFile(pdf,'%PDF-1.4\nContenido de prueba inválido SECRET-F7\n%%EOF');const past=new Date(Date.now()-5000);await utimes(pdf,past,past);
 try {
 await page.goto('/login');await page.getByLabel('Usuario',{exact:true}).fill('admin-e2e');await page.getByLabel('Contraseña',{exact:true}).fill('Browser-fixture-password-2026');await page.getByRole('button',{name:'Entrar',exact:true}).click();
 await page.getByRole('link',{name:'Bibliotecas',exact:true}).click();await page.getByRole('button',{name:'Nueva biblioteca'}).click();await page.getByLabel('Nombre de la biblioteca').fill('Observabilidad fase 7');await page.getByLabel('Modalidad de biblioteca').selectOption('linked');await page.getByLabel('Asignarme como gestor de esta biblioteca').check();await page.getByRole('button',{name:'Crear biblioteca',exact:true}).click();
 const libraries=await(await page.request.get('/api/v1/libraries')).json();const library=libraries.items.find((item:{name:string})=>item.name==='Observabilidad fase 7');
 await page.getByRole('tab',{name:'Carpetas',exact:true}).click();await page.getByLabel('Ruta de la carpeta en el servidor').fill(directory);await page.getByRole('button',{name:'Inspeccionar carpeta'}).click();await expect(page.getByText('Carpeta disponible para registrar e indexar.')).toBeVisible();await page.getByRole('button',{name:'Registrar e indexar'}).click();await expect(page.getByText('Raíz registrada. El escaneo está en cola.')).toBeVisible();
 type Event={id:string;context:{library_id?:string};code:string};
 let event:Event|undefined;
 await expect.poll(async()=>{const response=await(await page.request.get('/api/v1/system/errors?module=processing')).json();event=response.items.find((entry:Event)=>entry.context.library_id===library.id);return Boolean(event)},{timeout:30000}).toBe(true);
 await page.getByRole('link',{name:'Dashboard',exact:true}).click();await expect(page.getByRole('heading',{name:'Dashboard',exact:true})).toBeVisible();
 const metrics=await(await page.request.get('/api/v1/system/dashboard')).json();await expect(page.locator('.metric').filter({hasText:'Total de documentos'}).locator('strong')).toHaveText(String(metrics.total));await expect(page.getByRole('table',{name:'Documentos por biblioteca'}).getByRole('row').filter({hasText:library.name})).toContainText('1');
 await page.setViewportSize({width:1800,height:1000});await page.screenshot({path:'test-results/phase7-dashboard-desktop.png',fullPage:true});
 await page.getByRole('link',{name:'Consultar errores del sistema'}).click();await expect(page.getByRole('heading',{name:'Errores y diagnósticos'})).toBeVisible();
 await page.getByRole('combobox',{name:'Módulo',exact:true}).selectOption('processing');await page.getByRole('combobox',{name:'Severidad',exact:true}).selectOption('error');await page.getByRole('button',{name:'Aplicar filtros',exact:true}).click();
 const row=page.getByRole('table',{name:'Errores del sistema'}).getByRole('row').filter({hasText:event!.id});await expect(row).toBeAttached();await row.getByRole('button',{name:'Copiar información de diagnóstico'}).click();
 const diagnostic=page.getByLabel('Diagnóstico para copiar');await expect(diagnostic).toBeVisible();const support=await diagnostic.inputValue();expect(support).toContain(event!.code);expect(support).toContain(event!.id);expect(support).toContain('2.0.0-alpha.8');for(const secret of ['SECRET-F7','privado-fase7.pdf','Browser-fixture-password-2026'])expect(support).not.toContain(secret);
 await row.getByRole('button',{name:'Marcar revisado',exact:true}).click();await expect(row.getByRole('button',{name:'Marcar pendiente',exact:true})).toBeVisible();await row.getByRole('button',{name:'Marcar pendiente',exact:true}).click();await expect(row.getByRole('button',{name:'Marcar revisado',exact:true})).toBeVisible();
 await page.getByText('Retención de diagnósticos',{exact:true}).click();await page.getByLabel('Días de conservación',{exact:true}).fill('31');await page.getByRole('button',{name:'Guardar retención',exact:true}).click();await expect(page.getByText(/^Retención guardada/)).toBeVisible();await page.reload();await page.getByText('Retención de diagnósticos',{exact:true}).click();await expect(page.getByLabel('Días de conservación',{exact:true})).toHaveValue('31');await page.getByLabel('Días de conservación',{exact:true}).fill('30');await page.getByRole('button',{name:'Guardar retención',exact:true}).click();await expect(page.getByText(/^Retención guardada/)).toBeVisible();
 await page.screenshot({path:'test-results/phase7-errors-desktop.png',fullPage:true});await page.setViewportSize({width:390,height:844});await page.screenshot({path:'test-results/phase7-errors-mobile.png',fullPage:true});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth+1)).toBe(true);
 await page.getByLabel('Desde',{exact:true}).fill('2000-01-01');await page.getByLabel('Hasta',{exact:true}).fill('2000-01-02');await page.getByRole('button',{name:'Aplicar filtros',exact:true}).click();await expect(page.getByText('No hay diagnósticos para estos filtros.')).toBeVisible();
 await page.goto('/admin/dashboard');await expect(page.getByRole('heading',{name:'Dashboard',exact:true})).toBeVisible();expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth+1)).toBe(true);await page.screenshot({path:'test-results/phase7-dashboard-mobile.png',fullPage:true});
 const documents=await(await page.request.get(`/api/v1/libraries/${library.id}/explorer`)).json();for(const item of documents.items){const document=await(await page.request.get(`/api/v1/documents/${item.id}`)).json();await request(page,'POST',`/documents/${item.id}/remove-index`,{reason:'Fin de fixture fase 7',expected_case_id:document.case_id},document.revision)}
 } finally { await rm(directory,{recursive:true,force:true}); }
});
