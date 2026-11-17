// PROTOTYPE (prototype/pare-back-ui, local://vin-page-spec.md). The plate
// (pkg/ui, page.html): served from /ui/r/<release>/plate.js under
// script-src 'self'. Without this script the plate shows what the server
// drew and the input below it is sent with Decode. With it:
//
// - the input lies over the plate's 17 cells (form.live) and what is typed
//   shows in them, cleaned as the server cleans it: letters and digits,
//   upper case, 17 at most, never a letter swapped;
// - the page asks the server as the visitor types (150 ms after the last
//   key, dropping an answer still on its way) and swaps in the answer and
//   the Developers dialog it drew: the same HTML the form's answer is;
// - the URL follows what is typed in place, and a whole VIN is a history
//   entry, so Back steps through VINs and not keystrokes;
// - a group whose characters were just completed is marked fresh, which
//   the stylesheet draws in;
// - the plate edits as a text field does, drawn on the cells: a click
//   puts the caret between two characters, where typing inserts and
//   Backspace takes out the one before; a drag selects the cells it
//   crosses and a double click one, and typing replaces what is selected.
//   A whole VIN takes no more: nothing is pushed off its end;
// - whole VINs that decoded are kept in this browser (the last 8) and
//   offered as chips in the examples' place.
//
// The listeners are the document's, so a page htmx swaps in needs no more
// than init.
'use strict';

