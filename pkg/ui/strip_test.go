package ui

import (
	"html"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
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
	stripNameRE = regexp.MustCompile(`<li data-part="[^"]*"(?: class="([^"]*)")?><a class="ptype swap" href="[^"]*"( aria-current="true")?><span class="nm">([^<]*)</span>`)
	keyClassRE  = regexp.MustCompile(`<li class="([^"]*)"><button type="button" id="more-types"`)
	keyRE       = regexp.MustCompile(`<span class="kn c(\d)">(\d+) more<span class="visually-hidden"> ([^<]*)</span></span>`)
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
		name := html.UnescapeString(m[3])
		if m[2] != "" {
			name = "*" + name
		}
		out = append(out, name)
	}
	return out
}

// laidOf is a section's strip as cols columns draw it without a script:
// the part types not past its two rows, " ×2" after one that takes two
// cells, then the key, "N more noun", where it is drawn.
func laidOf(s string, cols int) []string {
	c := strconv.Itoa(cols)
	if i := strings.Index(s, `<ul class="ptypes"`); i >= 0 {
		s = s[i : i+strings.Index(s[i:], "</ul>")]
	}
	has := func(classes, class string) bool { return slices.Contains(strings.Fields(classes), class) }
	var out []string
	for _, m := range stripNameRE.FindAllStringSubmatch(s, -1) {
		if has(m[1], "x"+c) {
			continue
		}
		name := html.UnescapeString(m[3])
		if has(m[1], "s"+c) {
			name += " ×2"
		}
		out = append(out, name)
	}
	key := keyClassRE.FindStringSubmatch(s)
	for _, m := range keyRE.FindAllStringSubmatch(s, -1) {
		if m[1] != c {
			continue
		}
		switch drawn := key != nil && has(key[1], "k"+c); {
		case drawn:
			out = append(out, m[2]+" more "+m[3])
		case m[2] != "0":
			out = append(out, "a key of "+m[2]+" not drawn")
		}
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
// first. At each width, of 5, 4, 3 and 2 columns, a tape takes two cells
// where its name does not fit one, one that does not fit at a row's end
// starts the next row, and two rows hold what they hold in that order:
// every part type, or as many as fit with the key, whose N is the rest.
// The dialog the key opens holds every part type under the headings of a
// part vocabulary on the catalog's spine (the one the catalog files its
// fitments under, else the first loaded): under each entry that maps to
// it, so under two entries for a type two map to, and under "Other" for
// one none does; by name with none loaded. Without a script nothing on
// the page is dead: the filters and the recent part types are hidden
// until plate.js shows them, and no part type is marked recent.
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
	laid := func(what, s string, want map[int][]string) {
		t.Helper()
		for _, cols := range []int{5, 4, 3, 2} {
			if got := laidOf(s, cols); !slices.Equal(got, want[cols]) {
				t.Errorf("%s at %d columns:\n%s\nwant\n%s", what, cols, strings.Join(got, "\n"), strings.Join(want[cols], "\n"))
			}
		}
	}

	s := fitSection(t, h, pickup)
	if got := stripOf(s); !slices.Equal(got, byName) {
		t.Errorf("the strip:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(byName, "\n"))
	}
	// Names of twelve characters fit a cell of 5 columns, eleven of 4 or
	// 3, twelve of 2. Brake Pads, Front does not fit the first row's end
	// at 5 columns, and starts the second, a cell left empty.
	laid("the pickup", s, map[int][]string{
		5: {"Alternator", "Battery", "Brake Caliper, Front ×2", "Brake Pads, Front ×2", "Brake Pads, Rear ×2", "13 more part types"},
		4: {"Alternator", "Battery", "Brake Caliper, Front ×2", "Brake Pads, Front ×2", "14 more part types"},
		3: {"Alternator", "Battery", "Brake Caliper, Front ×2", "15 more part types"},
		2: {"Alternator", "Battery", "16 more part types"},
	})
	if !strings.Contains(s, ` data-fit="{&#34;tape&#34;:46,`) {
		t.Errorf("the strip lacks the measure plate.js packs it by\n%s", s)
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
		`<li class="ptkey k5 k4 k3 k2"><button type="button" id="more-types" popovertarget="part-types"><span class="kn c5">13 more<span class="visually-hidden"> part types</span></span>`,
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
	laid("Water Pump pressed", s, map[int][]string{
		5: {"Water Pump", "Alternator", "Battery", "Brake Caliper, Front ×2", "Brake Pads, Front ×2", "Brake Pads, Rear ×2", "12 more part types"},
		4: {"Water Pump", "Alternator", "Battery", "Brake Caliper, Front ×2", "14 more part types"},
		3: {"Water Pump", "Alternator", "Battery", "Brake Caliper, Front ×2", "14 more part types"},
		2: {"Water Pump", "Alternator", "Battery", "15 more part types"},
	})

	// The Silverado's six fit two rows of 5 and 4 columns, where no key
	// is drawn; its long name takes two cells, and still ends in "…".
	s = fitSection(t, h, "/?from.vin=1GCVKREC6EZ123456")
	long := "Battery Cable, Positive, with Fusible Link and Terminal ×2"
	all := []string{"Alternator", "Battery", long, "Radiator", "Starter Motor ×2", "Water Pump"}
	laid("the Silverado", s, map[int][]string{
		5: all, 4: all,
		3: {"Alternator", "Battery", long, "3 more part types"},
		2: {"Alternator", "Battery", "4 more part types"},
	})
	if !strings.Contains(s, `<li class="ptkey k3 k2">`) {
		t.Errorf("the Silverado's key is drawn but at 3 and 2 columns\n%s", s)
	}

	// A tape of two cells starts the next row when one is left: the key
	// takes that cell, and its N may be 1.
	for _, c := range []struct {
		spans            []int
		cols, lead, want int
	}{
		{[]int{1, 1, 1, 1, 2}, 3, 0, 5},
		{[]int{1, 1, 1, 1, 1, 2}, 3, 0, 5},
		{[]int{2, 2, 1}, 3, 0, 3},
		{[]int{2, 2, 2}, 3, 0, 2},
		{[]int{1, 2, 1, 1, 1}, 3, 1, 2},
		{[]int{2, 1, 2}, 2, 0, 2},
	} {
		if got := packRows(c.spans, c.cols, c.lead); got != c.want {
			t.Errorf("%v in two rows of %d after %d: %d; want %d", c.spans, c.cols, c.lead, got, c.want)
		}
	}
	if tapeSpan("Brake Rotor, Front", 1, false, stripFit.Grids[0]) != 2 || tapeSpan("Alternator", 3, false, stripFit.Grids[0]) != 1 || tapeSpan("Water Pump", 1, true, stripFit.Grids[3]) != 2 {
		t.Errorf("a long name takes one cell, a short one two, or a Recent stamp is not counted")
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
