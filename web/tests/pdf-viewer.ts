import { expect, type Page } from '@playwright/test';

export async function expectPdfLoaded(page: Page) {
  const preview = page.locator('iframe[title="Vista previa del PDF"]');
  await expect(preview).toHaveAttribute('src', /\/api\/v1\/documents\/[^/]+\/content/);
  if (process.platform === 'win32') {
    // Headless Edge does not expose Chrome's PDF extension or loadProgress_.
    // Check the embedded navigation and authenticated PDF response instead.
    const source = await preview.getAttribute('src');
    const url = new URL(source!, page.url());
    const response = await page.request.get(url.origin + url.pathname);
    expect(response.status()).toBe(200);
    expect(response.headers()['content-type']).toContain('application/pdf');
    await expect.poll(() => page.frames().some(frame => frame.url().startsWith(url.origin + url.pathname))).toBe(true);
    return;
  }
  await expect.poll(async () => {
    const viewer = page.frames().find(frame => frame.url().startsWith('chrome-extension://'));
    if (!viewer) return 0;
    return viewer.locator('pdf-viewer').evaluate(element => (element as HTMLElement & { loadProgress_: number }).loadProgress_).catch(() => 0);
  }).toBe(100);
}
