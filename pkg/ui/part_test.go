package ui

import (
	"bytes"
	"html"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/joshuasprow/nitrovin/pkg/nitrovin"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// partModePath is a file of the part-number mode fixture
// (testdata/vocab/partmode): a catalog over the fitment fixture's part
// types and vehicles, with a part of two vehicles and one listed with
// none.
func partModePath(name string) string {
	return filepath.Join("..", "..", "testdata", "vocab", "partmode", name+".jsonl")
}

// partSection is the part's place of the page at target in part-number
// mode, as htmx asks for it, which must answer 200.
func partSection(t *testing.T, h *Handler, target string) string {
	t.Helper()
	return sectionOf(t, h, target, `<section class="part"`)
}

var vehicleRowRE = regexp.MustCompile(`<tr><th scope="row">([^<]*)</th><td class="yrs">([^<]*)</td><td class="opts">([^<]*)</td>(?:<td class="names">(.*?)</td>)?</tr>`)

// vehicleRows are a part's vehicles as its table lists them, a row each,
// "name | years | options | names", its names each "vocabulary name".
func vehicleRows(s string) []string {
	var out []string
	for _, m := range vehicleRowRE.FindAllStringSubmatch(s, -1) {
		names := html.UnescapeString(strings.TrimSpace(tagRE.ReplaceAllString(strings.ReplaceAll(m[4], "</span></span>", "; "), "")))
		out = append(out, strings.Join([]string{html.UnescapeString(m[1]), html.UnescapeString(m[2]), html.UnescapeString(m[3]), strings.TrimSuffix(names, ";")}, " | "))
	}
	return out
}

// TestPagePartMode: the page's switch turns the plate to a part type and
// a number (#325). The URL's fitment key is the mode, so a link draws
// the plate in it, with the part type chosen and the number struck; and
// below it the part's tag, with its stock strip and rows, and its
// vehicles, named in the vocabularies loaded, a page at a time; a retired
// number's replacement, stamped; and, where there is no part, a sentence
// in its place. Nothing of a VIN's answer is drawn. The part types dialog
// lists every part type of the catalog.
func TestPagePartMode(t *testing.T) {
	h := demoPages(t, true)
	const alternator = "/?from.fitment.type=alternator&from.fitment.number="

	// The page opens on VIN: the switch has VIN pressed, and links to the
	// mode with nothing on the plate.
	_, _, body := response(t, h, "GET", "/", nil)
	for _, want := range []string{
		`<nav class="modes" aria-label="Look up by">` + "\n" + `<a class="mode" href="/" aria-current="true">VIN</a><a class="mode" href="/?from.fitment.number=">Part number</a>`,
		`<form id="vin-form" class="plate-form"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GET /: lacks %s\n%s", want, body)
		}
	}
	if strings.Contains(body, `id="part-form"`) {
		t.Errorf("GET /: the part form in VIN mode\n%s", body)
	}

	// Part-number mode, nothing typed: the plate's two fields, the part
	// type to choose, and a sentence asking for both. No VIN's cards,
	// names or parts that fit.
	status, _, body := response(t, h, "GET", "/?from.fitment.number=", nil)
	for _, want := range []string{
		"<title>" + partTitle + " · nitrovin</title>",
		`<a class="mode" href="/">VIN</a><a class="mode" href="/?from.fitment.number=" aria-current="true">Part number</a>`,
		`<form id="part-form" class="plate-form part-form" method="get" action="/" aria-label="Look up a part number">`,
		`<button type="button" class="ptpick" popovertarget="part-types" aria-labelledby="ptype-l ptype-v"><span class="nm" id="ptype-v">Choose a part type</span>`,
		`<input type="text" id="from-number" name="from.fitment.number" value=""`,
		`<button type="submit" class="decode">Look up</button>`,
		`<p class="blank"><span>Choose a part type and type its number.</span></p>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("part-number mode lacks %s\n%s", want, body)
		}
	}
	for _, not := range []string{`id="vin-form"`, `class="cards"`, `id="aka"`, `id="fit"`, `name="from.fitment.type"`, `class="pleader"`} {
		if status != 200 || strings.Contains(body, not) {
			t.Errorf("part-number mode, nothing typed: %d, has %s\n%s", status, not, body)
		}
	}

	// The dialog lists every part type of the catalog, by name, each a
	// link to the mode with it chosen and the number kept; the one
	// chosen is pressed.
	s := partSection(t, h, alternator+"DEMO-4001")
	var listed []string
	for _, m := range regexp.MustCompile(`<a class="ptype swap" href="([^"]*)"( aria-current="true")?><span class="nm">([^<]*)</span></a>`).FindAllStringSubmatch(s, -1) {
		name := html.UnescapeString(m[3]) + " " + html.UnescapeString(m[1])
		if m[2] != "" {
			name = "*" + name
		}
		if !slices.Contains(listed, name) {
			listed = append(listed, name)
		}
	}
	slices.Sort(listed)
	want := []string{
		"*Alternator /?from.fitment.type=alternator&from.fitment.number=DEMO-4001",
		"Brake Rotor, Front /?from.fitment.type=brake-rotor-front&from.fitment.number=DEMO-4001",
		"Brake Rotor, Rear /?from.fitment.type=brake-rotor-rear&from.fitment.number=DEMO-4001",
		"Door Mirror /?from.fitment.type=mirror&from.fitment.number=DEMO-4001",
		"Fog Lamp /?from.fitment.type=fog-lamp&from.fitment.number=DEMO-4001",
		"Headlamp Assembly /?from.fitment.type=headlamp&from.fitment.number=DEMO-4001",
		"Starter Motor /?from.fitment.type=starter&from.fitment.number=DEMO-4001",
		"Transfer Case /?from.fitment.type=transfer-case&from.fitment.number=DEMO-4001",
	}
	if !slices.Equal(listed, want) {
		t.Errorf("the dialog lists %q; want every part type of demo-catalog %q", listed, want)
	}
	if !strings.Contains(s, `<dialog id="part-types" class="ptdialog" popover aria-labelledby="part-types-h" data-spine="demo-part-types">`) ||
		!strings.Contains(s, `<p class="hint">Every part type in demo-catalog · grouped by demo-part-codes</p>`) {
		t.Errorf("the part types dialog in part-number mode:\n%s", s)
	}

	// A part: the plate holds its type and number; its tag, its number
	// and type on its tape and how many vehicles it fits, ends in its
	// stock strip; its vehicles are a table, each named as the catalog
	// and the vocabularies loaded name it, its options verbatim; and one
	// line runs from the number's bracket to its eyelet.
	status, _, body = response(t, h, "GET", alternator+"DEMO-4001", nil)
	s = partSection(t, h, alternator+"DEMO-4001")
	for _, want := range []string{
		"<title>DEMO-4001 · Alternator · nitrovin</title>",
		`<span class="nm" id="ptype-v">Alternator</span>`,
		`<input type="hidden" name="from.fitment.type" value="alternator">`,
		`<input type="text" id="from-number" name="from.fitment.number" value="DEMO-4001"`,
		`<div class="punder" aria-hidden="true"><span class="brk g-serial done"></span></div>`,
		`<path class="g-serial wide" d="M680 0V18H170V36" vector-effect="non-scaling-stroke"/>`,
		`id="copy-url" data-copy="this page's URL">http://example.com/?from.fitment.type=alternator&amp;from.fitment.number=DEMO-4001</pre>`,
		`<pre tabindex="0" id="part-json" data-copy="the part&#39;s JSON">{`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("DEMO-4001's page lacks %s\n%s", want, body)
		}
	}
	for _, want := range []string{
		`<p class="numtape"><b class="num">DEMO-4001</b> <span class="ptn">Alternator</span></p>`,
		`<div><dt>Fits</dt><dd>2 vehicles in demo-catalog</dd></div>`,
		`<a class="stk swap" id="stk-part" href="/?from.fitment.type=alternator&amp;from.fitment.number=DEMO-4001&amp;fitment.number=DEMO-4001" aria-expanded="false"><span>Stock · 1 shelf · 2 yard`,
		`<thead><tr><th scope="col">Vehicle <span class="hint">as demo-catalog names it</span></th><th scope="col">Years</th><th scope="col">Options</th><th scope="col">Also known as</th></tr></thead>`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("DEMO-4001 lacks %s\n%s", want, s)
		}
	}
	if got, want := vehicleRows(s), []string{
		"Honda Civic | 2016–2021 | 1.5L turbo | demo-vehicles Honda Civic",
		"Honda CR-V | 2017–2022 | 1.5L turbo | demo-vehicles Honda CR-V",
	}; status != 200 || !slices.Equal(got, want) {
		t.Errorf("DEMO-4001's vehicles: %d, %q; want %q", status, got, want)
	}
	if strings.Contains(s, `<a href=`) || strings.Contains(s, "id=\"stock-rows\"") || strings.Contains(s, "more-cands") {
		t.Errorf("DEMO-4001: a vehicle is a link, its rows are open, or it has a next page\n%s", s)
	}

	// Its strip opens its rows, shelf and yard, under the tag; pressed
	// again, they close.
	s = partSection(t, h, alternator+"DEMO-4001&fitment.number=DEMO-4001")
	if !strings.Contains(s, `<a class="stk swap" id="stk-part" href="/?from.fitment.type=alternator&amp;from.fitment.number=DEMO-4001" aria-expanded="true" aria-controls="stock-rows">`) ||
		!strings.Contains(s, `<div class="prows" id="stock-rows"><span class="notch" aria-hidden="true"></span>`) ||
		!strings.Contains(s, `aria-label="Stock of DEMO-4001"`) ||
		strings.Count(s, `<th scope="row" class="id">`) != 3 || !strings.Contains(s, "DEMO-S-06") || !strings.Contains(s, "DEMO-Y-05") || !strings.Contains(s, "DEMO-Y-06") {
		t.Errorf("DEMO-4001's stock open: want its shelf row and its two yard vehicles under its tag\n%s", s)
	}

	// A retired number: its replacement's tag, stamped with the number
	// asked for, and the catalog's note on the retirement.
	s = partSection(t, h, alternator+"DEMO-0901")
	for _, want := range []string{
		`<p class="numtape"><b class="num">DEMO-1001</b> <span class="ptn">Alternator</span></p>`,
		`<p class="marks"><span class="mk replaces">Replaces DEMO-0901</span></p>`,
		`<div><dt>Note</dt><dd>demo: 200 A</dd></div>`,
		`<div><dt>Retirement</dt><dd>demo: DEMO-1 → DEMO-2</dd></div>`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("DEMO-0901 lacks %s\n%s", want, s)
		}
	}
	if got := vehicleRows(s); !slices.Equal(got, []string{"Ford F150 | 2015–2017 | 3.5L V6 EcoBoost (VIN G, 8th digit) | demo-vehicles Ford F150"}) {
		t.Errorf("DEMO-0901's vehicles: %q", got)
	}

	// A sided part asked for with no side: a link to each side; one
	// followed is its tag, its side a row.
	s = partSection(t, h, "/?from.fitment.type=headlamp&from.fitment.number=DEMO-1501")
	if !strings.Contains(s, `<p class="blank"><span>demo-catalog has Headlamp Assembly DEMO-1501 on more than one side.</span><a class="swap" href="/?from.fitment.type=headlamp&amp;from.fitment.number=DEMO-1501&amp;from.fitment.side=left">The left one</a><a class="swap" href="/?from.fitment.type=headlamp&amp;from.fitment.number=DEMO-1501&amp;from.fitment.side=right">The right one</a></p>`) {
		t.Errorf("DEMO-1501 with no side:\n%s", s)
	}
	if s := partSection(t, h, "/?from.fitment.type=headlamp&from.fitment.number=DEMO-1501&from.fitment.side=left"); !strings.Contains(s, `<div><dt>Side</dt><dd>left</dd></div>`) {
		t.Errorf("DEMO-1501 on the left:\n%s", s)
	}
	if s := partSection(t, h, "/?from.fitment.type=fog-lamp&from.fitment.number=DEMO-1401"); !strings.Contains(s, `<div><dt>Side</dt><dd>both</dd></div>`) {
		t.Errorf("DEMO-1401, one side, asked with none:\n%s", s)
	}

	// Paged: the rest of the vehicles a link away, each page's own.
	paged := demoPages(t, true)
	paged.vehicles = 1
	s = partSection(t, paged, alternator+"DEMO-4001")
	next := regexp.MustCompile(`<p class="more-cands">This page shows 1 of 2 vehicles. <a class="swap" href="([^"]*)">Next page</a></p>`).FindStringSubmatch(s)
	if got := vehicleRows(s); next == nil || len(got) != 1 || !strings.HasPrefix(got[0], "Honda Civic") {
		t.Fatalf("the first page of DEMO-4001's vehicles: %q\n%s", got, s)
	}
	s = partSection(t, paged, html.UnescapeString(next[1]))
	if got := vehicleRows(s); len(got) != 1 || !strings.HasPrefix(got[0], "Honda CR-V") || strings.Contains(s, "more-cands") || !strings.Contains(s, "Stock · 1 shelf · 2 yard") {
		t.Errorf("the next page of DEMO-4001's vehicles: %q\n%s", got, s)
	}

	// Where there is no part, a sentence in its place, and no line.
	empty := pagesOver(t, fitmentPath("part-types"), fitmentPath("catalog-models"), partModePath("partmode-catalog"))
	for _, c := range []struct {
		h      *Handler
		target string
		want   string
	}{
		{h, alternator + "DEMO-9999", `<p class="blank"><span>demo-catalog has no Alternator numbered DEMO-9999.</span></p>`},
		{h, "/?from.fitment.number=DEMO-4001", `<p class="blank"><span>Choose the part type of DEMO-4001: a number is looked up with its part type.</span></p>`},
		{h, "/?from.fitment.type=alternator&from.fitment.number=", `<p class="blank"><span>Choose a part type and type its number.</span></p>`},
		{h, "/?from.fitment.type=wiper&from.fitment.number=W-1", `<p class="blank"><span>demo-catalog has no wiper numbered W-1.</span></p>`},
		{demoPages(t, false), alternator + "DEMO-4001", `<p class="blank"><span>No catalog loaded.</span><button type="button" popovertarget="vocabularies">Add a catalog</button></p>`},
		{empty, "/?from.fitment.type=engine&from.fitment.number=EN-0000", `<p class="blank"><span>partmode-catalog has Engine EN-0000, and lists no vehicle for it.</span></p>`},
	} {
		s := partSection(t, c.h, c.target)
		if !strings.Contains(s, c.want) || strings.Contains(s, `class="ptag"`) {
			t.Errorf("%s: lacks %s, or has a tag\n%s", c.target, c.want, s)
		}
		if _, _, body := response(t, c.h, "GET", c.target, htmx); strings.Contains(body, `class="pleader"`) {
			t.Errorf("%s: a line runs to no tag\n%s", c.target, body)
		}
	}
	if _, _, body := response(t, demoPages(t, false), "GET", alternator+"DEMO-4001", nil); !strings.Contains(body, `<button type="button" class="ptpick" popovertarget="vocabularies" aria-labelledby="ptype-l ptype-v"><span class="nm" id="ptype-v">No catalog loaded</span>`) || strings.Contains(body, `id="part-types"`) {
		t.Errorf("no catalog: the part type does not open the Vocabularies dialog\n%s", body)
	}
	// The fixture's part of two vehicles: their names are the catalog's
	// own, which no vocabulary on vPIC's spine renames.
	if got := vehicleRows(partSection(t, empty, "/?from.fitment.type=brake-rotor-front&from.fitment.number=BR-3001")); !slices.Equal(got, []string{
		"Dodge Ram 1500 | 2009–2012 |  | catalog-models Dodge Ram 1500",
		"Chevrolet Silverado 1500 | 2014–2018 | w/ heavy duty brakes | catalog-models Chevrolet Silverado 1500",
	}) {
		t.Errorf("BR-3001's vehicles: %q", got)
	}

	// A URL is one mode: a VIN beside a part number is refused, on the
	// VIN's plate, empty, with a link to each. The part type is named
	// as the catalog names it.
	status, _, body = response(t, h, "GET", "/?from.vin=1FTEW1EG7FFA12345&from.fitment.type=alternator&from.fitment.number=DEMO-4001", htmx)
	for _, want := range []string{
		`<p class="error" role="alert">This link names a VIN, 1FTEW1EG7FFA12345, and a part number, Alternator DEMO-4001.</p>`,
		`<p class="ways"><a href="/?from.vin=1FTEW1EG7FFA12345">Open the VIN</a> · <a href="/?from.fitment.type=alternator&amp;from.fitment.number=DEMO-4001">Open the part number</a></p>`,
		`<input type="text" id="from-vin" name="from.vin" value=""`,
		`<a class="mode" href="/" aria-current="true">VIN</a>`,
	} {
		if status != http.StatusBadRequest || !strings.Contains(body, want) {
			t.Errorf("a VIN and a part number: %d, lacks %s\n%s", status, want, body)
		}
	}
	if strings.Contains(body, `id="part-form"`) || strings.Contains(body, "No catalog loaded") {
		t.Errorf("a VIN and a part number: the part plate, or no catalog, with one loaded\n%s", body)
	}

	// A side the fit request does not know is refused in its words.
	status, _, body = response(t, h, "GET", alternator+"DEMO-4001&from.fitment.side=up", htmx)
	if status != http.StatusBadRequest || !strings.Contains(body, `<p class="error" role="alert">`) || strings.Contains(body, `class="ptag"`) {
		t.Errorf("a side that is none: %d\n%s", status, body)
	}

	// A sided part found with no side asked keeps the side it was found
	// on, with stock loaded: on its strip and its Next page, which a
	// later page needs no retry to follow.
	sided := pagesOver(t, fitmentPath("part-types"), fitmentPath("catalog-models"), partModePath("partmode-catalog"), filepath.Join("..", "..", "testdata", "vocab", "stock", "yard-stock.jsonl"))
	sided.vehicles = 1
	s = partSection(t, sided, "/?from.fitment.type=mirror&from.fitment.number=MR-3001")
	next = regexp.MustCompile(`<p class="more-cands">This page shows 1 of 2 vehicles. <a class="swap" href="([^"]*)">Next page</a></p>`).FindStringSubmatch(s)
	if got := vehicleRows(s); next == nil || len(got) != 1 || !strings.Contains(next[1], "from.fitment.side=left") || !strings.Contains(s, `<div><dt>Side</dt><dd>left</dd></div>`) ||
		!regexp.MustCompile(`<a class="stk swap" id="stk-part" href="[^"]*from\.fitment\.side=left`).MatchString(s) {
		t.Fatalf("MR-3001, its side found: %q, next %q\n%s", got, next, s)
	}
	if got := vehicleRows(partSection(t, sided, html.UnescapeString(next[1]))); len(got) != 1 || !strings.HasPrefix(got[0], "Chevrolet Silverado 1500") {
		t.Errorf("MR-3001's next page: %q", got)
	}

	// A vPIC catalog's vehicle is named as VIN mode names one: its vPIC
	// reference and the data month's name for it.
	vpic := pagesOver(t, stripPath("strip-parts"), stripPath("strip-catalog"))
	if s := partSection(t, vpic, "/?from.fitment.type=alternator&from.fitment.number=ST-1001"); !strings.Contains(s, `<tr><th scope="row">vPIC model 1801 (F-150)</th><td class="yrs">2015–2017</td>`) {
		t.Errorf("a vPIC catalog's vehicle:\n%s", s)
	}
}

