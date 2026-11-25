package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// A candidate's stock on the page (ADR 0019, #323). The candidates
// request asks for the stock of the sources loaded beside the catalog it
// answers from (stockKinds), so every candidate arrives with its shelf's
// and its yard's totals and first rows (contract.StockFirst), and its row
// ends in a strip that counts them. No other request asks for stock but
// to show more of it: the part types' asks for none, and a page with no
// part type pressed asks for no candidates. The strip is a link to the
// page with the candidate open, the URL's third parameter, named as the
// fit request's fitment key names it (fitment.number; fitment.side for a
// fitment with a side; fitment.type for one whose type is not the
// pressed part type's, which a redirect reaches), so it works without a
// script and a link reopens it; the open candidate's strip links to the
// page with none open. The open candidate's rows are the ones its
// candidate arrived with; "Show all" is the page with rows=all, which
// asks for the whole of its stock, a fit from its fitment key with the
// same stock, a page at a time.
//
// The rows are a print in columns (stockColumn): a row's id, its
// location, then the fields its sources' headers declare (#390,
// contract.StockCounts.Fields), a column each. A row's field is printed
// in the column of its name, whether or not its own source declares it
// (#400). What a row has beyond the print's columns folds under it as
// name and value (stockRow), a list or an object as how many it holds,
// so a print none of whose sources declares anything is an id and a
// location a row, and the rest folded. A declared field whose value is
// the row's id on every row is no column (#397), and a field the header
// pins (#404) is held beside the id or at the print's trailing end. The
// stylesheet sets the columns on the row the print hangs from: the id
// and the next column under the number's tape, the rest under its ink,
// and the last under its stock strip.
//
// With shelf stock asked for, the candidates' answer also carries the
// shelf rows with no number for the part type and the vehicle, which
// belong to no candidate. The answer pages them with the candidates, so
// the page lists the first page's once, after the first page's
// candidates, and their "Show all" is the page with rows=unnumbered,
// which asks for every page of them. They never keep the candidates'
// "Next page": candidatesNext asks the catalog alone whether more
// candidates follow.

const (
	// typeParam, numberParam and sideParam are the page's third
	// parameter, the candidate whose stock is open, as the fit request's
	// fitment key names it: its number; its side when it has one; and
	// its type when it is not the pressed part type.
	typeParam   = "fitment.type"
	numberParam = "fitment.number"
	sideParam   = "fitment.side"
	// rowsParam shows every row of a stock panel: rowsAll the open
	// candidate's, rowsUnnumbered the shelf rows with no number.
	rowsParam      = "rows"
	rowsAll        = "all"
	rowsUnnumbered = "unnumbered"
)

// fitAsk is what the page's URL asks of the parts that fit: the part
// type pressed and its candidates' page, the candidate whose stock is
// open (Type "" for the pressed part type), which panel shows every row,
// the kinds of stock loaded, and what each loaded source's header
// declares, by the source's name (stockDeclared).
type fitAsk struct {
	Part, After        string
	Type, Number, Side string
	Rows               string
	Stock              []string
	Declared           map[string]stockHeader
}

// stockHeader is what a stock source's header says of its print: the
// fields it declares (contract.StockCounts.Fields), a column each, and
// the one or two of them it pins (#404), held in sight where the print
// scrolls.
type stockHeader struct {
	Fields, Pin []string
}

// stockHref is the page with vin on the plate, part pressed from its
// candidates' page after, the candidate k open when it has a number, and
// every row of the panel rows names shown.
func stockHref(vin, part, after string, k fitKey, rows string) string {
	s := fitHref(vin, part, after)
	if k.Number != "" {
		if k.Type != part {
			s += "&" + typeParam + "=" + url.QueryEscape(k.Type)
		}
		s += "&" + numberParam + "=" + url.QueryEscape(k.Number)
		if k.Side != "" {
			s += "&" + sideParam + "=" + url.QueryEscape(k.Side)
		}
	}
	if rows != "" {
		s += "&" + rowsParam + "=" + rows
	}
	return s
}

// fitKey is a fitment key by its type, number and side, as the page's
// URL names a candidate.
type fitKey struct {
	Type, Number, Side string
}

// key is ask's open candidate's fitment key, its type the pressed part
// type's unless the URL names another.
func (ask fitAsk) key() fitKey {
	k := fitKey{Type: ask.Type, Number: ask.Number, Side: ask.Side}
	if k.Type == "" {
		k.Type = ask.Part
	}
	return k
}

