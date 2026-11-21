// PROTOTYPE, not the page: three materials for the shelf list, switched by
// ?variant= on the page's own route. Throwaway; lives only on this branch.
//
// The variant's class, mat-<key>, goes on every stock panel (.rows, .prows,
// .stock-panel), and the rules for it are the PROTOTYPE block at the foot of
// ui.css. A floating bar cycles the variants; so do the arrow keys.
(() => {
  const VARIANTS = [['built', 'As built'], ['receipt', 'Thermal receipt'], ['placard', 'Tire placard'], ['tyvek', 'Tyvek wire tags'], ['marker', 'Paint marker on steel']];
  const key = () => { const v = new URLSearchParams(location.search).get('variant'); return VARIANTS.some(([k]) => k === v) ? v : 'built'; };
  const apply = () => {
    const v = key();
    for (const e of document.querySelectorAll('.rows, .prows, .stock-panel')) {
      for (const [k] of VARIANTS) e.classList.toggle('mat-' + k, k === v && k !== 'built');
    }
    const lab = document.querySelector('#proto-bar output');
    const text = v + ' (' + VARIANTS.find(([k]) => k === v)[1] + ')';
    if (lab && lab.textContent !== text) lab.textContent = text;
  };
  const step = (d) => {
    const i = VARIANTS.findIndex(([k]) => k === key());
    const u = new URL(location.href); u.searchParams.set('variant', VARIANTS[(i + d + VARIANTS.length) % VARIANTS.length][0]);
    history.replaceState(history.state, '', u); apply();
  };
  const bar = document.createElement('div');
  bar.id = 'proto-bar';
  bar.style.cssText = 'position:fixed;z-index:9999;left:50%;bottom:14px;transform:translateX(-50%);display:flex;align-items:center;gap:10px;padding:6px 10px;font:600 13px system-ui,sans-serif;color:#fff;background:#d6006c;border-radius:999px;box-shadow:0 4px 14px rgb(0 0 0/.4)';
  bar.innerHTML = '<button type="button" aria-label="Previous variant">←</button><output></output><button type="button" aria-label="Next variant">→</button>';
  for (const b of bar.querySelectorAll('button')) b.style.cssText = 'all:unset;cursor:pointer;padding:2px 8px;font-size:16px';
  bar.firstChild.onclick = () => step(-1); bar.lastChild.onclick = () => step(1);
  document.body.append(bar);
  document.addEventListener('keydown', (e) => {
    if (e.target.closest('input, textarea, [contenteditable]')) return;
    if (e.key === 'ArrowLeft') step(-1); else if (e.key === 'ArrowRight') step(1);
  });
  // htmx swaps the panel in; the links it follows drop ?variant=, so the
  // variant is kept in the session and put back.
  const keep = () => {
    const u = new URL(location.href);
    if (u.searchParams.has('variant')) sessionStorage.setItem('proto-variant', u.searchParams.get('variant'));
    else if (sessionStorage.getItem('proto-variant')) { u.searchParams.set('variant', sessionStorage.getItem('proto-variant')); history.replaceState(history.state, '', u); }
    apply();
  };
  new MutationObserver(keep).observe(document.body, { childList: true, subtree: true });
  keep();
})();
