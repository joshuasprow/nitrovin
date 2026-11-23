package main

import (
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/chromedp/chromedp"
)

// TestHeaderKeys runs the page in headless Chrome against a server that
// loaded the demo set, at 1280, 768 and 390 px. The page's keys are each
// where they belong and no menu holds them: the header is the brand and
// Vocabularies, with Update now between them while there is an update;
// VINs is at the end of the row over the plate, beside the mode switch;
// the foot is Developers and the release. Nothing is wider than the
// window. The VINs menu hangs under its key and holds the recent
// vehicles, each with the part type that was open on it, then the
// vehicles the yard holds most of, a model once, the most first, and
// Surprise me, which opens one of them. It passes axe open.
func TestHeaderKeys(t *testing.T) {
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
	const f150 = "/?from.vin=1FTEW1EG7FFA12345"
	stocked := []string{"2018 Toyota Camry|5 in the yard", "2017 Toyota RAV4|4 in the yard", "2015 Ram 1500|3 in the yard", "2015 Ford F-150|2 in the yard", "2014 Chevrolet Silverado|1 in the yard"}
	for _, size := range [][2]int64{{1280, 900}, {768, 900}, {390, 800}} {
		t.Run(strconv.FormatInt(size[0], 10), func(t *testing.T) {
			tb := newTab(t, path, srv.URL, log, true)
			tb.window(size[0], size[1])
			tb.do(chromedp.Navigate(srv.URL + f150 + "&part.id=alternator"))
			tb.wait("form.live")
			tb.wait("#fit .cand")

			type placed struct {
				Header, Row, Foot []string
				Menus             int
				Wide              float64
				VinsRight         float64
				SameRow           bool
				UpdateShown       bool
				UpdateInWindow    bool
			}
			p := eval[placed](tb, `(() => {
				const names = (sel) => [...document.querySelectorAll(sel)].filter((e) => getComputedStyle(e).display !== "none").map((e) => e.className || e.id || e.tagName.toLowerCase());
				const r = (sel) => document.querySelector(sel).getBoundingClientRect();
				document.documentElement.dataset.update = "waiting";
				const u = r("#update-now");
				const out = {
					Header: names("header > *"), Row: names(".moderow > *"), Foot: names("footer > *"),
					Menus: document.querySelectorAll(".open-nav, nav.top").length,
					Wide: document.documentElement.scrollWidth - innerWidth,
					VinsRight: r(".plate-form").right - r(".open-vins").right,
					SameRow: Math.abs(r(".open-vins").top - r(".modes").top) < 4 && r(".open-vins").left >= r(".modes").right,
					UpdateShown: getComputedStyle(document.getElementById("update-now")).display !== "none",
					UpdateInWindow: u.left >= 0 && u.right <= innerWidth,
				};
				delete document.documentElement.dataset.update;
				return out;
			})()`)
			if got, want := strings.Join(p.Header, " "), "brand update-now open-vocabularies"; got != want {
				t.Errorf("the header holds %q; want %q", got, want)
			}
			if got, want := strings.Join(p.Row, " "), "modes open-vins"; got != want {
				t.Errorf("the row over the plate holds %q; want %q", got, want)
			}
			if got, want := strings.Join(p.Foot, " "), "open-developers version"; got != want {
				t.Errorf("the foot holds %q; want %q", got, want)
			}
			if p.Menus != 0 {
				t.Errorf("%d of a Menu key and a header nav; want neither", p.Menus)
			}
			if p.Wide > 0.5 || !p.SameRow || p.VinsRight < -0.5 || p.VinsRight > 0.5 || !p.UpdateShown || !p.UpdateInWindow {
				t.Errorf("the page %v wider than the window, VINs beside the switch %t and %v from the plate's end, Update now shown %t and in the window %t", p.Wide, p.SameRow, p.VinsRight, p.UpdateShown, p.UpdateInWindow)
			}

			// The menu: under its key, in the window; the vehicle just
			// looked at with its part type, then the yard's.
			tb.do(chromedp.Click(chromedp.CSS("button.open-vins")))
			tb.wait("#vins:popover-open")
			type menu struct {
				Under, In bool
				Recent    []string
				Stocked   []string
				Examples  bool
			}
			m := eval[menu](tb, `(() => {
				const k = document.querySelector(".open-vins").getBoundingClientRect(), v = document.getElementById("vins").getBoundingClientRect();
				const rows = (sel) => [...document.querySelectorAll(sel)].map((a) => a.querySelector("b").textContent + "|" + (a.querySelector("small") || {textContent: ""}).textContent + "|" + a.getAttribute("href"));
				return {
					Under: v.top >= k.bottom, In: v.left >= 0 && v.right <= innerWidth,
					Recent: rows("#recent a"), Stocked: rows("#stocked a"),
					Examples: !document.getElementById("examples").hidden,
				};
			})()`)
			if !m.Under || !m.In {
				t.Errorf("the VINs menu is under its key %t and in the window %t", m.Under, m.In)
			}
			if want := "2015 FORD F-150|Alternator|/?from.vin=1FTEW1EG7FFA12345&part.id=alternator"; len(m.Recent) != 1 || m.Recent[0] != want || m.Examples {
				t.Errorf("recent %q, examples shown %t; want %q alone", m.Recent, m.Examples, want)
			}
			var got []string
			for _, s := range m.Stocked {
				got = append(got, s[:strings.LastIndex(s, "|")])
				if href := s[strings.LastIndex(s, "|")+1:]; !strings.HasPrefix(href, "/?from.vin=") || strings.Contains(href, "&") {
					t.Errorf("a stocked vehicle links to %s; want a VIN alone", href)
				}
			}
			if strings.Join(got, "; ") != strings.Join(stocked, "; ") {
				t.Errorf("in stock:\n%q\nwant\n%q", got, stocked)
			}
			tb.audit("the VINs menu, a recent vehicle and the yard's")

			// A stocked vehicle opens with its VIN on the plate.
			tb.do(chromedp.Click(chromedp.CSS("#stocked li:first-child a")))
			tb.wait(`#answer[data-label="2018 TOYOTA Camry"]`)
			// The first vehicle again, with no part type: its recent keeps
			// the one it had.
			tb.do(chromedp.Navigate(srv.URL + f150))
			tb.wait(`#answer[data-vin="1FTEW1EG7FFA12345"]`)
			tb.until("the recents", `document.querySelectorAll("#recent a").length === 2`)
			if got := attributes(tb, "#recent a"); got[0]["href"] != f150+"&part.id=alternator" || !strings.HasPrefix(got[1]["href"], "/?from.vin=4T1B11HK") || strings.Contains(got[1]["href"], "&") {
				t.Errorf("recents %v; want the F-150 with its alternator, then the Camry", got)
			}
			// Surprise me is a vehicle of the yard's.
			tb.do(chromedp.Navigate(srv.URL + "/?surprise"))
			tb.wait("#answer[data-vin]")
			vin := eval[string](tb, `document.getElementById("answer").dataset.vin`)
			if !eval[bool](tb, `[...document.querySelectorAll("#stocked a")].some((a) => a.getAttribute("href") === "/?from.vin=`+vin+`")`) {
				t.Errorf("Surprise me opened %s, which is not one of the yard's vehicles", vin)
			}
			tb.mu.Lock()
			defer tb.mu.Unlock()
			if len(tb.thrown) > 0 {
				t.Errorf("thrown: %q", tb.thrown)
			}
		})
	}
}
