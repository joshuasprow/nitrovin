// A small CDP driver over headless Chrome: no dependencies.
import { spawn } from 'node:child_process';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const CHROME = process.env.NITROVIN_CHROME || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';

export async function launch() {
  const dir = mkdtempSync(join(tmpdir(), 'cdp-'));
  const proc = spawn(CHROME, ['--headless=new', '--remote-debugging-port=0', `--user-data-dir=${dir}`, '--no-first-run', '--hide-scrollbars', 'about:blank'], { stdio: ['ignore', 'ignore', 'pipe'] });
  const wsURL = await new Promise((res, rej) => {
    let buf = '';
    proc.stderr.on('data', d => { buf += d; const m = buf.match(/DevTools listening on (ws:\/\/\S+)/); if (m) res(m[1]); });
    setTimeout(() => rej(new Error('chrome did not start: ' + buf)), 15000);
  });
  const port = new URL(wsURL).port;
  const targets = await (await fetch(`http://127.0.0.1:${port}/json`)).json();
  const page = targets.find(t => t.type === 'page');
  const ws = new WebSocket(page.webSocketDebuggerUrl);
  await new Promise(r => ws.addEventListener('open', r));
  let id = 0; const waits = new Map(); const events = [];
  ws.addEventListener('message', e => {
    const m = JSON.parse(e.data);
    if (m.id && waits.has(m.id)) { const { res, rej } = waits.get(m.id); waits.delete(m.id); m.error ? rej(new Error(JSON.stringify(m.error))) : res(m.result); }
    else events.push(m);
  });
  const send = (method, params = {}) => new Promise((res, rej) => { waits.set(++id, { res, rej }); ws.send(JSON.stringify({ id, method, params })); });
  await send('Page.enable'); await send('Runtime.enable');
  const sleep = ms => new Promise(r => setTimeout(r, ms));
  const api = {
    send, sleep,
    async size(w, h = 900, scale = 1) { await send('Emulation.setDeviceMetricsOverride', { width: w, height: h, deviceScaleFactor: scale, mobile: false }); },
    async scheme(s) { await send('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: s }] }); },
    async goto(url, wait = 700) { await send('Page.navigate', { url }); await sleep(wait); for (let i = 0; i < 40; i++) { if (await api.eval('document.readyState') === 'complete') break; await sleep(100); } await sleep(wait); },
    async eval(expression) {
      const r = await send('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true });
      if (r.exceptionDetails) throw new Error(r.exceptionDetails.exception?.description || JSON.stringify(r.exceptionDetails));
      return r.result.value;
    },
    async shot(path, full = true) {
      const r = await send('Page.captureScreenshot', { format: 'png', captureBeyondViewport: full });
      writeFileSync(path, Buffer.from(r.data, 'base64'));
    },
    async type(text) { for (const ch of text) { await send('Input.dispatchKeyEvent', { type: 'keyDown', text: ch, key: ch }); await send('Input.dispatchKeyEvent', { type: 'keyUp', key: ch }); } },
    async close() { ws.close(); proc.kill(); },
  };
  return api;
}