// candidateKey is c's fitment key.
func candidateKey(c contract.FitCandidate) fitKey {
	return fitKey{Type: c.Fitment.Type, Number: c.Fitment.Number, Side: c.Fitment.Side}
}

// stockSources are the stock sources loaded beside the catalog the page's
// fit answers from: the server's own; in the browser, the server's while
// it holds the catalog, else the browser's own, which info lists.
func (h *Handler) stockSources(info contract.Info) []contract.VocabularyFile {
	files := info.Vocabularies
	if h.lib != nil && h.lib.ServerCatalog().Name != "" {
		files = h.lib.ServerStock()
	}
	var out []contract.VocabularyFile
	for _, v := range files {
		if v.Kind == string(vocab.KindStock) && v.StockCounts != nil {
			out = append(out, v)
		}
	}
	return out
}

// stockKinds are the kinds of stock those sources hold, shelf first.
func (h *Handler) stockKinds(info contract.Info) []string {
	sources := h.stockSources(info)
	var out []string
	for _, kind := range []string{contract.StockShelf, contract.StockYard} {
		if slices.ContainsFunc(sources, func(v contract.VocabularyFile) bool { return v.Source == kind }) {
			out = append(out, kind)
		}
	}
	return out
}

// stockDeclared are each of those sources' headers, by the source's
// name; a source that declares no field has no entry, and pins none.
func (h *Handler) stockDeclared(info contract.Info) map[string]stockHeader {
	out := map[string]stockHeader{}
	for _, v := range h.stockSources(info) {
		if len(v.Fields) > 0 {
			out[v.Name] = stockHeader{Fields: v.Fields, Pin: v.Pin}
		}
	}
	return out
}

// stockStrip is the strip at a row's trailing end. Kind is "counts", a
// link that opens the candidate's rows (Href) or, Open, closes them;
// "none", stock loaded and none for it; or "absent", no stock loaded,
// which opens the Vocabularies dialog. Text is its words, which a
// screen reader is told with Number; Counts are what is drawn, a cell
// for each kind of stock.
type stockStrip struct {
	Kind, Text, Href, ID, Number string
	Open                         bool
	Counts                       []stockCount
}

// stockCount is one kind's cell of a strip: N its total, large, over
// Kind, its name; Down says none of its sources answered, and N is "–".
type stockCount struct {
	N, Kind string
	Down    bool
}

// newStockStrip is c's strip, kinds the stock loaded: each kind's total,
// or "unavailable" when none of its sources answered, so a 0 is always a
// loaded source with none; "none" when every total is 0 and every source
// answered.
func newStockStrip(c contract.FitCandidate, kinds []string) stockStrip {
	s := stockStrip{Number: c.Fitment.Number}
	if len(kinds) == 0 {
		s.Kind, s.Text = "absent", "Stock · not loaded"
		return s
	}
	words := []string{"Stock"}
	some := false
	for _, kind := range kinds {
		l := candidateList(c, kind)
		if l == nil {
			continue
		}
		down := slices.ContainsFunc(l.Sources, func(s contract.StockSource) bool { return s.Status == contract.StockUnavailable })
		if listDown(l) {
			words = append(words, kind+" unavailable")
			s.Counts = append(s.Counts, stockCount{N: "–", Kind: kind, Down: true})
		} else {
			words = append(words, strconv.Itoa(l.Total)+" "+kind)
			s.Counts = append(s.Counts, stockCount{N: strconv.Itoa(l.Total), Kind: kind})
		}
		some = some || l.Total > 0 || down
	}
	if !some {
		s.Kind, s.Text = "none", "Stock · none"
		return s
	}
	s.Kind, s.Text = "counts", strings.Join(words, " · ")
	return s
}

// listDown says l has sources and none of them answered: what it holds
// is not known, which is not a 0.
func listDown(l *contract.StockList) bool {
	return len(l.Sources) > 0 && !slices.ContainsFunc(l.Sources, func(s contract.StockSource) bool { return s.Status != contract.StockUnavailable })
}

// candidateList is c's stock list of kind, nil when it was not asked for.
func candidateList(c contract.FitCandidate, kind string) *contract.StockList {
	if kind == contract.StockShelf {
		return c.Shelf
	}
	return c.Yard
}

