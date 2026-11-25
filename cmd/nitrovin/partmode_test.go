package main

import (
	"math"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/chromedp/chromedp"
)

// TestPartMode runs the plate's part-number mode (#325) in a browser, on a
// server that loaded the demo set: the switch turns the plate from a VIN
// to a part type and a number, and back; choosing a part type in the
// dialog fills the plate's field; a number, typed with JavaScript and sent
// by the plate's Look up without, answers with the part's tag and its
// vehicles, a row each; its stock strip opens its rows; the URL carries
// the mode and the part, so a reload draws the same page and Back the
// ones before. One line runs from the number's bracket into the tag's
// eyelet, and a trunk from the tag's bottom edge down through the
// vehicles. At 390 wide the switch and the VINs key are the plate's
// width, the fields stack, the vehicles follow the tag's rows, and the
// page does not scroll sideways. axe audits each state. CI (CI set) needs
// the browser; a machine without one skips.
func TestPartMode(t *testing.T) {
	path := chromePath()
	if path == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("no Chrome or Chromium to run the page in: set NITROVIN_CHROME")
		}
		t.Skip("no Chrome or Chromium: set NITROVIN_CHROME to run the page in a browser")
	}
	log := &requestLog{h: vocabServer(t, nil, demoDir)}
	srv := httptest.NewServer(log)
	t.Cleanup(srv.Close)
	const (
		empty  = "/?from.fitment.number="
		chosen = "/?from.fitment.type=alternator&from.fitment.number="
		found  = chosen + "DEMO-4001"
		open   = found + "&fitment.number=DEMO-4001"
	)
	for _, js := range []bool{false, true} {
		name := map[bool]string{false: "js-off", true: "js-on"}[js]
		t.Run(name, func(t *testing.T) {
			tb := newTab(t, path, srv.URL, log, js)
			tb.window(1280, 900)
			live := func() {
				t.Helper()
				if js {
					tb.wait("form.live")
				}
			}
			tb.do(chromedp.Navigate(srv.URL + "/"))
			tb.wait("#vin-form")
			live()
			log.take()

			// The switch: part-number mode, its plate in the VIN's place,
			// nothing of a VIN's answer below it.
			tb.do(chromedp.Click(chromedp.CSS(`.modes a.mode[href="/?from.fitment.number="]`)))
			tb.wait("#part-form")
			live()
			tb.requests(name+", the switch to part numbers", js, empty)
			if got := tb.at(); got != empty {
				t.Errorf("switched, the tab is at %s; want %s", got, empty)
			}
			if tb.count("#vin-form") != 0 || tb.count("#answer .cards") != 0 || tb.count(`.modes a.mode[aria-current="true"][href="/?from.fitment.number="]`) != 1 {
				t.Errorf("switched: the VIN's plate or cards stayed, or Part number is not pressed\n%s", tb.html("#lookup"))
			}
			if js && !eval[bool](tb, `document.activeElement === document.getElementById("from-number")`) {
				t.Errorf("switched, the focus is on %s; want the number", eval[string](tb, `document.activeElement.outerHTML`))
			}
			tb.audit(name + ", part-number mode")

			// Choosing a part type in the dialog fills the plate's field.
			tb.do(chromedp.Click(chromedp.CSS(".ptpick")))
			tb.wait("#part-types:popover-open")
			tb.audit(name + ", the part types dialog")
			tb.do(chromedp.Click(chromedp.CSS(`#part-types .ptgroups li[data-part="alternator"] a.ptype`)))
			tb.wait(`#part-form input[name="from.fitment.type"][value="alternator"]`)
			tb.requests(name+", Alternator chosen", js, chosen)
			if got := strings.TrimSpace(eval[string](tb, `document.getElementById("ptype-v").textContent`)); got != "Alternator" || tb.count("#part-types:popover-open") != 0 {
				t.Errorf("Alternator chosen: the field reads %q, the dialog open %d", got, tb.count("#part-types:popover-open"))
			}

			// A number: typed, it answers as it is typed; without a
			// script, Look up sends it.
			tb.do(chromedp.SendKeys(chromedp.CSS("#from-number"), "DEMO-4001"))
			if !js {
				tb.do(chromedp.Click(chromedp.CSS(`#part-form button[type="submit"]`)))
			}
			tb.wait(`#answer[data-q="from.fitment.type=alternator&from.fitment.number=DEMO-4001"] .ptag`)
			seen := log.take()
			if len(seen) == 0 || seen[len(seen)-1].target != found || slices.ContainsFunc(seen, func(r pageRequest) bool { return r.htmx != js }) {
				t.Errorf("%s, DEMO-4001 typed: requests %+v; want the last %s, by htmx %t", name, seen, found, js)
			}
			if got := tb.at(); got != found {
				t.Errorf("DEMO-4001 typed, the tab is at %s; want %s", got, found)
			}
			if n := tb.count(".vehs > .cand.veh"); n != 2 || !strings.Contains(tb.html(".ptag"), "2 vehicles in demo-catalog") {
				t.Errorf("DEMO-4001: %d vehicles; want 2\n%s", n, tb.html("#part"))
			}
			if js && !eval[bool](tb, `document.activeElement === document.getElementById("from-number")`) {
				t.Errorf("typing, the focus left the number for %s", eval[string](tb, `document.activeElement.outerHTML`))
			}

			// One line, from the number's bracket into the tag's eyelet:
			// down from the bracket's foot, along, and down into the tag
			// at its leading end, where its eyelet is. And one trunk from
			// the tag's bottom edge to the last vehicle's stub.
			for _, width := range []int64{1280, 768} {
				tb.window(width, 900)
				if ends := measure[[]float64](tb, leaderJS); len(ends) != 6 || math.Abs(ends[0]) > 1.5 || math.Abs(ends[1]) > 1.5 || math.Abs(ends[2]) > 1.5 || ends[3] > 0.5 || math.Abs(ends[4]-34) > 1.5 || ends[5] < 12 || ends[5] > 24 {
					t.Errorf("%d wide, the line is %v [off the bracket's middle, under its foot, its bend past its drop, a gap between the two, into the tag from its leading edge, and from its top]; want it from the bracket's foot into the eyelet, 34 px in and above 24 px down", width, ends)
				}
				trunk := measure[[]float64](tb, `(() => {
					const piece = (el, pseudo) => { const b = el.getBoundingClientRect(), s = getComputedStyle(el, pseudo), top = b.top + parseFloat(s.top); return {top, bottom: top + parseFloat(s.height), left: b.left + parseFloat(s.left)}; };
					const rows = [...document.querySelectorAll(".vehs > .cand")], pieces = rows.map(c => piece(c, "::after")), tag = document.querySelector(".ptag").getBoundingClientRect();
					return [pieces[0].top - tag.bottom, Math.max(...pieces.slice(1).map((p, i) => p.top - pieces[i].bottom)), pieces[pieces.length - 1].bottom - piece(rows[rows.length - 1], "::before").bottom, pieces[0].left - tag.left];
				})()`)
				if len(trunk) != 4 || math.Abs(trunk[0]) > 1 || trunk[1] > 0.5 || math.Abs(trunk[2]) > 1 || trunk[3] <= 0 || trunk[3] > 20 {
					t.Errorf("%d wide, the trunk is %v [past the tag's bottom edge, the gap between its two pieces, past the last vehicle's stub, in from the tag's leading edge]; want one line from the tag to the last vehicle, in the gutter", width, trunk)
				}
			}
			tb.window(1280, 900)
			tb.audit(name + ", DEMO-4001")

			// Its stock strip opens its rows, under its tag.
			tb.do(chromedp.Click(chromedp.CSS("#stk-part")))
			tb.wait(`#stk-part[aria-expanded="true"]`)
			tb.wait(`#stock-region[aria-label="Stock of DEMO-4001"]`)
			tb.requests(name+", DEMO-4001's stock", js, open)
			if got := tb.at(); got != open {
				t.Errorf("stock open, the tab is at %s; want %s", got, open)
			}
			if rows := tb.count("#stock-region tbody"); rows != 3 || tb.count(".phead > .ptag + #stock-rows") != 1 {
				t.Errorf("DEMO-4001's stock: %d rows, under its tag %d; want 3, 1", rows, tb.count(".phead > .ptag + #stock-rows"))
			}
			// The print ends where the vehicles' rows do, as a VIN's print
			// ends where its candidates' rows do: one right edge in both
			// modes (#400).
			if edge := measure[float64](tb, `document.querySelector("#stock-rows .receipt").getBoundingClientRect().right - document.querySelector(".vehs .line").getBoundingClientRect().right`); math.Abs(edge) > 1 {
				t.Errorf("1280 wide: DEMO-4001's print ends %v px past the vehicles' rows; want their right edge", edge)
			}
			tb.audit(name + ", DEMO-4001's stock")

			// The URL is the page: a reload draws it again, and Back steps
			// through the mode's pages and back to a VIN.
			tb.do(chromedp.Reload())
			tb.wait(`#stock-region[aria-label="Stock of DEMO-4001"]`)
			live()
			log.take()
			tb.history(-1)
			tb.wait(`#answer[data-q="from.fitment.type=alternator&from.fitment.number=DEMO-4001"] .ptag`)
			if tb.count("#stock-rows") != 0 {
				t.Error("back from the stock, its rows are still open")
			}
			back := []string{empty, "/"}
			if !js {
				// Without a script the type was chosen in a page of its own.
				back = []string{chosen, empty, "/"}
			}
			for _, want := range back {
				tb.history(-1)
				if want == "/" {
					tb.wait("#vin-form")
				} else {
					tb.wait(`#answer[data-q="` + strings.TrimPrefix(want, "/?") + `"]`)
				}
				if got := tb.at(); got != want {
					t.Errorf("back, the tab is at %s; want %s", got, want)
				}
			}
			log.take()

			// Each mode key is the page in its mode with nothing on the
			// plate, the pressed one too, with a script as without.
			for _, c := range []struct{ from, key, want, wait string }{
				{found, `.modes a.mode[aria-current]`, empty, `#answer[data-q="from.fitment.number="]`},
				{found, `.modes a.mode[href="/"]`, "/", `#answer[data-vin=""]`},
				{"/?from.vin=1FTEW1EG7FFA12345", `.modes a.mode[aria-current]`, "/", `#answer[data-vin=""]`},
			} {
				tb.do(chromedp.Navigate(srv.URL + c.from))
				tb.wait("#answer")
				live()
				log.take()
				tb.do(chromedp.Click(chromedp.CSS(c.key)))
				tb.wait(c.wait)
				live()
				tb.requests(name+", "+c.key+" from "+c.from, js, c.want)
				if got := tb.at(); got != c.want {
					t.Errorf("%s from %s: the tab is at %s; want %s", c.key, c.from, got, c.want)
				}
				if v := measure[string](tb, `(document.getElementById("from-number") || document.getElementById("from-vin")).value`); v != "" || tb.count(".ptag") != 0 {
					t.Errorf("%s from %s: the plate holds %q, %d tags; want it empty", c.key, c.from, v, tb.count(".ptag"))
				}
			}
			log.take()

			// A retired number answers with its replacement, stamped.
			tb.do(chromedp.Navigate(srv.URL + chosen + "DEMO-0901"))
			tb.wait(".ptag .mk.replaces")
			if got := eval[string](tb, `document.querySelector(".ptag .num").textContent + " " + document.querySelector(".ptag .mk.replaces").textContent`); got != "DEMO-1001 Replaces DEMO-0901" {
				t.Errorf("DEMO-0901: %q", got)
			}
			tb.audit(name + ", a retired number")
			log.take()

			// 390 wide: the switch's row, with the VINs key, is the plate's width, the fields stack,
			// the tag's rows are under it and the vehicles under those, one
			// to a row at one width, the line still runs into the eyelet,
			// and the page does not scroll sideways.
			tb.window(390, 844)
			tb.do(chromedp.Navigate(srv.URL + open))
			tb.wait("#stock-region")
			live()
			log.take()
			m := measure[[]float64](tb, `(() => {
				const r = sel => document.querySelector(sel).getBoundingClientRect();
				const rows = [...document.querySelectorAll(".vehs > .cand > .line")].map(e => e.getBoundingClientRect()), tag = r(".ptag"), print = r("#stock-rows");
				return [r(".moderow").width - r(".pplate").width, r("#part-type-field").bottom - r(".pnum").top,
					Math.max(...rows.map(b => Math.abs(b.left - rows[0].left) + Math.abs(b.width - rows[0].width))), rows[1].top - rows[0].bottom,
					print.top - tag.bottom, rows[0].top - print.bottom, document.documentElement.scrollWidth - innerWidth];
			})()`)
			if math.Abs(m[0]) > 1 || m[1] > 0 || m[2] > 1 || m[3] < 0 || m[4] < -3 || m[5] < 0 || m[6] > 0 {
				t.Errorf("390 wide: the switch's row %v wider than the plate, the type %v over the number, the vehicles' rows %v off one width, %v between two, the tag's rows %v under it, the vehicles %v under those, the page %v wider than the window",
					m[0], m[1], m[2], m[3], m[4], m[5], m[6])
			}
			if ends := measure[[]float64](tb, leaderJS); len(ends) != 6 || math.Abs(ends[0]) > 1.5 || math.Abs(ends[4]-34) > 1.5 || ends[5] < 12 || ends[5] > 24 {
				t.Errorf("390 wide, the line is %v; want it from the bracket's middle into the eyelet", ends)
			}
			tb.audit(name + ", 390 wide")
		})
	}
}

// leaderJS measures part-number mode's line, the two boxes .pleader
// draws: [its drop's middle past the bracket's, its top past the
// bracket's foot, its bend's trailing end past the drop's, the bend's top
// past the drop's foot, the bend's leading edge's middle past the tag's
// leading edge, and its foot past the tag's top].
const leaderJS = `(() => {
	const lead = document.querySelector(".pleader"), b = lead.getBoundingClientRect(), drop = getComputedStyle(lead, "::before"), bend = getComputedStyle(lead, "::after");
	const brk = document.querySelector(".punder .brk").getBoundingClientRect(), tag = document.querySelector(".ptag").getBoundingClientRect();
	const x = b.left + parseFloat(drop.left) + parseFloat(drop.width) / 2, top = b.top + parseFloat(drop.top), turn = b.top + parseFloat(bend.top);
	return [x - (brk.left + brk.right) / 2, top - brk.bottom, b.left + parseFloat(bend.left) + parseFloat(bend.width) - (x + parseFloat(drop.width) / 2), turn - (top + parseFloat(drop.height)),
		b.left + parseFloat(bend.left) + 2.5 - tag.left, turn + parseFloat(bend.height) - tag.top];
})()`
