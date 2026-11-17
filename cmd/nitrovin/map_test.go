package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

// linesOff are the map's lines that miss their cards as the browser lays
// the page out, "map-e3 end" each: a mapping's line leaves the middle of
// a card's right edge and reaches the middle of a card's left edge, and a
// model's tie leaves its make's foot. The server draws the lines from the
// cards' sizes, which the stylesheet sets (pkg/ui, map.go and ui.css).
const linesOff = `(() => {
	const svg = document.querySelector(".map-edges"), m = svg.getScreenCTM();
	const at = (p, l) => p.getPointAtLength(l).matrixTransform(m);
	const cards = [...document.querySelectorAll(".map .node")].map(n => n.getBoundingClientRect());
	const near = (a, b) => Math.abs(a - b) < 1;
	const out = [];
	for (const p of svg.querySelectorAll("path")) {
		const a = at(p, 0), b = at(p, p.getTotalLength());
		const starts = cards.some(r => p.classList.contains("under")
			? near(a.y, r.bottom) && a.x > r.left && a.x < r.right
			: near(a.x, r.right) && near(a.y, r.top + r.height / 2));
		const ends = cards.some(r => near(b.x, r.left) && near(b.y, r.top + r.height / 2));
		if (!starts || !ends) out.push(p.id + (starts ? "" : " start") + (ends ? "" : " end"));
	}
	return out;
})()`

// stacked reports whether the map's columns stand one above the other,
// in order, its lines and their key hidden, and each card's relation
// shown as a line of its own; and that nothing is wider than the window.
const stacked = `(() => {
	const cols = [...document.querySelectorAll(".map-col")].map(c => c.getBoundingClientRect());
	const inOrder = cols.every((r, i) => i === 0 || r.top >= cols[i - 1].bottom);
	const hidden = sel => [...document.querySelectorAll(sel)].every(e => getComputedStyle(e).display === "none");
	const shown = [...document.querySelectorAll(".map .node .rel")].every(e => e.getBoundingClientRect().height > 10);
	return inOrder && hidden(".map-edges") && hidden(".map-key .key-lines") && shown &&
		document.documentElement.scrollWidth <= innerWidth;
})()`

// accName is the accessible name the browser gives the element sel
// matches.
func accName(tb *tab, sel string) string {
	tb.t.Helper()
	var name string
	tb.call(func(ctx context.Context) error {
		nodes, err := chromedp.Run(ctx, chromedp.Nodes(chromedp.CSS(sel)))
		if err != nil {
			return err
		}
		relatives := false
		r, err := chromedp.Call(ctx, accessibility.GetPartialAXTree, accessibility.GetPartialAXTreeParams{BackendNodeID: nodes[0].BackendNodeID, FetchRelatives: &relatives})
		if err != nil || len(r.Nodes) == 0 || r.Nodes[0].Name == nil {
			return err
		}
		return json.Unmarshal(r.Nodes[0].Name.Value, &name)
	})
	return name
}

// point moves the mouse to the middle of the element sel matches,
// scrolled into view.
func (tb *tab) point(sel string) {
	tb.t.Helper()
	xy := eval[[2]float64](tb, `(e => { e.scrollIntoView({block: "center"}); const r = e.getBoundingClientRect(); return [r.x + r.width / 2, r.y + r.height / 2]; })(document.querySelector(`+jsString(sel)+`))`)
	tb.do(chromedp.MouseEvent(input.DispatchMouseEventTypeMouseMoved, xy[0], xy[1]))
}

