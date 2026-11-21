package ui

import (
	"bytes"
	"context"
	"html/template"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// The page's fit card (ADR 0018, #322): under "Also known as", the part
// types the loaded catalog has for the VIN's vehicle, every page of them,
// each with its count of candidates, and the candidates of the one
// pressed. A picker names the pressed one and opens a dialog of all of
// them under the headings of a part vocabulary (partGroups), its list
// (#366); plate.js lists the ones this browser opened recently under the
// picker, and filters the list. A
// part type is pressed by a link to the page with it as the second
// parameter, part.id, the fit request's own part by its id on the
// catalog's spine; nothing is pressed without it, and no candidate is
// asked for. So the card is a URL too, works without a script, and
// plate.js only swaps the answer in place. A candidate is shown as the
// catalog has it, its option text verbatim, with marks for what nitrovin
// did and did not check against the decode (candidateMarks); nothing is
// chosen for the visitor. Where there is nothing to list, the part types'
// place says why, and where it can, offers a way on: the Vocabularies
// dialog, or the catalog's lines (entries).
//
// The server's card is its own fit's; the browser's is its own catalog's,
// over the page's decode, or the server's /v1/fit while the server holds
// the catalog (Library.pageFit).

const (
	// partParam is the page's second parameter, the pressed part type.
	partParam = "part.id"
	// afterParam is the page of the pressed part type's candidates after
	// the first, the fit's own after.
	afterParam = "after"
	// candidatesPage is how many candidates a page shows: ten rows of
	// three on a wide page (Handler's candidates, when not 0).
	candidatesPage = 30
)

// fitCard is the card: the catalog and what was asked of it, the part
// types and the candidates of the pressed one, or, where there are none,
// what the part types' place says instead.
type fitCard struct {
	// Catalog is the loaded catalog's name, "" with none loaded; Lines its
	// lines' page.
	Catalog, Lines string
	// Part and After are the page's own part.id and after, and Type,
	// Number, Side and Rows its open candidate and the panel it shows
	// every row of (fitstock.go), which plate.js tells its history by.
	Part, After        string
	Type, Number, Side string
	Rows               string
	// Model and Year are what the strip says was asked: the catalog's
	// vehicle the decode reached, and the year, "any year" without one.
	Model, Year string
	// Make is the make the catalog files the vehicle under, in its own
	// word; "" where its vehicles name none, or differ.
	Make string
	// Types are the vehicle's part types, by name. Spine is the catalog's
	// spine, which names them, and which plate.js keeps recent part types
	// by. PickCount and PickNoun are what the picker counts: the pressed
	// part type's candidates, or with none pressed the part types. Clear
	// is the page with none pressed, the key beside the picker.
	Types     []partTypeLink
	Spine     string
	PickCount int
	PickNoun  string
	Clear     string
	// Groups are the part types dialog's: under each heading, an entry of
	// GroupedBy, the part vocabulary the headings come from, the part
	// types it maps to, then "Other"; without one, a group with no
	// heading, by name. Vehicle is the vehicle the dialog says it lists.
	Groups    []partGroup
	GroupedBy string
	Vehicle   string
	// filedUnder is the loaded catalog's filed-under part vocabulary, ""
	// when it names none.
	filedUnder string
	// Pressed is the pressed part type's name, "" with none, and Page
	// the page's own URL with it pressed.
	Pressed, Page string
	// Facts are what the decode says of the vehicle that a catalog's
	// option text names, on the vehicle's block, for the reader to match a
	// candidate's options against (#366); nil where the decode has none.
	Facts *vehicleFacts
	// Tags are its candidates, a page of them, of Total in every page;
	// Next is the next page's URL, "" on the last.
	Tags  []candidateTag
	Total int
	Next  string
	// Blank is what the part types' place says in their stead, "" when
	// it lists them; Ways the links on from it, none for none; Add the
	// control that opens the Vocabularies dialog.
	Blank string
	Ways  []wayOn
	Add   bool
	// Invite asks for a press: nothing is pressed. NoneFit says the
	// pressed part type has no candidate.
	Invite  bool
	NoneFit string
	// NoNumber are the shelf rows with no number for the pressed part
	// type and the vehicle, nil for none, under their strip, NoNumberName.
	NoNumber     *stockColumn
	NoNumberName string
}

// wayOn is a link on from an empty place.
type wayOn struct {
	Text, Href string
}

// partTypeLink is a part type of the dialog: its spine id, its name and
// count, and the page with it pressed, which for the pressed one is the
// page itself: pressing it again changes nothing (#366).
type partTypeLink struct {
	ID, Name, Href string
	Count          int
	Pressed        bool
}

// partGroup is one heading of the part types dialog and its part types,
// by name; Heading is "" in a dialog without headings.
type partGroup struct {
	Heading string
	Types   []partTypeLink
}

// ptDialog is the part types dialog: the line saying what it lists, its
// groups, and the spine plate.js keeps its recent part types by, "" when
// the strip beside it says (data-spine).
type ptDialog struct {
	Hint, Spine string
	Groups      []partGroup
}

// Dialog is c's part types dialog: the vehicle's part types.
func (c fitCard) Dialog() ptDialog {
	hint := c.Vehicle + " in " + c.Catalog
	if c.GroupedBy != "" {
		hint += " · grouped by " + c.GroupedBy
	}
	return ptDialog{Hint: hint, Groups: c.Groups}
}

// vehicleFacts are the vehicle as the decode has it, for matching a
// candidate's option text by eye: the Vehicle card's rows but the model,
// which the strip names, and the VIN's characters 4 to 8, the ones a
// catalog cites by position ("VIN G, 8th digit"). The page reads no
// option text: it puts the two side by side.
type vehicleFacts struct {
	Rows  []cardRow
	Chars []vinChar
}

// vinChar is one character of the VIN and its position, from 1.
type vinChar struct {
	Char string
	Pos  int
}

// newVehicleFacts are d's facts, nil when it has neither a row nor the
// Vehicle group's five characters.
func newVehicleFacts(d *diagram) *vehicleFacts {
	var f vehicleFacts
	for _, r := range d.group(groupVDS).Rows {
		if r.Label != "Model" {
			f.Rows = append(f.Rows, r)
		}
	}
	if vin := []rune(d.VIN); len(vin) >= 8 {
		for i := 3; i < 8; i++ {
			f.Chars = append(f.Chars, vinChar{Char: string(vin[i]), Pos: i + 1})
		}
	}
	if len(f.Rows) == 0 && len(f.Chars) == 0 {
		return nil
	}
	return &f
}

// candidateTag is one candidate as its tag shows it.
type candidateTag struct {
	Number, Side, Years, Note string
	// Options are its applications' option texts, verbatim, each once.
	Options []string
	Marks   []fitMark
	// Strip is its stock strip; Panel its rows, when it is open.
	Strip stockStrip
	Panel *stockPanel
}

// fitMark is one mark on a tag: Kind is "checked", "unchecked", "text",
// "none" or "redirect"; Text says it.
type fitMark struct {
	Kind, Text string
}

// fitHref is the page with vin on the plate and part pressed, from
// candidates' page after: the VIN first, as the plate's own URL has it.
func fitHref(vin, part, after string) string {
	s := vinHref(vin)
	if part != "" {
		s += "&" + partParam + "=" + url.QueryEscape(part)
	}
	if after != "" {
		s += "&" + afterParam + "=" + url.QueryEscape(after)
	}
	return s
}

// loadedCatalog is the loaded catalog as /v1/info lists it, a zero Name
// with none: in the browser, the server's while it holds one, else the
// browser's.
func (h *Handler) loadedCatalog(info contract.Info) contract.VocabularyFile {
	if h.lib != nil && h.lib.ServerCatalog().Name != "" {
		return h.lib.ServerCatalog()
	}
	for _, v := range info.Vocabularies {
		if v.Kind == string(vocab.KindCatalog) {
			return v
		}
	}
	return contract.VocabularyFile{}
}

// linesHref is the catalog's lines page, kept to where's fields.
func linesHref(catalog string, where url.Values) string {
	s := vocabulariesPath + "/" + url.PathEscape(catalog)
	if len(where) > 0 {
		s += "?" + where.Encode()
	}
	return s
}

// idleFit is the card before a model is known: none loaded, or waiting.
func (h *Handler) idleFit(info contract.Info) fitCard {
	loaded := h.loadedCatalog(info)
	c := fitCard{Catalog: loaded.Name, filedUnder: loaded.FiledUnder}
	if c.Catalog == "" {
		c.Blank, c.Add = "No catalog loaded.", true
		return c
	}
	c.Lines = linesHref(c.Catalog, nil)
	c.Blank = "Part types turn up once the model is known."
	return c
}

// fit answers req: on the server its Pages; in the browser its own
// catalog over dec, the page's decode, or the server's, with raw the
// server's /v1/fit body as it came, nil for any other.
func (h *Handler) fit(ctx context.Context, req contract.FitRequest, dec *contract.VINDecode) (fit contract.Fit, raw []byte, err error) {
	if h.lib != nil {
		return h.lib.pageFit(ctx, req, dec)
	}
	fit, err = h.p.Fit(ctx, req)
	return fit, nil, err
}

// fitVIN fills d's card for its VIN, whose model is known: its part
// types, and, when ask presses one, its candidates from their page
// after, each with the stock loaded, the one ask opens with its rows. cat
// is what is loaded, which names the catalog's vehicles' fields. Its
// answers' JSON goes in res.
func (h *Handler) fitVIN(ctx context.Context, d *diagram, dec *contract.VINDecode, cat []contract.CatalogEntry, elements contract.Elements, ask fitAsk, res *result) error {
	part, after := ask.Part, ask.After
	c := &d.Fit
	if c.Catalog == "" {
		return nil
	}
	req := contract.FitRequest{From: contract.FitFrom{VIN: d.VIN}, Limit: contract.MaxListLimit}
	fit, raw, err := h.fit(ctx, req, dec)
	if err != nil {
		return err
	}
	var fields []string
	if fit.Catalog != nil {
		if i := slices.IndexFunc(cat, func(e contract.CatalogEntry) bool { return e.Name == fit.Catalog.Vehicles }); i >= 0 {
			fields = cat[i].Fields
		}
	}
	if err := h.fitJSON(res, "fit-json", "The part types", "the part types' JSON", fit, raw); err != nil {
		return err
	}
	types := fit.PartTypes
	for fit.Next != "" {
		req.After = fit.Next
		if fit, _, err = h.fit(ctx, req, dec); err != nil {
			return err
		}
		types = append(types, fit.PartTypes...)
	}
	year := "any year"
	if fit.Vehicle != nil && fit.Vehicle.ModelYear.Valid {
		year = strconv.Itoa(int(fit.Vehicle.ModelYear.V))
	}
	switch fit.Status {
	case contract.FitNoCatalog:
		c.Catalog, c.Lines, c.Blank, c.Add = "", "", "No catalog loaded.", true
		return nil
	case contract.FitUnknown:
		return nil
	case contract.FitNoVehicle:
		c.Blank = c.Catalog + " has no " + d.Label + "."
		c.Ways = []wayOn{{Text: "See the vehicles " + c.Catalog + " covers", Href: c.Lines}}
		return nil
	case contract.FitNoFitment:
		names := modelNames(fit.Models, fields, true)
		c.Blank = c.Catalog + " has the " + strings.Join(names, " and ") + ", and nothing in it fits " + yearWords(year) + "."
		for i, m := range fit.Models {
			c.Ways = append(c.Ways, wayOn{Text: "See " + c.Catalog + "'s " + names[i] + " lines", Href: linesHref(c.Catalog, vehicleWhere(m.Vehicle))})
		}
		return nil
	}
	c.Blank = ""
	c.Model, c.Year = strings.Join(modelNames(fit.Models, fields, false), " / "), year
	c.Make = sharedMake(fit.Models)
	vehicle := strings.Join(modelNames(fit.Models, fields, true), " / ")
	if year != "any year" {
		vehicle = year + " " + vehicle
	}
	slices.SortStableFunc(types, func(a, b contract.FitPartType) int {
		return strings.Compare(strings.ToLower(partTypeName(a)), strings.ToLower(partTypeName(b)))
	})
	names := map[string]string{}
	pressed := false
	var byName []partTypeLink
	for _, t := range types {
		names[t.Type] = partTypeName(t)
		l := partTypeLink{ID: t.Type, Name: partTypeName(t), Count: t.Count, Href: fitHref(d.VIN, t.Type, "")}
		if t.Type == part {
			l.Pressed, c.Pressed, pressed = true, l.Name, true
			c.PickCount, c.PickNoun, c.Clear = t.Count, plural(t.Count, "candidate", "candidates"), vinHref(d.VIN)
		}
		byName = append(byName, l)
	}
	c.Spine = fit.Catalog.Spine
	c.Types = byName
	c.Facts = newVehicleFacts(d)
	if !pressed {
		c.PickCount, c.PickNoun = len(byName), plural(len(byName), "part type", "part types")
	}
	voc := headingVocabulary(cat, c.Spine, c.filedUnder)
	if c.Groups, err = h.partGroups(byName, voc); err != nil {
		return err
	}
	if voc != nil {
		c.GroupedBy = voc.Name
	}
	c.Vehicle = strings.Join(modelNames(fit.Models, fields, true), " / ")
	if fit.Vehicle != nil && fit.Vehicle.ModelYear.Valid {
		c.Vehicle = year + " " + c.Vehicle
	}
	switch {
	case part == "":
		c.Invite = true
		return nil
	case !pressed:
		c.NoneFit = "Nothing " + c.Catalog + " lists as " + strconv.Quote(part) + " fits this vehicle."
		return nil
	}
	limit := candidatesPage
	if h.candidates != 0 {
		limit = h.candidates
	}
	req = contract.FitRequest{From: contract.FitFrom{VIN: d.VIN}, Part: &contract.FitPart{Spine: fit.Catalog.Spine, ID: part}, Stock: ask.Stock, Limit: limit, After: after}
	if fit, raw, err = h.fit(ctx, req, dec); err != nil {
		return err
	}
	if err := h.fitJSON(res, "candidates-json", "The candidates", "the candidates' JSON", fit, raw); err != nil {
		return err
	}
	for _, cand := range fit.Candidates {
		c.Tags = append(c.Tags, newCandidateTag(cand, elements, names))
	}
	c.Total = fit.Total
	if err := h.fitStock(ctx, c, d.VIN, ask, req, fit, dec, vehicle, res); err != nil {
		return err
	}
	next, err := h.candidatesNext(ctx, req, fit, dec)
	if err != nil {
		return err
	}
	if next != "" {
		c.Next = fitHref(d.VIN, part, next)
	}
	if len(c.Tags) == 0 {
		c.NoneFit = "Nothing " + c.Catalog + " lists as " + c.Pressed + " fits this vehicle."
	}
	return nil
}

// fitJSON adds fit's JSON to the Developers dialog: raw, the server's
// /v1/fit body, indented, when the browser asked the server, so the
// dialog shows every field the server wrote. As a translation's, it names
// no request on /v1: the browser's own catalog is the browser's alone,
// and the page is the same wherever it is drawn.
func (h *Handler) fitJSON(res *result, id, title, what string, fit contract.Fit, raw []byte) error {
	var text template.HTML
	var err error
	if raw != nil {
		text, err = indentText(bytes.TrimSpace(raw))
	} else {
		text, err = jsonText(fit)
	}
	if err != nil {
		return err
	}
	res.JSON = append(res.JSON, jsonDoc{ID: id, Title: title, What: what, Text: text})
	return nil
}

// partTypeName is a part type's name, its id when its spine entry has
// none.
func partTypeName(t contract.FitPartType) string {
	if t.Name != "" {
		return t.Name
	}
	return t.Type
}

// headingVocabulary is the part vocabulary the part types dialog's
// headings come from, of cat, what is loaded: the catalog's filed-under
// one (filedUnder) when it names one, otherwise the first loaded on the
// catalog's spine; nil with none.
func headingVocabulary(cat []contract.CatalogEntry, spine, filedUnder string) *contract.CatalogEntry {
	onSpine := func(e contract.CatalogEntry) bool {
		return e.Kind == string(vocab.KindVocabulary) && e.Subject == string(vocab.SubjectPart) && e.Spine == spine
	}
	if i := slices.IndexFunc(cat, func(e contract.CatalogEntry) bool { return e.Name == filedUnder && onSpine(e) }); filedUnder != "" && i >= 0 {
		return &cat[i]
	}
	if i := slices.IndexFunc(cat, onSpine); i >= 0 {
		return &cat[i]
	}
	return nil
}

// partGroups are the part types dialog's groups of types, which are by
// name: under each entry of voc, in its list order, the types it maps
// to, then under "Other" the ones no entry maps to. An entry mapping to
// none of them is no heading. A type two entries map to is under both.
// Without voc, one group with no heading holds them all.
func (h *Handler) partGroups(types []partTypeLink, voc *contract.CatalogEntry) ([]partGroup, error) {
	if voc == nil {
		return []partGroup{{Types: types}}, nil
	}
	var groups []partGroup
	claimed := map[string]bool{}
	req := contract.ListRequest{Of: voc.Name, Limit: contract.MaxListLimit}
	for {
		l, err := h.p.List(req)
		if err != nil {
			return nil, err
		}
		for _, e := range l.Entries {
			entry, ok := e.(contract.VocabularyEntry)
			if !ok {
				continue
			}
			ids := map[string]bool{}
			for _, m := range entry.Mappings {
				ids[m.ID] = true
			}
			var words []string
			for _, f := range voc.Fields {
				words = append(words, entry.Entry[f])
			}
			g := partGroup{Heading: strings.Join(words, " ")}
			for _, t := range types {
				if ids[t.ID] {
					g.Types = append(g.Types, t)
					claimed[t.ID] = true
				}
			}
			if len(g.Types) > 0 {
				groups = append(groups, g)
			}
		}
		if l.Next == "" {
			break
		}
		req.After = l.Next
	}
	other := partGroup{Heading: "Other"}
	for _, t := range types {
		if !claimed[t.ID] {
			other.Types = append(other.Types, t)
		}
	}
	if len(other.Types) > 0 {
		groups = append(groups, other)
	}
	return groups, nil
}

// yearWords is a year as "nothing in it fits" ends: "a 2015", or "this
// vehicle" with none.
func yearWords(year string) string {
	if year == "any year" {
		return "this vehicle"
	}
	return "a " + year
}

// modelNames are the catalog's vehicles a fit reached, as the catalog
// names them: a vocabulary entry's fields in the vocabulary's order (its
// fields) but its make, or all of them with make, else the vPIC name.
func modelNames(models []contract.FitModel, fields []string, withMake bool) []string {
	var out []string
	for _, m := range models {
		if m.Vehicle.Entry == nil {
			out = append(out, m.Name)
			continue
		}
		var words []string
		for _, f := range fields {
			if s := m.Vehicle.Entry[f]; s != "" && (withMake || f != "make") {
				words = append(words, s)
			}
		}
		out = append(out, strings.Join(words, " "))
	}
	return out
}

// sharedMake is the make every one of the catalog's vehicles a fit
// reached is filed under, "" where one names none or two differ.
func sharedMake(models []contract.FitModel) string {
	var make string
	for i, m := range models {
		s := m.Vehicle.Entry["make"]
		if s == "" || i > 0 && s != make {
			return ""
		}
		make = s
	}
	return make
}

// vehicleWhere is a catalog's vehicle as its lines' filter: its entry's
// fields, or its vPIC make, model or extension.
func vehicleWhere(v contract.CatalogVehicle) url.Values {
	q := url.Values{}
	for f, s := range v.Entry {
		q.Set("where."+f, s)
	}
	switch {
	case v.Make != 0:
		q.Set("where.make", strconv.Itoa(int(v.Make)))
	case v.Model != 0:
		q.Set("where.model", strconv.Itoa(int(v.Model)))
	case v.Ext != "":
		q.Set("where.ext", v.Ext)
	}
	return q
}

// newCandidateTag is c as its tag shows it, names the part types' names
// by id.
func newCandidateTag(c contract.FitCandidate, elements contract.Elements, names map[string]string) candidateTag {
	t := candidateTag{Number: c.Fitment.Number, Side: c.Fitment.Side, Note: c.Fitment.Note}
	var years []string
	for _, a := range c.Applications {
		if y := yearsText(a.First, a.Last); y != "" && !slices.Contains(years, y) {
			years = append(years, y)
		}
		if a.Options != "" && !slices.Contains(t.Options, a.Options) {
			t.Options = append(t.Options, a.Options)
		}
	}
	t.Years = strings.Join(years, ", ")
	t.Marks = candidateMarks(c, elements, names)
	return t
}

// candidateMarks are what nitrovin did and did not check of c against
// the decode (ADR 0018's amendment of 2026-11-19), weakest wins: a tag
// never claims more than the weakest way the candidate fits. Each
// application earns its own marks from its confirmed and unconfirmed:
// "Checked" names the elements a decoded value met; "Not checked" those
// it could not check, and the model year when none was known; with
// neither, "Match the options yourself" when it has option text to read and "No
// conditions" when it has none. A confirmed model year alone checks
// nothing a person needs told: every candidate's years held, or it would
// not be one. The tag shows the union of its applications' marks, once
// each kind, a kind's elements merged, the weaker first: Not checked,
// Match the options yourself, No conditions, then Checked. So one application
// checked by Drive Type beside one with no condition reads "No
// conditions" and "Checked · Drive Type", never Checked alone. A
// candidate a redirect reached says so last, with the part type the
// catalog lists it under. Each mark is a stamp, a label and a list ("Not
// checked · Drive Type, Model year"), naming elements as "Also known as"
// does (elementNames), whose sentence reads "Not checked: Drive Type,
// model year" (unconfirmedText).
func candidateMarks(c contract.FitCandidate, elements contract.Elements, names map[string]string) []fitMark {
	var checked, unchecked []int32
	year, text, none := false, false, false
	for _, a := range c.Applications {
		for _, id := range a.Confirmed.Elements {
			if !slices.Contains(checked, id) {
				checked = append(checked, id)
			}
		}
		for _, id := range a.Unconfirmed.Elements {
			if !slices.Contains(unchecked, id) {
				unchecked = append(unchecked, id)
			}
		}
		year = year || a.Unconfirmed.ModelYear
		switch {
		case len(a.Confirmed.Elements) > 0 || len(a.Unconfirmed.Elements) > 0 || a.Unconfirmed.ModelYear:
		case strings.TrimSpace(a.Options) != "":
			text = true
		default:
			none = true
		}
	}
	var out []fitMark
	if len(unchecked) > 0 || year {
		words := elementNames(unchecked, elements)
		if year {
			words = append(words, "Model year")
		}
		out = append(out, fitMark{Kind: "unchecked", Text: "Not checked · " + strings.Join(words, ", ")})
	}
	if text {
		out = append(out, fitMark{Kind: "text", Text: "Match the options yourself"})
	}
	if none {
		out = append(out, fitMark{Kind: "none", Text: "No conditions"})
	}
	if len(checked) > 0 {
		out = append(out, fitMark{Kind: "checked", Text: "Checked · " + strings.Join(elementNames(checked, elements), ", ")})
	}
	if c.Reached == contract.ReachedRedirect {
		under := names[c.Fitment.Type]
		if under == "" {
			under = c.Fitment.Type
		}
		out = append(out, fitMark{Kind: "redirect", Text: "By redirect · " + under})
	}
	return out
}
