package nitrovin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/stock"
)

// ranked is a live yard that names vehicles, in the order a Stocker
// gives them, or fails with err.
type ranked struct {
	failing
	vehicles []stock.StockedVehicle
	asOf     string
}

func (s ranked) Stocked(_ context.Context, q stock.StockedQuery) ([]stock.StockedVehicle, stock.Meta, error) {
	if s.err != nil {
		return nil, stock.Meta{}, s.err
	}
	return s.vehicles[:min(q.Limit, len(s.vehicles))], stock.Meta{AsOf: s.asOf}, nil
}

// waiting is a live yard whose ranking never answers before its context
// ends.
type waiting struct{ failing }

func (s waiting) Stocked(ctx context.Context, _ stock.StockedQuery) ([]stock.StockedVehicle, stock.Meta, error) {
	<-ctx.Done()
	return nil, stock.Meta{}, ctx.Err()
}

func stockedText(s contract.Stocked) string {
	var out []string
	for _, v := range s.Vehicles {
		var names []string
		for _, e := range v.Spine {
			name := e.Name
			if e.Drift {
				name = fmt.Sprintf("drift %d", e.Model)
			}
			names = append(names, name)
		}
		out = append(out, fmt.Sprintf("%s %d %s x%d %s", v.VIN, v.ModelYear, strings.Join(names, " "), v.Yard, v.Source))
	}
	return strings.Join(out, "; ")
}

func stockedSources(s contract.Stocked) string {
	return sourcesText(&contract.StockList{Sources: s.Sources})
}

// TestStocked: stocked answers the vehicles the loaded yards hold most
// of. A yard file names its own, each with its vPIC make and model as the
// data month names them. Live yards are merged with it: a model two yards
// hold is kept from the one with more of it, the most rows come first,
// then the lower model id, and limit cuts the merged list. A yard that
// cannot name vehicles is available and gives none; one that fails is
// unavailable with its reason, never its error's text; one that has not
// answered when the context ends says so. A shelf is no yard: it has no
// block. A model the data month lacks is served alone, marked as drift.
// A request out of range is refused by pointer.
func TestStocked(t *testing.T) {
	tr := stockTranslator(t)
	ctx := context.Background()
	s, err := tr.Stocked(ctx, contract.StockedRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := stockedText(s), "1D7RV1GT5AS123456 2010 Dodge Ram x2 yard-stock"; got != want {
		t.Errorf("the fixture's yard: %s; want %s", got, want)
	}
	if got, want := stockedSources(s), "yard-stock file available"; got != want || s.DataMonth == "" || s.DecoderVersion == "" {
		t.Errorf("sources %s, data month %q, decoder %q; want %s", got, s.DataMonth, s.DecoderVersion, want)
	}
	if b, err := json.Marshal(s.Vehicles[0].Spine); err != nil || string(b) != `[{"make":476,"name":"Dodge"},{"model":1938,"name":"Ram"}]` {
		t.Errorf("spine %s, %v", b, err)
	}
	if s, err := tr.Stocked(ctx, contract.StockedRequest{Location: []string{"nowhere"}}); err != nil || len(s.Vehicles) != 0 || s.Vehicles == nil {
		t.Errorf("at a location with no yard row: %+v, %v; want an empty list", s.Vehicles, err)
	}

	live := func(name string) failing { return failing{name: name, holds: stock.Yard} }
	tr.core.Stock = append(tr.core.Stock,
		ranked{failing: live("live-a"), asOf: "2026-11-20", vehicles: []stock.StockedVehicle{
			{Model: 1850, Year: 2012, Rows: 9, VIN: "VINA1850"}, {Model: 1801, Year: 2015, Rows: 5, VIN: "VINA1801"}, {Model: 1938, Year: 2011, Rows: 5, VIN: "VINA1938"}, {Model: 99999999, Year: 2001, Rows: 1, VIN: "VINDRIFT"}}},
		ranked{failing: live("live-b"), vehicles: []stock.StockedVehicle{{Model: 1850, Year: 2014, Rows: 4, VIN: "VINB1850"}}},
		live("no-ranking"),
		ranked{failing: failing{"down", stock.Yard, &stock.Unavailable{Reason: "the database is not answering"}}},
		ranked{failing: failing{"broken", stock.Yard, errors.New("dial tcp 10.0.0.5:5432: user=stock password=secret")}},
		failing{"live-shelf", stock.Shelf, errors.New("never asked")})
	s, err = tr.Stocked(ctx, contract.StockedRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := stockedText(s), "VINA1850 2012 Chevrolet Silverado x9 live-a; VINA1801 2015 Ford F-150 x5 live-a; VINA1938 2011 Dodge Ram x5 live-a; VINDRIFT 2001 drift 99999999 x1 live-a"; got != want {
		t.Errorf("merged:\n%s\nwant\n%s", got, want)
	}
	if got, want := stockedSources(s), "yard-stock file available, live-a live available, live-b live available, no-ranking live available, down live unavailable (the database is not answering), broken live unavailable (the source failed)"; got != want {
		t.Errorf("sources:\n%s\nwant\n%s", got, want)
	}
	if s.Sources[1].AsOf != "2026-11-20" || s.Sources[2].AsOf != "" {
		t.Errorf("as_of %q and %q; want live-a's alone", s.Sources[1].AsOf, s.Sources[2].AsOf)
	}
	if s, err := tr.Stocked(ctx, contract.StockedRequest{Limit: 2}); err != nil || len(s.Vehicles) != 2 || s.Vehicles[1].VIN != "VINA1801" {
		t.Errorf("limit 2: %s, %v", stockedText(s), err)
	}

	tr.core.Stock = append(tr.core.Stock[:1:1], waiting{live("slow")})
	done, cancel := context.WithCancel(ctx)
	cancel()
	if s, err := tr.Stocked(done, contract.StockedRequest{}); err != nil || !strings.HasSuffix(stockedSources(s), "slow live unavailable ("+contract.StockCancelled+")") {
		t.Errorf("a context that ended: %s, %v", stockedSources(s), err)
	}

	for _, c := range []struct {
		req     contract.StockedRequest
		pointer string
	}{
		{contract.StockedRequest{Limit: contract.MaxStockedLimit + 1}, "/limit"},
		{contract.StockedRequest{Limit: -1}, "/limit"},
		{contract.StockedRequest{Location: []string{"north", ""}}, "/location/1"},
	} {
		_, err := tr.Stocked(ctx, c.req)
		if p := contract.AsProblem(err); err == nil || p.Type != contract.InvalidRequest || p.Pointer != c.pointer {
			t.Errorf("%+v: %v; want an invalid request at %s", c.req, err, c.pointer)
		}
	}
}
