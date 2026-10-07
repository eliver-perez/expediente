import { test, expect } from '@playwright/test';

test('fase 8: Dashboard con gráficas locales, Ajustes agrupados y contraste', async ({ page }) => {
  const pageErrors:string[]=[];const external:string[]=[];
  page.on('pageerror', error=>pageErrors.push(error.message));
  page.on('request', request=>{if(!request.url().startsWith('http://127.0.0.1:8099')&&!request.url().startsWith('data:'))external.push(request.url())});
  await page.goto('/login');await page.getByLabel('Usuario',{exact:true}).fill('admin-e2e');await page.getByLabel('Contraseña',{exact:true}).fill('Browser-fixture-password-2026');await page.getByRole('button',{name:'Entrar',exact:true}).click();
  const main=page.getByRole('navigation',{name:'Principal',exact:true});
  await expect(main.getByRole('link',{name:'Ajustes',exact:true})).toHaveCount(1);
  for(const name of ['Panel general','Configuración avanzada','Archivos y procesamiento','Vistas previas','Procesamiento','Acceso y red'])await expect(main.getByRole('link',{name,exact:true})).toHaveCount(0);
  await main.getByRole('link',{name:'Dashboard',exact:true}).click();await expect(page.getByRole('heading',{name:'Dashboard',exact:true})).toBeVisible();
  const metrics=await(await page.request.get('/api/v1/system/dashboard')).json();
  expect(metrics.daily).toHaveLength(30);expect(metrics.daily.at(-1).count).toBe(metrics.today);
  const trend=page.getByRole('img',{name:/^Incorporaciones diarias\./});await expect(trend).toBeVisible();
  await expect.poll(()=>trend.evaluate(canvas=>(canvas as HTMLCanvasElement).width)).toBeGreaterThan(0);
  await page.getByText('Ver datos de incorporaciones diarias',{exact:true}).click();await expect(page.getByRole('table',{name:'Incorporaciones diarias',exact:true}).getByRole('row')).toHaveCount(31);await page.getByText('Ver datos de incorporaciones diarias',{exact:true}).click();
  await page.setViewportSize({width:1800,height:1100});await page.screenshot({path:'test-results/phase8-dashboard-desktop.png',fullPage:true});
  await main.getByRole('link',{name:'Ajustes',exact:true}).click();await expect(page.getByRole('heading',{name:'Ajustes',exact:true})).toBeVisible();
  const sections=page.getByRole('navigation',{name:'Secciones de configuración'});
  const entries=[['Archivos','/admin/files'],['Procesamiento y recursos','/admin/processing'],['Automatización, OCR y vigilancia','/admin/advanced'],['Vistas previas y caché','/admin/previews'],['Acceso y red','/admin/network']];
  for(const [label,path] of entries){
    await sections.getByRole('link',{name:label,exact:true}).click();await expect(page).toHaveURL(new RegExp(`${path}$`));await expect(sections.getByRole('link',{name:label,exact:true})).toHaveAttribute('aria-current','page');await expect(main.getByRole('link',{name:'Ajustes',exact:true})).toHaveAttribute('aria-current','page');
    await expect(page.getByRole('heading',{level:1})).toHaveText('Ajustes');
  }
  // Existing deep links stay valid, including reload and history navigation.
  await page.reload();await expect(page.getByRole('heading',{name:'Acceso y red',exact:true})).toBeVisible();
  await sections.getByRole('link',{name:'Archivos',exact:true}).focus();await page.keyboard.press('Enter');await expect(page).toHaveURL(/\/admin\/files$/);await page.goBack();await expect(page).toHaveURL(/\/admin\/network$/);
  const contrast=await sections.evaluate(nav=>{
    function luminance(color:string){const rgb=(color.match(/[\d.]+/g)||[]).slice(0,3).map(Number).map(value=>{const c=value/255;return c<=.04045?c/12.92:((c+.055)/1.055)**2.4});return rgb[0]*.2126+rgb[1]*.7152+rgb[2]*.0722}
    return Array.from(nav.querySelectorAll('a')).map(anchor=>{const style=getComputedStyle(anchor);const bg=style.backgroundColor==='rgba(0, 0, 0, 0)'?getComputedStyle(document.body).backgroundColor:style.backgroundColor;const a=luminance(style.color),b=luminance(bg);return(Math.max(a,b)+.05)/(Math.min(a,b)+.05)});
  });expect(Math.min(...contrast)).toBeGreaterThanOrEqual(4.5);
  await page.screenshot({path:'test-results/phase8-settings-desktop.png',fullPage:true});
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.screenshot({path:'test-results/phase8-settings-mobile.png',fullPage:true});
  await main.getByRole('link',{name:'Dashboard',exact:true}).click();await expect(trend).toBeVisible();expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.screenshot({path:'test-results/phase8-dashboard-mobile.png',fullPage:true});
  expect(pageErrors).toEqual([]);expect(external).toEqual([]);
});