// translationMap: a VIN's answer to a vocabulary is its translation's
// map, each line from its card to its card. The From card alone is a
// link, to the answer itself, and its name says what it is; the others
// take focus. A card focused lights the route through it, and one under
// the pointer its own until the pointer leaves, the focused card's then;
// a long name shows whole under the pointer. A narrow window stacks the
// columns, hiding the lines and showing each card's relation. An example
// is another answer, which htmx follows, and back and forward ask the
// server, so nothing left on the page by a script, here an empty style
// attribute, comes back for CSP to refuse. Without JavaScript the map is
// drawn the same. axe audits each state.
func translationMap(t *testing.T, path, base string) {
	const translation = "/?from.vin=1D7RV1GT5AS123456&to=yard"
	drawn := func(tb *tab, step string, lines int) {
		t.Helper()
		if got := tb.count(".map-edges path"); got != lines {
			t.Errorf("%s: %d lines; want %d", step, got, lines)
		}
		if off := eval[[]string](tb, linesOff); len(off) > 0 {
			t.Errorf("%s: lines miss their cards: %q", step, off)
		}
	}
	lit := func(tb *tab, step string, cards, lines int) {
		t.Helper()
		if c, l := tb.count(".map .node.lit"), tb.count(".map-edges path.lit"); c != cards || l != lines {
			t.Errorf("%s lights %d cards and %d lines; want %d and %d", step, c, l, cards, lines)
		}
	}

	tb := newTab(t, path, base, &requestLog{}, true)
	tb.do(chromedp.EmulateViewport(1440, 900))
	tb.open(translation)
	tb.wait(".map")
	// The VIN to its model, the model to its make, the model's two
	// entries and the make's one.
	drawn(tb, "the map", 5)
	if links, cards := tb.count(".map a.node[href]"), tb.count(".map .node"); links != 1 || cards != 6 {
		t.Errorf("%d of %d cards are links; want the From card alone of 6", links, cards)
	}
	if href := attributes(tb, ".map-col:nth-child(1) a.node"); len(href) != 1 || href[0]["href"] != translation {
		t.Errorf("the From card: %v; want a link to %s", href, translation)
	}
	// Its relation is part of its name; the browser spaces out each part.
	const fromName = "1D7RV1GT5AS123456 , what was translated, a link to this answer ; VIN, model year 2010"
	if got := accName(tb, ".map-col:nth-child(1) a.node"); got != fromName {
		t.Errorf("the From card's name %q; want %q", got, fromName)
	}
	tb.audit("the map")

	// Dodge Other, reached through the make, which the model led to. A
	// click first: a page never clicked has no focus to move.
	const other, ram1500 = ".map-col:nth-child(3) .node:nth-of-type(1)", ".map-col:nth-child(3) .node:nth-of-type(2)"
	tb.do(chromedp.Click(chromedp.CSS(".map-key")), chromedp.Focus(chromedp.CSS(other)))
	lit(tb, "Dodge Other focused", 4, 3)
	// The pointer over Dodge Ram 1500 lights its route, which Dodge Other
	// is not on; leaving it gives the light back to Dodge Other.
	tb.point(ram1500)
	if tb.count(ram1500+".lit") != 1 || tb.count(other+".lit") != 0 {
		t.Errorf("Dodge Ram 1500 under the pointer: its route not lit, or Dodge Other's")
	}
	tb.point(".map-key")
	lit(tb, "the pointer gone, Dodge Other focused", 4, 3)
	tb.audit("the map, a route lit")

	// A name too long for two lines shows whole under the pointer, over
	// the cards below, which stay put.
	tb.do(chromedp.Click(chromedp.CSS(".map-key")))
	const long = `document.querySelector(".map-col:nth-child(3) .node:nth-of-type(2) .name")`
	const below = `document.querySelector(".map-col:nth-child(3) .node:nth-of-type(3)").offsetTop`
	eval[bool](tb, long+`.textContent = "Dodge Ram 1500 Quad Cab Long Bed Heavy Duty Cummins Turbo Diesel Laramie Longhorn Limited Edition", true`)
	if h := eval[float64](tb, long+`.getBoundingClientRect().height`); h > 41 {
		t.Errorf("a long name: %v px; want two lines at most", h)
	}
	before := eval[float64](tb, below)
	tb.point(ram1500)
	if whole := eval[bool](tb, `(n => n.scrollHeight <= n.clientHeight && n.getBoundingClientRect().height > 41)(`+long+`)`); !whole {
		t.Errorf("a long name under the pointer is not whole")
	}
	if after := eval[float64](tb, below); after != before {
		t.Errorf("a long name under the pointer moved the card below from %v to %v", before, after)
	}
	tb.point(".map-key")

	tb.do(chromedp.EmulateViewport(390, 844))
	if !eval[bool](tb, stacked) {
		t.Errorf("the map at 390: not stacked")
	}
	tb.audit("the map, narrow")
	tb.do(chromedp.EmulateViewport(1440, 900))

	// An example is another answer, to the same vocabulary, which htmx
	// follows. A script's empty style attribute left on the page first.
	eval[bool](tb, `window.kept = true`)
	eval[bool](tb, `(n => { n.style.color = "red"; n.style.color = ""; return n.hasAttribute("style"); })(document.querySelector(".map .node"))`)
	const example = teslaExample + "&to=yard"
	tb.do(chromedp.Click(chromedp.CSS(`#controls a.example[href="` + example + `"]`)))
	tb.until("the example", `location.pathname + location.search === "`+example+`" && document.querySelector(".map .node .meta").textContent === "VIN, model year 2018"`)
	if !eval[bool](tb, `window.kept === true`) {
		t.Errorf("the example loaded as a page; want htmx's swap")
	}
	// The VIN to its model and the model to its make: yard has no Tesla.
	drawn(tb, "the map of an example", 2)

	// Back is the translation again, forward the example's, each from the
	// server: CSP refuses nothing.
	eval[bool](tb, `history.back(), true`)
	tb.until("back", `location.search === "`+translation[1:]+`" && document.querySelectorAll(".map-edges path").length === 5`)
	drawn(tb, "back", 5)
	eval[bool](tb, `history.forward(), true`)
	tb.until("forward", `location.pathname + location.search === "`+example+`" && document.querySelectorAll(".map-edges path").length === 2`)
	if !eval[bool](tb, `window.kept === true`) {
		t.Errorf("back and forward reloaded the page")
	}
	if refused := eval[[]string](tb, `refused`); len(refused) > 0 {
		t.Errorf("back and forward: CSP refused %q", refused)
	}

	off := newTab(t, path, base, &requestLog{}, false)
	off.do(chromedp.EmulateViewport(1440, 900))
	off.open(translation)
	off.wait(".map")
	if got := off.count(".map-edges path"); got != 5 {
		t.Errorf("JS off, the map: %d lines; want 5", got)
	}
	off.audit("the map, JS off")
	off.do(chromedp.EmulateViewport(390, 844))
	off.scripts(true)
	if !eval[bool](off, stacked) {
		t.Errorf("JS off, the map at 390: not stacked")
	}
	off.scripts(false)
	off.audit("the map, JS off, narrow")
}
