package main

import (
	"net/http/httptest"
	"os"
	"slices"
	"testing"

	"github.com/chromedp/chromedp"
)

// stripFixture is a vPIC catalog with eighteen part types for a 2015
// F-150, one name too long for a line, and a part vocabulary of their
// groups (#324).
var stripFixture = abs("../../testdata/vocab/strip")

// TestPartTypePicker runs the part type picker (#366) in a browser, on a
// server that loaded the strip fixture, without JavaScript and with it,
// at 1280 and 390. The picker asks for a part type and counts the
// vehicle's eighteen. Pressing it opens its list, every part type under
// its headings: where the browser has anchor positioning, hung from the
// picker at its width, inside the window. A part type chosen there is the
// picker's, with its candidates' count and the key that clears it beside
// it. Pressing the pressed one in the list changes nothing; the key
// clears it. Without JavaScript there is no filter and nothing is recent.
// With it, the filter narrows the list and a filter matching nothing says
// so; a part type opened before sits under the picker, never the pressed
// one, and pressing it there presses it; the focus stays on the picker.
// axe audits each state. CI (CI set) needs the browser; a machine without
// one skips.
func TestPartTypePicker(t *testing.T) {
	path := chromePath()
	if path == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("no Chrome or Chromium to run the page in: set NITROVIN_CHROME")
		}
		t.Skip("no Chrome or Chromium: set NITROVIN_CHROME to run the page in a browser")
	}
	srv := httptest.NewServer(vocabServer(t, nil, stripFixture))
	t.Cleanup(srv.Close)
	const (
		pickup = "/?from.vin=1FTEW1EG7FFA12345"
		pump   = pickup + "&part.id=water-pump"
		fog    = pickup + "&part.id=fog-lamp"
	)
	// picker is what the picker reads: its name, its count and what it
	// counts.
	const picker = `[document.querySelector("#pick-type .nm").textContent, document.querySelector("#pick-type .n").textContent]`
	// still answers once the page has stopped scrolling: a press scrolls
	// what it opened into the window (plate.js's reveal), smoothly, and the
	// picker's list, hung from the picker, moves with it under a click.
	const still = `(async () => { for (let y = -1, n = 0; n < 3; n = scrollY === y ? n + 1 : 0, y = scrollY) await new Promise((r) => setTimeout(r, 60)); return true; })()`
	for _, size := range []struct {
		name          string
		width, height int64
	}{{"1280", 1280, 900}, {"390", 390, 844}} {
		for _, js := range []bool{false, true} {
			name := size.name + map[bool]string{false: ", js-off", true: ", js-on"}[js]
			t.Run(name, func(t *testing.T) {
				tb := newTab(t, path, srv.URL, &requestLog{}, js)
				settle := func() {
					if js {
						eval[bool](tb, still)
					}
				}
				tb.window(size.width, size.height)
				tb.do(chromedp.Navigate(srv.URL + pickup))
				tb.wait(`#pick-type`)
				if js {
					tb.wait("#vin-form.live")
					tb.wait(".ptwrap.live")
				}
				if got, want := eval[[]string](tb, picker), []string{"Choose a part type", "18 part types"}; !slices.Equal(got, want) {
					t.Errorf("nothing pressed, the picker reads %q; want %q", got, want)
				}
				if tb.count("#part-types:popover-open") != 0 || tb.count("#clear-type") != 0 || tb.count(".ptrecents:not([hidden])") != 0 {
					t.Error("nothing pressed: the list is open, a key clears, or a part type is recent")
				}
				tb.audit(name + ", nothing pressed")

				// The list, opened: every part type, and where the browser
				// can hang it, under the picker at its width, in the window.
				settle()
				tb.do(chromedp.Click(chromedp.CSS(`#pick-type`)))
				tb.wait(`#part-types:popover-open`)
				if n := tb.count(`#part-types .ptgroups li[data-part]`); n != 19 {
					t.Errorf("the list holds %d part types; want 19, Headlamp Assembly under two headings", n)
				}
				if n := tb.count(`#part-types .ptf:not([hidden])`); (n == 1) != js {
					t.Errorf("the list's filter is shown %d times with scripts %v", n, js)
				}
				if eval[bool](tb, `CSS.supports("anchor-name: --a")`) {
					off := eval[[]float64](tb, `(() => { const p = document.querySelector(".ptsel").getBoundingClientRect(), d = document.getElementById("part-types").getBoundingClientRect();
						return [Math.abs(d.left - p.left), Math.abs(d.width - p.width), Math.min(d.top, innerHeight - d.bottom, d.left, innerWidth - d.right),
							Math.min(Math.abs(d.top - p.bottom), Math.abs(p.top - d.bottom))]; })()`)
					if off[0] > 1 || off[1] > 1 || off[2] < 0 || off[3] > 8 {
						t.Errorf("the list is %v px off the picker's edge, %v px off its width, %v px inside the window and %v px from the picker; want it hung from the picker, in the window", off[0], off[1], off[2], off[3])
					}
				}
				tb.audit(name + ", the list open")

				if js {
					// The filter narrows the list, headings with it, and a
					// filter matching nothing says so.
					tb.do(chromedp.SendKeys("#ptf-dialog", "brake"))
					if n, h := tb.count(`#part-types .ptgroups li[data-part]:not([hidden])`), tb.count(`#part-types .ptgroups .ptgroup:not([hidden])`); n != 5 || h != 1 {
						t.Errorf(`"brake": %d part types under %d headings; want 5 under 1`, n, h)
					}
					tb.do(chromedp.SendKeys("#ptf-dialog", "zz"))
					if got := eval[string](tb, `document.querySelector("#part-types .ptnone").textContent`); got != "No part type’s name has “brakezz”." {
						t.Errorf("a filter matching nothing says %q", got)
					}
					eval[bool](tb, `(document.getElementById("ptf-dialog").value = "", document.getElementById("ptf-dialog").dispatchEvent(new Event("input", { bubbles: true })), true)`)
				}

				// Water Pump, chosen in the list: the picker's, with its
				// candidate's count and the key that clears it.
				tb.do(chromedp.Click(chromedp.CSS(`#part-types .ptgroups li[data-part="water-pump"] a.ptype`)))
				tb.wait(`#answer[data-part="water-pump"] #clear-type`)
				if got := tb.at(); got != pump {
					t.Errorf("Water Pump chosen, the tab is at %s; want %s", got, pump)
				}
				if got, want := eval[[]string](tb, picker), []string{"Water Pump", "1 candidate"}; !slices.Equal(got, want) {
					t.Errorf("Water Pump chosen, the picker reads %q; want %q", got, want)
				}
				if tb.count("#part-types:popover-open") != 0 || tb.count(`#fit .cand`) != 1 {
					t.Errorf("Water Pump chosen: the list is still open, or %d candidates; want 1", tb.count(`#fit .cand`))
				}
				if js && eval[bool](tb, `document.activeElement !== document.getElementById("pick-type")`) {
					t.Errorf("the picker lost the focus to %s", eval[string](tb, `document.activeElement.outerHTML`))
				}
				if n := tb.count(".ptrecents:not([hidden]) li"); n != 0 {
					t.Errorf("Water Pump pressed: %d part types under the picker; want none, the pressed one is not listed", n)
				}
				tb.audit(name + ", Water Pump pressed")

				// Pressed again in the list, it stays pressed.
				settle()
				tb.do(chromedp.Click(chromedp.CSS(`#pick-type`)))
				tb.wait(`#part-types:popover-open`)
				tb.do(chromedp.Click(chromedp.CSS(`#part-types .ptgroups a.ptype[aria-current="true"]`)))
				tb.wait(`#part-types:not(:popover-open)`)
				tb.wait(`#answer[data-part="water-pump"] #clear-type`)
				if got := tb.at(); got != pump {
					t.Errorf("Water Pump pressed again, the tab is at %s; want %s", got, pump)
				}

				// Fog Lamp chosen: with scripts, Water Pump sits under the
				// picker, and pressing it there presses it.
				settle()
				tb.do(chromedp.Click(chromedp.CSS(`#pick-type`)))
				tb.wait(`#part-types:popover-open`)
				tb.do(chromedp.Click(chromedp.CSS(`#part-types .ptgroups li[data-part="fog-lamp"] a.ptype`)))
				tb.wait(`#answer[data-part="fog-lamp"] #clear-type`)
				if got := tb.at(); got != fog {
					t.Errorf("Fog Lamp chosen, the tab is at %s; want %s", got, fog)
				}
				if js {
					if got := eval[[]string](tb, `[...document.querySelectorAll(".ptrecents:not([hidden]) li")].map((li) => li.dataset.part)`); !slices.Equal(got, []string{"water-pump"}) {
						t.Errorf("Fog Lamp pressed, under the picker: %q; want Water Pump alone", got)
					}
					tb.audit(name + ", a recent part type under the picker")
					settle()
					tb.do(chromedp.Click(chromedp.CSS(`.ptrecents li[data-part="water-pump"] a.ptype`)))
					tb.wait(`#answer[data-part="water-pump"] .ptrecents li[data-part="fog-lamp"]`)
					if got := tb.at(); got != pump {
						t.Errorf("Water Pump pressed under the picker, the tab is at %s; want %s", got, pump)
					}
				} else if n := tb.count(".ptrecents:not([hidden])"); n != 0 {
					t.Error("without scripts a part type is recent")
				}

				// The key beside the picker clears it.
				settle()
				tb.do(chromedp.Click(chromedp.CSS(`#clear-type`)))
				tb.wait(`#answer:not([data-part]) #pick-type`)
				if got := tb.at(); got != pickup {
					t.Errorf("cleared, the tab is at %s; want %s", got, pickup)
				}
				if got := eval[[]string](tb, picker); got[0] != "Choose a part type" || tb.count("#clear-type") != 0 || tb.count(`#fit .cand`) != 0 {
					t.Errorf("cleared, the picker reads %q, with %d keys that clear and %d candidates", got, tb.count("#clear-type"), tb.count(`#fit .cand`))
				}
				if js && eval[bool](tb, `document.activeElement !== document.getElementById("pick-type")`) {
					t.Errorf("cleared, the picker lost the focus to %s", eval[string](tb, `document.activeElement.outerHTML`))
				}
			})
		}
	}
}
