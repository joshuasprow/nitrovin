package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/chromedp/chromedp"
)

// TestVocabTranslate runs the page's "Also known as" in headless Chrome or
// Chromium, each run in a fresh profile
// (https://github.com/joshuasprow/nitrovin/issues/194): a VIN's names in
// every vocabulary loaded on vPIC's spine, a stack to a vocabulary, one
// with a name before one without. Without JavaScript, on a server that
// loaded the fixture and the EPA sample (cmd/nitrovin-epa/testdata), a
// stack of several names shows its first, and all of them at a click on
// its stamp, which needs no script. With JavaScript, the worker draws the
// page over the files the browser holds: a file loaded is on the page at
// once, a file removed is off it, and the server hears of neither. axe
// audits every state.
func TestVocabTranslate(t *testing.T) {
	path := chromePath()
	if path == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("no Chrome or Chromium to run the page in: set NITROVIN_CHROME")
		}
		t.Skip("no Chrome or Chromium: set NITROVIN_CHROME to run the page in a browser")
	}
	if testing.Short() {
		t.Skip("builds the worker's module")
	}
	fixtureData(t)
	epaPath, epa := epaSample(t)

	off := httptest.NewServer(vocabServer(t, nil, fixture, epaPath))
	t.Cleanup(off.Close)
	namesJSOff(t, path, off.URL)

	module := buildWorker(t)
	module.EPA = epa
	wire := &wireLog{h: vocabServer(t, &module, filepath.Join(fixture, "classics.jsonl"), filepath.Join(fixture, "parts.jsonl"), filepath.Join(fixture, "listing.jsonl"))}
	on := httptest.NewServer(wire)
	t.Cleanup(on.Close)
	namesJSOn(t, path, on.URL, wire)
}

// ramNamed is the fixture's 2010 Ram, which fleet has three names for and
// the EPA sample none, and fleetNames those names.
const ramNamed = "/?from.vin=1D7RV1GT5AS123456"

var fleetNames = []string{"Dodge Other", "Dodge Ram 1500", "Dodge Ram 1500 4x4"}

// stack is one vocabulary's stack of "Also known as": the names on its
// plates, and how many of them the page displays now.
type stack struct {
	Vocabulary string   `json:"vocabulary"`
	Names      []string `json:"names"`
	Shown      int      `json:"shown"`
}

// stacks are the stacks of "Also known as" in page order, each name its
// make and the rest of its entry; none before a vocabulary is loaded or
// the model is known. Without JavaScript the page's scripts never ran:
// scripts are let run for the reading alone.
func stacks(tb *tab) []stack {
	tb.t.Helper()
	if !tb.js {
		tb.scripts(true)
		defer tb.scripts(false)
	}
	return eval[[]stack](tb, `[...document.querySelectorAll("#aka li.stack")].filter(s => s.querySelector(".stamp")).map(s => {
		const plates = [...s.querySelectorAll("li.tagplate:not(.blank)")];
		return {
			vocabulary: (s.querySelector(".voc") || s.querySelector(".stamp")).textContent,
			names: plates.map(p => [...p.querySelectorAll(".seg.g-wmi, .seg.g-vds")].map(seg => seg.textContent).join(" ")).sort(),
			shown: plates.filter(p => p.checkVisibility({visibilityProperty: true})).length,
		};
	})`)
}

// checkStacks checks the stacks of "Also known as": their vocabularies in
// order, fleet's with its three names, shown of them displayed, and every
// other without a name.
func checkStacks(tb *tab, step string, shown int, vocabularies ...string) {
	tb.t.Helper()
	got := stacks(tb)
	var names []string
	for _, s := range got {
		names = append(names, s.Vocabulary)
		switch {
		case s.Vocabulary == "fleet" && (!slices.Equal(s.Names, fleetNames) || s.Shown != shown):
			tb.t.Errorf("%s: fleet's names %q, %d displayed; want %q, %d displayed", step, s.Names, s.Shown, fleetNames, shown)
		case s.Vocabulary != "fleet" && len(s.Names) != 0:
			tb.t.Errorf("%s: %s names %q; want none", step, s.Vocabulary, s.Names)
		}
	}
	if !slices.Equal(names, vocabularies) {
		tb.t.Errorf("%s: stacks %q; want %q", step, names, vocabularies)
	}
}

