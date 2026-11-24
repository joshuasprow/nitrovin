package main

import (
	"net/http/httptest"
	"os"
	"testing"
)

// TestReveal runs a press whose answer opens below the window (#366) in
// a browser, on a server that loaded the demo set, at 1280, 768 and 390
// wide. The control is put at the window's bottom edge and clicked, with
// no focus on it, as a pointer leaves it. The picker pressed opens its
// list downward, and the page scrolls the list into the window with the
// picker still above it; a part type chosen there has its first two
// candidates in the window under the picker; and a stock strip pressed
// scrolls its rows in with its candidate's tag above them. CI (CI set)
// needs the browser; a machine without one skips.
func TestReveal(t *testing.T) {
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

	// press clicks the control, put at the window's bottom edge first
	// when place says so, and answers once what and from have both
	// settled in the window, or after two seconds: [what's bottom past
	// the window's, from's top].
	const press = `(async (control, place, opened, what, from) => {
		const c = document.querySelector(control);
		if (place) window.scrollTo({ top: scrollY + c.getBoundingClientRect().bottom - innerHeight + 4, behavior: "instant" });
		document.activeElement.blur();
		c.click();
		let last = "", at = [0, 0];
		for (let i = 0; i < 40; i++) {
			await new Promise((r) => setTimeout(r, 50));
			const w = document.querySelector(opened) && document.querySelector(what), f = document.querySelector(from);
			if (!w || !f) continue;
			at = [Math.round(w.getBoundingClientRect().bottom - innerHeight), Math.round(f.getBoundingClientRect().top)];
			if (at.join() === last && at[0] <= 0) break;
			last = at.join();
		}
		return at;
	})`
	for _, size := range []struct {
		name          string
		width, height int64
	}{{"1280", 1280, 800}, {"768", 768, 1024}, {"390", 390, 844}} {
		t.Run(size.name, func(t *testing.T) {
			tb := newTab(t, path, srv.URL, demo, true)
			tb.window(size.width, size.height)
			tb.load("/?from.vin=1FTEW1EG7FFA12345")

			at := eval[[]int](tb, press+`('#pick-type', true, '#part-types:popover-open', '#part-types', '#fit .ptwrap')`)
			if len(at) != 2 || at[0] > 0 || at[1] < 0 {
				t.Errorf("the picker pressed at the window's bottom edge: its list ends %d px past the window and the picker starts at %d; want the list in the window under the picker", at[0], at[1])
			}
			if eval[bool](tb, `CSS.supports("anchor-name: --a") && document.getElementById("part-types").getBoundingClientRect().top < document.querySelector("#fit .ptwrap").getBoundingClientRect().bottom`) {
				t.Error("the picker's list opened over the picker; want it under the picker, over what follows")
			}

			at = eval[[]int](tb, press+`('#part-types .ptgroups li[data-part="alternator"] a.ptype', false, '#fit .ptype[aria-current]', '#fit .cand:nth-child(2) > .line', '#fit .ptwrap')`)
			if len(at) != 2 || at[0] > 0 || at[1] < 0 {
				t.Errorf("Alternator chosen in the list: its second candidate ends %d px past the window and the picker starts at %d; want two candidates in the window under the picker", at[0], at[1])
			}

			at = eval[[]int](tb, press+`('#stk-1', true, '#stk-1[aria-expanded="true"]', '#stock-rows', '.cand:has(#stk-1) > .line')`)
			if len(at) != 2 || at[0] > 0 || at[1] < 0 {
				t.Errorf("DEMO-1001's strip pressed at the window's bottom edge: its rows end %d px past the window and its tag starts at %d; want the rows in the window under the tag", at[0], at[1])
			}
		})
	}
}
