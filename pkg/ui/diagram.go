package ui

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/joshuasprow/nitrovin/pkg/contract"
)

// The page (docs/research/vin-page.md has its references) draws the VIN as
// its diagram. The plate is 17 cells under six brackets, the standard
// groups of a VIN; each bracket leads to a card of what its characters
// decode to, and a derived card beneath them, "Also known as", holds the
// vehicle's names in every vocabulary loaded. The server draws all of it
// for whatever is typed so far, a partial VIN included, so the page with
// JavaScript (plate.js, which asks as the visitor types) and without it
// (the form) show the same thing.

// vinLength is a VIN's length, and the plate's cells.
const vinLength = 17

// The groups' keys: each names its colour, its bracket and its card.
const (
	groupWMI    = "wmi"
	groupVDS    = "vds"
	groupCheck  = "check"
	groupYear   = "year"
	groupPlant  = "plant"
	groupSerial = "serial"
)

// vinGroups are the six groups, in order: positions From to To, 1-based.
var vinGroups = []struct {
	key, label, title string
	from, to          int
}{
	{groupWMI, "Maker", "Maker", 1, 3},
	{groupVDS, "Vehicle", "Vehicle", 4, 8},
	{groupCheck, "Check", "Check digit", 9, 9},
	{groupYear, "Year", "Model year", 10, 10},
	{groupPlant, "Plant", "Plant", 11, 11},
	{groupSerial, "Serial", "Serial", 12, 17},
}

// vPIC's element ids the cards read, beside the summary's (decode.go).
const (
	elementBodyClass    = 5
	elementFuelType     = 24
	elementManufacturer = 27
	elementVehicleType  = 39
	elementPlantCountry = 75
)

// cell is one character's cell on the plate: its character, "" before it
// is typed, the group it is of, and whether a VIN can hold it.
type cell struct {
	Char, Group string
	Bad         bool
}

// cardRow is one line of a card: what it is and its value, a country's
// with its flag.
type cardRow struct {
	Label, Value, Flag string
}

// checkStep is one character of the check digit's sum.
type checkStep struct {
	Char           string
	Value, Weight  int
	Product        int
	IsCheckPostion bool
}

// checkSum is the check digit's arithmetic: each character's value times
// its position's weight, their sum, its remainder by 11 (10 is X), and the
// digit the VIN carries.
type checkSum struct {
	Steps          []checkStep
	Sum, Remainder int
	Want, Got      string
	Valid          bool
}

// yearCode is one code of the year strip: the character and its year.
type yearCode struct {
	Char    string
	Year    int
	Current bool
}

// vinGroup is one bracket and its card. Done says its characters are all
// typed: until then the card is there, empty.
type vinGroup struct {
	Key, Label, Title string
	From, To          int
	Done              bool
	Rows              []cardRow
	// Text is the card's one large value when it has no rows: the year,
	// the serial, the check digit's verdict.
	Text string
	// Bad marks a check digit that does not match.
	Bad   bool
	Check *checkSum
	Years []yearCode
}

// akaTag is one name of the vehicle in a vocabulary, as a small plate of
// its own: the vocabulary stamped on it, then its words under the brackets
// of the VIN's groups they answer to, its make under the maker's, the rest
// of the entry under the vehicle's, the years its mapping holds for under
// the year's. Doubt is drawn where it applies: Rule (the mapping was
// matched by rule, not listed) dashes the lines into the make and the
// model and the marks under them, YearUnsure and ModelUnsure (the model
// year, or a qualifier, could not be checked) dash that bracket and mark.
// Each segment's note says so in words, and Details says how the name was
// reached.
//
// In a stack of several, the top plate's stamp is the stack's switch: Fan
// is the id of the checkbox it is the label of, and Count the names in
// the stack. Under marks the plates below the top one, which carry no
// stamp of their own.
//
// Check, on the top plate alone, is the id of the first years in the stack
// that are unsure, "" when every name's are sure: where the label's "Check
// years" link goes. Those years carry it (YearsID).
type akaTag struct {
	ID, Vocabulary                string
	Make, Model, Years            string
	Rule, YearUnsure, ModelUnsure bool
	MakeNote, ModelNote, YearNote string
	Details                       string
	Fan                           string
	Count                         int
	Under                         bool
	Check                         string
}

