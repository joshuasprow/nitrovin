package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"

	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// TestWorkerCatalog runs a catalog in the browser (ADR 0023) in a fresh
// profile: against a server that has a catalog, the dialog refuses one;
// against one with the fixture's spine, the browser loads the catalog's
// vehicle vocabulary, then compiles catalog.jsonl picked in the dialog
// (in a dedicated worker), which is listed with the drift kinds it could
// not measure, survives a reload, and downloads as a compiled catalog that
// opens, and that a server loads and fits from as it does from the same
// files.
func TestWorkerCatalog(t *testing.T) {
	path := chromePath()
	if path == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("no Chrome or Chromium to run the worker in: set NITROVIN_CHROME")
		}
		t.Skip("no Chrome or Chromium: set NITROVIN_CHROME to run the worker in a browser")
	}
	if testing.Short() {
		t.Skip("builds the module and compiles a catalog in a browser")
	}
	module := buildWorker(t)
	fitment := filepath.Join(fixture, "fitment")
	catalog := filepath.Join(fitment, "catalog.jsonl")

	// One catalog per nitrovin: a server's refuses a browser's.
	withCatalog := httptest.NewServer(vocabServer(t, &module, fitment))
	t.Cleanup(withCatalog.Close)
	tb := newTab(t, path, withCatalog.URL, &requestLog{}, true)
	tb.do(chromedp.Navigate(withCatalog.URL + "/"))
	tb.vocabularies()
	tb.do(chromedp.SetUploadFiles(chromedp.CSS("#vocab-files"), []string{catalog}))
	tb.waitPrefix("#load-status", "Nothing was compiled: This server has a catalog, parts-catalog; one catalog per nitrovin")

	wire := &wireLog{h: vocabServer(t, &module, filepath.Join(fitment, "part-types.jsonl"), filepath.Join(fitment, "part-codes.jsonl"))}
	srv := httptest.NewServer(wire)
	t.Cleanup(srv.Close)
	tb = newTab(t, path, srv.URL, &requestLog{}, true)
	tb.do(chromedp.Navigate(srv.URL + "/"))
	tb.vocabularies()
	served := func(target string) bool {
		return slices.ContainsFunc(wire.all(), func(r wireRequest) bool { return r.target == target })
	}

	tb.do(chromedp.SetUploadFiles(chromedp.CSS("#vocab-files"), []string{filepath.Join(fitment, "catalog-models.jsonl")}))
	tb.waitPrefix("#load-status", "Loaded ")
	tb.do(chromedp.SetUploadFiles(chromedp.CSS("#vocab-files"), []string{catalog}))
	tb.waitPrefix("#load-status", "Compiled parts-catalog (5 fitments")
	if s := tb.html("#load-status"); !strings.Contains(s, "Not checked here: model-years, model-qualifier drift") {
		t.Errorf("the compile said %q", s)
	}
	if bodies := slices.ContainsFunc(wire.all(), func(r wireRequest) bool { return r.body > 0 }); bodies {
		t.Error("the server got a request with a body: a file left the browser")
	}

	// It survives a reload.
	tb.do(chromedp.Navigate(srv.URL + "/"))
	tb.vocabularies()
	if got := eval[[]string](tb, `[...document.querySelectorAll("#browser-files tr[data-where=browser] th")].map(th => th.textContent.trim())`); !slices.Equal(got, []string{"catalog-models", "parts-catalog"}) {
		t.Errorf("after a reload: stored %q", got)
	}
	if s := tb.html("#browser-files"); !strings.Contains(s, "catalog, 5 fitments") || !strings.Contains(s, "model-years, model-qualifier drift") {
		t.Errorf("the panel: %s", s)
	}

	// The compiled file downloads, and opens.
	b64 := eval[string](tb, `fetch("/ui/files/catalog").then(r => r.blob()).then(b => new Promise(resolve => {
		const f = new FileReader(); f.onload = () => resolve(f.result.split(",")[1]); f.readAsDataURL(b);
	}))`)
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}
	f, err := vocab.OpenCompiled("parts-catalog.compiled", bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("the download does not open: %v", err)
	}
	if cc := f.Compiled(); f.Name != "parts-catalog" || !slices.Equal(cc.Skipped, []vocab.DriftKind{vocab.DriftModelYears, vocab.DriftModelQualifier}) {
		t.Errorf("the download: %s, skipped %v", f.Name, cc.Skipped)
	}

	// A server that loads it fits from it as from the same catalog as
	// JSON Lines.
	files := []string{filepath.Join(fitment, "part-types.jsonl"), filepath.Join(fitment, "part-codes.jsonl"), filepath.Join(fitment, "catalog-models.jsonl")}
	fitsAsServer(t, data, files, catalog,
		"from.model=1938&model_year=2010",
		"from.vin=1D7RV1GT5AS123456",
		"from.vocabulary=catalog-models&from.entry.make=Dodge&from.entry.model=Ram+1500&model_year=2010&part.spine=part-types&part.id=mirror&side=left",
		"from.vocabulary=catalog-models&from.entry.make=Chevrolet&from.entry.model=Silverado+1500&part.vocabulary=part-codes&part.entry.code=ROT",
		"from.model=1938&related=true",
		"from.fitment.type=brake-rotor-front&from.fitment.number=BR-1999",
	)

	// The Download link, clicked, saves those bytes, never asking the
	// server: a browser may send a link's download past the worker.
	if got := tb.download(`#browser-files a[href="/ui/files/catalog"]`); !bytes.Equal(got, data) {
		t.Errorf("the download clicked saved %d bytes; the worker's file is %d", len(got), len(data))
	}
	if served("/ui/files/catalog") {
		t.Error("the download went to the server")
	}

	// Opening the files while the catalog is reset (a compile in another
	// tab just stored one) answers, and so does the next request.
	got := eval[[]int](tb, `(async () => {
		const ask = (op) => new Promise((resolve) => { const ch = new MessageChannel(); ch.port1.onmessage = () => resolve(); navigator.serviceWorker.controller.postMessage({op}, [ch.port2]); });
		await ask("catalog");
		const first = fetch("/ui/files", {headers: {"HX-Request": "true"}}).then((r) => r.status);
		await ask("catalog");
		const late = new Promise((resolve) => setTimeout(() => resolve(0), 20000));
		return [await Promise.race([first, late]), await Promise.race([fetch("/ui/files", {headers: {"HX-Request": "true"}}).then((r) => r.status), late])];
	})()`)
	if !slices.Equal(got, []int{200, 200}) {
		t.Errorf("requests around a reset answered %v; want [200 200]", got)
	}
}

