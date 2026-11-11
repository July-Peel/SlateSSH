const { test, expect } = require('@playwright/test');

function instruction(...values) {
  return `${values.map(value => `${String(value).length}.${value}`).join(',')};`;
}

function parseInstruction(raw) {
  const values = [];
  let offset = 0;
  while (offset < raw.length) {
    const dot = raw.indexOf('.', offset);
    const length = Number(raw.slice(offset, dot));
    const start = dot + 1;
    values.push(raw.slice(start, start + length));
    offset = start + length + 1;
    if (raw[start + length] === ';') break;
  }
  return values;
}

test('RDP virtual drive exposes a browsable filesystem', async ({ page }) => {
  const uploadedPaths = [];
  await page.route('**/api/**', route => {
    const url = route.request().url();
    let data = {};
    if (url.endsWith('needs-setup')) data = { needsSetup: false };
    else if (url.endsWith('/status')) data = { isAuthenticated: true, user: { username: 'test' } };
    else if (url.endsWith('/connections')) data = [{ id: 1, name: 'Windows desktop', host: 'windows', port: 3389, username: 'test', type: 'RDP' }];
    return route.fulfill({ json: data });
  });

  await page.routeWebSocket('**/rdp/*/tunnel?*', ws => {
    ws.onMessage(raw => {
      const fields = parseInstruction(raw);
      if (fields[0] === 'put') {
        uploadedPaths.push(fields[4]);
        ws.send(instruction('ack', fields[2], 'OK', '0'));
        return;
      }
      if (fields[0] === 'blob') {
        ws.send(instruction('ack', fields[1], 'OK', '0'));
        return;
      }
      if (fields[0] !== 'get') return;
      const json = JSON.stringify({ '/hello.txt': 'text/plain' });
      const blob = Buffer.from(json).toString('base64');
      ws.send(instruction('body', '0', '1', 'application/vnd.glyptodon.guacamole.stream-index+json', '/'));
      ws.send(instruction('blob', '1', blob));
      ws.send(instruction('end', '1'));
    });
    ws.send('0.,8.$test-id;');
    setTimeout(() => ws.send(instruction('filesystem', '0', 'SlateSSH')), 300);
  });

  await page.goto('/');
  await page.locator('.launcher-row .primary').click();
  await expect(page.locator('.rdp-file-panel')).toBeVisible();
  await expect(page.locator('.rdp-file-panel').getByText('hello.txt', { exact: true })).toBeVisible();
  await page.locator('input[type="file"]').setInputFiles({ name: 'upload.txt', mimeType: 'text/plain', buffer: Buffer.from('hello') });
  await expect.poll(() => uploadedPaths).toContain('/upload.txt');
});