(() => {
  const LENGTH = 17;
  const WAIT = 150;
  const KEPT = 8;
  const STORE = 'nitrovin.recent';

  const clean = (s) => s.toUpperCase().replace(/[^A-Z0-9]/g, '').slice(0, LENGTH);
  const urlFor = (vin) => '/' + (vin ? '?' + new URLSearchParams({ 'from.vin': vin }) : '');

  // paint shows vin in the cells; a cell just filled is stamped.
  const paint = (vin, stamp) => {
    const form = document.getElementById('vin-form');
    if (!form) return;
    form.querySelectorAll('.cell').forEach((cell, i) => {
      const c = vin[i] || '';
      const ch = cell.firstElementChild;
      if (stamp && c && ch.textContent !== c) {
        ch.classList.remove('stamp');
        void ch.offsetWidth; // restart the animation
        ch.classList.add('stamp');
      }
      ch.textContent = c;
      cell.classList.toggle('filled', c !== '');
      cell.classList.toggle('bad', 'IOQ'.includes(c) && c !== '');
      cell.classList.toggle('next', i === vin.length);
    });
  };

  const done = (answer) => new Set((answer && answer.dataset.done || '').split(' ').filter(Boolean));

  // swap puts the answer and the Developers dialog of doc, a fragment the
  // server drew, in the page's place, and marks the groups it completes.
  const swap = (doc) => {
    const old = document.getElementById('answer');
    const answer = doc.getElementById('answer');
    if (!old || !answer) return;
    const before = done(old);
    old.replaceWith(answer);
    for (const key of done(answer)) {
      if (!before.has(key)) answer.querySelectorAll('.g-' + key).forEach((el) => el.classList.add('fresh'));
    }
    if (!before.has('aka') && answer.querySelector('#aka.done') && !old.querySelector('#aka.done')) {
      answer.querySelector('#aka').classList.add('fresh');
    }
    const dev = document.getElementById('dev');
    const next = doc.getElementById('dev');
    if (dev && next && !dev.querySelector(':popover-open')) dev.replaceWith(next);
    if (doc.title) document.title = doc.title;
    for (const id of ['answer', 'dev']) {
      const el = document.getElementById(id);
      if (el && window.htmx) {
        htmx.process(el);
        htmx.trigger(el, 'htmx:load');
      }
    }
    remember(answer);
  };

  let timer = 0;
  let pending = null;
  // ask asks the server for vin's answer; how is the history's part:
  // 'push', 'replace' or '' to leave it alone.
  const ask = async (vin, how) => {
    clearTimeout(timer);
    if (pending) pending.abort();
    const mine = pending = new AbortController();
    const url = urlFor(vin);
    let html;
    try {
      const r = await fetch(url, { headers: { 'HX-Request': 'true' }, signal: mine.signal });
      html = await r.text();
    } catch (err) {
      if (err.name === 'AbortError') return;
      throw err;
    }
    if (pending === mine) pending = null;
    swap(new DOMParser().parseFromString(html, 'text/html'));
    const whole = vin.length === LENGTH;
    if (how === 'push') history.pushState({ whole }, '', url);
    else if (how === 'replace') history.replaceState({ whole }, '', url);
  };

  // A whole VIN is an entry of its own: reaching one pushes, and so does
  // the first change to one, so Back returns to it.
  const how = (vin) => (vin.length === LENGTH || (history.state && history.state.whole) ? 'push' : 'replace');

  // show draws the input's selection on the cells: the cells selected,
  // or the caret, a bar before the cell it stands at (after the last
  // cell of a whole VIN) and the next empty cell's mark at the end.
  const show = () => {
    const input = document.getElementById('from-vin');
    const form = document.getElementById('vin-form');
    if (!input || !form) return;
    const on = document.activeElement === input;
    const n = input.value.length;
    const from = input.selectionStart;
    const to = input.selectionEnd;
    const caret = on && from === to;
    form.querySelectorAll('.cell').forEach((cell, i) => {
      cell.classList.toggle('sel', on && i >= from && i < to);
      cell.classList.toggle('next', i === n && !(on && from < n));
      cell.classList.toggle('caret', caret && from < n && i === from);
      cell.classList.toggle('caret-end', caret && from === LENGTH && i === LENGTH - 1);
    });
  };
  // edgeAt is the gap between cells nearest x, of those the VIN reaches;
  // cellAt the cell under x.
  const where = (x) => {
    const box = document.querySelector('#vin-form .cells').getBoundingClientRect();
    return (x - box.left) / box.width * LENGTH;
  };
  const edgeAt = (x, n) => Math.max(0, Math.min(n, Math.round(where(x))));
  const cellAt = (x) => Math.max(0, Math.min(LENGTH - 1, Math.floor(where(x))));
  const select = (input, a, b) => {
    input.setSelectionRange(Math.min(a, b), Math.max(a, b), b < a ? 'backward' : 'forward');
    show();
  };

  // A whole VIN takes no more: a character typed at the caret is refused,
  // not let push the last one off. One no VIN holds is not typed at all:
  // it would only take out what is selected.
  document.addEventListener('beforeinput', (e) => {
    const input = e.target;
    if (input.id !== 'from-vin' || e.inputType !== 'insertText' || !e.data) return;
    const room = LENGTH - input.value.length + (input.selectionEnd - input.selectionStart);
    if (!clean(e.data) || room <= 0) e.preventDefault();
  });

  document.addEventListener('input', (e) => {
    const input = e.target;
    if (input.id !== 'from-vin') return;
    const vin = clean(input.value);
    const at = clean(input.value.slice(0, input.selectionStart)).length;
    if (input.value !== vin) {
      input.value = vin;
      input.setSelectionRange(at, at);
    }
    paint(vin, true);
    show();
    clearTimeout(timer);
    timer = setTimeout(() => ask(vin, how(vin)).catch(console.error), vin.length === LENGTH ? 0 : WAIT);
  });

  // Pasted text goes in at the caret or over the selection, as much of it
  // as there is room for; a whole VIN replaces everything. insertText
  // keeps the input's own undo.
  document.addEventListener('paste', (e) => {
    const input = e.target;
    if (input.id !== 'from-vin' || !e.clipboardData) return;
    e.preventDefault();
    let text = clean(e.clipboardData.getData('text'));
    if (text.length === LENGTH) input.select();
    text = text.slice(0, LENGTH - input.value.length + (input.selectionEnd - input.selectionStart));
    if (!text) return;
    if (!document.execCommand('insertText', false, text)) {
      input.setRangeText(text, input.selectionStart, input.selectionEnd, 'end');
      input.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText' }));
    }
  });

  document.addEventListener('submit', (e) => {
    if (e.target.id !== 'vin-form') return;
    e.preventDefault();
    const vin = clean(document.getElementById('from-vin').value);
    ask(vin, how(vin)).catch(console.error);
  });

  // A click puts the caret in the gap nearest it, a drag selects the
  // cells it crosses, Shift and a click selects from the caret to there,
  // a double click the cell under it and a third the whole VIN. The
  // input's text is not where the cells are, so its own click is not used.
  let anchor = -1;
  document.addEventListener('mousedown', (e) => {
    const input = e.target;
    if (input.id !== 'from-vin' || e.button !== 0 || !input.closest('.live')) return;
    e.preventDefault();
    input.focus({ preventScroll: true });
    const n = input.value.length;
    if (e.detail > 2) {
      select(input, 0, n);
      return;
    }
    if (e.detail === 2) {
      const i = Math.min(cellAt(e.clientX), n);
      select(input, i, Math.min(i + 1, n));
      return;
    }
    const edge = edgeAt(e.clientX, n);
    anchor = edge;
    if (e.shiftKey) anchor = input.selectionDirection === 'backward' ? input.selectionEnd : input.selectionStart;
    select(input, anchor, edge);
  });
  document.addEventListener('mousemove', (e) => {
    if (anchor < 0) return;
    if (e.buttons !== 1) {
      anchor = -1;
      return;
    }
    const input = document.getElementById('from-vin');
    select(input, anchor, edgeAt(e.clientX, input.value.length));
  });
  document.addEventListener('mouseup', () => { anchor = -1; });

  // The keys are the input's own; the cells follow its selection.
  document.addEventListener('focusin', (e) => {
    if (e.target.id === 'from-vin') show();
  });
  document.addEventListener('focusout', (e) => {
    if (e.target.id === 'from-vin') setTimeout(show);
  });
  document.addEventListener('selectionchange', show);
  document.addEventListener('select', show);

  window.addEventListener('popstate', () => {
    const vin = clean(new URLSearchParams(location.search).get('from.vin') || '');
    const input = document.getElementById('from-vin');
    if (!input) return;
    input.value = vin;
    paint(vin, false);
    ask(vin, '').catch(console.error);
  });

  // Recent VINs, this browser's own.
  const recent = () => {
    try {
      return JSON.parse(localStorage.getItem(STORE) || '[]').filter((r) => r && r.vin && r.label).slice(0, KEPT);
    } catch {
      return [];
    }
  };
  const chips = () => {
    const list = document.getElementById('recent');
    if (!list) return;
    const kept = recent();
    list.replaceChildren(...kept.map((r) => {
      const li = document.createElement('li');
      const a = document.createElement('a');
      a.className = 'example recent';
      a.href = urlFor(r.vin);
      a.title = r.vin;
      a.textContent = r.label;
      li.append(a);
      return li;
    }));
    list.hidden = kept.length === 0;
    document.getElementById('examples').hidden = kept.length > 0;
    document.getElementById('clear-recent').hidden = kept.length === 0;
  };
  function remember(answer) {
    if (!answer || !('complete' in answer.dataset) || !answer.dataset.label || answer.querySelector('.card.bad')) return;
    const vin = answer.dataset.vin;
    const kept = [{ vin, label: answer.dataset.label }, ...recent().filter((r) => r.vin !== vin)].slice(0, KEPT);
    try {
      localStorage.setItem(STORE, JSON.stringify(kept));
    } catch {
      return; // no storage: no recent VINs
    }
    chips();
  }
  document.addEventListener('click', (e) => {
    if (e.target.id !== 'clear-recent') return;
    localStorage.removeItem(STORE);
    chips();
  });

  const init = () => {
    const form = document.getElementById('vin-form');
    if (!form) return;
    form.classList.add('live');
    const input = document.getElementById('from-vin');
    paint(clean(input.value), false);
    if (!history.state) history.replaceState({ whole: input.value.length === LENGTH }, '', location.href);
    chips();
    remember(document.getElementById('answer'));
  };
  init();
  document.addEventListener('htmx:afterSettle', (e) => {
    if (e.detail.target && e.detail.target.id === 'ui-main') init();
  });
})();
