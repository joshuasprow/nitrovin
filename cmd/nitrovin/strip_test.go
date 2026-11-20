package main

import (
	"context"
	"fmt"
	"math"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

// stripFixture is a vPIC catalog with eighteen part types for a 2015
// F-150 and six for a 2014 Silverado, one with a name too long for its
// tape, and a part vocabulary of their groups (#324).
var stripFixture = abs("../../testdata/vocab/strip")

const (
	stripPickup    = "/?from.vin=1FTEW1EG7FFA12345"
	stripSilverado = "/?from.vin=1GCVKREC6EZ123456"
	longName       = "Battery Cable, Positive, with Fusible Link and Terminal"
)

// stripWidth is a window the strip is laid out in.
type stripWidth struct{ width, height int64 }

var stripWidths = []stripWidth{{1280, 900}, {768, 1024}, {390, 844}}

// stripShown is how many part types two rows show beside the key at each
// of stripWidths, without a script and with one, as strip.go packs them:
// a tape takes two cells where its name does not fit one, the script's
// filter takes the first cell but on a phone, and its Recent stamp
// widens a tape. 0 is a strip that holds them all, with no key.
type stripShown struct{ off, on [3]int }

var (
	// The pickup's: Brake Caliper, Front and the brake pads take two
	// cells each.
	pickupShown = stripShown{[3]int{5, 4, 2}, [3]int{5, 3, 2}}
	// Water Pump pressed first, with a script stamped Recent.
	pumpShown = stripShown{[3]int{6, 4, 3}, [3]int{5, 4, 2}}
	// The Silverado's six fit two rows of 5 columns, and of 4 without the
	// filter.
	silveradoShown = stripShown{[3]int{0, 0, 2}, [3]int{0, 4, 2}}
)

// at is the count s says for width i of stripWidths, with a script or
// without.
func (s stripShown) at(js bool, i int) int {
	if js {
		return s.on[i]
	}
	return s.off[i]
}

// TestPartTypeStrip runs the strip of part types (#324) in a browser, on a
// server that loaded the strip fixture, without JavaScript and with it.
// At 1280, 768 and 390 the F-150's eighteen part types are two rows and
// "N more", a tape taking two cells where its name does not fit one: the
// part types past them are not drawn and out of the keyboard's reach, and
// the key is drawn, on top, the next stop after the last part type, its
// name saying N. Where every part type fits, as the Silverado's do at 1280
// and at 768 without the filter, there is no key to see or reach. A name
// too long even for two cells ends in "…" and keeps its count. On the demo
// set's F-150 no name is cut at 1280 or 768.
// "N more" opens the part types dialog; a part type chosen there leads the
// strip, pressed, and the rails hang from it at every width. Without
// JavaScript there is no filter and nothing is recent. With it, typing
// narrows the strip across the part types it had no room for, the pressed
// one kept, and a filter matching nothing says so beside the pressed one,
// clear of its rails; the dialog's own filter narrows the dialog; part
// types opened lead the strip on the next VIN, stamped Recent, the stamp
// flush with a pressed one, and are listed in the dialog; the dialog's
// Clear recent forgets them, keeps the VINs and keeps the focus in the
// dialog, and the VINs menu's forgets both. axe audits each state. CI (CI
// set) needs the browser; a machine without one skips.
func TestPartTypeStrip(t *testing.T) {
	path := chromePath()
	if path == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("no Chrome or Chromium to run the page in: set NITROVIN_CHROME")
		}
		t.Skip("no Chrome or Chromium: set NITROVIN_CHROME to run the page in a browser")
	}
	srv := httptest.NewServer(vocabServer(t, nil, stripFixture))
	t.Cleanup(srv.Close)
	for _, js := range []bool{false, true} {
		name := map[bool]string{false: "js-off", true: "js-on"}[js]
		t.Run(name, func(t *testing.T) {
			tb := newTab(t, path, srv.URL, &requestLog{}, js)
			for i, w := range stripWidths {
				tb.window(w.width, w.height)
				tb.strip(stripPickup)
				tb.capped(fmt.Sprintf("%s, %d", name, w.width), pickupShown.at(js, i))
				tb.strip(stripSilverado)
				if n := silveradoShown.at(js, i); n > 0 {
					tb.capped(fmt.Sprintf("%s, the Silverado, %d", name, w.width), n)
				} else {
					tb.fits(fmt.Sprintf("%s, the Silverado, %d", name, w.width))
				}
			}
			if js {
				// Before any part type is recent, plate.js packs as the
				// server does from the same numbers: the same tapes take
				// two cells at every width, and on a phone, where its
				// filter takes no cell, the same are past the rows.
				tb.strip(stripPickup)
				if got := eval[string](tb, `fetch(location.href).then((r) => r.text()).then((html) => {
					const served = new DOMParser().parseFromString(html, 'text/html');
					const marks = (doc) => [...doc.querySelectorAll('.ptypes > li[data-part]')].map((li) =>
						li.dataset.part + ':' + [...li.classList].filter((c) => /^(s[2-5]|x2)$/.test(c)).sort().join(','));
					const a = marks(served).join(' '), b = marks(document).join(' ');
					return a === b ? '' : 'served ' + a + '\nlaid ' + b;
				})`); got != "" {
					t.Errorf("plate.js packs the strip unlike the server:\n%s", got)
				}
			}
			tb.strip(stripSilverado + "&part.id=battery-cable")
			tb.longName(name + ", the Silverado, 390")
			tb.audit(name + ", the Silverado at 390")
			tb.window(1280, 900)
			tb.strip(stripPickup)
			tb.audit(name + ", the capped strip at 1280")

			// "N more" opens the dialog, a popover; a part type chosen in
			// it leads the strip, pressed, the rails hung from it.
			tb.do(chromedp.Click(chromedp.CSS("#more-types")))
			tb.wait("#part-types:popover-open")
			tb.audit(name + ", the part types dialog")
			tb.do(chromedp.Click(chromedp.CSS(`#part-types .ptgroups a.ptype[href$="part.id=water-pump"]`)))
			tb.wait(`#answer[data-part="water-pump"] #fit .cand`)
			if got := measure[string](tb, `document.querySelector('.ptypes > li[data-part]').dataset.part + (document.querySelector('.ptypes > li[data-part] .ptype[aria-current]') ? ' pressed' : '')`); got != "water-pump pressed" {
				t.Errorf("chosen in the dialog, the strip leads with %s; want water-pump pressed", got)
			}
			if n := tb.count("#part-types:popover-open"); n != 0 {
				t.Errorf("chosen in the dialog, the dialog is still open")
			}
			for i, w := range stripWidths {
				tb.window(w.width, w.height)
				tb.strip(stripPickup + "&part.id=water-pump")
				tb.capped(fmt.Sprintf("%s, Water Pump pressed, %d", name, w.width), pumpShown.at(js, i))
				tb.hangs(fmt.Sprintf("%s, Water Pump pressed, %d", name, w.width))
			}
			tb.window(1280, 900)

			if !js {
				// Nothing is drawn that needs a script.
				if n := measure[int](tb, `document.querySelectorAll('.ptf:not([hidden]), .rstamp, .ptrecent:not([hidden])').length`); n != 0 {
					t.Errorf("without JavaScript, %d filters, Recent stamps or recent part types show", n)
				}
				return
			}

			// The filter narrows the strip across every part type, the
			// pressed one kept: Wiper Motor is past the two rows.
			tb.strip(stripPickup + "&part.id=water-pump")
			if eval[bool](tb, `document.querySelector('.ptypes > li[data-part="wiper-motor"]').getClientRects().length > 0`) {
				t.Errorf("Wiper Motor is in the two rows; want it past them")
			}
			tb.do(chromedp.SendKeys("#ptf-strip", "wip"))
			if got := drawn(tb, ".ptypes > li[data-part]"); !slices.Equal(got, []string{"water-pump", "wiper-motor"}) {
				t.Errorf("filtered by wip, the strip shows %q; want water-pump (pressed) and wiper-motor", got)
			}
			if eval[bool](tb, `document.getElementById('more-types').getClientRects().length > 0`) {
				t.Errorf("filtered to two, the strip still has its key")
			}
			tb.audit("js-on, the strip filtered")
			typeIn(tb, "#ptf-strip", "")
			tb.capped("js-on, the filter emptied", pumpShown.on[0])

			// Matching nothing, at 390, the strip says so beside the
			// pressed part type, where no rail runs.
			tb.window(390, 844)
			typeIn(tb, "#ptf-strip", "zz")
			if got := eval[string](tb, `document.querySelector('.ptypes > .ptnone:not([hidden])')?.textContent ?? ''`); got != "No part type’s name has “zz”." {
				t.Errorf("filtered by zz, the strip says %q", got)
			}
			if eval[bool](tb, `(() => {
				const fit = document.getElementById('fit').getBoundingClientRect();
				const m = document.querySelector('.ptypes > .ptnone').getBoundingClientRect();
				return [':before', ':after'].some((p) => {
					const r = getComputedStyle(document.querySelector('.risers'), p);
					const left = fit.left + parseFloat(r.left), top = fit.top + parseFloat(r.top), bottom = fit.bottom - parseFloat(r.bottom);
					return left < m.right && left + 5 > m.left && top < m.bottom && bottom > m.top;
				});
			})()`) {
				t.Errorf("filtered by zz at 390, a rail runs through the line saying so")
			}
			tb.hangs("js-on, filtered by zz, 390")
			typeIn(tb, "#ptf-strip", "")
			tb.window(1280, 900)

			// The dialog's filter narrows the dialog, and the headings
			// left with nothing go.
			tb.do(chromedp.Click(chromedp.CSS("#more-types")))
			tb.wait("#part-types:popover-open")
			typeIn(tb, "#ptf-dialog", "lamp")
			if got := drawn(tb, ".ptgroups .ptlist > li"); !slices.Equal(got, []string{"headlamp", "fog-lamp", "headlamp", "tail-lamp"}) {
				t.Errorf("the dialog filtered by lamp shows %q", got)
			}
			if got := eval[[]string](tb, `[...document.querySelectorAll('.ptgroups .ptgroup:not([hidden]) h3')].map(h => h.textContent)`); !slices.Equal(got, []string{"Electrical", "Lighting"}) {
				t.Errorf("the dialog filtered by lamp has headings %q", got)
			}
			typeIn(tb, "#ptf-dialog", "")
			tb.do(chromedp.Click(chromedp.CSS("#part-types .dialog-head button")))
			tb.wait("#part-types:not(:popover-open)")

			// Opened, a part type leads the strip on the next VIN, the
			// newest first, up to two, each stamped Recent; the dialog
			// lists every recent one the vehicle has.
			for _, part := range []string{"battery", "starter", "water-pump"} {
				tb.do(chromedp.Click(chromedp.CSS("#more-types")))
				tb.wait("#part-types:popover-open")
				tb.do(chromedp.Click(chromedp.CSS(`#part-types .ptgroups a.ptype[href$="part.id=` + part + `"]`)))
				tb.wait(`#answer[data-part="` + part + `"] .cand`)
			}
			// Pressed, a recent part type's stamp is flush with it.
			if got := eval[string](tb, `getComputedStyle(document.querySelector('.ptypes > li:has(.ptype[aria-current]) > .rstamp')).transform`); got != "matrix(1, 0, 0, 1, 0, 2)" {
				t.Errorf("the pressed part type's Recent stamp is at %s; want 2px down, flush", got)
			}
			tb.strip(stripSilverado)
			want := []string{"water-pump", "starter", "alternator", "battery", "battery-cable", "radiator"}
			if got := eval[[]string](tb, `[...document.querySelectorAll('.ptypes > li[data-part]')].map(li => li.dataset.part)`); !slices.Equal(got, want) {
				t.Errorf("the next VIN's strip %q; want %q", got, want)
			}
			if got := eval[[]string](tb, `[...document.querySelectorAll('.ptypes > li:has(> .rstamp)')].map(li => li.dataset.part)`); !slices.Equal(got, want[:2]) {
				t.Errorf("the next VIN's Recent stamps are on %q; want %q", got, want[:2])
			}
			tb.fits("js-on, the next VIN")
			tb.strip(stripPickup)
			tb.do(chromedp.Click(chromedp.CSS("#more-types")))
			tb.wait("#part-types:popover-open")
			if got := drawn(tb, ".ptrecent .ptlist > li"); !slices.Equal(got, []string{"water-pump", "starter", "battery"}) {
				t.Errorf("the next VIN's dialog lists %q as opened recently", got)
			}
			tb.audit("js-on, the next VIN's dialog, opened recently")

			// The dialog's Clear recent forgets the part types, keeps the
			// VINs, and keeps the focus in the dialog; the VINs menu's
			// forgets both.
			tb.do(chromedp.Click(chromedp.CSS(".ptrecent .clear-parts")))
			if n := tb.count(".rstamp, .ptrecent:not([hidden])"); n != 0 || eval[bool](tb, `localStorage.getItem('nitrovin.recent-parts') !== null`) {
				t.Errorf("cleared in the dialog, %d Recent stamps or recent lists show", n)
			}
			if eval[bool](tb, `localStorage.getItem('nitrovin.recent') === null`) {
				t.Errorf("cleared in the dialog, the recent VINs are gone too")
			}
			if got := eval[string](tb, `document.activeElement.id`); got != "ptf-dialog" {
				t.Errorf("cleared in the dialog, the focus is on %q; want the dialog's filter", got)
			}
			tb.do(chromedp.Click(chromedp.CSS("#part-types .dialog-head button")))
			tb.wait("#part-types:not(:popover-open)")
			tb.strip(stripSilverado + "&part.id=radiator")
			if n := tb.count(".rstamp"); n != 1 {
				t.Errorf("Radiator opened, %d Recent stamps; want 1", n)
			}
			tb.vins()
			tb.do(chromedp.Click(chromedp.CSS("#clear-recent")))
			if n := tb.count(".rstamp"); n != 0 || eval[bool](tb, `localStorage.getItem('nitrovin.recent-parts') !== null || localStorage.getItem('nitrovin.recent') !== null`) {
				t.Errorf("cleared in the VINs menu, %d Recent stamps, or something kept", n)
			}

			tb.mu.Lock()
			defer tb.mu.Unlock()
			if len(tb.thrown) > 0 {
				t.Errorf("thrown: %q", tb.thrown)
			}
		})
	}

	// On the demo set's F-150 no name drawn is cut at 1280 or 768, pressed
	// or not, stamped Recent or not; without a script both brake rotors
	// are drawn, in full. With one, the Recent stamps leading the strip
	// can take the second's place at 768.
	demo := httptest.NewServer(vocabServer(t, nil, demoDir))
	t.Cleanup(demo.Close)
	for _, js := range []bool{false, true} {
		name := map[bool]string{false: "demo, js-off", true: "demo, js-on"}[js]
		t.Run(map[bool]string{false: "demo-js-off", true: "demo-js-on"}[js], func(t *testing.T) {
			tb := newTab(t, path, demo.URL, &requestLog{}, js)
			for _, w := range stripWidths[:2] {
				tb.window(w.width, w.height)
				for _, target := range []string{stripPickup, stripPickup + "&part.id=brake-rotor-front", stripPickup + "&part.id=alternator"} {
					tb.strip(target)
					want := []string{"Brake Rotor, Front", "Brake Rotor, Rear"}
					if js {
						want = nil
					}
					tb.uncut(fmt.Sprintf("%s, %d, %s", name, w.width, target), want...)
				}
			}
		})
	}
}

