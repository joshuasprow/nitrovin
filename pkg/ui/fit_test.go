package ui

import (
	"bytes"
	"encoding/json"
	"errors"
	"html"
	"io"
	"log"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/nitrovin"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// served is `nitrovin serve` as its pages see it (cmd/nitrovin's pages):
// the embedded decoder and the files loaded beside it.
type served struct{ *nitrovin.Translator }

func (s served) Decode(req contract.Request) (contract.Decode, error) {
	return s.Decoder().Decode(req)
}
func (served) ErrorTexts() map[int]string   { return nil }
func (served) SpineBundle() ([]byte, error) { return nil, errors.New("no spine bundle in this test") }
func (served) File(string) ([]byte, error)  { return nil, errors.New("no files in this test") }

// demoPages is the server's pages over the demo set, its catalog among it
// when catalog is set.
func demoPages(t *testing.T, catalog bool) *Handler {
	t.Helper()
	return New(served{demoTranslator(t, embedded(t), catalog)}, log.New(io.Discard, "", 0))
}

// demoTranslator is `nitrovin serve --vocab pkg/ui/demo` over d, without
// the catalog and the stock that needs it unless catalog is set.
func demoTranslator(t *testing.T, d *nitrovin.Decoder, catalog bool) *nitrovin.Translator {
	t.Helper()
	files, err := vocab.Open(demoDir)
	if err != nil {
		t.Fatal(err)
	}
	if !catalog {
		files = slices.DeleteFunc(files, func(f *vocab.File) bool { return f.Kind == vocab.KindCatalog || f.Kind == vocab.KindStock })
	}
	tr, err := nitrovin.NewTranslator(d, files...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tr.Close)
	return tr
}

// fitSection is the parts that fit of the page at target, as htmx asks
// for it, which must answer 200.
func fitSection(t *testing.T, h *Handler, target string) string {
	t.Helper()
	return sectionOf(t, h, target, `<section class="fit"`)
}

// sectionOf is the section of the page at target that open starts, as
// htmx asks for it, which must answer 200.
func sectionOf(t *testing.T, h *Handler, target, open string) string {
	t.Helper()
	status, _, body := response(t, h, "GET", target, htmx)
	start := strings.Index(body, open)
	if status != 200 || start < 0 {
		t.Fatalf("GET %s: %d, no %s\n%s", target, status, open, body)
	}
	// The section holds the dialog's sections: it ends at the close that
	// balances its open.
	depth := 0
	for i := range len(body) - start {
		switch s := body[start+i:]; {
		case strings.HasPrefix(s, "<section"):
			depth++
		case strings.HasPrefix(s, "</section>"):
			if depth--; depth == 0 {
				return body[start : start+i+len("</section>")]
			}
		}
	}
	t.Fatalf("GET %s: %s never ends\n%s", target, open, body)
	return ""
}

var (
	partTypeRE = regexp.MustCompile(`<a class="ptype swap" href="([^"]*)"( aria-current="true")?><span class="nm">([^<]*)</span> <span class="n">(\d+)`)
	markRE     = regexp.MustCompile(`<span class="mk (\w+)">(?:<svg[^>]*><path[^>]*/></svg>)?([^<]*)</span>`)
	numberRE   = regexp.MustCompile(`<span class="num">([^<]*)</span>`)
)

// partTypes are a section's part types, each once, by name, whatever
// headings its dialog lists them under: "name count", "*" before the
// pressed one's name.
func partTypes(s string) []string {
	if i := strings.Index(s, `<div class="ptgroups"`); i >= 0 {
		s = s[i : i+strings.Index(s[i:], "</dialog>")]
	}
	var out []string
	for _, m := range partTypeRE.FindAllStringSubmatch(s, -1) {
		name := html.UnescapeString(m[3]) + " " + m[4]
		if m[2] != "" {
			name = "*" + name
		}
		if !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	slices.SortFunc(out, func(a, b string) int {
		return strings.Compare(strings.ToLower(strings.TrimPrefix(a, "*")), strings.ToLower(strings.TrimPrefix(b, "*")))
	})
	return out
}

// tags are a section's candidates: each its number and its marks, "kind:
// text" each.
func tags(s string) []string {
	var out []string
	for _, li := range strings.Split(s, `<li class="cand">`)[1:] {
		tag := numberRE.FindStringSubmatch(li)[1]
		for _, m := range markRE.FindAllStringSubmatch(li, -1) {
			tag += " | " + m[1] + ": " + html.UnescapeString(m[2])
		}
		out = append(out, tag)
	}
	return out
}

// TestPageFit: under the names, the page lists the part types the demo
// catalog has for a VIN's vehicle, by name with counts and none pressed,
// and, for the part type its URL presses, the candidates as the catalog
// has them with what nitrovin checked of each. Where there is nothing to
// list, it says why, with a way on.
func TestPageFit(t *testing.T) {
	h := demoPages(t, true)
	const pickup = "/?from.vin=1FTEW1EG7FFA12345"

	// By name, not the answer's order (by part type id), and nothing
	// pressed: no candidate is asked for.
	s := fitSection(t, h, pickup)
	want := []string{"Alternator 3", "Brake Rotor, Front 1", "Brake Rotor, Rear 1", "Door Mirror 2", "Fog Lamp 1", "Headlamp Assembly 2", "Starter Motor 1", "Transfer Case 2"}
	if got := partTypes(s); !slices.Equal(got, want) {
		t.Errorf("the pickup's part types %q; want %q", got, want)
	}
	for _, want := range []string{
		`<p class="stamp">This vehicle · demo-catalog</p>`,
		// The catalog's make, in its own word, with the Maker's line to it.
		`<span class="seg g-wmi"><span class="w">Ford</span></span>` + "\n" + `<span class="seg g-vds"><span class="w">F150</span></span>`,
		`<span class="seg g-year"><span class="w">2015</span></span>`,
		`<a class="ptype swap" href="/?from.vin=1FTEW1EG7FFA12345&amp;part.id=alternator">`,
		`<a class="ptype swap" href="/?from.vin=1FTEW1EG7FFA12345&amp;part.id=mirror"><span class="nm">Door Mirror</span>`,
		"Choose a part type to see what fits this vehicle.",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("the pickup lacks %s\n%s", want, s)
		}
	}
	if strings.Contains(s, "aria-current") || strings.Contains(s, `class="cands"`) {
		t.Errorf("the pickup has a part type pressed\n%s", s)
	}

	// Each part type pressed: its candidates, in the answer's order, and
	// the marks the demo set's notes promise (#322): three alternators,
	// all catalog text only; one front rotor, the decode's drive type
	// ruling the other out; one rear rotor with no conditions. A year
	// confirmed alone checks nothing worth a stamp. Marks are each
	// application's, weakest wins: the tag shows their union, the weaker
	// first. The demo set's candidates have one application a vehicle;
	// the fitment fixture's left mirror has two for a 2010 Ram 1500 4WD,
	// one with option text alone, one checked by Drive Type.
	fixture := New(served{serverTranslator(t, embedded(t))}, log.New(io.Discard, "", 0))
	for _, c := range []struct {
		target  string
		pressed string
		tags    []string
		on      *Handler // nil: the demo set's pages
	}{
		{pickup + "&part.id=alternator", "*Alternator 3", []string{
			"DEMO-1001 | text: Match the options yourself", "DEMO-1002 | text: Match the options yourself", "DEMO-1003 | text: Match the options yourself"}, nil},
		{pickup + "&part.id=brake-rotor-front", "*Brake Rotor, Front 1", []string{"DEMO-1102 | checked: Checked · Drive Type"}, nil},
		{pickup + "&part.id=brake-rotor-rear", "*Brake Rotor, Rear 1", []string{"DEMO-1111 | none: No conditions"}, nil},
		// The CR-V's decode has no drive type to check an AWD fitment by.
		{"/?from.vin=2HKRW2H53KH612345&part.id=transfer-case", "*Transfer Case 1", []string{"DEMO-4301 | unchecked: Not checked · Drive Type"}, nil},
		// The Silverado's rear rotor is listed with the front, and its
		// series checked on the way to its catalog vehicle.
		{"/?from.vin=1GCVKREC6EZ123456&part.id=brake-rotor-rear", "*Brake Rotor, Rear 1", []string{"DEMO-2101 | checked: Checked · Series | redirect: By redirect · Brake Rotor, Front"}, nil},
		// Eight characters name the model and no year: every year's, the
		// year a doubt before the drive type's check.
		{"/?from.vin=1FTEW1EG&part.id=brake-rotor-front", "*Brake Rotor, Front 1", []string{"DEMO-1102 | unchecked: Not checked · Model year | checked: Checked · Drive Type"}, nil},
		{"/?from.vin=1D7RV1GT5AS123456&part.id=mirror", "*Door Mirror 2", []string{
			"MR-1001 | text: Match the options yourself | checked: Checked · Drive Type", "MR-1001 | text: Match the options yourself"}, fixture},
	} {
		on := h
		if c.on != nil {
			on = c.on
		}
		s := fitSection(t, on, c.target)
		if pressed := slices.DeleteFunc(partTypes(s), func(n string) bool { return !strings.HasPrefix(n, "*") }); !slices.Equal(pressed, []string{c.pressed}) {
			t.Errorf("%s: pressed %q; want %s", c.target, pressed, c.pressed)
		}
		if got := tags(s); !slices.Equal(got, c.tags) {
			t.Errorf("%s: candidates %q; want %q\n%s", c.target, got, c.tags, s)
		}
		if strings.Contains(s, "Press a part type") {
			t.Errorf("%s: invites a press with one pressed", c.target)
		}
	}

	// A candidate's row: a list item whose heading is its number, with its
	// years, on its tape; then the option text verbatim and the note. The
	// pressed part type's link takes it away.
	s = fitSection(t, h, pickup+"&part.id=alternator")
	for _, want := range []string{
		`<a class="ptype swap" href="/?from.vin=1FTEW1EG7FFA12345&amp;part.id=alternator" aria-current="true"><span class="nm">Alternator</span> <span class="n">3`,
		`<span class="nm">Alternator</span><span class="n">3<span class="visually-hidden"> candidates</span></span>`,
		`<a class="ptclear swap" id="clear-type" data-focus="pick-type" href="/?from.vin=1FTEW1EG7FFA12345" aria-label="Clear Alternator">×</a>`,
		`<ol class="cands" aria-label="Candidates: Alternator">`,
		`<h3 class="numtape"><span class="num">DEMO-1001</span> <span class="of"><span class="yrs">2015–2017</span></span></h3>`,
		`<p class="opts">2.7L V6 EcoBoost (VIN P, 8th digit), w/ start-stop</p>`,
		`<span class="nt"><span class="k">Note</span> demo: 240 A</span>`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("the alternators lack %s\n%s", want, s)
		}
	}
	// A side is on the tape with the number it tells apart from its pair.
	if s := fitSection(t, h, pickup+"&part.id=mirror"); !strings.Contains(s, `<span class="num">DEMO-1301</span> <span class="of"><span class="side">left</span> `) || !strings.Contains(s, `<span class="num">DEMO-1301</span> <span class="of"><span class="side">right</span> `) {
		t.Errorf("the mirrors lack their sides\n%s", s)
	}
	// The page's own URL, to copy, is the pressed part type's.
	if _, _, body := response(t, h, "GET", pickup+"&part.id=alternator", nil); !strings.Contains(body, `id="copy-url" data-copy="this page's URL">http://example.com/?from.vin=1FTEW1EG7FFA12345&amp;part.id=alternator</pre>`) ||
		!strings.Contains(body, `<pre tabindex="0" id="candidates-json" data-copy="the candidates&#39; JSON">{`) {
		t.Errorf("the alternators' page lacks its URL or its fit's JSON\n%s", body)
	}
	// A part type the vehicle has none of.
	if s := fitSection(t, h, pickup+"&part.id=wiper"); !strings.Contains(s, "Nothing demo-catalog lists as &#34;wiper&#34; fits this vehicle.") || strings.Contains(s, "aria-current") {
		t.Errorf("an unknown part type:\n%s", s)
	}

	// Paged: the rest of the candidates a link away, each page's own.
	paged := demoPages(t, true)
	paged.candidates = 2
	s = fitSection(t, paged, pickup+"&part.id=alternator")
	next := regexp.MustCompile(`<p class="more-cands">This page shows 2 of 3 candidates. <a class="swap" href="([^"]*)">Next page</a></p>`).FindStringSubmatch(s)
	if got := tags(s); next == nil || len(got) != 2 {
		t.Fatalf("the first page of alternators: %q\n%s", got, s)
	}
	s = fitSection(t, paged, html.UnescapeString(next[1]))
	if got := tags(s); !slices.Equal(got, []string{"DEMO-1003 | text: Match the options yourself"}) || strings.Contains(s, "more-cands") {
		t.Errorf("the next page of alternators: %q\n%s", got, s)
	}

	// Nothing to list: each says why in the part types' place, and where
	// it can, offers a way on.
	for _, c := range []struct {
		h       *Handler
		target  string
		want    []string
		wayRows int // the rows the way on leads to, -1 for no link
	}{
		{demoPages(t, false), "/", []string{`<p class="blank"><span>No catalog loaded.</span><button type="button" popovertarget="vocabularies">Add a catalog</button></p>`}, -1},
		{demoPages(t, false), pickup, []string{"No catalog loaded."}, -1},
		{h, "/", []string{`<p class="blank"><span>Part types turn up once the model is known.</span></p>`}, -1},
		{h, "/?from.vin=1FT", []string{"Part types turn up once the model is known."}, -1},
		{h, "/?from.vin=1HGCM82633A004352", []string{`<p class="blank"><span>demo-catalog has no 2003 HONDA Accord.</span><a href="/ui/vocabularies/demo-catalog">See the vehicles demo-catalog covers</a></p>`}, 0},
		{h, "/?from.vin=1FTFW1EF6DFA12345", []string{`<p class="blank"><span>demo-catalog has the Ford F150, and nothing in it fits a 2013.</span><a href="/ui/vocabularies/demo-catalog?where.make=Ford&amp;where.model=F150">See demo-catalog&#39;s Ford F150 lines</a></p>`},
			len(demoApplications(t, func(_ string, v map[string]string) bool { return v["model"] == "F150" }))},
	} {
		s := fitSection(t, c.h, c.target)
		for _, want := range c.want {
			if !strings.Contains(s, want) {
				t.Errorf("%s: lacks %s\n%s", c.target, want, s)
			}
		}
		if strings.Contains(s, `class="ptwrap"`) {
			t.Errorf("%s: lists part types\n%s", c.target, s)
		}
		m := regexp.MustCompile(`<a href="(/ui/vocabularies/[^"]*)">`).FindStringSubmatch(s)
		if c.wayRows < 0 || m == nil {
			if (m != nil) != (c.wayRows >= 0) {
				t.Errorf("%s: a way on %q; want one: %t", c.target, m, c.wayRows >= 0)
			}
			continue
		}
		// The way on leads to the catalog's lines, for the vehicle when
		// it names one.
		_, _, body := response(t, c.h, "GET", html.UnescapeString(m[1]), nil)
		if rows := strings.Count(body, "<tr><td>"); c.wayRows > 0 && rows != c.wayRows || c.wayRows == 0 && rows == 0 {
			t.Errorf("%s: its way on, %s, has %d rows; want %d\n%s", c.target, m[1], rows, c.wayRows, body)
		}
	}
}

