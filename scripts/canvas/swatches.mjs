// Builds the canvas's swatch boards, the page "Swatches", in two rows: what
// was made for the page, in use and tried, over materials found for its look,
// none of those settled. One board per file of
// scripts/canvas/swatches/, a card per material: a chip in its colours, where
// it is seen, its texture, its colours and the sites it was found on.
//
//   node scripts/canvas/swatches.mjs
//
// It writes docs/design/canvas/project/Swatches-<Name>.dc.html and places the
// boards on the page "swatches" in canvas.json, changing no other key. It
// needs Chrome, to measure a board's height (NITROVIN_CHROME overrides its
// path), and no server. It does not publish (AGENTS.md, "Design Canvas").
//
// The data files are a web search's findings. Dimensions and conventions are
// their sources'; every colour is an estimate, since no source publishes one.
import { writeFileSync, readFileSync, mkdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { launch } from './cdp.mjs';

const REPO = fileURLToPath(new URL('../..', import.meta.url)).replace(/\/$/, '');
const OUT = `${REPO}/docs/design/canvas/project`, TMP = `${REPO}/build/canvas`;
mkdirSync(TMP, { recursive: true });
const c = await launch();
// ---- the swatch boards: one per research file ----------------------------
const esc = (s) => String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/\{\{/g, '{&#8203;{');
const BOARDS = [
  { file: 'Swatches-Page', src: 'page', row: 0, title: 'Swatches: in use', intro: 'What the page is built from today, as pkg/ui/static/ui.css defines it and DESIGN.md describes it: its materials, then the marks and fittings made of them. A chip is the stylesheet\'s own value. These are settled.' },
  { file: 'Swatches-Tried', src: 'tried', row: 0, title: 'Swatches: tried', intro: 'What was drawn for the page and is not on it: one material proposed and open, and those set aside when another was chosen. Each names where its drawing is kept. A chip is the drawing\'s own value.' },
  { file: 'Swatches-Thermal', src: 'thermal', row: 1, found: true, title: 'Swatches: thermal paper', intro: 'Direct-thermal paper and its print, as a counter\'s receipts and a yard\'s part tags come off the printer.' },
  { file: 'Swatches-Yard', src: 'yard', row: 1, found: true, title: 'Swatches: the yard and the shelf', intro: 'What parts and vehicles are tagged, labelled and marked with in a yard, a warehouse and a shop.' },
  { file: 'Swatches-Vehicle', src: 'vehicle', row: 1, found: true, title: 'Swatches: the vehicle and the garage', intro: 'Surfaces of the vehicle itself, its labels and instruments, and of the garage around it.' },
];
const FOUND = " Found by a web search on 2026-11-20. Dimensions and conventions are the sources'; every colour is an estimate, since no source publishes one, and no recipe here is settled.";
// The addresses of the page's two textures on the maintainer's canvas, as scripts/canvas/engine.mjs has them.
const BLOB = { 'brushed.svg': '/_blob/9d26d515fa68a38e6638bb4a880cacf5', 'paper.svg': '/_blob/8df67eb7edb3679ce1a61d297a1fcd76' };
const hex = (h) => /^#[0-9a-fA-F]{3,8}$/.test(h || '') ? h : null;
const swatches = [];
for (const b of BOARDS) {
  const list = JSON.parse(readFileSync(`${REPO}/scripts/canvas/swatches/${b.src}.json`, 'utf8'));
  let css = '', cards = '';
  list.forEach((s, i) => {
    const base = hex(s.base) || '#cccccc', ink = hex(s.ink) || '#111111', accent = hex(s.accent);
    let bg = String(s.css || '').trim().replace(/^background(-image)?\s*:/, '').replace(/;\s*$/, '').trim();
    // The page's two textures are images uploaded to the canvas; no other url() is drawn.
    bg = bg.replace(/url\((brushed|paper)\.svg\)/g, (_, f) => `url(${BLOB[f + '.svg']})`);
    if (!bg || /url\((?!\/_blob\/[0-9a-f]{32}\))|[<>{}";]/.test(bg)) bg = base;
    css += `.sw-${i} .chip{background:${bg};color:${ink}}\n`;
    const dot = (h, l) => h ? `<span class="hx"><i style="background: ${h}"></i>${l} ${h.toLowerCase()}</span>` : '';
    // A source is a link to the page itself, named by its site; a second page of one site is numbered.
    const seen = {};
    const links = (s.sources || []).filter((u) => /^https?:\/\/[^\s"<>]+$/.test(u)).map((u) => {
      const h = new URL(u).hostname.replace(/^www\./, '').toLowerCase(); seen[h] = (seen[h] || 0) + 1;
      return `<a href="${esc(u)}">${esc(h)}${seen[h] > 1 ? ' (' + seen[h] + ')' : ''}</a>`;
    });
    cards += `<article class="sw sw-${i}">
<div class="chip"><b>DEMO-S-01</b><span>demo-north · grade A</span>${accent ? `<i style="background: ${accent}"></i>` : ''}</div>
<h3>${esc(s.name)}</h3>
<p class="fam">${esc(s.family)}</p>
<p>${esc(s.where_seen)}</p>
<p class="tx">${esc(s.texture)}</p>
<p class="hxs">${dot(base, 'base')}${dot(ink, 'ink')}${dot(accent, 'accent')}</p>
<p class="src">${s.in ? 'ui.css: ' + esc(s.in) : s.from ? esc(s.from) : links.join(' · ')}</p>
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
.sw .src a{color:#0b4fa8}.sw .src a:hover{color:#14181d}
${css}`;
  const body = (h) => `<div class="swb" style="height: ${h}">
<div><h1>${esc(b.title)}</h1><p class="intro">${esc(b.intro)}${b.found ? FOUND : ''}</p></div>
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

// The index: the page, its boards in a row, and the row's title.
const PAGE = { id: 'swatches', name: 'Swatches' };
const path = `${OUT}/canvas.json`, idx = JSON.parse(readFileSync(path, 'utf8'));
if (!idx.pages.some((p) => p.id === PAGE.id)) idx.pages.push(PAGE);
// Two rows: what was made, in use and tried, over what was found.
const rowOf = (r) => swatches.filter((s) => s.row === r);
const foundY = Math.max(...rowOf(0).map((s) => s.h)) + 520;
for (const r of [0, 1]) rowOf(r).forEach((s, i) => {
  const k = `${s.file}.dc.html`;
  idx.boards[k] = { ...(idx.boards[k] || {}), h: s.h, page: PAGE.id, title: s.title, w: s.w, x: i * 1360, y: r ? foundY : 0 };
  if (!idx.order.includes(k)) idx.order.push(k);
});
const title = (id, text, n, y) => {
  idx.notes[id] = { ...(idx.notes[id] || {}), kind: 'title1', maxW: n * 1360 - 80, page: PAGE.id, text, w: 240, x: 0, y };
  idx.noteOrder = idx.noteOrder || []; if (!idx.noteOrder.includes(id)) idx.noteOrder.push(id);
};
title('swatches', 'Made: in use, and tried', rowOf(0).length, -300);
title('swatches-found', 'Found: by a web search, none settled', rowOf(1).length, foundY - 300);
writeFileSync(path, JSON.stringify(idx, null, 2) + '\n');