// strip opens target and waits for its strip, laid by the plate's script
// where scripts run.
func (tb *tab) strip(target string) {
	tb.t.Helper()
	tb.do(chromedp.Navigate(tb.base + target))
	tb.wait(".ptwrap")
	if tb.js {
		tb.wait(".ptwrap.live")
	}
}

// laidStrip is the strip as a visitor has it: the part types drawn, of
// all of them; the rows the part types and the key are drawn in; whether
// the key is drawn, the element at its centre is the key, and its one
// label shown; the part types drawn after one that is not, which strip
// order forbids; and where Tab goes from the last part type drawn.
type laidStrip struct {
	Drawn, Total, Rows int
	Key, OnTop         bool
	Label, Stray, Next string
}

// layStrip reads the strip as it is drawn, then presses Tab from its last
// part type drawn as a keyboard does.
func (tb *tab) layStrip() laidStrip {
	tb.t.Helper()
	got := measure[laidStrip](tb, `(() => {
		const items = [...document.querySelectorAll('.ptypes > li[data-part]')];
		const drawn = items.filter((li) => li.getClientRects().length > 0);
		const key = document.getElementById('more-types');
		const keyDrawn = key.getClientRects().length > 0;
		const tops = drawn.map((li) => li.getBoundingClientRect().top);
		let onTop = false;
		if (keyDrawn) {
			const r = key.getBoundingClientRect();
			tops.push(r.top);
			key.scrollIntoView({block: 'center'});
			const k = key.getBoundingClientRect();
			onTop = key.contains(document.elementFromPoint((k.left + k.right) / 2, (k.top + k.bottom) / 2));
		}
		const labels = [...key.querySelectorAll('.kn')].filter((s) => s.getClientRects().length > 0);
		const stray = items.filter((li, i) => drawn.includes(li) && items.slice(0, i).some((before) => !drawn.includes(before))).map((li) => li.dataset.part).join(' ');
		drawn.at(-1).querySelector('.ptype').focus();
		return {
			Drawn: drawn.length, Total: items.length,
			Rows: new Set(tops.map((t) => Math.round(t))).size,
			Key: keyDrawn, OnTop: onTop,
			Label: labels.length === 1 ? labels[0].textContent : labels.length + ' labels',
			Stray: stray,
		};
	})()`)
	tb.call(func(ctx context.Context) error {
		key := input.DispatchKeyEventParams{Type: input.DispatchKeyEventTypeKeyDown, Key: "Tab", Code: "Tab", WindowsVirtualKeyCode: 9}
		if _, err := chromedp.Call(ctx, input.DispatchKeyEvent, key); err != nil {
			return err
		}
		key.Type = input.DispatchKeyEventTypeKeyUp
		_, err := chromedp.Call(ctx, input.DispatchKeyEvent, key)
		return err
	})
	got.Next = measure[string](tb, `document.activeElement.id || document.activeElement.closest('li')?.dataset.part || document.activeElement.tagName`)
	return got
}

