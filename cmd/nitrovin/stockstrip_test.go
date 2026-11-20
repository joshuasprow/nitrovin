package main

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chromedp/chromedp"
)

// moreStockFixture is testdata/vocab/stock/more: a shelf with more of
// one rotor than the page shows before "Show all".
var moreStockFixture = abs("../../testdata/vocab/stock/more")

// TestStockStrip runs the candidates' stock strips (#323) in a browser,
// on a server that loaded the demo set: pressing a strip opens its
// candidate's rows, a link without JavaScript and an answer swapped in
// place with it, and pressing another closes the first; the URL carries
// the open candidate, so a reload draws the same page and Back the one
// before. At 390 wide the shelf and the yard are one row that scrolls
// sideways. On the fitment fixture, a rotor with more rows than the page
// shows: "Show all" loads every row, in place where scripts run. axe
// audits each state. CI (CI set) needs the browser; a machine without
// one skips.
func TestStockStrip(t *testing.T) {
	path := chromePath()
	if path == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("no Chrome or Chromium to run the page in: set NITROVIN_CHROME")
		}
		t.Skip("no Chrome or Chromium: set NITROVIN_CHROME to run the page in a browser")
	}
	demo := &requestLog{h: vocabServer(t, nil, demoDir)}
	srv := httptest.NewServer(demo)
	t.Cleanup(srv.Close)
	ram := &requestLog{h: vocabServer(t, nil, fixture, fitmentFixture, stockFixture, moreStockFixture)}
	ramSrv := httptest.NewServer(ram)
	t.Cleanup(ramSrv.Close)
	const (
		alternators = "/?from.vin=1FTEW1EG7FFA12345&part.id=alternator"
		first       = alternators + "&fitment.number=DEMO-1001"
		second      = alternators + "&fitment.number=DEMO-1002"
		rotor       = "/?from.vin=1D7RV1GT5AS123456&part.id=brake-rotor-front&fitment.number=BR-2001"
	)
	for _, js := range []bool{false, true} {
		name := map[bool]string{false: "js-off", true: "js-on"}[js]
		t.Run(name, func(t *testing.T) {
			tb := newTab(t, path, srv.URL, demo, js)
			tb.do(chromedp.Navigate(srv.URL + alternators))
			tb.wait(`#fit a.stk`)
			if js {
				tb.wait("#vin-form.live")
			}
			demo.take()
			if n := tb.count(`#fit a.stk`); n != 3 || tb.count(`#stock-rows`) != 0 {
				t.Errorf("the alternators: %d strips, %d rows open; want 3, none", n, tb.count(`#stock-rows`))
			}

			// Pressing DEMO-1001's strip: its rows, shelf and yard, right
			// after its tag, and the URL its own.
			tb.do(chromedp.Click(chromedp.CSS(`#stk-1`)))
			tb.wait(`#stk-1[aria-expanded="true"]`)
			tb.wait(`#stock-region[aria-label="Stock of DEMO-1001"]`)
			tb.requests(name+", DEMO-1001 pressed", js, first)
			if got := tb.at(); got != first {
				t.Errorf("DEMO-1001 pressed, the tab is at %s; want %s", got, first)
			}
			if cols, rows, notes := tb.count(`#stock-rows .stock-col`), tb.count(`#stock-rows tr`), tb.count(`#stock-rows .note`); cols != 2 || rows != 4 || notes != 2 {
				t.Errorf("DEMO-1001's rows: %d columns, %d rows, %d notes; want 2, 4 and 2", cols, rows, notes)
			}
			if tb.count(`.cand:has(#stk-1) + #stock-rows`) != 1 {
				t.Error("DEMO-1001's rows are not right after its tag")
			}
			if js && eval[bool](tb, `document.activeElement !== document.getElementById("stk-1")`) {
				t.Errorf("the pressed strip lost the focus to %s", eval[string](tb, `document.activeElement.outerHTML`))
			}
			tb.audit(name + ", DEMO-1001 open")

			// Pressing DEMO-1002's: its rows, and DEMO-1001's closed.
			tb.do(chromedp.Click(chromedp.CSS(`#stk-2`)))
			tb.wait(`#stk-2[aria-expanded="true"]`)
			tb.requests(name+", DEMO-1002 pressed", js, second)
			if tb.count(`#stock-rows`) != 1 || tb.count(`#stock-region[aria-label="Stock of DEMO-1002"]`) != 1 || tb.count(`#stk-1[aria-expanded="false"]`) != 1 {
				t.Errorf("DEMO-1002 pressed: %d rows open, DEMO-1001's strip %s", tb.count(`#stock-rows`), tb.html(`#stk-1`))
			}

			// A reload draws the same page; Back, DEMO-1001 open.
			tb.do(chromedp.Reload())
			tb.wait(`#stock-region[aria-label="Stock of DEMO-1002"]`)
			demo.take()
			tb.history(-1)
			tb.wait(`#answer[data-number="DEMO-1001"] #stock-region[aria-label="Stock of DEMO-1001"]`)
			if got := tb.at(); got != first {
				t.Errorf("back, the tab is at %s; want %s", got, first)
			}
			demo.take()

			// 390 wide: the shelf and the yard are one row that scrolls
			// sideways, the yard's column at the edge, and the page does
			// not.
			tb.window(390, 844)
			tb.do(chromedp.Navigate(srv.URL + first))
			tb.wait(`#stock-region`)
			demo.take()
			m := measure[[]float64](tb, `(() => {
				const r = document.getElementById("stock-region"), box = r.getBoundingClientRect();
				const before = r.children[1].getBoundingClientRect().left;
				r.scrollTo({left: r.scrollWidth, behavior: "instant"});
				const after = r.children[1].getBoundingClientRect();
				return [r.scrollWidth, r.clientWidth, before, after.left, after.right, box.left, box.right, document.documentElement.scrollWidth, innerWidth];
			})()`)
			scrollW, clientW, before, left, right, boxL, boxR, pageW, winW := m[0], m[1], m[2], m[3], m[4], m[5], m[6], m[7], m[8]
			if scrollW <= clientW || before < boxR-3*16 || before >= boxR || left < boxL || right > boxR+1 || pageW > winW {
				t.Errorf("390 wide: the columns hold %v in %v, the yard's from %v to %v–%v in %v–%v, the page %v in %v; want the yard at the edge, then scrolled in, and the page not to scroll",
					scrollW, clientW, before, left, right, boxL, boxR, pageW, winW)
			}
			tb.audit(name + ", 390 wide")
			tb.window(1280, 900)

			// Past ten rows, "Show all": every row, in place where scripts
			// run.
			rb := newTab(t, path, ramSrv.URL, ram, js)
			rb.do(chromedp.Navigate(ramSrv.URL + rotor))
			rb.wait(`#stock-region`)
			if js {
				rb.wait("#vin-form.live")
				eval[bool](rb, `window.kept = true`)
			}
			ram.take()
			if n := rb.count(`#stock-region .stock-col:first-child tr`); n != 10 {
				t.Errorf("BR-2001's shelf: %d rows; want 10", n)
			}
			rb.do(chromedp.Click(chromedp.CSS(`.more-rows a`)))
			rb.wait(`#answer[data-rows="all"]`)
			rb.requests(name+", Show all", js, rotor+"&rows=all")
			if n := rb.count(`#stock-region .stock-col:first-child tr`); n != 15 || rb.count(`.more-rows`) != 0 {
				t.Errorf("BR-2001's shelf, all of it: %d rows, %d Show all; want 15, none", n, rb.count(`.more-rows`))
			}
			if js && !eval[bool](rb, `window.kept === true && document.activeElement === document.getElementById("stock-region")`) {
				t.Errorf("Show all loaded another document, or left the focus on %s", eval[string](rb, `document.activeElement.outerHTML`))
			}
			rb.audit(name + ", BR-2001, every row")
		})
	}
}
