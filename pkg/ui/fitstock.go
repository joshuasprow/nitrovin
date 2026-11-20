package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// A candidate's stock on the page (ADR 0019, #323). The candidates
// request asks for the stock of the sources loaded beside the catalog it
// answers from (stockKinds), so every candidate arrives with its shelf's
// and its yard's totals and first rows (contract.StockFirst), and its tag
// ends in a strip that counts them. No other request asks for stock: the
// part types' asks for none, and a page with no part type pressed asks
// for no candidates. The strip is a link to the page with the candidate
// open, the URL's third parameter, named as the fit request's fitment key
// names it (fitment.number, with fitment.side for a fitment with a side),
// so it works without a script and a link reopens it; the open
// candidate's strip links to the page with none open. The open
// candidate's rows are the ones its candidate arrived with; "Show all"
// is the page with rows=all, which asks for the whole of its stock, a
// fit from its fitment key with the same stock, a page at a time.
//
// With shelf stock asked for, the candidates' answer also carries the
// shelf rows with no number for the part type and the vehicle, which
// belong to no candidate: they are listed once, after the candidates.

const (
	// numberParam and sideParam are the page's third parameter, the
	// candidate whose stock is open: its number, and its side when it
	// has one, as the fit request's fitment key names them.
	numberParam = "fitment.number"
	sideParam   = "fitment.side"
	// rowsParam, "all", shows every row of the open candidate's stock.
	rowsParam = "rows"
	// stockPanelID is the open candidate's rows, which its strip
	// controls; stockRegionID their columns, the region that scrolls
	// sideways on a narrow page, which "Show all" leaves the focus on.
	stockPanelID  = "stock-rows"
	stockRegionID = "stock-region"
)

// fitAsk is what the page's URL asks of the parts that fit: the part
// type pressed and its candidates' page, the candidate whose stock is
// open, whether every row of it is shown, and the kinds of stock loaded.
type fitAsk struct {
	Part, After  string
	Number, Side string
	All          bool
	Stock        []string
}

// stockHref is the page with vin on the plate, part pressed from its
// candidates' page after, and the candidate number and side open, every
// row of its stock shown when all is set.
func stockHref(vin, part, after, number, side string, all bool) string {
	s := fitHref(vin, part, after) + "&" + numberParam + "=" + url.QueryEscape(number)
	if side != "" {
		s += "&" + sideParam + "=" + url.QueryEscape(side)
	}
	if all {
		s += "&" + rowsParam + "=all"
	}
	return s
}

// stockKinds are the kinds of stock loaded beside the catalog the page's
// fit answers from, shelf first: the server's own; in the browser, the
// server's while it holds the catalog, else the browser's own, which
// info lists.
func (h *Handler) stockKinds(info contract.Info) []string {
	files := info.Vocabularies
	if h.lib != nil && h.lib.ServerCatalog().Name != "" {
		files = h.lib.ServerStock()
	}
	var out []string
	for _, kind := range []string{contract.StockShelf, contract.StockYard} {
		if slices.ContainsFunc(files, func(v contract.VocabularyFile) bool {
			return v.Kind == string(vocab.KindStock) && v.StockCounts != nil && v.Source == kind
		}) {
			out = append(out, kind)
		}
	}
	return out
}

// stockStrip is the strip along a tag's bottom edge. Kind is "counts",
// a link that opens the candidate's rows (Href) or, Open, closes them;
// "none", stock loaded and none for it; or "absent", no stock loaded,
// which opens the Vocabularies dialog. Text is its words.
type stockStrip struct {
	Kind, Text, Href, ID, Number string
	Open                         bool
}

// newStockStrip is c's strip, kinds the stock loaded: every kind's total,
// or "none" when each is 0 and every source answered.
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
		words = append(words, strconv.Itoa(l.Total)+" "+kind)
		some = some || l.Total > 0 || slices.ContainsFunc(l.Sources, func(s contract.StockSource) bool { return s.Status == contract.StockUnavailable })
	}
	if !some {
		s.Kind, s.Text = "none", "Stock · none"
		return s
	}
	s.Kind, s.Text = "counts", strings.Join(words, " · ")
	return s
}

// candidateList is c's stock list of kind, nil when it was not asked for.
func candidateList(c contract.FitCandidate, kind string) *contract.StockList {
	if kind == contract.StockShelf {
		return c.Shelf
	}
	return c.Yard
}

// stockPanel is the open candidate's rows: a column for each kind of
// stock loaded. Col is its tag's column on a wide page, 1 to 3, where
// the panel's notch points; MoreWide and MoreNarrow say that tags follow
// it, on a wide page and a narrow one, so the rails run on down past it.
type stockPanel struct {
	Number               string
	Col                  int
	MoreWide, MoreNarrow bool
	Columns              []stockColumn
}

// stockColumn is one kind's rows: its heading, with the count, and its
// sources' as-of dates; the rows shown; and, when there are more,
// how many of them are shown (Shown of Total) and the page showing all.
type stockColumn struct {
	ID, Heading, AsOf string
	Rows              []stockRow
	Shown, Total      int
	More              string
}