// capped checks the strip shows two rows: the first want part types and
// the key on top, the next stop after them, its one label "N more part
// types", N the rest, which are not drawn.
func (tb *tab) capped(step string, want int) {
	tb.t.Helper()
	got := tb.layStrip()
	noun := "part types"
	if got.Total-want == 1 {
		noun = "part type"
	}
	label := fmt.Sprintf("%d more %s", got.Total-want, noun)
	if got.Drawn != want || got.Rows != 2 || !got.Key || !got.OnTop || got.Label != label || got.Stray != "" || got.Next != "more-types" {
		tb.t.Errorf("%s: %d of %d part types drawn in %d rows, the key drawn %t, on top %t, labelled %q, %q drawn past it, Tab goes on to %q; want %d in 2 rows and the key, on top, labelled %q, the next stop",
			step, got.Drawn, got.Total, got.Rows, got.Key, got.OnTop, got.Label, got.Stray, got.Next, want, label)
	}
}

// fits checks a strip whose part types all fit: every one drawn, and no
// key to see or to reach.
func (tb *tab) fits(step string) {
	tb.t.Helper()
	got := tb.layStrip()
	if got.Drawn != got.Total || got.Key || got.Next == "more-types" {
		tb.t.Errorf("%s: %d of %d part types drawn, the key drawn %t, Tab goes on to %q; want all, and no key", step, got.Drawn, got.Total, got.Key, got.Next)
	}
}

