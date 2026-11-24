package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// The decode form in a browser (ADR 0013): headless Chrome or Chromium
// driven over the DevTools protocol, each run in a browser of its own with
// a fresh profile. An installed Service Worker survives JavaScript being
// turned off, so a profile that ran the pages with JavaScript would answer
// the JavaScript-off run from the worker.

// axeVersion is testdata/axe.min.js: axe.min.js of npm's axe-core at this
// version, byte for byte, held to axeSHA256 (TestAxePin). scripts/tools.sh
// outdated reports a newer release. To bump, replace the file from the
// package, then this version and the hash.
const axeVersion = "4.14.1"

// axeSHA256 is testdata/axe.min.js's sha256; npm lists the tarball's
// integrity as
// sha512-FHfa+lmNjQZfJz1q8wKqIDwC6DExcHt/pkjNQb09a3NPjCKg2rhu5qb2S4r76ASyFNNtuFooJpZe37OgudVGtQ==.
const axeSHA256 = "29348999450cdf2849a8e70b7912fc2da85a849da39c0d0d5394f51d4a9ed95c"

//go:embed testdata/axe.min.js
var axeJS string

// TestAxePin holds the vendored axe-core to its pin.
func TestAxePin(t *testing.T) {
	sum := sha256.Sum256([]byte(axeJS))
	if got := hex.EncodeToString(sum[:]); got != axeSHA256 {
		t.Errorf("testdata/axe.min.js: sha256 %s; want %s, axe-core %s's", got, axeSHA256, axeVersion)
	}
	if !strings.HasPrefix(axeJS, "/*! axe v"+axeVersion+"\n") {
		t.Errorf("testdata/axe.min.js does not say it is axe v%s", axeVersion)
	}
}

