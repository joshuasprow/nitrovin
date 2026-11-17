package main

import (
	"context"
	"fmt"
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

// TestVocabTranslate runs the page's form in headless Chrome or Chromium,
// each run in a fresh profile
// (https://github.com/joshuasprow/nitrovin/issues/194). Every example is a
// link to the page with the form filled in, and fills it: without
// JavaScript as a page, on a server that loaded the fixture and the EPA
// sample (cmd/nitrovin-epa/testdata); with JavaScript by htmx, the worker
// drawing the form over the files the browser holds. With JavaScript the
// targets ticked are the visitor's selection, kept in IndexedDB beside the
// files, by spine: a file loaded is ticked, and the selection survives a
// second load, a removal (the removed file drops out), a reload and a
// cold start of the worker, while a translation's or a link's own to=
// replaces it. A translation is drawn as its map (translationMap). axe
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
	examplesJSOff(t, path, off.URL)
	translationMap(t, path, off.URL)

	module := buildWorker(t)
	module.EPA = epa
	wire := &wireLog{h: vocabServer(t, &module, filepath.Join(fixture, "classics.jsonl"), filepath.Join(fixture, "parts.jsonl"), filepath.Join(fixture, "listing.jsonl"))}
	on := httptest.NewServer(wire)
	t.Cleanup(on.Close)
	selectionJSOn(t, path, on.URL, wire)
}

// The examples: three VINs, which need no file.
const (
	vinExample   = "/?from.vin=" + vin
	teslaExample = movedLink
	ramExample   = "/?from.vin=" + ram2015
)

// open loads the page at target, bare or a lookup, and waits for its
// controls.
func (tb *tab) open(target string) {
	tb.t.Helper()
	tb.do(chromedp.Navigate(tb.base + target))
	tb.wait("#controls")
}

// examplesJSOff: without JavaScript the page has its examples, the VINs,
// and every example is a page with the form filled in. On a page asked
// with targets the examples go to them.
func examplesJSOff(t *testing.T, path, base string) {
	tb := newTab(t, path, base, &requestLog{}, false)
	want := []string{vinExample, teslaExample, ramExample}
	tb.open("/")
	if got := exampleHrefs(tb, "#controls"); !slices.Equal(got, want) {
		t.Errorf("JS off: examples\n%q\nwant\n%q", got, want)
	}
	tb.audit("/, JS off")
	for i, href := range want {
		tb.open("/")
		tb.do(chromedp.Click(chromedp.CSS(fmt.Sprintf("#controls ul.examples li:nth-child(%d) a", i+1))))
		tb.wait("#result")
		if got := tb.at(); got != href {
			t.Errorf("JS off, example %d: at %s; want %s", i+1, got, href)
		}
		checkFilled(tb, "JS off, example "+href, href)
		if i == 0 {
			tb.audit("an example, JS off")
		}
	}

	const both = "&to=epa-vehicles&to=yard"
	tb.open(vinExample + both)
	tb.wait("#result")
	checkFilled(tb, "JS off, a translation", vinExample+both)
	if got, want := exampleHrefs(tb, "#controls"), []string{vinExample + both, teslaExample + both, ramExample + both}; !slices.Equal(got, want) {
		t.Errorf("JS off: the page's examples do not keep its targets: %q; want %q", got, want)
	}
}

