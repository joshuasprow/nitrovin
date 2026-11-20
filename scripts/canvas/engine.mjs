// The frame builder behind scripts/canvas.sh. A frame is the page's own
// markup, captured from a running server, and the rules of
// pkg/ui/static/ui.css that the markup uses, with the stylesheet's width and
// colour-scheme queries resolved for each specimen's box.
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { launch } from './cdp.mjs';

export const REPO = fileURLToPath(new URL('../..', import.meta.url)).replace(/\/$/, '');
// The server with the demo set, and one with no vocabulary at all.
export const BASE = process.env.NITROVIN_CANVAS_DEMO || 'http://127.0.0.1:8791';
export const BARE = process.env.NITROVIN_CANVAS_BARE || 'http://127.0.0.1:8793';
const BLOB = { 'brushed.svg': '/_blob/9d26d515fa68a38e6638bb4a880cacf5', 'paper.svg': '/_blob/8df67eb7edb3679ce1a61d297a1fcd76' };
// A specimen's width class: the window the page is captured at, and the
// box it is drawn in on the frame.
export const VP = { wide: { win: 1280, box: 1040 }, tab: { win: 768, box: 768 }, phone: { win: 390, box: 390 } };

// ---- page-side helpers ---------------------------------------------------
const PAGE_LIB = String.raw`
window.__flat = (css) => {
  const sheet = new CSSStyleSheet(); sheet.replaceSync(css);
  const out = [];
  const walk = (rules, ctx) => { for (const r of rules) {
    if (r instanceof CSSStyleRule) out.push({ t: 'style', sel: r.selectorText, body: r.style.cssText, ctx });
    else if (r instanceof CSSMediaRule) walk(r.cssRules, [...ctx, { t: 'media', c: r.conditionText }]);
    else if (r instanceof CSSSupportsRule) walk(r.cssRules, [...ctx, { t: 'supports', c: r.conditionText }]);
    else out.push({ t: 'raw', text: r.cssText, ctx });
  } };
  walk(sheet.cssRules, []);
  return out;
};
window.__media = (conds) => Object.fromEntries(conds.map(c => [c, matchMedia(c).matches]));
window.__snap = (o) => {
  o = o || {};
  const all = [...document.querySelectorAll('*')];
  const mark = (sel, cls) => { for (const e of all) { try { if (e.matches(sel)) e.classList.add(cls); } catch (_) {} } };
  mark(':focus', 'is-f'); mark(':focus-within', 'is-fw'); mark(':focus-visible', 'is-fv'); mark(':popover-open', 'is-open');
  if (location.hash.length > 1) mark(':target', 'is-target');
  for (const s of (o.open || [])) for (const e of document.querySelectorAll(s)) e.classList.add('is-open');
  for (const [s, cls] of Object.entries(o.mark || {})) for (const e of document.querySelectorAll(s)) e.classList.add(...cls.split(' '));
  for (const e of document.querySelectorAll('[popover]')) e.classList.add('popover');
  for (const e of document.querySelectorAll('input, textarea')) {
    if (e.type === 'checkbox' || e.type === 'radio') { e.checked ? e.setAttribute('checked', '') : e.removeAttribute('checked'); }
    else if (e.tagName === 'TEXTAREA') e.textContent = e.value;
    else if (e.type !== 'file') e.setAttribute('value', e.value);
  }
  const body = document.body.cloneNode(true);
  for (const e of body.querySelectorAll('script, link, style, template, noscript, .popover:not(.is-open):not(nav.top)')) e.remove();
  for (const s of (o.drop || [])) for (const e of body.querySelectorAll(s)) e.remove();
  if (o.keep && o.keep.length) {
    const live = new Set();
    for (const s of o.keep) for (const e of body.querySelectorAll(s)) {
      live.add(e); for (const d of e.querySelectorAll('*')) live.add(d);
      for (let p = e.parentElement; p; p = p.parentElement) live.add(p);
    }
    for (const e of [...body.querySelectorAll('*')]) if (!live.has(e)) e.remove();
  }
  const VOID = new Set(['area','base','br','col','embed','hr','img','input','link','meta','source','track','wbr']);
  const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/\{\{/g, '{&#8203;{');
  const escA = (s) => esc(s).replace(/"/g, '&quot;');
  const ser = (n) => {
    if (n.nodeType === 3) return esc(n.nodeValue);
    if (n.nodeType !== 1) return '';
    const tag = n.namespaceURI === 'http://www.w3.org/1999/xhtml' ? n.tagName.toLowerCase() : n.tagName;
    let a = '';
    for (const at of n.attributes) {
      let k = at.name, v = at.value;
      if (/^(hx-|on|popover|autofocus|data-hx|nonce|action|method|name$|form$|inert|autocomplete|autocapitalize|spellcheck|enterkeyhint|inputmode|pattern|required|maxlength|minlength)/.test(k)) continue;
      if (k === 'href') v = '#';
      if (k === 'type' && tag === 'button') continue;
      a += ' ' + k + '="' + escA(v) + '"';
    }
    if (tag === 'button') a += ' type="button"';
    if (tag === 'dialog') a += ' open=""';
    if (VOID.has(tag)) return '<' + tag + a + '>';
    let inner = ''; for (const k of n.childNodes) inner += ser(k);
    return '<' + tag + a + '>' + inner + '</' + tag + '>';
  };
  let html = ''; for (const k of body.childNodes) html += ser(k);
  return html.replace(/\n\s*\n+/g, '\n');
};
window.__prune = (tests) => tests.map(t => { try { return !!document.querySelector(t); } catch (_) { return true; } });
`;

