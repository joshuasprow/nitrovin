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
// vPIC catalog with eighteen part types for a 2015 F-150, and a part
// vocabulary of their groups.
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
	headingRE = regexp.MustCompile(`<h3 id="ptg-\d+">([^<]*)</h3>`)
	pickerRE  = regexp.MustCompile(`<button type="button" id="pick-type" class="ptfield" popovertarget="part-types"><span class="plab">Part type</span><span class="nm">([^<]*)</span><span class="n">(\d+)<span class="visually-hidden"> ([^<]*)</span></span>`)
	clearRE   = regexp.MustCompile(`<a class="ptclear swap" id="clear-type" data-focus="pick-type" href="([^"]*)" aria-label="([^"]*)">`)
)

// pickerOf is a section's picker as its markup has it: "name · count
// noun", then " · clear href" where it has the key that clears it.
func pickerOf(s string) string {
	m := pickerRE.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	out := html.UnescapeString(m[1]) + " · " + m[2] + " " + m[3]
	if c := clearRE.FindStringSubmatch(s); c != nil {
		out += " · " + html.UnescapeString(c[2]) + " " + html.UnescapeString(c[1])
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

// TestPartTypePicker: the part types of a vehicle with many (#366). The
// picker asks for one and counts them; with one pressed it names it,
// counts its candidates, and the key beside it links to the page with
// none. Its list, the part types dialog, holds every part type under the
// headings of a part vocabulary on the catalog's spine (the one the
// catalog files its fitments under, else the first loaded): under each
// entry that maps to it, so under two entries for a type two map to, and
// under "Other" for one none does; by name with none loaded. The pressed
// part type's link in the list is the page itself, so pressing it again
// changes nothing. Without a script nothing on the page is dead: the
// filter and the recent part types are hidden until plate.js shows them.
func TestPartTypePicker(t *testing.T) {
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
	if got, want := pickerOf(s), "Choose a part type · 18 part types"; got != want {
		t.Errorf("nothing pressed, the picker: %q; want %q\n%s", got, want, s)
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
		`<div class="ptwrap" data-spine="strip-parts">`,
		`<dialog id="part-types" class="ptdialog" popover aria-labelledby="part-types-h">`,
		`<p class="hint">2015 F-150 / Ford in strip-catalog · grouped by strip-groups</p>`,
		// Without a script: no filter, and no recent part types.
		`<p class="ptf" hidden><label class="visually-hidden" for="ptf-dialog">Filter part types</label>`,
		`<section class="ptgroup ptrecent" aria-labelledby="ptrecent-h" hidden>`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("the pickup's part types lack %s\n%s", want, s)
		}
	}
	// The recent part types are the list's alone: the picker's tape holds
	// the picker and the key that clears it.
	if strings.Contains(s, "ptclear") || strings.Contains(s, "Seats") || strings.Contains(s, `class="ptypes"`) || strings.Contains(s, "ptrecents") {
		t.Errorf("nothing pressed: a key that clears, a heading that holds none, the strip of part types, or recent ones on the tape\n%s", s)
	}

	// Pressed, a part type is the picker's, with its candidates' count
	// and the key that clears it; in the list it is pressed under its
	// heading, where every part type keeps its place and its link is the
	// page as it is.
	const pumped = pickup + "&part.id=water-pump"
	s = fitSection(t, h, pumped)
	if got, want := pickerOf(s), "Water Pump · 1 candidate · Clear Water Pump "+pickup; got != want {
		t.Errorf("Water Pump pressed, the picker: %q; want %q\n%s", got, want, s)
	}
	if got := dialogOf(s); len(got) != len(groups) || got[1] != "Cooling: Radiator / Thermostat / *Water Pump" {
		t.Errorf("Water Pump pressed, the dialog:\n%s", strings.Join(got, "\n"))
	}
	if !strings.Contains(s, `<a class="ptype swap" href="`+html.EscapeString(pumped)+`" aria-current="true"><span class="nm">Water Pump</span>`) {
		t.Errorf("Water Pump pressed: its link in the list is not the page itself\n%s", s)
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
