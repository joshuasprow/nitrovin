package stock

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// TestYardRoutes: a yard row takes its first route that gives a vPIC model
// and a year. A VIN that decodes a model and no year falls to the row's
// catalog vehicle and year when it names one, and otherwise matches the
// model in every year; a VIN that decodes no model falls to the vehicle.
func TestYardRoutes(t *testing.T) {
	dir := t.TempDir()
	data := `{"format":1,"kind":"stock","name":"y","version":"1","source":"yard","as_of":"2026-11-01"}
{"id":"no-year-vehicle","location":"a","vin":"NOYEAR","vehicle":{"make":"Chevrolet","model":"Silverado 1500"},"year":2010}
{"id":"no-year","location":"a","vin":"NOYEAR"}
{"id":"no-model","location":"b","vin":"NOMODEL","vehicle":{"make":"Chevrolet","model":"Silverado 1500"},"year":2011}
{"id":"decoded","location":"b","decoded":{"make":467,"model":1850,"year":2012},"vin":"NOMODEL"}
`
	if err := os.WriteFile(filepath.Join(dir, "y.jsonl"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := vocab.Open("../../testdata/vocab/fitment", dir)
	if err != nil {
		t.Fatal(err)
	}
	var f *vocab.File
	for _, g := range files {
		if g.Name == "y" {
			f = g
		}
	}
	src := NewFile(f, func(vin string) (int32, *int, bool) {
		if vin == "NOYEAR" {
			return 1850, nil, true
		}
		return 0, nil, false
	})
	silverado := vocab.Vehicle{Entry: map[string]string{"make": "Chevrolet", "model": "Silverado 1500"}}
	ids := func(q YardQuery, p Page) []string {
		t.Helper()
		rs, meta, err := src.Yard(context.Background(), []YardQuery{q}, p)
		if err != nil || len(rs) != 1 || meta.AsOf != "2026-11-01" || meta.Release != "" {
			t.Fatalf("Yard: %v, %d results, %+v", err, len(rs), meta)
		}
		var out []string
		for _, r := range rs[0].Rows {
			out = append(out, r.ID+"/"+string(r.Matched))
		}
		return out
	}
	y := func(n int) *int { return &n }
	if got, want := ids(YardQuery{Models: []ModelYears{{Model: 1850, First: y(2012), Last: y(2012)}}}, Page{Limit: 10}), []string{"no-year/vin", "decoded/vin"}; !slices.Equal(got, want) {
		t.Errorf("model 1850 in 2012: %q, want %q", got, want)
	}
	if got, want := ids(YardQuery{Vehicles: []VehicleYears{{Vehicle: silverado}}}, Page{Limit: 10}), []string{"no-year-vehicle/catalog-vehicle", "no-model/catalog-vehicle"}; !slices.Equal(got, want) {
		t.Errorf("the Silverado by vehicle: %q, want %q", got, want)
	}
	both := YardQuery{Models: []ModelYears{{Model: 1850}}, Vehicles: []VehicleYears{{Vehicle: silverado}}}
	if got, want := ids(both, Page{Location: []string{"b"}, Offset: 1, Limit: 10}), []string{"decoded/vin"}; !slices.Equal(got, want) {
		t.Errorf("both routes at b, after the first: %q, want %q", got, want)
	}
	if rs, _, _ := src.ShelfByFitment(context.Background(), [][]vocab.FitmentKey{{{Type: "mirror", Number: "MR-1001"}}}, Page{Limit: 10}); len(rs) != 1 || rs[0].Total != 0 || rs[0].Rows == nil {
		t.Errorf("a yard's shelf lookup: %+v", rs)
	}
}

// TestStocked: a yard file names its best-stocked vehicles, each model
// once in its best year, the most rows first, each with the least VIN of
// its rows. A row with no year, a row
// reached by its catalog vehicle and a vehicle with no VIN among its rows
// are left out; a location keeps its own rows.
func TestStocked(t *testing.T) {
	dir := t.TempDir()
	data := `{"format":1,"kind":"stock","name":"y","version":"1","source":"yard","as_of":"2026-11-01"}
{"id":"1","location":"a","vin":"B1850Y2012"}
{"id":"2","location":"b","vin":"A1850Y2012"}
{"id":"3","location":"a","decoded":{"make":467,"model":1850,"year":2012}}
{"id":"4","location":"a","vin":"C1850Y2013"}
{"id":"5","location":"a","vin":"D1851Y2013"}
{"id":"6","location":"b","vin":"E1851Y2013"}
{"id":"7","location":"a","vin":"NOYEAR"}
{"id":"8","location":"a","decoded":{"make":467,"model":1852,"year":2014}}
{"id":"9","location":"a","vehicle":{"make":"Chevrolet","model":"Silverado 1500"},"year":2010}
`
	if err := os.WriteFile(filepath.Join(dir, "y.jsonl"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := vocab.Open("../../testdata/vocab/fitment", dir)
	if err != nil {
		t.Fatal(err)
	}
	var f *vocab.File
	for _, g := range files {
		if g.Name == "y" {
			f = g
		}
	}
	decoded := map[string][2]int{"B1850Y2012": {1850, 2012}, "A1850Y2012": {1850, 2012}, "C1850Y2013": {1850, 2013}, "D1851Y2013": {1851, 2013}, "E1851Y2013": {1851, 2013}, "NOYEAR": {1850, 0}}
	var src Stocker = NewFile(f, func(vin string) (int32, *int, bool) {
		d, ok := decoded[vin]
		if !ok {
			return 0, nil, false
		}
		if d[1] == 0 {
			return int32(d[0]), nil, true
		}
		return int32(d[0]), &d[1], true
	})
	for _, c := range []struct {
		q    StockedQuery
		want []StockedVehicle
	}{
		{StockedQuery{Limit: 10}, []StockedVehicle{{1850, 2012, 3, "A1850Y2012"}, {1851, 2013, 2, "D1851Y2013"}}},
		{StockedQuery{Limit: 1}, []StockedVehicle{{1850, 2012, 3, "A1850Y2012"}}},
		{StockedQuery{Limit: 10, Location: []string{"b"}}, []StockedVehicle{{1850, 2012, 1, "A1850Y2012"}, {1851, 2013, 1, "E1851Y2013"}}},
		{StockedQuery{}, []StockedVehicle{}},
	} {
		got, meta, err := src.Stocked(context.Background(), c.q)
		if err != nil || meta.AsOf != "2026-11-01" || !slices.Equal(got, c.want) {
			t.Errorf("Stocked(%+v): %+v, %+v, %v; want %+v", c.q, got, meta, err, c.want)
		}
	}
}