// stockPanel is the open candidate's rows: a print for each kind of
// stock loaded, under the candidate's row.
type stockPanel struct {
	Number  string
	Columns []stockColumn
}

// stockColumn is one kind's print: its heading, before the count, and
// its sources' as-of dates; its columns beside the id and the location
// (Lead, Heads and Trail); the rows shown; and, when there are more, how
// many of them are shown (Shown of Total) and the page showing all, whose
// link leaves the focus on the element Focus names. Level is its
// heading's: one under what the print is of, a candidate's number or a
// part's. Down says none of its sources answered: the print states no
// count, a dash in its place, and no "None.".
type stockColumn struct {
	ID, Heading, AsOf string
	Down              bool
	Level             int
	// Lead, Heads and Trail are the fields the headers of its rows'
	// sources declare, each once, in the headers' order and the rows',
	// but a field whose value is its row's id on every row: none where
	// no row's source declares any, and the print is then an id and a
	// location a row. The first of the fields the first pinning source
	// pins is Lead, printed beside the id, before the location; the
	// second is Trail, printed last. Heads are the rest, after the
	// location.
	Lead, Trail string
	Heads       []string
	// Last says the last of Heads is printed under the stock strip, as
	// Trail is: there is no Trail, and two Heads or more.
	Last bool
	// IDWidth is how many characters the widest of the rows' ids is, up
	// to the 24 a column holds before it wraps: where the print holds
	// Lead beside the id, the id's column is that wide.
	IDWidth      int
	Rows         []stockRow
	Shown, Total int
	More, Focus  string
}

// Bare says the print has no column beside the id and the location.
func (c stockColumn) Bare() bool {
	return c.Lead == "" && c.Trail == "" && len(c.Heads) == 0
}

// Span is how many cells a row of the print has: the id, the location
// and a cell for each of Lead, Heads and Trail.
func (c stockColumn) Span() int {
	n := 2 + len(c.Heads)
	if c.Lead != "" {
		n++
	}
	if c.Trail != "" {
		n++
	}
	return n
}

// The print's columns stand on the row it hangs from (#400): the id
// under the number's tape, the next column under the tape's second
// module, three columns under the ink, and the last under the stock
// strip. Where fewer than three stand under the ink, the last of them
// fills the ink, and the strip's too where nothing is under the strip:
// it is the column the stylesheet leaves to take the line's rest.

// inkHeads is how many of Heads stand under the ink, before the strip's.
func (c stockColumn) inkHeads() int {
	if c.Last {
		return len(c.Heads) - 1
	}
	return len(c.Heads)
}

// LocFill says the location takes the line's rest: no head stands
// between it and the strip's column, or the line's end.
func (c stockColumn) LocFill() bool {
	return c.inkHeads() == 0
}

// HeadClass is the class of the i'th of Heads in the print's head:
// "last" under the stock strip, "fill" where it takes the line's rest,
// else none.
func (c stockColumn) HeadClass(i int) string {
	ink := c.inkHeads()
	under := ink
	if c.Lead != "" {
		under++ // the location stands under the ink too
	}
	switch {
	case c.Last && i == len(c.Heads)-1:
		return "last"
	case i == ink-1 && under < 3:
		return "fill"
	}
	return ""
}

// stockRow is one row: its id and its location; under each of its
// print's columns, the value it has for the field of that name, "" where
// it has none or a null (Lead, Cells under Heads, and Trail); Folded,
// what it has beyond those, under the row; and two notes: the retired
// number a shelf row is tagged with, and whether a yard row is the VIN
// on the plate.
type stockRow struct {
	ID, Location string
	Lead, Trail  stockField
	Cells        []stockField
	Folded       []stockField
	Retired      string
	ThisVIN      bool
}

// Rest says the row has a second line: something folded, or a note.
func (r stockRow) Rest() bool {
	return len(r.Folded) > 0 || r.Retired != "" || r.ThisVIN
}

// stockField is one of a row's fields, its value as the source wrote it:
// a string's text, a number or a boolean as its JSON, nothing for a
// null. JSON marks a list or an object, whose Value is how many items or
// members it holds (#400): it is never a column's. Long marks a folded
// value past foldLines: it is cut to two lines, and pressed, printed
// whole.
type stockField struct {
	Name, Value string
	JSON, Long  bool
}

