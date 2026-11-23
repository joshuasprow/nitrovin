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
// vehicles; its stock strip opens its rows; the URL carries the mode and
// the part, so a reload draws the same page and Back the ones before. One
// line runs from the number's bracket to the tag's eyelet, ending at both.
// At 390 wide the switch and the VINs key are the plate's width, the fields stack, the
// vehicles are one to a row, and the page does not scroll sideways. axe
// audits each state. CI (CI set) needs the browser; a machine without one
// skips.
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
			if n := tb.count(".vehicles tbody tr"); n != 2 || !strings.Contains(tb.html(".ptag"), "2 vehicles in demo-catalog") {
				t.Errorf("DEMO-4001: %d vehicles; want 2\n%s", n, tb.html("#part"))
			}
			if js && !eval[bool](tb, `document.activeElement === document.getElementById("from-number")`) {
				t.Errorf("typing, the focus left the number for %s", eval[string](tb, `document.activeElement.outerHTML`))
			}

			// One line, from the number's bracket to the tag's eyelet: on a
			// wide page to the first third's centre, on a narrow one the
			// first half's.
			for _, w := range []struct {
				width int64
				path  string
			}{{1280, "wide"}, {768, "mid"}} {
				tb.window(w.width, 900)
				ends := measure[[]float64](tb, `(() => {
					const c = el => { const r = el.getBoundingClientRect(); return [(r.left + r.right) / 2, r.top, r.bottom]; };
					const line = document.querySelector(".pleader path.`+w.path+`").getBoundingClientRect();
					const [brk, , brkBottom] = c(document.querySelector(".punder .brk"));
					const [tag, tagTop] = c(document.querySelector(".ptag"));
					return [line.left, line.right, line.top, line.bottom, brk, brkBottom, tag, tagTop];
				})()`)
				near := func(a, b float64) bool { return math.Abs(a-b) < 1.5 }
				if !near(ends[0], ends[6]) || !near(ends[1], ends[4]) || !near(ends[2], ends[5]) || !near(ends[3], ends[7]) {
					t.Errorf("%d wide, the line runs %v–%v, %v–%v; want from the bracket's foot (%v, %v) to the tag's top centre (%v, %v)", w.width, ends[0], ends[1], ends[2], ends[3], ends[4], ends[5], ends[6], ends[7])
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
			if rows := tb.count("#stock-region tr"); rows != 3 || tb.count(".pcol > .ptag + #stock-rows") != 1 {
				t.Errorf("DEMO-4001's stock: %d rows, under its tag %d; want 3, 1", rows, tb.count(".pcol > .ptag + #stock-rows"))
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
			// the vehicles are one to a row under the tag, the line runs
			// straight down to its eyelet, and the page does not scroll
			// sideways.
			tb.window(390, 844)
			tb.do(chromedp.Navigate(srv.URL + open))
			tb.wait("#stock-region")
			live()
			log.take()
			m := measure[[]float64](tb, `(() => {
				const r = sel => document.querySelector(sel).getBoundingClientRect();
				const rows = [...document.querySelectorAll(".vehicles tbody tr")].map(tr => tr.getBoundingClientRect());
				const line = r(".pleader path.narrow"), tag = r(".ptag"), table = r(".vehicles tbody");
				return [r(".moderow").width - r(".pplate").width, r("#part-type-field").bottom - r(".pnum").top,
					Math.max(...rows.map(b => Math.abs(b.left - table.left) + Math.abs(b.width - table.width))), rows[1].top - rows[0].bottom,
					line.left - (tag.left + tag.right) / 2, line.bottom - tag.top, r(".vehicles").top - tag.bottom,
					document.documentElement.scrollWidth - innerWidth];
			})()`)
			if math.Abs(m[0]) > 1 || m[1] > 0 || m[2] > 1 || m[3] < -1 || math.Abs(m[4]) > 1.5 || math.Abs(m[5]) > 1.5 || m[6] < 0 || m[7] > 0 {
				t.Errorf("390 wide: the switch's row %v wider than the plate, the type %v over the number, rows %v off the table's width, %v between rows, the line %v from the eyelet and %v from the tag's top, the vehicles %v under the tag, the page %v wider than the window",
					m[0], m[1], m[2], m[3], m[4], m[5], m[6], m[7])
			}
			tb.audit(name + ", 390 wide")
		})
	}
}
