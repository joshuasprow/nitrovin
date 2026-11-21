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
// every vocabulary loaded on vPIC's spine, a tag to a vocabulary, one
// with a name before one without. Without JavaScript, on a server that
// loaded the fixture and the EPA sample (cmd/nitrovin-epa/testdata), a
// tag of several names shows every one, a line each. With JavaScript, the worker draws the
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

// stack is one vocabulary's tag of "Also known as": the names on its
// lines, and how many of them the page displays.
type stack struct {
	Vocabulary string   `json:"vocabulary"`
	Names      []string `json:"names"`
	Shown      int      `json:"shown"`
}

// stacks are the tags of "Also known as" in page order, each name its
// make and the rest of its entry; none before a vocabulary is loaded or
// the model is known. Without JavaScript the page's scripts never ran:
// scripts are let run for the reading alone.
func stacks(tb *tab) []stack {
	tb.t.Helper()
	if !tb.js {
		tb.scripts(true)
		defer tb.scripts(false)
	}
	return eval[[]stack](tb, `[...document.querySelectorAll("#aka li.name")].map(tag => {
		const make = tag.querySelector(":scope > .nw.g-wmi"), rows = [...tag.querySelectorAll(".nrow")];
		return {
			vocabulary: tag.querySelector(".nvoc").firstChild.textContent.trim(),
			names: rows.map(r => [(r.querySelector(".g-wmi") || make || {textContent: ""}).textContent, r.querySelector(".g-vds").textContent].join(" ").trim()).sort(),
			shown: rows.filter(r => r.checkVisibility({visibilityProperty: true})).length,
		};
	})`)
}

// checkStacks checks the tags of "Also known as": their vocabularies in
// order, fleet's with its three names, every one displayed, and every
// other without a name.
func checkStacks(tb *tab, step string, vocabularies ...string) {
	tb.t.Helper()
	got := stacks(tb)
	var names []string
	for _, s := range got {
		names = append(names, s.Vocabulary)
		switch {
		case s.Vocabulary == "fleet" && (!slices.Equal(s.Names, fleetNames) || s.Shown != len(fleetNames)):
			tb.t.Errorf("%s: fleet's names %q, %d displayed; want %q, all displayed", step, s.Names, s.Shown, fleetNames)
		case s.Vocabulary != "fleet" && len(s.Names) != 0:
			tb.t.Errorf("%s: %s names %q; want none", step, s.Vocabulary, s.Names)
		}
	}
	if !slices.Equal(names, vocabularies) {
		tb.t.Errorf("%s: tags %q; want %q", step, names, vocabularies)
	}
}

// namesJSOff: without JavaScript a VIN's page has a tag for each of the
// server's vocabularies on vPIC's spine, listing, on the parts spine, not
// among them: fleet's first, with its three names.
func namesJSOff(t *testing.T, path, base string) {
	tb := newTab(t, path, base, &requestLog{}, false)
	tb.do(chromedp.Navigate(base + ramNamed))
	tb.wait("#aka")
	checkStacks(tb, "JS off", "fleet", "epa-vehicles")
	tb.audit("a VIN's names, JS off")
}

// namesJSOn: with JavaScript, in a fresh profile, against a server that
// loaded classics, parts and listing, none of them a vocabulary on vPIC's
// spine, and offers the EPA sample.
func namesJSOn(t *testing.T, path, base string, wire *wireLog) {
	tb := newTab(t, path, base, &requestLog{}, true)
	tb.do(chromedp.Navigate(base + ramNamed))
	tb.wait("#vin-form.live")
	checkStacks(tb, "no file in the browser")
	// A page that reloads loses this.
	eval[bool](tb, `window.stayed = true`)

	// A file loaded is on the page at once, and a second beside it: the
	// vocabulary with a name first.
	tb.vocabularies()
	tb.do(chromedp.SetUploadFiles(chromedp.CSS("#vocab-files"), []string{filepath.Join(fixture, "fleet.jsonl")}))
	tb.waitPrefix("#load-status", "Loaded fleet (")
	tb.until("fleet's names on the page", `document.querySelectorAll("#aka li.name").length === 1`)
	checkStacks(tb, "a load", "fleet")
	tb.do(chromedp.Click(chromedp.CSS("#load-epa")))
	tb.waitPrefix("#load-status", "Loaded epa-vehicles (")
	tb.until("EPA's tag on the page", `document.querySelectorAll("#aka li.name").length === 2`)
	checkStacks(tb, "a second load", "fleet", "epa-vehicles")
	tb.audit("the dialog, two files loaded")
	tb.closeVocabularies()
	tb.audit("a VIN's names in the browser")

	// A removed file is off the page; the rest stay.
	tb.vocabularies()
	tb.do(chromedp.Click(chromedp.CSS(`form.inline:has(input[value="fleet"]) button`)))
	tb.until("removing fleet", `!document.querySelector('a[href="/ui/vocabularies/fleet"]') && document.querySelectorAll("#aka li.name").length === 1`)
	checkStacks(tb, "a removal", "epa-vehicles")
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
