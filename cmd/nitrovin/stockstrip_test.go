package main

import (
	"context"
	"math"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/input"
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
// before. The candidates are rows down the page at every width, the open
// one's rows a print under its tag. On a wide page everything under the
// cards stands on the cards' columns, and one trunk runs from the
// vehicle's block to the last candidate, the picker and each candidate
// tied to it. On the fitment fixture, a rotor with more rows than the
// page shows: "Show all" loads every row, in place where scripts run; and
// on a phone the Tab key goes down the mirrors' page as it is drawn, into
// the rows with no number. axe audits each state. CI (CI set) needs the
// browser; a machine without one skips.
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
	pinned := &requestLog{h: vocabServer(t, nil, fixture, fitmentFixture, stockFixture, orderStockFixture)}
	pinnedSrv := httptest.NewServer(pinned)
	t.Cleanup(pinnedSrv.Close)
	const (
		alternators = "/?from.vin=1FTEW1EG7FFA12345&part.id=alternator"
		first       = alternators + "&fitment.number=DEMO-1001"
		second      = alternators + "&fitment.number=DEMO-1002"
		rotor       = "/?from.vin=1D7RV1GT5AS123456&part.id=brake-rotor-front&fitment.number=BR-2001"
		mirrors     = "/?from.vin=1D7RV1GT5AS123456&part.id=mirror"
		engine      = "/?from.vin=1D7RV1GT5AS123456&part.id=engine&fitment.number=EN-57"
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
			if kinds, rows, notes := tb.count(`#stock-rows .stock-kind`), tb.count(`#stock-rows tbody`), tb.count(`#stock-rows .note`); kinds != 2 || rows != 4 || notes != 2 {
				t.Errorf("DEMO-1001's rows: %d kinds of stock, %d rows, %d notes; want 2, 4 and 2", kinds, rows, notes)
			}
			if tb.count(`.cand:has(#stk-1) > .line + #stock-rows`) != 1 {
				t.Error("DEMO-1001's rows are not right after its tag, in its list item")
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

			// 390 wide: the candidates are rows down the page, at one
			// leading edge, the open one's rows under its tag and no wider
			// than it, shelf over yard, and the page does not scroll
			// sideways.
			tb.window(390, 844)
			tb.do(chromedp.Navigate(srv.URL + first))
			tb.wait(`#stock-region`)
			demo.take()
			m := measure[[]float64](tb, `(() => {
				const lines = [...document.querySelectorAll(".cands > .cand > .line")].map(e => e.getBoundingClientRect());
				const rows = document.getElementById("stock-rows"), t = rows.previousElementSibling.getBoundingClientRect(), r = rows.getBoundingClientRect();
				const kinds = [...document.querySelectorAll("#stock-region > .stock-kind")].map(e => e.getBoundingClientRect());
				return [lines.length, new Set(lines.map(l => Math.round(l.left))).size, Math.min(...lines.slice(1).map((l, i) => l.top - lines[i].bottom)),
					t.left, t.right, t.bottom, r.left, r.right, r.top, kinds[0].bottom, kinds[1].top, kinds[0].left, kinds[1].left, document.documentElement.scrollWidth, innerWidth];
			})()`)
			count, edges, gap, tagL, tagR, tagB, rowsL, rowsR, rowsT := m[0], m[1], m[2], m[3], m[4], m[5], m[6], m[7], m[8]
			shelfB, yardT, shelfL, yardL, pageW, winW := m[9], m[10], m[11], m[12], m[13], m[14]
			if count != 3 || edges != 1 || gap < 0 || pageW > winW {
				t.Errorf("390 wide: %v candidates at %v leading edges, the nearest two %v apart, the page %v in %v; want three rows down a page that does not scroll sideways", count, edges, gap, pageW, winW)
			}
			if rowsL < tagL-1 || rowsR > tagR+1 || rowsT < tagB-3 || tagR > winW {
				t.Errorf("390 wide: the tag from %v to %v, down to %v, its rows from %v to %v, from %v; want the rows under their tag and no wider than it", tagL, tagR, tagB, rowsL, rowsR, rowsT)
			}
			if yardT < shelfB || math.Abs(yardL-shelfL) > 1 {
				t.Errorf("390 wide: the shelf at %v down to %v, the yard at %v from %v; want the shelf over the yard", shelfL, shelfB, yardL, yardT)
			}
			// A phone is pressed with a finger: the picker's key and a
			// stock strip are 44 px high or more. The names keep their
			// place over the picker, as at every width. The print keeps
			// its columns and scrolls sideways by itself: scrolled to
			// its end, the row's id is at its leading edge, the location
			// has slid under it, and the page has not moved. The yard's
			// print holds the id and the row its header pins in view, and
			// has no column for the stock number that repeats the id
			// (#397).
			phone := measure[[]float64](tb, `(() => {
				const r = e => e.getBoundingClientRect(), cands = [...document.querySelectorAll(".cands > .cand")], [print, yard] = document.querySelectorAll("#stock-region .pscroll");
				const at = sel => r(print.querySelector(sel)).left - r(print).left, more = print.scrollWidth - print.clientWidth, first = at("tbody th.id"), loc = at("td.loc");
				print.scrollTo({left: print.scrollWidth, behavior: "instant"});
				const scrolled = print.scrollLeft, held = at("tbody th.id"), slid = loc - at("td.loc");
				print.scrollTo({left: 0, behavior: "instant"});
				const y = r(yard), inView = sel => { const b = r(yard.querySelector(sel)); return b.width > 0 && b.left >= y.left - 1 && b.right <= y.right + 1 ? 1 : 0; };
				return [r(document.getElementById("pick-type")).height, Math.min(...[...document.querySelectorAll(".cands a.stk")].map(e => r(e).height)),
					r(document.querySelector(".ptwrap")).top - r(document.querySelector("#aka .ledger")).bottom, more, first, scrolled, held, slid, document.documentElement.scrollWidth - innerWidth, scrollX,
					inView("tbody th.id") + inView("tbody td.lead"), yard.querySelector("tbody td.lead")?.textContent === "D-1" ? 1 : 0, yard.querySelector("thead").textContent.includes("stock_number") ? 1 : 0];
			})()`)
			if len(phone) != 13 || phone[0] < 44 || phone[1] < 44 || phone[2] < -1 {
				t.Errorf("390 wide: the picker's key is %v px high, the lowest strip %v, and the picker %v px under the names; want 44 px or more to press, and the names over the picker", phone[0], phone[1], phone[2])
			} else if phone[3] <= 0 || math.Abs(phone[4]) > 1 || phone[5] <= 0 || math.Abs(phone[6]) > 1 || phone[7] <= 0 || phone[8] > 0 || phone[9] != 0 {
				t.Errorf("390 wide: the print holds %v px more than it shows, its id %v px in; scrolled %v, the id is %v px in and the location slid %v px, the page %v px wider than the window and scrolled %v; want the print to scroll sideways by itself, its id kept in sight", phone[3], phone[4], phone[5], phone[6], phone[7], phone[8], phone[9])
			} else if phone[10] != 2 || phone[11] != 1 || phone[12] != 0 {
				t.Errorf("390 wide: the yard's print has %v of its id and its row in view, the row D-1 %v, a stock_number column %v; want both in view, and no column that repeats the id", phone[10], phone[11], phone[12])
			}
			tb.order("390 wide")
			// On a phone no line runs from the cards to the block (#401):
			// its facts row leads with the details, so the words are past
			// the row's end at rest.
			if drawn := measure[[]string](tb, `[".carry", ".cards"].map(s => getComputedStyle(document.querySelector(s), s === ".cards" ? "::before" : null).display)`); len(drawn) != 2 || drawn[0] != "none" || drawn[1] != "none" {
				t.Errorf("390 wide: the lines from the cards to the block are drawn %v; want none", drawn)
			}
			tb.audit(name + ", 390 wide")
			// A tablet's: the vehicle's line is carried from the cards,
			// which scroll, to the vehicle's block, which does not: under
			// the row it comes down where the row left it, wherever the
			// row is.
			tb.window(768, 1024)
			tb.order("768 wide")
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
					t.Errorf("768 wide: under the row the vehicle's line is at %v, and in the row at %v; want one line, wherever the row is scrolled", carried[i], carried[i+1])
				}
			}
			tb.window(1440, 900)
			tb.order("1440 wide")
			// Wide again, the names are over the picker, and before it
			// in the document.
			tb.window(1280, 900)
			tb.until("1280 wide: the names before the picker in the document", `document.querySelector("#fit > .ptwrap").previousElementSibling.id === "aka"`)

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
			// Under the block, one trunk: it leaves the block's bottom
			// edge in the gutter beside the first card's column and runs
			// straight down, past the names, a piece an item with no gap
			// between two, past the open candidate's rows, to the last
			// candidate's stub. The picker and every candidate are tied to
			// it by a stub, which starts on the trunk and ends at the
			// picker's tape or in the candidate's lead, where its eyelet is.
			trunk := measure[[]float64](tb, `(() => {
				const piece = (el, pseudo) => { const b = el.getBoundingClientRect(), s = getComputedStyle(el, pseudo), top = b.top + el.clientTop + parseFloat(s.top), left = b.left + el.clientLeft + parseFloat(s.left);
					return {left, right: left + parseFloat(s.width), top, bottom: top + parseFloat(s.height)}; };
				const block = document.querySelector(".fitstrip").getBoundingClientRect(), card = document.querySelector(".cards .card").getBoundingClientRect();
				const tape = document.querySelector(".ptwrap"), cands = [...document.querySelectorAll(".cands > .cand")];
				const pieces = [piece(document.getElementById("aka"), "::before"), piece(tape, "::after"), ...cands.map(c => piece(c, "::after"))], stubs = cands.map(c => piece(c, "::before")), key = piece(tape, "::before");
				const tags = cands.map(c => c.querySelector(".line").getBoundingClientRect()), last = pieces[pieces.length - 1];
				return [pieces.length, pieces[0].top - block.bottom, Math.max(...pieces.map(p => Math.abs(p.left - pieces[0].left))), pieces[0].left - block.left, card.left - pieces[0].right,
					Math.max(...pieces.slice(1).map((p, i) => p.top - pieces[i].bottom)), last.bottom - stubs[stubs.length - 1].bottom,
					Math.max(...stubs.map(s => Math.abs(s.left - pieces[0].right))), Math.min(...stubs.map((s, i) => s.right - tags[i].left)), Math.max(...stubs.map((s, i) => s.right - tags[i].left)),
					Math.abs(key.left - pieces[0].right), Math.abs(key.right - tape.getBoundingClientRect().left)];
			})()`)
			if len(trunk) != 12 || trunk[0] != 5 || math.Abs(trunk[1]) > 1 || trunk[2] > 0.5 || trunk[3] <= 0 || trunk[4] <= 0 || trunk[5] > 0.5 || math.Abs(trunk[6]) > 1 {
				t.Errorf("1280 wide: the trunk is %v [pieces, its top past the block's bottom edge, their spread sideways, in from the block's edge, short of the first card's, the widest gap between two, its end past the last stub]; want five pieces, one line in the gutter from the block past the names to the last candidate's stub", trunk)
			} else if trunk[7] > 1 || trunk[8] <= 0 || trunk[9] > 28 || trunk[10] > 1 || trunk[11] > 1.5 {
				t.Errorf("1280 wide: the stubs are %v [off the trunk, the least and the most into their tag, the picker's off the trunk, and short of its tape]; want each from the trunk into its tag's lead, and the picker's to its tape", trunk[7:])
			}
			if eval[int](tb, `document.querySelectorAll("#fit .drops, #fit .trunk, #fit .ptrecents").length`) != 0 {
				t.Error("1280 wide: an element draws a line to the picker, or recent part types are on its tape")
			}
			// Everything under the block stands on the cards' columns: the
			// picker's tape is the first two, a candidate all six, its
			// number's tape ending with the second and its strip the
			// sixth; the candidates are a column, and the open one's rows
			// run from its tag's lead to its trailing edge.
			cols := measure[[]float64](tb, `(() => {
				const r = e => e.getBoundingClientRect(), cards = [...document.querySelectorAll(".cards .card")].map(r), tape = r(document.querySelector(".ptwrap"));
				const lines = [...document.querySelectorAll(".cands > .cand > .line")], spread = (sel, at) => Math.max(...lines.map(l => Math.abs(at(r(sel ? l.querySelector(sel) : l)))));
				const rows = r(document.querySelector("#stock-rows .receipt")), open = r(document.getElementById("stock-rows").previousElementSibling);
				return [tape.left - cards[0].left, tape.right - cards[1].right, spread("", b => b.left - cards[0].left), spread("", b => b.right - cards[5].right),
					spread(".numtape", b => b.right - cards[1].right), spread(".stk", b => b.left - cards[5].left), spread(".stk", b => b.right - cards[5].right),
					new Set(lines.map(l => Math.round(r(l).top))).size - lines.length, rows.left - open.left, rows.right - open.right];
			})()`)
			for i, what := range []string{"the picker's tape from the first card's leading edge", "the picker's tape from the second card's trailing edge", "a candidate from the first card's leading edge",
				"a candidate from the sixth card's trailing edge", "a number's tape from the second card's trailing edge", "a strip from the sixth card's leading edge", "a strip from the sixth card's trailing edge"} {
				if len(cols) != 10 || cols[i] > 1 || cols[i] < -1 {
					t.Errorf("1280 wide: %s is %v px; want it on the column", what, cols)
					break
				}
			}
			if len(cols) == 10 && (cols[7] != 0 || cols[8] < 20 || cols[8] > 36 || math.Abs(cols[9]) > 1) {
				t.Errorf("1280 wide: %v candidates share a row, and the open one's rows start %v px in from its tag's leading edge and end %v px from its trailing one; want a column of rows, and the print from the tag's lead to its end", -cols[7], cols[8], cols[9])
			}
			// The print's columns stand on the cards' columns (#400): each
			// of shelf's and yard's starts on a card's leading edge, the
			// id's on the number's tape's, with its text where the
			// number's is; the first under the ink has its text where the
			// option text's is; and the print ends with the sixth card.
			paper := measure[[]float64](tb, `(() => {
				const r = e => e.getBoundingClientRect(), pad = e => r(e).left + parseFloat(getComputedStyle(e).paddingLeft);
				const cards = [...document.querySelectorAll(".cards .card")].map(r), tag = document.getElementById("stock-rows").previousElementSibling, tape = tag.querySelector(".numtape");
				const edges = [r(tape).left, ...cards.map(c => c.left)], heads = [...document.querySelectorAll("#stock-rows .print thead tr")];
				const cells = heads.flatMap(tr => [...tr.children]);
				return [cells.length, Math.max(...cells.map(c => Math.min(...edges.map(e => Math.abs(r(c).left - e))))),
					Math.max(...heads.map(tr => Math.abs(pad(tr.children[0]) - pad(tape)))), Math.max(...heads.map(tr => Math.abs(pad(tr.children[2]) - pad(tag.querySelector(".ink"))))),
					r(document.querySelector("#stock-rows .receipt")).right - cards[5].right];
			})()`)
			if len(paper) != 5 || paper[0] != 6 || paper[1] > 1 || paper[2] > 1 || paper[3] > 1 || math.Abs(paper[4]) > 1 {
				t.Errorf("1280 wide: the print's columns are %v [cells in its heads, the farthest off a card's edge, the id's text off the number's, the third column's off the option text's, its end past the sixth card's]; want each on a card's edge, its text on the row's", paper)
			}
			if len(lines) != 6 {
				t.Fatalf("1280 wide: %d measures of the block's lines; want 6, two of each of three", len(lines))
			}
			for i := 0; i < len(lines); i += 2 {
				if lines[i] > 0.5 || lines[i+1] < 0 || lines[i+1] > 2 {
					t.Errorf("1280 wide: line %d into the block starts %v px under the stepped lines and ends %v px into the block; want it to start at them and end on the block's top edge", i/2+1, lines[i], lines[i+1])
				}
			}

			// A bad VIN's problems are a dialog: the block's stamp is a
			// key that opens it, with no script too, the focus on its
			// Close. Nothing of them is at the page's foot.
			tb.do(chromedp.Navigate(srv.URL + "/?from.vin=1FTEW1EG0UFA12345&part.id=alternator"))
			tb.wait(`#fit .stamp .vbad`)
			if js {
				tb.wait("#vin-form.live")
			}
			demo.take()
			tb.do(chromedp.Click(chromedp.CSS(`#fit .stamp .vbad`)))
			opened := measure[[]string](tb, `[String(document.getElementById("problems").matches(":popover-open")), document.activeElement.textContent.trim(), String(document.querySelectorAll("#problems .plist li").length), String(!!document.querySelector(".problems, a[href='#problems']"))]`)
			if !slices.Equal(opened, []string{"true", "Close", "3", "false"}) {
				t.Errorf("the year-code VIN's stamp pressed: [open, focus, problems, anything at the foot] %q; want the VIN problems open with the focus on Close, three problems, nothing at the foot", opened)
			}
			tb.audit(name + ", the VIN problems")

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
			if n := rb.count(`#stock-region .stock-kind:first-child tbody`); n != 10 {
				t.Errorf("BR-2001's shelf: %d rows; want 10", n)
			}
			rb.do(chromedp.Click(chromedp.CSS(`.more-rows a`)))
			rb.wait(`#answer[data-rows="all"]`)
			rb.requests(name+", Show all", js, rotor+"&rows=all")
			if n := rb.count(`#stock-region .stock-kind:first-child tbody`); n != 15 || rb.count(`.more-rows`) != 0 {
				t.Errorf("BR-2001's shelf, all of it: %d rows, %d Show all; want 15, none", n, rb.count(`.more-rows`))
			}
			if js && !eval[bool](rb, `window.kept === true && document.activeElement === document.getElementById("stock-region")`) {
				t.Errorf("Show all loaded another document, or left the focus on %s", eval[string](rb, `document.activeElement.outerHTML`))
			}
			rb.audit(name + ", BR-2001, every row")

			// A phone, the mirrors and their shelf rows with no number.
			// The Tab key goes down the page as it is drawn, in the
			// markup's order at every width, scripts or not: from the
			// block's details to the names, the picker, the candidates,
			// then the rows with no number, whose print it stops on.
			rb.window(390, 844)
			rb.do(chromedp.Navigate(ramSrv.URL + mirrors))
			rb.wait(`#stock-nonum`)
			if js {
				rb.wait("#vin-form.live")
			}
			want := "names, picker, candidates, no number"
			if got := strings.Join(slices.Compact(rb.tabStops("#fit .vdet", 60)), ", "); got != want {
				t.Errorf("390 wide, the mirrors: the Tab key goes through %s; want %s", got, want)
			}

			// A print's columns keep one order at every width (#400): the
			// id, the field pinned first, the location, the declared
			// fields in the header's order, the field pinned second. EN-57's
			// shelf pins two. Its head is in that order at 390, 768 and
			// 1440, and what is in view, at rest and scrolled to the end,
			// is in that order too: a held column covers a column, never
			// passes it.
			pb := newTab(t, path, pinnedSrv.URL, pinned, js)
			pb.do(chromedp.Navigate(pinnedSrv.URL + engine))
			pb.wait(`#stock-region`)
			const shelfOrder = "Id,stock_number,Location,condition,warranty,price"
			for _, width := range []int64{390, 768, 1440} {
				pb.window(width, 900)
				got := measure[[]string](pb, `(() => {
					const sc = document.querySelector("#stock-region .pscroll"), cells = [...sc.querySelectorAll("thead tr > *")];
					const name = c => c.textContent.trim(), dom = cells.map(name);
					sc.scrollIntoView({block: "center", behavior: "instant"});
					const seen = () => { const v = sc.getBoundingClientRect(); return cells.filter(c => { const b = c.getBoundingClientRect(), y = b.top + b.height / 2;
						for (let x = Math.max(b.left, v.left) + 1; x < Math.min(b.right, v.right) - 1; x += 2) { const e = document.elementFromPoint(x, y); if (e && c.contains(e)) return true; }
						return false; }).sort((a, b) => a.getBoundingClientRect().left - b.getBoundingClientRect().left).map(name); };
					sc.scrollLeft = 0; const rest = seen(); sc.scrollLeft = sc.scrollWidth; const end = seen(); sc.scrollLeft = 0;
					return [dom.join(","), rest.join(","), end.join(",")];
				})()`)
				if len(got) != 3 || got[0] != shelfOrder {
					t.Errorf("%d wide: EN-57's shelf heads %q; want %s", width, got, shelfOrder)
					continue
				}
				order := strings.Split(shelfOrder, ",")
				for _, inView := range got[1:] {
					at := 0
					for _, c := range strings.Split(inView, ",") {
						i := slices.Index(order[at:], c)
						if i < 0 {
							t.Errorf("%d wide: EN-57's shelf shows %s; want its columns in the order %s", width, inView, shelfOrder)
							break
						}
						at += i + 1
					}
				}
			}
		})
	}
}