// longName checks the long-named part type's tape: its name cut with an
// ellipsis, its count inside the tape, and the link's name whole.
func (tb *tab) longName(step string) {
	tb.t.Helper()
	type tape struct {
		Cut, Inside bool
		Overflow    string
		Name        string
	}
	got := measure[tape](tb, `(() => {
		const a = document.querySelector('.ptypes > li[data-part="battery-cable"] .ptype');
		const nm = a.querySelector('.nm'), n = a.querySelector('.n');
		const ar = a.getBoundingClientRect(), nr = n.getBoundingClientRect();
		return {
			Cut: nm.scrollWidth > nm.clientWidth, Overflow: getComputedStyle(nm).textOverflow,
			Inside: a.getClientRects().length > 0 && nr.left >= ar.left && nr.right <= ar.right,
			Name: a.textContent,
		};
	})()`)
	if !got.Cut || got.Overflow != "ellipsis" || !got.Inside || !strings.HasPrefix(got.Name, longName+" ") {
		tb.t.Errorf("%s: the long name cut %t with %q, its count inside its tape %t, the link's text %q; want it cut with an ellipsis, the count inside and the name whole", step, got.Cut, got.Overflow, got.Inside, got.Name)
	}
}

// uncut checks no name drawn on the strip is cut, and that each of names
// is drawn.
func (tb *tab) uncut(step string, names ...string) {
	tb.t.Helper()
	type tape struct {
		Name string
		Cut  bool
	}
	got := measure[[]tape](tb, `[...document.querySelectorAll('.ptypes > li[data-part]')].filter((li) => li.getClientRects().length > 0).map((li) => {
		const nm = li.querySelector('.nm');
		return {Name: nm.textContent, Cut: nm.scrollWidth > nm.clientWidth};
	})`)
	var drawn []string
	for _, g := range got {
		drawn = append(drawn, g.Name)
		if g.Cut {
			tb.t.Errorf("%s: %q is cut", step, g.Name)
		}
	}
	for _, n := range names {
		if !slices.Contains(drawn, n) {
			tb.t.Errorf("%s: %q is not drawn, of %q", step, n, drawn)
		}
	}
}