// YearsID is the id of the tag's years, which only unsure years carry: a
// "Check years" link's target.
func (t akaTag) YearsID() string {
	if !t.YearUnsure {
		return ""
	}
	return t.ID + "-years"
}

// akaRow is one vocabulary's stack of "Also known as": its names, the
// first on top and the rest under it until the stack is fanned out. A
// vocabulary with no name keeps its place. Fan is its switch's id, "" for
// a stack of one. Late says where the stack falls past the block's cap
// (akaShown): "third" for the one a phone keeps back, "rest" for those
// every width does, "" for a stack always shown.
type akaRow struct {
	Vocabulary, Fan string
	Names           []akaTag
	Late            string
}

// Depth is how many plates show under the top one of a closed stack: the
// rest, three at most.
func (r akaRow) Depth() int { return min(max(len(r.Names)-1, 0), 3) }

// akaCard is "Also known as", the first derived card: the tags of every
// vocabulary loaded, hung from the maker's, the vehicle's and the year's
// lines. Waiting says the model is not known yet; None that no vocabulary
// is loaded.
type akaCard struct {
	Rows          []akaRow
	Waiting, None bool
}

// The block's cap: the vocabularies "Also known as" shows before its
// "more" switch, on a phone (34rem and narrower) and on anything wider.
const (
	akaShownPhone = 2
	akaShown      = 3
)

// Capped says the block has vocabularies to keep back, at some width.
func (c akaCard) Capped() bool { return len(c.Rows) > akaShownPhone }

// More is the switch's words where three vocabularies show, "" when no
// more are loaded; MorePhone its words where two do.
func (c akaCard) More() string      { return moreText(len(c.Rows) - akaShown) }
func (c akaCard) MorePhone() string { return moreText(len(c.Rows) - akaShownPhone) }

// moreText is "N more vocabularies", "" for none.
func moreText(n int) string {
	switch {
	case n <= 0:
		return ""
	case n == 1:
		return "1 more vocabulary"
	}
	return strconv.Itoa(n) + " more vocabularies"
}

// diagram is the plate and everything hanging off it.
type diagram struct {
	// VIN is what was typed, cleaned (cleanVIN).
	VIN    string
	Cells  []cell
	Groups []vinGroup
	Names  akaCard
	// Fit is the parts that fit, under the names (fit.go).
	Fit      fitCard
	Complete bool
	// Label names the vehicle, for the title and a recent chip: "2003
	// HONDA Accord".
	Label string
	// Problems are the decode's error texts for a complete VIN, and Fix
	// the page with the VIN the decoder suggests instead, "" for none.
	Problems  []string
	Fix, FixV string
	// Error is a refusal that names no cell.
	Error string
}