// order checks the vehicle's block, its names, the picker and the
// candidates are drawn down the page in the order the document holds
// them, each under the one before, at the window's present width (#409).
func (tb *tab) order(step string) {
	tb.t.Helper()
	got := measure[[]string](tb, `(() => {
		scrollTo(0, 0);
		const els = ["#fit > .fitstrip", "#fit > #aka", "#fit > .ptwrap", "#fit > .cands"].map(s => document.querySelector(s));
		if (els.some(e => !e)) return ["missing"];
		const bad = [];
		for (let i = 1; i < els.length; i++) {
			const a = els[i - 1], b = els[i];
			if (!(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING)) bad.push("document " + i);
			if (b.getBoundingClientRect().top < a.getBoundingClientRect().bottom - 1) bad.push("drawn " + i);
		}
		return bad;
	})()`)
	if len(got) != 0 {
		tb.t.Errorf("%s: the block, the names, the picker and the candidates are out of order: %v; want them in that order in the document and down the page", step, got)
	}
}

// tabStops presses Tab from the element at sel, up to n times, until the
// focus leaves the parts that fit, and returns where each stop was: the
// vehicle's block, the names, the picker, the candidates or the rows
// with no number.
func (tb *tab) tabStops(sel string, n int) []string {
	tb.t.Helper()
	tb.do(chromedp.Focus(chromedp.CSS(sel)))
	var stops []string
	for range n {
		tb.call(func(ctx context.Context) error {
			key := input.DispatchKeyEventParams{Type: input.DispatchKeyEventTypeKeyDown, Key: "Tab", Code: "Tab", WindowsVirtualKeyCode: 9}
			if _, err := chromedp.Call(ctx, input.DispatchKeyEvent, key); err != nil {
				return err
			}
			key.Type = input.DispatchKeyEventTypeKeyUp
			_, err := chromedp.Call(ctx, input.DispatchKeyEvent, key)
			return err
		})
		at := measure[string](tb, `(() => {
			const on = document.activeElement;
			for (const [name, sel] of [["names", "#aka"], ["no number", ".nonum"], ["candidates", ".cands"], ["block", ".fitstrip"], ["picker", "#fit"]]) if (on.closest(sel)) return name;
			return "";
		})()`)
		if at == "" {
			break
		}
		stops = append(stops, at)
	}
	return stops
}
