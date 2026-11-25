package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/chromedp/chromedp"

	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/nitrovin"
	"github.com/joshuasprow/nitrovin/pkg/ui"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// release is the embedded decoder reporting another decoder version in
// its info, its decodes and its translations, as a release build does.
type release struct{ versioned }

func (r release) Decode(req contract.Request) (contract.Decode, error) {
	d, err := r.versioned.Decode(req)
	d.DecoderVersion = r.version
	return d, err
}

func (r release) Translate(req contract.TranslateRequest) (contract.Translation, error) {
	tr, err := r.versioned.Translate(req)
	tr.DecoderVersion = r.version
	return tr, err
}

// releaseServer serves the API with the embedded data, the files at paths
// loaded and the worker's module, its decoder reporting version: one
// release of nitrovin, as a deploy of it serves. It returns its Info too.
func releaseServer(t *testing.T, w ui.Worker, version string, paths ...string) (*api, contract.Info) {
	t.Helper()
	d, err := nitrovin.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	files, err := vocab.Open(paths...)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := nitrovin.NewTranslator(d, files...)
	if err != nil {
		t.Fatal(err)
	}
	svc := release{versioned{loaded{d, tr}, version}}
	a, err := newAPI(svc, log.New(io.Discard, "", 0), io.Discard, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	a.ui.WithWorker(w)
	return a, svc.Info()
}

// deploy is the server at one address across a deploy: it serves one
// release, then the next.
type deploy struct{ current atomic.Pointer[api] }

func (d *deploy) ServeHTTP(w http.ResponseWriter, r *http.Request) { d.current.Load().ServeHTTP(w, r) }

// pageHTML is a whole page as server h draws it for target.
func pageHTML(t *testing.T, h http.Handler, base, target string) string {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", base+target, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("%s: HTTP %d from the server", target, w.Code)
	}
	return w.Body.String()
}