// demoApplications are the demo catalog's applications as its file has
// them, a row each, "type number side | vehicle | years | options", kept
// to those keep holds.
func demoApplications(t *testing.T, keep func(typ string, vehicle map[string]string) bool) []string {
	t.Helper()
	var out []string
	for _, line := range bytes.Split(theDemo().catalog.data, []byte("\n")) {
		var l struct {
			Fitment      struct{ Type, Number, Side string }
			Applications []struct {
				Vehicle     map[string]string
				First, Last int
				Options     string
			}
		}
		if json.Unmarshal(line, &l) != nil || l.Fitment.Number == "" {
			continue
		}
		for _, a := range l.Applications {
			if keep(l.Fitment.Type, a.Vehicle) {
				out = append(out, strings.Join([]string{l.Fitment.Type, l.Fitment.Number, l.Fitment.Side, a.Vehicle["make"] + " " + a.Vehicle["model"], yearsText(contract.Value(int32(a.First)), contract.Value(int32(a.Last))), a.Options}, " | "))
			}
		}
	}
	return out
}

var cellRE = regexp.MustCompile(`<tr><td>(.*?)</td><td>(.*?)</td><td>(.*?)</td><td>(.*?)</td><td>(.*?)</td><td>(.*?)</td></tr>`)

// lineRows are an entries page's rows of a catalog's lines, as
// demoApplications writes them, and its next page, "" on the last.
func lineRows(body string) ([]string, string) {
	var out []string
	for _, m := range cellRE.FindAllStringSubmatch(body, -1) {
		for i := range m[1:] {
			m[i+1] = html.UnescapeString(m[i+1])
		}
		out = append(out, strings.Join(m[1:], " | "))
	}
	next := regexp.MustCompile(`<a href="([^"]*)">Next page</a>`).FindStringSubmatch(body)
	if next == nil {
		return out, ""
	}
	return out, html.UnescapeString(next[1])
}

