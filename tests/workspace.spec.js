const { test, expect } = require('@playwright/test');

const hosts = Array.from({ length: 12 }, (_, i) => ({ id: i + 1, name: i ? `staging-${i}` : 'production-api-with-a-long-name', host: i ? `10.0.0.${i}` : 'long-hostname.internal.example.com', port: 22, username: 'deploy', type: 'SSH' }));

async function mockApi(page, authenticated = true) {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.route('**/api/**', route => {
    const url = route.request().url();
    let data = {};
    if (url.endsWith('needs-setup')) data = { needsSetup: false };
    else if (url.endsWith('/status')) data = { isAuthenticated: authenticated, user: { username: 'test' } };
    else if (url.endsWith('/connections')) data = hosts;
    else if (url.endsWith('/login')) data = { user: { username: 'test' } };
    return route.fulfill({ json: data });
  });
  return errors;
}

async function mockSsh(page) {
  const inputs = [];
  await page.routeWebSocket('**/ws', ws => {
    ws.onMessage(raw => {
      const message = JSON.parse(raw);
      if (message.type === 'ssh:connect') {
        ws.send(JSON.stringify({ type: 'ssh:connected', sessionId: 'test-session', payload: { connectionId: message.payload.connectionId } }));
        setTimeout(() => ws.send(JSON.stringify({ type: 'ssh:output', sessionId: 'test-session', payload: { data: '\x1b[32mdeploy@production:~$\x1b[0m uname -a\r\nLinux production 6.8.0 x86_64 GNU/Linux\r\n\r\nWelcome to your remote workspace.\r\n\x1b[32mdeploy@production:~$\x1b[0m ' } })), 150);
      }
      if (message.type === 'ssh:input') inputs.push(message.payload.data);
    });
  });
  return inputs;
}

async function openTerminal(page) {
  const errors = await mockApi(page);
  const inputs = await mockSsh(page);
  await page.goto('/');
  await page.locator('.launcher-row .primary').first().click();
  await expect(page.locator('.xterm-screen')).toBeVisible();
  return { errors, inputs };
}

async function assertFits(page, selector, scrollable = false) {
  const result = await page.locator(selector).evaluate(el => {
    const r = el.getBoundingClientRect();
    return { left: r.left, right: r.right, top: r.top, bottom: r.bottom, width: innerWidth, height: visualViewport.height + visualViewport.offsetTop };
  });
  expect(result.left).toBeGreaterThanOrEqual(-1);
  expect(result.right).toBeLessThanOrEqual(result.width + 1);
  if (!scrollable) expect(result.bottom).toBeLessThanOrEqual(result.height + 1);
}

test('login supports password autofill, scrolling and a single initialization', async ({ page }, info) => {
  let bootstrapRequests = 0;
  page.on('request', request => { if (request.url().endsWith('needs-setup')) bootstrapRequests++; });
  const errors = await mockApi(page, false);
  await mockSsh(page);
  await page.goto('/');
  await expect(page.getByLabel('login as:')).toBeVisible();
  await expect(page.getByLabel('password:')).toHaveAttribute('autocomplete', 'current-password');
  if (info.project.name.includes('phone')) expect(await page.evaluate(() => document.activeElement.tagName)).not.toBe('INPUT');
  await page.screenshot({ path: `test-results/${info.project.name}-login.png` });
  await page.getByLabel('login as:').fill('operator');
  await page.getByLabel('password:').fill('fixture-only');
  await page.getByRole('checkbox').check();
  await page.getByRole('button', { name: '[ 连接工作台 → ]', exact: true }).click();
  await expect(page.locator('.launcher-row')).toHaveCount(12);
  expect(bootstrapRequests).toBe(1);
  expect(errors).toEqual([]);
});

test('host list and manager fit phone, landscape, tablet and PC widths', async ({ page }, info) => {
  const errors = await mockApi(page);
  await mockSsh(page);
  await page.goto('/');
  await expect(page.locator('.launcher-row')).toHaveCount(12);
  await page.screenshot({ path: `test-results/${info.project.name}-hosts.png` });
  for (const width of [320, 402, 874, 1024, 1440]) {
    await page.setViewportSize({ width, height: width === 874 ? 402 : 874 });
    await assertFits(page, '.launcher-list-wrap', true);
    await page.locator('.launcher-actions-top').getByText('[ Manager ]', { exact: true }).click();
    await assertFits(page, '.connection-modal');
    const overflow = await page.locator('.connection-layout').evaluate(el => el.scrollWidth > el.clientWidth + 1);
    expect(overflow, `manager overflow at ${width}`).toBe(false);
    await page.keyboard.press('Escape');
  }
  await page.setViewportSize({ width: 402, height: 440 });
  await page.locator('.launcher-actions-top').getByText('[ + New Host ]', { exact: true }).click();
  await page.getByLabel('Host', { exact: true }).fill('192.168.1.2');
  await page.getByRole('button', { name: '[ Save Host ]', exact: true }).scrollIntoViewIfNeeded();
  await expect(page.getByRole('button', { name: '[ Save Host ]', exact: true })).toBeInViewport();
  expect(errors).toEqual([]);
});

