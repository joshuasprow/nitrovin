package ui

import (
	"context"
	"html"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/nitrovin"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// askedFits is a server's pages that keep every fit asked of them.
type askedFits struct {
	served
	asked *[]contract.FitRequest
}

func (a askedFits) Fit(ctx context.Context, req contract.FitRequest) (contract.Fit, error) {
	*a.asked = append(*a.asked, req)
	return a.served.Fit(ctx, req)
}

// stockPages are the server's pages over tr, keeping the fits asked of
// them in asked.
func stockPages(tr *nitrovin.Translator, asked *[]contract.FitRequest) *Handler {
	return New(askedFits{served{tr}, asked}, log.New(io.Discard, "", 0))
}

// demoStock is `nitrovin serve --vocab pkg/ui/demo` with its catalog and
// only the stock files named.
func demoStock(t *testing.T, stock ...string) *nitrovin.Translator {
	t.Helper()
	files, err := vocab.Open(demoDir)
	if err != nil {
		t.Fatal(err)
	}
	files = slices.DeleteFunc(files, func(f *vocab.File) bool { return f.Kind == vocab.KindStock && !slices.Contains(stock, f.Name) })
	tr, err := nitrovin.NewTranslator(embedded(t), files...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tr.Close)
	return tr
}

var (
	stripRE     = regexp.MustCompile(`(?s)<(a|p|button) [^>]*class="stk[^"]*"[^>]*>(.*?)</(?:a|p|button)>`)
	stripTagRE  = regexp.MustCompile(`<(?:a|p|button) [^>]*class="stk[^"]*"[^>]*>`)
	saidRE      = regexp.MustCompile(`<span class="visually-hidden">([^<]*)</span>`)
	stockRowRE  = regexp.MustCompile(`(?s)<tbody>\n<tr><th scope="row" class="id">([^<]*)</th><td class="loc">([^<]*)</td>(.*?)</tr>\n(?:<tr class="rest"><td colspan="\d+">(.*?)</td></tr>\n)?</tbody>`)
	stockCellRE = regexp.MustCompile(`<td data-k="([^"]*)">([^<]*)</td>`)
	stockFoldRE = regexp.MustCompile(`<div(?: class="json")?><dt>([^<]*)</dt> <dd>([^<]*)</dd></div>`)
	stockNoteRE = regexp.MustCompile(`(?s)<p class="note">(.*?)</p>`)
)

// moreStockNames are the stock fixtures with shelf-more among them, in
// the browser's order: by name.
func moreStockNames() []string {
	return slices.Sorted(slices.Values(append([]string{"shelf-more"}, stockNames...)))
}

// strips are a section's candidates' stock strips, "number what: words",
// what a link, words, or the Vocabularies dialog's button. A link's
// words are what a screen reader is told, less the number they end in:
// its cells are drawn for the eye alone.
func strips(s string) []string {
	var out []string
	for _, li := range strings.Split(s, `<li class="cand">`)[1:] {
		number := numberRE.FindStringSubmatch(li)[1]
		m := stripRE.FindStringSubmatch(li)
		if m == nil {
			out = append(out, number+" no strip")
			continue
		}
		what, words := map[string]string{"a": "link", "p": "words", "button": "dialog"}[m[1]], m[2]
		if said := saidRE.FindStringSubmatch(m[2]); m[1] == "a" && said != nil {
			words = strings.TrimSuffix(said[1], ", "+number)
		}
		out = append(out, number+" "+what+": "+plain(words))
	}
	return out
}

// strip is the opening tag of the strip of the candidate numbered number.
func strip(s, number string) string {
	for _, li := range strings.Split(s, `<li class="cand">`)[1:] {
		if numberRE.FindStringSubmatch(li)[1] == number {
			return html.UnescapeString(stripTagRE.FindString(li))
		}
	}
	return ""
}

// stockRows are rows of a print, "id | location | folded fields and
// notes", with "name=value" for each of the print's columns before the
// folded fields where it has any.
func stockRows(s string) []string {
	var out []string
	for _, m := range stockRowRE.FindAllStringSubmatch(s, -1) {
		parts := []string{m[1], m[2]}
		var cells, folded []string
		for _, c := range stockCellRE.FindAllStringSubmatch(m[3], -1) {
			cells = append(cells, c[1]+"="+c[2])
		}
		if cells != nil {
			parts = append(parts, strings.Join(cells, ", "))
		}
		for _, f := range stockFoldRE.FindAllStringSubmatch(m[4], -1) {
			folded = append(folded, f[1]+" "+f[2])
		}
		rest := strings.Join(folded, " · ")
		for _, n := range stockNoteRE.FindAllStringSubmatch(m[4], -1) {
			rest += " " + plain(n[1])
		}
		out = append(out, html.UnescapeString(strings.Join(append(parts, strings.TrimSpace(rest)), " | ")))
	}
	return out
}

// openRows are the open candidate's rows in a section: the candidate
// whose list item holds them, then each kind of stock, its heading, its
// rows and its "Show all" line, if any; nil with none open.
func openRows(s string) [][]string {
	start := strings.Index(s, `<div class="rows" id="stock-rows">`)
	if start < 0 {
		return nil
	}
	before := strings.Split(s[:start], `<li class="cand">`)
	out := [][]string{{numberRE.FindStringSubmatch(before[len(before)-1])[1]}}
	panel := s[start:]
	panel = panel[:strings.Index(panel, "</li>")]
	for _, sec := range strings.Split(panel, `<section class="stock-kind"`)[1:] {
		col := []string{plain(sec[strings.Index(sec, `<p class="kind"`):strings.Index(sec, "</p>")])}
		col = append(col, stockRows(sec)...)
		if i := strings.Index(sec, `<p class="more-rows">`); i >= 0 {
			col = append(col, plain(sec[i:strings.Index(sec[i:], "</p>")+i]))
		}
		out = append(out, col)
	}
	return out
}

// moreHref is the "Show all" link of a section's open candidate.
func moreHref(s string) string {
	m := regexp.MustCompile(`<p class="more-rows">[^<]*<a class="swap" href="([^"]*)"`).FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return html.UnescapeString(m[1])
}

// stockAsked are the stock each fit asked, "from: stock", from a VIN with
// a part type, a VIN alone, or a fitment key.
func stockAsked(asked []contract.FitRequest) []string {
	var out []string
	for _, r := range asked {
		from := "vin"
		switch {
		case r.From.Fitment != nil:
			from = "fitment " + r.From.Fitment.Number
		case r.Part != nil:
			from = "vin+part"
		}
		out = append(out, from+": "+strings.Join(r.Stock, ","))
	}
	return out
}

// TestPageStock: every candidate ends in a stock strip that counts the
// stock loaded, the candidates' request asking for it, and no other
// request; it names only the sources loaded, says none, or opens the
// Vocabularies dialog when none is. A strip is a link to the page with
// its candidate's rows open, right after its tag: shelf and yard under
// their counts and as-of dates, a row's id, location and fields, a shelf
// row tagged with a retired number saying which and the yard's vehicle
// that is the VIN looked up saying so. Ten rows of each, and "Show all"
// past ten, a page with every row. The shelf rows with no number are
// listed once, after the candidates.
func TestPageStock(t *testing.T) {
	const (
		pickup      = "/?from.vin=1FTEW1EG7FFA12345"
		alternators = pickup + "&part.id=alternator"
	)
	var asked []contract.FitRequest
	h := stockPages(demoStock(t, "demo-shelf", "demo-yard-stock"), &asked)

	// Nothing pressed: no candidate, no strip, and no stock asked for.
	if s := fitSection(t, h, pickup); strings.Contains(s, `class="stk`) {
		t.Errorf("the pickup has a strip with no part type pressed\n%s", s)
	}
	if got := stockAsked(asked); !slices.Equal(got, []string{"vin: "}) {
		t.Errorf("the pickup asked %q; want its part types, no stock", got)
	}

	// Both sources: every strip counts both, the candidates in the
	// catalog's order, all from the one request.
	asked = nil
	s := fitSection(t, h, alternators)
	want := []string{"DEMO-1001 link: Stock · 2 shelf · 2 yard", "DEMO-1002 link: Stock · 0 shelf · 2 yard", "DEMO-1003 link: Stock · 0 shelf · 2 yard"}
	if got := strips(s); !slices.Equal(got, want) {
		t.Errorf("the alternators' strips %q; want %q\n%s", got, want, s)
	}
	if got := stockAsked(asked); !slices.Equal(got, []string{"vin: ", "vin+part: shelf,yard"}) {
		t.Errorf("the alternators asked %q; want the part types, then the candidates with both kinds of stock", got)
	}
	if openRows(s) != nil {
		t.Errorf("the alternators have rows open\n%s", s)
	}
	if got := strip(s, "DEMO-1001"); got != `<a class="stk swap" id="stk-1" href="/?from.vin=1FTEW1EG7FFA12345&part.id=alternator&fitment.number=DEMO-1001" aria-expanded="false">` {
		t.Errorf("DEMO-1001's strip: %s", got)
	}

	// Opened: its rows right after its tag, from the same one request.
	asked = nil
	opened := alternators + "&fitment.number=DEMO-1001"
	s = fitSection(t, h, opened)
	wantRows := [][]string{
		{"DEMO-1001"},
		{"On the shelf · 2 as of 2026-11-12",
			"DEMO-S-01 | demo-north | stock_number DEMO-S-01 · condition demo: used, grade A",
			"DEMO-S-02 | demo-south | stock_number DEMO-S-02 · condition demo: used, grade B; tagged with its old number Tagged with a retired number, DEMO-0901"},
		{"In the yard · 2 as of 2026-11-12",
			"DEMO-Y-01 | demo-north | stock_number DEMO-Y-01 · color demo: white · row D-1 This VIN : the vehicle being looked up",
			"DEMO-Y-02 | demo-south | stock_number DEMO-Y-02 · color demo: blue · row D-1"},
	}
	if got := openRows(s); !slices.EqualFunc(got, wantRows, slices.Equal) {
		t.Errorf("DEMO-1001 open: %q; want %q\n%s", got, wantRows, s)
	}
	if got := strip(s, "DEMO-1001"); got != `<a class="stk swap" id="stk-1" href="/?from.vin=1FTEW1EG7FFA12345&part.id=alternator" aria-expanded="true" aria-controls="stock-rows">` {
		t.Errorf("DEMO-1001's strip, open: %s", got)
	}
	if got := stockAsked(asked); !slices.Equal(got, []string{"vin: ", "vin+part: shelf,yard"}) {
		t.Errorf("DEMO-1001 open asked %q; want no more than the candidates' request", got)
	}
	if _, _, body := response(t, h, "GET", opened, nil); !strings.Contains(body, `data-number="DEMO-1001"`) ||
		!strings.Contains(body, `id="copy-url" data-copy="this page's URL">http://example.com/?from.vin=1FTEW1EG7FFA12345&amp;part.id=alternator&amp;fitment.number=DEMO-1001</pre>`) {
		t.Errorf("the page with DEMO-1001 open lacks its URL\n%s", body)
	}
	// Opening another closes the first.
	s = fitSection(t, h, alternators+"&fitment.number=DEMO-1002")
	if got := openRows(s); len(got) != 3 || got[0][0] != "DEMO-1002" || got[1][0] != "On the shelf · 0 as of 2026-11-12" || len(got[1]) != 1 || len(got[2]) != 3 ||
		!strings.Contains(strip(s, "DEMO-1001"), `aria-expanded="false"`) || strings.Count(s, `id="stock-rows"`) != 1 {
		t.Errorf("DEMO-1002 open: %q\n%s", got, s)
	}

	// A candidate a redirect reaches is of another type than the part
	// type pressed: its strip names its type, and only its whole key
	// opens it.
	const rears = "/?from.vin=1GCVKREC6EZ123456&part.id=brake-rotor-rear"
	s = fitSection(t, h, rears)
	if got := strip(s, "DEMO-2101"); !strings.Contains(got, `href="`+rears+`&fitment.type=brake-rotor-front&fitment.number=DEMO-2101"`) {
		t.Errorf("the redirected rotor's strip: %s", got)
	}
	if got := openRows(fitSection(t, h, rears+"&fitment.type=brake-rotor-front&fitment.number=DEMO-2101")); len(got) != 3 || got[0][0] != "DEMO-2101" {
		t.Errorf("the redirected rotor, its whole key: %q", got)
	}
	if got := openRows(fitSection(t, h, rears+"&fitment.number=DEMO-2101")); got != nil {
		t.Errorf("the redirected rotor, opened as a rear rotor: %q", got)
	}

	// One source: the strip names it alone; with none for a candidate it
	// says so and presses nothing, and a URL cannot open it.
	asked = nil
	shelf := stockPages(demoStock(t, "demo-shelf"), &asked)
	s = fitSection(t, shelf, alternators)
	want = []string{"DEMO-1001 link: Stock · 2 shelf", "DEMO-1002 words: Stock · none", "DEMO-1003 words: Stock · none"}
	if got := strips(s); !slices.Equal(got, want) {
		t.Errorf("the alternators, a shelf alone: %q; want %q\n%s", got, want, s)
	}
	if got := stockAsked(asked); !slices.Equal(got, []string{"vin: ", "vin+part: shelf"}) {
		t.Errorf("a shelf alone, asked %q; want the candidates with the shelf", got)
	}
	if got := openRows(fitSection(t, shelf, alternators+"&fitment.number=DEMO-1001")); len(got) != 2 || got[1][0] != "On the shelf · 2 as of 2026-11-12" {
		t.Errorf("DEMO-1001 open, a shelf alone: %q; want one column", got)
	}
	if got := openRows(fitSection(t, shelf, alternators+"&fitment.number=DEMO-1002")); got != nil {
		t.Errorf("DEMO-1002, with none, opened: %q", got)
	}

	// A yard alone names the yard alone. With both sources loaded and
	// nothing in either, the strip says none: the 2010 Ram's alternator.
	yard := stockPages(demoStock(t, "demo-yard-stock"), &asked)
	want = []string{"DEMO-1001 link: Stock · 2 yard", "DEMO-1002 link: Stock · 2 yard", "DEMO-1003 link: Stock · 2 yard"}
	if got := strips(fitSection(t, yard, alternators)); !slices.Equal(got, want) {
		t.Errorf("the alternators, a yard alone: %q; want %q", got, want)
	}
	if got := strips(fitSection(t, h, "/?from.vin=1D7RV1GT5AS123456&part.id=alternator")); !slices.Equal(got, []string{"DEMO-3001 words: Stock · none"}) {
		t.Errorf("the Ram's alternator, both sources loaded: %q; want none", got)
	}

	// A yard that does not answer, a connector this build cannot reach:
	// the strip says so, never a 0, and still opens, its column saying
	// why.
	down := stockPages(demoStockWith(t, []string{filepath.Join("..", "..", "testdata", "vocab", "stock", "live", "yard-stock.jsonl")}, "demo-shelf"), &asked)
	s = fitSection(t, down, alternators)
	want = []string{"DEMO-1001 link: Stock · 2 shelf · yard unavailable", "DEMO-1002 link: Stock · 0 shelf · yard unavailable", "DEMO-1003 link: Stock · 0 shelf · yard unavailable"}
	if got := strips(s); !slices.Equal(got, want) {
		t.Errorf("the alternators, a yard down: %q; want %q\n%s", got, want, s)
	}
	if got := openRows(fitSection(t, down, alternators+"&fitment.number=DEMO-1002")); len(got) != 3 ||
		got[2][0] != "In the yard · 0 unavailable: this build reads stock files only: a connector needs the nitrovin binary" || len(got[2]) != 1 {
		t.Errorf("DEMO-1002 open, a yard down: %q", got)
	}

	// No stock loaded: every strip says so and opens the Vocabularies
	// dialog, and the candidates ask for none.
	asked = nil
	none := stockPages(demoStock(t), &asked)
	s = fitSection(t, none, alternators)
	want = []string{"DEMO-1001 dialog: Stock · not loaded Add stock", "DEMO-1002 dialog: Stock · not loaded Add stock", "DEMO-1003 dialog: Stock · not loaded Add stock"}
	if got := strips(s); !slices.Equal(got, want) || strip(s, "DEMO-1001") != `<button type="button" class="stk" popovertarget="vocabularies">` {
		t.Errorf("the alternators, no stock: %q, %s; want %q", got, strip(s, "DEMO-1001"), want)
	}
	if got := stockAsked(asked); !slices.Equal(got, []string{"vin: ", "vin+part: "}) {
		t.Errorf("no stock, asked %q; want no stock asked for", got)
	}

	// Past ten rows: ten, and "Show all", a page with every row, which
	// asks the fitment's own stock.
	asked = nil
	fixture := stockPages(stockServerOf(t, embedded(t), moreStockNames()), &asked)
	const rotors = "/?from.vin=1D7RV1GT5AS123456&part.id=brake-rotor-front"
	s = fitSection(t, fixture, rotors+"&fitment.number=BR-2001")
	got := openRows(s)
	if len(got) != 3 || len(got[1]) != 12 || got[1][0] != "On the shelf · 16 shelf-annex as of 2026-10-30 · shelf-more as of 2026-11-02 · shelf-orphans as of 2026-11-01 · shelf-stock as of 2026-11-01T08:00:00Z" ||
		got[1][11] != "The first 10 of 16. Show all" {
		t.Errorf("BR-2001 open: %q\n%s", got, s)
	}
	wantYard := []string{"In the yard · 3 as of 2026-11-01T06:30:00Z",
		"Y-1 | north | stock_number Y100 · color red · row B · photos [\"y100-1.jpg\",\"y100-2.jpg\"] This VIN : the vehicle being looked up",
		"Y-2 | south | stock_number Y101 · color white · row C This VIN : the vehicle being looked up",
		"Y-3 | south | stock_number Y102 · color blue · row A · added 2026-10-28"}
	if !slices.Equal(got[2], wantYard) {
		t.Errorf("BR-2001's yard: %q; want %q", got[2], wantYard)
	}
	all := moreHref(s)
	if all != rotors+"&fitment.number=BR-2001&rows=all" {
		t.Fatalf("BR-2001's Show all: %q", all)
	}
	asked = nil
	s = fitSection(t, fixture, all)
	if got := openRows(s); len(got) != 3 || len(got[1]) != 17 || len(got[2]) != 4 || moreHref(s) != "" ||
		!slices.Contains(got[1], "S-3 | north | stock_number A102 · condition A · price 52 Tagged with a retired number, BR-1999") {
		t.Errorf("BR-2001, every row: %q\n%s", got, s)
	}
	if got := stockAsked(asked); !slices.Equal(got, []string{"vin: ", "vin+part: shelf,yard", "fitment BR-2001: shelf,yard"}) {
		t.Errorf("every row asked %q; want the fitment's own stock last", got)
	}
	if _, _, body := response(t, fixture, "GET", all, nil); !strings.Contains(body, `<pre tabindex="0" id="stock-json" data-copy="the stock&#39;s JSON">{`) || !strings.Contains(body, `data-rows="all"`) {
		t.Errorf("every row: the page lacks the stock's JSON\n%s", body)
	}

	// The shelf rows with no number: once, after the candidates, under a
	// strip naming the part type and the vehicle in their own case, their
	// count and as-of dates.
	s = fitSection(t, fixture, "/?from.vin=1D7RV1GT5AS123456&part.id=mirror")
	at := strings.Index(s, `<div class="nonum" id="stock-nonum" tabindex="-1">`)
	if at < 0 || at < strings.LastIndex(s, "</ol>") || !strings.Contains(s, `<h3 class="nonum-tape">On the shelf with no number · Door Mirror · 2010 Dodge Ram 1500</h3>`) ||
		!strings.Contains(plain(s[at:]), "On the shelf · 2 shelf-annex as of 2026-10-30 · shelf-more as of 2026-11-02 · shelf-orphans as of 2026-11-01 · shelf-stock as of 2026-11-01T08:00:00Z") {
		t.Fatalf("the mirrors' rows with no number:\n%s", s)
	}
	if got, want := stockRows(s[at:]), []string{"A-2 | east | stock_number C301", "S-4 | north | stock_number A103 · comments no number: tag reads 'left mirror'"}; !slices.Equal(got, want) || len(stockRows(s)) != 2 {
		t.Errorf("the mirrors' rows with no number: %q; want %q, once", got, want)
	}
	if got := openRows(fitSection(t, fixture, "/?from.vin=1D7RV1GT5AS123456&part.id=mirror&fitment.number=MR-1001&fitment.side=right")); len(got) != 3 || got[0][0] != "MR-1001" || got[1][0] != "On the shelf · 0 shelf-annex as of 2026-10-30 · shelf-more as of 2026-11-02 · shelf-orphans as of 2026-11-01 · shelf-stock as of 2026-11-01T08:00:00Z" {
		t.Errorf("the right mirror open: %q", got)
	}

	// They are the first page's alone, and never keep the candidates'
	// "Next page": one rotor a page, three rows with no number, the first
	// shown, and "Show all" a page with all three.
	extra := filepath.Join(t.TempDir(), "shelf-loose.jsonl")
	loose := `{"format":1,"kind":"stock","name":"shelf-loose","version":"1","source":"shelf","as_of":"2026-11-03","catalog":"parts-catalog"}` + "\n"
	for _, id := range []string{"L-1", "L-2", "L-3"} {
		loose += `{"id":"` + id + `","location":"west","part":{"spine":"part-types","id":"brake-rotor-front"},"vehicle":{"make":"Dodge","model":"Ram 1500"},"year":2010,"fields":{"stock_number":"` + id + `"}}` + "\n"
	}
	if err := os.WriteFile(extra, []byte(loose), 0o600); err != nil {
		t.Fatal(err)
	}
	paths := []string{fitmentPath("part-types"), fitmentPath("part-codes"), fitmentPath("catalog-models"), fitmentPath("catalog"), extra}
	files, err := vocab.Open(paths...)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := nitrovin.NewTranslator(embedded(t), files...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tr.Close)
	paged := stockPages(tr, &asked)
	paged.candidates = 1
	s = fitSection(t, paged, rotors)
	at = strings.Index(s, `<div class="nonum"`)
	if got := stockRows(s); at < 0 || !slices.Equal(got, []string{"L-1 | west | stock_number L-1"}) || strings.Contains(s, "more-cands") ||
		!strings.Contains(plain(s[at:]), "On the shelf · 3 as of 2026-11-03") || !strings.Contains(plain(s[at:]), "The first 1 of 3. Show all") {
		t.Errorf("one rotor, three rows with no number: %q\n%s", got, s)
	}
	m := regexp.MustCompile(`<p class="more-rows">[^<]*<a class="swap" href="([^"]*)" data-focus="stock-nonum">Show all</a>`).FindStringSubmatch(s)
	if m == nil || html.UnescapeString(m[1]) != rotors+"&rows=unnumbered" {
		t.Fatalf("the rows with no number's Show all: %q", m)
	}
	s = fitSection(t, paged, rotors+"&rows=unnumbered")
	if got := stockRows(s); !slices.Equal(got, []string{"L-1 | west | stock_number L-1", "L-2 | west | stock_number L-2", "L-3 | west | stock_number L-3"}) || strings.Contains(s, "more-rows") || strings.Contains(s, "more-cands") {
		t.Errorf("every row with no number: %q\n%s", got, s)
	}
	// Two mirrors, one a page: the first page's "Next page" is the
	// candidates', and the second page lists no rows with no number.
	paged = stockPages(stockServerOf(t, embedded(t), moreStockNames()), &asked)
	paged.candidates = 1
	s = fitSection(t, paged, "/?from.vin=1D7RV1GT5AS123456&part.id=mirror")
	next := regexp.MustCompile(`<a class="swap" href="([^"]*)">Next page</a>`).FindStringSubmatch(s)
	if next == nil || !strings.Contains(s, `<div class="nonum"`) {
		t.Fatalf("the first mirror's page:\n%s", s)
	}
	s = fitSection(t, paged, html.UnescapeString(next[1]))
	if got := tags(s); len(got) != 1 || strings.Contains(s, `class="nonum"`) || strings.Contains(s, "more-cands") {
		t.Errorf("the second mirror's page: %q\n%s", got, s)
	}
}

// demoStockWith is demoStock with the files at paths loaded besides.
func demoStockWith(t *testing.T, paths []string, stock ...string) *nitrovin.Translator {
	t.Helper()
	files, err := vocab.Open(append([]string{demoDir}, paths...)...)
	if err != nil {
		t.Fatal(err)
	}
	files = slices.DeleteFunc(files, func(f *vocab.File) bool {
		return f.Kind == vocab.KindStock && f.Connector == nil && !slices.Contains(stock, f.Name)
	})
	tr, err := nitrovin.NewTranslator(embedded(t), files...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tr.Close)
	return tr
}

// TestStockPrint: a candidate's or a part's open rows are a print in
// columns. The region names the part; a kind of stock is a section under
// its heading; a row is its id, its location, and under each field its
// source's header declares (#390), in the header's order, the row's value
// for it, as its file wrote it. What a row has beyond those is under it,
// a name and its value each: a null and an empty string print nothing,
// and a list or an object, which is no column's though declared, is its
// JSON. A source that declares nothing has no column but the id and the
// location. The rows with no number are a print under the tape that
// names them. No print has a head, a foot or a barcode.
func TestStockPrint(t *testing.T) {
	order := func(name string) string {
		return filepath.Join("..", "..", "testdata", "vocab", "stock", "order", name+".jsonl")
	}
	h := pagesOver(t, fitmentPath("part-types"), fitmentPath("part-codes"), fitmentPath("catalog-models"), fitmentPath("catalog"), order("shelf-order"), order("yard-order"))
	s := partSection(t, h, "/?from.fitment.type=engine&from.fitment.number=EN-57&fitment.number=EN-57")
	for _, want := range []string{
		`<div class="rows" id="stock-rows">` + "\n" + `<div class="receipt" id="stock-region" role="region" aria-label="Stock of EN-57" tabindex="-1">`,
		// A kind of stock is a heading one level under the part's number.
		`<section class="stock-kind" aria-labelledby="stock-shelf-h">` + "\n" + `<p class="kind" id="stock-shelf-h" role="heading" aria-level="3"><span class="bar">On the shelf · 4</span> <span class="asof">as of 2026-11-03</span></p>`,
		`<thead><tr><th scope="col" class="id"><span class="visually-hidden">Id</span></th><th scope="col" class="loc"><span class="visually-hidden">Location</span></th>` +
			`<th scope="col">stock_number</th><th scope="col">condition</th><th scope="col">price</th><th scope="col">warranty</th></tr></thead>`,
		`<th scope="col">stock_number</th><th scope="col">row</th><th scope="col">photos</th></tr></thead>`,
		`<div class="json"><dt>photos</dt> <dd>[{&#34;url&#34;:&#34;v900-front.jpg&#34;,`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("EN-57's print lacks %s\n%s", want, s)
		}
	}
	want := []string{
		"R-1 | west | stock_number=E500, condition=A, price=1850.00, warranty= | tag T-9 · miles 61250 · status on shelf",
		`R-2 | west | stock_number=E501, condition=, price=9.5e2, warranty= | comments turns over; sold "as is"`,
		"R-3 | west | stock_number=, condition=, price=, warranty= | tag T-11",
		"R-4 | west | stock_number=E503, condition=B, price=1200, warranty= | tag T-12",
		`V-1 | west | stock_number=V900, row=K, photos= | photos [{"url":"v900-front.jpg","w":1600,"h":1200},{"url":"v900-rear.jpg","w":1600,"h":1200}] · color grey · added 2026-10-30`,
		"V-2 | west | stock_number=, row=, photos= | color black · keys true",
	}
	if got := stockRows(s); !slices.Equal(got, want) {
		t.Errorf("EN-57's rows\n%s\nwant\n%s\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"), s)
	}
	for _, not := range []string{"receipt-head", "receipt-foot", "barcode", `class="print bare"`} {
		if strings.Contains(s, not) {
			t.Errorf("EN-57's print has %s\n%s", not, s)
		}
	}

	// Sources that declare nothing: an id and a location, the rest folded.
	var asked []contract.FitRequest
	fixture := stockPages(stockServerOf(t, embedded(t), moreStockNames()), &asked)
	s = fitSection(t, fixture, "/?from.vin=1D7RV1GT5AS123456&part.id=brake-rotor-front&fitment.number=BR-2001")
	if !strings.Contains(s, `<table class="print bare">`) || strings.Contains(s, `<table class="print">`) || strings.Contains(s, "data-k=") ||
		!strings.Contains(s, `<p class="more-rows">The first 10 of 16. <a class="swap"`) {
		t.Errorf("BR-2001's print, of sources that declare no field: want no column but the id and the location, and its sixteen rows counted\n%s", s)
	}

	// The rows with no number: the same print, under their tape.
	s = fitSection(t, fixture, "/?from.vin=1D7RV1GT5AS123456&part.id=mirror")
	at := strings.Index(s, `<div class="nonum"`)
	if at < 0 {
		t.Fatalf("the mirrors have no rows with no number:\n%s", s)
	}
	if n := s[at:]; !strings.Contains(n, `</h3>`+"\n"+`<div class="receipt">`+"\n"+`<section class="stock-kind" aria-labelledby="stock-nonum-h">`+"\n"+`<p class="kind" id="stock-nonum-h" role="heading" aria-level="4">`) || len(stockRows(n)) != 2 {
		t.Errorf("the rows with no number: want a print of two rows under their tape:\n%s", n)
	}
}

