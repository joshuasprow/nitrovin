package contract

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"

	"github.com/joshuasprow/nitrovin/pkg/stock"
)

// Stocked limits from docs/api/README.md.
const (
	// DefaultStockedLimit is how many vehicles a stocked request with no
	// limit asks for.
	DefaultStockedLimit = 10
	// MaxStockedLimit caps a stocked request's limit.
	MaxStockedLimit = 100
)

// StockedRequest is one stocked request, the same on every surface
// (ADR 0028): the vehicles the loaded yards hold most of, for a caller
// that wants a vehicle with stock before it has one to ask about.
type StockedRequest struct {
	// Location keeps the yard rows at one of these locations; none keeps
	// every location.
	Location []string
	// Limit is how many vehicles, 1 to MaxStockedLimit; 0 means
	// DefaultStockedLimit.
	Limit int
}

// stockedFields are StockedRequest's properties.
var stockedFields = []string{"location", "limit"}

// Validate checks r against the request rules a Go caller can break. The
// error is a *Problem.
func (r StockedRequest) Validate() error {
	if p := r.validate(); p != nil {
		return p
	}
	return nil
}

func (r StockedRequest) validate() *Problem {
	for i, l := range r.Location {
		if l == "" {
			return NewProblem(InvalidRequest, "/location/"+strconv.Itoa(i), "a location is not empty")
		}
	}
	if r.Limit < 0 || r.Limit > MaxStockedLimit {
		return NewProblem(InvalidRequest, "/limit", fmt.Sprintf("limit must be an integer from 1 to %d", MaxStockedLimit))
	}
	return nil
}

// ParseStockedRequest parses one stocked request object: a StockedRequest
// of decode.schema.json. Unknown fields are refused. The error is a
// *Problem.
func ParseStockedRequest(data []byte) (StockedRequest, error) {
	if !json.Valid(data) {
		return StockedRequest{}, NewProblem(MalformedJSON, "", "the request is not JSON")
	}
	fields, p := objectFields(data, "", "a request", stockedFields)
	if p != nil {
		return StockedRequest{}, p
	}
	r, p := readStocked(fields)
	if p != nil {
		return StockedRequest{}, p
	}
	return r, nil
}

// readStocked reads a stocked request's fields, unknown ones already
// refused.
func readStocked(fields map[string]json.RawMessage) (StockedRequest, *Problem) {
	var r StockedRequest
	if raw, ok := fields["location"]; ok {
		var items []json.RawMessage
		if firstByte(raw) != '[' || json.Unmarshal(raw, &items) != nil || len(items) == 0 {
			return r, NewProblem(InvalidRequest, "/location", "location must be a non-empty array of strings")
		}
		r.Location = make([]string, len(items))
		for i, item := range items {
			if !jsonString(item, &r.Location[i]) {
				return r, NewProblem(InvalidRequest, "/location/"+strconv.Itoa(i), "location must be an array of strings")
			}
		}
	}
	if raw, ok := fields["limit"]; ok {
		n, ok := jsonInt32(raw)
		if !ok || n < 1 || n > MaxStockedLimit {
			return r, NewProblem(InvalidRequest, "/limit", fmt.Sprintf("limit must be an integer from 1 to %d", MaxStockedLimit))
		}
		r.Limit = int(n)
	}
	return r, r.validate()
}

// ParseStockedQuery parses GET /v1/stocked: location, repeated, one value
// each, and limit, at most once; an unknown parameter is refused. As in
// ParseListQuery, the parameters are read as the JSON request they spell.
// The error is a *Problem.
func ParseStockedQuery(query url.Values) (StockedRequest, error) {
	req := map[string]json.RawMessage{}
	for _, name := range slices.Sorted(maps.Keys(query)) {
		vs := query[name]
		switch name {
		case "location":
			items := make([]json.RawMessage, len(vs))
			for i, v := range vs {
				items[i] = queryString(v)
			}
			req[name] = marshalRaw(items)
		case "limit":
			if len(vs) != 1 {
				return StockedRequest{}, NewProblem(InvalidRequest, "/limit", "limit is given more than once")
			}
			req[name] = queryNumber(vs[0])
		default:
			return StockedRequest{}, NewProblem(InvalidRequest, queryPointer(name, "where."), "unknown query parameter "+strconv.Quote(name))
		}
	}
	return ParseStockedRequest(marshalRaw(req))
}