// partModeTargets are the part-number pages the browser draws as the
// server does.
var partModeTargets = []string{
	"/?from.fitment.number=",
	"/?from.fitment.type=alternator&from.fitment.number=DEMO-4001",
	"/?from.fitment.type=alternator&from.fitment.number=DEMO-4001&fitment.number=DEMO-4001",
	"/?from.fitment.type=alternator&from.fitment.number=DEMO-4001&fitment.number=DEMO-4001&rows=all",
	"/?from.fitment.type=alternator&from.fitment.number=DEMO-0901",
	"/?from.fitment.type=alternator&from.fitment.number=DEMO-9999",
	"/?from.fitment.type=headlamp&from.fitment.number=DEMO-1501",
	"/?from.fitment.type=headlamp&from.fitment.number=DEMO-1501&from.fitment.side=right",
}

// TestLibraryPartMode: the browser's library draws part-number mode as
// the server's pages do with the same files: from the catalog and stock
// it holds, and, while the server holds the catalog, from the server's
// /v1/fit, asked from the fitment key, with the vocabularies to name the
// vehicles in.
func TestLibraryPartMode(t *testing.T) {
	d := embedded(t)
	srv := New(served{demoTranslator(t, d, true)}, log.New(io.Discard, "", 0))

	s := newDecoderServer(t, d)
	l := openWith(t, s, newDisk())
	loadDemo(t, l)
	browser := NewInBrowser(l, BrowserConfig{}, log.New(io.Discard, "", 0))
	for _, target := range partModeTargets {
		if want, got := partSection(t, srv, target), partSection(t, browser, target); got != want {
			t.Errorf("%s in the browser:\n%s\nwant the server's\n%s", target, got, want)
		}
	}

	s = newDecoderServer(t, d)
	for _, f := range theDemo().files {
		s.serve(t, f.Name, f.data)
	}
	s.fits = demoTranslator(t, d, true)
	l = openWith(t, s, newDisk())
	browser = NewInBrowser(l, BrowserConfig{}, log.New(io.Discard, "", 0))
	for _, target := range partModeTargets {
		s.gets = nil
		if want, got := partSection(t, srv, target), partSection(t, browser, target); got != want {
			t.Errorf("%s in the browser, the server's catalog:\n%s\nwant the server's\n%s", target, got, want)
		}
		if strings.Contains(target, "DEMO-4001") && !slices.ContainsFunc(s.gets, func(g string) bool {
			return strings.HasPrefix(g, "/v1/fit?") && strings.Contains(g, "from.fitment.number=DEMO-4001") && strings.Contains(g, "to=demo-vehicles")
		}) {
			t.Errorf("%s: the browser asked %q; want the server's fit of the key", target, s.gets)
		}
	}

	// The server holds the catalog and the browser vehicle vocabularies
	// of its own: the server's /v1/fit is asked to name the vehicles in
	// the server's alone, which it knows, and the part is drawn as the
	// server draws it.
	s = newDecoderServer(t, d)
	for _, f := range theDemo().files {
		s.serve(t, f.Name, f.data)
	}
	s.fits = demoTranslator(t, d, true)
	l = openWith(t, s, newDisk())
	if _, err := l.Load([]Picked{{FileName: "classics.jsonl", Data: fixtureFile(t, "classics")}, {FileName: "fleet.jsonl", Data: fixtureFile(t, "fleet")}}, quiet); err != nil {
		t.Fatal(err)
	}
	browser = NewInBrowser(l, BrowserConfig{}, log.New(io.Discard, "", 0))
	const found = "/?from.fitment.type=alternator&from.fitment.number=DEMO-4001"
	s.gets = nil
	if want, got := partSection(t, srv, found), partSection(t, browser, found); got != want || !strings.Contains(got, `class="ptag"`) {
		t.Errorf("%s in a browser holding fleet, the server's catalog:\n%s\nwant the server's\n%s", found, got, want)
	}
	if slices.ContainsFunc(s.gets, func(g string) bool { return strings.Contains(g, "to=fleet") }) {
		t.Errorf("the browser asked the server to name vehicles in its own vocabulary: %q", s.gets)
	}
	// A side the fit request does not know is refused alike.
	errRE := regexp.MustCompile(`<p class="error" role="alert">[^<]*</p>`)
	_, _, want := response(t, srv, "GET", found+"&from.fitment.side=up", htmx)
	if status, _, body := response(t, browser, "GET", found+"&from.fitment.side=up", htmx); status != http.StatusBadRequest || errRE.FindString(want) == "" || errRE.FindString(body) != errRE.FindString(want) {
		t.Errorf("a side that is none in the browser: %d, %q; want 400, the server's %q", status, errRE.FindString(body), errRE.FindString(want))
	}

	// A vPIC catalog, the browser's own and the server's: its vehicles
	// named as the server names them.
	const st = "/?from.fitment.type=alternator&from.fitment.number=ST-1001"
	vpicWant := partSection(t, pagesOver(t, stripPath("strip-parts"), stripPath("strip-catalog")), st)
	var picked []Picked
	for _, n := range []string{"strip-parts", "strip-catalog"} {
		picked = append(picked, Picked{FileName: n + ".jsonl", Data: mustRead(t, stripPath(n))})
	}
	l = openWith(t, newDecoderServer(t, d), newDisk())
	if _, err := l.Load(picked[:1], quiet); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Compile(bytes.NewReader(picked[1].Data), picked[1].FileName, int64(len(picked[1].Data)), vocab.DirScratch(t.TempDir()), quiet); err != nil {
		t.Fatal(err)
	}
	if got := partSection(t, NewInBrowser(l, BrowserConfig{}, log.New(io.Discard, "", 0)), st); got != vpicWant || !strings.Contains(got, "vPIC model 1801 (F-150)") {
		t.Errorf("%s in the browser's vPIC catalog:\n%s\nwant the server's\n%s", st, got, vpicWant)
	}
	s = newDecoderServer(t, d)
	for _, p := range picked {
		s.serve(t, strings.TrimSuffix(p.FileName, ".jsonl"), p.Data)
	}
	files, err := vocab.Open(stripPath("strip-parts"), stripPath("strip-catalog"))
	if err != nil {
		t.Fatal(err)
	}
	if s.fits, err = nitrovin.NewTranslator(d, files...); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.fits.Close)
	if got := partSection(t, NewInBrowser(openWith(t, s, newDisk()), BrowserConfig{}, log.New(io.Discard, "", 0)), st); got != vpicWant {
		t.Errorf("%s in the browser, the server's vPIC catalog:\n%s\nwant the server's\n%s", st, got, vpicWant)
	}
}
