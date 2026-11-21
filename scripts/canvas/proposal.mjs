// PROTOTYPE: the Proposals page's frames for the shelf list in new
// materials, and the swatch boards of the main page. Throwaway, with the
// prototype it captures (pkg/ui/static/prototype.js).
//
//   NITROVIN_CANVAS_DEMO=http://127.0.0.1:8811 node scripts/canvas/proposal.mjs
//
// The server is this branch's, with the demo set. It writes the frames and
// boards into docs/design/canvas/project and places them in canvas.json.
import { writeFileSync, readFileSync, mkdirSync, copyFileSync } from 'node:fs';
import { open, buildFrame, dcFile, previewFile, REPO } from './engine.mjs';

const OUT = `${REPO}/docs/design/canvas/project`, TMP = `${REPO}/build/canvas`;
mkdirSync(TMP, { recursive: true });
for (const f of ['brushed.svg', 'paper.svg']) copyFileSync(`${REPO}/pkg/ui/static/${f}`, `${TMP}/${f}`);
const VIN = '/?from.vin=1FTEW1EG7FFA12345&part.id=alternator&fitment.number=DEMO-1001&variant=';
const PART = '/?from.fitment.type=alternator&from.fitment.number=DEMO-1001&fitment.number=DEMO-1001&variant=';
const VARIANTS = [
  { key: 'receipt', file: 'Receipt', name: 'thermal receipt', intro: 'Thermal paper. One strip the tag\'s width, fed out from under the tag\'s stock strip: shelf over yard on the one tape, torn at the foot. The print is the printer\'s: one face, double-height heads, rules of dashes, a field\'s name led by dots to its value, a barcode at the foot.' },
  { key: 'placard', file: 'Placard', name: 'tire placard', intro: 'The tire and loading placard of a door jamb: a white label, black bands and rules, one yellow. A ruled table, shelf beside yard, each row a line of cells: the id on yellow, a field\'s name small over its value.' },
  { key: 'tyvek', file: 'Tyvek', name: 'Tyvek wire tags', intro: 'Wire-on tags of spunbonded olefin, yellow for the shelf and orange for the yard. There is no panel: each row is its own tag, a steel eyelet and a twist of wire at one end, its fields printed as blanks filled in.' },
  { key: 'marker', file: 'Marker', name: 'paint marker on steel', intro: 'Yard paint on steel: a sheet of bare, rusting steel written on with a yellow paint marker and a white tire crayon. The id first and large, the rest beside it, no rule but the hand\'s.' },
];
const frames = VARIANTS.map((v) => ({
  file: `Proposal-Shelf-${v.file}`, title: `Proposal: the shelf list, ${v.name}`, w: 1280,
  heading: `The shelf list: ${v.name}`, intro: v.intro,
  items: [
    { vp: 'wide', url: VIN + v.key, keep: ['#fit .cands', '#fit .rails'], cap: 'Under a candidate', note: 'The first candidate\'s stock strip is open.' },
    { row: [
      { vp: 'tab', url: VIN + v.key, keep: ['#fit .cands', '#fit .rails'], cap: 'Narrow' },
      { vp: 'phone', url: VIN + v.key, keep: ['#fit .cands', '#fit .rails'], cap: 'Phone', note: 'Under its tag.' },
    ] },
    { vp: 'wide', url: PART + v.key, keep: ['#answer'], cap: 'Part-number mode', note: 'Under the part\'s tag, a column.' },
  ],
}));

const c = await open();
const built = [];
for (const frame of frames) {
  const pair = {};
  for (const scheme of ['light', 'dark']) pair[scheme] = await buildFrame(c, frame, scheme);
  frame.h = frame.measured;
  for (const scheme of ['light', 'dark']) {
    const b = pair[scheme]; b.root = b.root.replace('height: 4000px', `height: ${frame.h}px`);
    const name = frame.file + (scheme === 'dark' ? '-Dark' : '');
    writeFileSync(`${OUT}/${name}.dc.html`, dcFile(frame, scheme, b));
    writeFileSync(`${TMP}/${name}.html`, previewFile(frame, scheme, b));
    await c.size(frame.w, 800); await c.scheme(scheme);
    await c.goto(`file://${TMP}/${name}.html`, 300);
    await c.shot(`${TMP}/${name}.png`);
    console.log(name, b.stats, `${frame.w} x ${frame.h}`);
  }
  built.push(frame);
}