// fanOut clicks fleet's stamp, its stack's switch, which needs no script.
func (tb *tab) fanOut() {
	tb.t.Helper()
	tb.do(chromedp.Click(chromedp.CSS(`label[for="fan-fleet"]`)))
	tb.wait("#fan-fleet:checked")
}

// namesJSOff: without JavaScript a VIN's page has a stack for each of the
// server's vocabularies on vPIC's spine, listing, on the parts spine, not
// among them: fleet's first, with its first name, and all three at a click
// on its stamp.
func namesJSOff(t *testing.T, path, base string) {
	tb := newTab(t, path, base, &requestLog{}, false)
	tb.do(chromedp.Navigate(base + ramNamed))
	tb.wait("#aka")
	checkStacks(tb, "JS off", 1, "fleet", "epa-vehicles")
	tb.audit("a VIN's names, JS off")
	tb.fanOut()
	checkStacks(tb, "JS off, fleet fanned out", 3, "fleet", "epa-vehicles")
	tb.audit("a vocabulary's every name, JS off")
}

// namesJSOn: with JavaScript, in a fresh profile, against a server that
// loaded classics, parts and listing, none of them a vocabulary on vPIC's
// spine, and offers the EPA sample.
func namesJSOn(t *testing.T, path, base string, wire *wireLog) {
	tb := newTab(t, path, base, &requestLog{}, true)
	tb.do(chromedp.Navigate(base + ramNamed))
	tb.wait("#vin-form.live")
	checkStacks(tb, "no file in the browser", 0)
	// A page that reloads loses this.
	eval[bool](tb, `window.stayed = true`)

	// A file loaded is on the page at once, and a second beside it: the
	// vocabulary with a name first.
	tb.vocabularies()
	tb.do(chromedp.SetUploadFiles(chromedp.CSS("#vocab-files"), []string{filepath.Join(fixture, "fleet.jsonl")}))
	tb.waitPrefix("#load-status", "Loaded fleet (")
	tb.until("fleet's names on the page", `document.querySelectorAll("#aka li.stack:has(.stamp)").length === 1`)
	checkStacks(tb, "a load", 1, "fleet")
	tb.do(chromedp.Click(chromedp.CSS("#load-epa")))
	tb.waitPrefix("#load-status", "Loaded epa-vehicles (")
	tb.until("EPA's stack on the page", `document.querySelectorAll("#aka li.stack:has(.stamp)").length === 2`)
	checkStacks(tb, "a second load", 1, "fleet", "epa-vehicles")
	tb.audit("the dialog, two files loaded")
	tb.closeVocabularies()
	tb.audit("a VIN's names in the browser")
	tb.fanOut()
	checkStacks(tb, "fleet fanned out", 3, "fleet", "epa-vehicles")
	tb.audit("a vocabulary's every name in the browser")

	// A removed file is off the page; the rest stay.
	tb.vocabularies()
	tb.do(chromedp.Click(chromedp.CSS(`form.inline:has(input[value="fleet"]) button`)))
	tb.until("removing fleet", `!document.querySelector('a[href="/ui/vocabularies/fleet"]') && document.querySelectorAll("#aka li.stack:has(.stamp)").length === 1`)
	checkStacks(tb, "a removal", 0, "epa-vehicles")
	tb.audit("the dialog after a removal")
	if !eval[bool](tb, `window.stayed === true`) {
		t.Error("loads and a removal reloaded the page")
	}

	// No file reached the server, nothing threw and CSP refused nothing.
	for _, r := range wire.all() {
		if (r.method != http.MethodGet && r.method != http.MethodHead) || r.body != 0 {
			t.Errorf("the server got %s %s with %d bytes of body", r.method, r.target, r.body)
		}
		u, err := url.Parse(r.target)
		if err != nil || strings.HasPrefix(u.Path, "/ui/r/") {
			continue
		}
		for _, name := range []string{"fleet", "epa-vehicles"} {
			if strings.Contains(r.target, name) {
				t.Errorf("the server got %s, which names %s, a file the browser holds", r.target, name)
			}
		}
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
