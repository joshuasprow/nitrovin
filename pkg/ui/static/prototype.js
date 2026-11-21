// PROTOTYPE, never for main (#366): the vehicle block. The "Parts that
// fit" strip becomes the block that says what the vehicle is: the model
// and the year on its top row, and on the same tape "This vehicle", the
// catalog, the body, the engine and VIN 4 to 8. The "Also known as" names
// leave their strips of tape and hang under the block as manila tags, one
// a vocabulary: its name on tape, then make, model and years with no
// labels, each over its group's colour.
//
//   0  the page as it is
//   A  the block, the name tags in a row under it, then the picker
//   B  the block, then the picker with the name tags beside it
(() => {
  const VARIANTS = [['0', 'The page as it is'], ['A', 'Name tags under the block'], ['B', 'Name tags beside the picker']];
  let variant = new URLSearchParams(location.search).get('variant') || sessionStorage.getItem('proto-variant') || 'A';
  const el = (tag, cls, text) => { const e = document.createElement(tag); if (cls) e.className = cls; if (text != null) e.textContent = text; return e; };

  const build = () => {
    const answer = document.getElementById('answer'), fit = document.getElementById('fit');
    if (!answer) return;
    document.body.dataset.proto = variant;
    answer.querySelectorAll('.pnames').forEach((e) => e.remove());
    const strip = fit && fit.querySelector('.fitstrip');
    const facts = fit && fit.querySelector('.vfacts');
    const stamp = strip && strip.querySelector('.stamp');
    if (stamp && !stamp.dataset.was) stamp.dataset.was = stamp.textContent;
    if (variant === '0') {
      if (stamp) stamp.textContent = stamp.dataset.was;
      if (facts && strip && facts.parentElement === strip) fit.querySelector('.ptwrap').append(facts);
      state.textContent = 'the page as built';
      return;
    }
    if (!strip) { state.textContent = 'no vehicle block: the model is not known, or no catalog'; return; }
    stamp.textContent = stamp.dataset.was.replace('Parts that fit', 'This vehicle');
    if (facts) strip.append(facts);
    // The names, a tag a name: read from the strips the page drew.
    const names = el('ol', 'pnames');
    names.setAttribute('aria-label', 'Also known as');
    // One tag a vocabulary: its name on the tape, with its count where it
    // has several names; the make once where every name shares it; then a
    // line a name, the model and the years it holds for.
    for (const stack of answer.querySelectorAll('#aka .stack')) {
      const plates = [...stack.querySelectorAll('.tagplate')];
      const voc = stack.querySelector('.voc') || stack.querySelector('.stamp');
      const tag = el('li', 'card done pname');
      const named = plates.filter((p) => !p.querySelector(':scope > .none:not(.seg)'));
      tag.append(el('p', 'pvoc', (voc ? voc.textContent.trim() : 'Also known as') + (named.length > 1 ? ' · ' + named.length : '')));
      const word = (plate, g) => { const w = plate.querySelector('.seg.' + g + ' .w'); return w ? w.textContent.trim() : ''; };
      const makes = [...new Set(named.map((p) => word(p, 'g-wmi')))];
      const shared = makes.length === 1 && makes[0];
      if (shared) tag.append(el('p', 'pw g-wmi', shared));
      for (const plate of plates) {
        const none = plate.querySelector(':scope > .none:not(.seg)');
        if (none) { tag.append(el('p', 'pnone', none.textContent)); continue; }
        const row = el('p', 'prow2');
        if (!shared && word(plate, 'g-wmi')) row.append(el('span', 'pw g-wmi', word(plate, 'g-wmi')));
        row.append(el('span', 'pw g-vds' + (plate.querySelector('.seg.g-vds.unsure') ? ' unsure' : ''), word(plate, 'g-vds')));
        row.append(el('span', 'pw g-year' + (plate.querySelector('.seg.g-year.unsure') ? ' unsure' : ''), word(plate, 'g-year')));
        tag.append(row);
      }
      names.append(tag);
    }
    if (names.children.length) (variant === 'B' ? fit.querySelector('.ptwrap') : strip).after(names);
    state.textContent = names.children.length + ' name tag(s) · facts on the tape: ' + (facts ? 'yes' : 'none for this vehicle');
  };

  const bar = el('div', 'proto-bar'), label = el('span', 'proto-label'), state = el('span', 'proto-state');
  const go = (d) => {
    const i = (VARIANTS.findIndex((v) => v[0] === variant) + d + VARIANTS.length) % VARIANTS.length;
    variant = VARIANTS[i][0];
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
    new MutationObserver(() => {
      if (busy || document.querySelector('#answer .pnames') || variant === '0') return;
      busy = true; build(); busy = false;
    }).observe(document.getElementById('ui-main'), { childList: true, subtree: true });
  };
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start); else start();
})();
