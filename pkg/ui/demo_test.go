package ui

import (
	"bytes"
	"encoding/json"
	"html"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
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

// epaSample is cmd/nitrovin-epa's sample of EPA's vocabulary, which the
// demo's translation into EPA's names is asked beside.
var epaSample = filepath.Join("..", "..", "cmd", "nitrovin-epa", "testdata", "vehicles.jsonl")

// demoPicked are the demo set's files of the kinds given, as vocab.js
// picks them from the release's assets.
func demoPicked(kinds ...string) []Picked {
	var out []Picked
	for _, f := range theDemo().files {
		if slices.Contains(kinds, f.Kind) {
			out = append(out, Picked{FileName: f.FileName, Data: f.data})
		}
	}
	return out
}

// loadDemo loads the demo set into l as vocab.js's Load the demo does:
// its vocabularies as one batch, its catalog compiled, then its shelf and
// yard as one batch.
func loadDemo(t *testing.T, l *Library) {
	t.Helper()
	if _, err := l.Load(demoPicked("spine", "vocabulary", "extension-set"), quiet); err != nil {
		t.Fatal(err)
	}
	c := theDemo().catalog
	if _, err := l.Compile(bytes.NewReader(c.data), c.FileName, int64(len(c.data)), vocab.DirScratch(t.TempDir()), quiet); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Load(demoPicked("stock"), quiet); err != nil {
		t.Fatal(err)
	}
}

// demoServer is `nitrovin serve --vocab pkg/ui/demo --vocab <EPA's>`: the
// answers the browser's must give.
func demoServer(t *testing.T, d *nitrovin.Decoder) *nitrovin.Translator {
	t.Helper()
	files, err := vocab.Open(demoDir, epaSample)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := nitrovin.NewTranslator(d, files...)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

// TestLibraryDemoFit: the demo set, loaded into the browser as Load demo
// loads it (its catalog compiled there), answers byte for byte as
// `nitrovin serve --vocab` does with the same files, a sweep over its
// vehicles, VINs, part types, part codes, fitments, sides and stock; and
// only the yard's VINs went to the server.
func TestLibraryDemoFit(t *testing.T) {
	d := embedded(t)
	srv := demoServer(t, d)
	s, disk := newDecoderServer(t, d), newDisk()
	l := openWith(t, s, disk)
	if _, err := l.Load([]Picked{{FileName: "epa-vehicles.jsonl", Data: mustRead(t, epaSample)}}, quiet); err != nil {
		t.Fatal(err)
	}
	loadDemo(t, l)
	var names []string
	for _, f := range l.Files() {
		if f.Fault != "" {
			t.Errorf("%s left out: %s", f.Name, f.Fault)
		}
		names = append(names, f.Name)
	}
	if want := []string{"demo-catalog", "demo-classics", "demo-part-codes", "demo-part-types", "demo-shelf", "demo-vehicles", "demo-yard-stock", "epa-vehicles"}; !slices.Equal(names, want) {
		t.Fatalf("stored %q; want %q", names, want)
	}

	// The yard's VINs went to the server, each once, in one batch, and
	// nothing else of the demo did.
	var vins []string
	for _, p := range s.posts {
		var items []map[string]any
		if err := json.Unmarshal(p, &items); err != nil {
			t.Fatal(err)
		}
		for _, it := range items {
			if len(it) != 2 || it["full"] != true {
				t.Errorf("POSTed %v; want a VIN to decode", it)
			}
			vins = append(vins, it["vin"].(string))
		}
	}
	if len(s.posts) != 1 || len(vins) != 20 {
		t.Errorf("%d POSTs of %d VINs; want the yard's 20 VINs in one", len(s.posts), len(vins))
	}
	for _, field := range []string{"DEMO-Y-", "DEMO-S-", "stock_number", "demo-north"} {
		if strings.Contains(string(bytes.Join(s.posts, nil))+strings.Join(s.gets, " "), field) {
			t.Errorf("%q went to the server", field)
		}
	}

	found := sameDemoFits(t, srv, l)
	t.Logf("%d answers found", found)
	if found < 500 {
		t.Fatalf("%d answers found; want many", found)
	}
}

// pageFitTargets are pages the demo set's parts that fit answer: part
// types, each mark, a redirect, a VIN with no year, and each place with
// nothing to list.
var pageFitTargets = []string{
	"/?from.vin=1FTEW1EG7FFA12345",
	"/?from.vin=1FTEW1EG7FFA12345&part.id=alternator",
	"/?from.vin=1FTEW1EG7FFA12345&part.id=brake-rotor-front",
	"/?from.vin=1FTEW1EG7FFA12345&part.id=brake-rotor-rear",
	"/?from.vin=1FTEW1EG7FFA12345&part.id=mirror",
	"/?from.vin=1FTEW1EG7FFA12345&part.id=wiper",
	"/?from.vin=2HKRW2H53KH612345&part.id=transfer-case",
	"/?from.vin=1GCVKREC6EZ123456&part.id=brake-rotor-rear",
	"/?from.vin=1FTEW1EG&part.id=alternator",
	"/?from.vin=1HGCM82633A004352",
	"/?from.vin=1FTFW1EF6DFA12345",
	"/?from.vin=1FT",
}

// TestLibraryPageFit: the browser's library answers the page's parts
// that fit as the server's pages do, from the catalog the browser holds,
// over the page's own decode, asked for once; and, while the server holds
// the catalog instead, from the server's /v1/fit, its Developers JSON the
// server's every field (a redirect's candidate's redirects and via
// included). The catalog's lines are the browser's too.
func TestLibraryPageFit(t *testing.T) {
	d := embedded(t)
	srv := New(served{demoTranslator(t, d, true)}, log.New(io.Discard, "", 0))

	s := newDecoderServer(t, d)
	l := openWith(t, s, newDisk())
	loadDemo(t, l)
	browser := NewInBrowser(l, BrowserConfig{}, log.New(io.Discard, "", 0))
	for _, target := range pageFitTargets {
		s.gets = nil
		want, got := fitSection(t, srv, target), fitSection(t, browser, target)
		if got != want {
			t.Errorf("%s in the browser:\n%s\nwant the server's\n%s", target, got, want)
		}
		decodes := 0
		for _, g := range s.gets {
			if strings.HasPrefix(g, "/v1/decode/") {
				decodes++
			}
		}
		if decodes > 1 {
			t.Errorf("%s: the browser asked for %d decodes: %q", target, decodes, s.gets)
		}
	}
	for _, target := range []string{"/ui/vocabularies/demo-catalog?where.make=Ford&where.model=F150", "/ui/vocabularies/demo-catalog?limit=3&where.type=alternator"} {
		_, _, body := response(t, srv, "GET", target, nil)
		want, wantNext := lineRows(body)
		_, _, body = response(t, browser, "GET", target, nil)
		if got, next := lineRows(body); !slices.Equal(got, want) || next != wantNext || len(got) == 0 {
			t.Errorf("%s in the browser: %q, next %q; want the server's %q, next %q", target, got, next, want, wantNext)
		}
	}

	// The server holds the catalog: the browser's page asks its /v1/fit.
	s = newDecoderServer(t, d)
	for _, f := range theDemo().files {
		s.serve(t, f.Name, f.data)
	}
	s.fits = demoTranslator(t, d, true)
	l = openWith(t, s, newDisk())
	if l.ServerCatalog().Name != "demo-catalog" || l.ServerCatalog().Vehicles != "demo-vehicles" {
		t.Fatalf("the server's catalog %+v; want demo-catalog of demo-vehicles", l.ServerCatalog())
	}
	browser = NewInBrowser(l, BrowserConfig{}, log.New(io.Discard, "", 0))
	for _, target := range pageFitTargets {
		s.gets = nil
		if want, got := fitSection(t, srv, target), fitSection(t, browser, target); got != want {
			t.Errorf("%s in the browser, the server's catalog:\n%s\nwant the server's\n%s", target, got, want)
		}
		if strings.Contains(target, "&part.id=alternator") && !slices.ContainsFunc(s.gets, func(g string) bool {
			return strings.HasPrefix(g, "/v1/fit?") && strings.Contains(g, "part.id=alternator")
		}) {
			t.Errorf("%s: the browser asked %q; want the server's fit", target, s.gets)
		}
	}
	silverado := "/?from.vin=1GCVKREC6EZ123456&part.id=brake-rotor-rear"
	for _, id := range []string{"fit-json", "candidates-json"} {
		if want, got := devJSON(t, srv, silverado, id), devJSON(t, browser, silverado, id); got == "" || got != want {
			t.Errorf("%s's %s in the browser, the server's catalog:\n%s\nwant the server's\n%s", silverado, id, got, want)
		}
	}
	if got := devJSON(t, browser, silverado, "candidates-json"); !strings.Contains(got, `"redirects": [`) || !strings.Contains(got, `"via": [`) {
		t.Errorf("%s's candidates' JSON in the browser lacks the server's redirects and via:\n%s", silverado, got)
	}
}

// devJSON is the Developers dialog's JSON of id on target's page, as
// text, without its "now" line: each fit is asked at its own second.
func devJSON(t *testing.T, h *Handler, target, id string) string {
	t.Helper()
	_, _, body := response(t, h, "GET", target, nil)
	m := regexp.MustCompile(`(?s)<pre tabindex="0" id="` + id + `"[^>]*>(.*?)</pre>`).FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	lines := strings.Split(html.UnescapeString(m[1]), "\n")
	return strings.Join(slices.DeleteFunc(lines, func(l string) bool { return strings.HasPrefix(strings.TrimSpace(l), `"now":`) }), "\n")
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// sameDemoFits asks the sweep of the demo set of both: every vehicle of
// its vehicle vocabulary, extension and yard VIN, in and across years,
// with every part type and part code on every side, and related; and
// every fitment and retired number; each with and without stock. It
// returns how many answers were found.
func sameDemoFits(t *testing.T, srv *nitrovin.Translator, l *Library) int {
	t.Helper()
	d := theDemo()
	var froms []contract.FitFrom
	var parts []*contract.FitPart
	var keys []contract.FitmentKey
	for _, f := range d.files {
		file, err := vocab.Read(f.FileName, bytes.NewReader(f.data))
		if err != nil {
			t.Fatal(err)
		}
		switch f.Name {
		case "demo-vehicles":
			for _, e := range file.Entries {
				froms = append(froms, contract.FitFrom{Vocabulary: f.Name, Entry: entryOf(file, e)})
			}
		case "demo-classics":
			for _, x := range file.Extensions {
				froms = append(froms, contract.FitFrom{SpineRef: contract.SpineRef{Ext: "ext:" + f.Name + "/" + x.ID}})
			}
		case "demo-part-types":
			for _, e := range file.SpineEntries {
				parts = append(parts, &contract.FitPart{Spine: f.Name, ID: e.ID})
			}
		case "demo-part-codes":
			for _, e := range file.Entries {
				parts = append(parts, &contract.FitPart{Vocabulary: f.Name, Entry: entryOf(file, e)})
			}
		case "demo-yard-stock":
			for _, r := range file.Stock {
				if r.VIN != "" {
					froms = append(froms, contract.FitFrom{VIN: r.VIN})
				}
			}
		}
	}
	lines, err := l.List(contract.ListRequest{Of: d.catalog.Name, Limit: 1000})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range lines.Entries {
		keys = append(keys, e.(contract.FitmentLine).Fitment)
	}
	keys = append(keys, contract.FitmentKey{Type: "alternator", Number: "DEMO-0901"}, contract.FitmentKey{Type: "alternator", Number: "DEMO-0000"})
	years := []contract.Null[int32]{{}, contract.Value[int32](1950), contract.Value[int32](1974), contract.Value[int32](2012), contract.Value[int32](2015), contract.Value[int32](2018), contract.Value[int32](2019)}
	asks := []stockAsk{{stock: []string{"shelf", "yard"}}, {stock: []string{"shelf", "yard"}, location: []string{"demo-north"}}}
	found := 0
	for _, from := range froms {
		ys := years
		if from.VIN != "" {
			ys = years[:1]
		}
		for _, year := range ys {
			for _, to := range [][]string{nil, {"demo-vehicles", "epa-vehicles"}} {
				found += sameFitStock(t, srv, l, nil, contract.FitRequest{From: from, ModelYear: year, To: to})
				found += sameFitStock(t, srv, l, asks, contract.FitRequest{From: from, ModelYear: year, To: to, Related: true, Limit: 1})
				for _, part := range parts {
					for _, side := range []string{"", "left", "right", "both"} {
						found += sameFitStock(t, srv, l, asks, contract.FitRequest{From: from, ModelYear: year, Part: part, Side: side, To: to, Limit: 1})
					}
				}
			}
		}
	}
	for _, k := range keys {
		for _, year := range []contract.Null[int32]{{}, contract.Value[int32](2018)} {
			found += sameFitStock(t, srv, l, asks, contract.FitRequest{From: contract.FitFrom{Fitment: &k}, ModelYear: year, To: []string{"demo-vehicles"}, Limit: 1})
		}
	}
	return found
}

// entryOf is e, an entry of f, as a request names it.
func entryOf(f *vocab.File, e vocab.Entry) map[string]string {
	out := map[string]string{}
	for i, name := range f.Fields {
		out[name] = e.Fields[i]
	}
	return out
}

// demoPanelOf is the worker's panel over l, offering the demo.
func demoPanelOf(t *testing.T, l *Library) string {
	t.Helper()
	h := NewInBrowser(l, BrowserConfig{Info: l.Info()}, log.New(io.Discard, "", 0))
	r := httptest.NewRequest("GET", filesPath, nil)
	r.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("/ui/files: HTTP %d: %s", w.Code, w.Body)
	}
	return w.Body.String()
}

// TestDemoPanel: the worker's panel offers the demo set as one click, its
// files at the release's URLs; holds it once loaded, with Remove demo
// naming every file; offers a refresh, replacing what it holds, once one
// file is another; and refuses it in words while the server or the
// browser holds another catalog, or the server a file of a demo name.
func TestDemoPanel(t *testing.T) {
	d := embedded(t)
	s, disk := newDecoderServer(t, d), newDisk()
	l := openWith(t, s, disk)
	panel := demoPanelOf(t, l)
	m := regexp.MustCompile(`<button type="button" id="load-demo" data-demo="([^"]*)"[^>]*>Load demo</button>`).FindStringSubmatch(panel)
	if m == nil {
		t.Fatalf("no Load demo in the panel:\n%s", panel)
	}
	var load demoLoad
	if err := json.Unmarshal([]byte(htmlUnescape(m[1])), &load); err != nil {
		t.Fatal(err)
	}
	if len(load.Files) != 7 || len(load.Remove) != 0 {
		t.Fatalf("Load the demo loads %+v", load)
	}
	for _, f := range load.Files {
		if !strings.HasPrefix(f.Name, "demo-") || !strings.HasPrefix(f.URL, assetsPath(Release(l.Info()))+"/vocab/") || !strings.HasSuffix(f.URL, "/"+f.File) {
			t.Errorf("the demo's %+v", f)
		}
	}

	loadDemo(t, l)
	panel = demoPanelOf(t, l)
	if !strings.Contains(panel, `id="remove-demo"`) || strings.Contains(panel, `id="load-demo"`) {
		t.Errorf("with the demo held, the panel:\n%s", panel)
	}
	for _, f := range theDemo().files {
		if !strings.Contains(panel, `<input type="hidden" name="name" value="`+f.Name+`">`) {
			t.Errorf("Remove the demo does not name %s", f.Name)
		}
	}

	// Another catalog in the browser refuses it, naming it.
	other := newDisk()
	lo := openWith(t, s, other)
	if _, err := lo.Load([]Picked{{FileName: "catalog-models.jsonl", Data: fitmentFile(t, "catalog-models")}, {FileName: "part-types.jsonl", Data: fitmentFile(t, "part-types")}, {FileName: "part-codes.jsonl", Data: fitmentFile(t, "part-codes")}}, quiet); err != nil {
		t.Fatal(err)
	}
	if _, err := compileFixture(t, lo, fitmentFile(t, "catalog")); err != nil {
		t.Fatal(err)
	}
	if p := demoPanelOf(t, lo); !strings.Contains(p, "The demo brings its own catalog, and this browser holds one, parts-catalog: one catalog per nitrovin. Remove parts-catalog to load the demo.") || strings.Contains(p, `id="load-demo"`) {
		t.Errorf("with another catalog in the browser, the panel:\n%s", p)
	}

	// A demo file changed (a newer release's) is offered as a refresh,
	// which removes what the browser holds first.
	shelf := theDemo().file("demo-shelf")
	changed := bytes.Replace(shelf.data, []byte(`"version":"2026.11-demo"`), []byte(`"version":"2026.10-demo"`), 1)
	if _, err := l.Load([]Picked{{FileName: shelf.FileName, Data: changed}}, quiet); err != nil {
		t.Fatal(err)
	}
	panel = demoPanelOf(t, l)
	m = regexp.MustCompile(`<button type="button" id="load-demo" data-demo="([^"]*)"[^>]*>Refresh demo</button>`).FindStringSubmatch(panel)
	if m == nil {
		t.Fatalf("no Refresh demo in the panel:\n%s", panel)
	}
	if err := json.Unmarshal([]byte(htmlUnescape(m[1])), &load); err != nil {
		t.Fatal(err)
	}
	if len(load.Remove) != 7 {
		t.Errorf("a refresh removes %q first; want every demo file held", load.Remove)
	}
	if err := l.Remove(load.Remove, false); err != nil {
		t.Fatalf("removing the demo: %v", err)
	}
	if got := stored(l); len(got) != 0 {
		t.Errorf("after removing the demo: %q", got)
	}

	// A server with a catalog refuses it, and so does one with a file of
	// a demo name; neither page tells the visitor to load it.
	withCatalog := newDecoderServer(t, d, "part-types", "catalog-models")
	withCatalog.serve(t, "catalog", fitmentFile(t, "catalog"))
	lc := openWith(t, withCatalog, newDisk())
	if p := demoPanelOf(t, lc); !strings.Contains(p, "The demo brings its own catalog, and this server has one, catalog: one catalog per nitrovin, so the demo cannot load here.") || strings.Contains(p, `id="load-demo"`) {
		t.Errorf("with a catalog on the server, the panel:\n%s", p)
	}
	named := newDecoderServer(t, d)
	named.serve(t, "demo-part-types", theDemo().file("demo-part-types").data)
	ln := openWith(t, named, newDisk())
	if p := demoPanelOf(t, ln); !strings.Contains(p, "This server loaded demo-part-types, named like a file of the demo&#39;s: one name is one file (ADR 0011), so the demo cannot load here.") || strings.Contains(p, `id="load-demo"`) {
		t.Errorf("with a file of a demo name on the server, the panel:\n%s", p)
	}
	for _, refused := range []*Library{lc, ln} {
		h := NewInBrowser(refused, BrowserConfig{Info: refused.Info()}, log.New(io.Discard, "", 0))
		if h.demoLoadable() {
			t.Errorf("a refused demo is loadable beside %v", refused.info.Vocabularies)
		}
	}
}

func htmlUnescape(s string) string {
	return strings.NewReplacer("&#34;", `"`, "&quot;", `"`, "&amp;", "&", "&lt;", "<", "&gt;", ">", "&#39;", "'").Replace(s)
}
