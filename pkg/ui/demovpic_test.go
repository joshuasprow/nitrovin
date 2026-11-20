package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// vpicDemoDir is the demo set's second catalog (#350), whose vehicles are
// vPIC's own: a directory of its own, since one catalog per nitrovin
// holds, which loads in place of demo-catalog.
var vpicDemoDir = filepath.Join(demoDir, "vpic")

// vpicDemoPaths are what `nitrovin --vocab pkg/ui/demo/vpic --vocab
// pkg/ui/demo/demo-part-types.jsonl --vocab pkg/ui/demo/demo-classics.jsonl
// --vocab pkg/ui/demo/demo-part-codes.jsonl serve` loads: the catalog, the
// spine and extension set it names, and the part vocabulary on the spine.
// No vehicle vocabulary: the catalog's vehicles are vPIC's.
var vpicDemoPaths = []string{
	vpicDemoDir,
	filepath.Join(demoDir, "demo-part-types.jsonl"),
	filepath.Join(demoDir, "demo-classics.jsonl"),
	filepath.Join(demoDir, "demo-part-codes.jsonl"),
}

// vpicDemoApplications are the vPIC demo catalog's applications as its
// file has them, a row each as lineRows writes them, "type | number |
// side | vehicle | years | options", the vehicle "vPIC model N", "vPIC
// make N" or the extension, kept to those keep holds.
func vpicDemoApplications(t *testing.T, keep func(typ string, vehicle map[string]any) bool) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(vpicDemoDir, "demo-vpic-catalog.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, line := range bytes.Split(data, []byte("\n")) {
		var l struct {
			Fitment      struct{ Type, Number, Side string }
			Applications []struct {
				Vehicle     map[string]any
				First, Last int
				Options     string
			}
		}
		if json.Unmarshal(line, &l) != nil || l.Fitment.Number == "" {
			continue
		}
		for _, a := range l.Applications {
			if !keep(l.Fitment.Type, a.Vehicle) {
				continue
			}
			var vehicle string
			switch {
			case a.Vehicle["model"] != nil:
				vehicle = fmt.Sprintf("vPIC model %v", a.Vehicle["model"])
			case a.Vehicle["make"] != nil:
				vehicle = fmt.Sprintf("vPIC make %v", a.Vehicle["make"])
			default:
				vehicle = a.Vehicle["ext"].(string)
			}
			out = append(out, strings.Join([]string{l.Fitment.Type, l.Fitment.Number, l.Fitment.Side, vehicle, yearsText(contract.Value(int32(a.First)), contract.Value(int32(a.Last))), a.Options}, " | "))
		}
	}
	return out
}

