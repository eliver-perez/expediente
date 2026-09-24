import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests', fullyParallel: false, workers: 1, timeout: 60_000,
  use: { baseURL: 'http://127.0.0.1:8099', channel: process.env.PLAYWRIGHT_CHANNEL || 'chrome',
    trace: 'retain-on-failure', screenshot: 'only-on-failure' },
  webServer: {
    command: 'go run -tags e2e,development ./internal/e2eserver', cwd: '..',
    url: 'http://127.0.0.1:8099/health/live', reuseExistingServer: false, timeout: 120_000
  },
  reporter: 'list'
});
