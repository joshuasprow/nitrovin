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
// (https://github.com/joshuasprow/nitrovin/issues/320), without JavaScript
// and with it, on a server that loaded the fixture and three vocabularies
// with no name for the VIN, four stacks in all. On a wide page (1280) the
// six cards share one row, each holding its text with every value under
// its label, the stacks are one block, and nothing scrolls sideways. On a narrow one (768, and a phone's 390) the cards and the
// names are one region that scrolls sideways, each part of a name under
// its card, while a vocabulary's label stays put. The names block shows
// three vocabularies, two on a phone, and the rest at a press of its
// switch. "Check years" brings years that are unsure into view, out of a
// closed stack and from beyond the region's edge. axe audits each state.
// CI (CI set) needs the browser; a machine without one skips.
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
			for _, w := range []struct {
				width, height int64
				shown         int
			}{{768, 1024, 3}, {390, 844, 2}} {
				tb.narrow(w.width, w.height, w.shown)
			}
			tb.mu.Lock()
			defer tb.mu.Unlock()
			if len(tb.thrown) > 0 {
				t.Errorf("thrown: %q", tb.thrown)
			}
		})
	}
}

// ramPartial is the fixture's Ram before its year is typed: fleet's name
// for it holds to 2012, so its years are unsure.
const ramPartial = "/?from.vin=1D7RV1GT5"

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
	// Gaps are the room above each displayed stack's strip after the
	// first, from the stack before it.
	Gaps []float64 `json:"gaps"`
	// Parts are the first name's make, model and years; Label its
	// vocabulary's label.
	Parts []box `json:"parts"`
	Label box   `json:"label"`
	// Region is the cards' and names' region: how wide it is and how wide
	// what it holds, its name and whether a keyboard reaches it.
	RegionWidth  float64 `json:"regionWidth"`
	RegionScroll float64 `json:"regionScroll"`
	RegionName   string  `json:"regionName"`
	RegionTab    int     `json:"regionTab"`
	// PageScroll is how wide the page's own content is, and Window the
	// window.
	PageScroll float64 `json:"pageScroll"`
	Window     float64 `json:"window"`
	// Shown is how many vocabularies' stacks are displayed, of Stacks; More
	// the words on the block's switch, in lower case (the tape's are
	// capitals).
	Shown  int    `json:"shown"`
	Stacks int    `json:"stacks"`
	More   string `json:"more"`
}