// TestWorkerUpdate: a worker installed from one release (v0.3.2) while the
// server, at the same address, moves on to the next (v0.3.3), as a deploy
// does. The old worker steps aside: the next release's page renders where
// the old worker would have drawn its own, and no request names the old
// release's assets once the server has said its release. The page says
// the translator is outdated and, once the next release's worker is
// waiting, offers Update now, which hands the page to it and reloads; the
// files stored survive, and the new worker answers for them. Every page's
// foot shows the release's version, the worker's pages as the server's.
func TestWorkerUpdate(t *testing.T) {
	path := chromePath()
	if path == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("no Chrome or Chromium to run the worker in: set NITROVIN_CHROME")
		}
		t.Skip("no Chrome or Chromium: set NITROVIN_CHROME to run the worker in a browser")
	}
	if testing.Short() {
		t.Skip("builds the module and runs it in a browser")
	}
	fixtureData(t)
	module := buildWorker(t)
	classics := filepath.Join(fixture, "classics.jsonl")
	old, oldInfo := releaseServer(t, module, "v0.3.2", classics)
	next, nextInfo := releaseServer(t, module, "v0.3.3", classics)
	dep := &deploy{}
	dep.current.Store(old)
	wire := &wireLog{h: dep}
	srv := httptest.NewServer(wire)
	t.Cleanup(srv.Close)

	// The foot of server h's page, which names its release, as this browser parses it.
	foot := func(tb *tab, h http.Handler) string {
		return eval[string](tb, `new DOMParser().parseFromString(`+jsString(pageHTML(t, h, srv.URL, "/"))+`, "text/html").querySelector("footer").outerHTML`)
	}
	served := func(since int, target string) bool {
		return slices.ContainsFunc(wire.all()[since:], func(r wireRequest) bool { return r.target == target })
	}

	tb := newTab(t, path, srv.URL, &requestLog{}, true)
	tb.do(chromedp.Navigate(srv.URL + "/"))

	// A tab with no worker hears of a deploy from the server's next
	// answer: Update now shows in the header, outside the dialog, and
	// reloads the page, which is then the next release's.
	tb.wait("#browser")
	if got := eval[string](tb, `getComputedStyle(document.getElementById("update-now")).display`); got != "none" {
		t.Errorf("no update: Update now is displayed %q; want none", got)
	}
	dep.current.Store(next)
	eval[bool](tb, `htmx.ajax("GET", "/?from.vin=1D7RV1GT5AS123456", {target: "#ui-main", swap: "innerHTML"}).then(() => true, () => true)`)
	tb.until("Update now with no worker", `document.documentElement.dataset.update === "server" && getComputedStyle(document.getElementById("update-now")).display !== "none"`)
	if eval[bool](tb, `document.getElementById("vocabularies").contains(document.getElementById("update-now"))`) {
		t.Error("Update now is inside the Vocabularies dialog; want the header")
	}
	tb.audit("the header, Update now showing")
	tb.do(chromedp.Click(chromedp.CSS("#update-now")))
	tb.wait(`#browser[data-release="` + ui.Release(nextInfo) + `"]`)
	if got := eval[string](tb, `document.documentElement.dataset.update || ""`); got != "" {
		t.Errorf("after Update now with no worker: data-update %q; want none", got)
	}
	dep.current.Store(old)
	tb.do(chromedp.Navigate(srv.URL + "/"))
	tb.wait(`#browser[data-release="` + ui.Release(oldInfo) + `"]`)
	tb.vocabularies()
	tb.do(chromedp.SetUploadFiles(chromedp.CSS("#vocab-files"), []string{filepath.Join(fixture, "fleet.jsonl")}))
	tb.waitPrefix("#load-status", "Loaded ")
	tb.closeVocabularies()

	// The old release's worker draws the page over fleet, a VIN's names in
	// it; its page's foot is the server's: nitrovin v0.3.2, linked to
	// its release.
	entry := "/?from.vin=1D7RV1GT5AS123456"
	since := len(wire.all())
	tb.do(chromedp.Navigate(srv.URL + entry))
	tb.wait("#aka .lrow")
	if served(since, entry) {
		t.Fatalf("%s: the server answered it; want the worker, fleet is the browser's", entry)
	}
	if got, want := eval[string](tb, `document.querySelector("footer").outerHTML`), foot(tb, old); got != want ||
		!strings.Contains(got, `<a href="https://github.com/joshuasprow/nitrovin/releases/tag/v0.3.2">v0.3.2</a>`) {
		t.Errorf("the worker's foot differs from the server's, or names no v0.3.2\nworker: %s\nserver: %s", got, want)
	}

	// The deploy: the server now runs v0.3.3.
	oldAssets := strings.TrimSuffix(ui.SpinePath(oldInfo), "spine.json")
	since = len(wire.all())
	dep.current.Store(next)

	// The page, which the old worker drew while the files were stored, is
	// the next release's now, the server's.
	tb.do(chromedp.Navigate(srv.URL + "/"))
	tb.wait("#browser")
	if !served(since, "/") {
		t.Error("/ after the deploy: the old worker answered it; want the server")
	}
	if got := attributes(tb, "#browser"); len(got) != 1 || got[0]["data-release"] != ui.Release(nextInfo) {
		t.Errorf("/ after the deploy: #browser %v; want the page of %s", got, ui.Release(nextInfo))
	}
	if got, want := mainHTML(tb), mainAs(tb, mainOf(t, next, srv.URL, "/", false)); got != want {
		t.Errorf("/ after the deploy: the page differs from the next release's\npage: %s\nserver: %s", got, want)
	}
	if got := foot(tb, next); !strings.Contains(got, `<a href="https://github.com/joshuasprow/nitrovin/releases/tag/v0.3.3">v0.3.3</a>`) ||
		eval[string](tb, `document.querySelector("footer").outerHTML`) != got {
		t.Errorf("/ after the deploy: the foot is not v0.3.3's: %s", eval[string](tb, `document.querySelector("footer").outerHTML`))
	}

	// The page says so, and once the next release's worker waits, offers
	// Update now; the dialog passes axe so.
	tb.waitState("outdated")
	tb.until("the update waiting", `document.documentElement.dataset.update === "waiting" && getComputedStyle(document.getElementById("update-now")).display !== "none"`)
	if s := tb.html("#browser-status"); !strings.Contains(s, "v0.3.2") || !strings.Contains(s, "v0.3.3") {
		t.Errorf("outdated: the page names not both releases: %s", s)
	}
	// A load is refused in words.
	tb.do(chromedp.Click(chromedp.CSS("button.open-vocabularies")))
	tb.wait("#vocabularies:popover-open")
	tb.audit("the dialog, its translator outdated, an update waiting")
	if n := tb.count("#vocab-files"); n != 0 {
		t.Error("outdated: a file input, with no worker to take the files")
	}
	tb.dropFiles(filepath.Join(fixture, "parts.jsonl"))
	tb.waitPrefix("#load-status", "Nothing was loaded: the translator in this browser is of another release of nitrovin")

	// Not one request named the old release's assets after the deploy.
	for _, r := range wire.all()[since:] {
		if strings.HasPrefix(r.target, oldAssets) {
			t.Errorf("after the deploy, the server got %s %s, an asset of the old release", r.method, r.target)
		}
	}

	// Update now: the next release's worker takes over and the page
	// reloads; the files stored survive, and it answers for them.
	eval[bool](tb, `(window.beforeUpdate = true)`)
	since = len(wire.all())
	tb.do(chromedp.Click(chromedp.CSS("#update-now")))
	tb.wait(`#browser[data-state="ready"]`)
	if eval[bool](tb, `window.beforeUpdate === true`) {
		t.Error("Update now: the page never reloaded")
	}
	if got := eval[string](tb, `new Promise(resolve => {
		const ch = new MessageChannel();
		ch.port1.onmessage = e => resolve(e.data.release);
		navigator.serviceWorker.controller.postMessage({op: "release"}, [ch.port2]);
	})`); got != ui.Release(nextInfo) {
		t.Errorf("after Update now: the worker in control is of %s; want %s", got, ui.Release(nextInfo))
	}
	if served(since, "/") {
		t.Error("after Update now: the server answered /; want the next release's worker, files stored")
	}
	if got := storedNames(tb); !slices.Equal(got, []string{"fleet"}) {
		t.Errorf("after Update now: stored files %q; want fleet", got)
	}
	since = len(wire.all())
	tb.do(chromedp.Navigate(srv.URL + entry))
	tb.wait("#aka .lrow")
	if served(since, entry) {
		t.Errorf("%s after Update now: the server answered it; want the next release's worker", entry)
	}
	same, _ := releaseServer(t, module, "v0.3.3", classics, filepath.Join(fixture, "fleet.jsonl"))
	if got, want := mainHTML(tb), mainAs(tb, mainOf(t, same, srv.URL, entry, false)); got != want {
		t.Errorf("%s after Update now: the worker's page differs from the server's\nworker: %s\nserver: %s", entry, got, want)
	}
	if got, want := eval[string](tb, `document.querySelector("footer").outerHTML`), foot(tb, next); got != want {
		t.Errorf("after Update now: the worker's foot differs from the server's\nworker: %s\nserver: %s", got, want)
	}

	// A rollout's old container answers once more, after the new one. The
	// worker hears v0.3.2 and steps aside, as it must; it asks again, and
	// once the server names v0.3.3 the files are back, with no reload and
	// no update offered.
	eval[bool](tb, `(window.beforeFlap = true, document.getElementById("browser").dataset.retry = "500", true)`)
	dep.current.Store(old)
	eval[bool](tb, `htmx.ajax("GET", "/", {target: "#ui-main", swap: "innerHTML"}).then(() => true, () => true)`)
	tb.waitState("outdated")
	dep.current.Store(next)
	tb.waitState("ready")
	if !eval[bool](tb, `window.beforeFlap === true`) {
		t.Error("after the old release's stray answer: the page reloaded; want it to resume in place")
	}
	if got := eval[string](tb, `document.documentElement.dataset.update || ""`); got != "" {
		t.Errorf("after the old release's stray answer: data-update %q; want none", got)
	}
	since = len(wire.all())
	tb.do(chromedp.Navigate(srv.URL + entry))
	tb.wait("#aka .lrow")
	if served(since, entry) {
		t.Errorf("%s after the old release's stray answer: the server answered it; want the worker again", entry)
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()
	if len(tb.thrown) > 0 {
		t.Errorf("thrown: %q", tb.thrown)
	}
	if refused := eval[[]string](tb, `window.refused || []`); len(refused) > 0 {
		t.Errorf("CSP refused %q", refused)
	}
}