// foldLines is how many characters two lines of a folded value hold, at
// the 80 a line the print wraps them at.
const foldLines = 160

// stockHeadings are each kind's column heading, before its count.
var stockHeadings = map[string]string{contract.StockShelf: "On the shelf", contract.StockYard: "In the yard"}

// newStockColumn is l, a list of kind, as its print: declared are each
// source's header, by source name, vin the VIN on the plate, and more
// the page with every row, "" when all are shown.
func newStockColumn(kind string, l *contract.StockList, declared map[string]stockHeader, vin, more string) stockColumn {
	col := stockColumn{ID: "stock-" + kind, Heading: stockHeadings[kind], AsOf: sourcesText(l.Sources), Down: listDown(l), Level: 4,
		Shown: len(l.Rows), Total: l.Total, Focus: "stock-region"}
	fields := make([][]stockField, len(l.Rows))
	var heads, pin []string
	for i, r := range l.Rows {
		fields[i] = stockFields(r.Fields)
		h := declared[r.Source]
		for _, name := range h.Fields {
			if !slices.Contains(heads, name) {
				heads = append(heads, name)
			}
		}
		if pin == nil {
			pin = h.Pin
		}
		col.IDWidth = max(col.IDWidth, min(utf8.RuneCountInString(r.ID), 24))
	}
	// A field that is its row's id on every row says what the id does:
	// it is no column, and it does not fold.
	var same []string
	heads = slices.DeleteFunc(heads, func(name string) bool {
		for i, r := range l.Rows {
			if v, ok := columnValue(fields[i], name); !ok || v != r.ID {
				return false
			}
		}
		same = append(same, name)
		return true
	})
	if len(pin) > 0 && slices.Contains(heads, pin[0]) {
		col.Lead = pin[0]
	}
	if len(pin) > 1 && slices.Contains(heads, pin[1]) {
		col.Trail = pin[1]
	}
	col.Heads = slices.DeleteFunc(heads, func(name string) bool { return name == col.Lead || name == col.Trail })
	col.Last = col.Trail == "" && len(col.Heads) > 1
	cols := slices.Concat([]string{col.Lead}, col.Heads, []string{col.Trail})
	for i, r := range l.Rows {
		col.Rows = append(col.Rows, newStockRow(r, fields[i], cols, same, vin))
	}
	if more != "" && l.Total > len(l.Rows) {
		col.More = more
	}
	return col
}

// columnValue is the value fields has for name, which a column of that
// name prints: the first it has, and no list or object. ok is false
// where it has none.
func columnValue(fields []stockField, name string) (v string, ok bool) {
	for _, f := range fields {
		if f.Name == name && !f.JSON {
			return f.Value, true
		}
	}
	return "", false
}

// newStockRow is r, whose fields are fields, as its print shows it: cols
// its print's columns, Lead first and Trail last, "" where it has none;
// same the declared fields that are the id on every row, which it leaves
// out; and vin the VIN on the plate. A field is printed in the column of
// its name, whichever source declares that column (#400), when its value
// is no list or object; every other one that has a value folds under
// the row, in the source's order.
func newStockRow(r contract.StockRow, fields []stockField, cols, same []string, vin string) stockRow {
	row := stockRow{ID: r.ID, Location: r.Location, ThisVIN: r.VIN != "" && r.VIN == vin}
	cells := make([]stockField, len(cols))
	for i, name := range cols {
		cells[i].Name = name
	}
	if r.Reached == contract.ReachedRetired && r.Fitment != nil {
		row.Retired = r.Fitment.Number
	}
	filled := make([]bool, len(cols))
	for _, f := range fields {
		if !f.JSON && f.Name != "" {
			if i := slices.Index(cols, f.Name); i >= 0 && !filled[i] {
				cells[i].Value, filled[i] = f.Value, true
				continue
			}
			if slices.Contains(same, f.Name) && f.Value == r.ID {
				continue
			}
		}
		if f.Value != "" {
			f.Long = !f.JSON && utf8.RuneCountInString(f.Value) > foldLines
			row.Folded = append(row.Folded, f)
		}
	}
	row.Lead, row.Cells, row.Trail = cells[0], cells[1:len(cells)-1], cells[len(cells)-1]
	return row
}

