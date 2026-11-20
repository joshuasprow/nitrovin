package main

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/chromedp/chromedp"
)

// TestFitCard runs the parts that fit (#322) in a browser, on a server
// that loaded the demo set, as `serve --vocab pkg/ui/demo` does: pressing
// a part type shows its candidates, a link without JavaScript and an
// answer swapped in place with it; the URL carries the part type, so a
// reload draws the same page and Back the page with none pressed. axe
// audits each state. CI (CI set) needs the browser; a machine without
// one skips.
func TestFitCard(t *testing.T) {
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
		pickup  = "/?from.vin=1FTEW1EG7FFA12345"
		pressed = pickup + "&part.id=alternator"
	)
	for _, js := range []bool{false, true} {
		name := map[bool]string{false: "js-off", true: "js-on"}[js]
		t.Run(name, func(t *testing.T) {
			tb := newTab(t, path, srv.URL, log, js)
			tb.do(chromedp.Navigate(srv.URL + pickup))
			tb.wait(`#fit a.ptype`)
			log.take()
			if n := tb.count(`#fit a.ptype`); n != 8 || tb.count(`#fit [aria-current]`) != 0 || tb.count(`#fit .cand`) != 0 {
				t.Errorf("%s: %d part types, %d pressed, %d candidates; want 8, none, none", pickup, n, tb.count(`#fit [aria-current]`), tb.count(`#fit .cand`))
			}
			tb.audit(name + ", the pickup's part types")

			// Pressing Alternator: its three candidates, each catalog text
			// only, and the URL its own.
			tb.do(chromedp.Click(chromedp.CSS(`#fit a.ptype[href$="part.id=alternator"]`)))
			tb.wait(`#fit a.ptype[aria-current="true"]`)
			tb.wait(`#fit .cand`)
			tb.requests(name+", Alternator pressed", js, pressed)
			if got := tb.at(); got != pressed {
				t.Errorf("pressed, the tab is at %s; want %s", got, pressed)
			}
			if n, text := tb.count(`#fit .cand .mk`), tb.count(`#fit .cand .mk.text`); n != 3 || text != 3 || !strings.Contains(tb.html(`#fit .cand .mk`), "Catalog text only") {
				t.Errorf("the alternators' marks: %d, %d catalog text only; want 3 of 3", n, text)
			}
			if js && eval[bool](tb, `document.activeElement !== document.querySelector('#fit a.ptype[aria-current="true"]')`) {
				t.Errorf("the pressed part type lost the focus to %s", eval[string](tb, `document.activeElement.outerHTML`))
			}
			tb.audit(name + ", the alternators")

			// A reload draws the same page.
			tb.do(chromedp.Reload())
			tb.wait(`#fit a.ptype[aria-current="true"]`)
			if n := tb.count(`#fit .cand`); n != 3 {
				t.Errorf("reloaded: %d candidates; want 3", n)
			}
			log.take()

			// Back: the pickup with nothing pressed.
			tb.history(-1)
			tb.wait(`#answer[data-vin="1FTEW1EG7FFA12345"]:not([data-part]) #fit a.ptype`)
			if n := tb.count(`#fit [aria-current]`); n != 0 {
				t.Errorf("back: %d pressed", n)
			}
			if got := tb.at(); got != pickup {
				t.Errorf("back, the tab is at %s; want %s", got, pickup)
			}
			log.take()
		})
	}
}
