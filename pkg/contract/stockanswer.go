package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"sync/atomic"
	"time"
)

// This file is where a fit answer's stock options apply, once every list
// of its batch is whole: a request's stock_fields (fieldCuts) and a
// batch's stock rows once (FitBatchRows). Fit and FitBatch answer through
// it, and so every surface: HTTP, stdio, MCP and the browser.

// fieldCuts cuts the fields of a batch's stock rows to each request's
// stock_fields, remembering every distinct object it cut, so a row that
// many lists hold, or many sources' rows with the same fields, is read
// once a cut.
type fieldCuts map[string]map[string]json.RawMessage

// cut cuts the fields of every stock row of out to names; nil names keep
// every field and leave out as it is.
func (c fieldCuts) cut(out *Fit, names []string) {
	if names == nil {
		return
	}
	key, err := json.Marshal(names) // tells apart every list of names
	if err != nil {                 // strings: unreachable
		panic(err)
	}
	done := c[string(key)]
	if done == nil {
		done = map[string]json.RawMessage{}
		c[string(key)] = done
	}
	for _, l := range out.stockLists() {
		for i, r := range l.Rows {
			kept, ok := done[string(r.Fields)]
			if !ok {
				kept = cutFields(r.Fields, names)
				done[string(r.Fields)] = kept
			}
			l.Rows[i].Fields = kept
		}
	}
}

// cutFields is object, a row's fields, with only its members named in
// names, in its own order, their values verbatim.
func cutFields(object json.RawMessage, names []string) json.RawMessage {
	dec := json.NewDecoder(bytes.NewReader(object))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		// A source serves fields as an object: a file's lines and a
		// connector's rows are refused or replaced by {} otherwise.
		panic("stock: a row's fields are not a JSON object: unreachable")
	}
	out := []byte{'{'}
	for dec.More() {
		t, err := dec.Token()
		if err != nil { // a valid object: unreachable
			panic(err)
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil { // a valid object: unreachable
			panic(err)
		}
		name := t.(string)
		if !slices.Contains(names, name) {
			continue
		}
		if len(out) > 1 {
			out = append(out, ',')
		}
		k, err := json.Marshal(name)
		if err != nil { // a string: unreachable
			panic(err)
		}
		out = append(append(append(out, k...), ':'), v...)
	}
	return append(out, '}')
}

// stockLists are every stock list of f: each candidate's, each related
// vehicle's, then the answer's own.
func (f *Fit) stockLists() []*StockList {
	var out []*StockList
	for _, c := range f.Candidates {
		out = append(out, c.Shelf, c.Yard)
	}
	for _, r := range f.Related {
		out = append(out, r.Yard)
	}
	return slices.DeleteFunc(append(out, f.Shelf, f.Yard, f.ShelfByVehicle), func(l *StockList) bool { return l == nil })
}

// FitBatchRows is a batch's answer with its stock rows once (ADR 0018),
// POST /v1/fit?stock_rows=once: Fits, the fits in request order, each as
// the batch would answer it but that every stock list's rows are indexes
// into StockRows, the batch's distinct rows, in the order the batch first
// lists them. A row is one row of StockRows for every list holding it,
// whatever its fit, list or kind: a row is its kind, its source and its id,
// and two lists holding it differently (cut by different stock_fields, or
// under different fitments) hold two rows.
type FitBatchRows struct {
	Fits      []FitRefs  `json:"fits"`
	StockRows []StockRow `json:"stock_rows"`
}

// FitRefs is a Fit of a FitBatchRows: the Fit, whose own stock lists and
// list items holding them (Candidates, Related, Shelf, Yard,
// ShelfByVehicle) are nil, replaced by these, holding StockRefs.
type FitRefs struct {
	Fit
	Candidates     []FitCandidateRefs   `json:"candidates,omitzero"`
	Related        []RelatedVehicleRefs `json:"related,omitzero"`
	Shelf          *StockRefs           `json:"shelf,omitempty"`
	Yard           *StockRefs           `json:"yard,omitempty"`
	ShelfByVehicle *StockRefs           `json:"shelf_by_vehicle,omitempty"`
}

// FitCandidateRefs is a FitCandidate of a FitRefs: its Shelf and Yard nil,
// replaced by these.
type FitCandidateRefs struct {
	FitCandidate
	Shelf *StockRefs `json:"shelf,omitempty"`
	Yard  *StockRefs `json:"yard,omitempty"`
}