// TestWorkerCatalogParts compiles in the browser a generated catalog of a
// few MB, so its sorted runs and its compiled file are kept in IndexedDB in
// several parts of a MiB, written across them and read across them: its
// download, clicked, opens, and a server that loads it fits from it as
// from the same file. A read IndexedDB never answers gives up after 10 s,
// in words, and the next request is answered.
func TestWorkerCatalogParts(t *testing.T) {
	path := chromePath()
	if path == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("no Chrome or Chromium to run the worker in: set NITROVIN_CHROME")
		}
		t.Skip("no Chrome or Chromium: set NITROVIN_CHROME to run the worker in a browser")
	}
	if testing.Short() {
		t.Skip("builds the module and compiles a catalog in a browser")
	}
	module := buildWorker(t)
	fitment := filepath.Join(fixture, "fitment")
	catalog := filepath.Join(t.TempDir(), "generated.jsonl")
	writeGenerated(t, catalog, 20000)

	wire := &wireLog{h: vocabServer(t, &module, filepath.Join(fitment, "part-types.jsonl"), filepath.Join(fitment, "part-codes.jsonl"))}
	srv := httptest.NewServer(wire)
	t.Cleanup(srv.Close)
	tb := newTab(t, path, srv.URL, &requestLog{}, true)
	tb.do(chromedp.Navigate(srv.URL + "/"))
	tb.vocabularies()
	tb.do(chromedp.SetUploadFiles(chromedp.CSS("#vocab-files"), []string{filepath.Join(fitment, "catalog-models.jsonl")}))
	tb.waitPrefix("#load-status", "Loaded ")
	tb.do(chromedp.SetUploadFiles(chromedp.CSS("#vocab-files"), []string{catalog}))
	tb.waitPrefix("#load-status", "Compiled parts-catalog (20,000 fitments")

	data := tb.download(`#browser-files a[href="/ui/files/catalog"]`)
	f, err := vocab.OpenCompiled("parts-catalog.compiled", bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("the download clicked does not open: %v", err)
	}
	if n := f.Catalog().Info().Fitments; n != 20000 || len(data) < 3<<20 {
		t.Errorf("the download: %d fitments in %d bytes; want 20,000 in several MiB parts", n, len(data))
	}
	files := []string{filepath.Join(fitment, "part-types.jsonl"), filepath.Join(fitment, "part-codes.jsonl"), filepath.Join(fitment, "catalog-models.jsonl")}
	fitsAsServer(t, data, files, catalog,
		"from.model=1938&model_year=2010",
		"from.model=1938&model_year=2010&part.spine=part-types&part.id=engine",
		"from.model=1850&related=true",
		"from.fitment.type=engine&from.fitment.number=GEN-000001",
		"from.fitment.type=brake-rotor-front&from.fitment.number=GEN-019998",
	)

	// IndexedDB stops answering a read: the request gives up after 10 s
	// with a 503 in words, and the next one is answered. In the worker
	// itself, all at once: the module is made to open its files again,
	// which reads the catalog, its reads are held and the panel is asked
	// of it, then they are let go and it is asked again.
	panel := srv.URL + "/ui/files"
	got := eval[[]string](tb.serviceWorker(), `(async () => {
		const slice = Blob.prototype.slice;
		let held = 0;
		Blob.prototype.slice = function (...a) { const b = slice.apply(this, a); b.arrayBuffer = () => { held++; return new Promise(() => {}); }; return b; };
		self.nitrovin.reset();
		const t0 = Date.now();
		const r = await self.nitrovin.serve("GET", `+jsString(panel)+`, "true", "", "");
		const took = Date.now() - t0;
		Blob.prototype.slice = slice;
		const next = await self.nitrovin.serve("GET", `+jsString(panel)+`, "true", "", "");
		return [String(r.status), r.headers["Retry-After"] || "", new TextDecoder().decode(r.body), String(held), String(took), String(next.status),
			String(new TextDecoder().decode(next.body).includes("/ui/vocabularies/parts-catalog"))];
	})()`)
	if got[0] != "503" || got[1] != "5" || got[2] != "This browser's storage did not answer a read of its files in 10 s. Reload the page to try again." || got[3] == "0" || got[5] != "200" || got[6] != "true" {
		t.Errorf("a read that never settles: %q", got)
	}
	// The worker holds its databases open, yet a page deleting one is not
	// kept waiting: the worker closes it, and opens it again when next it
	// needs it.
	deleted := eval[[]string](tb, `(async () => {
		const gone = await new Promise((resolve) => {
			const r = indexedDB.deleteDatabase("nitrovin-catalog");
			r.onsuccess = () => resolve("deleted");
			r.onerror = () => resolve("failed: " + r.error);
			setTimeout(() => resolve("still waiting after 10 s"), 10000);
		});
		const r = await fetch("/ui/files/catalog");
		return [gone, String(r.status)];
	})()`)
	if !slices.Equal(deleted, []string{"deleted", "404"}) {
		t.Errorf("deleting the catalog's database from the page: %q; want deleted, then 404", deleted)
	}
	if bodies := slices.ContainsFunc(wire.all(), func(r wireRequest) bool { return r.body > 0 || r.target == "/ui/files/catalog" }); bodies {
		t.Error("the server got the catalog or a request with a body")
	}
}