// stockFields are a row's fields, a JSON object, in the source's order.
// Anything else a source wrote in their place is one field with no name.
func stockFields(raw json.RawMessage) []stockField {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		if f := newStockField("", raw); f.Value != "" {
			return []stockField{f}
		}
		return nil
	}
	var out []stockField
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			break
		}
		name, _ := t.(string)
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			break
		}
		out = append(out, newStockField(name, v))
	}
	return out
}

// newStockField is the field name with the JSON value v: a list or an
// object as how many items or members it holds.
func newStockField(name string, v json.RawMessage) stockField {
	f := stockField{Name: name}
	v = bytes.TrimSpace(v)
	switch {
	case len(v) == 0 || string(v) == "null":
	case v[0] == '"' && json.Unmarshal(v, &f.Value) == nil:
	case v[0] == '[':
		var items []json.RawMessage
		if err := json.Unmarshal(v, &items); err != nil { // stockFields decoded it: unreachable
			panic(err)
		}
		f.Value, f.JSON = strconv.Itoa(len(items)), true
	case v[0] == '{':
		var members map[string]json.RawMessage
		if err := json.Unmarshal(v, &members); err != nil { // stockFields decoded it: unreachable
			panic(err)
		}
		f.Value, f.JSON = strconv.Itoa(len(members)), true
	default:
		f.Value = string(v)
	}
	return f
}

// sourcesText is what a column's heading says of its sources: each one's
// as-of date, or why it did not answer, named when there are several.
func sourcesText(sources []contract.StockSource) string {
	var out []string
	for _, s := range sources {
		var t string
		switch {
		case s.Status == contract.StockUnavailable:
			t = "unavailable: " + s.Reason
		case s.AsOf != "":
			t = "as of " + s.AsOf
		default:
			continue
		}
		if len(sources) > 1 {
			t = s.Name + " " + t
		}
		out = append(out, t)
	}
	return strings.Join(out, " · ")
}

// fitStock fills the candidates' strips and the open one's rows, from
// fit, the answer to req, the candidates' request, the page's tags
// (c.Tags, one a candidate, in order); and, on the candidates' first
// page, the shelf rows with no number, named for the pressed part type
// and vehicle. With every row asked for, the open candidate's are its
// fitment key's whole stock, and the rows with no number every page of
// req's, dec the VIN's decode; the first page's JSON goes in res.
func (h *Handler) fitStock(ctx context.Context, c *fitCard, vin string, ask fitAsk, req contract.FitRequest, fit contract.Fit, dec *contract.VINDecode, vehicle string, res *result) error {
	open, want := -1, ask.key()
	for i, cand := range fit.Candidates {
		s := newStockStrip(cand, ask.Stock)
		s.ID = "stk-" + strconv.Itoa(i+1)
		if open < 0 && s.Kind == "counts" && ask.Number != "" && candidateKey(cand) == want {
			open, s.Open, s.Href = i, true, fitHref(vin, ask.Part, ask.After)
		} else {
			s.Href = stockHref(vin, ask.Part, ask.After, candidateKey(cand), "")
		}
		c.Tags[i].Strip = s
	}
	var key fitKey
	if open >= 0 {
		key = candidateKey(fit.Candidates[open])
	}
	rows := ""
	if l := fit.ShelfByVehicle; ask.After == "" && l != nil && len(l.Rows) > 0 {
		more := stockHref(vin, ask.Part, ask.After, key, rowsUnnumbered)
		if ask.Rows == rowsUnnumbered {
			var err error
			if l, err = h.allUnnumbered(ctx, req, dec, res); err != nil {
				return err
			}
			more, rows = "", rowsUnnumbered
		}
		col := newStockColumn(contract.StockShelf, l, ask.Declared, vin, more)
		col.ID, col.Focus = "stock-nonum", "stock-nonum"
		c.NoNumber, c.NoNumberName = &col, "On the shelf with no number · "+c.Pressed+" · "+vehicle
	}
	if open < 0 {
		c.Page = stockHref(vin, ask.Part, ask.After, key, rows)
		return nil
	}
	cand := fit.Candidates[open]
	if ask.Rows == rowsAll {
		rows = rowsAll
	}
	c.Page = stockHref(vin, ask.Part, ask.After, key, rows)
	cols, err := h.stockColumns(ctx, key, cand, ask.Stock, ask.Declared, ask.Rows == rowsAll, stockHref(vin, ask.Part, ask.After, key, rowsAll), vin, res)
	if err != nil {
		return err
	}
	c.Tags[open].Panel = &stockPanel{Number: cand.Fitment.Number, Columns: cols}
	return nil
}