// selectionJSOn: with JavaScript, in a fresh profile, against a server
// that loaded classics, parts and listing, a vocabulary on the parts
// spine, which a VIN's form never offers, and offers the EPA sample.
func selectionJSOn(t *testing.T, path, base string, wire *wireLog) {
	tb := newTab(t, path, base, &requestLog{}, true)
	// stay opens the page at target and marks it: a page that reloads
	// loses the mark.
	stay := func(target string) {
		tb.t.Helper()
		tb.open(target)
		tb.ticked()
		eval[bool](tb, `window.stayed = true`)
	}
	tb.open("/")
	tb.ticked()
	if got := exampleHrefs(tb, "#controls"); !slices.Equal(got, []string{vinExample, teslaExample, ramExample}) {
		t.Errorf("no file in the browser: examples %q", got)
	}
	checkTicked(tb, "no file in the browser", nil)

	// A file loaded is ticked, and a second load keeps the first. The
	// page's form follows the files: yard is among its targets at once.
	tb.vocabularies()
	tb.do(chromedp.SetUploadFiles(chromedp.CSS("#vocab-files"), []string{filepath.Join(fixture, "yard.jsonl")}))
	tb.waitPrefix("#load-status", "Loaded yard (")
	tb.until("yard among the form's targets", `document.querySelector('#controls input[name=to][value="yard"]') !== null`)
	checkTicked(tb, "a load", selection{"vpic": {"yard"}})
	tb.vocabularies()
	tb.do(chromedp.Click(chromedp.CSS("#load-epa")))
	tb.waitPrefix("#load-status", "Loaded epa-vehicles (")
	tb.audit("the dialog, two files loaded")
	checkTicked(tb, "a second load", selection{"vpic": {"epa-vehicles", "yard"}})

	// Each example fills the form by htmx, to the targets ticked.
	const both = "&to=epa-vehicles&to=yard"
	want := []string{vinExample + both, teslaExample + both, ramExample + both}
	stay("/")
	if got := exampleHrefs(tb, "#controls"); !slices.Equal(got, want) {
		t.Errorf("JS on: examples\n%q\nwant\n%q", got, want)
	}
	for i, href := range want {
		if i > 0 {
			stay("/")
		}
		tb.do(chromedp.Click(chromedp.CSS(fmt.Sprintf("#controls ul.examples li:nth-child(%d) a", i+1))))
		tb.wait("#result")
		if got := tb.at(); got != href {
			t.Errorf("JS on, example %d: at %s; want %s", i+1, got, href)
		}
		checkFilled(tb, "JS on, example "+href, href)
		if !eval[bool](tb, `window.stayed === true`) {
			t.Errorf("JS on, example %d went as a page", i+1)
		}
		if i == 0 {
			tb.audit("an example in the browser")
		}
	}
	all := selection{"vpic": {"epa-vehicles", "yard"}}
	waitSelection(tb, "every example", all)

	// A target cleared is dropped at once, and an example clicked then
	// goes to what is ticked.
	stay("/")
	tb.waitState("ready")
	tb.ticked()
	tb.do(chromedp.Click(chromedp.CSS(`#controls input[name=to][value="yard"]`)))
	cleared := selection{"vpic": {"epa-vehicles"}}
	waitSelection(tb, "clearing yard", cleared)
	tb.do(chromedp.Click(chromedp.CSS("#controls ul.examples li:nth-child(1) a")))
	tb.wait("#result")
	if got, want := tb.at(), vinExample+"&to=epa-vehicles"; got != want {
		t.Errorf("an example after clearing yard: at %s; want %s", got, want)
	}
	checkTicked(tb, "a translation", cleared)
	tb.waitState("ready")
	tb.do(chromedp.Click(chromedp.CSS(`#controls input[name=to][value="yard"]`)))
	waitSelection(tb, "ticking yard", all)

	// A removed file drops out; the rest stay ticked.
	tb.vocabularies()
	tb.do(chromedp.Click(chromedp.CSS(`form.inline:has(input[value="yard"]) button`)))
	tb.until("removing yard", `!document.querySelector('a[href="/ui/vocabularies/yard"]') && !document.querySelector('#controls input[name=to][value="yard"]')`)
	tb.audit("the dialog after a removal")
	checkTicked(tb, "a removal", cleared)

	// A cold start of the worker keeps the selection.
	tb.stopWorkers()
	checkTicked(tb, "a cold start", cleared)

	// A link's own to= replaces the selection.
	tb.vocabularies()
	tb.do(chromedp.SetUploadFiles(chromedp.CSS("#vocab-files"), []string{filepath.Join(fixture, "yard.jsonl")}))
	tb.waitPrefix("#load-status", "Loaded yard (")
	checkTicked(tb, "yard loaded again", all)
	const link = vinExample + "&to=yard"
	tb.open(link)
	tb.wait("#result")
	checkFilled(tb, "a link with to=", link)
	linked := selection{"vpic": {"yard"}}
	waitSelection(tb, "a link with to=", linked)
	checkTicked(tb, "after a link with to=", linked)

	// No file reached the server, nothing threw and CSP refused nothing.
	for _, r := range wire.all() {
		if (r.method != http.MethodGet && r.method != http.MethodHead) || r.body != 0 {
			t.Errorf("the server got %s %s with %d bytes of body", r.method, r.target, r.body)
		}
		u, err := url.Parse(r.target)
		if err != nil || strings.HasPrefix(u.Path, "/ui/r/") {
			continue
		}
		for _, name := range []string{"yard", "epa-vehicles"} {
			if u.Path == "/ui/vocabularies/"+name || slices.Contains(u.Query()["to"], name) {
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

// attributes are the attributes of each element sel matches, as the page
// has them, none when it matches nothing: no script needed.
func attributes(tb *tab, sel string) []map[string]string {
	tb.t.Helper()
	if tb.count(sel) == 0 {
		return nil
	}
	var out []map[string]string
	tb.call(func(ctx context.Context) (err error) {
		out, err = chromedp.Run(ctx, chromedp.AttributesAll(chromedp.CSSAll(sel)))
		return err
	})
	return out
}

// exampleHrefs are the hrefs of the examples under the element sel
// matches, as the page has them: no script needed.
func exampleHrefs(tb *tab, sel string) []string {
	tb.t.Helper()
	var out []string
	for _, a := range attributes(tb, sel+" a.example") {
		out = append(out, a["href"])
	}
	return out
}

// checkFilled checks that the form of the page asked with target holds
// target's VIN as the page has it and ticks its targets alone: no script
// needed.
func checkFilled(tb *tab, step, target string) {
	tb.t.Helper()
	u, err := url.Parse(target)
	if err != nil {
		tb.t.Fatal(err)
	}
	q := u.Query()
	const inForm = "#controls form "
	var want []string
	for k, vs := range q {
		switch k {
		case "from.vin":
			want = append(want, inForm+"#from-vin[value="+cssString(vs[0])+"]")
		case "to":
			for _, name := range vs {
				want = append(want, inForm+"input[name=to][value="+cssString(name)+"][checked]")
			}
		default:
			tb.t.Fatalf("%s: no field for %s", step, k)
		}
	}
	for _, sel := range want {
		if n := tb.count(sel); n != 1 {
			tb.t.Errorf("%s: %d of %s", step, n, sel)
		}
	}
	// Without to= the page ticks its default targets, the selection.
	if n := tb.count(inForm + "input[name=to][checked]"); q.Has("to") && n != len(q["to"]) {
		tb.t.Errorf("%s: %d targets ticked; want %d", step, n, len(q["to"]))
	}
}

// cssString is s as a CSS string.
func cssString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// selection is a selection as IndexedDB keeps it: names by spine, each
// list sorted, no empty one.
type selection map[string][]string

// checkTicked loads the page and checks that its form ticks want's
// targets on its spine alone, and that IndexedDB keeps want.
func checkTicked(tb *tab, step string, want selection) {
	tb.t.Helper()
	tb.open("/")
	tb.ticked()
	for _, f := range targetsOffered(tb, "#controls") {
		if !slices.Equal(f.Ticked, want[f.Spine]) {
			tb.t.Errorf("%s: the form ticks %q; want %q", step, f.Ticked, want[f.Spine])
		}
	}
	if got := storedSelection(tb); fmt.Sprint(got) != fmt.Sprint(want) {
		tb.t.Errorf("%s: IndexedDB keeps the selection %v; want %v", step, got, want)
	}
}

// ticked waits for the page's first tick of the visitor's selection to
// be done (#browser's data-ticked), whatever came of it: the form is
// ticked once the worker answers, so a box read or clicked before then
// is the page's as drawn, and the tick lands on it afterwards.
func (tb *tab) ticked() {
	tb.t.Helper()
	tb.wait("#browser[data-ticked]")
}

// offered is the targets a form offers, by name: the spine they are on,
// all of them and those ticked.
type offered struct {
	Spine           string
	Targets, Ticked []string
}

// targetsOffered are the targets each form under the element root
// matches offers, in page order, as they are ticked now.
func targetsOffered(tb *tab, root string) []offered {
	tb.t.Helper()
	return eval[[]offered](tb, `[...document.querySelectorAll(`+jsString(root+" fieldset[data-spine]")+`)].map(f => {
		const boxes = [...f.querySelectorAll("input[name=to]")];
		return {
			spine: f.dataset.spine,
			targets: boxes.map(b => b.value).sort(),
			ticked: boxes.filter(b => b.checked).map(b => b.value).sort(),
		};
	})`)
}

// waitSelection waits, up to 10 s, for IndexedDB to keep want.
func waitSelection(tb *tab, step string, want selection) {
	tb.t.Helper()
	got := storedSelection(tb)
	for i := 0; i < 100 && fmt.Sprint(got) != fmt.Sprint(want); i++ {
		eval[bool](tb, `new Promise(resolve => setTimeout(() => resolve(true), 100))`)
		got = storedSelection(tb)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		tb.t.Errorf("%s: IndexedDB keeps the selection %v; want %v", step, got, want)
	}
}

// storedSelection is the selection IndexedDB keeps: none before the
// worker made its database, which this never opens first (opened, it
// would be made without the worker's store).
func storedSelection(tb *tab) selection {
	tb.t.Helper()
	return eval[selection](tb, `indexedDB.databases().then(dbs => dbs.some(d => d.name === "nitrovin-state") ? new Promise((resolve, reject) => {
		const r = indexedDB.open("nitrovin-state");
		r.onerror = () => reject(r.error);
		r.onsuccess = () => {
			const db = r.result;
			const q = db.transaction("state").objectStore("state").get("selection");
			q.onsuccess = () => {
				db.close();
				const out = {};
				for (const [spine, names] of Object.entries(q.result || {})) if (names && names.length) out[spine] = [...names].sort();
				resolve(out);
			};
			q.onerror = () => reject(q.error);
		};
	}) : {})`)
}
