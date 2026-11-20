package main

import (
	"fmt"
	"math"
	"net/http/httptest"
	"os"
	"slices"
	"testing"

	"github.com/chromedp/chromedp"
)

// stripFixture is a vPIC catalog with eighteen part types for a 2015
// F-150 and five for a 2014 Silverado, and a part vocabulary of their
// groups (#324).
var stripFixture = abs("../../testdata/vocab/strip")

const (
	stripPickup    = "/?from.vin=1FTEW1EG7FFA12345"
	stripSilverado = "/?from.vin=1GCVKREC6EZ123456"
)

// TestPartTypeStrip runs the strip of part types (#324) in a browser, on a
// server that loaded the strip fixture, without JavaScript and with it.
// At 1280, 768 and 390 the F-150's eighteen part types are two rows and
// "N more", N the count of those the rows have no room for. "N more"
// opens the part types dialog; a part type chosen there leads the strip,
// pressed, and the rails hang from it at every width. Without JavaScript
// there is no filter and nothing is recent. With it, typing narrows the
// strip across the part types it had no room for, the pressed one kept;
// the dialog's own filter narrows the dialog; part types opened lead the
// strip on the next VIN, stamped Recent, and are listed in the dialog;
// the dialog's Clear recent forgets them and keeps the VINs, and the VINs
// menu's forgets both. axe audits each state. CI (CI set) needs the
// browser; a machine without one skips.
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
	widths := []struct{ width, height int64 }{{1280, 900}, {768, 1024}, {390, 844}}
	for _, js := range []bool{false, true} {
		name := map[bool]string{false: "js-off", true: "js-on"}[js]
		t.Run(name, func(t *testing.T) {
			tb := newTab(t, path, srv.URL, &requestLog{}, js)
			for _, w := range widths {
				tb.window(w.width, w.height)
				tb.strip(stripPickup)
				tb.capped(fmt.Sprintf("%s, %d", name, w.width))
			}
			tb.audit(name + ", the capped strip at 390")
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
			for _, w := range widths {
				tb.window(w.width, w.height)
				tb.strip(stripPickup + "&part.id=water-pump")
				tb.capped(fmt.Sprintf("%s, Water Pump pressed, %d", name, w.width))
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
			if !eval[bool](tb, `document.querySelector('.ptypes > li[data-part="wiper-motor"]').classList.contains('over')`) {
				t.Errorf("Wiper Motor is in the two rows; want it past them")
			}
			tb.do(chromedp.SendKeys("#ptf-strip", "wip"))
			if got := shown(tb, ".ptypes > li[data-part]"); !slices.Equal(got, []string{"water-pump", "wiper-motor"}) {
				t.Errorf("filtered by wip, the strip shows %q; want water-pump (pressed) and wiper-motor", got)
			}
			if eval[bool](tb, `!document.getElementById('more-types').hidden`) {
				t.Errorf("filtered to two, the strip still has its key")
			}
			tb.audit("js-on, the strip filtered")
			typeIn(tb, "#ptf-strip", "zz")
			if got := eval[string](tb, `document.querySelector('.ptwrap + .ptnone:not([hidden])')?.textContent ?? ''`); got != "No part type’s name has “zz”." {
				t.Errorf("filtered by zz, the strip says %q", got)
			}
			typeIn(tb, "#ptf-strip", "")
			tb.capped("js-on, the filter emptied")

			// The dialog's filter narrows the dialog, and the headings
			// left with nothing go.
			tb.do(chromedp.Click(chromedp.CSS("#more-types")))
			tb.wait("#part-types:popover-open")
			typeIn(tb, "#ptf-dialog", "lamp")
			if got := shown(tb, ".ptgroups .ptlist > li"); !slices.Equal(got, []string{"headlamp", "fog-lamp", "headlamp", "tail-lamp"}) {
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
			tb.strip(stripSilverado)
			want := []string{"water-pump", "starter", "alternator", "battery", "radiator"}
			if got := eval[[]string](tb, `[...document.querySelectorAll('.ptypes > li[data-part]')].map(li => li.dataset.part)`); !slices.Equal(got, want) {
				t.Errorf("the next VIN's strip %q; want %q", got, want)
			}
			if got := eval[[]string](tb, `[...document.querySelectorAll('.ptypes > li:has(> .rstamp)')].map(li => li.dataset.part)`); !slices.Equal(got, want[:2]) {
				t.Errorf("the next VIN's Recent stamps are on %q; want %q", got, want[:2])
			}
			// The Silverado's five fit in its two rows: it has no key.
			if n := tb.count("#more-types:not([hidden])"); n != 0 {
				t.Errorf("the Silverado's strip has a key with nothing past its rows")
			}
			tb.strip(stripPickup)
			tb.do(chromedp.Click(chromedp.CSS("#more-types")))
			tb.wait("#part-types:popover-open")
			if got := shown(tb, ".ptrecent .ptlist > li"); !slices.Equal(got, []string{"water-pump", "starter", "battery"}) {
				t.Errorf("the next VIN's dialog lists %q as opened recently", got)
			}
			tb.audit("js-on, the next VIN's dialog, opened recently")

			// The dialog's Clear recent forgets the part types and keeps
			// the VINs; the VINs menu's forgets both.
			tb.do(chromedp.Click(chromedp.CSS(".ptrecent .clear-parts")))
			if n := tb.count(".rstamp, .ptrecent:not([hidden])"); n != 0 || eval[bool](tb, `localStorage.getItem('nitrovin.recent-parts') !== null`) {
				t.Errorf("cleared in the dialog, %d Recent stamps or recent lists show", n)
			}
			if eval[bool](tb, `localStorage.getItem('nitrovin.recent') === null`) {
				t.Errorf("cleared in the dialog, the recent VINs are gone too")
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

// capped checks the strip shows two rows of part types and its key says
// how many it has no room for: the count the last of them shows, which
// lies on top of the others'. With the script, the ones it has no room
// for are out of the keyboard's reach, and the key is named with their
// count.
func (tb *tab) capped(step string) {
	tb.t.Helper()
	type strip struct {
		Rows, Shown, Total, Over int
		Count, Key               string
	}
	got := measure[strip](tb, `(() => {
		const wrap = document.querySelector('.ptwrap');
		const items = [...wrap.querySelectorAll('.ptypes > li[data-part]')].filter((li) => !li.hidden);
		const bottom = wrap.getBoundingClientRect().bottom;
		const shown = items.filter((li) => li.getBoundingClientRect().top < bottom);
		const key = document.getElementById('more-types');
		return {
			Rows: new Set(shown.map((li) => Math.round(li.getBoundingClientRect().top))).size,
			Shown: shown.length, Total: items.length,
			Over: wrap.querySelectorAll('.ptypes > li.over').length,
			Count: shown.at(-1).querySelector('.ptmore').textContent,
			Key: key.hidden ? '' : key.getAttribute('aria-label') || '',
		};
	})()`)
	if got.Rows != 2 || got.Shown >= got.Total || got.Count != fmt.Sprintf("%d more", got.Total-got.Shown) {
		tb.t.Errorf("%s: %d rows, %d of %d shown, the key reads %q; want 2 rows and the rest more", step, got.Rows, got.Shown, got.Total, got.Count)
	}
	if want := fmt.Sprintf("%d more part types", got.Total-got.Shown); tb.js && (got.Over != got.Total-got.Shown || got.Key != want) {
		tb.t.Errorf("%s: %d out of reach and the key named %q; want %d and %q", step, got.Over, got.Key, got.Total-got.Shown, want)
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

// shown are the data-part of each element sel matches that is not hidden.
func shown(tb *tab, sel string) []string {
	tb.t.Helper()
	return eval[[]string](tb, `[...document.querySelectorAll(`+jsString(sel)+`)].filter((li) => !li.hidden).map((li) => li.dataset.part)`)
}

// typeIn puts text in the field sel matches, as a paste or an autofill
// does, which tells the page of an input.
func typeIn(tb *tab, sel, text string) {
	tb.t.Helper()
	eval[bool](tb, `(() => { const f = document.querySelector(`+jsString(sel)+`); f.value = `+jsString(text)+`; f.dispatchEvent(new Event('input', {bubbles: true})); return true; })()`)
}