// fitsAsServer holds a compiled catalog the browser made, data, to the
// file it was compiled from, catalog: a server that loads it beside files
// answers each query of /v1/fit as one that loaded catalog beside them
// does.
func fitsAsServer(t *testing.T, data []byte, files []string, catalog string, queries ...string) {
	t.Helper()
	compiled := filepath.Join(t.TempDir(), "catalog.compiled")
	if err := os.WriteFile(compiled, data, 0o644); err != nil {
		t.Fatal(err)
	}
	fit := func(h http.Handler, q string) string {
		t.Helper()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/v1/fit?"+q+pinned, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("/v1/fit?%s: HTTP %d\n%s", q, w.Code, w.Body)
		}
		return w.Body.String()
	}
	compiledIn, compiledOn := vocabServer(t, nil, append(slices.Clone(files), compiled)...), vocabServer(t, nil, append(slices.Clone(files), catalog)...)
	for _, q := range queries {
		if got, want := fit(compiledIn, q), fit(compiledOn, q); got != want {
			t.Errorf("/v1/fit?%s: the browser's compiled catalog fits differently from the server's\nbrowser: %.2000s\nserver: %.2000s", q, got, want)
		}
	}
}

// writeGenerated writes to path a catalog of n fitments over the fitment
// fixture's vehicles: engines and front rotors, numbered GEN-000000 on.
func writeGenerated(t *testing.T, path string, n int) {
	t.Helper()
	head, err := os.ReadFile(filepath.Join(fixture, "fitment", "catalog.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	b.Write(head[:bytes.IndexByte(head, '\n')+1])
	for i := range n {
		typ := [2]string{"brake-rotor-front", "engine"}[i%2]
		fmt.Fprintf(&b, `{"fitment":{"type":%q,"number":"GEN-%06d"},"note":"generated %d","applications":[{"vehicle":{"make":"Dodge","model":"Ram 1500"},"first":%d,"last":2012,"options":"option %d"},{"vehicle":{"make":"Chevrolet","model":"Silverado 1500"},"first":2007,"last":2013,"options":""}]}`+"\n",
			typ, i, i, 2009+i%4, i%97)
	}
	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// download clicks sel and waits for what it saves, the file's bytes.
func (tb *tab) download(sel string) []byte {
	tb.t.Helper()
	dir := tb.t.TempDir()
	ctx, cancel := context.WithTimeout(tb.ctx, time.Minute)
	defer cancel()
	progress := chromedp.Events(ctx, browser.DownloadProgress)
	on := true
	tb.do(chromedp.Func(func(ctx context.Context, _ *chromedp.Target) error {
		_, err := chromedp.Call(ctx, browser.SetDownloadBehavior, browser.SetDownloadBehaviorParams{
			Behavior: browser.SetDownloadBehaviorBehaviorAllowAndName, DownloadPath: dir, EventsEnabled: &on,
		})
		return err
	}), chromedp.Click(chromedp.CSS(sel)))
	for ev, err := range progress {
		if err != nil {
			tb.t.Fatalf("waiting for the download (the page said %q): %v", tb.html("#load-status"), err)
		}
		switch ev.State {
		case browser.DownloadProgressStateCompleted:
			b, err := os.ReadFile(filepath.Join(dir, ev.GUID))
			if err != nil {
				tb.t.Fatal(err)
			}
			return b
		case browser.DownloadProgressStateCanceled:
			tb.t.Fatal("the download was canceled")
		}
	}
	tb.t.Fatal("no download completed in a minute")
	return nil
}

// serviceWorker is a tab on the page's Service Worker, for eval.
func (tb *tab) serviceWorker() *tab {
	tb.t.Helper()
	var id target.ID
	tb.call(func(ctx context.Context) error {
		infos, err := chromedp.Targets(ctx)
		for _, i := range infos {
			if i.Type == "service_worker" && strings.HasPrefix(i.URL, tb.base+"/sw.js") {
				id = i.TargetID
			}
		}
		if err == nil && id == "" {
			err = fmt.Errorf("no Service Worker target among %d", len(infos))
		}
		return err
	})
	ctx, cancel := chromedp.NewContext(tb.ctx, chromedp.WithTargetID(id))
	tb.t.Cleanup(cancel)
	return &tab{t: tb.t, ctx: ctx, base: tb.base, log: tb.log, js: true}
}
