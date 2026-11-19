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
// types the loaded catalog has for the VIN's vehicle, by name, each with
// its count of candidates, and the candidates of the one pressed. A part
// type is pressed by a link to the page with it as the second parameter,
// part.id, the fit request's own part by its id on the catalog's spine;
// nothing is pressed without it, and no candidate is asked for. So the
// card is a URL too, works without a script, and plate.js only swaps the
// answer in place. A candidate is shown as the catalog has it, its option
// text verbatim, with marks for what nitrovin did and did not check
// against the decode (candidateMarks); nothing is chosen for the
// visitor. Where there is nothing to list, the part types' place says
// why, and where it can, offers a way on: the Vocabularies dialog, or the
// catalog's lines (entries).
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
	// Part and After are the page's own part.id and after, which plate.js
	// tells its history by.
	Part, After string
	// Model and Year are what the strip says was asked: the catalog's
	// vehicle the decode reached, and the year, "any year" without one.
	Model, Year string
	// Types are the vehicle's part types, by name.
	Types []partTypeLink
	// Pressed is the pressed part type's name, "" with none, and Page
	// the page's own URL with it pressed.
	Pressed, Page string
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
}

// wayOn is a link on from an empty place.
type wayOn struct {
	Text, Href string
}

// partTypeLink is a part type of the strip: its name and count, and the
// page with it pressed, or, for the pressed one, with none.
type partTypeLink struct {
	Name, Href string
	Count      int
	Pressed    bool
}

// candidateTag is one candidate as its tag shows it.
type candidateTag struct {
	Number, Side, Years, Note string
	// Options are its applications' option texts, verbatim, each once.
	Options []string
	Marks   []fitMark
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
	c := fitCard{Catalog: h.loadedCatalog(info).Name}
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
// types, and part's candidates, from their page after, when part is
// pressed. cat is what is loaded, which names the catalog's vehicles'
// fields. Its answers' JSON goes in res.
func (h *Handler) fitVIN(ctx context.Context, d *diagram, dec *contract.VINDecode, cat []contract.CatalogEntry, elements contract.Elements, part, after string, res *result) error {
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
	slices.SortStableFunc(types, func(a, b contract.FitPartType) int {
		return strings.Compare(strings.ToLower(partTypeName(a)), strings.ToLower(partTypeName(b)))
	})
	names := map[string]string{}
	pressed := false
	for _, t := range types {
		names[t.Type] = partTypeName(t)
		l := partTypeLink{Name: partTypeName(t), Count: t.Count, Href: fitHref(d.VIN, t.Type, "")}
		if t.Type == part {
			l.Pressed, l.Href, c.Pressed, pressed = true, vinHref(d.VIN), l.Name, true
		}
		c.Types = append(c.Types, l)
	}
	switch {
	case part == "":
		c.Invite = true
		return nil
	case !pressed:
		c.NoneFit = "Nothing " + c.Catalog + " lists as " + strconv.Quote(part) + " fits this vehicle."
		return nil
	}
	c.Page = fitHref(d.VIN, part, after)
	limit := candidatesPage
	if h.candidates != 0 {
		limit = h.candidates
	}
	req = contract.FitRequest{From: contract.FitFrom{VIN: d.VIN}, Part: &contract.FitPart{Spine: fit.Catalog.Spine, ID: part}, Limit: limit, After: after}
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
	if fit.Next != "" {
		c.Next = fitHref(d.VIN, part, fit.Next)
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
// neither, "Catalog text only" when it has option text to read and "No
// conditions" when it has none. A confirmed model year alone checks
// nothing a person needs told: every candidate's years held, or it would
// not be one. The tag shows the union of its applications' marks, once
// each kind, a kind's elements merged, the weaker first: Not checked,
// Catalog text only, No conditions, then Checked. So one application
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
		out = append(out, fitMark{Kind: "text", Text: "Catalog text only"})
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