// stockColumns are an open candidate's or part's rows, a print for each
// of kinds: the rows cand arrived with, the page with every row more; or,
// all asked for, the whole of key's stock (allStock), with no more.
// declared are each source's header (stockDeclared), and vin is the
// VIN on the plate, "" for none.
func (h *Handler) stockColumns(ctx context.Context, key fitKey, cand contract.FitCandidate, kinds []string, declared map[string]stockHeader, all bool, more, vin string, res *result) ([]stockColumn, error) {
	lists := map[string]*contract.StockList{contract.StockShelf: cand.Shelf, contract.StockYard: cand.Yard}
	if all {
		var err error
		if lists, err = h.allStock(ctx, key, kinds, res); err != nil {
			return nil, err
		}
		more = ""
	}
	var cols []stockColumn
	for _, kind := range kinds {
		if l := lists[kind]; l != nil {
			cols = append(cols, newStockColumn(kind, l, declared, vin, more))
		}
	}
	return cols, nil
}

// candidatesNext is the candidates' next page after fit, the answer to
// req: fit's own Next, but with shelf stock asked for, the answer's Next
// stays while the shelf rows with no number have more, and those are
// listed on the first page alone, so the catalog alone, asked again
// with no stock, says whether more candidates follow.
func (h *Handler) candidatesNext(ctx context.Context, req contract.FitRequest, fit contract.Fit, dec *contract.VINDecode) (string, error) {
	if fit.Next == "" || fit.ShelfByVehicle == nil {
		return fit.Next, nil
	}
	if len(fit.Candidates) < req.Limit {
		return "", nil
	}
	req.Stock = nil
	again, _, err := h.fit(ctx, req, dec)
	return again.Next, err
}

// allUnnumbered is every shelf row with no number for req's part type
// and vehicle: req, the candidates' request, asked of the shelf alone
// from its first page, every page of it, the first page's JSON in res.
func (h *Handler) allUnnumbered(ctx context.Context, req contract.FitRequest, dec *contract.VINDecode, res *result) (*contract.StockList, error) {
	req.Stock, req.Limit, req.After = []string{contract.StockShelf}, contract.MaxListLimit, ""
	var out *contract.StockList
	for {
		fit, raw, err := h.fit(ctx, req, dec)
		if err != nil {
			return nil, err
		}
		if out == nil {
			if err := h.fitJSON(res, "unnumbered-json", "The shelf rows with no number", "the shelf rows with no number's JSON", fit, raw); err != nil {
				return nil, err
			}
			out = &contract.StockList{Total: fit.ShelfByVehicle.Total, Sources: fit.ShelfByVehicle.Sources}
		}
		out.Rows = append(out.Rows, fit.ShelfByVehicle.Rows...)
		if fit.Next == "" || len(out.Rows) >= out.Total {
			return out, nil
		}
		req.After = fit.Next
	}
}

// allStock is every row of key's stock of kinds, by kind: a fit from the
// fitment key with the same stock, every page of it, the first page's
// JSON in res.
func (h *Handler) allStock(ctx context.Context, k fitKey, kinds []string, res *result) (map[string]*contract.StockList, error) {
	req := contract.FitRequest{From: contract.FitFrom{Fitment: &contract.FitmentKey{Type: k.Type, Number: k.Number, Side: k.Side}}, Stock: kinds, Limit: contract.MaxListLimit}
	out := map[string]*contract.StockList{}
	for {
		fit, raw, err := h.fit(ctx, req, nil)
		if err != nil {
			return nil, err
		}
		if req.After == "" {
			if err := h.fitJSON(res, "stock-json", "The stock", "the stock's JSON", fit, raw); err != nil {
				return nil, err
			}
		}
		for kind, l := range map[string]*contract.StockList{contract.StockShelf: fit.Shelf, contract.StockYard: fit.Yard} {
			switch {
			case l == nil:
			case out[kind] == nil:
				out[kind] = &contract.StockList{Rows: slices.Clone(l.Rows), Total: l.Total, Sources: l.Sources}
			default:
				out[kind].Rows = append(out[kind].Rows, l.Rows...)
			}
		}
		if fit.Next == "" {
			return out, nil
		}
		req.After = fit.Next
	}
}
