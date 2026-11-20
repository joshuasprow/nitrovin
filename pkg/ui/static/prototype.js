// PROTOTYPE, never for main (#366): four variants of the part types
// control, on the real page, switched by ?variant= and the bar at the
// bottom of the window. The page's own strip stays in the DOM, hidden, and
// every press here clicks its link, so the candidates, the URL and Back are
// the page's own. What each variant changes is what the control looks like
// and what a press on the pressed part type does: here, nothing. Clearing
// is its own control.
//
//   0  the page as it is
//   A  Chips: every part type, small, in the catalog's order, wrapping
//   B  One line: the "Parts that fit" strip holds the part type field
//   C  Rail: one row that scrolls sideways, in the catalog's order
//   D  Index: an always-open list under the part vocabulary's headings
(() => {
  const VARIANTS = [['0', 'The page as it is'], ['A', 'Chips, fixed order'], ['B', 'One line: strip + field'], ['C', 'Sideways rail'], ['D', 'Index']];
  const q = new URLSearchParams(location.search);
  let variant = q.get('variant') || sessionStorage.getItem('proto-variant') || 'A';
  let filter = '';
  let open = null; // B and C: whether the list is open; null = the variant's default

  const el = (tag, cls, text) => { const e = document.createElement(tag); if (cls) e.className = cls; if (text != null) e.textContent = text; return e; };
  const recent = () => JSON.parse(sessionStorage.getItem('proto-recent') || '[]');

  // The part types, in the dialog's order (the part vocabulary's), each
  // with the strip's own link.
  const read = (fit) => {
    const links = new Map([...fit.querySelectorAll('.ptypes > li[data-part] > a.ptype')].map((a) => [a.parentElement.dataset.part, a]));
    const seen = new Set(), types = [];
    for (const g of fit.querySelectorAll('#part-types .ptgroups .ptgroup')) {
      const heading = (g.querySelector('h3') || {}).textContent || '';
      for (const li of g.querySelectorAll('li[data-part]')) {
        const id = li.dataset.part, a = links.get(id);
        if (seen.has(id) || !a) continue;
        seen.add(id);
        const n = a.querySelector('.n');
        types.push({ id, heading, a, name: a.querySelector('.nm').textContent, n: n ? parseInt(n.textContent, 10) : 0, pressed: a.hasAttribute('aria-current') });
      }
    }
    return types;
  };

  const press = (t) => {
    if (t.pressed) return; // the pressed part type stays pressed
    sessionStorage.setItem('proto-recent', JSON.stringify([t.id, ...recent().filter((x) => x !== t.id)].slice(0, 6)));
    open = null; filter = '';
    t.a.click();
  };
  const clear = (t) => { open = null; t.a.click(); };

  const item = (t, cls) => {
    const a = el('a', cls + (t.pressed ? ' pp' : '') + (!t.pressed && recent().includes(t.id) ? ' pr' : ''));
    a.href = t.pressed ? location.href : t.a.href;
    a.append(el('span', 'nm', t.name), el('span', 'n', String(t.n)));
    if (t.pressed) a.setAttribute('aria-current', 'true');
    a.addEventListener('click', (e) => { if (e.metaKey || e.ctrlKey) return; e.preventDefault(); press(t); });
    return a;
  };
  const clearKey = (t) => {
    const b = el('button', 'pclear', '×');
    b.type = 'button';
    b.setAttribute('aria-label', 'Clear ' + t.name);
    b.addEventListener('click', () => clear(t));
    return b;
  };
  const find = (host, redraw) => {
    const i = el('input', 'pfind');
    i.type = 'search'; i.placeholder = 'Filter part types'; i.value = filter;
    i.setAttribute('aria-label', 'Filter part types');
    i.addEventListener('input', () => { filter = i.value; redraw(); const n = host.querySelector('.pfind'); n.focus(); n.setSelectionRange(filter.length, filter.length); });
    return i;
  };
  const shown = (types) => types.filter((t) => t.pressed || t.name.toLowerCase().includes(filter.toLowerCase()));
  const grouped = (types, cls) => {
    const box = el('div', cls);
    let g, h = null;
    for (const t of shown(types)) {
      if (t.heading !== h) { h = t.heading; g = el('section', 'pg'); if (h) g.append(el('h4', 'pgh', h)); g.append(el('ul', 'pgl')); box.append(g); }
      const li = el('li'); const row = item(t, 'prow'); li.append(row); if (t.pressed) li.append(clearKey(t));
      g.lastChild.append(li);
    }
    return box;
  };

  const draw = {
    A(host, types, redraw) {
      const bar = el('div', 'pa');
      if (types.length > 6) bar.append(find(host, redraw));
      for (const t of shown(types)) {
        const c = el('span', 'pchip' + (t.pressed ? ' on' : ''));
        c.append(item(t, 'pc')); if (t.pressed) c.append(clearKey(t));
        bar.append(c);
      }
      host.append(bar);
    },
    // B: the "Parts that fit" strip's own first cell is the field. The
    // strip names the catalog, the part type, the model and the year in
    // one line; the list opens under the whole strip.
    B(host, types, redraw, fit) {
      const on = types.find((t) => t.pressed);
      const isOpen = open == null ? !on : open;
      const strip = fit.querySelector('.fitstrip'), stamp = strip.querySelector('.stamp');
      strip.querySelectorAll('.pbcell').forEach((e) => e.remove());
      stamp.classList.add('proto-off');
      const cell = el('div', 'pbcell');
      const field = el('button', 'pfield');
      field.type = 'button'; field.setAttribute('aria-expanded', String(isOpen));
      field.append(el('span', 'plab', stamp.textContent), el('span', 'pval', on ? on.name : 'Choose a part type'), el('span', 'pcount', String(on ? on.n : types.length)), el('span', 'pchev', isOpen ? '▴' : '▾'));
      field.addEventListener('click', () => { open = !isOpen; redraw(); });
      cell.append(field);
      if (on) cell.append(clearKey(on));
      stamp.after(cell);
      // Where the lines to the candidates leave from: between the model
      // and the year, under the strip.
      host.append(el('span', 'panchor pp'));
      const rec = recent().map((id) => types.find((t) => t.id === id && !t.pressed)).filter(Boolean).slice(0, 3);
      if (rec.length && !isOpen) {
        const r = el('div', 'pbrecent'); r.append(el('span', 'plab', 'Recent'));
        for (const t of rec) r.append(item(t, 'pc'));
        host.append(r);
      }
      if (isOpen) {
        const panel = el('div', 'pbpanel');
        panel.append(find(host, redraw), grouped(types, 'pcols'));
        host.append(panel);
      }
    },
    C(host, types, redraw) {
      const isOpen = !!open;
      const wrap = el('div', 'pcwrap');
      const all = el('button', 'pall', 'All ' + types.length + (isOpen ? ' ▴' : ' ▾'));
      all.type = 'button'; all.setAttribute('aria-expanded', String(isOpen));
      all.addEventListener('click', () => { open = !isOpen; redraw(); });
      const rail = el('div', 'prail'); rail.tabIndex = 0; rail.setAttribute('aria-label', 'Part types');
      for (const t of types) { const c = el('span', 'pchip' + (t.pressed ? ' on' : '')); c.append(item(t, 'pc')); if (t.pressed) c.append(clearKey(t)); rail.append(c); }
      wrap.append(all, rail);
      host.append(wrap);
      if (isOpen) { const panel = el('div', 'pbpanel'); panel.append(find(host, redraw), grouped(types, 'pcols')); host.append(panel); }
      const on = rail.querySelector('.on');
      if (on) requestAnimationFrame(() => { rail.scrollLeft = on.offsetLeft - rail.clientWidth / 2 + on.offsetWidth / 2; });
    },
    D(host, types, redraw) {
      const box = el('div', 'pd');
      if (types.length > 6) box.append(find(host, redraw));
      box.append(grouped(types, 'pcols index'));
      host.append(box);
    },
  };

  const build = () => {
    const fit = document.getElementById('fit');
    const wrap = fit && fit.querySelector('.ptwrap');
    if (!wrap) return;
    wrap.querySelectorAll(':scope > .proto').forEach((e) => e.remove());
    fit.querySelectorAll('.pbcell').forEach((e) => e.remove());
    fit.querySelectorAll('.proto-off').forEach((e) => e.classList.remove('proto-off'));
    fit.classList.toggle('proto-b', variant === 'B');
    wrap.classList.toggle('proto-host', variant !== '0');
    const types = read(fit);
    if (variant !== '0' && types.length) {
      const host = el('div', 'proto proto-' + variant);
      const redraw = () => { host.replaceChildren(); draw[variant](host, types, redraw, fit); };
      redraw();
      wrap.append(host);
    }
    const on = types.find((t) => t.pressed);
    state.textContent = 'pressed: ' + (on ? on.id : 'none') + ' · recent: ' + (recent().filter((id) => !on || id !== on.id).join(', ') || 'none') + ' · ' + types.length + ' part types';
  };

  // The bar: previous, the variant, next; and the state under it.
  const bar = el('div', 'proto-bar');
  const label = el('span', 'proto-label'), state = el('span', 'proto-state');
  const go = (d) => {
    const i = (VARIANTS.findIndex((v) => v[0] === variant) + d + VARIANTS.length) % VARIANTS.length;
    variant = VARIANTS[i][0]; open = null; filter = '';
    sessionStorage.setItem('proto-variant', variant);
    const u = new URL(location.href); u.searchParams.set('variant', variant); history.replaceState(history.state, '', u);
    paint();
  };
  const paint = () => { label.textContent = variant + ' · ' + VARIANTS.find((v) => v[0] === variant)[1]; build(); };
  const prev = el('button', '', '←'), next = el('button', '', '→');
  prev.type = next.type = 'button';
  prev.addEventListener('click', () => go(-1)); next.addEventListener('click', () => go(1));
  const top = el('div', 'proto-top'); top.append(prev, label, next);
  bar.append(top, state);
  document.addEventListener('keydown', (e) => {
    if (e.target.closest('input, textarea, [contenteditable]')) return;
    if (e.key === 'ArrowLeft') go(-1);
    if (e.key === 'ArrowRight') go(1);
  });

  let busy = false;
  const start = () => {
    document.body.append(bar);
    paint();
    // The page swaps its answer in place: draw again on each one.
    new MutationObserver(() => {
      if (busy) return;
      const wrap = document.querySelector('#fit .ptwrap');
      if (wrap && variant !== '0' && wrap.querySelector(':scope > .proto')) return;
      busy = true; build(); busy = false;
    }).observe(document.getElementById('ui-main'), { childList: true, subtree: true });
  };
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start); else start();
})();
