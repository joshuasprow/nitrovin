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
	stripItemRE = regexp.MustCompile(`<li data-part="[^"]*"><a class="ptype swap" href="[^"]*"( aria-current="true")?>([^<]*) <span class="n">.*?</a><span class="ptmore( none)?" aria-hidden="true">([^<]*)</span></li>`)
	headingRE   = regexp.MustCompile(`<h3 id="ptg-\d+">([^<]*)</h3>`)
)

// stripOf is a section's strip as its markup has it: each part type in
// order as "name | N more", "name |" for the last, which counts nothing,
// and "*" before the pressed one's name.
func stripOf(s string) []string {
	var out []string
	for _, m := range stripItemRE.FindAllStringSubmatch(s, -1) {
		item := html.UnescapeString(m[2]) + " | " + m[4]
		if m[1] != "" {
			item = "*" + item
		}
		if (m[3] != "") != (m[4] == "") {
			item += " (a count marked none, or none unmarked)"
		}
		out = append(out, strings.TrimSpace(item))
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
// first, each with the count of those after it, the last with none: the
// stylesheet shows the count of the last its two rows hold, "N more",
// and the key under it opens the part types dialog. The dialog holds
// every part type under the headings of a part vocabulary on the
// catalog's spine (the one the catalog files its fitments under, else
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
	want := []string{
		"Alternator | 17 more", "Battery | 16 more", "Brake Caliper, Front | 15 more", "Brake Pads, Front | 14 more", "Brake Pads, Rear | 13 more",
		"Brake Rotor, Front | 12 more", "Brake Rotor, Rear | 11 more", "Door Mirror | 10 more", "Fog Lamp | 9 more", "Fuel Pump | 8 more",
		"Headlamp Assembly | 7 more", "Radiator | 6 more", "Starter Motor | 5 more", "Tail Lamp | 4 more", "Thermostat | 3 more",
		"Transfer Case | 2 more", "Water Pump | 1 more", "Wiper Motor |",
	}
	if got := stripOf(s); !slices.Equal(got, want) {
		t.Errorf("the strip:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
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
		`<button type="button" id="more-types" popovertarget="part-types">More<span class="visually-hidden"> part types</span></button>`,
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
	want = []string{"*Water Pump | 17 more", "Alternator | 16 more", "Battery | 15 more"}
	if got := stripOf(s); len(got) != len(byName) || !slices.Equal(got[:3], want) || got[len(got)-1] != "Wiper Motor |" {
		t.Errorf("Water Pump pressed, the strip:\n%s\nwant it to start\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if got := dialogOf(s); len(got) != len(groups) || got[1] != "Cooling: Radiator / Thermostat / *Water Pump" {
		t.Errorf("Water Pump pressed, the dialog:\n%s", strings.Join(got, "\n"))
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