// TestPageDemoVPIC: the server's pages over the vPIC demo catalog (#350),
// with no vehicle vocabulary loaded, show its vehicles in vPIC's words on
// each surface. The fit card lists a demo VIN's part types and
// candidates, checked as the demo catalog's are; the catalog's lines
// filter by one of make, model and ext; and part-number mode's vehicles
// table names each vehicle by its vPIC reference and the data month's
// name for it.
func TestPageDemoVPIC(t *testing.T) {
	h := pagesOver(t, vpicDemoPaths...)
	const pickup = "/?from.vin=1FTEW1EG7FFA12345"

	// The fit card: the 2015 F-150's part types, by name with counts, the
	// same the demo catalog has; its vehicle is vPIC's F-150 of Ford.
	s := fitSection(t, h, pickup)
	want := []string{"Alternator 3", "Brake Rotor, Front 1", "Brake Rotor, Rear 1", "Door Mirror 2", "Fog Lamp 1", "Headlamp Assembly 2", "Starter Motor 1", "Transfer Case 2"}
	if got := partTypes(s); !slices.Equal(got, want) {
		t.Errorf("the pickup's part types %q; want %q", got, want)
	}
	for _, want := range []string{
		`<h2 id="fit-h" class="stamp">Parts that fit · demo-vpic-catalog</h2>`,
		`<span class="seg g-vds"><span class="w">F-150 / Ford</span></span>`,
		"Choose a part type to see what fits this vehicle.",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("the pickup lacks %s\n%s", want, s)
		}
	}
	for _, c := range []struct {
		target string
		tags   []string
	}{
		{pickup + "&part.id=alternator", []string{
			"DEMO-V1001 | text: Match the options yourself", "DEMO-V1002 | text: Match the options yourself", "DEMO-V1003 | text: Match the options yourself"}},
		// The qualified applications: the 4WD rotor by the decode's drive
		// type; the Silverado's by its series, through the redirect.
		{pickup + "&part.id=brake-rotor-front", []string{"DEMO-V1102 | checked: Checked · Drive Type"}},
		{"/?from.vin=1GCVKREC6EZ123456&part.id=brake-rotor-rear", []string{"DEMO-V2101 | checked: Checked · Series | redirect: By redirect · Brake Rotor, Front"}},
		// A make's application: every Toyota of its years.
		{"/?from.vin=4T1B11HK7JU512345&part.id=starter", []string{"DEMO-V5501 | text: Match the options yourself"}},
	} {
		if got := tags(fitSection(t, h, c.target)); !slices.Equal(got, c.tags) {
			t.Errorf("%s: candidates %q; want %q", c.target, got, c.tags)
		}
	}
	// A year it has nothing for: the way on is the model's lines and the
	// make's, by their vPIC ids.
	s = fitSection(t, h, "/?from.vin=1FTFW1EF6DFA12345")
	if want := `<p class="blank"><span>demo-vpic-catalog has the F-150 and Ford, and nothing in it fits a 2013.</span><a href="/ui/vocabularies/demo-vpic-catalog?where.model=1801">See demo-vpic-catalog&#39;s F-150 lines</a><a href="/ui/vocabularies/demo-vpic-catalog?where.make=460">See demo-vpic-catalog&#39;s Ford lines</a></p>`; !strings.Contains(s, want) {
		t.Errorf("a 2013 F-150 lacks %s\n%s", want, s)
	}

	// The lines view: a filter for each of make, model and ext, each
	// keeping exactly its vehicle's rows, the vehicle by its vPIC id or
	// its extension.
	_, _, body := response(t, h, "GET", "/ui/vocabularies/demo-vpic-catalog", nil)
	for _, want := range []string{
		"<h1>demo-vpic-catalog</h1>",
		`<input type="search" id="entry-filter-make" name="where.make" value=""`,
		`<input type="search" id="entry-filter-model" name="where.model" value=""`,
		`<input type="search" id="entry-filter-ext" name="where.ext" value=""`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the catalog's lines lack %s\n%s", want, body)
		}
	}
	for _, c := range []struct {
		query string
		keep  func(typ string, v map[string]any) bool
		rows  int
	}{
		{"where.model=1801", func(_ string, v map[string]any) bool { return v["model"] == 1801.0 }, 14},
		{"where.make=448", func(_ string, v map[string]any) bool { return v["make"] == 448.0 }, 1},
		{"where.ext=ext:demo-classics/vw-type-2", func(_ string, v map[string]any) bool { return v["ext"] == "ext:demo-classics/vw-type-2" }, 3},
	} {
		target := "/ui/vocabularies/demo-vpic-catalog?" + c.query
		status, _, body := response(t, h, "GET", target, nil)
		got, next := lineRows(body)
		want := vpicDemoApplications(t, c.keep)
		if status != 200 || next != "" || len(want) != c.rows || !slices.Equal(got, want) {
			t.Errorf("%s: %d, next %q,\n%s\nwant %d rows\n%s", target, status, next, strings.Join(got, "\n"), c.rows, strings.Join(want, "\n"))
		}
	}

	// Part-number mode: the vehicles table names each vehicle as VIN mode
	// does, by its vPIC reference and the data month's name, or by its
	// extension and the set's name; no vocabulary renames them.
	for _, c := range []struct {
		target string
		rows   []string
	}{
		{"/?from.fitment.type=alternator&from.fitment.number=DEMO-V1001", []string{"vPIC model 1801 (F-150) | 2015–2017 | 3.5L V6 EcoBoost (VIN G, 8th digit) | "}},
		{"/?from.fitment.type=alternator&from.fitment.number=DEMO-V4001", []string{
			"vPIC model 1863 (Civic) | 2016–2021 | 1.5L turbo | ", "vPIC model 1865 (CR-V) | 2017–2022 | 1.5L turbo | "}},
		{"/?from.fitment.type=brake-rotor-front&from.fitment.number=DEMO-V3101", []string{
			"vPIC model 1938 (Ram) | 2009–2012 |  | ", "vPIC model 13620 (1500) | 2013–2018 |  | "}},
		{"/?from.fitment.type=starter&from.fitment.number=DEMO-V5501", []string{"vPIC make 448 (Toyota) | 2018–2022 | 2.5L 4-cyl | "}},
		{"/?from.fitment.type=mirror&from.fitment.number=DEMO-V7201&from.fitment.side=left", []string{"ext:demo-classics/vw-type-2 (Type 2) | 1968–1979 | chrome, manual | "}},
		// A retired number: its replacement's vehicles.
		{"/?from.fitment.type=alternator&from.fitment.number=DEMO-V0901", []string{"vPIC model 1801 (F-150) | 2015–2017 | 3.5L V6 EcoBoost (VIN G, 8th digit) | "}},
	} {
		s := partSection(t, h, c.target)
		if got := vehicleRows(s); !slices.Equal(got, c.rows) {
			t.Errorf("%s: vehicles %q; want %q\n%s", c.target, got, c.rows, s)
		}
		if !strings.Contains(s, `<thead><tr><th scope="col">Vehicle <span class="hint">as demo-vpic-catalog names it</span></th><th scope="col">Years</th><th scope="col">Options</th></tr></thead>`) {
			t.Errorf("%s: its vehicles' head names other columns\n%s", c.target, s)
		}
	}
}

