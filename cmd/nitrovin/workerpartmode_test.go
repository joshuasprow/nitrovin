package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/chromedp/chromedp"
)

// partTypesList is the page's part types dialog as it is drawn: each
// group as "heading: part type / part type".
const partTypesList = `[...document.querySelectorAll("#part-types .ptgroups section.ptgroup")].map(s =>
	((s.querySelector("h3") || {}).textContent || "") + ": " + [...s.querySelectorAll("li[data-part]")].map(li => li.dataset.part).join(" / "))`

// TestWorkerPartMode runs part-number mode with the worker in control on
// a server that holds a catalog (#393), the demo set's, in a fresh
// profile. The browser loads vocabulary files of its own, so the worker
// answers the page (ADR 0027); then a link to a part, and the plate's
// switch from a VIN, are each answered by the worker and not the server,
// 200, with the part-number plate and the part type list the server's own
// page draws. The worker knows the server's catalog only by what /sw.js
// says of it.
func TestWorkerPartMode(t *testing.T) {
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
	wire := &wireLog{h: vocabServer(t, &module, demoDir)}
	srv := httptest.NewServer(wire)
	t.Cleanup(srv.Close)
	const (
		empty = "/?from.fitment.number="
		found = "/?from.fitment.type=alternator&from.fitment.number=DEMO-4001"
	)
	// status is the page's own, as the browser got it, and its title.
	status := func(tb *tab) (int, string) {
		t.Helper()
		return eval[int](tb, `performance.getEntriesByType("navigation")[0].responseStatus`), eval[string](tb, `document.title`)
	}
	// asked are the pages the server itself was asked for since n.
	asked := func(n int) []string {
		var out []string
		for _, r := range wire.all()[n:] {
			if strings.HasPrefix(r.target, "/?") || r.target == "/" {
				out = append(out, r.target)
			}
		}
		return out
	}

	// The server's own page, before the browser holds a file.
	tb := newTab(t, path, srv.URL, &requestLog{}, true)
	tb.do(chromedp.Navigate(srv.URL + empty))
	tb.wait("#part-form")
	want := eval[[]string](tb, partTypesList)
	if len(want) < 2 || !slices.Equal(asked(0), []string{empty}) {
		t.Fatalf("the server's page: part types %q, the server asked for %q; want its headings, and the page asked of it", want, asked(0))
	}

	// Files of the browser's own: the worker answers from here on.
	tb.vocabularies()
	tb.do(chromedp.SetUploadFiles(chromedp.CSS("#vocab-files"), []string{filepath.Join(fixture, "fleet.jsonl"), filepath.Join(fixture, "classics.jsonl")}))
	tb.waitPrefix("#load-status", "Loaded ")
	if eval[bool](tb, `navigator.serviceWorker.controller === null`) {
		t.Fatal("files loaded, yet no worker controls the page")
	}

	// By link.
	before := len(wire.all())
	tb.do(chromedp.Navigate(srv.URL + found))
	if code, title := status(tb); code != 200 || len(asked(before)) != 0 {
		t.Fatalf("%s with the worker in control: %d %q, the server asked for %q; want 200 from the worker", found, code, title, asked(before))
	}
	tb.wait(`#answer[data-q="from.fitment.type=alternator&from.fitment.number=DEMO-4001"] .ptag`)
	if tb.count("#part-form") != 1 || tb.count("#vin-form") != 0 || !strings.Contains(tb.html(".ptag"), "2 vehicles in demo-catalog") {
		t.Errorf("%s from the worker: no part-number plate, or not the part's tag\n%s", found, tb.html("#lookup"))
	}
	if got := eval[[]string](tb, partTypesList); !slices.Equal(got, want) {
		t.Errorf("%s from the worker, the part types:\n%s\nwant the server's\n%s", found, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	// By the plate's switch, from a VIN's page.
	tb.do(chromedp.Navigate(srv.URL + "/"))
	tb.wait("#vin-form")
	tb.wait("form.live")
	before = len(wire.all())
	tb.do(chromedp.Click(chromedp.CSS(`.modes a.mode[href="/?from.fitment.number="]`)))
	turned := eval[bool](tb, `new Promise(resolve => {
		const start = Date.now();
		(function poll() {
			if (document.getElementById("part-form")) resolve(true); else if (Date.now() - start > 10000) resolve(false); else setTimeout(poll, 50);
		})();
	})`)
	if !turned || tb.at() != empty || tb.count("#vin-form") != 0 || tb.count(`.modes a.mode[aria-current="true"][href="/?from.fitment.number="]`) != 1 {
		t.Fatalf("the switch with the worker in control: the plate turned %t, the tab at %s\n%s", turned, tb.at(), tb.html("#lookup"))
	}
	if got := asked(before); len(got) != 0 {
		t.Errorf("the switch: the server was asked for %q; want the worker's answer", got)
	}
	if got := eval[[]string](tb, partTypesList); !slices.Equal(got, want) {
		t.Errorf("switched, the worker's part types:\n%s\nwant the server's\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	tb.mu.Lock()
	thrown := slices.Clone(tb.thrown)
	tb.mu.Unlock()
	if len(thrown) != 0 {
		t.Errorf("the page threw %q", thrown)
	}
}