// ---- selector rewriting --------------------------------------------------
function splitTop(s) {
  const out = []; let d = 0, q = null, cur = '';
  for (const ch of s) {
    if (q) { cur += ch; if (ch === q) q = null; continue; }
    if (ch === '"' || ch === "'") { q = ch; cur += ch; continue; }
    if (ch === '(' || ch === '[') d++;
    if (ch === ')' || ch === ']') d--;
    if (ch === ',' && d === 0) { out.push(cur.trim()); cur = ''; continue; }
    cur += ch;
  }
  if (cur.trim()) out.push(cur.trim());
  return out;
}
const ROOTS = /(^|[\s,>+~(])(?::root|html|body)(?![-\w])/g;
const STATE = [[':focus-visible', '.is-fv'], [':focus-within', '.is-fw'], [':focus', '.is-f'], [':active', '.is-active'], [':hover', '.is-hover'], [':target', '.is-target']];
function emitSel(sel) {
  let s = sel.replace(ROOTS, '$1&').replace(/:popover-open/g, '.is-open').replace(/\[popover\]/g, '.popover');
  for (const [p, c] of STATE) s = s.replace(new RegExp(p.replace(/[-:]/g, m => '\\' + m) + '(?![-\\w])', 'g'), `:is(${p},${c})`);
  return s;
}
function testSel(sel, P) {
  let s = sel.replace(ROOTS, '$1&').replace(/:popover-open/g, '.is-open').replace(/\[popover\]/g, '.popover');
  s = s.replace(/::[\w-]+(\([^)]*\))?/g, '');
  s = s.replace(/:(focus-visible|focus-within|focus|active|hover|target|checked|disabled|placeholder-shown|user-invalid|invalid|open)(?![-\w(])/g, '');
  return splitTop(s).map(x => { x = x.trim(); if (!x) x = '*'; return x.includes('&') ? x.replaceAll('&', P) : P + ' ' + x; }).join(',');
}
const fixBody = (b) => b.replace(/url\("?[^")]*?(brushed\.svg|paper\.svg)"?\)/g, (_, f) => `url(${BLOB[f]})`).replace(/100vw/g, 'var(--vw)').replace(/100d?vh/g, 'var(--vh)');

// ---- parsing the stylesheet's source: the CSSOM's own serialisation of a
// shorthand holding var() loses it, so the declarations are kept as written.
function flatten(src) {
  src = src.replace(/\/\*[\s\S]*?\*\//g, '');
  const out = [];
  const walk = (text, ctx) => {
    let i = 0;
    while (i < text.length) {
      const b = text.indexOf('{', i); if (b < 0) break;
      const semi = text.indexOf(';', i);
      if (semi >= 0 && semi < b) { i = semi + 1; continue; }
      let d = 1, j = b + 1, q = null;
      for (; j < text.length && d; j++) { const ch = text[j]; if (q) { if (ch === q) q = null; } else if (ch === '"' || ch === "'") q = ch; else if (ch === '{') d++; else if (ch === '}') d--; }
      const pre = text.slice(i, b).trim().replace(/\s+/g, ' '), inner = text.slice(b + 1, j - 1);
      if (pre.startsWith('@media')) walk(inner, [...ctx, { t: 'media', c: pre.slice(6).trim() }]);
      else if (pre.startsWith('@supports')) walk(inner, [...ctx, { t: 'supports', c: pre.slice(9).trim() }]);
      else if (pre.startsWith('@')) out.push({ t: 'raw', text: pre + '{' + inner.replace(/\s+/g, ' ') + '}', ctx });
      else out.push({ t: 'style', sel: pre, body: inner.trim().replace(/\s+/g, ' ').replace(/;$/, ''), ctx });
      i = j;
    }
  };
  walk(src, []);
  return out;
}

// ---- the build -----------------------------------------------------------
export async function open() {
  const c = await launch();
  await c.size(1280, 900); await c.scheme('light');
  await c.goto(BASE + '/');
  const css = readFileSync(`${REPO}/pkg/ui/static/ui.css`, 'utf8');
  c.lib = async () => { await c.eval(PAGE_LIB + ';1'); };
  await c.lib();
  c.flat = flatten(css);
  c.conds = [...new Set(c.flat.flatMap(r => r.ctx.filter(x => x.t === 'media').map(x => x.c)))];
  c.mediaCache = {};
  c.media = async (vp, scheme) => {
    const k = vp + scheme; if (c.mediaCache[k]) return c.mediaCache[k];
    await c.size(VP[vp].win, 900); await c.scheme(scheme);
    return c.mediaCache[k] = await c.eval(`__media(${JSON.stringify(c.conds)})`);
  };
  return c;
}

// capture one specimen: { vp, url, keep, drop, open, mark, pre }
export async function capture(c, scheme, sp) {
  await c.size(VP[sp.vp].win, sp.h || 900); await c.scheme(scheme);
  // What the page remembers (recent VINs and part types) starts empty for
  // every specimen, so a frame comes out the same built alone or with the rest.
  for (const o of [BASE, BARE]) await c.send('Storage.clearDataForOrigin', { origin: o, storageTypes: 'local_storage' });
  for (const u of (sp.visit || [])) await c.goto((sp.base || BASE) + u, sp.wait || 300);
  await c.goto((sp.base || BASE) + sp.url, sp.wait || 500);
  await c.lib();
  if (sp.pre) await sp.pre(c);
  let html = await c.eval(`__snap(${JSON.stringify({ keep: sp.keep, drop: sp.drop, open: sp.open, mark: sp.mark })})`);
  if (sp.edit) html = sp.edit(html);
  return html;
}

const SHEET = `
.sheet{box-sizing:border-box;margin:0;padding:40px;display:flex;flex-direction:column;gap:26px}
.sheet .sh-h1{margin:0;font:900 21px/1.2 var(--label-font);letter-spacing:.04em;text-transform:uppercase;border:0;padding:0}
.sheet .sh-intro{margin:6px 0 0;color:var(--muted);max-width:820px}
.sheet .sh-h2{margin:0;padding-bottom:4px;font:900 14px/1.2 var(--label-font);letter-spacing:.08em;text-transform:uppercase;border-bottom:3px solid var(--steel)}
.sheet .sh-cap{margin:0 0 10px;font:900 11px/1.3 var(--label-font);letter-spacing:.08em;text-transform:uppercase;color:var(--fg)}
.sheet .sh-cap span{font:400 13px/1.3 system-ui,sans-serif;letter-spacing:0;text-transform:none;color:var(--muted);margin-left:10px}
.sheet .sh-row{display:flex;gap:40px;align-items:flex-start}
.sheet .vp{box-sizing:border-box;position:relative;margin:0;padding:0;min-height:0;background:none;anchor-scope:all;--vh:900px;overflow:clip;overflow-clip-margin:24px}
.sheet .vp.pad{padding:0 var(--gutter)}
.sheet .vp header,.sheet .vp main{margin-bottom:0}
.sheet .vp .popover.is-open{display:block !important;position:relative !important;inset:auto !important;margin:0 !important;max-height:none !important;overflow:visible !important}
`;

export async function buildFrame(c, frame, scheme) {
  const dark = scheme === 'dark';
  // markup
  let body = `<div><h1 class="sh-h1">${frame.heading}</h1><p class="sh-intro">${frame.intro}</p></div>\n`;
  const used = new Set();
  const spec = async (sp) => {
    used.add(sp.vp);
    const html = await capture(c, scheme, sp);
    const w = sp.box || VP[sp.vp].box;
    const cap = sp.cap ? `<p class="sh-cap">${sp.cap} <span>${sp.note || ''}</span></p>\n` : '';
    return `<section>\n${cap}<div class="vp vp-${sp.vp}${sp.pad === false ? '' : ' pad'}" style="width: ${w}px; --vw: ${VP[sp.vp].win}px${sp.style ? '; ' + sp.style : ''}">\n${html}\n</div>\n</section>\n`;
  };
  for (const it of frame.items) {
    if (it.h2) body += `<h2 class="sh-h2">${it.h2}</h2>\n`;
    else if (it.row) { body += `<div class="sh-row">\n`; for (const sp of it.row) body += await spec(sp); body += `</div>\n`; }
    else body += await spec(it);
  }
  const root = `<div class="root sheet" data-scheme="${scheme}" style="width: ${frame.w}px; height: ${frame.h || 4000}px; color: var(--fg); background: var(--bg)">\n${body}</div>`;

  // rules: resolve media per width, wrap in the width's class
  const vps = [...used];
  const mm = {}; for (const v of vps) mm[v] = await c.media(v, scheme);
  const rules = []; const raws = []; let sheetVars = '';
  for (const r of c.flat) {
    if (r.t === 'raw') { if (!r.ctx.length) raws.push(fixBody(r.text)); continue; }
    if (/::backdrop/.test(r.sel) || !r.body) continue;
    const on = vps.filter(v => r.ctx.filter(x => x.t === 'media').every(x => mm[v][x.c]));
    if (!on.length) continue;
    if (r.sel === 'body' && !r.ctx.some(x => x.t === 'supports')) sheetVars += r.body + ';';
    const P = on.length === vps.length ? '.vp' : on.length === 1 ? `.vp-${on[0]}` : `:is(${on.map(v => '.vp-' + v).join(',')})`;
    rules.push({ P, sup: r.ctx.filter(x => x.t === 'supports').map(x => x.c), sel: emitSel(r.sel), test: testSel(r.sel, P), body: fixBody(r.body) });
  }
  // prune what matches nothing on this frame
  await c.goto('about:blank', 100); await c.lib();
  await c.eval(`document.body.innerHTML = ${JSON.stringify(root)}; 1`);
  const keep = await c.eval(`__prune(${JSON.stringify(rules.map(r => r.test))})`);
  let css = raws.join('\n') + '\n'; let open = null;
  const close = () => { if (open) { css += '}'.repeat(1 + open.sup.length) + '\n'; open = null; } };
  rules.forEach((r, i) => {
    if (!keep[i]) return;
    const key = r.P + '|' + r.sup.join('|');
    if (!open || open.key !== key) { close(); css += r.P + '{' + r.sup.map(s => `@supports ${s}{`).join(''); open = { key, sup: r.sup }; }
    css += `\n${r.sel}{${r.body}}`;
  });
  close();
  css += `.sheet{${fixBody(sheetVars)}}` + SHEET + (frame.css || '');
  const total = rules.length, kept = keep.filter(Boolean).length;
  if (!frame.h) {
    await c.size(frame.w, 800);
    await c.eval(`document.head.insertAdjacentHTML('beforeend', '<style>body{margin:0}' + ${JSON.stringify(css)}.replaceAll('/_blob/', 'x') + '</style>'); document.querySelector('.sheet').style.height = 'auto'; 1`);
    await c.sleep(200);
    frame.measured = Math.max(frame.measured || 0, Math.ceil(await c.eval(`document.querySelector('.sheet').getBoundingClientRect().height`) / 10) * 10 + 100);
  }
  return { root, css, stats: `${kept}/${total} rules` };
}

export function dcFile(frame, scheme, built) {
  const title = frame.title + (scheme === 'dark' ? ', dark' : '');
  return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>${title}</title>
<script src="./support.js"></script>
</head>
<body>
<x-dc>
<helmet>
<style>
body{margin:0}
${built.css}
</style>
</helmet>
${built.root}
</x-dc>
<script type="text/x-dc" data-dc-script data-props='{"$preview":{"width":${frame.w},"height":${frame.h}}}'>
class Component extends DCLogic {
renderVals() {
return {};
}
}
</script>
</body>
</html>
`;
}

export function previewFile(frame, scheme, built) {
  const fix = (s) => s.replaceAll(BLOB['brushed.svg'], 'brushed.svg').replaceAll(BLOB['paper.svg'], 'paper.svg');
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="color-scheme" content="${scheme}"><style>body{margin:0}\n${fix(built.css)}</style></head><body>${fix(built.root)}</body></html>`;
}
