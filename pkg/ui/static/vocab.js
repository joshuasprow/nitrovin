// The page's browser files (ADR 0013), behind its Vocabularies button:
// registers the Service Worker (worker.js), says what state it is in,
// swaps the worker's panel (the merged catalog, with the browser's files)
// over the server's catalog in the dialog, and hands the files the visitor
// picks or drops, EPA's vocabulary on a click and the demo set on another,
// to the worker by postMessage. A catalog picked on its own, and the demo
// set's, is compiled in a dedicated worker it starts (compile.js, ADR
// 0023), the File handed over and read in place. Without this script the
// dialog says that loading files needs JavaScript; everything else works.
//
// The worker is the module's 12 MB, so a visitor who never opens the
// dialog never gets it: it is registered when the dialog first opens or a
// file is dropped on the page, and started at once only in a browser that
// has it already.
//
// States, on #browser's data-state and in words in #browser-status:
// none before the worker is asked for, then starting, ready (the worker
// controls the page and its panel is in #browser-files), paused (an
// installed worker does not control this page: a hard reload bypasses
// it), outdated (the worker in control is of another release than the
// page, #browser's data-release, or the server, or does not answer the
// page at all: it answers nothing, so every page is the server's and the
// visitor's files wait for the update), unsupported (no Service Workers),
// and failed.
//
// A new release is offered in the header, outside the dialog: Update now
// (#update-now, ui.html) shows while the document's data-update is set.
// "waiting" is a new release's worker waiting for the old one's pages to
// close, said beside the state too; Update now hands the page to it at
// once and reloads. "server" is a server that runs another release than
// the one that drew this page, with no worker waiting: Update now
// reloads. The page learns the server's release from every answer htmx
// gets, and asks /sw.js for it when the tab is shown again, five minutes
// apart at most, so a tab left open hears of a deploy without a worker.
// An outdated state is not final: a rollout's old container may answer
// once after the new one. The page asks the worker again every fifteen
// seconds, the worker asks the server, and the files come back without a
// reload once the releases agree.
'use strict';

