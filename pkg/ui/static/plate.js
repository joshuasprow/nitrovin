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
// - a part type pressed among the parts that fit, a link to the page, is
//   asked for and swapped in the same way, a history entry of its own;
// - a group whose characters were just completed is marked fresh, which
//   the stylesheet draws in;
// - the plate edits as a text field does, because it is one: the input's
//   characters are spaced a cell apart (ui.css), so its own caret and
//   selection, by click, drag, touch or key, fall on the cells, which
//   draw them. A whole VIN takes no more: nothing is pushed off its end;
// - whole VINs that decoded are kept in this browser (the last 8) and
//   offered as chips in the examples' place;
// - the part types a page pressed are kept too, and lead the strip of
//   part types, which the visitor can filter as they type (strip);
// - the plate's switch (page.html's modes), two links, swaps in the other
//   mode's plate and answer in place, a history entry of its own; in
//   part-number mode the page asks as the number is typed, as it does for
//   a VIN, keeps the part type chosen in the dialog, and puts the caret
//   back in the number, ready for the next.
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
  // partMode is whether q, a query, asks for part-number mode: any of the
  // fitment key's parameters (part.go).
  const partMode = (q) => ['from.fitment.type', 'from.fitment.number', 'from.fitment.side'].some((p) => q.has(p));
  // partURL is the page with the part form's part type and number: the
  // type first, as the form sends it, and the number always, so the URL
  // stays in the mode.
  const partURL = () => {
    const q = new URLSearchParams();
    const type = document.querySelector('#part-form input[name="from.fitment.type"]');
    if (type && type.value) q.set('from.fitment.type', type.value);
    q.set('from.fitment.number', document.getElementById('from-number').value.trim());
    return '/?' + q;
  };

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
  // An answer in the other mode, or one a mode key asked for (plate),
  // brings its plate and switch (#lookup); one in part-number mode, the
  // part type on the plate, which the answer's dialog chose. The number
  // on the plate is the visitor's, and stays.
  const swap = (doc, plate) => {
    const old = document.getElementById('answer');
    const answer = doc.getElementById('answer');
    if (!old || !answer) return;
    const lookup = document.getElementById('lookup');
    const drawn = doc.getElementById('lookup');
    const moved = !!(lookup && drawn) && (plate || !!lookup.querySelector('#part-form') !== !!drawn.querySelector('#part-form'));
    const field = document.getElementById('part-type-field');
    const chosen = doc.getElementById('part-type-field');
    if (moved) lookup.replaceWith(drawn);
    else if (field && chosen) field.replaceWith(chosen);
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
    parts(answer);
    if (moved) init();
  };

  let timer = 0;
  let pending = null;
  // ask asks the server for url's answer, the page with a VIN and perhaps
  // a part type pressed; how is the history's part: 'push', 'replace' or
  // '' to leave it alone; plate, whether the answer's plate replaces the
  // page's (swap).
  const ask = async (url, how, plate) => {
    clearTimeout(timer);
    if (pending) pending.abort();
    const mine = pending = new AbortController();
    let html;
    try {
      const r = await fetch(url, { headers: { 'HX-Request': 'true' }, signal: mine.signal });
      html = await r.text();
    } catch (err) {
      if (err.name === 'AbortError') return;
      throw err;
    }
    if (pending === mine) pending = null;
    swap(new DOMParser().parseFromString(html, 'text/html'), plate);
    const whole = (new URL(url, location.href).searchParams.get('from.vin') || '').length === LENGTH;
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
    timer = setTimeout(() => ask(urlFor(vin), how(vin)).catch(console.error), vin.length === LENGTH ? 0 : WAIT);
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
    ask(urlFor(vin), how(vin)).catch(console.error);
  });

  // Part-number mode: the number is asked for as it is typed, each answer
  // in the same history entry, and Look up (Enter) makes it one of its
  // own. The type goes with it, the one the plate holds.
  document.addEventListener('input', (e) => {
    if (e.target.id !== 'from-number') return;
    clearTimeout(timer);
    timer = setTimeout(() => ask(partURL(), 'replace').catch(console.error), WAIT);
  });
  document.addEventListener('submit', (e) => {
    if (e.target.id !== 'part-form') return;
    e.preventDefault();
    ask(partURL(), 'push').catch(console.error);
  });

  // The switch: the page in a mode with nothing on its plate, the plate
  // and all swapped in place, whichever mode the page was in, and its
  // field focused, as the page's own load focuses it.
  document.addEventListener('click', (e) => {
    const a = e.target.closest && e.target.closest('.modes a.mode');
    if (!a || e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    e.preventDefault();
    const u = new URL(a.href);
    ask(u.pathname + u.search, 'push', true).then(() => {
      const field = document.getElementById('from-number') || document.getElementById('from-vin');
      if (field) field.focus({ preventScroll: true });
    }).catch(console.error);
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

  // A part type pressed, the next page of its candidates, or a
  // candidate's stock strip, is a link to the page (page.html's fit): its
  // answer is swapped in place, as a VIN's is, and is a history entry of
  // its own. The link the visitor followed keeps the focus, in the answer
  // that replaced it: a part type, chosen in the picker's list or under the
  // picker, leaves it on the picker; anything else the element of its id, else the
  // link in its place. "Show all", which the answer it brings has no
  // more, leaves the focus on what it names (data-focus), the rows it
  // showed. In part-number mode a part type chosen in the dialog fills
  // the plate's field, and the focus is the number's, its caret at the
  // end, for the next number.
  // reveal scrolls the page down just far enough to show what a press
  // opened, when it opened below the window's bottom edge: the candidates
  // under a pressed part type, the rows under a candidate's stock strip.
  // It never scrolls what was pressed (from) off the top, so the part type
  // or the stock strip stays in sight above its answer, and it never
  // scrolls up. Following the link's focus would do as much for a
  // keyboard; a pointer has no focus to follow.
  const reveal = (what, from) => {
    if (!what) return;
    const pad = 16;
    const over = what.getBoundingClientRect().bottom + pad - window.innerHeight;
    const room = (from || what).getBoundingClientRect().top - pad;
    const by = Math.min(over, room);
    if (by > 0) window.scrollBy({ top: by, behavior: matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth' });
  };

  document.addEventListener('click', (e) => {
    const a = e.target.closest && e.target.closest('#answer a.swap');
    if (!a || e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    e.preventDefault();
    // The pressed part type, pressed again in the picker's list: the list
    // shuts and the page stays as it is (#366).
    if (a.matches('#fit .ptdialog .ptype[aria-current]')) {
      a.closest('[popover]').hidePopover();
      document.getElementById('pick-type').focus({ preventScroll: true });
      return;
    }
    const part = a.classList.contains('ptype') && a.parentElement.dataset.part;
    const i = [...document.querySelectorAll('#answer a.swap:not(.ptype)')].indexOf(a);
    const focused = document.activeElement === a;
    const id = a.dataset.focus || a.id;
    const dialog = a.closest('[popover]');
    if (dialog) dialog.hidePopover();
    const opens = a.classList.contains('stk') && a.getAttribute('aria-expanded') === 'false';
    // On a phone the candidates are a row that scrolls sideways: the
    // answer that replaces it keeps its place, and shows the candidate
    // whose rows it opened.
    const row = a.closest('.cands');
    const left = row ? row.scrollLeft : 0;
    const u = new URL(a.href);
    ask(u.pathname + u.search, 'push').then(() => {
      const cands = row && document.querySelector('#answer .cands');
      if (cands) {
        cands.scrollLeft = left;
        const open = cands.querySelector('.cand:has(+ .rows)');
        if (open) open.scrollIntoView({ block: 'nearest', inline: 'nearest' });
      }
      // What the press opened is shown: a stock strip's rows under it, a
      // part type's candidates, or the sentence in their place, under it.
      // The stock strip stays in sight, not its whole tag: the rows are a
      // strip of paper as tall as its rows are many, and its head says
      // which part it is of.
      const fit = document.getElementById('fit');
      if (opens) {
        const strip = id && document.getElementById(id);
        reveal(document.getElementById('stock-rows'), strip);
      } else if (part && fit && fit.querySelector('.ptype[aria-current]')) {
        reveal(fit.querySelector('.cands') || fit.querySelector('p.blank'), fit.querySelector('.ptwrap'));
      }
      const number = part && document.getElementById('from-number');
      if (number) {
        number.focus({ preventScroll: true });
        number.setSelectionRange(number.value.length, number.value.length);
        return;
      }
      const again = part ? document.getElementById('pick-type') : (id && document.getElementById(id)) || document.querySelectorAll('#answer a.swap:not(.ptype)')[i];
      if ((focused || dialog) && again) again.focus({ preventScroll: true });
    }).catch(console.error);
  });

  window.addEventListener('popstate', () => {
    const q = new URLSearchParams(location.search);
    const answer = document.getElementById('answer');
    const inPart = !!document.getElementById('part-form');
    // The other mode's entry brings its own plate (swap). One of part-
    // number mode is the page's own unless its query is another: the
    // number on the plate is the entry's.
    if (partMode(q) !== inPart) {
      ask(location.pathname + location.search, '').catch(console.error);
      return;
    }
    if (inPart) {
      if (answer && answer.dataset.q === location.search.slice(1)) return;
      document.getElementById('from-number').value = q.get('from.fitment.number') || '';
      ask(location.pathname + location.search, '').catch(console.error);
      return;
    }
    const vin = clean(q.get('from.vin') || '');
    const input = document.getElementById('from-vin');
    if (!input) return;
    // A move within the page (a link to a part of it, as "Check years"
    // is) is no other VIN, part type nor open candidate: the answer stays,
    // and what the link points at stays in it.
    const same = (key, param) => (answer.dataset[key] || '') === (q.get(param) || '');
    if (input.value === vin && answer && answer.dataset.vin === vin && same('part', 'part.id') && same('after', 'after') &&
      same('type', 'fitment.type') && same('number', 'fitment.number') && same('side', 'fitment.side') && same('rows', 'rows')) return;
    input.value = vin;
    paint(vin, false);
    ask(location.pathname + location.search, '').catch(console.error);
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
    document.getElementById('clear-recent').hidden = kept.length === 0 && recentParts().length === 0;
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
    try {
      localStorage.removeItem(STORE);
      localStorage.removeItem(PARTS);
    } catch {
      // no storage: nothing was kept
    }
    chips();
    strip(document.getElementById('answer'));
  });

  // The part types (page.html's fit, #366; part-number mode's dialog,
  // #325). Without this script a VIN's page has the picker, which opens
  // the part types dialog as its list, every part type under its headings.
  // With it:
  //
  // - a part type a page presses is kept in this browser (the last 8,
  //   newest first, by its spine). Up to two the vehicle has, the
  //   pressed one aside, sit beside the picker on its tape where the page
  //   is wide enough to show them (ui.css), in the order they were
  //   opened; the list, and part-number mode's dialog, lead with every
  //   one it has under "Opened recently", whose Clear recent forgets
  //   them, as the VINs menu's forgets them and the VINs;
  // - the filter narrows the list as it is typed, by name, hiding a
  //   heading left with none;
  // - pressing the pressed part type in the list shuts the list and
  //   changes nothing: the key beside the picker clears it.
  //
  // Nothing is kept when storage fails.
  const PARTS = 'nitrovin.recent-parts';
  const UNDER = 2;
  const recentParts = () => {
    try {
      return JSON.parse(localStorage.getItem(PARTS) || '[]').filter((r) => r && r.spine && r.id).slice(0, KEPT);
    } catch {
      return [];
    }
  };
  // listed are the part types answer lists, its dialog's, and the spine
  // they are kept by: the picker's, or, in part-number mode, where the
  // dialog stands alone, the dialog's.
  const listed = (answer) => {
    const dialog = answer && answer.querySelector('.ptdialog');
    if (!dialog) return null;
    const wrap = answer.querySelector('.ptwrap');
    const spine = wrap ? wrap.dataset.spine : dialog.dataset.spine;
    return spine ? { wrap, dialog, spine, items: [...dialog.querySelectorAll('.ptgroups li[data-part]')] } : null;
  };
  // keep keeps the part type answer presses, when its list has it, or
  // the one the plate holds, in part-number mode.
  const keep = (answer) => {
    const list = listed(answer);
    const id = answer && answer.dataset.part;
    if (!list || !id || !list.items.some((li) => li.dataset.part === id)) return;
    const spine = list.spine;
    const kept = [{ spine, id }, ...recentParts().filter((r) => r.spine !== spine || r.id !== id)].slice(0, KEPT);
    try {
      localStorage.setItem(PARTS, JSON.stringify(kept));
    } catch {
      return; // no storage: no recent part types
    }
    chips();
  };
  const nameOf = (li) => li.querySelector('.ptype .nm').textContent.toLowerCase();
  // filter narrows the dialog's list to the names holding what input has,
  // and hides a heading left with none.
  const filter = (input) => {
    const q = input.value.trim().toLowerCase();
    const scope = input.closest('.ptdialog');
    let found = 0;
    for (const li of scope.querySelectorAll('.ptlist > li')) {
      const match = nameOf(li).includes(q);
      if (match) found++;
      li.hidden = !match;
    }
    for (const sec of scope.querySelectorAll('.ptgroup')) sec.hidden = !sec.querySelector('.ptlist > li:not([hidden])');
    const none = scope.querySelector('.ptnone');
    none.hidden = found > 0;
    none.textContent = found ? '' : 'No part type’s name has “' + input.value.trim() + '”.';
  };
  document.addEventListener('input', (e) => {
    if (e.target.classList && e.target.classList.contains('ptfind')) filter(e.target);
  });
  // strip draws answer's part types as this browser has them: the filter
  // shown, and the recent part types under the picker and in the dialog's
  // "Opened recently".
  function strip(answer) {
    const list = listed(answer);
    if (!list) return;
    const { wrap, dialog, items } = list;
    answer.querySelectorAll('.ptf').forEach((f) => { f.hidden = false; });
    const has = new Set(items.map((li) => li.dataset.part));
    const recent = recentParts().filter((r) => r.spine === list.spine && has.has(r.id)).map((r) => r.id);
    const copy = (id) => dialog.querySelector('.ptgroups li[data-part="' + CSS.escape(id) + '"]').cloneNode(true);
    if (wrap) {
      wrap.classList.add('live');
      const under = recent.filter((id) => id !== answer.dataset.part).slice(0, UNDER);
      const row = wrap.querySelector('.ptrecents');
      row.replaceChildren(...under.map(copy));
      row.hidden = under.length === 0;
    }
    const sec = dialog.querySelector('.ptrecent');
    sec.querySelector('.ptlist').replaceChildren(...recent.map(copy));
    sec.hidden = recent.length === 0;
    for (const input of answer.querySelectorAll('.ptfind')) if (input.value) filter(input);
  }
  const parts = (answer) => {
    keep(answer);
    strip(answer);
  };
  document.addEventListener('click', (e) => {
    if (!e.target.classList || !e.target.classList.contains('clear-parts')) return;
    try {
      localStorage.removeItem(PARTS);
    } catch {
      // no storage: nothing was kept
    }
    chips();
    strip(document.getElementById('answer'));
    // The button's section is gone with what it listed: the focus stays
    // in the dialog, on its filter.
    const find = e.target.closest('.ptdialog')?.querySelector('.ptfind');
    if (find) find.focus();
  });

  const init = () => {
    // Part-number mode: its field is the form's own text field, struck
    // into the plate, and needs only the page's listeners.
    const part = document.getElementById('part-form');
    if (part) {
      part.classList.add('live');
      const number = document.getElementById('from-number');
      if (document.activeElement === number) number.setSelectionRange(number.value.length, number.value.length);
      if (!history.state) history.replaceState({ whole: false }, '', location.href);
      chips();
      parts(document.getElementById('answer'));
      return;
    }
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
    parts(document.getElementById('answer'));
  };
  init();
  document.addEventListener('htmx:afterSettle', (e) => {
    if (e.detail.target && e.detail.target.id === 'ui-main') init();
  });
})();
