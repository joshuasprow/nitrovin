package nitrovin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/stock"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// stockTranslator loads testdata/vocab, the synthetic catalog and the
// synthetic shelf and yard in testdata/vocab/stock.
func stockTranslator(t *testing.T) *Translator {
	t.Helper()
	files, err := vocab.Open("../../testdata/vocab", "../../testdata/vocab/fitment", "../../testdata/vocab/stock", "../../testdata/vocab/stock/drift")
	if err != nil {
		t.Fatal(err)
	}
	tr, err := NewTranslator(embeddedDecoder(t), files...)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

// stockText writes a list as "total: row row …", each row its id, how it
// matched, and how its number reached the fitment when not directly.
func stockText(l *contract.StockList) string {
	if l == nil {
		return "absent"
	}
	var rows []string
	for _, r := range l.Rows {
		s := r.ID + "/" + r.Matched
		if r.Reached == contract.ReachedRetired {
			s += "/retired"
		}
		rows = append(rows, s)
	}
	return fmt.Sprintf("%d: %s", l.Total, strings.Join(rows, " "))
}

// sourcesText writes a list's sources: name, kind, status and mismatch.
func sourcesText(l *contract.StockList) string {
	var out []string
	for _, s := range l.Sources {
		t := s.Name + " " + s.Kind + " " + s.Status
		if s.CatalogMismatch {
			t += " mismatch"
		}
		if s.Reason != "" {
			t += " (" + s.Reason + ")"
		}
		out = append(out, t)
	}
	return strings.Join(out, ", ")
}

// TestFitStock: every row form answers through fit. A candidate's shelf
// holds the rows of its number and of the retired number it replaced, from
// every shelf; its yard the vehicles among its applications, by a
// precomputed decode, a decoded VIN or a catalog vehicle; a shelf numbered
// under another catalog release is marked, and one naming no release is
// not; an orphan row appears under no candidate; rows with no number
// answer as shelf_by_vehicle; and related vehicles count their yard.
func TestFitStock(t *testing.T) {
	tr := stockTranslator(t)
	const ram = `"from":{"vocabulary":"catalog-models","entry":{"make":"Dodge","model":"Ram 1500"}},"model_year":2010`

	f, err := fit(t, tr, `{`+ram+`,"part":{"vocabulary":"part-codes","entry":{"code":"ROT"}},"stock":["shelf","yard"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Candidates) != 1 || f.Candidates[0].Fitment.Number != "BR-2001" {
		t.Fatalf("candidates %+v", f.Candidates)
	}
	c := f.Candidates[0]
	if got, want := stockText(c.Shelf), "4: A-1/fitment S-2/fitment S-3/fitment/retired O-2/fitment/retired"; got != want {
		t.Errorf("BR-2001's shelf %s, want %s", got, want)
	}
	if got, want := sourcesText(c.Shelf), "shelf-annex file available, shelf-stock file available mismatch, shelf-orphans file available"; got != want {
		t.Errorf("BR-2001's shelf sources %s, want %s", got, want)
	}
	if s := c.Shelf.Sources[1]; s.Release != "7.01" || s.Catalog != "parts-catalog" || s.AsOf != "2026-11-01T08:00:00Z" {
		t.Errorf("the shelf's block %+v", s)
	}
	if s := c.Shelf.Sources[0]; s.Release != "" || s.CatalogMismatch || s.AsOf != "2026-10-30" {
		t.Errorf("the annex's block, no release: %+v", s)
	}
	if got, want := stockText(c.Yard), "3: Y-1/vin Y-2/vin Y-3/catalog-vehicle"; got != want {
		t.Errorf("BR-2001's yard %s, want %s", got, want)
	}
	if got, want := sourcesText(c.Yard), "yard-stock file available"; got != want {
		t.Errorf("BR-2001's yard sources %s, want %s", got, want)
	}
	if got := string(c.Yard.Rows[0].Fields); !strings.Contains(got, `"photos":["y100-1.jpg","y100-2.jpg"]`) {
		t.Errorf("Y-1's fields are not verbatim: %s", got)
	}
	if f.ShelfByVehicle == nil || f.ShelfByVehicle.Total != 0 {
		t.Errorf("rotors' shelf_by_vehicle %s", stockText(f.ShelfByVehicle))
	}

	f, err = fit(t, tr, `{`+ram+`,"part":{"vocabulary":"part-codes","entry":{"code":"MIR"}},"stock":["shelf"]}`)
	if err != nil {
		t.Fatal(err)
	}
	var mirrors []string
	for _, c := range f.Candidates {
		mirrors = append(mirrors, c.Fitment.Side+" "+stockText(c.Shelf)+" yard "+stockText(c.Yard))
	}
	if want := []string{"left 1: S-1/fitment yard absent", "right 0:  yard absent"}; !slices.Equal(mirrors, want) {
		t.Errorf("mirrors %q, want %q: the orphan O-1 is under none", mirrors, want)
	}
	if got, want := stockText(f.ShelfByVehicle), "2: A-2/vehicle S-4/vehicle"; got != want {
		t.Errorf("mirrors' shelf_by_vehicle %s, want %s", got, want)
	}
	if r := f.ShelfByVehicle.Rows[1]; r.Part != "mirror" || r.Vehicle == nil || r.Year != contract.Value(int32(2010)) {
		t.Errorf("S-4 as answered %+v", r)
	}

	f, err = fit(t, tr, `{"from":{"model":1850},"model_year":2010,"related":true,"stock":["yard"]}`)
	if err != nil {
		t.Fatal(err)
	}
	var related []string
	for _, r := range f.Related {
		related = append(related, vehicleText(r.Vehicle)+" "+stockText(r.Yard))
	}
	if want := []string{"Dodge/Ram 1500 2: "}; !slices.Equal(related, want) {
		t.Errorf("related %q, want %q", related, want)
	}
	if _, err := tr.Fit(context.Background(), contract.FitRequest{From: contract.FitFrom{SpineRef: contract.SpineRef{Model: 1850}}, Related: true, Stock: []string{"shelf"}}); contract.AsProblem(err).Pointer != "/stock" {
		t.Errorf("related with stock shelf: %v, want a refusal at /stock", err)
	}

	// The engine is in stock with no number, for a Silverado.
	f, err = fit(t, tr, `{"from":{"vocabulary":"catalog-models","entry":{"make":"Chevrolet","model":"Silverado 1500"}},"model_year":2010,"part":{"spine":"part-types","id":"engine"},"stock":["shelf"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := stockText(f.ShelfByVehicle), "1: S-5/vehicle"; got != want {
		t.Errorf("a Silverado's engines by vehicle %s, want %s", got, want)
	}
}

// TestFitStockLocation: location keeps the rows at its locations and drops
// the others, in every list and across pages.
func TestFitStockLocation(t *testing.T) {
	tr := stockTranslator(t)
	const ram = `"from":{"vocabulary":"catalog-models","entry":{"make":"Dodge","model":"Ram 1500"}},"model_year":2010`
	f, err := fit(t, tr, `{`+ram+`,"part":{"vocabulary":"part-codes","entry":{"code":"ROT"}},"stock":["shelf","yard"],"location":["north","east"]}`)
	if err != nil {
		t.Fatal(err)
	}
	c := f.Candidates[0]
	if got, want := stockText(c.Shelf)+" | "+stockText(c.Yard), "2: A-1/fitment S-3/fitment/retired | 1: Y-1/vin"; got != want {
		t.Errorf("BR-2001 at north and east: %s, want %s", got, want)
	}
	f, err = fit(t, tr, `{`+ram+`,"part":{"vocabulary":"part-codes","entry":{"code":"MIR"}},"stock":["shelf"],"location":["north"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := stockText(f.Candidates[0].Shelf)+" | "+stockText(f.ShelfByVehicle), "1: S-1/fitment | 1: S-4/vehicle"; got != want {
		t.Errorf("mirrors at north: %s, want %s", got, want)
	}
	pages := stockPages(t, tr, `{"from":{"fitment":{"type":"brake-rotor-front","number":"BR-1999"}},"stock":["shelf","yard"],"location":["north","east"],"limit":1`)
	want := []string{
		"1 vehicles, shelf 2: A-1/fitment, yard 1: Y-1/vin",
		"1 vehicles, shelf 2: S-3/fitment/retired, yard 1: ",
	}
	if !slices.Equal(pages, want) {
		t.Errorf("pages at north and east\n%q\nwant\n%q", pages, want)
	}
}

// stockPages fits request (a JSON object without its closing brace) page
// by page, writing each page's vehicles count and stock lists.
func stockPages(t *testing.T, tr *Translator, request string) []string {
	t.Helper()
	var pages []string
	after := ""
	for range 10 {
		req := request
		if after != "" {
			req += `,"after":"` + after + `"`
		}
		f, err := fit(t, tr, req+`}`)
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case f.Shelf != nil:
			pages = append(pages, fmt.Sprintf("%d vehicles, shelf %s, yard %s", len(f.Vehicles), stockText(f.Shelf), stockText(f.Yard)))
		default:
			var cs []string
			for _, c := range f.Candidates {
				cs = append(cs, c.Fitment.Number+" "+c.Fitment.Side)
			}
			pages = append(pages, fmt.Sprintf("candidates %q, by vehicle %s", cs, stockText(f.ShelfByVehicle)))
		}
		if after = f.Next; after == "" {
			break
		}
	}
	return pages
}

// TestFitStockPaging: from a fitment key, after and limit page the
// fitment's shelf and yard with its vehicles; with part, they page
// shelf_by_vehicle with the candidates; and next goes on while any list
// has more.
func TestFitStockPaging(t *testing.T) {
	tr := stockTranslator(t)
	pages := stockPages(t, tr, `{"from":{"fitment":{"type":"brake-rotor-front","number":"BR-1999"}},"stock":["shelf","yard"],"limit":1`)
	want := []string{
		"1 vehicles, shelf 4: A-1/fitment, yard 3: Y-1/vin",
		"1 vehicles, shelf 4: S-2/fitment, yard 3: Y-2/vin",
		"0 vehicles, shelf 4: S-3/fitment/retired, yard 3: Y-3/catalog-vehicle",
		"0 vehicles, shelf 4: O-2/fitment/retired, yard 3: ",
	}
	if !slices.Equal(pages, want) {
		t.Errorf("pages\n%q\nwant\n%q", pages, want)
	}
	pages = stockPages(t, tr, `{"from":{"vocabulary":"catalog-models","entry":{"make":"Dodge","model":"Ram 1500"}},"model_year":2010,"part":{"spine":"part-types","id":"mirror"},"stock":["shelf"],"limit":1`)
	want = []string{
		`candidates ["MR-1001 left"], by vehicle 2: A-2/vehicle`,
		`candidates ["MR-1001 right"], by vehicle 2: S-4/vehicle`,
	}
	if !slices.Equal(pages, want) {
		t.Errorf("candidate pages\n%q\nwant\n%q", pages, want)
	}
	pages = stockPages(t, tr, `{"from":{"vocabulary":"catalog-models","entry":{"make":"Dodge","model":"Ram 1500"}},"model_year":2010,"part":{"spine":"part-types","id":"engine"},"stock":["shelf"],"limit":1`)
	if want := []string{`candidates ["EN-57 "], by vehicle 0: `}; !slices.Equal(pages, want) {
		t.Errorf("engine pages %q, want %q", pages, want)
	}
}

// counting is a stock source counting its lookups, by source and lookup.
type counting struct {
	stock.Source
	mu    *sync.Mutex
	calls map[string]int
}

func (c counting) count(lookup string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls[c.Info().Name+" "+lookup]++
}
func (c counting) ShelfByFitment(ctx context.Context, lists [][]vocab.FitmentKey, p stock.Page) ([]stock.Result, stock.Meta, error) {
	c.count("shelf by fitment")
	return c.Source.ShelfByFitment(ctx, lists, p)
}
func (c counting) ShelfByVehicle(ctx context.Context, lists []stock.ShelfVehicle, p stock.Page) ([]stock.Result, stock.Meta, error) {
	c.count("shelf by vehicle")
	return c.Source.ShelfByVehicle(ctx, lists, p)
}
func (c counting) Yard(ctx context.Context, lists []stock.YardQuery, p stock.Page) ([]stock.Result, stock.Meta, error) {
	c.count("yard")
	return c.Source.Yard(ctx, lists, p)
}

// TestFitBatchStock: a batch's fitment keys share their stock lookups
// (#354). Forty keys that page their lists from the first row cost each
// source one lookup, as one key does; a key at other locations and a key
// past its first page each cost one more. Every answer is, byte for byte,
// the one its key gets fitted alone.
func TestFitBatchStock(t *testing.T) {
	tr := stockTranslator(t)
	var mu sync.Mutex
	calls := map[string]int{}
	want := map[string]int{}
	for i, s := range tr.core.Stock {
		tr.core.Stock[i] = counting{s, &mu, calls}
		lookup := "yard"
		if s.Info().Holds == stock.Shelf {
			lookup = "shelf by fitment"
		}
		want[s.Info().Name+" "+lookup] = 3
	}
	keys := []string{
		`{"type":"brake-rotor-front","number":"BR-1999"}`,
		`{"type":"brake-rotor-front","number":"BR-2001"}`,
		`{"type":"brake-rotor-front","number":"BR-2002"}`,
		`{"type":"mirror","number":"MR-1001","side":"left"}`,
		`{"type":"mirror","number":"MR-1001","side":"right"}`,
		`{"type":"engine","number":"EN-57"}`,
	}
	item := func(key, more string) string {
		return `{"from":{"fitment":` + key + `},"stock":["shelf","yard"],"limit":1` + more + `}`
	}
	first, err := fit(t, tr, item(keys[1], ""))
	if err != nil || first.Next == "" {
		t.Fatalf("%s: next %q, %v; want a next page", item(keys[1], ""), first.Next, err)
	}
	var items []string
	for i := range 40 {
		items = append(items, item(keys[i%len(keys)], ""))
	}
	items = append(items, item(keys[1], `,"location":["north"]`), item(keys[1], `,"after":"`+first.Next+`"`))
	reqs := make([]contract.FitRequest, len(items))
	for i, it := range items {
		if reqs[i], err = contract.ParseFitRequest([]byte(it)); err != nil {
			t.Fatal(err)
		}
		reqs[i].Now = fitNow
	}
	clear(calls)
	fits, err := tr.FitBatch(context.Background(), reqs)
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(calls, want) {
		t.Errorf("lookups %v, want %v", calls, want)
	}
	var rows int
	for i, it := range items {
		alone, err := fit(t, tr, it)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := json.Marshal(fits[i])
		wantJSON, _ := json.Marshal(alone)
		if string(got) != string(wantJSON) {
			t.Errorf("%d %s\nin the batch %s\nalone        %s", i, it, got, wantJSON)
		}
		rows += len(fits[i].Shelf.Rows) + len(fits[i].Yard.Rows)
	}
	if rows == 0 {
		t.Error("no answer carried a stock row")
	}
}

// withStockFields is item, a fit request's JSON, with stock_fields names;
// nil leaves it out.
func withStockFields(item string, names []string) string {
	if names == nil {
		return item
	}
	b, _ := json.Marshal(names)
	return strings.TrimSuffix(item, "}") + `,"stock_fields":` + string(b) + `}`
}

// decodeJSON is v's JSON decoded, as a caller reads it.
func decodeJSON(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// eachStockList calls fn with every stock list in v, a decoded answer,
// and the key that holds it: shelf, yard or shelf_by_vehicle.
func eachStockList(v any, fn func(key string, list map[string]any)) {
	switch v := v.(type) {
	case map[string]any:
		for k, e := range v {
			if l, ok := e.(map[string]any); ok && (k == "shelf" || k == "yard" || k == "shelf_by_vehicle") {
				if _, ok := l["sources"]; ok {
					fn(k, l)
					continue
				}
			}
			eachStockList(e, fn)
		}
	case []any:
		for _, e := range v {
			eachStockList(e, fn)
		}
	}
}

// cutRows is what stock_fields names asks of v, a decoded answer: each
// stock row's fields with only the members named, nothing else of it.
func cutRows(v any, names []string) any {
	eachStockList(v, func(_ string, l map[string]any) {
		for _, r := range l["rows"].([]any) {
			fields := r.(map[string]any)["fields"].(map[string]any)
			for k := range fields {
				if !slices.Contains(names, k) {
					delete(fields, k)
				}
			}
		}
	})
	return v
}

// TestFitStockFields: stock_fields cuts every stock row's fields, on every
// kind of stock list, to the names given, and nothing else: a row keeps
// its source, id, location, how it matched and what it names; a name no
// row has is ignored; empty keeps no field (#371).
func TestFitStockFields(t *testing.T) {
	tr := stockTranslator(t)
	const ram = `"from":{"vocabulary":"catalog-models","entry":{"make":"Dodge","model":"Ram 1500"}},"model_year":2010`
	items := []string{
		`{` + ram + `,"part":{"vocabulary":"part-codes","entry":{"code":"ROT"}},"stock":["shelf","yard"]}`,
		`{` + ram + `,"part":{"spine":"part-types","id":"mirror"},"stock":["shelf"],"location":["north"]}`,
		`{"from":{"fitment":{"type":"brake-rotor-front","number":"BR-1999"}},"stock":["shelf","yard"]}`,
		`{"from":{"model":1850},"model_year":2010,"related":true,"stock":["yard"]}`,
	}
	var cut, kept int
	for _, item := range items {
		full, err := fit(t, tr, item)
		if err != nil {
			t.Fatal(err)
		}
		for _, names := range [][]string{{"stock_number", "row", "nope"}, {"nope"}, {}} {
			req := withStockFields(item, names)
			got, err := fit(t, tr, req)
			if err != nil {
				t.Fatal(err)
			}
			want := cutRows(decodeJSON(t, full), names)
			if g := decodeJSON(t, got); !reflect.DeepEqual(g, want) {
				gb, _ := json.Marshal(g)
				wb, _ := json.Marshal(want)
				t.Errorf("%s\ngot  %s\nwant %s", req, gb, wb)
			}
			eachStockList(decodeJSON(t, got), func(_ string, l map[string]any) {
				for _, r := range l["rows"].([]any) {
					row := r.(map[string]any)
					for _, k := range []string{"source", "id", "location", "matched"} {
						if _, ok := row[k]; !ok {
							t.Errorf("%s: a row without %s: %v", req, k, row)
						}
					}
					if n := len(row["fields"].(map[string]any)); len(names) > 0 && names[0] == "stock_number" && n > 0 {
						kept++
					}
				}
			})
		}
		eachStockList(decodeJSON(t, full), func(_ string, l map[string]any) {
			for _, r := range l["rows"].([]any) {
				if len(r.(map[string]any)["fields"].(map[string]any)) > 2 {
					cut++
				}
			}
		})
	}
	if cut == 0 || kept == 0 {
		t.Errorf("%d rows with fields to cut, %d keeping a field named; want some of each", cut, kept)
	}
}

// expandRows is v, a decoded FitBatchRows, as its array of fits: every
// stock list's indexes replaced by the rows of stock_rows they name.
func expandRows(t *testing.T, v any) any {
	t.Helper()
	o := v.(map[string]any)
	table := o["stock_rows"].([]any)
	fits := o["fits"]
	eachStockList(fits, func(_ string, l map[string]any) {
		rows := l["rows"].([]any)
		for i, r := range rows {
			n, ok := r.(float64)
			if !ok || n != float64(int(n)) || int(n) < 0 || int(n) >= len(table) {
				t.Fatalf("a list's row %v is no index into %d stock rows", r, len(table))
			}
			b, _ := json.Marshal(table[int(n)]) // a copy: each list's row is its own
			var row any
			if err := json.Unmarshal(b, &row); err != nil {
				t.Fatal(err)
			}
			rows[i] = row
		}
	})
	return fits
}

// TestFitBatchRows: a batch's stock rows once (#371). Expanded, the answer
// is the batch's array of fits, cuts and order included; a row that many
// keys and lists hold alike is one row of stock_rows, and one cut three
// ways three; every row is referenced; and shelf and yard rows never
// share an index.
func TestFitBatchRows(t *testing.T) {
	tr := stockTranslator(t)
	const ram = `"from":{"vocabulary":"catalog-models","entry":{"make":"Dodge","model":"Ram 1500"}},"model_year":2010`
	keys := []string{
		`{"type":"brake-rotor-front","number":"BR-1999"}`,
		`{"type":"brake-rotor-front","number":"BR-2001"}`,
		`{"type":"mirror","number":"MR-1001","side":"left"}`,
		`{"type":"mirror","number":"MR-1001","side":"right"}`,
	}
	const whole = 12 // the first items, keys asking for every field
	var items []string
	for i := range whole {
		items = append(items, `{"from":{"fitment":`+keys[i%len(keys)]+`},"stock":["shelf","yard"]}`)
	}
	items = append(items,
		`{`+ram+`,"part":{"vocabulary":"part-codes","entry":{"code":"ROT"}},"stock":["shelf","yard"],"stock_fields":["stock_number"]}`,
		`{`+ram+`,"part":{"spine":"part-types","id":"mirror"},"stock":["shelf"]}`,
		`{"from":{"model":1850},"model_year":2010,"related":true,"stock":["yard"]}`,
		`{"from":{"fitment":`+keys[1]+`},"stock":["shelf","yard"],"limit":1,"stock_fields":[]}`,
		`{`+ram+`}`,
	)
	reqs := make([]contract.FitRequest, len(items))
	for i, it := range items {
		var err error
		if reqs[i], err = contract.ParseFitRequest([]byte(it)); err != nil {
			t.Fatal(err)
		}
		reqs[i].Now = fitNow
	}
	fits, err := tr.FitBatch(context.Background(), reqs)
	if err != nil {
		t.Fatal(err)
	}
	once, err := tr.FitBatchRows(context.Background(), reqs)
	if err != nil {
		t.Fatal(err)
	}
	conforms(t, "FitBatchRows", once)
	got, want := expandRows(t, decodeJSON(t, once)), decodeJSON(t, fits)
	if !reflect.DeepEqual(got, want) {
		gb, _ := json.Marshal(got)
		wb, _ := json.Marshal(want)
		t.Errorf("expanded\n%s\nwant FitBatch's\n%s", gb, wb)
	}

	kinds := map[int]string{}
	eachStockList(decodeJSON(t, once).(map[string]any)["fits"], func(key string, l map[string]any) {
		kind := strings.TrimSuffix(key, "_by_vehicle")
		for _, r := range l["rows"].([]any) {
			i := int(r.(float64))
			if k, ok := kinds[i]; ok && k != kind {
				t.Errorf("stock row %d is in a %s list and a %s list", i, k, kind)
			}
			kinds[i] = kind
		}
	})
	if len(kinds) != len(once.StockRows) {
		t.Errorf("%d of %d stock rows referenced; want every one", len(kinds), len(once.StockRows))
	}
	// The keys' rows, each whole: one index a row, however many lists
	// hold it.
	at := map[string][]int{}
	var refs int
	for _, f := range once.Fits[:whole] {
		for _, l := range []*contract.StockRefs{f.Shelf, f.Yard} {
			for _, i := range l.Rows {
				r := once.StockRows[i]
				if id := r.Source + "/" + r.ID; !slices.Contains(at[id], i) {
					at[id] = append(at[id], i)
				}
				refs++
			}
		}
	}
	for id, is := range at {
		if len(is) != 1 {
			t.Errorf("%s is stock rows %v; want one", id, is)
		}
	}
	if refs <= len(at) {
		t.Errorf("%d references to the keys' %d rows; want rows shared", refs, len(at))
	}
	// A-1, under BR-2001: whole for the keys, cut to stock_number for the
	// Ram's rotors, with no field for the paged key.
	var a1 []string
	for _, r := range once.StockRows {
		if r.Source == "shelf-annex" && r.ID == "A-1" {
			a1 = append(a1, string(r.Fields))
		}
	}
	if want := []string{`{"stock_number":"C300","condition":"new"}`, `{"stock_number":"C300"}`, `{}`}; !slices.Equal(a1, want) {
		t.Errorf("A-1's stock rows' fields %q; want %q", a1, want)
	}
}

// failing is a stock source that is down, holding holds, failing every
// lookup with err.
type failing struct {
	name  string
	holds stock.Holds
	err   error
}

func (s failing) Info() stock.Info {
	return stock.Info{Name: s.name, Holds: s.holds, Kind: stock.KindLive}
}
func (s failing) ShelfByFitment(context.Context, [][]vocab.FitmentKey, stock.Page) ([]stock.Result, stock.Meta, error) {
	return nil, stock.Meta{}, s.err
}
func (s failing) ShelfByVehicle(context.Context, []stock.ShelfVehicle, stock.Page) ([]stock.Result, stock.Meta, error) {
	return nil, stock.Meta{}, s.err
}
func (s failing) Yard(context.Context, []stock.YardQuery, stock.Page) ([]stock.Result, stock.Meta, error) {
	return nil, stock.Meta{}, s.err
}

// TestFitStockUnavailable: a source that fails answers unavailable, with
// its *stock.Unavailable reason or, for any other error, a fixed reason
// that never shows the error's text, and no rows, in every list a fit
// gives: candidates', shelf_by_vehicle, related counts and a fitment's.
// The rest of the answer stands.
func TestFitStockUnavailable(t *testing.T) {
	tr := stockTranslator(t)
	tr.core.Stock = append(tr.core.Stock,
		failing{"live-shelf", stock.Shelf, &stock.Unavailable{Reason: "the database is not answering"}},
		failing{"live-yard", stock.Yard, errors.New("dial tcp 10.0.0.5:5432: user=stock password=secret")})
	const liveShelf = "live-shelf live unavailable (the database is not answering)"
	const liveYard = "live-yard live unavailable (the source failed)"

	f, err := fit(t, tr, `{"from":{"fitment":{"type":"brake-rotor-front","number":"BR-2001"}},"stock":["shelf","yard"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := stockText(f.Yard), "3: Y-1/vin Y-2/vin Y-3/catalog-vehicle"; got != want {
		t.Errorf("yard %s, want %s", got, want)
	}
	if got, want := sourcesText(f.Yard), "yard-stock file available, "+liveYard; got != want {
		t.Errorf("yard sources %s, want %s", got, want)
	}
	if got := sourcesText(f.Shelf); !strings.HasSuffix(got, ", "+liveShelf) || f.Shelf.Total != 4 || f.Status != contract.FitFound || len(f.Vehicles) != 2 {
		t.Errorf("shelf %s (%s), status %s, %d vehicles", stockText(f.Shelf), got, f.Status, len(f.Vehicles))
	}

	f, err = fit(t, tr, `{"from":{"vocabulary":"catalog-models","entry":{"make":"Dodge","model":"Ram 1500"}},"model_year":2010,"part":{"spine":"part-types","id":"mirror"},"stock":["shelf","yard"]}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range f.Candidates {
		if !strings.HasSuffix(sourcesText(c.Shelf), liveShelf) || !strings.HasSuffix(sourcesText(c.Yard), liveYard) {
			t.Errorf("%s %s: shelf %s, yard %s", c.Fitment.Number, c.Fitment.Side, sourcesText(c.Shelf), sourcesText(c.Yard))
		}
	}
	if got := sourcesText(f.ShelfByVehicle); !strings.HasSuffix(got, liveShelf) || f.ShelfByVehicle.Total != 2 {
		t.Errorf("shelf_by_vehicle %s (%s)", stockText(f.ShelfByVehicle), got)
	}

	f, err = fit(t, tr, `{"from":{"model":1850},"model_year":2010,"related":true,"stock":["yard"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if r := f.Related[0].Yard; r.Total != 2 || !strings.HasSuffix(sourcesText(r), liveYard) {
		t.Errorf("related yard %s (%s)", stockText(r), sourcesText(r))
	}
}

// TestInfoStock: Info lists each stock source with what it holds and its
// rows, and counts its orphans, with examples, as drift.
func TestInfoStock(t *testing.T) {
	info := stockTranslator(t).Info()
	conforms(t, "Info", info)
	var got []string
	for _, v := range info.Vocabularies {
		if v.Kind != string(vocab.KindStock) {
			continue
		}
		s := fmt.Sprintf("%s %s %s %q %d rows", v.Name, v.Source, v.Catalog, v.Release, *v.Rows)
		for _, d := range v.Drift {
			s += fmt.Sprintf(", %d %s at line %d", d.Count, d.Kind, d.Examples[0].Line)
		}
		got = append(got, s)
	}
	want := []string{
		`shelf-annex shelf parts-catalog "" 2 rows`,
		`shelf-stock shelf parts-catalog "7.01" 5 rows`,
		`yard-stock yard  "" 4 rows`,
		`shelf-orphans shelf parts-catalog "7.02" 2 rows, 1 orphan at line 2`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("stock in Info\n%q\nwant\n%q", got, want)
	}
}

// liveFiles are testdata/vocab and the synthetic catalog with the
// connector files of testdata/vocab/stock/live.
func liveFiles(t *testing.T) []*vocab.File {
	t.Helper()
	files, err := vocab.Open("../../testdata/vocab", "../../testdata/vocab/fitment", "../../testdata/vocab/stock/live")
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// stockStatus writes Info's stock entries as "name connector relation
// status (reason)".
func stockStatus(info contract.Info) []string {
	var out []string
	for _, v := range info.Vocabularies {
		if v.StockCounts != nil {
			s := strings.Join([]string{v.Name, v.Connector, v.Relation, v.Status}, " ")
			if v.Reason != "" {
				s += " (" + v.Reason + ")"
			}
			out = append(out, s)
		}
	}
	return out
}

// TestConnectorWithoutDriver: NewTranslator, as the C library calls it,
// loads a connector's file as a live source that is always unavailable,
// saying why, in fit's blocks and in Info.
func TestConnectorWithoutDriver(t *testing.T) {
	tr, err := NewTranslator(embeddedDecoder(t), liveFiles(t)...)
	if err != nil {
		t.Fatal(err)
	}
	const why = "this build reads stock files only: a connector needs the nitrovin binary"
	f, err := fit(t, tr, `{"from":{"fitment":{"type":"brake-rotor-front","number":"BR-2001"}},"stock":["shelf","yard"]}`)
	if err != nil {
		t.Fatal(err)
	}
	want := "shelf-annex live unavailable (" + why + "), shelf-stock live unavailable (" + why + ")"
	if got := sourcesText(f.Shelf); got != want {
		t.Errorf("shelf sources %s\nwant %s", got, want)
	}
	if s := f.Yard.Sources[0]; s.Connector != "postgres" || s.Relation != "stock.yard_stock" {
		t.Errorf("the yard's block %+v", s)
	}
	info := tr.Info()
	conforms(t, "Info", info)
	wantInfo := []string{
		"shelf-annex postgres stock.shelf_annex unavailable (" + why + ")",
		"shelf-stock postgres stock.shelf_stock unavailable (" + why + ")",
		"yard-stock postgres stock.yard_stock unavailable (" + why + ")",
	}
	if got := stockStatus(info); !slices.Equal(got, wantInfo) {
		t.Errorf("Info's stock\n%q\nwant\n%q", got, wantInfo)
	}
}

// liveFake is a live source whose health is set by the test, counting its
// Close calls.
type liveFake struct {
	blocking
	health stock.Health
	closed int
}

func (l *liveFake) Health() stock.Health { return l.health }
func (l *liveFake) Close()               { l.closed++ }

// TestConnectorStatus: NewTranslatorWith hands each connector's file to
// its Connect with the catalog its vehicles are of and that catalog's
// vehicle vocabulary; Info reads each source's health on every call:
// unchecked, available, then unavailable with its reason; and Close
// closes every connector's source, as often as it is called.
func TestConnectorStatus(t *testing.T) {
	var fakes []*liveFake
	var got []string
	connect := func(d stock.Declared) stock.Live {
		got = append(got, d.File.Name+" "+d.Catalog.Name+" "+d.Vehicles.Name)
		l := &liveFake{blocking: blocking{name: d.File.Name, holds: d.File.Holds}}
		fakes = append(fakes, l)
		return l
	}
	tr, err := NewTranslatorWith(embeddedDecoder(t), TranslatorConfig{Connect: connect}, liveFiles(t)...)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"shelf-annex parts-catalog catalog-models", "shelf-stock parts-catalog catalog-models", "yard-stock parts-catalog catalog-models"}; !slices.Equal(got, want) {
		t.Errorf("connect got %q, want %q", got, want)
	}
	yard := fakes[2]
	for _, tc := range []struct {
		health stock.Health
		want   string
	}{
		{stock.Health{}, "yard-stock postgres stock.yard_stock unchecked"},
		{stock.Health{Checked: true}, "yard-stock postgres stock.yard_stock available"},
		{stock.Health{Checked: true, Down: &stock.Unavailable{Reason: "the database is unreachable"}}, "yard-stock postgres stock.yard_stock unavailable (the database is unreachable)"},
	} {
		yard.health = tc.health
		for _, info := range []contract.Info{tr.Info(), tr.LoadInfo()} {
			conforms(t, "Info", info)
			if s := stockStatus(info); s[2] != tc.want {
				t.Errorf("Info's yard %q, want %q", s[2], tc.want)
			}
		}
	}
	tr.Close()
	tr.Close()
	for _, l := range fakes {
		if l.closed != 2 {
			t.Errorf("%s closed %d times by two Closes; want 2, each passed on", l.name, l.closed)
		}
	}
	files, err := NewTranslator(embeddedDecoder(t), liveFiles(t)...)
	if err != nil {
		t.Fatal(err)
	}
	files.Close() // stock.Down sources: nothing to release
}

// blocking is a stock source that never answers by itself: it waits for
// its lookup's ctx, or, when deaf, for nothing it is given.
type blocking struct {
	name  string
	holds stock.Holds
	deaf  chan struct{} // nil: the source honours ctx
}

func (s blocking) Info() stock.Info {
	return stock.Info{Name: s.name, Holds: s.holds, Kind: stock.KindLive}
}
func (s blocking) wait(ctx context.Context) ([]stock.Result, stock.Meta, error) {
	if s.deaf != nil {
		<-s.deaf
		return nil, stock.Meta{}, errors.New("released")
	}
	<-ctx.Done()
	return nil, stock.Meta{}, ctx.Err()
}
func (s blocking) ShelfByFitment(ctx context.Context, _ [][]vocab.FitmentKey, _ stock.Page) ([]stock.Result, stock.Meta, error) {
	return s.wait(ctx)
}
func (s blocking) ShelfByVehicle(ctx context.Context, _ []stock.ShelfVehicle, _ stock.Page) ([]stock.Result, stock.Meta, error) {
	return s.wait(ctx)
}
func (s blocking) Yard(ctx context.Context, _ []stock.YardQuery, _ stock.Page) ([]stock.Result, stock.Meta, error) {
	return s.wait(ctx)
}

// TestFitStockCancelled: a stock lookup still running when the fit's
// context ends answers its source unavailable, with the fixed reason
// contract.StockCancelled, whether the source stops with ctx or ignores
// it; the fit returns, and the file sources, which answer from memory,
// still give their rows. A context already ended asks no live source.
func TestFitStockCancelled(t *testing.T) {
	tr := stockTranslator(t)
	deaf := make(chan struct{})
	defer close(deaf)
	tr.core.Stock = append(tr.core.Stock, blocking{"slow-shelf", stock.Shelf, nil}, blocking{"deaf-yard", stock.Yard, deaf})
	req := contract.FitRequest{From: contract.FitFrom{Fitment: &contract.FitmentKey{Type: "brake-rotor-front", Number: "BR-2001"}}, Stock: []string{"shelf", "yard"}, Now: fitNow}
	const cancelled = "unavailable (" + contract.StockCancelled + ")"

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	f, err := tr.Fit(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("the fit took %s after its deadline", d)
	}
	if got := sourcesText(f.Shelf); !strings.HasSuffix(got, "slow-shelf live "+cancelled) || f.Shelf.Total != 4 {
		t.Errorf("shelf %s (%s)", stockText(f.Shelf), got)
	}
	if got := sourcesText(f.Yard); !strings.HasSuffix(got, "deaf-yard live "+cancelled) || f.Yard.Total != 3 {
		t.Errorf("yard %s (%s)", stockText(f.Yard), got)
	}

	done, stop := context.WithCancel(context.Background())
	stop()
	fits, err := tr.FitBatch(done, []contract.FitRequest{req})
	if err != nil {
		t.Fatal(err)
	}
	if got := sourcesText(fits[0].Yard); got != "yard-stock file available, deaf-yard live "+cancelled || fits[0].Yard.Total != 3 || fits[0].Status != contract.FitFound {
		t.Errorf("an ended context: yard %s, status %s", got, fits[0].Status)
	}
}

// rendezvous is a live stock source whose lookups answer empty only once
// every source sharing its group has started one: the lookups of one fit
// must run at once for any of them to answer.
type rendezvous struct {
	name  string
	holds stock.Holds
	group *sync.WaitGroup
}

func (s rendezvous) Info() stock.Info {
	return stock.Info{Name: s.name, Holds: s.holds, Kind: stock.KindLive}
}
func (s rendezvous) meet(ctx context.Context, n int) ([]stock.Result, stock.Meta, error) {
	s.group.Done()
	met := make(chan struct{})
	go func() { s.group.Wait(); close(met) }()
	select {
	case <-met:
		out := make([]stock.Result, n)
		for i := range out {
			out[i].Rows = []stock.Row{}
		}
		return out, stock.Meta{AsOf: "2026-11-12"}, nil
	case <-ctx.Done():
		return nil, stock.Meta{}, ctx.Err()
	}
}
func (s rendezvous) ShelfByFitment(ctx context.Context, lists [][]vocab.FitmentKey, _ stock.Page) ([]stock.Result, stock.Meta, error) {
	return s.meet(ctx, len(lists))
}
func (s rendezvous) ShelfByVehicle(ctx context.Context, lists []stock.ShelfVehicle, _ stock.Page) ([]stock.Result, stock.Meta, error) {
	return s.meet(ctx, len(lists))
}
func (s rendezvous) Yard(ctx context.Context, lists []stock.YardQuery, _ stock.Page) ([]stock.Result, stock.Meta, error) {
	return s.meet(ctx, len(lists))
}

// TestFitStockConcurrent: one answer's lookups of different list kinds run
// at once: a candidate's shelf, its yard and shelf_by_vehicle each answer
// only once all three have started, and do within the deadline.
func TestFitStockConcurrent(t *testing.T) {
	tr := stockTranslator(t)
	var group sync.WaitGroup
	group.Add(3) // the candidates' shelf, their yard, and shelf_by_vehicle
	tr.core.Stock = append(tr.core.Stock, rendezvous{"live-shelf", stock.Shelf, &group}, rendezvous{"live-yard", stock.Yard, &group})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	f, err := tr.Fit(ctx, contract.FitRequest{
		From: contract.FitFrom{Vocabulary: "catalog-models", Entry: map[string]string{"make": "Dodge", "model": "Ram 1500"}}, ModelYear: contract.Value[int32](2010),
		Part: &contract.FitPart{Spine: "part-types", ID: "brake-rotor-front"}, Stock: []string{"shelf", "yard"}, Now: fitNow,
	})
	if err != nil {
		t.Fatal(err)
	}
	c := f.Candidates[0]
	for _, l := range []*contract.StockList{c.Shelf, c.Yard, f.ShelfByVehicle} {
		if got := sourcesText(l); !strings.HasSuffix(got, " live available") {
			t.Errorf("sources %s: the lookups did not run at once", got)
		}
	}
}

// panicking is a stock source with a fault: every lookup panics, once
// release (when not nil) is closed, closing panicked as it does.
type panicking struct {
	value    any // what it panics with; nil: a string naming it
	name     string
	holds    stock.Holds
	kind     stock.Kind
	release  chan struct{}
	panicked chan struct{}
}

func (s panicking) Info() stock.Info {
	return stock.Info{Name: s.name, Holds: s.holds, Kind: s.kind}
}
func (s panicking) fail() ([]stock.Result, stock.Meta, error) {
	if s.release != nil {
		<-s.release
		defer close(s.panicked)
	}
	if s.value != nil {
		panic(s.value)
	}
	panic("a fault in " + s.name)
}

// damagedCatalog is an error like a damaged compiled catalog's reader
// panics with.
type damagedCatalog struct{ path string }

func (e *damagedCatalog) Error() string {
	return "compiled catalog " + e.path + ": a string out of range: the file is damaged"
}
func (s panicking) ShelfByFitment(context.Context, [][]vocab.FitmentKey, stock.Page) ([]stock.Result, stock.Meta, error) {
	return s.fail()
}
func (s panicking) ShelfByVehicle(context.Context, []stock.ShelfVehicle, stock.Page) ([]stock.Result, stock.Meta, error) {
	return s.fail()
}
func (s panicking) Yard(context.Context, []stock.YardQuery, stock.Page) ([]stock.Result, stock.Meta, error) {
	return s.fail()
}

// TestFitStockPanic: a panic in a stock lookup, on a goroutine of its own,
// reaches Fit's caller, who can recover it, as a *contract.LookupPanic
// with the original value and the stack it happened on, whether the
// source is a file, looked up on concurrently's goroutine, or live, on
// the lookup's own. A live lookup that panics after the fit stopped
// waiting for it is dropped, and the process lives on.
func TestFitStockPanic(t *testing.T) {
	req := contract.FitRequest{
		From: contract.FitFrom{Vocabulary: "catalog-models", Entry: map[string]string{"make": "Dodge", "model": "Ram 1500"}}, ModelYear: contract.Value[int32](2010),
		Part: &contract.FitPart{Spine: "part-types", ID: "brake-rotor-front"}, Stock: []string{"shelf", "yard"}, Now: fitNow,
	}
	for _, kind := range []stock.Kind{stock.KindFile, stock.KindLive} {
		t.Run(string(kind), func(t *testing.T) {
			tr := stockTranslator(t)
			tr.core.Stock = append(tr.core.Stock, panicking{name: "faulty-yard", holds: stock.Yard, kind: kind})
			var v any
			func() {
				defer func() { v = recover() }()
				tr.Fit(context.Background(), req)
			}()
			p, ok := v.(*contract.LookupPanic)
			if !ok || p.Value != "a fault in faulty-yard" || fmt.Sprint(p) != "a fault in faulty-yard" || !strings.Contains(string(p.Stack), "panicking.fail") {
				t.Fatalf("recovered %#v; want a *contract.LookupPanic of the source's panic, with its stack", v)
			}
		})
	}

	// An error it panics with is found through the LookupPanic.
	tr := stockTranslator(t)
	damaged := &damagedCatalog{"catalog.compiled"}
	tr.core.Stock = append(tr.core.Stock, panicking{value: damaged, name: "damaged-yard", holds: stock.Yard, kind: stock.KindLive})
	var v any
	func() {
		defer func() { v = recover() }()
		tr.Fit(context.Background(), req)
	}()
	err, _ := v.(error)
	var found *damagedCatalog
	if !errors.As(err, &found) || found != damaged || !errors.Is(err, damaged) || err.Error() != damaged.Error() {
		t.Errorf("recovered %#v; want a *contract.LookupPanic unwrapping to the damaged catalog's error", v)
	}
	if p, _ := v.(*contract.LookupPanic); p == nil || errors.Unwrap(&contract.LookupPanic{Value: "not an error"}) != nil {
		t.Errorf("a LookupPanic of a string unwraps to an error")
	}

	tr = stockTranslator(t)
	late := panicking{name: "late-yard", holds: stock.Yard, kind: stock.KindLive, release: make(chan struct{}), panicked: make(chan struct{})}
	tr.core.Stock = append(tr.core.Stock, late)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	f, err := tr.Fit(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if got := sourcesText(f.Candidates[0].Yard); !strings.HasSuffix(got, "late-yard live unavailable ("+contract.StockCancelled+")") {
		t.Errorf("the late yard's block: %s", got)
	}
	close(late.release)
	<-late.panicked
	time.Sleep(100 * time.Millisecond) // a panic unrecovered would end the test binary here
}