// Stocked is the answer to a StockedRequest (ADR 0028): the vehicles the
// loaded yards hold most of, each vPIC model once, the most yard rows
// first. No yard loaded, or none that can name a vehicle, is never a
// refusal: Vehicles is empty, and Sources says what each yard answered.
type Stocked struct {
	Vehicles []StockedVehicle `json:"vehicles"`
	// Sources are the loaded yard sources' blocks, in load order: every
	// one, whether or not it gave a vehicle.
	Sources        []StockSource `json:"sources"`
	DataMonth      string        `json:"data_month"`
	DecoderVersion string        `json:"decoder_version"`
}

// StockedVehicle is one vehicle a yard holds: a VIN of the yard's, to ask
// the rest with; the vPIC make and model its decode gives, as the data
// month names them, and its model year; and how many of the source's rows
// are of that model and year.
type StockedVehicle struct {
	VIN string `json:"vin"`
	// Spine is the vehicle's vPIC make, then its model. A model the data
	// month lacks comes alone, marked Drift.
	Spine     []SpineEntry `json:"spine"`
	ModelYear int32        `json:"model_year"`
	// Yard is how many yard rows of Source are of the model and year.
	Yard   int    `json:"yard"`
	Source string `json:"source"`
}

// Stocked answers one stocked request (ADR 0028) from the loaded yard
// sources that can name their vehicles (stock.Stocker): each asked once,
// their answers merged, a model two yards hold kept from the one with more
// of it, the most rows first, then by model. A yard that cannot name
// vehicles is available and gives none; one that failed is unavailable,
// with its reason. ctx bounds the lookups, as a fit's. The error is a
// *Problem: an InvalidRequest for a request Validate refuses.
func (f Fitter) Stocked(ctx context.Context, req StockedRequest) (Stocked, error) {
	if p := req.validate(); p != nil {
		return Stocked{}, p
	}
	limit := req.Limit
	if limit == 0 {
		limit = DefaultStockedLimit
	}
	out := Stocked{Vehicles: []StockedVehicle{}, Sources: []StockSource{}, DataMonth: f.DataMonth, DecoderVersion: f.DecoderVersion}
	best := map[int32]StockedVehicle{}
	for _, src := range f.Stock {
		info := src.Info()
		if info.Holds != stock.Yard {
			continue
		}
		block := StockSource{Name: info.Name, Kind: string(info.Kind), Connector: info.Connector, Relation: info.Relation, Status: StockAvailable}
		stocker, ok := src.(stock.Stocker)
		if !ok {
			out.Sources = append(out.Sources, block)
			continue
		}
		var found []stock.StockedVehicle
		_, meta, err := lookup(ctx, src, stock.Page{Location: req.Location, Limit: limit}, func(ctx context.Context, _ stock.Source, p stock.Page) ([]stock.Result, stock.Meta, error) {
			vs, meta, err := stocker.Stocked(ctx, stock.StockedQuery{Location: p.Location, Limit: p.Limit})
			found = vs
			return nil, meta, err
		})
		switch {
		case errors.Is(err, errCancelled):
			block.Status, block.Reason = StockUnavailable, StockCancelled
		case err != nil:
			block.Status, block.Reason = StockUnavailable, StockFailed
			if u, ok := errors.AsType[*stock.Unavailable](err); ok {
				block.Reason = u.Reason
			}
		default:
			block.AsOf = meta.AsOf
			for _, v := range found {
				if b, ok := best[v.Model]; !ok || v.Rows > b.Yard {
					best[v.Model] = StockedVehicle{VIN: v.VIN, Spine: f.vpicSpine(v.Model), ModelYear: int32(v.Year), Yard: v.Rows, Source: info.Name}
				}
			}
		}
		out.Sources = append(out.Sources, block)
	}
	models := slices.SortedFunc(maps.Keys(best), func(a, b int32) int {
		return cmp.Or(cmp.Compare(best[b].Yard, best[a].Yard), cmp.Compare(a, b))
	})
	for _, m := range models[:min(limit, len(models))] {
		out.Vehicles = append(out.Vehicles, best[m])
	}
	return out, nil
}

// vpicSpine is vPIC model id's make, then the model, as the data month
// names them; the model alone, marked Drift, when the month lacks it.
func (f Fitter) vpicSpine(model int32) []SpineEntry {
	mk, name, ok := f.Set.Month().Model(model)
	if !ok {
		return []SpineEntry{{SpineRef: SpineRef{Model: model}, Drift: true}}
	}
	makeName, _ := f.Set.Month().Make(mk)
	return []SpineEntry{{SpineRef: SpineRef{Make: mk}, Name: makeName}, {SpineRef: SpineRef{Model: model}, Name: name}}
}