// chromePath is the browser the runs use: $NITROVIN_CHROME, else a Chrome
// or Chromium on PATH or in /Applications; "" when there is none.
func chromePath() string {
	if p := os.Getenv("NITROVIN_CHROME"); p != "" {
		return p
	}
	for _, name := range []string{
		"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Chromium.app/Contents/MacOS/Chromium",
	} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// pageRequest is a request for one of the pages, as the server got it.
type pageRequest struct {
	target string // path and query
	// htmx is whether it asked for a fragment (HX-Request).
	htmx bool
}

// requestLog serves h and records the requests for pages, assets aside.
type requestLog struct {
	h    http.Handler
	mu   sync.Mutex
	seen []pageRequest
}

// ServeHTTP records the requests for pages, / and under /ui/, assets
// aside: not the browser's own, such as /favicon.ico.
func (l *requestLog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p := r.URL.Path; p == "/" || strings.HasPrefix(p, "/ui/") && !strings.HasPrefix(p, "/ui/r/") {
		l.mu.Lock()
		l.seen = append(l.seen, pageRequest{r.URL.RequestURI(), r.Header.Get("HX-Request") == "true"})
		l.mu.Unlock()
	}
	l.h.ServeHTTP(w, r)
}

// take is the requests since the last take.
func (l *requestLog) take() []pageRequest {
	l.mu.Lock()
	defer l.mu.Unlock()
	seen := l.seen
	l.seen = nil
	return seen
}

// tab is one browser's one tab, at base, the form's server, with
// JavaScript or without.
type tab struct {
	t    *testing.T
	ctx  context.Context
	base string
	log  *requestLog
	js   bool
	mu   sync.Mutex
	// thrown are the exceptions thrown in the tab.
	thrown []string
}

// newTab starts a browser with a fresh profile, JavaScript off or on. With
// JavaScript on, every page records what CSP refused, and may use the
// clipboard. The back/forward cache is off, besides chromedp's defaults:
// a page it restores is no new document to the DevTools protocol, so
// queries wait on the one that left. Back and forward without JavaScript
// load the page again; with it, the plate asks the server for the answer
// (plate.js), which the cache never sees.
func newTab(t *testing.T, path, base string, log *requestLog, js bool) *tab {
	t.Helper()
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(path), chromedp.NoSandbox,
		chromedp.Flag("disable-features", "Translate,BlinkGenPropertyTrees,BackForwardCache"))
	actx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(actx)
	t.Cleanup(func() { cancel(); cancelAlloc() })
	tb := &tab{t: t, ctx: ctx, base: base, log: log, js: js}
	if err := chromedp.Listen(ctx, func(ev any) {
		if e, ok := ev.(*runtime.EventExceptionThrown); ok {
			tb.mu.Lock()
			tb.thrown = append(tb.thrown, e.ExceptionDetails.Text+" "+e.ExceptionDetails.URL)
			tb.mu.Unlock()
		}
	}); err != nil {
		t.Fatal(err)
	}
	if !js {
		tb.call(func(ctx context.Context) error {
			_, err := chromedp.Call(ctx, emulation.SetScriptExecutionDisabled, emulation.SetScriptExecutionDisabledParams{Value: true})
			return err
		})
	} else {
		tb.call(func(ctx context.Context) error {
			_, err := chromedp.Call(ctx, page.AddScriptToEvaluateOnNewDocument, page.AddScriptToEvaluateOnNewDocumentParams{
				Source: `window.refused = []; document.addEventListener("securitypolicyviolation", e => refused.push(e.violatedDirective + " " + e.blockedURI));`,
			})
			if err != nil {
				return err
			}
			for _, name := range []string{"clipboard-read", "clipboard-write"} {
				_, err = chromedp.CallBrowser(ctx, browser.SetPermission, browser.SetPermissionParams{
					Permission: &browser.PermissionDescriptor{Name: name}, Setting: browser.PermissionSettingGranted, Origin: base,
				})
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	return tb
}

// call runs f with a minute to finish, failing the test on its error.
func (tb *tab) call(f func(ctx context.Context) error) {
	tb.t.Helper()
	ctx, cancel := context.WithTimeout(tb.ctx, time.Minute)
	defer cancel()
	if err := f(ctx); err != nil {
		tb.t.Fatal(err)
	}
}

// do runs the actions in order.
func (tb *tab) do(actions ...chromedp.Action[chromedp.Void]) {
	tb.t.Helper()
	tb.call(func(ctx context.Context) error { return chromedp.Do(ctx, actions...) })
}

// eval is the value of expression in the page.
func eval[T any](tb *tab, expression string) T {
	tb.t.Helper()
	var v T
	tb.call(func(ctx context.Context) (err error) {
		v, err = chromedp.Run(ctx, chromedp.Evaluate[T](expression, chromedp.EvalAwaitPromise))
		return err
	})
	return v
}

// at is the tab's URL, path and query, from its history: no script needed.
func (tb *tab) at() string {
	tb.t.Helper()
	var h page.GetNavigationHistoryResult
	tb.call(func(ctx context.Context) (err error) {
		h, err = chromedp.Run(ctx, chromedp.NavigationEntries())
		return err
	})
	return strings.TrimPrefix(h.Entries[h.CurrentIndex].URL, tb.base)
}

// html is the outer HTML of the first element sel matches, waiting for it.
func (tb *tab) html(sel string) string {
	tb.t.Helper()
	var s string
	tb.call(func(ctx context.Context) (err error) {
		s, err = chromedp.Run(ctx, chromedp.OuterHTML(chromedp.CSS(sel)))
		return err
	})
	return s
}

// count is how many elements sel matches now.
func (tb *tab) count(sel string) int {
	tb.t.Helper()
	var nodes []*chromedp.Node
	tb.call(func(ctx context.Context) (err error) {
		nodes, err = chromedp.Run(ctx, chromedp.Nodes(chromedp.CSSAll(sel), chromedp.AtLeast(0)))
		return err
	})
	return len(nodes)
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

// wait waits until sel matches.
func (tb *tab) wait(sel string) {
	tb.t.Helper()
	tb.do(chromedp.WaitReady(chromedp.CSS(sel)))
}

// requests checks the page requests since the last check: their targets,
// and that htmx sent them or not.
func (tb *tab) requests(step string, htmx bool, targets ...string) {
	tb.t.Helper()
	seen := tb.log.take()
	var got []string
	for _, r := range seen {
		got = append(got, r.target)
		if r.htmx != htmx {
			tb.t.Errorf("%s: %s with HX-Request %v; want %v", step, r.target, r.htmx, htmx)
		}
	}
	if !slices.Equal(got, targets) {
		tb.t.Errorf("%s: requests %q; want %q", step, got, targets)
	}
}

// violation is one axe finding: the rule, its impact, where, and axe's
// account of the first place.
type violation struct {
	ID      string   `json:"id"`
	Impact  string   `json:"impact"`
	Help    string   `json:"help"`
	Targets []string `json:"targets"`
	Summary string   `json:"summary"`
}

// audit runs axe on the page as it is now, in the light color scheme and
// in the dark, and fails on any violation: WCAG 2.2 A and AA, and axe's
// best practices. An answer the plate swapped in is drawn in by
// animations that end, and the audit waits for them: a frame of one is no
// state of the page. Without JavaScript, the page's scripts never ran;
// scripts are let run for the audit alone, which runs none of the page's.
func (tb *tab) audit(state string) {
	tb.t.Helper()
	if !tb.js {
		tb.scripts(true)
		defer tb.scripts(false)
	}
	if !eval[bool](tb, `typeof axe === "object"`) {
		tb.call(func(ctx context.Context) error {
			_, err := chromedp.Run(ctx, chromedp.Evaluate[chromedp.Void](axeJS))
			return err
		})
	}
	for _, scheme := range []string{"light", "dark"} {
		tb.call(func(ctx context.Context) error {
			_, err := chromedp.Call(ctx, emulation.SetEmulatedMedia, emulation.SetEmulatedMediaParams{
				Features: []*emulation.MediaFeature{{Name: "prefers-color-scheme", Value: scheme}},
			})
			return err
		})
		found := eval[[]violation](tb, `Promise.all(document.getAnimations()
			.filter(a => a.effect && Number.isFinite(a.effect.getComputedTiming().endTime))
			.map(a => a.finished.catch(() => {}))
		).then(() => axe.run(document, {resultTypes: ["violations"]})).then(r => r.violations.map(v => ({
			id: v.id, impact: v.impact, help: v.help, targets: v.nodes.map(n => n.target.join(" ")), summary: v.nodes[0].failureSummary,
		})))`)
		for _, v := range found {
			tb.t.Errorf("%s, %s: axe %s (%s): %s, at %q: %s", state, scheme, v.ID, v.Impact, v.Help, v.Targets, v.Summary)
		}
	}
	tb.call(func(ctx context.Context) error {
		_, err := chromedp.Call(ctx, emulation.SetEmulatedMedia, emulation.SetEmulatedMediaParams{})
		return err
	})
}

// history goes delta steps back (negative) or forward in the tab's history,
// without waiting for a load: a page from the back/forward cache, or one
// htmx restores, fires none.
func (tb *tab) history(delta int) {
	tb.t.Helper()
	tb.call(func(ctx context.Context) error {
		h, err := chromedp.Run(ctx, chromedp.NavigationEntries())
		if err != nil {
			return err
		}
		i := int(h.CurrentIndex) + delta
		if i < 0 || i >= len(h.Entries) {
			tb.t.Fatalf("history %+d from entry %d of %d", delta, h.CurrentIndex, len(h.Entries))
		}
		_, err = chromedp.Call(ctx, page.NavigateToHistoryEntry, page.NavigateToHistoryEntryParams{EntryID: h.Entries[i].ID})
		return err
	})
}

// scripts lets the page run scripts, or not.
func (tb *tab) scripts(on bool) {
	tb.t.Helper()
	tb.call(func(ctx context.Context) error {
		_, err := chromedp.Call(ctx, emulation.SetScriptExecutionDisabled, emulation.SetScriptExecutionDisabledParams{Value: !on})
		return err
	})
}

// waitText waits, up to 10 s, for the text of the element sel matches to
// be want, and fails with what it is instead.
func (tb *tab) waitText(sel, want string) {
	tb.t.Helper()
	js, err := json.Marshal([2]string{sel, want})
	if err != nil {
		tb.t.Fatal(err)
	}
	got := eval[string](tb, `new Promise(resolve => {
		const [sel, want] = `+string(js)+`, start = Date.now();
		(function poll() {
			const el = document.querySelector(sel), text = el ? el.textContent : "(no " + sel + ")";
			if (text === want || Date.now() - start > 10000) resolve(text); else setTimeout(poll, 50);
		})();
	})`)
	if got != want {
		tb.t.Errorf("%s: %q; want %q", sel, got, want)
	}
}

// browserForm runs the page's VIN form in a browser, each run in a fresh
// profile: without JavaScript every step is a page the browser loads, and
// with it the plate asks the server for each answer as a fragment
// (HX-Request), the page never reloading. Its dialogs open either way.
// Then the plate is typed on, key by key, and pasted into. axe audits
// each state. CI (CI set) needs the browser; a machine without one skips.
func browserForm(t *testing.T, srv *httptest.Server) {
	path := chromePath()
	if path == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("no Chrome or Chromium to run the form in: set NITROVIN_CHROME")
		}
		t.Skip("no Chrome or Chromium: set NITROVIN_CHROME to run the form in a browser")
	}
	log := &requestLog{h: srv.Config.Handler}
	bsrv := httptest.NewServer(log)
	t.Cleanup(bsrv.Close)
	t.Run("js-off", func(t *testing.T) { formJSOff(t, path, bsrv.URL, log) })
	t.Run("js-on", func(t *testing.T) { formJSOn(t, path, bsrv.URL, log) })
	t.Run("plate", func(t *testing.T) { plateJSOn(t, path, bsrv.URL, log) })
	t.Run("paste", func(t *testing.T) { pasteJSOn(t, path, bsrv.URL, log) })
}

const (
	vin = "1HGCM82633A004352"
	// submitted is the form's request for vin.
	submitted = "/?from.vin=" + vin
	// oldLink is an old link to the decode form, and movedLink where it
	// lands.
	oldLink   = "/?vin=5YJ3E1EA2JF000316"
	movedLink = "/?from.vin=5YJ3E1EA2JF000316"
	// emptyPlate matches the answer to a plate with nothing on it.
	emptyPlate = `#answer[data-vin=""]`
)

// dialogs opens each of an answer's dialogs from its button, audits it
// open, and closes it by its own button: popovers, which need no script.
// The JSON's button is in the Developers dialog, which stays open under
// it.
func (tb *tab) dialogs(step string) {
	tb.t.Helper()
	open := func(dialog, opener string) {
		tb.t.Helper()
		tb.do(chromedp.Click(chromedp.CSS(opener)))
		tb.wait(dialog + ":popover-open")
		tb.audit(step + ", " + dialog + " open")
	}
	shut := func(dialog string) {
		tb.t.Helper()
		tb.do(chromedp.Click(chromedp.CSS(dialog + " .dialog-head button")))
		tb.wait(dialog + ":not(:popover-open)")
	}
	open("#vocabularies", "button.open-vocabularies")
	shut("#vocabularies")
	open("#check-sum", "button.how")
	shut("#check-sum")
	open("#developers", "button.open-developers")
	open("#result-json", "button.open-dialog")
	shut("#result-json")
	shut("#developers")
}

// vins opens the header's VINs menu, a popover.
func (tb *tab) vins() {
	tb.t.Helper()
	tb.do(chromedp.Click(chromedp.CSS("button.open-vins")))
	tb.wait("#vins:popover-open")
}

// send puts text in the plate's field with no key typed, as a browser's
// autofill does, and presses Enter as a keyboard does: the key down with
// its character, which the browser types only when the page lets the key
// through, and up. chromedp's own SetValue tells the page of an input,
// which the plate answers without a submit, and its keys send the
// character apart from the key, cancelled or not.
func (tb *tab) send(text string) {
	tb.t.Helper()
	eval[bool](tb, `(document.getElementById("from-vin").value = `+jsString(text)+`, true)`)
	tb.enter()
}

// enter presses Enter in the plate's field (send).
func (tb *tab) enter() {
	tb.t.Helper()
	tb.do(chromedp.Focus(chromedp.CSS("#from-vin")))
	tb.call(func(ctx context.Context) error {
		key := input.DispatchKeyEventParams{Type: input.DispatchKeyEventTypeKeyDown, Key: "Enter", Code: "Enter", WindowsVirtualKeyCode: 13, Text: "\r", UnmodifiedText: "\r"}
		if _, err := chromedp.Call(ctx, input.DispatchKeyEvent, key); err != nil {
			return err
		}
		key.Type, key.Text, key.UnmodifiedText = input.DispatchKeyEventTypeKeyUp, "", ""
		_, err := chromedp.Call(ctx, input.DispatchKeyEvent, key)
		return err
	})
}

// formJSOff: the page without JavaScript, in a fresh profile. Each step is
// a page; no copy buttons appear; back and forward are the browser's own,
// an example in the VINs menu is a link to the page with it on the plate,
// and an old link moves to the page.
func formJSOff(t *testing.T, path, base string, log *requestLog) {
	tb := newTab(t, path, base, log, false)

	tb.do(chromedp.Navigate(base + "/"))
	tb.wait(emptyPlate)
	if n := tb.count("button.copy"); n != 0 {
		t.Errorf("/: %d copy buttons; JavaScript ran", n)
	}
	tb.requests("/", false, "/")
	tb.audit("/")

	tb.do(chromedp.SetValue(chromedp.CSS("#from-vin"), vin), chromedp.Click(chromedp.CSS(`#vin-form button[type=submit]`)))
	tb.wait("#copy-url")
	if got := tb.at(); got != submitted {
		t.Errorf("submit: at %s; want %s", got, submitted)
	}
	if !strings.Contains(tb.html("main"), "<dt>Make</dt><dd>HONDA</dd>") {
		t.Errorf("submit: no Make HONDA")
	}
	tb.requests("submit", false, submitted)
	tb.audit("decode")
	tb.dialogs("decode, JS off")

	// Back is the empty page again, forward the decode.
	tb.history(-1)
	tb.wait(emptyPlate)
	if got := tb.at(); got != "/" {
		t.Errorf("back: at %s; want /", got)
	}
	tb.history(+1)
	tb.wait("#copy-url")
	if got := tb.at(); got != submitted {
		t.Errorf("forward: at %s; want %s", got, submitted)
	}
	tb.log.take() // the browser may or may not ask again: its cache's choice

	// A refusal is the page with the refusal below the plate: a query
	// the server cannot read.
	const refused = "/?from.vin=" + vin + "&%zz"
	tb.do(chromedp.Navigate(base + refused))
	tb.wait("#answer .error")
	tb.requests("refusal", false, refused)
	tb.audit("refusal")

	// The VINs menu opens without a script, and an example in it is a link
	// to the page with that VIN on the plate.
	tb.vins()
	tb.audit("the VINs menu")
	examples := attributes(tb, "#examples a.example")
	if len(examples) == 0 {
		t.Fatal("the VINs menu: no example")
	}
	example := examples[0]["href"]
	tb.do(chromedp.Click(chromedp.CSS("#examples a.example")))
	tb.wait("#copy-url")
	if got := tb.at(); got != example {
		t.Errorf("example: at %s; want %s", got, example)
	}
	tb.requests("example", false, example)

	// An old link lands on the same decode on the page.
	tb.do(chromedp.Navigate(base + oldLink))
	tb.wait(`#answer[data-vin="` + strings.TrimPrefix(movedLink, "/?from.vin=") + `"]`)
	if got := tb.at(); got != movedLink {
		t.Errorf("old link: at %s; want %s", got, movedLink)
	}
	tb.requests("old link", false, oldLink, movedLink)
}

// formJSOn: the page with JavaScript, in a fresh profile. Enter sends the
// form: the plate asks for the answer as a fragment (HX-Request), swaps it
// in and pushes its URL, and the page never reloads; copy buttons come
// with each answer; back and forward ask the server for each state the
// same way; the VIN that decoded is kept in this browser, and the VINs
// menu offers it; nothing is refused by CSP and nothing throws.
func formJSOn(t *testing.T, path, base string, log *requestLog) {
	tb := newTab(t, path, base, log, true)

	tb.do(chromedp.Navigate(base + "/"))
	tb.wait("#vin-form.live")
	tb.wait("#mcp-local + button.copy")
	tb.requests("/", false, "/")
	tb.audit("/")
	// Laid over the cells, the plate answers as it is typed: the key that
	// sends it without a script is not drawn, so the Tab key stops on
	// nothing a visitor cannot see.
	if shown := eval[string](tb, `getComputedStyle(document.querySelector("#vin-form .decode")).display`); shown != "none" {
		t.Errorf("/: with a script the Decode key is drawn %q; want it not drawn, and so out of the Tab order", shown)
	}
	// A page that reloads loses this.
	eval[bool](tb, `window.stayed = true`)

	tb.send(vin)
	tb.wait("#copy-url + button.copy")
	if got := tb.at(); got != submitted {
		t.Errorf("submit: at %s; want %s", got, submitted)
	}
	if !strings.Contains(tb.html("main"), "<dt>Make</dt><dd>HONDA</dd>") {
		t.Errorf("submit: no Make HONDA")
	}
	if title := eval[string](tb, `document.title`); title != "2003 HONDA Accord · nitrovin" {
		t.Errorf("submit: title %q", title)
	}
	tb.requests("submit", true, submitted)
	tb.audit("decode")
	tb.dialogs("decode")

	// A copy button copies its text and says so: the decode's JSON, in
	// its dialog.
	tb.do(chromedp.Click(chromedp.CSS("button.open-developers")))
	tb.wait("#developers:popover-open")
	tb.do(chromedp.Click(chromedp.CSS("button.open-dialog")))
	tb.wait("#result-json:popover-open")
	tb.do(chromedp.Click(chromedp.CSS(`#decode-json + button.copy`)))
	tb.waitText("#copy-status", "Copied the decode's JSON")
	if got, want := eval[string](tb, `navigator.clipboard.readText()`), eval[string](tb, `document.getElementById("decode-json").textContent`); got != want {
		t.Errorf("copy: clipboard %q; want the decode's JSON", got)
	}
	tb.do(chromedp.Click(chromedp.CSS("#result-json .dialog-head button")))
	tb.wait("#result-json:not(:popover-open)")
	tb.do(chromedp.Click(chromedp.CSS("#developers .dialog-head button")))
	tb.wait("#developers:not(:popover-open)")

	// Back and forward ask the server for each state, a fragment the
	// plate swaps in: live copy buttons, one each, and one main and one
	// Vocabularies button.
	eval[bool](tb, `history.back(), true`)
	tb.wait(emptyPlate)
	if got := tb.at(); got != "/" {
		t.Errorf("back: at %s; want /", got)
	}
	eval[bool](tb, `history.forward(), true`)
	tb.wait("#copy-url + button.copy")
	if got := tb.at(); got != submitted {
		t.Errorf("forward: at %s; want %s", got, submitted)
	}
	if copies, buttons := tb.count("[data-copy]"), tb.count("button.copy"); copies != buttons {
		t.Errorf("forward: %d copy buttons for %d texts", buttons, copies)
	}
	tb.do(chromedp.Click(chromedp.CSS("button.open-developers")))
	tb.wait("#developers:popover-open")
	tb.do(chromedp.Click(chromedp.CSS(`#copy-cli + button.copy`)))
	tb.waitText("#copy-status", "Copied the nitrovin decode command")
	tb.do(chromedp.Click(chromedp.CSS("#developers .dialog-head button")))
	tb.wait("#developers:not(:popover-open)")
	if mains, buttons := tb.count("main"), tb.count("button.open-vocabularies"); mains != 1 || buttons != 1 {
		t.Errorf("forward: %d mains, %d Vocabularies buttons", mains, buttons)
	}
	tb.requests("back and forward", true, "/", submitted)

	// The VIN that decoded is kept in this browser: the VINs menu offers
	// it by its vehicle's name, in the examples' place.
	tb.vins()
	tb.waitText("#recent a", "2003 HONDA Accord")
	if got := attributes(tb, "#recent a"); len(got) != 1 || got[0]["href"] != submitted {
		t.Errorf("the VINs menu: recent %v; want one link to %s", got, submitted)
	}
	if n := tb.count("#examples:not([hidden])"); n != 0 {
		t.Errorf("the VINs menu: the examples show beside a recent VIN")
	}
	tb.audit("the VINs menu, a recent VIN")

	if !eval[bool](tb, `window.stayed === true`) {
		t.Errorf("the page reloaded")
	}
	if refused := eval[[]string](tb, `refused`); len(refused) > 0 {
		t.Errorf("CSP refused %q", refused)
	}
	tb.mu.Lock()
	if len(tb.thrown) > 0 {
		t.Errorf("thrown: %q", tb.thrown)
	}
	tb.mu.Unlock()

	// An old link lands on the same decode on the page.
	tb.do(chromedp.Navigate(base + oldLink))
	tb.wait(`#answer[data-vin="` + strings.TrimPrefix(movedLink, "/?from.vin=") + `"]`)
	if got := tb.at(); got != movedLink {
		t.Errorf("old link: at %s; want %s", got, movedLink)
	}
	tb.requests("old link", false, oldLink, movedLink)
}

// plateJSOn: the plate with JavaScript, in a fresh profile (pkg/ui's
// plate.js). Each character typed is answered without a submit, the card
// of a group filled once its last character is typed, and kept as the
// server keeps it: letters and digits, upper case, no 18th. The URL
// follows what is typed in place, and the whole VIN alone is a history
// entry, which Back returns from and forward to. Enter asks at once for
// what the plate holds, and adds no entry for the VIN the page is at.
// Every answer is a fragment (HX-Request), and the page never reloads.
func plateJSOn(t *testing.T, path, base string, log *requestLog) {
	tb := newTab(t, path, base, log, true)
	tb.do(chromedp.Navigate(base + "/"))
	tb.wait("#vin-form.live")
	tb.requests("/", false, "/")
	eval[bool](tb, `window.stayed = true`)
	entries := eval[int](tb, `history.length`)
	field := chromedp.CSS("#from-vin")
	target := func(typed string) string { return "/?from.vin=" + typed }
	// held checks where the tab is and what the plate holds.
	held := func(step, want string, entries int) {
		t.Helper()
		if at, value, n := tb.at(), eval[string](tb, `document.getElementById("from-vin").value`), eval[int](tb, `history.length`); at != target(want) || value != want || n != entries {
			t.Errorf("%s: at %s, the plate holds %q, %d history entries; want %s, %q, %d", step, at, value, n, target(want), want, entries)
		}
	}

	// The VIN a key at a time, in lower case: each is answered, once.
	cards := map[int]string{
		3:  "<dt>Make</dt><dd>HONDA</dd>",
		8:  "<dt>Model</dt><dd>Accord</dd>",
		11: "<dt>City</dt><dd>MARYSVILLE, OHIO</dd>",
		17: `<p class="big">Checks out</p>`,
	}
	var asked []string
	for i := range len(vin) {
		typed := vin[:i+1]
		tb.do(chromedp.SendKeys(field, strings.ToLower(vin[i:i+1])))
		tb.wait(`#answer[data-vin="` + typed + `"]`)
		asked = append(asked, target(typed))
		if want, ok := cards[len(typed)]; ok && !strings.Contains(tb.html("#answer"), want) {
			t.Errorf("%s typed: the answer lacks %s", typed, want)
		}
		if len(typed) == 3 {
			// A character no VIN holds is not typed, and the URL follows
			// in place: no entry.
			tb.do(chromedp.SendKeys(field, "-"))
			held("the maker typed", typed, entries)
		}
	}
	// The whole VIN takes no more, and is one entry more.
	tb.do(chromedp.SendKeys(field, "9"))
	held("the VIN typed", vin, entries+1)
	tb.requests("typing", true, asked...)

	// Back from the whole VIN is the plate as it stood before it, asked
	// of the server again, and forward the VIN.
	before := vin[:len(vin)-1]
	eval[bool](tb, `history.back(), true`)
	tb.wait(`#answer[data-vin="` + before + `"]`)
	held("back", before, entries+1)
	tb.requests("back", true, target(before))
	eval[bool](tb, `history.forward(), true`)
	tb.wait(`#answer[data-vin="` + vin + `"]`)
	held("forward", vin, entries+1)
	tb.requests("forward", true, submitted)

	// Enter asks at once for what the plate holds, a new answer, and the
	// VIN it is at already is no new entry.
	eval[bool](tb, `window.answered = document.getElementById("answer"), true`)
	tb.enter()
	tb.until("Enter", `document.getElementById("answer") !== window.answered`)
	held("Enter", vin, entries+1)
	tb.requests("Enter", true, submitted)

	if !eval[bool](tb, `window.stayed === true`) {
		t.Errorf("the page reloaded")
	}
	if refused := eval[[]string](tb, `refused`); len(refused) > 0 {
		t.Errorf("CSP refused %q", refused)
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if len(tb.thrown) > 0 {
		t.Errorf("thrown: %q", tb.thrown)
	}
}

// pasteJSOn: pasting into the plate with JavaScript, in a fresh profile
// (plate.js's paste). Each paste is a paste event on the field with its
// text on the clipboard, which the plate takes for its own. A VIN pasted
// with spaces, dashes and lower case is its 17 characters, answered once
// and one history entry; a whole VIN pasted over another replaces it;
// pasted into a plate partly typed, as much goes in at the caret as there
// is room for; and text with no letter or digit changes nothing.
func pasteJSOn(t *testing.T, path, base string, log *requestLog) {
	tb := newTab(t, path, base, log, true)
	tb.do(chromedp.Navigate(base + "/"))
	tb.wait("#vin-form.live")
	tb.requests("/", false, "/")
	eval[bool](tb, `window.stayed = true`)
	// The field's input events are counted: each is one change to the
	// plate, and the plate asks for an answer after one alone.
	eval[bool](tb, `(window.inputs = 0, document.addEventListener("input", (e) => { if (e.target.id === "from-vin") inputs++; }), true)`)
	entries := eval[int](tb, `history.length`)
	const other = "5YJ3E1EA2JF000316"
	target := func(held string) string { return "/?from.vin=" + held }
	// paste pastes text into the field, where its caret or selection is,
	// and checks that the plate took the paste from the browser.
	paste := func(step, text string) {
		t.Helper()
		if !eval[bool](tb, `(() => {
			const field = document.getElementById("from-vin"), data = new DataTransfer();
			field.focus();
			data.setData("text/plain", `+jsString(text)+`);
			return !field.dispatchEvent(new ClipboardEvent("paste", { clipboardData: data, bubbles: true, cancelable: true }));
		})()`) {
			t.Errorf("%s: the paste was left to the browser", step)
		}
	}
	// held waits for want's answer and checks where the tab is, what the
	// plate holds and shows, and how many changes it took.
	held := func(step, want string, entries, inputs int) {
		t.Helper()
		tb.wait(`#answer[data-vin="` + want + `"]`)
		tb.until(step, `location.pathname + location.search === `+jsString(target(want)))
		at, value, n := tb.at(), eval[string](tb, `document.getElementById("from-vin").value`), eval[int](tb, `history.length`)
		if at != target(want) || value != want || n != entries {
			t.Errorf("%s: at %s, the plate holds %q, %d history entries; want %s, %q, %d", step, at, value, n, target(want), want, entries)
		}
		if cells := eval[string](tb, `[...document.querySelectorAll("#vin-form .cell")].map((c) => c.firstElementChild.textContent).join("")`); cells != want {
			t.Errorf("%s: the cells show %q; want %q", step, cells, want)
		}
		if got := eval[int](tb, `inputs`); got != inputs {
			t.Errorf("%s: %d changes to the plate; want %d", step, got, inputs)
		}
	}

	// A VIN as it is copied from a listing, into the empty plate: its 17
	// characters, asked for once, one entry.
	paste("a VIN pasted", " 1hgcm-82633 a004352\n")
	held("a VIN pasted", vin, entries+1, 1)
	if body := tb.html("#answer"); !strings.Contains(body, "<dt>Make</dt><dd>HONDA</dd>") || !strings.Contains(body, `<p class="big">Checks out</p>`) {
		t.Errorf("a VIN pasted: not its answer\n%s", body)
	}
	tb.requests("a VIN pasted", true, submitted)

	// Another whole VIN replaces it, wherever the caret is: here after
	// the last character, where nothing typed would go in.
	paste("a VIN pasted over a VIN", strings.ToLower(other[:9])+" - "+other[9:])
	held("a VIN pasted over a VIN", other, entries+2, 2)
	if body := tb.html("#answer"); !strings.Contains(body, "<dt>Make</dt><dd>TESLA</dd>") {
		t.Errorf("a VIN pasted over a VIN: not its answer\n%s", body)
	}
	tb.requests("a VIN pasted over a VIN", true, target(other))

	// Less than a VIN goes in over the selection: here all of it.
	eval[bool](tb, `(document.getElementById("from-vin").select(), true)`)
	paste("three characters pasted over a VIN", vin[:3])
	held("three characters pasted over a VIN", vin[:3], entries+3, 3)
	tb.requests("three characters pasted over a VIN", true, target(vin[:3]))

	// More than fits, pasted after them: the 14 there is room for, and
	// the rest is dropped.
	paste("more than fits pasted", strings.ToLower(vin[3:10])+"-"+vin[10:]+"9Z")
	held("more than fits pasted", vin, entries+4, 4)
	tb.requests("more than fits pasted", true, submitted)

	// Text with no letter or digit changes nothing, with the caret in the
	// plate or a selection it would otherwise replace: no change, so no
	// request, and the selection as it was.
	for _, selection := range [][2]int{{17, 17}, {5, 5}, {2, 9}} {
		step := fmt.Sprintf("nothing pasted at %d-%d", selection[0], selection[1])
		eval[bool](tb, fmt.Sprintf(`(document.getElementById("from-vin").setSelectionRange(%d, %d), true)`, selection[0], selection[1]))
		paste(step, " -- \t\n.")
		held(step, vin, entries+4, 4)
		if got := eval[[]int](tb, `(() => { const f = document.getElementById("from-vin"); return [f.selectionStart, f.selectionEnd]; })()`); !slices.Equal(got, selection[:]) {
			t.Errorf("%s: the selection is %v; want %v", step, got, selection)
		}
	}
	tb.requests("nothing pasted", true)

	if !eval[bool](tb, `window.stayed === true`) {
		t.Errorf("the page reloaded")
	}
	if refused := eval[[]string](tb, `refused`); len(refused) > 0 {
		t.Errorf("CSP refused %q", refused)
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if len(tb.thrown) > 0 {
		t.Errorf("thrown: %q", tb.thrown)
	}
}
