package ui

import (
	"cmp"
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// The plate's part-number mode (#325). The page answers a second
// question: what a part number fits, and whether there is another. A
// switch on the plate, two links, turns it from a VIN to a part type and
// a number, the fit request's own fitment key, and the page answers with
// one fit from that key: the part, how it was reached (directly, or
// through the retired number asked for), every vehicle it fits a page at
// a time, and its stock. The mode is the URL's: its parameters are the
// key's own (from.fitment.type, from.fitment.number and
// from.fitment.side), as the VIN's is from.vin, and a URL names one or
// the other. So a link reproduces the mode and the part, the page opens
// on a VIN, and nothing is remembered between visits.
//
// The part type is chosen in the part types dialog (#324), which here
// lists every part type of the catalog: the entries of its spine, under
// the headings of a part vocabulary as on a VIN's page. The number is
// sent by the plate's own submit, as a VIN is; plate.js asks again as it
// is typed. The part's tag carries its stock strip, which opens its shelf
// and yard rows under it as a candidate's does (fitstock.go), named in
// the URL the same way. A key with no side that names a sided part is
// asked again on each side: one found is the answer, several are links.

const (
	// fromTypeParam, fromNumberParam and fromSideParam are the page's
	// part-number mode: the fit request's fitment key, as GET /v1/fit
	// names it. Any of them puts the page in the mode.
	fromTypeParam   = "from.fitment.type"
	fromNumberParam = "from.fitment.number"
	fromSideParam   = "from.fitment.side"
	// vehiclesPage is how many vehicles a part's page lists (Handler's
	// vehicles, when not 0).
	vehiclesPage = 30
	// partTitle is the page's title in the mode before it has a part.
	partTitle = "Part number"
)

// partMode reports whether q asks the page in part-number mode.
func partMode(q url.Values) bool {
	return q.Has(fromTypeParam) || q.Has(fromNumberParam) || q.Has(fromSideParam)
}

// partPage is the page in part-number mode: the plate's key, the part
// types dialog, and below the plate the part and its vehicles, or what
// their place says instead.
type partPage struct {
	// Type, Number and Side are the URL's key, the number trimmed;
	// TypeName the type's name on the catalog's spine, its id when the
	// spine has none, "" with none chosen.
	Type, Number, Side, TypeName string
	// Query is the query the page answers, which plate.js tells a
	// history entry by.
	Query string
	// Catalog is the loaded catalog's name, "" with none.
	Catalog string
	// Groups are the part types dialog's, every part type of the
	// catalog's spine (partGroups), GroupedBy the part vocabulary their
	// headings come from and Spine the catalog's spine, which plate.js
	// keeps recent part types by.
	Groups           []partGroup
	GroupedBy, Spine string
	// Tag is the part, nil when the place says something else; Vehicles
	// its vehicles, a page of them of Total, named in the vehicle
	// vocabularies loaded when Named; Next the next page, "" on the last.
	Tag      *partTag
	Vehicles []vehicleRow
	Named    bool
	Total    int
	Next     string
	// After is the URL's page of vehicles, and Open and Rows the stock
	// it opens (fitstock.go's fitment.number, fitment.side, fitment.type
	// and rows).
	After string
	Open  fitKey
	Rows  string
	// Blank is what the place of the tag and the vehicles says instead,
	// Ways the links on from it, Add the control that opens the
	// Vocabularies dialog; Error a refusal.
	Blank string
	Ways  []wayOn
	Add   bool
	Error string
}

// Dialog is p's part types dialog: every part type of the catalog.
func (p *partPage) Dialog() ptDialog {
	hint := "Every part type in " + p.Catalog
	if p.GroupedBy != "" {
		hint += " · grouped by " + p.GroupedBy
	}
	return ptDialog{Hint: hint, Spine: p.Spine, Groups: p.Groups}
}

// partTag is the part as its tag shows it: its number and its part
// type's name on its tape, its side, how many vehicles it fits, its note,
// the retired number asked for that it replaces with the catalog's note
// on the retirement, its stock strip, and its rows, when open.
type partTag struct {
	Number, TypeName, Side, Fits, Note string
	Replaces, RetiredNote              string
	Strip                              stockStrip
	Panel                              *stockPanel
}

// vehicleRow is one vehicle a part fits: the catalog's name for it, its
// years, its option text verbatim, and its names in the vocabularies
// loaded, a vocabulary once with every name it has for the vehicle.
type vehicleRow struct {
	Name, Years, Options string
	Names                []vehicleNames
}

// vehicleNames are a vehicle's names in one vocabulary.
type vehicleNames struct {
	Vocabulary string
	Names      []string
}

// partKey is the plate's key: a part type, a number and a side.
type partKey struct {
	Type, Number, Side string
}

// partHref is the page in part-number mode with k on the plate, its
// vehicles' page after, the stock of open shown (none when its number is
// ""), and every row of it with rows: the key first, as the plate's form
// sends it, the type omitted when none is chosen and the number always
// there, so the URL stays in the mode.
func partHref(k partKey, after string, open fitKey, rows string) string {
	s := homePath + "?"
	if k.Type != "" {
		s += fromTypeParam + "=" + url.QueryEscape(k.Type) + "&"
	}
	s += fromNumberParam + "=" + url.QueryEscape(k.Number)
	if k.Side != "" {
		s += "&" + fromSideParam + "=" + url.QueryEscape(k.Side)
	}
	if after != "" {
		s += "&" + afterParam + "=" + url.QueryEscape(after)
	}
	if open.Number != "" {
		if open.Type != k.Type {
			s += "&" + typeParam + "=" + url.QueryEscape(open.Type)
		}
		s += "&" + numberParam + "=" + url.QueryEscape(open.Number)
		if open.Side != "" {
			s += "&" + sideParam + "=" + url.QueryEscape(open.Side)
		}
	}
	if rows != "" {
		s += "&" + rowsParam + "=" + rows
	}
	return s
}

// partModeHref is the page in part-number mode with nothing on the plate:
// the switch's link to it.
const partModeHref = homePath + "?" + fromNumberParam + "="

// vehicleVocabularies are the vehicle vocabularies of cat, what is
// loaded, by name: those on vPIC's spine, which name a vehicle.
func vehicleVocabularies(cat []contract.CatalogEntry) []string {
	var out []string
	for _, c := range cat {
		if c.Kind == string(vocab.KindVocabulary) && c.Spine == vocab.Reserved {
			out = append(out, c.Name)
		}
	}
	slices.Sort(out)
	return out
}

// partVocabularies are the vehicle vocabularies a part's vehicles are
// named in: every one loaded, or, in the browser while the server holds
// the catalog, the server's alone, as the server's fit answers and it
// knows no other. A browser's own vocabulary then names no vehicle of a
// part.
func (h *Handler) partVocabularies(cat []contract.CatalogEntry) []string {
	to := vehicleVocabularies(cat)
	if h.lib == nil || h.lib.ServerCatalog().Name == "" {
		return to
	}
	server := h.lib.Server()
	return slices.DeleteFunc(to, func(name string) bool {
		return !slices.ContainsFunc(server, func(f StoredFile) bool { return f.Name == name })
	})
}

// mixedHome is GET / for a link that names a VIN and a part number, which
// a page does not: the VIN's plate, empty, and a link to each.
func (h *Handler) mixedHome(q url.Values, info contract.Info, page *homePage) (view, error) {
	vin := cleanVIN(q.Get("from.vin"))
	k := partKey{Type: q.Get(fromTypeParam), Number: strings.TrimSpace(q.Get(fromNumberParam)), Side: q.Get(fromSideParam)}
	part := k.Number
	if k.Type != "" {
		name := k.Type
		if loaded := h.loadedCatalog(info); loaded.Name != "" {
			names, err := h.spineNames(loaded.Spine)
			if err != nil {
				return view{}, err
			}
			name = typeName(names, k.Type)
		}
		part = strings.TrimSpace(name + " " + k.Number)
	}
	d := newDiagram("")
	d.Fit = h.idleFit(info)
	d.Error = "This link names a VIN, " + vin + ", and a part number, " + part + "."
	d.Ways = []wayOn{{Text: "Open the VIN", Href: vinHref(vin)}, {Text: "Open the part number", Href: partHref(k, "", fitKey{}, "")}}
	page.Diagram = d
	return view{title: homeTitle, template: "home", data: page, target: mainTarget, vocabDialog: true, status: http.StatusBadRequest}, nil
}

// partHome is GET / in part-number mode, q its query: the plate with the
// key the URL names, and below it the part, or why there is none.
func (h *Handler) partHome(r *http.Request, q url.Values, info contract.Info, page *homePage) (view, error) {
	p := &partPage{Type: q.Get(fromTypeParam), Number: strings.TrimSpace(q.Get(fromNumberParam)), Side: q.Get(fromSideParam), Query: r.URL.RawQuery,
		After: q.Get(afterParam), Open: fitKey{Type: q.Get(typeParam), Number: q.Get(numberParam), Side: q.Get(sideParam)}, Rows: q.Get(rowsParam)}
	page.Part = p
	v := view{title: partTitle, template: "home", data: page, target: mainTarget, vocabDialog: true}
	loaded := h.loadedCatalog(info)
	if loaded.Name == "" {
		p.Blank, p.Add = "No catalog loaded.", true
		return v, nil
	}
	p.Catalog, p.Spine = loaded.Name, loaded.Spine
	cat, err := h.catalog()
	if err != nil {
		return view{}, err
	}
	names, err := h.spineNames(loaded.Spine)
	if err != nil {
		return view{}, err
	}
	p.TypeName = p.Type
	if n, ok := names[p.Type]; ok {
		p.TypeName = n
	}
	var types []partTypeLink
	for id, name := range names {
		types = append(types, partTypeLink{ID: id, Name: name, Href: partHref(partKey{Type: id, Number: p.Number}, "", fitKey{}, ""), Pressed: id == p.Type})
	}
	slices.SortFunc(types, func(a, b partTypeLink) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	voc := headingVocabulary(cat, loaded.Spine, loaded.FiledUnder)
	if p.Groups, err = h.partGroups(types, voc); err != nil {
		return view{}, err
	}
	if voc != nil {
		p.GroupedBy = voc.Name
	}
	switch {
	case p.Type == "" && p.Number != "":
		p.Blank = "Choose the part type of " + p.Number + ": a number is looked up with its part type."
		return v, nil
	case p.Type == "" || p.Number == "":
		p.Blank = "Choose a part type and type its number."
		return v, nil
	}

	res := &result{URL: requestOrigin(r) + partHref(partKey{Type: p.Type, Number: p.Number, Side: p.Side}, p.After, p.Open, p.Rows)}
	page.Result = res
	kinds := h.stockKinds(info)
	limit := vehiclesPage
	if h.vehicles != 0 {
		limit = h.vehicles
	}
	to := h.partVocabularies(cat)
	p.Named = len(to) > 0
	// The request is the fit request's own query, so a key it refuses
	// (a side that is none) is refused in its words wherever it is asked.
	fq := url.Values{fromTypeParam: {p.Type}, fromNumberParam: {p.Number}, "limit": {strconv.Itoa(limit)}}
	if len(to) > 0 {
		fq["to"] = to
	}
	if len(kinds) > 0 {
		fq["stock"] = kinds
	}
	if p.Side != "" {
		fq.Set(fromSideParam, p.Side)
	}
	if p.After != "" {
		fq.Set(afterParam, p.After)
	}
	refuse := func(err error) (view, error) {
		status, words, fault := refusal(err)
		if fault != nil {
			return view{}, fault
		}
		v.status, p.Error = status, words
		return v, nil
	}
	req, err := contract.ParseFitQuery(fq)
	if err != nil {
		return refuse(err)
	}
	fit, raw, sides, err := h.partFit(r.Context(), &req)
	if err != nil {
		return refuse(err)
	}
	if len(sides) > 1 {
		p.Blank = p.Catalog + " has " + p.TypeName + " " + p.Number + " on more than one side."
		for _, s := range sides {
			p.Ways = append(p.Ways, wayOn{Text: "The " + s + " one", Href: partHref(partKey{Type: p.Type, Number: p.Number, Side: s}, "", fitKey{}, "")})
		}
		return v, nil
	}
	// The key the answer came from: a side the retry found stays on
	// every link, so a later page asks with it and needs no retry.
	key := partKey{Type: p.Type, Number: p.Number, Side: req.From.Fitment.Side}
	if err := h.fitJSON(res, "part-json", "The part", "the part's JSON", fit, raw); err != nil {
		return view{}, err
	}
	switch {
	case fit.Status == contract.FitNoCatalog:
		p.Catalog, p.Groups, p.Blank, p.Add = "", nil, "No catalog loaded.", true
		return v, nil
	case fit.Fitment == nil:
		p.Blank = p.Catalog + " has no " + p.TypeName + " numbered " + p.Number + "."
		return v, nil
	case fit.Status == contract.FitNoFitment:
		p.Blank = p.Catalog + " has " + typeName(names, fit.Fitment.Type) + " " + fit.Fitment.Number + ", and lists no vehicle for it."
		return v, nil
	}
	tag := &partTag{Number: fit.Fitment.Number, TypeName: typeName(names, fit.Fitment.Type), Side: fit.Fitment.Side, Note: fit.Fitment.Note,
		Fits: strconv.Itoa(fit.Total) + " " + plural(fit.Total, "vehicle", "vehicles") + " in " + p.Catalog}
	if fit.Reached == contract.ReachedRetired && fit.Retired != nil {
		tag.Replaces, tag.RetiredNote = fit.Retired.Number, fit.Retired.Note
	}
	v.title = tag.Number + " · " + tag.TypeName
	p.Tag, p.Total = tag, fit.Total
	if err := h.partStock(r.Context(), p, key, fit, kinds, h.stockDeclared(info), res); err != nil {
		return view{}, err
	}
	fields := map[string][]string{}
	for _, c := range cat {
		fields[c.Name] = c.Fields
	}
	for _, fv := range fit.Vehicles {
		row := vehicleRow{Name: vehicleText(fv.Vehicle, fields[fit.Catalog.Vehicles]), Years: yearsText(fv.First, fv.Last), Options: fv.Options}
		// A vPIC catalog's vehicle is named as VIN mode names one: its
		// reference, and the spine's name for it when the answer has it.
		if fv.Vehicle.Entry == nil {
			ref := contract.SpineRef{Make: fv.Vehicle.Make, Model: fv.Vehicle.Model, Ext: fv.Vehicle.Ext}
			if i := slices.IndexFunc(fv.Spine, func(s contract.SpineEntry) bool { return s.SpineRef == ref && s.Name != "" }); i >= 0 {
				row.Name += " (" + fv.Spine[i].Name + ")"
			}
		}
		for _, t := range fv.To {
			names := vehicleNames{Vocabulary: t.Vocabulary}
			for _, e := range t.Entries {
				names.Names = append(names.Names, vehicleText(contract.CatalogVehicle{Entry: e.Entry}, fields[t.Vocabulary]))
			}
			if len(names.Names) > 0 {
				row.Names = append(row.Names, names)
			}
		}
		p.Vehicles = append(p.Vehicles, row)
	}
	next, err := h.vehiclesNext(r.Context(), req, fit)
	if err != nil {
		return view{}, err
	}
	if next != "" {
		p.Next = partHref(key, next, fitKey{}, "")
	}
	return v, nil
}

// partFit answers req, from a fitment key: as asked, or, when a key with
// no side is no fitment, on each side in turn. sides are the sides that
// found a fitment then; req is left the request answered, the first of
// them its side.
func (h *Handler) partFit(ctx context.Context, req *contract.FitRequest) (fit contract.Fit, raw []byte, sides []string, err error) {
	if fit, raw, err = h.fit(ctx, *req, nil); err != nil || fit.Status != contract.FitUnknown || req.From.Fitment.Side != "" {
		return fit, raw, nil, err
	}
	unknown, unknownRaw, asked := fit, raw, *req.From.Fitment
	for _, s := range []string{contract.SideLeft, contract.SideRight, contract.SideBoth} {
		k := asked
		k.Side = s
		try := *req
		try.From.Fitment = &k
		f, b, err := h.fit(ctx, try, nil)
		if err != nil {
			return contract.Fit{}, nil, nil, err
		}
		if f.Status != contract.FitUnknown {
			if len(sides) == 0 {
				fit, raw, *req = f, b, try
			}
			sides = append(sides, s)
		}
	}
	if len(sides) == 0 {
		return unknown, unknownRaw, nil, nil
	}
	return fit, raw, sides, nil
}

// spineNames are the entries of spine, a declared spine, by id: each
// one's name, its id when it has none.
func (h *Handler) spineNames(spine string) (map[string]string, error) {
	out := map[string]string{}
	req := contract.ListRequest{Of: spine, Limit: contract.MaxListLimit}
	for {
		l, err := h.p.List(req)
		if err != nil {
			return nil, err
		}
		for _, e := range l.Entries {
			if s, ok := e.(contract.SpineEntry); ok && s.ID != "" {
				out[s.ID] = cmp.Or(s.Name, s.ID)
			}
		}
		if l.Next == "" {
			return out, nil
		}
		req.After = l.Next
	}
}

// typeName is part type id's name in names, its id when it has none.
func typeName(names map[string]string, id string) string {
	return cmp.Or(names[id], id)
}

// plural is one when n is 1, else many.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// partStock fills the part's stock strip and, when the URL opens it, its
// rows, as a candidate's (stockColumns), key the key the part was
// answered from, kinds and declared the stock loaded.
func (h *Handler) partStock(ctx context.Context, p *partPage, key partKey, fit contract.Fit, kinds []string, declared map[string][]string, res *result) error {
	cand := contract.FitCandidate{Fitment: *fit.Fitment, Shelf: fit.Shelf, Yard: fit.Yard}
	own := candidateKey(cand)
	s := newStockStrip(cand, kinds)
	s.ID = "stk-part"
	want := fitKey{Type: cmp.Or(p.Open.Type, key.Type), Number: p.Open.Number, Side: p.Open.Side}
	open := s.Kind == "counts" && p.Open.Number != "" && want == own
	p.Tag.Strip = s
	if !open {
		p.Tag.Strip.Href = partHref(key, p.After, own, "")
		p.Open, p.Rows = fitKey{}, ""
		return nil
	}
	p.Tag.Strip.Open, p.Tag.Strip.Href = true, partHref(key, p.After, fitKey{}, "")
	if p.Rows != rowsAll {
		p.Rows = ""
	}
	cols, err := h.stockColumns(ctx, own, cand, kinds, declared, p.Rows == rowsAll, partHref(key, p.After, own, rowsAll), "", res)
	if err != nil {
		return err
	}
	// The part's number is the page's second heading, a candidate's a
	// third: a kind of its stock is one under it.
	for i := range cols {
		cols[i].Level = 3
	}
	p.Tag.Panel = &stockPanel{Number: own.Number, Columns: cols}
	return nil
}

// vehiclesNext is the vehicles' next page after fit, the answer to req:
// fit's own Next, but with stock asked for, the answer's Next stays while
// a stock list has more, so the catalog alone, asked again with no stock,
// says whether more vehicles follow.
func (h *Handler) vehiclesNext(ctx context.Context, req contract.FitRequest, fit contract.Fit) (string, error) {
	if fit.Next == "" || len(req.Stock) == 0 {
		return fit.Next, nil
	}
	if len(fit.Vehicles) < req.Limit {
		return "", nil
	}
	req.Stock = nil
	again, _, err := h.fit(ctx, req, nil)
	return again.Next, err
}
