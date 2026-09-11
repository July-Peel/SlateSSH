// Isolated static fixture server: no real credentials, database, or SSH hosts are used.
const http = require('node:http');
const fs = require('node:fs');
const path = require('node:path');
const root = path.resolve(__dirname, '../frontend');
const types = { '.html': 'text/html', '.js': 'application/javascript', '.css': 'text/css', '.json': 'application/json', '.webmanifest': 'application/manifest+json', '.png': 'image/png', '.svg': 'image/svg+xml', '.woff2': 'font/woff2' };
http.createServer((req, res) => {
  const url = new URL(req.url, 'http://localhost');
  if (url.pathname.startsWith('/api/')) {
    res.setHeader('Content-Type', 'application/json');
    const payload = url.pathname.endsWith('/needs-setup') ? { needsSetup: false } : url.pathname.endsWith('/status') ? { isAuthenticated: false } : {};
    res.end(JSON.stringify(payload));
    return;
  }
  const file = path.resolve(root, '.' + (url.pathname === '/' ? '/index.html' : decodeURIComponent(url.pathname)));
  if (!file.startsWith(root + path.sep)) { res.writeHead(403).end(); return; }
  fs.readFile(file, (error, data) => {
    if (error) { res.writeHead(404).end(); return; }
    res.setHeader('Content-Type', types[path.extname(file)] || 'application/octet-stream');
    res.setHeader('Cache-Control', 'no-cache');
    res.end(data);
  });
}).listen(4321, '127.0.0.1');