// stockRow is one row: its id, its location and its source's fields
// verbatim, and two notes: the retired number a shelf row is tagged
// with, and whether a yard row is the VIN on the plate.
type stockRow struct {
	ID, Location string
	Fields       []stockField
	Retired      string
	ThisVIN      bool
}

// stockField is one of a row's fields, its value as the source wrote it:
// a string's text, any other value's JSON.
type stockField struct {
	Name, Value string
}

// stockHeadings are each kind's column heading, before its count.
var stockHeadings = map[string]string{contract.StockShelf: "On the shelf", contract.StockYard: "In the yard"}

// newStockColumn is l, a list of kind, as its column, vin the VIN on the
// plate; more is the page with every row, "" when all are shown.
func newStockColumn(kind string, l *contract.StockList, vin, more string) stockColumn {
	col := stockColumn{ID: "stock-" + kind, Heading: stockHeadings[kind] + " · " + strconv.Itoa(l.Total), AsOf: sourcesText(l.Sources),
		Shown: len(l.Rows), Total: l.Total}
	for _, r := range l.Rows {
		col.Rows = append(col.Rows, newStockRow(r, vin))
	}
	if more != "" && l.Total > len(l.Rows) {
		col.More = more
	}
	return col
}

// newStockRow is r as its row shows it, vin the VIN on the plate.
func newStockRow(r contract.StockRow, vin string) stockRow {
	row := stockRow{ID: r.ID, Location: r.Location, Fields: stockFields(r.Fields), ThisVIN: r.VIN != "" && r.VIN == vin}
	if r.Reached == contract.ReachedRetired && r.Fitment != nil {
		row.Retired = r.Fitment.Number
	}
	return row
}

// stockFields are a row's fields, a JSON object, in the source's order.
func stockFields(raw json.RawMessage) []stockField {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return []stockField{{Value: string(raw)}}
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
		f := stockField{Name: name}
		if v[0] != '"' || json.Unmarshal(v, &f.Value) != nil {
			var b bytes.Buffer
			if json.Compact(&b, v) != nil {
				b.Write(v)
			}
			f.Value = b.String()
		}
		out = append(out, f)
	}
	return out
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
// fit, the candidates' answer, the page's tags (c.Tags, one a candidate,
// in order); and the shelf rows with no number, named for the pressed
// part type and vehicle. With every row asked for, the open candidate's
// are its fitment key's whole stock, its first page's JSON in res.
func (h *Handler) fitStock(ctx context.Context, c *fitCard, vin string, ask fitAsk, fit contract.Fit, vehicle string, res *result) error {
	open := -1
	for i, cand := range fit.Candidates {
		s := newStockStrip(cand, ask.Stock)
		s.ID = "stk-" + strconv.Itoa(i+1)
		if open < 0 && s.Kind == "counts" && ask.Number != "" && cand.Fitment.Number == ask.Number && cand.Fitment.Side == ask.Side {
			open, s.Open, s.Href = i, true, fitHref(vin, ask.Part, ask.After)
		} else {
			s.Href = stockHref(vin, ask.Part, ask.After, cand.Fitment.Number, cand.Fitment.Side, false)
		}
		c.Tags[i].Strip = s
	}
	if l := fit.ShelfByVehicle; l != nil && l.Total > 0 {
		col := newStockColumn(contract.StockShelf, l, vin, "")
		col.ID = "stock-nonum"
		c.NoNumber, c.NoNumberName = &col, "On the shelf with no number · "+c.Pressed+" · "+vehicle
	}
	if open < 0 {
		return nil
	}
	cand := fit.Candidates[open]
	c.Page = stockHref(vin, ask.Part, ask.After, cand.Fitment.Number, cand.Fitment.Side, ask.All)
	lists := map[string]*contract.StockList{contract.StockShelf: cand.Shelf, contract.StockYard: cand.Yard}
	more := stockHref(vin, ask.Part, ask.After, cand.Fitment.Number, cand.Fitment.Side, true)
	if ask.All {
		var err error
		if lists, err = h.allStock(ctx, contract.FitmentKey{Type: cand.Fitment.Type, Number: cand.Fitment.Number, Side: cand.Fitment.Side}, ask.Stock, res); err != nil {
			return err
		}
		more = ""
	}
	p := &stockPanel{Number: cand.Fitment.Number, Col: open%3 + 1, MoreWide: len(c.Tags) > (open/3+1)*3, MoreNarrow: open < len(c.Tags)-1}
	for _, kind := range ask.Stock {
		if l := lists[kind]; l != nil {
			p.Columns = append(p.Columns, newStockColumn(kind, l, vin, more))
		}
	}
	c.Tags[open].Panel = p
	return nil
}

// allStock is every row of key's stock of kinds, by kind: a fit from the
// fitment key with the same stock, every page of it, the first page's
// JSON in res.
func (h *Handler) allStock(ctx context.Context, key contract.FitmentKey, kinds []string, res *result) (map[string]*contract.StockList, error) {
	req := contract.FitRequest{From: contract.FitFrom{Fitment: &key}, Stock: kinds, Limit: contract.MaxListLimit}
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