(() => {
  const section = document.getElementById('browser');
  if (!section || !section.dataset.worker) return;
  const dialog = document.getElementById('vocabularies');
  const status = document.getElementById('browser-status');
  // release is the release that drew the page, which the worker in
  // control must be of.
  const release = section.dataset.release || '';
  const version = (r) => r.split('/')[0];
  // stall is how long a load may go without a word from the worker: a
  // minute, or #browser's data-stall (milliseconds), which a test sets.
  const stall = () => Number(section.dataset.stall) || 60000;
  let update = '';

  const say = (state, text) => {
    section.dataset.state = state;
    status.textContent = text + update;
  };
  // tell says how a load went, in #load-status, which no swap replaces.
  const tell = (text) => {
    document.getElementById('load-status').textContent = text;
  };

  if (!('serviceWorker' in navigator)) {
    say('unsupported', 'No Service Workers in this browser: no own files');
    return;
  }
  const sw = navigator.serviceWorker;
  let registration = null;

  // ask asks the worker in control msg on a MessageChannel, resolving with
  // its answer, or with null after ms without one: a worker of another
  // release may not know msg's op.
  const ask = (msg, ms, to = sw.controller) => new Promise((resolve) => {
    const ch = new MessageChannel();
    const timer = setTimeout(() => resolve(null), ms);
    ch.port1.onmessage = (e) => {
      clearTimeout(timer);
      resolve(e.data);
    };
    to.postMessage(msg, [ch.port2]);
  });

  const showFiles = () => htmx.ajax('GET', '/ui/files', { target: '#browser-files', swap: 'innerHTML' });
  // redraw asks for the page's main again: its answer is over the files
  // loaded, which just changed.
  const redraw = () => htmx.ajax('GET', location.pathname + location.search, { target: '#ui-main', swap: 'innerHTML' });

  // offer shows Update now in the header, or takes it back: waiting is a
  // new release's worker installed beside the one in control, heard the
  // release the server last named.
  const root = document.documentElement;
  const updateNow = document.getElementById('update-now');
  let waiting = false, heard = release;
  const offer = () => {
    const why = waiting ? 'waiting' : heard !== release ? 'server' : '';
    if (why) root.dataset.update = why;
    else delete root.dataset.update;
  };
  const hear = (r) => {
    if (!r || r === heard) return;
    heard = r;
    offer();
  };
  document.addEventListener('htmx:afterRequest', (e) => {
    if (e.detail.xhr) hear(e.detail.xhr.getResponseHeader('Nitrovin-Release'));
  });
  // A tab shown again asks the server its release: no worker needed.
  let checked = Date.now();
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState !== 'visible' || Date.now() - checked < (Number(section.dataset.recheck) || 300000)) return;
    checked = Date.now();
    fetch(section.dataset.worker, { method: 'HEAD', cache: 'no-store' }).then((r) => hear(r.headers.get('Nitrovin-Release')), () => {});
  });

  // Update now hands the page to the waiting worker at once (it skips
  // waiting) and reloads, once the new worker controls the page; another
  // page's Update now reloads this one too, once it was the old worker's.
  // A worker still installing is waited for, up to fifteen seconds, so
  // one click is the whole update. With none, or none that takes over in
  // five seconds, the page reloads: the server's page is the new release.
  let controlled = Boolean(sw.controller), updating = false;
  sw.addEventListener('controllerchange', () => {
    if (controlled || updating) location.reload();
    controlled = true;
  });
  const installed = (w, ms) => new Promise((resolve) => {
    if (w.state !== 'installing') return resolve();
    setTimeout(resolve, ms);
    w.addEventListener('statechange', () => { if (w.state !== 'installing') resolve(); });
  });
  if (updateNow) updateNow.addEventListener('click', async () => {
    updating = true;
    updateNow.disabled = true;
    updateNow.textContent = 'Updating…';
    const reg = registration || await sw.getRegistration('/').catch(() => null);
    if (reg && reg.installing) await installed(reg.installing, 15000);
    // None waits: another page took the update already, or the page alone
    // is old.
    if (!reg || !reg.waiting) return location.reload();
    reg.waiting.postMessage({ op: 'skipWaiting' });
    setTimeout(() => location.reload(), 5000);
  });

  // judge decides whether the worker waiting is an update worth
  // offering: it is of the release the server runs now, asked of /sw.js,
  // and of another than the worker in control. A worker installed from a
  // rollout's old container is neither: the browser is asked to look
  // again, and what it finds is judged in turn. A worker that does not
  // say its release speaks an older protocol, and is replaced.
  let judging = 0;
  const judge = async () => {
    const n = ++judging, reg = registration, w = reg && reg.waiting;
    const set = (on) => {
      waiting = on;
      update = on ? ' · New release ready' : '';
      status.textContent = status.textContent.replace(/ · New release.*$/, '') + update;
      offer();
    };
    if (!w || !sw.controller) return set(false);
    const [theirs, ours, server] = await Promise.all([
      ask({ op: 'release' }, 3000, w),
      mine(),
      fetch(section.dataset.worker, { method: 'HEAD', cache: 'no-store' }).then((r) => r.headers.get('Nitrovin-Release'), () => null),
    ]);
    if (n !== judging) return;
    hear(server);
    if (theirs && theirs.release !== heard) {
      set(false);
      reg.update().catch(() => {});
      return;
    }
    set(!(theirs && ours && theirs.release === ours.release));
  };
  const watchUpdates = (reg) => {
    // track waits for w, a worker installing, to be installed.
    const track = (w) => {
      if (w) w.addEventListener('statechange', () => { if (w.state === 'installed') judge(); });
    };
    judge();
    track(reg.installing);
    reg.addEventListener('updatefound', () => track(reg.installing));
  };

  // outdated says the worker in control answers nothing, mine its answer
  // to {op: "release"} ({release, server}), null when it gave none, and
  // has the browser look for the new release's worker. Until one takes
  // over it asks the worker again (retry), which asks the server: the
  // releases may agree after all.
  let retrying = 0;
  const outdated = (mine) => {
    const server = mine && mine.server && mine.server.includes('/') ? mine.server : release;
    const what = mine && version(mine.release) !== version(server) ? version(mine.release) + ' here, ' + version(server) + ' on the server' : 'another release here';
    say('outdated', 'Own files paused: ' + what);
    if (mine && mine.server) hear(mine.server.includes('/') ? mine.server : '');
    if (registration && !registration.waiting && !registration.installing) registration.update().catch(() => {});
    if (!retrying) retrying = setInterval(retry, Number(section.dataset.retry) || 15000);
  };
  const retry = async () => {
    if (section.dataset.state !== 'outdated' || updating) return;
    asked = null;
    const m = sw.controller && await mine();
    if (matched(m)) resume();
    else if (registration && !registration.waiting && !registration.installing) registration.update().catch(() => {});
  };
  // resume is the way back from outdated: the worker in control and the
  // server run the page's release after all. The files show again, and
  // the page's main is drawn over them.
  const resume = async () => {
    if (section.dataset.state !== 'outdated') return;
    clearInterval(retrying);
    retrying = 0;
    heard = release;
    judge();
    section.dataset.state = 'starting';
    try {
      await showFiles();
      say('ready', 'Files stay in this browser');
      redraw();
    } catch (err) {
      say('failed', 'The translator in this browser could not start: ' + err + '.');
    }
  };
  // The worker in control tells the page once it learns the server runs
  // another release, and once it runs the worker's again.
  sw.addEventListener('message', (e) => {
    const op = e.data && e.data.op;
    if (op === 'outdated' && section.dataset.state !== 'outdated') {
      // Before the worker was asked for, only the header says so.
      if (section.dataset.state && section.dataset.state !== 'none') outdated(e.data);
      else hear(e.data.server && e.data.server.includes('/') ? e.data.server : '');
    }
    if (op === 'current' && e.data.release === release) {
      asked = null;
      resume();
    }
  });
  sw.startMessages();

  // mine is the worker in control's answer to {op: "release"}, asked once
  // a page ({release, server}; null when it gave none, asked again next
  // time); matched reports whether that worker is of the page's release
  // and not outdated.
  let asked = null;
  const mine = () => asked || (asked = ask({ op: 'release' }, 5000).then((m) => {
    if (!m) asked = null;
    return m;
  }));
  const matched = (m) => Boolean(m) && !m.server && m.release === release;

  // start registers the worker, once, and settles when its state is said.
  let starting = null;
  const start = () => {
    if (!starting) {
      starting = (async () => {
        say('starting', 'Starting…');
        const before = await sw.getRegistration('/');
        if (before && before.active && !sw.controller) {
          // Installed, yet not in control: this load bypassed it (a hard reload).
          registration = before;
          watchUpdates(before);
          say('paused', 'Own files paused by a hard reload: reload normally');
          return;
        }
        const reg = await sw.register(section.dataset.worker, { scope: '/' });
        registration = reg;
        watchUpdates(reg);
        if (!sw.controller) await new Promise((resolve) => sw.addEventListener('controllerchange', resolve, { once: true }));
        const m = await mine();
        if (!matched(m)) {
          outdated(m);
          return;
        }
        await showFiles();
        say('ready', 'Files stay in this browser');
        // The panel settled before the page was ready, so what a compile
        // not finished left is dropped and said now.
        usage().catch(() => {});
      })().catch((err) => say('failed', 'The translator in this browser could not start: ' + err + '.'));
    }
    return starting;
  };
  sw.getRegistration('/').then((reg) => { if (reg) start(); });
  dialog.addEventListener('toggle', (e) => { if (e.newState === 'open') start(); });

  const ready = () => section.dataset.state === 'ready' && sw.controller;
  // why is why files cannot load now, by the state.
  const why = () => ({
    outdated: 'the translator in this browser is of another release of nitrovin than this server\'s. Update it (Update now, or close every nitrovin tab), then load them again.',
    paused: 'a hard reload bypassed the translator in this browser for this page. Reload the page normally, then load them again.',
    failed: 'the translator in this browser could not start. Reload the page, then load them again.',
  })[section.dataset.state] || 'files load once the translator in this browser is ready.';

  // A file removed changes what the page's answer is over.
  document.addEventListener('htmx:afterRequest', (e) => {
    if (e.detail.successful && e.detail.pathInfo.requestPath === '/ui/files/remove') redraw();
  });

  // usage says, below the panel, how much this site keeps in the browser
  // and whether the browser may clear it; and what a compile not finished
  // keeps, which the worker drops (left) whenever no compile runs: one
  // running in another tab, or this one, holds it until it is done.
  const usage = async () => {
    const line = document.getElementById('storage-usage');
    if (!line || !navigator.storage || !navigator.storage.estimate) return;
    const left = ready() && !compiling ? await ask({ op: 'left' }, 5000) : null;
    const { usage: used, quota } = await navigator.storage.estimate();
    const kept = navigator.storage.persisted && await navigator.storage.persisted();
    line.textContent = 'This site keeps ' + (used / 1e6).toFixed(1) + ' MB in this browser, of the ' + Math.round(quota / 1e6).toLocaleString('en') +
      ' MB the browser allows it' + (kept ? '; the browser keeps it until you remove it.' : '; the browser may clear it when it runs short of space.') +
      (left && left.bytes > 0 ? ' A compile not finished keeps ' + (left.bytes / 1e6).toFixed(1) + ' MB of it, dropped once no compile runs.' : '');
    section.dataset.leftBytes = left ? String(left.bytes) : '';
  };
  document.addEventListener('htmx:afterSettle', () => { usage().catch(() => {}); });

  // load hands files to the worker as one batch: [{name, read}], read
  // giving each file's bytes, which go by postMessage (transferred, never
  // copied to a request). what names the batch in the status line. It
  // resolves with how the load went, {ok, message}.
  const load = async (what, picked) => {
    if (navigator.storage && navigator.storage.persist) navigator.storage.persist().catch(() => {});
    tell('Reading ' + what + '…');
    const files = await Promise.all(picked.map(async (f) => ({ name: f.name, data: await f.read() })));
    const done = await new Promise((resolve) => {
      const ch = new MessageChannel();
      let timer;
      const wait = () => {
        clearTimeout(timer);
        timer = setTimeout(() => resolve({ ok: false, message: 'The load stalled: no word from the translator in ' + stall() / 1000 + ' s. Nothing was stored; try again.' }), stall());
      };
      ch.port1.onmessage = (e) => {
        if (e.data.done) {
          clearTimeout(timer);
          resolve(e.data);
          return;
        }
        tell(e.data.progress);
        wait();
      };
      wait();
      sw.controller.postMessage({ op: 'load', files }, [ch.port2, ...files.map((f) => f.data)]);
    });
    if (ready()) await showFiles();
    tell(done.ok ? done.message : 'Nothing was loaded: ' + done.message);
    if (done.ok) await redraw();
    return done;
  };
  const failed = (err) => tell('Nothing was loaded: ' + (err && err.message ? err.message : err));

  // fromDisk is a picked or dropped file to load.
  const fromDisk = (f) => ({
    name: f.name,
    read: () => f.arrayBuffer().catch((err) => { throw new Error(f.name + ' could not be read (' + err.message + ').'); }),
  });
  // kindOf is the kind f's header, its first line that is not blank,
  // says, read from f's first bytes alone: a catalog may be hundreds of
  // MB, never read into the page.
  const kindOf = async (f) => {
    const line = (await f.slice(0, 1 << 16).text()).split('\n').find((l) => l.trim() !== '');
    try {
      return (line && JSON.parse(line).kind) || '';
    } catch {
      return '';
    }
  };

  // compiling is true while a catalog compiles in this page: leaving it
  // stops the compile, so the browser asks first (stay).
  let compiling = false;
  const stay = (e) => e.preventDefault();

  // compile compiles file, a catalog's JSON Lines, as this browser's
  // catalog (ADR 0023), in a dedicated worker this page starts
  // (compile.js), which reads the File in place: the page says how far it
  // is and that the tab must stay open. Once it is stored the Service
  // Worker opens the files again. It resolves with how the compile went,
  // {ok, message}.
  const compile = async (file) => {
    if (navigator.storage && navigator.storage.persist) navigator.storage.persist().catch(() => {});
    const keep = ' Keep this tab open until it is done.';
    tell('Compiling ' + file.name + '…' + keep);
    const c = await ask({ op: 'compiler' }, 5000);
    if (!c || c.refused) {
      const message = c ? c.refused : 'the translator in this browser did not answer. Reload the page, then try again.';
      tell('Nothing was compiled: ' + message);
      return { ok: false, message };
    }
    compiling = true;
    window.addEventListener('beforeunload', stay);
    const worker = new Worker(c.config.assets + '/compile.js');
    let done;
    try {
      done = await new Promise((resolve) => {
        worker.onmessage = (e) => {
          if (e.data.done) resolve(e.data);
          else tell(e.data.progress + keep);
        };
        worker.onerror = (e) => resolve({ ok: false, message: 'the compiler in this browser stopped: ' + (e.message || 'an error') + '.' });
        worker.postMessage({ op: 'compile', config: c.config, file });
      });
    } finally {
      compiling = false;
      window.removeEventListener('beforeunload', stay);
      worker.terminate();
    }
    section.dataset.compileMemory = done.memory || '';
    if (done.ok) await ask({ op: 'catalog' }, 5000);
    if (ready()) await showFiles();
    tell(done.ok ? done.message : 'Nothing was compiled: ' + done.message);
    if (done.ok) await redraw();
    return done;
  };

  // loadFiles loads the files of list in the dialog, where a load is
  // told, once the worker is ready. A catalog picked on its own is
  // compiled instead.
  const loadFiles = async (list) => {
    const picked = [...list];
    if (picked.length === 0) return;
    if (!dialog.matches(':popover-open')) dialog.showPopover();
    await start();
    if (!ready()) {
      tell('Nothing was loaded: ' + why());
      return;
    }
    if (compiling) {
      tell('Nothing was loaded: a catalog is compiling in this tab. Wait until it is done, then load them again.');
      return;
    }
    const what = picked.map((f) => f.name).join(', ');
    tell('Reading ' + what + '…');
    const kinds = await Promise.all(picked.map(kindOf));
    if (kinds.includes('catalog')) {
      if (picked.length > 1) {
        tell('Nothing was loaded: ' + picked[kinds.indexOf('catalog')].name + ' is a catalog. Pick it on its own: a catalog is compiled, not loaded with other files.');
        return;
      }
      await compile(picked[0]);
      return;
    }
    await load(what, picked.map(fromDisk));
  };

  document.addEventListener('change', (e) => {
    if (e.target.id === 'vocab-files') loadFiles(e.target.files).catch(failed);
  });

  // Files dropped anywhere on the page load as if picked: some browsers
  // (embedded ones among them) open no dialog for a file input.
  const carriesFiles = (e) => e.dataTransfer && [...e.dataTransfer.types].includes('Files');
  let over = 0;
  document.addEventListener('dragenter', (e) => {
    if (!carriesFiles(e)) return;
    over++;
    section.classList.add('dropping');
  });
  document.addEventListener('dragleave', (e) => {
    if (carriesFiles(e) && --over <= 0) {
      over = 0;
      section.classList.remove('dropping');
    }
  });
  document.addEventListener('dragover', (e) => {
    if (!carriesFiles(e)) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = 'copy';
  });
  document.addEventListener('drop', (e) => {
    if (!carriesFiles(e)) return;
    e.preventDefault();
    over = 0;
    section.classList.remove('dropping');
    loadFiles(e.dataTransfer.files).catch(failed);
  });

  // Download: the compiled catalog is the worker's, from IndexedDB, as a
  // Blob (handed over by reference, never copied), saved from a blob URL.
  // A browser may send a link's download to the server without asking
  // the worker; the link stays for a page without this script.
  document.addEventListener('click', async (e) => {
    const a = e.target.closest && e.target.closest('a[href="/ui/files/catalog"]');
    if (!a || !ready()) return;
    e.preventDefault();
    const got = await ask({ op: 'download' }, 10000);
    if (!got || !got.blob) {
      tell('The catalog could not be downloaded: ' + (got && got.message ? got.message : 'the translator in this browser did not answer. Reload the page, then try again.'));
      return;
    }
    const url = URL.createObjectURL(got.blob);
    const save = document.createElement('a');
    save.href = url;
    save.download = got.name;
    document.body.append(save);
    save.click();
    save.remove();
    setTimeout(() => URL.revokeObjectURL(url), 60000);
  });

  // fetchAsset is one of this release's assets, url, as an ArrayBuffer;
  // what names it in the error that says why it could not be had.
  const fetchAsset = async (url, what) => {
    let r;
    try {
      r = await fetch(url);
    } catch {
      throw new Error(what + ' could not be downloaded: this server could not be reached, and this browser may be offline.');
    }
    if (r.ok) return r.arrayBuffer();
    const server = r.headers.get('Nitrovin-Release');
    throw new Error(what + ' could not be downloaded: this server answered HTTP ' + r.status + (server && server !== release
      ? ', as it now runs another release of nitrovin than this page. Update the translator in this browser (Update now, or close every nitrovin tab), then try again.'
      : '. Reload the page and try again.'));
  };

  // EPA's vocabulary, from this release's assets: one click.
  document.addEventListener('click', (e) => {
    const button = e.target.closest && e.target.closest('#load-epa');
    if (!button) return;
    if (!ready()) {
      tell('Nothing was loaded: ' + why());
      return;
    }
    load("EPA's vocabulary", [{ name: button.dataset.file, read: () => fetchAsset(button.dataset.url, "EPA's vocabulary") }]).catch(failed);
  });

  // loadDemo loads the demo set (pkg/ui's demo.go) from this release's
  // assets, demo being the panel's {files: [{name, file, kind, url}],
  // remove: [name]}: every file is fetched first; the demo files this
  // browser holds from before (remove) go; then its vocabularies load as
  // one batch, its catalog compiles here as a catalog picked on its own
  // does, and its shelf and yard load as one batch, the yard's VINs alone
  // going to the server to be decoded. A step that fails stops it, said.
  const loadDemo = async (demo) => {
    const t0 = performance.now();
    tell('Reading the demo set…');
    const got = await Promise.all(demo.files.map(async (f) => ({ ...f, data: await fetchAsset(f.url, 'The demo set') })));
    if (demo.remove.length > 0) {
      tell('Removing the demo this browser holds…');
      const r = await fetch('/ui/files/remove', { method: 'POST', headers: { 'HX-Request': 'true' }, body: new URLSearchParams(demo.remove.map((n) => ['name', n])) });
      if (!r.ok) {
        // A refusal is the panel, naming the files that link to the demo's.
        if (r.status === 409) htmx.swap('#browser-files', await r.text(), { swapStyle: 'innerHTML' });
        tell('Nothing was loaded: the demo this browser holds could not be removed first' + (r.status === 409 ? ' (above).' : ': HTTP ' + r.status + '.'));
        return;
      }
    }
    const picked = (kinds) => got.filter((f) => kinds.includes(f.kind)).map((f) => ({ name: f.file, read: async () => f.data }));
    const catalog = got.find((f) => f.kind === 'catalog');
    const steps = [
      () => load("the demo set's vocabularies", picked(['spine', 'vocabulary', 'extension-set'])),
      () => compile(new File([catalog.data], catalog.file, { type: 'application/jsonl' })),
      () => load("the demo set's shelf and yard", picked(['stock'])),
    ];
    const said = [];
    for (const step of steps) {
      const done = await step();
      if (!done.ok) return;
      said.push(done.message);
    }
    section.dataset.demoMs = String(Math.round(performance.now() - t0));
    tell('Loaded the demo set in ' + ((performance.now() - t0) / 1000).toFixed(1) + ' s. ' + said.join(' '));
  };
  document.addEventListener('click', (e) => {
    const button = e.target.closest && e.target.closest('#load-demo');
    if (!button) return;
    if (!ready()) {
      tell('Nothing was loaded: ' + why());
      return;
    }
    if (compiling) {
      tell('Nothing was loaded: a catalog is compiling in this tab. Wait until it is done, then load the demo again.');
      return;
    }
    button.disabled = true;
    loadDemo(JSON.parse(button.dataset.demo)).catch(failed).finally(() => { button.disabled = false; });
  });
})();