// TestCatalogEntries: the entries page draws the loaded catalog's fitment
// lines, a row to an application as the file has it, in key order;
// filtered to a part type, a vehicle or both, it shows exactly their
// rows, a page at a time end to end.
func TestCatalogEntries(t *testing.T) {
	h := demoPages(t, true)
	status, _, body := response(t, h, "GET", "/ui/vocabularies/demo-catalog", nil)
	for _, want := range []string{
		"<h1>demo-catalog</h1>",
		"<p>The catalog&#39;s fitment lines, in key order: a row to each application of a part to a vehicle, as the file has it.</p>",
		`<thead><tr><th scope="col">part type</th><th scope="col">number</th><th scope="col">side</th><th scope="col">vehicle</th><th scope="col">years</th><th scope="col">options</th></tr></thead>`,
		`<label for="entry-filter">Show only this part type</label>` + "\n" + `<input type="search" id="entry-filter" name="where.type" value=""`,
		`<label for="entry-filter-make">Show only this make</label>`,
		`<input type="search" id="entry-filter-model" name="where.model" value=""`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the catalog's lines lack %s\n%s", want, body)
		}
	}
	all := demoApplications(t, func(string, map[string]string) bool { return true })
	rows, next := lineRows(body)
	if status != 200 || len(rows) != len(all) || next != "" {
		t.Errorf("the catalog's lines: %d, %d rows; want %d on one page", status, len(rows), len(all))
	}
	slices.Sort(all)
	if slices.Sort(rows); !slices.Equal(rows, all) {
		t.Errorf("the catalog's lines:\n%s\nwant\n%s", strings.Join(rows, "\n"), strings.Join(all, "\n"))
	}

	for _, c := range []struct {
		query string
		keep  func(typ string, v map[string]string) bool
	}{
		{"where.make=Ford&where.model=F150", func(_ string, v map[string]string) bool { return v["model"] == "F150" }},
		{"where.make=toyota", func(_ string, v map[string]string) bool { return v["make"] == "Toyota" }},
		{"where.type=alternator&where.model=Civic", func(typ string, v map[string]string) bool { return typ == "alternator" && v["model"] == "Civic" }},
		{"where.type=alternator&where.make=", func(typ string, _ map[string]string) bool { return typ == "alternator" }},
		{"where.make=Lada", func(string, map[string]string) bool { return false }},
	} {
		want := demoApplications(t, c.keep)
		var got []string
		pages := 0
		for target := "/ui/vocabularies/demo-catalog?limit=2&" + c.query; target != ""; pages++ {
			status, _, body := response(t, h, "GET", target, nil)
			if status != 200 {
				t.Fatalf("%s: %d\n%s", target, status, body)
			}
			var rows []string
			rows, target = lineRows(body)
			got = append(got, rows...)
		}
		slices.Sort(got)
		if slices.Sort(want); !slices.Equal(got, want) {
			t.Errorf("%s: %d pages of\n%s\nwant\n%s", c.query, pages, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	}
}

// TestPageFitFacts: what the vehicle's block says of the vehicle (#366),
// for matching a candidate's option text, which the page cannot check.
// On the block's tape, with the model and the year, it draws what the
// decode says of the vehicle that a catalog's options name, the Vehicle
// card's rows but the model, and the VIN's characters 4 to 8, each with
// its position, so "VIN G, 8th digit" is matched at a glance. The mark on
// such a candidate says in plain words that the options are the reader's
// to match. The block says it whatever is pressed, and before the names
// and the candidates.
func TestPageFitFacts(t *testing.T) {
	h := demoPages(t, true)
	const pickup = "/?from.vin=1FTEW1EG7FFA12345"
	s := fitSection(t, h, pickup+"&part.id=alternator")
	for _, want := range []string{
		`<dl class="vfacts" aria-label="This vehicle, to match the options against">`,
		`<div><dt>Body</dt><dd>Pickup</dd></div>`,
		`<div><dt>Engine</dt><dd>3.5 L · 6 cyl · Gasoline</dd></div>`,
		`<div class="vchars"><dt class="visually-hidden">VIN 4–8</dt><dd><span class="vc"><b>E</b><small>4</small></span><span class="vc"><b>W</b><small>5</small></span><span class="vc"><b>1</b><small>6</small></span><span class="vc"><b>E</b><small>7</small></span><span class="vc"><b>G</b><small>8</small></span></dd></div>`,
		`<span class="mk text">Match the options yourself</span>`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("the alternators lack %s\n%s", want, s)
		}
	}
	if strings.Contains(s, "Catalog text only") || strings.Contains(s, `<dt>Model</dt>`) {
		t.Errorf("the alternators keep the old mark, or repeat the model the strip names\n%s", s)
	}
	facts, names, cands := strings.Index(s, `class="vfacts"`), strings.Index(s, `<ol class="nametags"`), strings.Index(s, `<ol class="cands"`)
	if facts < 0 || names < facts || cands < names {
		t.Errorf("want the vehicle's facts, then its names, then the candidates; they are at %d, %d and %d\n%s", facts, names, cands, s)
	}
	// A sure name has no "i": its note opens from the name itself.
	if !strings.Contains(s, `<button type="button" class="how" popovertarget="aka-`) || strings.Contains(s, `class="stems"`) {
		t.Errorf("a sure name lacks the key over its line, or the lines still run under the block\n%s", s)
	}
	// Nothing pressed: the block says the same.
	if s := fitSection(t, h, pickup); !strings.Contains(s, `<div><dt>Engine</dt><dd>3.5 L · 6 cyl · Gasoline</dd></div>`) {
		t.Errorf("nothing pressed, the block does not say what the vehicle is\n%s", s)
	}
}
