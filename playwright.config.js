const { defineConfig } = require('@playwright/test');
module.exports = defineConfig({
  testDir: './tests',
  timeout: 30000,
  fullyParallel: true,
  workers: 2,
  use: { baseURL: 'http://127.0.0.1:4321', serviceWorkers: 'block', screenshot: 'only-on-failure', trace: 'retain-on-failure' },
  projects: [
    { name: 'webkit-phone', use: { browserName: 'webkit', viewport: { width: 402, height: 874 }, deviceScaleFactor: 3, isMobile: true, hasTouch: true } },
    { name: 'chromium-desktop', use: { browserName: 'chromium', viewport: { width: 1440, height: 1000 } } }
  ],
  webServer: { command: 'node tests/server.cjs', url: 'http://127.0.0.1:4321', reuseExistingServer: !process.env.CI }
});