test('terminal tools, real control bytes, repeated taps and fullscreen dock', async ({ page }, info) => {
  const { errors, inputs } = await openTerminal(page);
  await page.setViewportSize({ width: 402, height: 874 });
  await expect(page.locator('.terminal-mobile-dock')).toBeVisible();
  await page.locator('[data-key="escape"]').click();
  await page.locator('[data-key="tab"]').click();
  await page.locator('[data-key="tab"]').click();
  await page.locator('[data-key="interrupt"]').click();
  await expect.poll(() => inputs.slice(0, 4)).toEqual(['\x1b', '\t', '\t', '\x03']);
  await page.locator('[data-action="ctrl"]').click();
  await page.locator('.xterm-helper-textarea').press('c');
  await expect.poll(() => inputs.at(-1)).toBe('\x03');
  await expect(page.locator('[data-action="ctrl"]')).toHaveAttribute('aria-pressed', 'false');
  await page.screenshot({ path: `test-results/${info.project.name}-terminal.png` });
  await page.getByRole('button', { name: '打开终端工具' }).click();
  await page.getByRole('button', { name: '全屏终端', exact: true }).click();
  await assertFits(page, '.terminal-mobile-dock');
  await page.getByRole('button', { name: '打开终端工具' }).click();
  await page.getByRole('button', { name: '主机目录', exact: true }).click();
  await expect(page.locator('.mobile-drawer')).toBeVisible();
  await page.getByRole('button', { name: '关闭主机目录', exact: true }).last().click();
  await page.getByTitle('Exit Fullscreen (ESC)').click();
  await page.getByRole('button', { name: '打开终端工具' }).click();
  await page.getByRole('button', { name: '粘贴命令', exact: true }).click();
  await page.getByLabel('待粘贴的命令').fill('echo hello');
  await page.getByRole('button', { name: '[ Send to PTY ]', exact: true }).click();
  await expect.poll(() => inputs.some(input => input.includes('echo hello'))).toBe(true);
  expect(errors).toEqual([]);
});

test('visual viewport keyboard resize preserves dock, modal and terminal dimensions', async ({ page }) => {
  await page.addInitScript(() => {
    window.testViewport = { height: 874, offsetTop: 0, scale: 1 };
    for (const key of Object.keys(window.testViewport)) Object.defineProperty(visualViewport, key, { get: () => window.testViewport[key] });
  });
  await openTerminal(page);
  await page.setViewportSize({ width: 402, height: 874 });
  await page.getByRole('button', { name: '打开软键盘', exact: true }).click();
  await page.evaluate(() => { testViewport.height = 390; testViewport.offsetTop = 30; visualViewport.dispatchEvent(new Event('resize')); });
  await expect(page.locator('#app')).toHaveClass(/keyboard-open/);
  await assertFits(page, '.terminal-mobile-dock');
  const before = await page.locator('.terminal-stage').boundingBox();
  expect(before.height).toBeGreaterThan(120);
  await page.getByRole('button', { name: '打开终端工具' }).click();
  await page.getByRole('button', { name: '远程文件', exact: true }).click();
  await assertFits(page, '.mobile-term-modal:visible');
  await page.keyboard.press('Escape');
  await page.evaluate(() => { testViewport.height = 874; testViewport.offsetTop = 0; visualViewport.dispatchEvent(new Event('resize')); });
  await expect(page.locator('#app')).not.toHaveClass(/keyboard-open/);
  await assertFits(page, '.terminal-mobile-dock');
});

test.describe('installed PWA', () => {
test.use({ serviceWorkers: 'allow' });
test('PWA shell installs, serves navigation and excludes private API data', async ({ page }) => {
  await page.goto('/');
  await page.evaluate(() => navigator.serviceWorker.ready);
  await expect.poll(() => page.evaluate(() => !!navigator.serviceWorker.controller)).toBe(true);
  const cached = await page.evaluate(async () => {
    const entries = await Promise.all((await caches.keys()).map(async key => (await (await caches.open(key)).keys()).map(request => request.url)));
    return entries.flat();
  });
  expect(cached.some(url => url.includes('/api/') || url.endsWith('/ws'))).toBe(false);
  const response = await page.reload();
  expect(response.fromServiceWorker()).toBe(true);
  await expect(page.locator('.update-notice')).not.toBeVisible();
});
test('PWA shell reloads while offline', async ({ page, context, browserName }) => {
  test.skip(browserName === 'webkit', 'Windows Playwright WebKit blocks even controlled fetches with an internal error in offline emulation; verify this on iPhone Safari.');
  await page.goto('/');
  await page.evaluate(() => navigator.serviceWorker.ready);
  await expect.poll(() => page.evaluate(() => !!navigator.serviceWorker.controller)).toBe(true);
  await context.setOffline(true);
  await page.reload();
  await expect(page.locator('.auth-console')).toBeVisible();
  await expect(page.getByText('网络已离线。应用界面仍可打开，连接服务器需要恢复网络。')).toBeVisible();
  await context.setOffline(false);
});
});
