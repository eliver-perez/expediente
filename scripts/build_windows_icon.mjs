// Rebuild only when the brand asset changes. Uses the project's browser tooling.
import { chromium } from '../web/node_modules/playwright-core/index.mjs';
import { readFile, mkdir, rm } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { tmpdir } from 'node:os';
const root = dirname(dirname(fileURLToPath(import.meta.url)));
const temporary = join(tmpdir(), `aibid-icon-${process.pid}`);
await mkdir(temporary);
const browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || 'chrome', headless: true });
try {
  const svg = await readFile(join(root, 'packaging/windows/aibid-icon.svg'), 'utf8');
  const sizes = [16, 24, 32, 40, 48, 64, 128, 256];
  const images = [];
  for (const size of sizes) {
    const page = await browser.newPage({ viewport: { width: size, height: size }, deviceScaleFactor: 1 });
    await page.setContent(`<style>html,body{margin:0;background:transparent}svg{display:block;width:100vw;height:100vh}</style>${svg}`);
    const path = join(temporary, `${size}.png`); await page.screenshot({ path, omitBackground: true }); images.push(path); await page.close();
  }
  execFileSync('magick', [...images, join(root, 'packaging/windows/aibid.ico')]);
  for (const command of ['aibid-launcher', 'gestor-documental']) {
    execFileSync('go', ['run', 'github.com/akavel/rsrc@v0.10.2', '-ico', 'packaging/windows/aibid.ico', '-arch', 'amd64', '-o', `cmd/${command}/resource_windows_amd64.syso`], { cwd: root, stdio: 'inherit' });
  }
} finally { await browser.close(); await rm(temporary, { recursive: true, force: true }); }
