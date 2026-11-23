package main

import (
	"math"
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
// before. At 390 wide the candidates are one row that scrolls
// sideways, the open one's rows under its tag. On the fitment fixture, a rotor with more rows than the page
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

			// 390 wide: the candidates are one row that scrolls sideways
			// from one edge of the window to the other, the open one's
			// rows under its tag, in its column, shelf over yard, and the
			// page does not scroll.
			tb.window(390, 844)
			tb.do(chromedp.Navigate(srv.URL + first))
			tb.wait(`#stock-region`)
			demo.take()
			m := measure[[]float64](tb, `(() => {
				const c = document.querySelector(".cands"), box = c.getBoundingClientRect();
				const rows = document.getElementById("stock-rows"), tag = rows.previousElementSibling;
				tag.scrollIntoView({block: "nearest", inline: "start", behavior: "instant"});
				const t = tag.getBoundingClientRect(), r = rows.getBoundingClientRect();
				const cols = [...document.getElementById("stock-region").children].map(e => e.getBoundingClientRect());
				return [c.scrollWidth, c.clientWidth, box.left, box.right, t.left, t.right, t.bottom, r.left, r.right, r.top, cols[0].bottom, cols[1].top, cols[0].left, cols[1].left, document.documentElement.scrollWidth, innerWidth];
			})()`)
			scrollW, clientW, boxL, boxR, tagL, tagR, tagB, rowsL, rowsR, rowsT := m[0], m[1], m[2], m[3], m[4], m[5], m[6], m[7], m[8], m[9]
			shelfB, yardT, shelfL, yardL, pageW, winW := m[10], m[11], m[12], m[13], m[14], m[15]
			if scrollW <= clientW || boxL > 0 || boxR < winW || pageW > winW {
				t.Errorf("390 wide: the candidates hold %v in %v, from %v to %v, the page %v in %v; want one row that scrolls, from edge to edge of a page that does not", scrollW, clientW, boxL, boxR, pageW, winW)
			}
			if math.Abs(rowsL-tagL) > 1 || math.Abs(rowsR-tagR) > 1 || rowsT < tagB || tagR >= winW {
				t.Errorf("390 wide: the tag from %v to %v, down to %v, its rows from %v to %v, from %v; want the rows under their tag, in its column, and the next tag at the edge", tagL, tagR, tagB, rowsL, rowsR, rowsT)
			}
			if yardT < shelfB || math.Abs(yardL-shelfL) > 1 {
				t.Errorf("390 wide: the shelf at %v down to %v, the yard at %v from %v; want the shelf over the yard", shelfL, shelfB, yardL, yardT)
			}
			// The vehicle's line is carried from the names, which scroll,
			// to the strip of parts, which does not: under the row it
			// comes down where the row left it, wherever the row is.
			carried := measure[[]float64](tb, `(async () => {
				if (!CSS.supports("animation-timeline: --a") || !CSS.supports("timeline-scope: --a")) return [];
				const hang = document.querySelector(".hang"), out = [];
				for (const to of [0, 120, hang.scrollWidth]) {
					hang.scrollTo({left: to, behavior: "instant"});
					await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
					const carry = document.querySelector(".carry"), piece = getComputedStyle(carry.firstElementChild, "::before");
					const cards = document.querySelector(".cards"), aka = cards.getBoundingClientRect(), line = getComputedStyle(cards, "::before");
					out.push(carry.getBoundingClientRect().left + parseFloat(piece.left), aka.left + parseFloat(line.left));
				}
				return out;
			})()`)
			for i := 0; i+1 < len(carried); i += 2 {
				if math.Abs(carried[i]-carried[i+1]) > 1.5 {
					t.Errorf("390 wide: under the row the vehicle's line is at %v, and in the row at %v; want one line, wherever the row is scrolled", carried[i], carried[i+1])
				}
			}
			tb.audit(name + ", 390 wide")
			tb.window(1280, 900)

			// Wide, each line into the vehicle's block is one piece: it
			// starts where the stepped line under the cards ends, and ends
			// on the block's top edge, not over the stamp inside it.
			lines := measure[[]float64](tb, `(() => {
				const strip = document.querySelector(".fitstrip").getBoundingClientRect(), joins = document.querySelector(".joins").getBoundingClientRect();
				return [...document.querySelectorAll(".fitstrip .seg")].flatMap(seg => {
					const line = getComputedStyle(seg, "::after"), top = seg.getBoundingClientRect().top + parseFloat(line.top);
					return [top - joins.bottom, top + parseFloat(line.height) - strip.top];
				});
			})()`)
			// Under the block, the Vehicle's and the Model year's lines
			// come straight down from under their words into the picker's
			// key, and end there: [each line's x past its word's middle,
			// its bottom past the picker's top, the key's edges past it].
			drops := measure[[]float64](tb, `(() => {
				const fit = document.querySelector("#fit").getBoundingClientRect(), key = document.querySelector("#pick-type").getBoundingClientRect(), out = [];
				for (const [group, pseudo] of [["vds", "::before"], ["year", "::after"]]) {
					const word = document.querySelector(".fitstrip .seg.g-" + group).getBoundingClientRect(), line = getComputedStyle(document.querySelector("#fit .drops"), pseudo);
					const x = fit.left + parseFloat(line.left) + 2.5;
					out.push(x - (word.left + word.right) / 2, fit.bottom - parseFloat(line.bottom) - key.top, x - key.left, key.right - x);
				}
				return out;
			})()`)
			for i := 0; i+3 < len(drops); i += 4 {
				if math.Abs(drops[i]) > 1.5 || drops[i+1] < -4 || drops[i+1] > 12 || drops[i+2] < 0 || drops[i+3] < 0 {
					t.Errorf("1280 wide: the line to the picker is %v px off its word's middle, ends %v px past the picker's top, and is %v and %v px inside the key's edges; want it under its word and into the key", drops[i], drops[i+1], drops[i+2], drops[i+3])
				}
			}
			if len(drops) != 8 || eval[int](tb, `document.querySelectorAll("#fit .rails, #fit .risers, #fit .hook").length`) != 0 {
				t.Errorf("1280 wide: %d measures of the lines to the picker, or a line from the picker to the candidates; want two lines that end at the picker", len(drops))
			}
			// The picker's tape is as wide as the vehicle's block, and the
			// candidates are one row hung from it: each tag's wire runs
			// from the tape's bottom edge to the tag's eyelet.
			hung := measure[[]float64](tb, `(() => {
				const tape = document.querySelector(".ptsel").getBoundingClientRect(), block = document.querySelector(".fitstrip").getBoundingClientRect();
				const tags = [...document.querySelectorAll(".cands > .cand")], tops = new Set(tags.map(t => Math.round(t.getBoundingClientRect().top)));
				const wires = tags.map(t => { const w = getComputedStyle(t, "::after"), top = t.getBoundingClientRect().top + parseFloat(w.top); return [top - tape.bottom, top + parseFloat(w.height) - t.getBoundingClientRect().top]; });
				return [tape.width - block.width, tags.length, tops.size, ...wires.flat()];
			})()`)
			if len(hung) < 5 || math.Abs(hung[0]) > 1 || hung[2] != 1 {
				t.Fatalf("1280 wide: the picker's tape is %v px wider than the block, with %v candidates on %v rows; want the block's width and one row", hung[0], hung[1], hung[2])
			}
			for i := 3; i+1 < len(hung); i += 2 {
				if math.Abs(hung[i]) > 3 || hung[i+1] < 8 || hung[i+1] > 20 {
					t.Errorf("1280 wide: a candidate's wire starts %v px from the tape's bottom edge and ends %v px into its tag; want it from the tape to the eyelet", hung[i], hung[i+1])
				}
			}
			if len(lines) != 6 {
				t.Fatalf("1280 wide: %d measures of the block's lines; want 6, two of each of three", len(lines))
			}
			for i := 0; i < len(lines); i += 2 {
				if lines[i] > 0.5 || lines[i+1] < 0 || lines[i+1] > 2 {
					t.Errorf("1280 wide: line %d into the block starts %v px under the stepped lines and ends %v px into the block; want it to start at them and end on the block's top edge", i/2+1, lines[i], lines[i+1])
				}
			}

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