// TestStockColumn: which of a row's fields are its print's columns and
// which fold under it. The columns are the fields the rows' sources
// declare, each once; a field is its row's column only when the row's
// own source declares it and its value is no list or object; a null and
// an empty string print nothing, in a column or folded; anything else
// folds, in the source's order, a list or an object as its JSON.
func TestStockColumn(t *testing.T) {
	row := func(source, id, fields string) contract.StockRow {
		return contract.StockRow{Source: source, ID: id, Fields: []byte(fields)}
	}
	l := &contract.StockList{Total: 6, Sources: []contract.StockSource{{Name: "bare"}, {Name: "a"}, {Name: "b"}, {Name: "unseen"}}, Rows: []contract.StockRow{
		row("bare", "1", `{"x":"1","note":"n"}`),
		row("a", "2", `{"x":"2","y":null,"z":"","list":[1, 2],"n":3.50,"ok":false}`),
		row("a", "3", `{"y":{"k": "v"},"x":["p"]}`),
		row("b", "4", `{"y":"4","w":0,"x":"not b's"}`),
		row("a", "5", `{}`),
		row("a", "6", `"loose"`),
	}}
	declared := map[string][]string{"a": {"x", "y"}, "b": {"y", "w"}, "unseen": {"never"}}
	text := func(fields []stockField) string {
		var out []string
		for _, f := range fields {
			s := f.Name + "=" + f.Value
			if f.JSON {
				s += " (JSON)"
			}
			out = append(out, s)
		}
		return strings.Join(out, ", ")
	}
	col := newStockColumn(contract.StockShelf, l, declared, "", "")
	if want := []string{"x", "y", "w"}; !slices.Equal(col.Heads, want) || col.Span() != 5 {
		t.Errorf("the columns %q, %d cells a row; want %q and 5: a source with no row here adds none", col.Heads, col.Span(), want)
	}
	want := []string{
		"1: x=, y=, w= | x=1, note=n",
		"2: x=2, y=, w= | list=[1,2] (JSON), n=3.50, ok=false",
		`3: x=, y=, w= | y={"k":"v"} (JSON), x=["p"] (JSON)`,
		"4: x=, y=4, w=0 | x=not b's",
		"5: x=, y=, w= | ",
		"6: x=, y=, w= | =loose",
	}
	var got []string
	for _, r := range col.Rows {
		got = append(got, r.ID+": "+text(r.Cells)+" | "+text(r.Folded))
	}
	if !slices.Equal(got, want) {
		t.Errorf("the rows\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if col.Rows[4].Rest() || !col.Rows[0].Rest() {
		t.Errorf("row 5, with no field, has a second line, or row 1 has none")
	}

	// No source declares a field: no column, and one cell fills the line.
	col = newStockColumn(contract.StockShelf, l, nil, "", "")
	if col.Heads != nil || col.Span() != 3 || len(col.Rows[1].Cells) != 0 || text(col.Rows[1].Folded) != "x=2, list=[1,2] (JSON), n=3.50, ok=false" {
		t.Errorf("nothing declared: columns %q, %d cells a row, row 2 %q | %q", col.Heads, col.Span(), text(col.Rows[1].Cells), text(col.Rows[1].Folded))
	}
}