// cleanVIN is s as the plate takes it: letters and digits alone, upper
// case, no more than a VIN's length. Letters are never swapped: an O stays
// an O, and its cell says so.
func cleanVIN(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if (r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') && b.Len() < vinLength {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// badVINChar reports whether c is a letter no VIN holds.
func badVINChar(c byte) bool { return c == 'I' || c == 'O' || c == 'Q' }

// newDiagram is the plate for vin, cleaned, before any decode: its cells
// and its six brackets, each done once its characters are typed.
func newDiagram(vin string) *diagram {
	d := &diagram{VIN: vin, Complete: len(vin) == vinLength, Cells: make([]cell, vinLength)}
	for _, g := range vinGroups {
		for p := g.from; p <= g.to; p++ {
			c := cell{Group: g.key}
			if p <= len(vin) {
				c.Char, c.Bad = vin[p-1:p], badVINChar(vin[p-1])
			}
			d.Cells[p-1] = c
		}
		d.Groups = append(d.Groups, vinGroup{Key: g.key, Label: g.label, Title: g.title, From: g.from, To: g.to, Done: len(vin) >= g.to})
	}
	d.Names.Waiting = true
	return d
}

// group is the diagram's group key.
func (d *diagram) group(key string) *vinGroup {
	for i := range d.Groups {
		if d.Groups[i].Key == key {
			return &d.Groups[i]
		}
	}
	panic("no group " + key)
}

// fill gives the cards what dec says of the characters typed so far.
func (d *diagram) fill(dec contract.Decode, elements contract.Elements, view *decodeView) {
	get := func(id int32) string { return strings.TrimSpace(attribute(dec, elements, id)) }
	rows := func(pairs ...string) []cardRow {
		var out []cardRow
		for i := 0; i < len(pairs); i += 2 {
			if pairs[i+1] != "" {
				out = append(out, cardRow{Label: pairs[i], Value: pairs[i+1]})
			}
		}
		return out
	}
	make_, model := get(elementMake), get(elementModel)
	if g := d.group(groupWMI); g.Done {
		g.Rows = rows("Make", make_, "Manufacturer", get(elementManufacturer), "Type", get(elementVehicleType))
	}
	if g := d.group(groupVDS); g.Done {
		trim := get(elementTrim)
		if trim == "" {
			trim = get(elementSeries)
		}
		var engine []string
		if l := get(elementDisplacement); l != "" {
			if f, err := strconv.ParseFloat(l, 64); err == nil {
				engine = append(engine, strconv.FormatFloat(f, 'f', 1, 64)+" L")
			}
		}
		if c := get(elementCylinders); c != "" {
			engine = append(engine, c+" cyl")
		}
		if f := get(elementFuelType); f != "" {
			engine = append(engine, f)
		}
		g.Rows = rows("Model", model, "Trim", trim, "Body", get(elementBodyClass), "Engine", strings.Join(engine, " · "))
	}
	year := 0
	if dec.ModelYear.Valid {
		year = int(dec.ModelYear.V)
	}
	if g := d.group(groupYear); g.Done && year != 0 {
		g.Text = strconv.Itoa(year)
		g.Years = yearStrip(d.VIN[9], year)
	}
	if g := d.group(groupPlant); g.Done {
		city := get(elementPlantCity)
		if s := get(elementPlantState); s != "" && city != "" {
			city += ", " + s
		}
		g.Rows = rows("City", city)
		if c := get(elementPlantCountry); c != "" {
			g.Rows = append(g.Rows, cardRow{Label: "Country", Value: c, Flag: flagOf(c)})
		}
	}
	if g := d.group(groupSerial); len(d.VIN) >= g.From {
		g.Text = d.VIN[g.From-1:]
	}
	if g := d.group(groupCheck); d.Complete {
		if sum := newCheckSum(d.VIN); sum != nil {
			g.Check, g.Bad = sum, !sum.Valid
			g.Text = "Checks out"
			if g.Bad {
				g.Text = "Should be " + sum.Want
			}
		}
	}
	var label []string
	if year != 0 {
		label = append(label, strconv.Itoa(year))
	}
	if make_ != "" {
		label = append(label, make_)
	}
	if model != "" {
		label = append(label, model)
	}
	d.Label = strings.Join(label, " ")
	if d.Complete {
		for _, c := range view.Codes {
			if c.Code != 0 && c.Text != "" {
				d.Problems = append(d.Problems, c.Text)
			}
		}
		if dec.CorrectedVIN.Valid && !strings.EqualFold(dec.CorrectedVIN.V, d.VIN) && !strings.ContainsAny(dec.CorrectedVIN.V, "!*") {
			d.FixV = dec.CorrectedVIN.V
		}
	}
	d.Names.Waiting = model == ""
}

// akaLess is the page's rule for which of a vocabulary's names lies on top
// of its stack, and the order of the rest. The translation's own order is
// by the names' fields, alphabetical, which says nothing of which a
// visitor means. So, in turn:
//
//  1. a name whose make is the decoded make comes before one of another
//     make (a tuner's or a coachbuilder's version of the vehicle);
//  2. a name with nothing unchecked before one with a doubt;
//  3. a listed mapping before one matched by rule;
//  4. the shorter model before the longer: the base name before its
//     variants;
//  5. and then the translation's order.
//
// It is a first judgment, to be revisited: it ranks nothing by how common
// a name is, which no file says.
func akaLess(vinMake string) func(a, b akaTag) int {
	rank := func(t akaTag) [4]int {
		var r [4]int
		if !strings.EqualFold(t.Make, vinMake) {
			r[0] = 1
		}
		if t.YearUnsure || t.ModelUnsure {
			r[1] = 1
		}
		if t.Rule {
			r[2] = 1
		}
		r[3] = len(t.Model)
		return r
	}
	return func(a, b akaTag) int {
		ra, rb := rank(a), rank(b)
		return slices.Compare(ra[:], rb[:])
	}
}

// names fills "Also known as" from tr, the VIN's translation to every
// vocabulary loaded on vPIC's spine, each vocabulary's names in the page's
// order (akaLess).
func (d *diagram) names(tr contract.Translation, cat []contract.CatalogEntry, elements contract.Elements) {
	vinMake := ""
	for _, s := range tr.Spine {
		if s.Make != 0 {
			vinMake = s.Name
		}
	}
	named := map[contract.SpineRef]string{}
	for _, s := range tr.Spine {
		named[s.SpineRef] = s.Name
	}
	name := func(r contract.SpineRef) string {
		if n := named[r]; n != "" {
			return refText(r) + " (" + n + ")"
		}
		return refText(r)
	}
	id := 0
	for _, t := range tr.To {
		row := akaRow{Vocabulary: t.Vocabulary}
		var fields []string
		if i := slices.IndexFunc(cat, func(c contract.CatalogEntry) bool { return c.Name == t.Vocabulary }); i >= 0 {
			fields = cat[i].Fields
		}
		var all []akaTag
		for _, e := range t.Entries {
			id++
			n := akaTag{ID: "aka-" + strconv.Itoa(id), Vocabulary: t.Vocabulary, Make: e.Entry["make"],
				MakeNote: "From the maker, positions 1–3", ModelNote: "From the vehicle, positions 4–8", YearNote: "The years this name holds for"}
			var model []string
			for _, f := range fields {
				if s := e.Entry[f]; f != "make" && s != "" {
					model = append(model, s)
				}
			}
			n.Model = strings.Join(model, " ")
			var details []string
			for _, via := range e.Via {
				details = append(details, viaText(via, name)+".")
				n.Rule = n.Rule || via.Origin == "derived"
			}
			switch len(e.Via) {
			case 0:
			case 1:
				n.Years = yearsText(e.Via[0].First, e.Via[0].Last)
			default:
				n.Years = strconv.Itoa(len(e.Via)) + " ranges"
			}
			if n.Years == "" {
				n.Years = "any year"
			}
			if n.Rule {
				n.MakeNote += ". Matched by rule"
				n.ModelNote += ". Matched by rule"
			}
			n.YearUnsure = e.Unconfirmed.ModelYear
			if n.YearUnsure {
				n.YearNote += ". Model year not checked"
			}
			if u := e.Unconfirmed; len(u.Elements) > 0 {
				n.ModelUnsure = true
				n.ModelNote += ". " + unconfirmedText(contract.Unconfirmed{Elements: u.Elements}, elements)
			}
			if u := unconfirmedText(e.Unconfirmed, elements); u != "" {
				details = append(details, u+".")
			}
			if e.Note != "" {
				details = append(details, "Note: "+e.Note)
			}
			n.Details = strings.Join(details, " ")
			all = append(all, n)
		}
		slices.SortStableFunc(all, akaLess(vinMake))
		if len(all) > 1 {
			row.Fan = "fan-" + t.Vocabulary
			all[0].Fan, all[0].Count = row.Fan, len(all)
			for i := range all[1:] {
				all[i+1].Under = true
			}
		}
		if i := slices.IndexFunc(all, func(n akaTag) bool { return n.YearUnsure }); i >= 0 {
			all[0].Check = all[i].YearsID()
		}
		row.Names = all
		d.Names.Rows = append(d.Names.Rows, row)
	}
	// The lines above end on the first stack, so a vocabulary with a name
	// goes before one without: a line never ends on a blank plate while a
	// name is there to take it.
	slices.SortStableFunc(d.Names.Rows, func(a, b akaRow) int {
		return cmp.Compare(min(len(b.Names), 1), min(len(a.Names), 1))
	})
	for i := range d.Names.Rows {
		switch {
		case i >= akaShown:
			d.Names.Rows[i].Late = "rest"
		case i >= akaShownPhone:
			d.Names.Rows[i].Late = "third"
		}
	}
}

// The check digit (49 CFR 565.15): each character's value, each
// position's weight.
const vinWeights = "8765432X098765432" // X is 10; position 9, the digit itself, weighs 0

func vinValue(c byte) (int, bool) {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0'), true
	case c >= 'A' && c <= 'H':
		return int(c-'A') + 1, true
	case c >= 'J' && c <= 'N':
		return int(c-'J') + 1, true
	case c == 'P':
		return 7, true
	case c == 'R':
		return 9, true
	case c >= 'S' && c <= 'Z':
		return int(c-'S') + 2, true
	}
	return 0, false
}

// newCheckSum is the check digit's arithmetic for vin, a whole VIN; nil
// when it holds a character with no value.
func newCheckSum(vin string) *checkSum {
	s := &checkSum{Got: vin[8:9]}
	for i := range vinLength {
		v, ok := vinValue(vin[i])
		if !ok {
			return nil
		}
		w := 10
		if vinWeights[i] != 'X' {
			w = int(vinWeights[i] - '0')
		}
		s.Steps = append(s.Steps, checkStep{Char: vin[i : i+1], Value: v, Weight: w, Product: v * w, IsCheckPostion: i == 8})
		s.Sum += v * w
	}
	s.Remainder = s.Sum % 11
	s.Want = strconv.Itoa(s.Remainder)
	if s.Remainder == 10 {
		s.Want = "X"
	}
	s.Valid = s.Want == s.Got
	return s
}

// yearCodes are the model year's characters in order, a cycle of 30 years.
const yearCodes = "ABCDEFGHJKLMNPRSTVWXY123456789"

// yearStrip is the year codes around c, which the decode reads as year:
// three either side.
func yearStrip(c byte, year int) []yearCode {
	i := strings.IndexByte(yearCodes, c)
	if i < 0 {
		return nil
	}
	var out []yearCode
	for off := -3; off <= 3; off++ {
		j := ((i+off)%len(yearCodes) + len(yearCodes)) % len(yearCodes)
		out = append(out, yearCode{Char: yearCodes[j : j+1], Year: year + off, Current: off == 0})
	}
	return out
}

// flags are the flags of the countries vPIC names most plants in, by its
// spelling of them. A country not here shows its name alone.
var flags = map[string]string{
	"UNITED STATES (USA)": "🇺🇸", "CANADA": "🇨🇦", "MEXICO": "🇲🇽", "JAPAN": "🇯🇵", "GERMANY": "🇩🇪",
	"SOUTH KOREA": "🇰🇷", "KOREA (SOUTH)": "🇰🇷", "UNITED KINGDOM (UK)": "🇬🇧", "ENGLAND": "🇬🇧", "ITALY": "🇮🇹", "FRANCE": "🇫🇷",
	"SWEDEN": "🇸🇪", "CHINA": "🇨🇳", "BRAZIL": "🇧🇷", "SPAIN": "🇪🇸", "AUSTRIA": "🇦🇹", "HUNGARY": "🇭🇺",
	"BELGIUM": "🇧🇪", "SLOVAKIA": "🇸🇰", "CZECH REPUBLIC": "🇨🇿", "TURKEY": "🇹🇷", "THAILAND": "🇹🇭", "INDIA": "🇮🇳",
	"AUSTRALIA": "🇦🇺", "SOUTH AFRICA": "🇿🇦", "NETHERLANDS": "🇳🇱", "FINLAND": "🇫🇮", "POLAND": "🇵🇱", "TAIWAN": "🇹🇼",
}

func flagOf(country string) string { return flags[strings.ToUpper(country)] }

// surpriseVINs are the VINs Surprise me draws from: one of each make in
// the parity corpus that decodes whole, across eras and kinds of vehicle,
// each with an invented serial and the check digit that serial needs.
var surpriseVINs = []string{
	"2HVAAAAA0GA779536", "1FCAA95A0GA042659", "1W1GAAAA1JC870323", "JT4RN50A4J0253273", "1M8GDM9A3KP771120", "9C62RR00XM0974828",
	"1HGCM82613A408525", "1GBDV13L150430307", "JM1BL1SFXA1429407", "5UXWX7C52BA575332", "3VWDX7AJ9BM211515", "1C4AD6AA4CA724551",
	"5NPE24AF0FH035854", "SAJWA3KD7F8731903", "1C4RJFAG1FC401177", "KNDJP3A58H7016879", "JH4UB1F53HA032806", "5YJ3E1EA7JF624080",
	"JSA1NJ4C2JA804306", "1N4AL3AP7JC232819", "WA1AXAF7XLA009291", "4JGFB8KA0MA967423", "JKAVN2D11RA714730", "KMTGE4S13RU371707",
}
