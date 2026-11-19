package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/chromedp/chromedp"
)

// TestWorkerDemo runs the demo set (#278) in a fresh profile against a
// server that loaded nothing, as nitrovin.app. "Load demo" loads it in one
// click, its catalog compiled in the browser, and the panel lists every
// file of it. One of its VINs then has its name in its vehicle vocabulary,
// EPA's loaded beside it, as on a server that loaded the same files
// (`serve --vocab pkg/ui/demo`), and its catalog, downloaded, fits on a
// server as the demo's own file does. A demo file of another version
// makes the offer a refresh. It survives a reload, Remove demo removes
// every file of it, and the only requests that ever carried a body to the
// server were the yard's VINs, in decode batches. It logs the module's
// retained memory with the demo loaded, against ADR 0023's 64 MiB.
func TestWorkerDemo(t *testing.T) {
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
	fixtureData(t)
	module := buildWorker(t)
	epaPath, epa := epaSample(t)
	module.EPA = epa
	wire := &wireLog{h: vocabServer(t, &module)}
	srv := httptest.NewServer(wire)
	t.Cleanup(srv.Close)

	tb := newTab(t, path, srv.URL, &requestLog{}, true)
	tb.do(chromedp.Navigate(srv.URL + "/"))
	tb.vocabularies()
	tb.do(chromedp.Click(chromedp.CSS("#load-demo")))
	tb.waitPrefix("#load-status", "Loaded the demo set in ")
	said := tb.html("#load-status")
	if !strings.Contains(said, "Compiled demo-catalog (") {
		t.Errorf("the demo's load said %q; want its catalog compiled here", said)
	}
	t.Logf("%s (%s ms)", text(said), eval[string](tb, `document.getElementById("browser").dataset.demoMs`))
	stored := func() []string {
		return eval[[]string](tb, `[...document.querySelectorAll("#browser-files tr[data-where=browser] th")].map(th => th.textContent.trim())`)
	}
	demoFiles := []string{"demo-catalog", "demo-classics", "demo-part-codes", "demo-part-types", "demo-shelf", "demo-vehicles", "demo-yard-stock"}
	if got := stored(); !slices.Equal(got, demoFiles) {
		t.Errorf("stored %q; want %q", got, demoFiles)
	}
	if n := tb.count("#remove-demo"); n != 1 {
		t.Error("the panel offers no Remove demo once it is loaded")
	}
	// EPA's vocabulary beside it, a second stack of names.
	tb.do(chromedp.Click(chromedp.CSS("#load-epa")))
	tb.waitPrefix("#load-status", "Loaded ")
	tb.audit("the dialog, the demo and EPA's vocabulary held")

	// The yard's Ram, of make Ram, has its name in the demo's vehicle
	// vocabulary, as the server draws it on the same files.
	same := vocabServer(t, nil, demoDir, epaPath)
	target := "/?from.vin=1C6RR7LT6FS512345"
	got := eval[[]string](tb, `fetch(`+jsString(target)+`, {headers: {"HX-Request": "true"}}).then(async r => [String(r.status), await r.text()])`)
	if want := mainOf(t, same, srv.URL, target, true); got[0] != "200" || timeless(got[1]) != want {
		t.Errorf("%s: the worker's page differs from the server's\nworker: %s %s\nserver: %s", target, got[0], got[1], want)
	}
	if !strings.Contains(strings.Join(strings.Fields(text(got[1])), " "), "Dodge Ram 1500") {
		t.Errorf("%s: the page lacks the demo's Dodge Ram 1500: %s", target, got[1])
	}
	// The page's parts that fit are the catalog compiled here: the part
	// types and a pressed one's candidates, as the server draws them on
	// the same files (#322).
	for _, target := range []string{"/?from.vin=1FTEW1EG7FFA12345", "/?from.vin=1FTEW1EG7FFA12345&part.id=alternator", "/?from.vin=1GCVKREC6EZ123456&part.id=brake-rotor-rear"} {
		got := eval[[]string](tb, `fetch(`+jsString(target)+`, {headers: {"HX-Request": "true"}}).then(async r => [String(r.status), await r.text()])`)
		if want := mainOf(t, same, srv.URL, target, true); got[0] != "200" || timeless(got[1]) != want || !strings.Contains(got[1], `<section class="fit"`) {
			t.Errorf("%s: the worker's page differs from the server's\nworker: %s %s\nserver: %s", target, got[0], got[1], want)
		}
	}
	if got := eval[string](tb, `fetch("/?from.vin=1FTEW1EG7FFA12345&part.id=alternator").then(r => r.text())`); strings.Count(got, `<li class="cand">`) != 3 {
		t.Errorf("the worker's alternators: %s", got)
	}

	// The catalog compiled here, downloaded, fits as the demo's own file.
	var files []string
	for _, n := range []string{"demo-classics", "demo-part-codes", "demo-part-types", "demo-vehicles"} {
		files = append(files, filepath.Join(demoDir, n+".jsonl"))
	}
	fitsAsServer(t, tb.download(`#browser-files a[href="/ui/files/catalog"]`), files, filepath.Join(demoDir, "demo-catalog.jsonl"),
		"from.vin=1FTEW1EG7FFA12345",
		"from.vin=1FTEW1EG7FFA12345&part.spine=demo-part-types&part.id=alternator",
		"from.vin=2HKRW2H53KH612345&part.spine=demo-part-types&part.id=transfer-case",
		"from.vin=2HKRW2H53KH612345&part.vocabulary=demo-part-codes&part.entry.code=DEMO-ROT",
		"from.vocabulary=demo-vehicles&from.entry.make=Honda&from.entry.model=Civic&model_year=2018&related=true",
		"from.ext=ext:demo-classics/vw-type-2&model_year=1974",
		"from.fitment.type=alternator&from.fitment.number=DEMO-0901",
	)

	// What the module holds with the demo loaded.
	timing := tb.timeFetch("/ui/files").Timing
	memory, heap := timingDesc(timing, "memory"), timingDesc(timing, "heap")
	t.Logf("retained with the demo loaded: Go heap %.1f MiB, WebAssembly memory %.1f MiB", float64(heap)/(1<<20), float64(memory)/(1<<20))
	if memory == 0 || memory > 64<<20 {
		t.Errorf("WebAssembly memory %d bytes; ADR 0023's budget is 64 MiB (%s)", memory, timing)
	}

	// A demo file of another version (an older release's) makes the
	// offer a refresh, which removes the demo held and loads it again.
	shelf, err := os.ReadFile(filepath.Join(demoDir, "demo-shelf.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	older := filepath.Join(t.TempDir(), "demo-shelf.jsonl")
	if err := os.WriteFile(older, []byte(strings.Replace(string(shelf), `"version":"2026.11-demo"`, `"version":"2026.10-demo"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	tb.do(chromedp.Navigate(srv.URL + "/"))
	tb.vocabularies()
	tb.do(chromedp.SetUploadFiles(chromedp.CSS("#vocab-files"), []string{older}))
	tb.waitPrefix("#load-status", "Loaded demo-shelf")
	if s := tb.html("#load-demo"); !strings.Contains(s, "Refresh demo") {
		t.Errorf("with an older demo file held, the offer: %s", s)
	}
	tb.do(chromedp.Click(chromedp.CSS("#load-demo")))
	tb.waitPrefix("#load-status", "Loaded the demo set in ")
	if n := tb.count("#remove-demo"); n != 1 {
		t.Errorf("after the refresh the demo is not held: %s", tb.html("#browser-files"))
	}

	// A reload keeps it; Remove demo removes every file of it, and the
	// offer is back.
	tb.do(chromedp.Navigate(srv.URL + "/"))
	tb.vocabularies()
	if got := stored(); !slices.Equal(got, append(slices.Clone(demoFiles), "epa-vehicles")) {
		t.Errorf("after a reload: stored %q", got)
	}
	tb.do(chromedp.Click(chromedp.CSS("#remove-demo")))
	tb.wait("#load-demo")
	if got := stored(); !slices.Equal(got, []string{"epa-vehicles"}) {
		t.Errorf("after Remove demo: stored %q", got)
	}

	// Nothing but the yard's VINs ever carried a body to the server.
	var vins []string
	for _, r := range wire.all() {
		if r.body == 0 {
			continue
		}
		var items []map[string]any
		if r.method != "POST" || r.target != "/v1/decode" || json.Unmarshal([]byte(r.data), &items) != nil {
			t.Errorf("the server got %s %s with a body: %s", r.method, r.target, r.data)
			continue
		}
		for _, it := range items {
			vin, _ := it["vin"].(string)
			if _, ok := demoVINs[vin]; !ok || len(it) != 2 || it["full"] != true {
				t.Errorf("POST /v1/decode carried %v; want only the yard's VINs", it)
			}
			vins = append(vins, vin)
		}
	}
	// Once for each load of the demo: the first, and the refresh.
	if len(vins) != 2*len(demoVINs) {
		t.Errorf("the server decoded %d of the yard's VINs; want each of its %d once a load, twice", len(vins), len(demoVINs))
	}
}
