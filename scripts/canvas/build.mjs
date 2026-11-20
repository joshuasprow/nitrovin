// Builds every frame of the design canvas, light and dark, into
// docs/design/canvas/project, and a plain-HTML preview and a screenshot of
// each into build/canvas. Run it through scripts/canvas.sh.
import { writeFileSync, readFileSync, mkdirSync, copyFileSync } from 'node:fs';
import { open, buildFrame, dcFile, previewFile, REPO } from './engine.mjs';
import { frames } from './frames.mjs';

const only = process.argv.slice(2);
const OUT = `${REPO}/docs/design/canvas/project`, TMP = `${REPO}/build/canvas`;
mkdirSync(TMP, { recursive: true });
for (const f of ['brushed.svg', 'paper.svg']) copyFileSync(`${REPO}/pkg/ui/static/${f}`, `${TMP}/${f}`);
const c = await open();
for (const frame of frames) {
  if (only.length && !only.includes(frame.file)) continue;
  const pair = {};
  for (const scheme of ['light', 'dark']) pair[scheme] = await buildFrame(c, frame, scheme);
  if (!frame.h) { frame.h = frame.measured; for (const k in pair) pair[k].root = pair[k].root.replace('height: 4000px', `height: ${frame.h}px`); }
  for (const scheme of ['light', 'dark']) {
    const built = pair[scheme];
    const name = frame.file + (scheme === 'dark' ? '-Dark' : '');
    writeFileSync(`${OUT}/${name}.dc.html`, dcFile(frame, scheme, built));
    writeFileSync(`${TMP}/${name}.html`, previewFile(frame, scheme, built));
    await c.size(frame.w, 800); await c.scheme(scheme);
    await c.goto(`file://${TMP}/${name}.html`, 300);
    await c.shot(`${TMP}/${name}.png`);
    console.log(name, built.stats, `${frame.w} x ${frame.h}`);
  }
}
await c.close();

// The index: each built frame's height, and the dark row under the tallest
// light frame. Positions across the row and everything else are the
// canvas's own and stay as read.
const path = `${OUT}/canvas.json`, idx = JSON.parse(readFileSync(path, 'utf8'));
for (const f of frames) if (f.h) for (const k of [`${f.file}.dc.html`, `${f.file}-Dark.dc.html`]) if (idx.boards[k]) idx.boards[k].h = f.h;
const light = Object.entries(idx.boards).filter(([k]) => !k.includes('-Dark'));
const darkY = Math.max(...light.map(([, b]) => b.y + b.h)) + 120;
for (const [k, b] of Object.entries(idx.boards)) if (k.includes('-Dark')) b.y = darkY;
writeFileSync(path, JSON.stringify(idx, null, 2) + '\n');