// TestDemoVPICNotOffered: the vPIC demo catalog is a server's and the
// tests' (#350), not the browser's demo. The package embeds no file of
// demo/vpic, so the demo set's files are exactly demo/*.jsonl, each as
// before; a server publishes none of it; and Load demo loads none of it.
func TestDemoVPICNotOffered(t *testing.T) {
	if _, err := fs.Stat(demoFS, "demo/vpic"); err == nil {
		t.Error("the package embeds demo/vpic")
	}
	top, err := filepath.Glob(filepath.Join(demoDir, "*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var want, got []string
	for _, p := range top {
		want = append(want, filepath.Base(p))
	}
	for _, f := range theDemo().files {
		got = append(got, f.FileName)
	}
	if !slices.Equal(got, want) || len(got) != 7 {
		t.Errorf("the demo set's files %q; want demo/*.jsonl, %q", got, want)
	}

	data, err := os.ReadFile(filepath.Join(vpicDemoDir, "demo-vpic-catalog.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := vocab.Read("demo-vpic-catalog.jsonl", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	vpic := contract.VocabularyFile{Name: f.Name, Kind: string(f.Kind), Blake3: fmt.Sprintf("%x", f.Hash)}
	if theDemo().file(vpic.Name) != nil || demoPublished(vpic.Name, vpic.Blake3) != nil {
		t.Errorf("the demo set holds or publishes %s", vpic.Name)
	}
	h := demoPages(t, false).WithWorker(Worker{Module: []byte("\x00asm"), Exec: []byte("// go")})
	if status, _, body := response(t, h, "GET", FilePath(h.info(), vpic), nil); status != 404 {
		t.Errorf("%s at its URL: %d\n%s", vpic.Name, status, body)
	}
	c := theDemo().catalog
	if status, _, _ := response(t, h, "GET", FilePath(h.info(), contract.VocabularyFile{Name: c.Name, Blake3: c.Blake3}), nil); status != 200 {
		t.Errorf("the demo's own catalog at its URL: %d", status)
	}

	panel := demoPanelOf(t, openWith(t, newDecoderServer(t, embedded(t)), newDisk()))
	if !strings.Contains(panel, `id="load-demo"`) || strings.Contains(panel, vpic.Name) {
		t.Errorf("Load demo offers %s, or is not offered\n%s", vpic.Name, panel)
	}
}