// RelatedVehicleRefs is a RelatedVehicle of a FitRefs: its Yard nil,
// replaced by this one.
type RelatedVehicleRefs struct {
	RelatedVehicle
	Yard *StockRefs `json:"yard,omitempty"`
}

// StockRefs is a StockList of a FitBatchRows: Rows are the list's rows'
// indexes into its StockRows, in the list's order.
type StockRefs struct {
	Rows    []int         `json:"rows"`
	Total   int           `json:"total"`
	Sources []StockSource `json:"sources"`
}

// rowTable is a FitBatchRows' StockRows being gathered: every distinct
// row, and where each row's kind, source and id are in it.
type rowTable struct {
	rows []StockRow
	at   map[rowKey][]int
}

type rowKey struct{ kind, source, id string }

// rowsOnce is fits, a batch's answers, with their stock rows once.
func rowsOnce(fits []Fit) FitBatchRows {
	t := rowTable{rows: []StockRow{}, at: map[rowKey][]int{}}
	out := FitBatchRows{Fits: make([]FitRefs, len(fits))}
	for i, f := range fits {
		r := FitRefs{Fit: f}
		if f.Candidates != nil {
			r.Candidates = make([]FitCandidateRefs, len(f.Candidates))
			for j, c := range f.Candidates {
				r.Candidates[j] = FitCandidateRefs{FitCandidate: c, Shelf: t.refs(StockShelf, c.Shelf), Yard: t.refs(StockYard, c.Yard)}
				r.Candidates[j].FitCandidate.Shelf, r.Candidates[j].FitCandidate.Yard = nil, nil
			}
		}
		if f.Related != nil {
			r.Related = make([]RelatedVehicleRefs, len(f.Related))
			for j, v := range f.Related {
				r.Related[j] = RelatedVehicleRefs{RelatedVehicle: v, Yard: t.refs(StockYard, v.Yard)}
				r.Related[j].RelatedVehicle.Yard = nil
			}
		}
		r.Shelf, r.Yard, r.ShelfByVehicle = t.refs(StockShelf, f.Shelf), t.refs(StockYard, f.Yard), t.refs(StockShelf, f.ShelfByVehicle)
		r.Fit.Candidates, r.Fit.Related, r.Fit.Shelf, r.Fit.Yard, r.Fit.ShelfByVehicle = nil, nil, nil, nil, nil
		out.Fits[i] = r
	}
	out.StockRows = t.rows
	return out
}

// refs is l, a list of kind, as references into t; nil for nil.
func (t *rowTable) refs(kind string, l *StockList) *StockRefs {
	if l == nil {
		return nil
	}
	out := &StockRefs{Rows: make([]int, len(l.Rows)), Total: l.Total, Sources: l.Sources}
	for i, r := range l.Rows {
		out.Rows[i] = t.index(kind, r)
	}
	return out
}

// index is row r's index in t, of kind, adding it when t has no row of
// its kind, source and id that is the same in every field.
func (t *rowTable) index(kind string, r StockRow) int {
	k := rowKey{kind, r.Source, r.ID}
	for _, i := range t.at[k] {
		if reflect.DeepEqual(t.rows[i], r) {
			return i
		}
	}
	t.at[k] = append(t.at[k], len(t.rows))
	t.rows = append(t.rows, r)
	return len(t.rows) - 1
}

// StockTime sums how long a fit's stock lookups took, every source's
// summed, for a caller that reports it (GET and POST /v1/fit's
// Server-Timing): WithStockTime gives a fit's context one, and each of its
// lookups adds its time, whether it answered or not. It never enters an
// answer. Safe for concurrent use.
type StockTime struct{ ns atomic.Int64 }

// Duration is the lookups' time so far.
func (s *StockTime) Duration() time.Duration { return time.Duration(s.ns.Load()) }

type stockTimeKey struct{}

// WithStockTime is ctx with s, which the stock lookups of a fit given it
// add their time to.
func WithStockTime(ctx context.Context, s *StockTime) context.Context {
	return context.WithValue(ctx, stockTimeKey{}, s)
}

// timeLookup starts timing one lookup under ctx, and returns the func
// that stops it; nil when ctx has no StockTime.
func timeLookup(ctx context.Context) func() {
	s, ok := ctx.Value(stockTimeKey{}).(*StockTime)
	if !ok {
		return nil
	}
	start := time.Now()
	return func() { s.ns.Add(int64(time.Since(start))) }
}
