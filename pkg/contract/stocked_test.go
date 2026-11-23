package contract

import (
	"net/url"
	"reflect"
	"testing"
)

// TestParseStocked: a stocked request is an object of location and limit,
// each optional; anything else is refused by pointer, on a request object,
// a query and a stdio line alike.
func TestParseStocked(t *testing.T) {
	for _, c := range []struct {
		in      string
		want    StockedRequest
		pointer string
	}{
		{`{}`, StockedRequest{}, ""},
		{`{"limit":100,"location":["north","south"]}`, StockedRequest{Limit: 100, Location: []string{"north", "south"}}, ""},
		{`{"limit":0}`, StockedRequest{}, "/limit"},
		{`{"limit":101}`, StockedRequest{}, "/limit"},
		{`{"limit":"3"}`, StockedRequest{}, "/limit"},
		{`{"location":[]}`, StockedRequest{}, "/location"},
		{`{"location":"north"}`, StockedRequest{}, "/location"},
		{`{"location":["north",3]}`, StockedRequest{}, "/location/1"},
		{`{"location":[""]}`, StockedRequest{}, "/location/0"},
		{`{"stock":["yard"]}`, StockedRequest{}, "/stock"},
		{`{"after":"x"}`, StockedRequest{}, "/after"},
	} {
		got, err := ParseStockedRequest([]byte(c.in))
		if c.pointer == "" {
			if err != nil || !reflect.DeepEqual(got, c.want) {
				t.Errorf("%s: %+v, %v; want %+v", c.in, got, err, c.want)
			}
		} else if p := AsProblem(err); err == nil || p.Type != InvalidRequest || p.Pointer != c.pointer {
			t.Errorf("%s: %v; want an invalid request at %s", c.in, err, c.pointer)
		}
		_, r, err := ParseStdio([]byte(`{"op":"stocked",` + c.in[1:]))
		if c.in == `{}` {
			_, r, err = ParseStdio([]byte(`{"op":"stocked"}`))
		}
		if c.pointer == "" {
			if err != nil || r.Op != OpStocked || !reflect.DeepEqual(r.Stocked, c.want) {
				t.Errorf("stdio %s: %+v, %v; want %+v", c.in, r, err, c.want)
			}
		} else if p := AsProblem(err); err == nil || p.Pointer != c.pointer {
			t.Errorf("stdio %s: %v; want a refusal at %s", c.in, err, c.pointer)
		}
	}
	if _, err := ParseStockedRequest([]byte(`{`)); AsProblem(err).Type != MalformedJSON {
		t.Errorf("not JSON: %v", err)
	}
	for _, c := range []struct {
		query   string
		want    StockedRequest
		pointer string
	}{
		{"", StockedRequest{}, ""},
		{"limit=5&location=north&location=south", StockedRequest{Limit: 5, Location: []string{"north", "south"}}, ""},
		{"limit=5&limit=6", StockedRequest{}, "/limit"},
		{"limit=many", StockedRequest{}, "/limit"},
		{"location=", StockedRequest{}, "/location/0"},
		{"of=yard", StockedRequest{}, "/of"},
	} {
		q, _ := url.ParseQuery(c.query)
		got, err := ParseStockedQuery(q)
		if c.pointer == "" {
			if err != nil || !reflect.DeepEqual(got, c.want) {
				t.Errorf("?%s: %+v, %v; want %+v", c.query, got, err, c.want)
			}
		} else if p := AsProblem(err); err == nil || p.Type != InvalidRequest || p.Pointer != c.pointer {
			t.Errorf("?%s: %v; want an invalid request at %s", c.query, err, c.pointer)
		}
	}
}
