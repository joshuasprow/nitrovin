// The repo's snapshot of the design canvas against the canvas itself
// (AGENTS.md, "Design Canvas"). The snapshot holds two of the canvas's
// pages and no other: the page as built and the swatch sheets. A proposal's
// page, and a page drawn by hand, stay on the canvas. Run it through
// scripts/canvas.sh; it needs no browser.
//
//   import <dir>    <dir> is the canvas as read down. Copies the allowlisted
//                   pages' frames into the snapshot, removes any frame the
//                   snapshot holds beside them, and writes the index cut to
//                   those pages.
//   publish <dir>   <dir> is the canvas as read down, just now. Writes
//                   build/canvas-publish/project: the built frames, and the
//                   read's index with their heights and the dark row's
//                   place set. That folder is what is published, so every
//                   key that is not a built frame's goes back as read.
//   lint            exits 1 naming what the snapshot holds off the
//                   allowlist. canvas.sh --check runs it first.
import { readFileSync, writeFileSync, readdirSync, copyFileSync, rmSync, mkdirSync, existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

export const REPO = fileURLToPath(new URL('../..', import.meta.url)).replace(/\/$/, '');
export const OUT = `${REPO}/docs/design/canvas/project`;
// The pages the snapshot holds, by id: the page as built, which is the
// canvas's first, and the swatch sheets.
export const PAGES = ['page-1', 'swatches'];
// The frames scripts/canvas.sh builds. frames.mjs has what each shows.
export const BUILT = ['Main', 'Materials', 'Diagram', 'Fitment', 'Controls', 'States'].flatMap((f) => [`${f}.dc.html`, `${f}-Dark.dc.html`]);

const json = (p) => JSON.parse(readFileSync(p, 'utf8'));
const write = (p, idx) => writeFileSync(p, JSON.stringify(idx, null, 2) + '\n');
// A read-down folder holds the files under project/, as the canvas serves
// them, or bare.
const project = (dir) => (existsSync(`${dir}/project/canvas.json`) ? `${dir}/project` : dir);
// A board or a note with no page is on the canvas's first.
const pageOf = (idx, item) => item.page ?? idx.pages[0].id;
const frameFiles = (dir) => readdirSync(dir).filter((f) => f.endsWith('.dc.html')).sort();

// cut is idx with only the allowlisted pages' boards, notes and pages.
export function cut(idx) {
  const keep = (item) => PAGES.includes(pageOf(idx, item));
  const pick = (obj) => Object.fromEntries(Object.entries(obj ?? {}).filter(([, v]) => keep(v)));
  const boards = pick(idx.boards), notes = pick(idx.notes);
  return {
    ...idx, boards, notes,
    order: (idx.order ?? []).filter((k) => k in boards),
    noteOrder: (idx.noteOrder ?? []).filter((k) => k in notes),
    pages: idx.pages.filter((p) => PAGES.includes(p.id)),
  };
}

// place sets, in idx, each built frame's height and the dark row's place
// under the tallest light frame. heights is a file's height where known.
export function place(idx, heights) {
  const own = Object.entries(idx.boards).filter(([k]) => BUILT.includes(k));
  for (const [k, b] of own) if (heights[k]) b.h = heights[k];
  const darkY = Math.max(...own.filter(([k]) => !k.includes('-Dark')).map(([, b]) => b.y + b.h)) + 120;
  for (const [k, b] of own) if (k.includes('-Dark')) b.y = darkY;
  return idx;
}

// problems are what the snapshot holds that the allowlist does not allow.
export function problems(dir = OUT) {
  const idx = json(`${dir}/canvas.json`), out = [];
  for (const p of idx.pages) if (!PAGES.includes(p.id)) out.push(`canvas.json names the page "${p.name}" (${p.id})`);
  for (const [k, b] of Object.entries(idx.boards)) {
    if (!PAGES.includes(pageOf(idx, b))) out.push(`canvas.json places ${k} on the page ${pageOf(idx, b)}`);
    else if (!existsSync(`${dir}/${k}`)) out.push(`canvas.json names ${k}, which the snapshot does not hold`);
  }
  for (const [k, n] of Object.entries(idx.notes ?? {})) if (!PAGES.includes(pageOf(idx, n))) out.push(`canvas.json places the note "${k}" on the page ${pageOf(idx, n)}`);
  for (const f of frameFiles(dir)) if (!(f in idx.boards)) out.push(`the snapshot holds ${f}, which canvas.json does not name`);
  return out;
}

function importFrom(dir) {
  const from = project(dir), read = json(`${from}/canvas.json`), idx = cut(read);
  const missing = Object.keys(idx.boards).filter((k) => !existsSync(`${from}/${k}`));
  if (missing.length) throw new Error(`${dir} lacks ${missing.join(', ')}: read every frame of the pages ${PAGES.join(' and ')} down`);
  for (const f of frameFiles(OUT)) if (!(f in idx.boards)) { rmSync(`${OUT}/${f}`); console.log('removed', f); }
  for (const k of Object.keys(idx.boards)) copyFileSync(`${from}/${k}`, `${OUT}/${k}`);
  write(`${OUT}/canvas.json`, idx);
  const left = Object.keys(read.boards).filter((k) => !(k in idx.boards));
  console.log(`canvas: ${Object.keys(idx.boards).length} frames of ${PAGES.join(', ')} in the snapshot` + (left.length ? `; left on the canvas: ${left.join(', ')}` : ''));
}

function publishFrom(dir) {
  const idx = json(`${project(dir)}/canvas.json`), mine = json(`${OUT}/canvas.json`), to = `${REPO}/build/canvas-publish/project`;
  const lost = BUILT.filter((k) => !(k in idx.boards));
  if (lost.length) throw new Error(`the canvas as read has no ${lost.join(', ')}: place the built frames on its first page before publishing`);
  rmSync(`${REPO}/build/canvas-publish`, { recursive: true, force: true });
  mkdirSync(to, { recursive: true });
  for (const k of BUILT) copyFileSync(`${OUT}/${k}`, `${to}/${k}`);
  write(`${to}/canvas.json`, place(idx, Object.fromEntries(BUILT.map((k) => [k, mine.boards[k]?.h]))));
  console.log(`canvas: publish ${to}: ${BUILT.length} frames, and canvas.json as read with their heights`);
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const [verb, dir] = process.argv.slice(2);
  try {
    if (verb === 'import' && dir) importFrom(dir);
    else if (verb === 'publish' && dir) publishFrom(dir);
    else if (verb === 'lint') {
      const found = problems();
      if (found.length) {
        console.error(`canvas: the snapshot holds what is not on the pages ${PAGES.join(' and ')}:\n  ${found.join('\n  ')}\nRun scripts/canvas.sh --import <the canvas as read down> (AGENTS.md, "Design Canvas").`);
        process.exit(1);
      }
    } else { console.error('usage: snapshot.mjs import <dir> | publish <dir> | lint'); process.exit(2); }
  } catch (e) { console.error(`canvas: ${e.message}`); process.exit(1); }
}
