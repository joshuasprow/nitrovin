package ui

import (
	"html"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/joshuasprow/nitrovin/pkg/nitrovin"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// stripPath is a file of the strip fixture (testdata/vocab/strip): a
// vPIC catalog with eighteen part types for a 2015 F-150, more than two
// rows of the strip hold, and a part vocabulary of their groups.
func stripPath(name string) string {
	return filepath.Join("..", "..", "testdata", "vocab", "strip", name+".jsonl")
}

// pagesOver is the server's pages over the files at paths, loaded in
// that order.
func pagesOver(t *testing.T, paths ...string) *Handler {
	t.Helper()
	files, err := vocab.Open(paths...)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := nitrovin.NewTranslator(embedded(t), files...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tr.Close)
	return New(served{tr}, log.New(io.Discard, "", 0))
}

// partVocabulary writes a part vocabulary named name on spine to a
// temporary file: one entry, group "Everything", mapping ids.
func partVocabulary(t *testing.T, name, spine string, ids ...string) string {
	t.Helper()
	var maps []string
	for _, id := range ids {
		maps = append(maps, `{"id":"`+id+`"}`)
	}
	p := filepath.Join(t.TempDir(), name+".jsonl")
	body := `{"format":1,"kind":"vocabulary","name":"` + name + `","version":"1","subject":"part","fields":["group"],"spine":"` + spine + `"}` + "\n" +
		`{"entry":{"group":"Everything"},"mappings":[` + strings.Join(maps, ",") + `]}` + "\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

var (
	stripNameRE = regexp.MustCompile(`<li data-part="[^"]*"><a class="ptype swap" href="[^"]*"( aria-current="true")?><span class="nm">([^<]*)</span>`)
	keyRE       = regexp.MustCompile(`<span class="kn c(\d)" data-slots="(\d+)">(\d+) more<span class="visually-hidden"> ([^<]*)</span></span>`)
	headingRE   = regexp.MustCompile(`<h3 id="ptg-\d+">([^<]*)</h3>`)
)

// stripOf is a section's strip as its markup has it: each part type's
// name in order, "*" before the pressed one's.
func stripOf(s string) []string {
	if i := strings.Index(s, `<ul class="ptypes"`); i >= 0 {
		s = s[i : i+strings.Index(s[i:], "</ul>")]
	}
	var out []string
	for _, m := range stripNameRE.FindAllStringSubmatch(s, -1) {
		name := html.UnescapeString(m[2])
		if m[1] != "" {
			name = "*" + name
		}
		out = append(out, name)
	}
	return out
}

// keyOf is a section's "N more" at each width: "columns: N more noun,
// slots beside the filter".
func keyOf(s string) []string {
	var out []string
	for _, m := range keyRE.FindAllStringSubmatch(s, -1) {
		out = append(out, m[1]+": "+m[3]+" more "+m[4]+", "+m[2])
	}
	return out
}

// dialogOf is a section's part types dialog: each group as "heading:
// name / name", the heading "" without one, "*" before the pressed one's
// name.
func dialogOf(s string) []string {
	start := strings.Index(s, `<div class="ptgroups"`)
	if start < 0 {
		return nil
	}
	s = s[start : start+strings.Index(s[start:], "</dialog>")]
	var out []string
	for _, sec := range strings.Split(s, `<section class="ptgroup"`)[1:] {
		heading := ""
		if m := headingRE.FindStringSubmatch(sec); m != nil {
			heading = html.UnescapeString(m[1])
		}
		var names []string
		for _, m := range partTypeRE.FindAllStringSubmatch(sec, -1) {
			n := html.UnescapeString(m[3])
			if m[2] != "" {
				n = "*" + n
			}
			names = append(names, n)
		}
		out = append(out, heading+": "+strings.Join(names, " / "))
	}
	return out
}

// TestPartTypeStrip: a vehicle with more part types than two rows of the
// strip hold (#324). The strip has every one, by name, the pressed one
// first, and its key says how many two rows of the strip have no room for
// at each width: of 5, 4, 3 and 2 columns, 2c − 1 part types and the key,
// or every part type and no key where they fit in 2c. The dialog the key
// opens holds every part type under the headings of a part vocabulary on
// the catalog's spine (the one the catalog files its fitments under, else
// the first loaded): under each entry that maps to it, so under two
// entries for a type two map to, and under "Other" for one none does; by
// name with none loaded. Without a script nothing on the page is dead:
// the filters and the recent part types are hidden until plate.js shows
// them, and no part type is marked recent.
func TestPartTypeStrip(t *testing.T) {
	const pickup = "/?from.vin=1FTEW1EG7FFA12345"
	byName := []string{
		"Alternator", "Battery", "Brake Caliper, Front", "Brake Pads, Front", "Brake Pads, Rear", "Brake Rotor, Front", "Brake Rotor, Rear",
		"Door Mirror", "Fog Lamp", "Fuel Pump", "Headlamp Assembly", "Radiator", "Starter Motor", "Tail Lamp", "Thermostat", "Transfer Case",
		"Water Pump", "Wiper Motor",
	}
	// A second part vocabulary on the spine, loaded after the strip's
	// groups: the first loaded is the one the headings come from.
	later := partVocabulary(t, "later-groups", "strip-parts", "alternator", "battery")
	h := pagesOver(t, stripPath("strip-parts"), stripPath("strip-groups"), later, stripPath("strip-catalog"))

	s := fitSection(t, h, pickup)
	if got := stripOf(s); !slices.Equal(got, byName) {
		t.Errorf("the strip:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(byName, "\n"))
	}
	// Eighteen: 9 of 10 cells and the key at 5 columns, 7 at 4, 5 at 3, 3
	// at 2; beside the script's filter, which takes a cell but on a
	// phone, one cell fewer.
	keys := []string{"5: 9 more part types, 9", "4: 11 more part types, 7", "3: 13 more part types, 5", "2: 15 more part types, 4"}
	if got := keyOf(s); !slices.Equal(got, keys) {
		t.Errorf("the key: %q; want %q", got, keys)
	}
	groups := []string{
		"Brakes: Brake Caliper, Front / Brake Pads, Front / Brake Pads, Rear / Brake Rotor, Front / Brake Rotor, Rear",
		"Cooling: Radiator / Thermostat / Water Pump",
		"Electrical: Alternator / Battery / Headlamp Assembly / Starter Motor",
		"Lighting: Fog Lamp / Headlamp Assembly / Tail Lamp",
		"Other: Door Mirror / Fuel Pump / Transfer Case / Wiper Motor",
	}
	if got := dialogOf(s); !slices.Equal(got, groups) {
		t.Errorf("the dialog:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(groups, "\n"))
	}
	for _, want := range []string{
		`<li class="ptkey"><button type="button" id="more-types" popovertarget="part-types"><span class="kn c5" data-slots="9">9 more<span class="visually-hidden"> part types</span></span>`,
		`<dialog id="part-types" class="ptdialog" popover aria-labelledby="part-types-h">`,
		`<p class="hint">2015 F-150 / Ford in strip-catalog · grouped by strip-groups</p>`,
		// Without a script: no filter, and no recent part types.
		`<li class="ptf" hidden><label class="visually-hidden" for="ptf-strip">Filter part types</label>`,
		`<p class="ptf" hidden><label class="visually-hidden" for="ptf-dialog">Filter part types</label>`,
		`<section class="ptgroup ptrecent" aria-labelledby="ptrecent-h" hidden>`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("the pickup's strip lacks %s\n%s", want, s)
		}
	}
	if strings.Contains(s, "rstamp") || strings.Contains(s, "Seats") {
		t.Errorf("the pickup's strip has a recent part type, or a heading that holds none\n%s", s)
	}

	// Pressed, a part type leads the strip, wherever it was, and is
	// pressed in the dialog under its heading.
	s = fitSection(t, h, pickup+"&part.id=water-pump")
	want := append([]string{"*Water Pump"}, slices.DeleteFunc(slices.Clone(byName), func(n string) bool { return n == "Water Pump" })...)
	if got := stripOf(s); !slices.Equal(got, want) {
		t.Errorf("Water Pump pressed, the strip:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if got := dialogOf(s); len(got) != len(groups) || got[1] != "Cooling: Radiator / Thermostat / *Water Pump" {
		t.Errorf("Water Pump pressed, the dialog:\n%s", strings.Join(got, "\n"))
	}

	// The Silverado's six fit two rows of 5, 4 and 3 columns: no width
	// but a phone's draws a key, whose count is never 1, as a strip one
	// over shows its last part type in the key's place.
	s = fitSection(t, h, "/?from.vin=1GCVKREC6EZ123456")
	keys = []string{"5: 0 more part types, 9", "4: 0 more part types, 7", "3: 0 more part types, 5", "2: 3 more part types, 4"}
	if got := keyOf(s); !slices.Equal(got, keys) {
		t.Errorf("the Silverado's key: %q; want %q", got, keys)
	}
	for n := range 30 {
		for _, k := range stripKeys(n) {
			if k.N == 1 || (k.N == 0) != (n <= 2*k.Cols) {
				t.Errorf("%d part types at %d columns: %d more", n, k.Cols, k.N)
			}
		}
	}

	// With no part vocabulary on the spine, the dialog is one list by
	// name, and says so.
	s = fitSection(t, pagesOver(t, stripPath("strip-parts"), stripPath("strip-catalog")), pickup)
	if got := dialogOf(s); !slices.Equal(got, []string{": " + strings.Join(byName, " / ")}) {
		t.Errorf("no part vocabulary, the dialog: %q", got)
	}
	if !strings.Contains(s, `<p class="hint">2015 F-150 / Ford in strip-catalog</p>`) || headingRE.MatchString(s) {
		t.Errorf("no part vocabulary, the dialog has headings or groups by one\n%s", s)
	}

	// A catalog that files its fitments under a part vocabulary takes its
	// headings from that one, though another was loaded first: the
	// fitment fixture's Ram has a front rotor two codes file.
	first := partVocabulary(t, "first-groups", "part-types", "brake-rotor-front", "engine", "mirror")
	var paths []string
	for _, n := range []string{"part-types", "part-codes", "catalog-models", "catalog"} {
		paths = append(paths, fitmentPath(n))
	}
	s = fitSection(t, pagesOver(t, slices.Insert(paths, 1, first)...), "/?from.vin=1D7RV1GT5AS123456")
	groups = []string{"ENG: Engine", "FRT: Brake Rotor, Front", "MIR: Door Mirror", "ROT: Brake Rotor, Front"}
	if got := dialogOf(s); !slices.Equal(got, groups) {
		t.Errorf("filed under part-codes, the dialog:\n%s\nwant\n%s\n%s", strings.Join(got, "\n"), strings.Join(groups, "\n"), s)
	}
	if !strings.Contains(s, " · grouped by part-codes</p>") {
		t.Errorf("filed under part-codes, the dialog does not say so\n%s", s)
	}
}
