// Builds every frame of the design canvas, light and dark, into
// docs/design/canvas/project, and a plain-HTML preview and a screenshot of
// each into build/canvas. Run it through scripts/canvas.sh.
import { writeFileSync, readFileSync, mkdirSync, copyFileSync } from 'node:fs';
import { open, buildFrame, dcFile, previewFile, REPO } from './engine.mjs';
import { frames } from './frames.mjs';
import { place } from './snapshot.mjs';

// --check writes nothing: it builds each frame and compares it with the
// committed one, and exits 1 naming the frames that differ. What a checkout
// cannot fix is left out of the comparison: a frame's height, which follows
// the machine's fonts; the data month, which go.mod's pin sets; and the
// release's hash.
const check = process.argv.includes('--check');
const only = process.argv.slice(2).filter(a => a !== '--check');
const norm = (s) => s.replace(/height: \d+px; color/, 'height: Hpx; color').replace(/"height":\d+/, '"height":H')
  .replace(/20\d\d_\d\d/g, 'MONTH').replace(/blake3:[0-9a-f]{64}/g, 'blake3:HASH');
const stale = [];
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
    const dc = dcFile(frame, scheme, built);
    if (check) {
      let have = ''; try { have = readFileSync(`${OUT}/${name}.dc.html`, 'utf8'); } catch (_) {}
      const same = norm(have) === norm(dc);
      if (!same) { stale.push(name); writeFileSync(`${TMP}/${name}.dc.html`, dc); }
      console.log(name, same ? 'current' : 'STALE');
      continue;
    }
    writeFileSync(`${OUT}/${name}.dc.html`, dc);
    writeFileSync(`${TMP}/${name}.html`, previewFile(frame, scheme, built));
    await c.size(frame.w, 800); await c.scheme(scheme);
    await c.goto(`file://${TMP}/${name}.html`, 300);
    await c.shot(`${TMP}/${name}.png`);
    console.log(name, built.stats, `${frame.w} x ${frame.h}`);
  }
}
await c.close();
if (check) {
  if (stale.length) {
    console.error(`canvas: ${stale.length} frame(s) differ from the page as built: ${stale.join(', ')}.\nRun scripts/canvas.sh, commit docs/design/canvas, and publish the changed frames (AGENTS.md, "Design Canvas").\nThe frames as built are in build/canvas.`);
    process.exit(1);
  }
  process.exit(0);
}

// The index: each built frame's height, and the dark row under the tallest
// light frame. Positions across the row and everything else are the
// canvas's own and stay as read.
const path = `${OUT}/canvas.json`, idx = JSON.parse(readFileSync(path, 'utf8'));
const heights = Object.fromEntries(frames.filter((f) => f.h).flatMap((f) => [[`${f.file}.dc.html`, f.h], [`${f.file}-Dark.dc.html`, f.h]]));
writeFileSync(path, JSON.stringify(place(idx, heights), null, 2) + '\n');