// hangs checks the rails hang from the pressed part type: the riser of
// Vehicle's rail starts 2px under its centre, as ui.css draws it, and runs
// down to its bar, 14px into the tags' top.
func (tb *tab) hangs(step string) {
	tb.t.Helper()
	type riser struct{ Centre, Bottom, Left, Top, End, Tags float64 }
	got := measure[riser](tb, `(() => {
		const fit = document.getElementById('fit').getBoundingClientRect();
		const a = document.querySelector('.ptypes .ptype[aria-current]').getBoundingClientRect();
		const r = getComputedStyle(document.querySelector('.risers'), '::before');
		return {
			Centre: (a.left + a.right) / 2 - fit.left, Bottom: a.bottom - fit.top,
			Left: parseFloat(r.left), Top: parseFloat(r.top), End: fit.height - parseFloat(r.bottom),
			Tags: document.querySelector('.cands').getBoundingClientRect().top - fit.top,
		};
	})()`)
	if math.Abs(got.Left-(got.Centre-7)) > 1 || math.Abs(got.Top-(got.Bottom+2)) > 1 || math.Abs(got.End-(got.Tags+14)) > 1 {
		tb.t.Errorf("%s: the riser runs from (%.1f, %.1f) to %.1f; want it from (%.1f, %.1f), under the pressed part type, to %.1f", step, got.Left, got.Top, got.End, got.Centre-7, got.Bottom+2, got.Tags+14)
	}
}

// drawn are the data-part of each element sel matches that is drawn.
func drawn(tb *tab, sel string) []string {
	tb.t.Helper()
	return eval[[]string](tb, `[...document.querySelectorAll(`+jsString(sel)+`)].filter((li) => li.getClientRects().length > 0).map((li) => li.dataset.part)`)
}

// typeIn puts text in the field sel matches, as a paste or an autofill
// does, which tells the page of an input.
func typeIn(tb *tab, sel, text string) {
	tb.t.Helper()
	eval[bool](tb, `(() => { const f = document.querySelector(`+jsString(sel)+`); f.value = `+jsString(text)+`; f.dispatchEvent(new Event('input', {bubbles: true})); return true; })()`)
}
