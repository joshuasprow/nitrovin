package nitrovin

import (
	"slices"
	"strings"
	"testing"

	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// TestFitCaseExact: a catalog whose vehicle and filed-under vocabularies
// are case-exact (testdata/vocab/case, ADR 0026) answers OLD3 and old3 as
// two vehicles, and MIR and mir as two codes: by vehicle, narrowed by a
// code, looked up by a code, as related vehicles and on a shelf; in both
// forms. A spelling the vocabulary does not hold is unknown.
func TestFitCaseExact(t *testing.T) {
	files, err := vocab.Open("../../testdata/vocab/case")
	if err != nil {
		t.Fatal(err)
	}
	jsonl, err := NewTranslator(embeddedDecoder(t), files...)
	if err != nil {
		t.Fatal(err)
	}
	compiled := compiledTwin(t, jsonl)
	const (
		old3  = `{"vocabulary":"codes","entry":{"code":"old3"}}`
		upper = `{"vocabulary":"codes","entry":{"code":"OLD3"}}`
	)
	for _, c := range []struct {
		name, request string
		want          []string
	}{
		{"OLD3", `{"from":` + upper + `,"part":{"spine":"part-types","id":"mirror"}}`, []string{
			"status found",
			"model OLD3",
			"candidate mirror M-1 left direct under [MIR]",
			`  OLD3 2009-2012 "" [model_year]`,
			"candidate mirror M-3 right direct under [MIR,mir]",
			`  OLD3 2009-2012 "" [model_year]`,
			"total 2",
		}},
		{"old3", `{"from":` + old3 + `,"part":{"spine":"part-types","id":"mirror"}}`, []string{
			"status found",
			"model old3",
			"candidate mirror M-2 left direct under [mir]",
			`  old3 2013-2018 "" [model_year]`,
			"candidate mirror M-3 right direct under [MIR,mir]",
			`  old3 2013-2018 "" [model_year]`,
			"total 2",
		}},
		{"OLD3 narrowed by mir", `{"from":` + upper + `,"part":{"vocabulary":"system-codes","entry":{"code":"mir"}}}`, []string{
			"status found",
			"model OLD3",
			"candidate mirror M-3 right direct under [MIR,mir]",
			`  OLD3 2009-2012 "" [model_year]`,
			"total 1",
		}},
		{"by mir", `{"from":{"fitment":{"filed_under":{"code":"mir"},"number":"M-2"}}}`, []string{
			"status found",
			"fitment mirror M-2 left direct under [mir]",
			`vehicle old3 2013-2018 "" → model:13620 make:496`,
			"total 1",
		}},
		{"related to OLD3", `{"from":` + upper + `,"related":true}`, []string{
			"status found",
			"model OLD3",
			"related old3 2013-2018 1 mirror:1 → model:13620 make:496",
			"total 1",
		}},
	} {
		for form, tr := range map[string]*Translator{"JSON Lines": jsonl, "compiled": compiled} {
			f, err := fit(t, tr, c.request)
			if err != nil {
				t.Fatalf("%s, %s: %v", c.name, form, err)
			}
			if got := fitSummary(f); !slices.Equal(got, c.want) {
				t.Errorf("%s, %s:\n%s\nwant:\n%s", c.name, form, strings.Join(got, "\n"), strings.Join(c.want, "\n"))
			}
		}
	}
	for _, c := range []struct {
		request string
		want    contract.FitStatus
	}{
		{`{"from":{"vocabulary":"codes","entry":{"code":"Old3"}}}`, contract.FitUnknown},
		{`{"from":{"fitment":{"filed_under":{"code":"MIR"},"number":"M-2"}}}`, contract.FitUnknown},
		{`{"from":{"fitment":{"filed_under":{"code":"Mir"},"number":"M-3"}}}`, contract.FitUnknown},
	} {
		for form, tr := range map[string]*Translator{"JSON Lines": jsonl, "compiled": compiled} {
			if f, err := fit(t, tr, c.request); err != nil || f.Status != c.want {
				t.Errorf("%s, %s: status %q, %v; want %q", c.request, form, f.Status, err, c.want)
			}
		}
	}

	// A shelf row names its vehicle as the vocabulary spells it.
	for vehicle, want := range map[string]string{upper: "1: S-2/vehicle", old3: "1: S-1/vehicle"} {
		for form, tr := range map[string]*Translator{"JSON Lines": jsonl, "compiled": compiled} {
			f, err := fit(t, tr, `{"from":`+vehicle+`,"part":{"spine":"part-types","id":"mirror"},"stock":["shelf"]}`)
			if err != nil {
				t.Fatal(err)
			}
			if got := stockText(f.ShelfByVehicle); got != want {
				t.Errorf("%s, %s: shelf_by_vehicle %s; want %s", vehicle, form, got, want)
			}
		}
	}
}