const layoutJS = `(() => {
	const box = el => { const r = el.getBoundingClientRect(); return {left: r.left, right: r.right, top: r.top, bottom: r.bottom}; };
	const hang = document.querySelector(".hang"), stacks = [...document.querySelectorAll("#aka li.stack")];
	const cards = [...document.querySelectorAll(".cards .card")];
	const plate = document.querySelector("#aka .tagplate:not(.blank)");
	const more = document.querySelector("label.more-voc");
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
		gaps: stacks.filter(s => s.checkVisibility()).slice(1).map(s => s.querySelector(".tagplate").getBoundingClientRect().top - s.previousElementSibling.getBoundingClientRect().bottom),
		parts: ["wmi", "vds", "year"].map(g => box(plate.querySelector(".seg.g-" + g))),
		label: box(plate.querySelector(".lbl")),
		regionWidth: hang.clientWidth, regionScroll: hang.scrollWidth,
		regionName: hang.getAttribute("role") + ": " + hang.getAttribute("aria-label"), regionTab: hang.tabIndex,
		pageScroll: document.documentElement.scrollWidth, window: innerWidth,
		shown: stacks.filter(s => s.checkVisibility()).length, stacks: stacks.length,
		more: more && more.checkVisibility() ? more.innerText.trim().toLowerCase() : "",
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

// wide: at 1280 the cards are one row across the page and nothing scrolls
// sideways; three of the four vocabularies show, and the fourth at a press
// of the block's switch, which needs no script.
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
	// The stacks are one block: a blank strip stands 8px off it, and
	// nothing stands further.
	for i, g := range l.Gaps {
		if g > 8 {
			tb.t.Errorf("%s: %vpx above stack %d; want the stacks one block, no more than 8px between them", step, g, i+2)
		}
	}
	tb.more(step, l, 3, "1 more vocabulary")
	tb.audit(step)
}

// more checks the names block shows shown of its four vocabularies and
// says how many more there are, and that a press of its switch shows them
// all.
func (tb *tab) more(step string, l laid, shown int, words string) {
	tb.t.Helper()
	if l.Stacks != 4 || l.Shown != shown || l.More != words {
		tb.t.Errorf("%s: %d of %d vocabularies shown, the switch %q; want %d of 4 and %q", step, l.Shown, l.Stacks, l.More, shown, words)
	}
	tb.do(chromedp.Click(chromedp.CSS("label.more-voc")))
	tb.wait("#aka-more:checked")
	if l := tb.layout(); l.Shown != 4 || l.More != "fewer vocabularies" {
		tb.t.Errorf("%s, the switch pressed: %d vocabularies shown, the switch %q; want 4 and %q", step, l.Shown, l.More, "fewer vocabularies")
	}
}

// narrow: at width the cards and the names are one region that scrolls
// sideways, the cards one row in the VIN's order and each part of a name
// under its card; scrolled, they move together and the vocabulary's label
// stays. shown vocabularies show before the block's switch. Then "Check
// years" on the Ram before its year is typed.
func (tb *tab) narrow(width, height int64, shown int) {
	tb.t.Helper()
	step := strconv.FormatInt(width, 10) + " wide"
	tb.window(width, height)
	tb.load(ramNamed)
	l := tb.layout()
	tb.oneRow(step, l)
	if l.RegionScroll <= l.RegionWidth || l.PageScroll > l.Window {
		tb.t.Errorf("%s: the region holds %v in %v, the page %v in %v; want the region alone to scroll sideways", step, l.RegionScroll, l.RegionWidth, l.PageScroll, l.Window)
	}
	if l.RegionName != "region: Cards and names" || l.RegionTab != 0 {
		tb.t.Errorf("%s: the region is %q, tabindex %d; want a named region a keyboard reaches", step, l.RegionName, l.RegionTab)
	}
	// The make is under Maker, the model under Vehicle and the years under
	// Model year: cards 1, 2 and 4.
	under := func(step string, l laid) {
		tb.t.Helper()
		for i, card := range []int{0, 1, 3} {
			part, c := l.Parts[i], l.Cards[card]
			if part.Left < c.Left-1 || part.Right > c.Right+1 || part.Top < c.Bottom {
				tb.t.Errorf("%s: part %d of the name at %+v; want it under card %d, at %+v", step, i+1, part, card+1, c)
			}
		}
	}
	under(step, l)

	const by = 180
	scrolled := measure[float64](tb, `(document.querySelector(".hang").scrollTo({left: `+strconv.Itoa(by)+`, behavior: "instant"}), document.querySelector(".hang").scrollLeft)`)
	s := tb.layout()
	if scrolled != by || s.Cards[0].Left != l.Cards[0].Left-by || s.Parts[0].Left != l.Parts[0].Left-by {
		tb.t.Errorf("%s: scrolled %v, the first card from %v to %v and the make from %v to %v; want all three to move %d", step, scrolled, l.Cards[0].Left, s.Cards[0].Left, l.Parts[0].Left, s.Parts[0].Left, by)
	}
	under(step+", scrolled", s)
	if s.Label.Left != l.Label.Left {
		tb.t.Errorf("%s: scrolled, the vocabulary's label moved from %v to %v; want it to stay", step, l.Label.Left, s.Label.Left)
	}
	tb.more(step, l, shown, map[int]string{3: "1 more vocabulary", 2: "2 more vocabularies"}[shown])
	tb.audit(step)
	tb.checkYears(step)
}

// checkYears follows fleet's "Check years" on the Ram before its year is
// typed. The first of fleet's unsure years are under its stack's top plate
// and, on a narrow page, in a column the region has not scrolled to: the
// link leaves them in the window, and the stack fanned out to show them.
func (tb *tab) checkYears(step string) {
	tb.t.Helper()
	step += ", Check years"
	tb.load(ramPartial)
	type seen struct {
		Links  int    `json:"links"`
		Href   string `json:"href"`
		Unsure bool   `json:"unsure"`
		In     bool   `json:"in"`
	}
	const seenJS = `(() => {
		const links = [...document.querySelectorAll("#aka a.check")], link = links.find(a => a.closest(".stack").querySelector("#fan-fleet"));
		const el = link && document.getElementById(link.getAttribute("href").slice(1));
		const r = el && el.getBoundingClientRect();
		return {
			links: links.length, href: link ? link.getAttribute("href") : "", unsure: !!el && el.matches(".seg.g-year.unsure"),
			in: !!el && el.checkVisibility({visibilityProperty: true}) && r.left >= 0 && r.right <= innerWidth && r.top >= 0 && r.bottom <= innerHeight,
		};
	})()`
	before := measure[seen](tb, seenJS)
	if before.Links != 1 || !before.Unsure || before.In {
		tb.t.Fatalf("%s: %+v; want one link, fleet's, to unsure years not yet in view", step, before)
	}
	tb.do(chromedp.Click(chromedp.CSS(`#aka a.check`)))
	// The region scrolls to them, smoothly where motion is welcome.
	if !measure[bool](tb, `new Promise(resolve => {
		const start = Date.now();
		(function poll() {
			if (`+seenJS+`.in && location.hash === `+jsString(before.Href)+`) resolve(true); else if (Date.now() - start > 10000) resolve(false); else setTimeout(poll, 50);
		})();
	})`) {
		tb.t.Errorf("%s: the unsure years at %s never came into the window: %+v", step, before.Href, measure[seen](tb, seenJS))
	}
	if got := tb.at(); got != ramPartial+before.Href {
		tb.t.Errorf("%s: at %s; want %s", step, got, ramPartial+before.Href)
	}
	tb.audit(step)
}
