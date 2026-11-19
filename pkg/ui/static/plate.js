// The plate
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
// - the plate edits as a text field does, because it is one: the input's
//   characters are spaced a cell apart (ui.css), so its own caret and
//   selection, by click, drag, touch or key, fall on the cells, which
//   draw them. A whole VIN takes no more: nothing is pushed off its end;
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
    // Asked again for the VIN the page is at (Enter on it), it is the
    // same entry: a second would make Back do nothing.
    if (how === 'push' && url === location.pathname + location.search) how = 'replace';
    if (how === 'push') history.pushState({ whole }, '', url);
    else if (how === 'replace') history.replaceState({ whole }, '', url);
  };

  // A whole VIN is an entry of its own: reaching one pushes, and so does
  // the first change to one, so Back returns to it.
  const how = (vin) => (vin.length === LENGTH || (history.state && history.state.whole) ? 'push' : 'replace');

  // show draws the input's selection on the cells: the cells selected,
  // the first and the last of them marked for a finger's handles (hs,
  // he), or the caret, a bar before the cell it stands at (after the last
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
    const some = on && from < to;
    form.classList.toggle('selecting', some);
    form.querySelectorAll('.cell').forEach((cell, i) => {
      cell.classList.toggle('sel', some && i >= from && i < to);
      cell.classList.toggle('hs', some && i === from);
      cell.classList.toggle('he', some && i === to - 1);
      cell.classList.toggle('next', i === n && !(on && from < n));
      cell.classList.toggle('caret', caret && from < n && i === from);
      cell.classList.toggle('caret-end', caret && from === LENGTH && i === LENGTH - 1);
    });
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

  // Enter sends the form: the field is a textarea (init), which would
  // take it as a line's end.
  document.addEventListener('keydown', (e) => {
    if (e.target.id !== 'from-vin' || e.key !== 'Enter' || e.isComposing) return;
    e.preventDefault();
    e.target.form.requestSubmit();
  });

  // gapAt is the gap between characters nearest a point: before the cell
  // under it or after, by the half the point is in, of the gaps the VIN
  // reaches. The cells may be in two rows.
  const gapAt = (x, y, n) => {
    let gap = 0;
    let nearest = Infinity;
    document.querySelectorAll('#vin-form .cell').forEach((cell, i) => {
      const r = cell.getBoundingClientRect();
      const mid = (r.left + r.right) / 2;
      const off = (y < r.top ? r.top - y : Math.max(y - r.bottom, 0)) * 1e4 + Math.abs(x - mid);
      if (off < nearest) {
        nearest = off;
        gap = i + (x > mid ? 1 : 0);
      }
    });
    return Math.min(gap, n);
  };

  // The keys are the input's own, and the cells follow its selection.
  // Focused by anything but a pointer (the page opening, a script), the
  // caret goes after what is there, not before it.
  let touched = -Infinity;
  for (const type of ['pointerdown', 'pointerup']) {
    document.addEventListener(type, () => { touched = performance.now(); }, true);
  }
  // Where a pointer puts the caret is worked out here, from the cell under
  // it (gapAt), because the browsers do not agree with the cells on their
  // own. Firefox halves a character at its glyph and not at its cell, so a
  // click in a cell's left half lands after it. A phone puts the caret at
  // an end of the word tapped, and a VIN is one word; and its tap beside
  // the caret only brings up the caret's menu, with no click (both seen in
  // the iPhone simulator, iOS 26.5). So a click or tap's gap, and a mouse
  // drag's cells, are taken from the pointer and held for a moment against
  // the browser's own placing. A phone's target is an element inside the
  // field, not the field, so the pointer is known by where it lands. The
  // second of two clicks or taps in one place stays the browser's own.
  //
  // For a finger the field's text is out of sight (ui.css), so that the
  // phone draws no highlight or handles of its own over the cells: the
  // handles are the cells' (show's hs and he), and dragging one is done
  // here.
  let down = null;
  let grab = null;
  let tapped = { at: -Infinity, x: 0, y: 0 };
  let want = null;
  const onPlate = (e) => {
    const plate = document.querySelector('#vin-form.live .plate');
    if (!plate) return false;
    const r = plate.getBoundingClientRect();
    return e.clientX >= r.left && e.clientX <= r.right && e.clientY >= r.top && e.clientY <= r.bottom;
  };
  const gapOf = (e) => gapAt(e.clientX, e.clientY, document.getElementById('from-vin').value.length);
  // handleAt is the handle a finger at e takes hold of: the selection's
  // start, at its first cell's left edge with its knob above, or its end,
  // at its last cell's right edge with its knob below.
  const handleAt = (e) => {
    const input = document.getElementById('from-vin');
    const from = input.selectionStart;
    const to = input.selectionEnd;
    if (document.activeElement !== input || from === to) return null;
    const cells = document.querySelectorAll('#vin-form .cell');
    const a = cells[from].getBoundingClientRect();
    const b = cells[to - 1].getBoundingClientRect();
    const near = (x, top, bottom) => Math.abs(e.clientX - x) < 20 && e.clientY > top && e.clientY < bottom;
    const start = near(a.left, a.top - 30, a.bottom + 6);
    const end = near(b.right, b.top - 6, b.bottom + 30);
    if (start && end) return Math.hypot(e.clientX - a.left, e.clientY - a.top) < Math.hypot(e.clientX - b.right, e.clientY - b.bottom) ? { fixed: to, start: true } : { fixed: from, start: false };
    if (start) return { fixed: to, start: true };
    return end ? { fixed: from, start: false } : null;
  };
  document.addEventListener('pointerdown', (e) => {
    want = null;
    down = null;
    grab = null;
    if (!onPlate(e) || (e.pointerType === 'mouse' && e.button !== 0)) return;
    if (e.pointerType === 'touch') {
      grab = handleAt(e);
      if (grab) return;
    }
    if (e.shiftKey) {
      // Shift and a click selects from where the selection began to there.
      const input = document.getElementById('from-vin');
      const anchor = input.selectionDirection === 'backward' ? input.selectionEnd : input.selectionStart;
      const gap = gapOf(e);
      want = { from: Math.min(anchor, gap), to: Math.max(anchor, gap), back: gap < anchor, until: performance.now() + 300 };
      hold();
      return;
    }
    down = { x: e.clientX, y: e.clientY, at: performance.now(), gap: gapOf(e), dragged: false };
  }, true);
  // A finger on a handle is the script's alone: the phone's own gestures
  // (a scroll, a long press) are kept off it.
  document.addEventListener('touchstart', (e) => {
    if (grab) e.preventDefault();
  }, { capture: true, passive: false });
  document.addEventListener('pointermove', (e) => {
    if (!grab || e.pointerType !== 'touch') return;
    const input = document.getElementById('from-vin');
    const gap = gapOf(e);
    // A handle stops a cell short of the other: a selection keeps a cell.
    if (grab.start) input.setSelectionRange(Math.min(gap, grab.fixed - 1), grab.fixed, 'backward');
    else input.setSelectionRange(grab.fixed, Math.max(gap, grab.fixed + 1), 'forward');
    show();
  }, true);
  for (const type of ['pointerup', 'pointercancel']) {
    document.addEventListener(type, () => { grab = null; }, true);
  }
  document.addEventListener('pointermove', (e) => {
    if (!down || e.pointerType !== 'mouse' || e.buttons !== 1) return;
    if (!down.dragged && Math.hypot(e.clientX - down.x, e.clientY - down.y) < 4) return;
    down.dragged = true;
    const gap = gapOf(e);
    want = { from: Math.min(down.gap, gap), to: Math.max(down.gap, gap), back: gap < down.gap, until: performance.now() + 1000 };
    hold();
  }, true);
  // A phone places its caret only once it knows the tap is not the first
  // of two, so the gap is held past that, or until a key is typed.
  document.addEventListener('beforeinput', () => { want = null; }, true);
  document.addEventListener('pointerup', (e) => {
    const now = performance.now();
    const was = down;
    down = null;
    if (!was) return;
    if (was.dragged) {
      if (want) want.until = now + 150;
      hold();
      return;
    }
    const touch = e.pointerType === 'touch';
    if (!onPlate(e) || (touch && now - was.at >= 400) || Math.hypot(e.clientX - was.x, e.clientY - was.y) >= 10) return;
    // The second of two in one place is the browser's (it selects).
    const second = now - tapped.at < 350 && Math.hypot(e.clientX - tapped.x, e.clientY - tapped.y) < 25;
    tapped = { at: now, x: e.clientX, y: e.clientY };
    if (second) {
      want = null;
      return;
    }
    const gap = gapOf(e);
    // A click or tap ends a selection, which the phone would leave be.
    document.getElementById('from-vin').setSelectionRange(gap, gap);
    want = { from: gap, to: gap, back: false, until: now + (touch ? 1000 : 150) };
    hold();
  }, true);
  // hold keeps the selection where the pointer put it: a caret only while
  // the browser shows a caret (a selection it made is its own), a drag's
  // cells whatever it shows.
  const hold = () => {
    const input = document.getElementById('from-vin');
    if (!want || !input || performance.now() > want.until) return;
    const caret = want.from === want.to;
    const same = input.selectionStart === want.from && input.selectionEnd === want.to;
    if (!same && (!caret || input.selectionStart === input.selectionEnd)) {
      input.setSelectionRange(want.from, want.to, want.back ? 'backward' : 'forward');
    }
    show();
  };
  document.addEventListener('selectionchange', hold);
  document.addEventListener('click', (e) => {
    if (e.target.id === 'from-vin') hold();
  });
  document.addEventListener('focusin', (e) => {
    const input = e.target;
    if (input.id !== 'from-vin') return;
    const n = input.value.length;
    if (performance.now() - touched > 700 && input.selectionStart === input.selectionEnd) input.setSelectionRange(n, n);
    show();
  });
  document.addEventListener('focusout', (e) => {
    if (e.target.id === 'from-vin') setTimeout(show);
  });
  document.addEventListener('selectionchange', show);
  document.addEventListener('select', show);

  // Under a mouse, the gap a click would put the caret in is marked: a
  // click anywhere in a cell's nearer half picks it, which the mark shows.
  const ghost = (at) => {
    document.querySelectorAll('#vin-form .cell').forEach((cell, i) => {
      cell.classList.toggle('ghost', i === at);
      cell.classList.toggle('ghost-end', at === LENGTH && i === LENGTH - 1);
    });
  };
  document.addEventListener('pointermove', (e) => {
    const input = e.target;
    if (input.id !== 'from-vin' || e.pointerType !== 'mouse' || e.buttons || !input.closest('.live')) return ghost(-1);
    const at = gapAt(e.clientX, e.clientY, input.value.length);
    const here = document.activeElement === input && input.selectionStart === at && input.selectionEnd === at;
    ghost(here ? -1 : at);
  });
  document.addEventListener('pointerdown', () => ghost(-1));

  window.addEventListener('popstate', () => {
    const vin = clean(new URLSearchParams(location.search).get('from.vin') || '');
    const input = document.getElementById('from-vin');
    if (!input) return;
    // A move within the page (a link to a part of it, as "Check years"
    // is) is no other VIN: the answer stays, and what the link points at
    // stays in it.
    const answer = document.getElementById('answer');
    if (input.value === vin && answer && answer.dataset.vin === vin) return;
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
    // The field over the cells is a textarea in the input's place: its
    // text wraps with the cells when they are in two rows. Without this
    // script the input stays, and Enter sends it.
    let input = document.getElementById('from-vin');
    if (input.tagName === 'INPUT') {
      const area = document.createElement('textarea');
      for (const name of ['id', 'name', 'placeholder', 'autocomplete', 'autocapitalize', 'spellcheck', 'aria-describedby']) {
        if (input.hasAttribute(name)) area.setAttribute(name, input.getAttribute(name));
      }
      area.rows = 1;
      area.enterKeyHint = 'go';
      area.value = input.value;
      const focus = document.activeElement === input || input.autofocus;
      input.replaceWith(area);
      if (focus) area.focus({ preventScroll: true });
      input = area;
    }
    form.classList.add('live');
    paint(clean(input.value), false);
    // The page opens with the caret after what is there, not before it;
    // again once the browser has put back what a reload restores.
    const toEnd = () => {
      if (input.selectionStart === input.selectionEnd) input.setSelectionRange(input.value.length, input.value.length);
      show();
    };
    toEnd();
    window.addEventListener('pageshow', () => setTimeout(toEnd), { once: true });
    if (!history.state) history.replaceState({ whole: input.value.length === LENGTH }, '', location.href);
    chips();
    remember(document.getElementById('answer'));
  };
  init();
  document.addEventListener('htmx:afterSettle', (e) => {
    if (e.detail.target && e.detail.target.id === 'ui-main') init();
  });
})();
