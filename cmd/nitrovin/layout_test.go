package main

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

// TestPageLayout lays the page out in headless Chrome or Chromium at the
// widths its cards and names change at
// (https://github.com/joshuasprow/nitrovin/issues/320, #366 and #409),
// without JavaScript and with it, on a server that loaded the fixture and
// three vocabularies with no name for the VIN, four in all. On a wide page
// (1280) the six cards share one row, each holding its text with every
// value under its label, and nothing scrolls sideways. On a narrow one
// (768, and a phone's 390) the cards are a region that scrolls sideways.
// At every width the names are one ledger tag, a line a vocabulary, every
// one on the page and every name in view without scrolling. axe audits
// each state. CI (CI set) needs the browser; a machine without one skips.
func TestPageLayout(t *testing.T) {
	path := chromePath()
	if path == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("no Chrome or Chromium to lay the page out in: set NITROVIN_CHROME")
		}
		t.Skip("no Chrome or Chromium: set NITROVIN_CHROME to lay the page out in a browser")
	}
	fixtureData(t)
	paths := []string{fixture}
	dir := t.TempDir()
	for i := range 3 {
		name := "extra-" + strconv.Itoa(i+1)
		p := filepath.Join(dir, name+".jsonl")
		if err := os.WriteFile(p, []byte(`{"format":1,"kind":"vocabulary","name":"`+name+`","version":"1","data_month":"`+fixtureMonth+`","subject":"vehicle","fields":["make","model"]}`+"\n"+
			`{"entry":{"make":"Chevy","model":"Silverado"},"mappings":[{"model":1850}]}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	srv := httptest.NewServer(vocabServer(t, nil, paths...))
	t.Cleanup(srv.Close)
	for _, c := range []struct {
		name string
		js   bool
	}{{"js-off", false}, {"js-on", true}} {
		t.Run(c.name, func(t *testing.T) {
			tb := newTab(t, path, srv.URL, &requestLog{}, c.js)
			tb.wide()
			for _, w := range []struct{ width, height int64 }{{768, 1024}, {390, 844}} {
				tb.narrow(w.width, w.height)
			}
			tb.mu.Lock()
			defer tb.mu.Unlock()
			if len(tb.thrown) > 0 {
				t.Errorf("thrown: %q", tb.thrown)
			}
		})
	}
}

// window sets the page's window to width by height CSS pixels.
func (tb *tab) window(width, height int64) {
	tb.t.Helper()
	tb.call(func(ctx context.Context) error {
		_, err := chromedp.Call(ctx, emulation.SetDeviceMetricsOverride, emulation.SetDeviceMetricsOverrideParams{Width: width, Height: height, DeviceScaleFactor: 1})
		return err
	})
}

// load opens target and waits for its names, and for the plate's script
// where scripts run.
func (tb *tab) load(target string) {
	tb.t.Helper()
	tb.do(chromedp.Navigate(tb.base + target))
	tb.wait("#aka")
	if tb.js {
		tb.wait("#vin-form.live")
	}
}

// measure is the value of expression in the page, which the page's own
// scripts need not be running for: scripts are let run for the reading
// alone.
func measure[T any](tb *tab, expression string) T {
	tb.t.Helper()
	if !tb.js {
		tb.scripts(true)
		defer tb.scripts(false)
	}
	return eval[T](tb, expression)
}

// box is where an element is drawn, in the window's CSS pixels.
type box struct {
	Left   float64 `json:"left"`
	Right  float64 `json:"right"`
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
}

// laid is the page's layout as a visitor has it.
type laid struct {
	// Cards are the six cards in document order, and Spills the cards with
	// content drawn outside their own box.
	Cards  []box    `json:"cards"`
	Spills []string `json:"spills"`
	// Ragged counts the cards' values that do not start under their label,
	// at its leading edge.
	Ragged int `json:"ragged"`
	// Tags are the ledger's lines, a vocabulary each, in document order;
	// Names the ledger: how wide it is and how wide what it holds, and its
	// name. Out counts the names drawn outside the ledger or the window.
	Tags        []box   `json:"tags"`
	NamesWidth  float64 `json:"namesWidth"`
	NamesScroll float64 `json:"namesScroll"`
	NamesName   string  `json:"namesName"`
	Out         int     `json:"out"`
	// Region is the cards' region: how wide it is and how wide what it
	// holds, its name and whether a keyboard reaches it.
	RegionWidth  float64 `json:"regionWidth"`
	RegionScroll float64 `json:"regionScroll"`
	RegionName   string  `json:"regionName"`
	RegionTab    int     `json:"regionTab"`
	// PageScroll is how wide the page's own content is, and Window the
	// window.
	PageScroll float64 `json:"pageScroll"`
	Window     float64 `json:"window"`
}

const layoutJS = `(() => {
	const box = el => { const r = el.getBoundingClientRect(); return {left: r.left, right: r.right, top: r.top, bottom: r.bottom}; };
	const hang = document.querySelector(".hang"), names = document.querySelector("#aka .ledger");
	const cards = [...document.querySelectorAll(".cards .card")];
	return {
		cards: cards.map(box),
		spills: cards.filter(c => {
			const own = c.getBoundingClientRect();
			return [...c.querySelectorAll("h2, dt, dd, p, li, button")].some(el => {
				if (el.closest("dialog")) return false;
				const r = el.getBoundingClientRect();
				return r.left < own.left - 1 || r.right > own.right + 1 || r.bottom > own.bottom + 1;
			});
		}).map(c => c.dataset.group),
		ragged: cards.flatMap(c => [...c.querySelectorAll("dl")]).flatMap(dl => [...dl.querySelectorAll("dd")].filter(dd => { const v = dd.getBoundingClientRect(), k = dd.previousElementSibling.getBoundingClientRect(); return Math.abs(v.left - k.left) > 1 || v.top < k.bottom - 1; })).length,
		tags: [...names.querySelectorAll(".lrow")].map(box),
		namesWidth: names.clientWidth, namesScroll: names.scrollWidth, namesName: names.querySelector(".lrows").getAttribute("aria-label"),
		out: (() => { const own = names.getBoundingClientRect(); return [...names.querySelectorAll(".lname, .lmk")].filter(n => { const r = n.getBoundingClientRect(); return r.width === 0 || r.left < Math.max(own.left, 0) - 1 || r.right > Math.min(own.right, innerWidth) + 1; }).length; })(),
		regionWidth: hang.clientWidth, regionScroll: hang.scrollWidth,
		regionName: hang.getAttribute("role") + ": " + hang.getAttribute("aria-label"), regionTab: hang.tabIndex,
		pageScroll: document.documentElement.scrollWidth, window: innerWidth,
	};
})()`

func (tb *tab) layout() laid {
	tb.t.Helper()
	return measure[laid](tb, layoutJS)
}

// oneRow checks the six cards are one row, in the VIN's order.
func (tb *tab) oneRow(step string, l laid) {
	tb.t.Helper()
	if len(l.Cards) != 6 {
		tb.t.Fatalf("%s: %d cards; want 6", step, len(l.Cards))
	}
	for i, c := range l.Cards[1:] {
		if prev := l.Cards[i]; c.Top != prev.Top || c.Bottom != prev.Bottom || c.Left < prev.Right {
			tb.t.Errorf("%s: card %d at %+v, card %d at %+v; want one row, as tall as its tallest card, in order", step, i+1, prev, i+2, c)
		}
	}
	if len(l.Spills) > 0 {
		tb.t.Errorf("%s: cards %q draw text outside their own box", step, l.Spills)
	}
	if l.Ragged > 0 {
		tb.t.Errorf("%s: %d of the cards' values are not under their label, at its leading edge", step, l.Ragged)
	}
}

// wide: at 1280 the cards are one row across the page and the page does
// not scroll sideways; the four vocabularies are the ledger's lines.
func (tb *tab) wide() {
	tb.t.Helper()
	const step = "1280 wide"
	tb.window(1280, 800)
	tb.load(ramNamed)
	l := tb.layout()
	tb.oneRow(step, l)
	if l.RegionScroll > l.RegionWidth || l.PageScroll > l.Window {
		tb.t.Errorf("%s: the region holds %v in %v, the page %v in %v; want nothing to scroll sideways", step, l.RegionScroll, l.RegionWidth, l.PageScroll, l.Window)
	}
	if first, last := l.Cards[0], l.Cards[5]; first.Left < 0 || last.Right > l.Window {
		tb.t.Errorf("%s: cards from %v to %v in a window of %v; want all six in it", step, first.Left, last.Right, l.Window)
	}
	tb.tags(step, l)
	tb.audit(step)
}

// tags checks the names are one ledger tag under the cards, a line a
// vocabulary down it in order, every name in view without scrolling.
func (tb *tab) tags(step string, l laid) {
	tb.t.Helper()
	if len(l.Tags) != 4 {
		tb.t.Fatalf("%s: %d ledger lines; want 4, a vocabulary each", step, len(l.Tags))
	}
	for i, tag := range l.Tags[1:] {
		if prev := l.Tags[i]; tag.Top < prev.Bottom-1 || tag.Left != prev.Left {
			tb.t.Errorf("%s: line %d at %+v, line %d at %+v; want one under the other, in order", step, i+1, prev, i+2, tag)
		}
	}
	if l.Tags[0].Top < l.Cards[0].Bottom {
		tb.t.Errorf("%s: the first line at %+v; want it under the cards, which end at %v", step, l.Tags[0], l.Cards[0].Bottom)
	}
	if l.NamesName != "Also known as" {
		tb.t.Errorf("%s: the names are %q; want them named", step, l.NamesName)
	}
	if l.NamesScroll > l.NamesWidth+1 || l.Out > 0 {
		tb.t.Errorf("%s: the ledger holds %v in %v, %d names out of view; want every name in view without scrolling", step, l.NamesScroll, l.NamesWidth, l.Out)
	}
}

// narrow: at width the cards are a region that scrolls sideways, one row
// in the VIN's order; the names' tags are a row of their own under them.
func (tb *tab) narrow(width, height int64) {
	tb.t.Helper()
	step := strconv.FormatInt(width, 10) + " wide"
	tb.window(width, height)
	tb.load(ramNamed)
	l := tb.layout()
	tb.oneRow(step, l)
	if l.RegionScroll <= l.RegionWidth || l.PageScroll > l.Window {
		tb.t.Errorf("%s: the region holds %v in %v, the page %v in %v; want the region alone to scroll sideways", step, l.RegionScroll, l.RegionWidth, l.PageScroll, l.Window)
	}
	if l.RegionName != "region: Cards" || l.RegionTab != 0 {
		tb.t.Errorf("%s: the region is %q, tabindex %d; want a named region a keyboard reaches", step, l.RegionName, l.RegionTab)
	}
	const by = 180
	scrolled := measure[float64](tb, `(document.querySelector(".hang").scrollTo({left: `+strconv.Itoa(by)+`, behavior: "instant"}), document.querySelector(".hang").scrollLeft)`)
	s := tb.layout()
	if scrolled != by || s.Cards[0].Left != l.Cards[0].Left-by || s.Tags[0].Left != l.Tags[0].Left {
		tb.t.Errorf("%s: scrolled %v, the first card from %v to %v and the first tag from %v to %v; want the cards alone to move %d", step, scrolled, l.Cards[0].Left, s.Cards[0].Left, l.Tags[0].Left, s.Tags[0].Left, by)
	}
	tb.tags(step, s)
	tb.audit(step)
}