// ---- the swatch boards: one per research file ----------------------------
const esc = (s) => String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/\{\{/g, '{&#8203;{');
const USED = { 'Fresh direct-thermal receipt paper': 'receipt', 'Tire and loading placard': 'placard', 'Yellow Tyvek wire tag': 'Tyvek tags', 'Yellow paint marker on steel': 'marker' };
const BOARDS = [
  { file: 'Swatches-Thermal', src: 'thermal', title: 'Swatches: thermal paper', intro: 'Direct-thermal paper and its print, as a counter\'s receipts and a yard\'s part tags come off the printer.' },
  { file: 'Swatches-Yard', src: 'yard', title: 'Swatches: the yard and the shelf', intro: 'What parts and vehicles are tagged, labelled and marked with in a yard, a warehouse and a shop.' },
  { file: 'Swatches-Vehicle', src: 'vehicle', title: 'Swatches: the vehicle and the garage', intro: 'Surfaces of the vehicle itself, its labels and instruments, and of the garage around it.' },
];
const hex = (h) => /^#[0-9a-fA-F]{3,8}$/.test(h || '') ? h : null;
const swatches = [];
for (const b of BOARDS) {
  const list = JSON.parse(readFileSync(`${REPO}/scripts/canvas/swatches/${b.src}.json`, 'utf8'));
  let css = '', cards = '';
  list.forEach((s, i) => {
    const base = hex(s.base) || '#cccccc', ink = hex(s.ink) || '#111111', accent = hex(s.accent);
    let bg = String(s.css || '').trim().replace(/^background(-image)?\s*:/, '').replace(/;\s*$/, '').trim();
    if (!bg || /url\(|[<>{}";]/.test(bg)) bg = base;
    css += `.sw-${i} .chip{background:${bg};color:${ink}}\n`;
    const dot = (h, l) => h ? `<span class="hx"><i style="background: ${h}"></i>${l} ${h.toLowerCase()}</span>` : '';
    const hosts = [...new Set((s.sources || []).map((u) => { try { return new URL(u).hostname.replace(/^www\./, ''); } catch (_) { return null; } }).filter(Boolean))];
    cards += `<article class="sw sw-${i}">
<div class="chip"><b>DEMO-S-01</b><span>demo-north · grade A</span>${accent ? `<i style="background: ${accent}"></i>` : ''}</div>
<h3>${esc(s.name)}${USED[s.name] ? ` <em>drawn: ${USED[s.name]}</em>` : ''}</h3>
<p class="fam">${esc(s.family)}</p>
<p>${esc(s.where_seen)}</p>
<p class="tx">${esc(s.texture)}</p>
<p class="hxs">${dot(base, 'base')}${dot(ink, 'ink')}${dot(accent, 'accent')}</p>
<p class="src">${hosts.map(esc).join(' · ')}</p>
</article>\n`;
  });
  const w = 1280;
  const style = `body{margin:0}
.swb{box-sizing:border-box;width:${w}px;padding:40px;display:flex;flex-direction:column;gap:22px;font:14px/1.45 system-ui,sans-serif;color:#14181d;background:#e6ebef}
.swb h1{margin:0;font:900 21px/1.2 "Arial Black","Helvetica Neue",system-ui,sans-serif;letter-spacing:.04em;text-transform:uppercase}
.swb .intro{margin:6px 0 0;max-width:820px;color:#47515c}
.swb .grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:18px}
.sw{display:flex;flex-direction:column;gap:5px;padding:0 0 12px;background:#fff;border:1px solid #9fb3c2;border-radius:4px;overflow:hidden}
.sw .chip{position:relative;display:flex;flex-direction:column;justify-content:center;gap:2px;height:104px;padding:0 16px 0 34px;border-bottom:1px solid #9fb3c2}
.sw .chip b{font:700 17px/1.2 ui-monospace,monospace;letter-spacing:.03em}
.sw .chip span{font-size:12px}
.sw .chip i{position:absolute;right:0;top:0;bottom:0;width:10px}
.sw h3,.sw p{margin:0;padding:0 12px}
.sw h3{margin-top:6px;font:700 14px/1.3 system-ui,sans-serif}
.sw h3 em{display:inline-block;padding:0 6px;font:700 10px/1.7 system-ui,sans-serif;letter-spacing:.06em;text-transform:uppercase;color:#fff;background:#0b4fa8;border-radius:2px;vertical-align:1px}
.sw p{font-size:12px;color:#14181d}
.sw .fam{font:700 10px/1.4 system-ui,sans-serif;letter-spacing:.08em;text-transform:uppercase;color:#47515c}
.sw .tx,.sw .src{color:#47515c}
.sw .hxs{display:flex;flex-wrap:wrap;gap:2px 10px;margin-top:2px;font:11px/1.5 ui-monospace,monospace}
.sw .hx i{display:inline-block;width:9px;height:9px;margin-right:4px;border:1px solid #47515c;vertical-align:-1px}
.sw .src{font-size:11px;overflow-wrap:anywhere}
${css}`;
  const body = (h) => `<div class="swb" style="height: ${h}">
<div><h1>${esc(b.title)}</h1><p class="intro">${esc(b.intro)} Found by a web search on 2026-11-20. Dimensions and conventions are the sources'; every colour is an estimate, since no source publishes one, and no recipe here is settled.</p></div>
<div class="grid">
${cards}</div>
</div>`;
  await c.size(w, 800); await c.scheme('light');
  await c.goto('about:blank', 100);
  await c.eval(`document.head.innerHTML = '<style>' + ${JSON.stringify(style)} + '</style>'; document.body.innerHTML = ${JSON.stringify(body('auto'))}; 1`);
  await c.sleep(200);
  const h = Math.ceil(await c.eval(`document.querySelector('.swb').getBoundingClientRect().height`) / 10) * 10;
  await c.shot(`${TMP}/${b.file}.png`);
  writeFileSync(`${OUT}/${b.file}.dc.html`, `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>${b.title}</title>
<script src="./support.js"></script>
</head>
<body>
<x-dc>
<helmet>
<style>
${style}</style>
</helmet>
${body(h + 'px')}
</x-dc>
<script type="text/x-dc" data-dc-script data-props='{"$preview":{"width":${w},"height":${h}}}'>
class Component extends DCLogic {
renderVals() {
return {};
}
}
</script>
</body>
</html>
`);
  swatches.push({ ...b, w, h });
  console.log(b.file, `${list.length} swatches`, `${w} x ${h}`);
}
await c.close();

// ---- the index -----------------------------------------------------------
const path = `${OUT}/canvas.json`, idx = JSON.parse(readFileSync(path, 'utf8'));
const add = (k, entry) => { idx.boards[k] = { ...(idx.boards[k] || {}), ...entry }; if (!idx.order.includes(k)) idx.order.push(k); };
const note = (k, entry) => { idx.notes[k] = { ...(idx.notes[k] || {}), ...entry }; if (!idx.noteOrder.includes(k)) idx.noteOrder.push(k); };
if (!idx.pages.some((p) => p.id === 'proposals')) idx.pages.push({ id: 'proposals', name: 'Proposals' });
const darkY = Math.max(...built.map((f) => f.h)) + 120;
built.forEach((f, i) => {
  add(`${f.file}.dc.html`, { h: f.h, page: 'proposals', title: `${f.title}, light`, w: f.w, x: i * 1360, y: 0 });
  add(`${f.file}-Dark.dc.html`, { h: f.h, page: 'proposals', title: `${f.title}, dark`, w: f.w, x: i * 1360, y: darkY });
});
note('proposal-shelf-materials', { kind: 'title1', maxW: built.length * 1360 - 80, page: 'proposals', text: 'Proposal: the shelf list in new materials', w: 240, x: 0, y: -300 });
// The swatches sit under the main page's rows, wherever those now end.
const main = Object.entries(idx.boards).filter(([k, b]) => !b.page && !k.startsWith('Swatches-'));
const swY = Math.max(...main.map(([, b]) => b.y + b.h)) + 520;
swatches.forEach((s, i) => add(`${s.file}.dc.html`, { h: s.h, title: s.title, w: s.w, x: i * 1360, y: swY }));
note('swatches', { kind: 'title1', maxW: swatches.length * 1360 - 80, text: 'Swatches: materials found, none settled', w: 240, x: 0, y: swY - 300 });
writeFileSync(path, JSON.stringify(idx, null, 2) + '\n');
