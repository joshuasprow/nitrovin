package nitrovin

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"testing"

	"github.com/joshuasprow/nitrovin/pkg/contract"
)

var updateStockGolden = flag.Bool("update-stock-golden", false, "rewrite testdata/fit-batch-stock.json from FitBatch's answer")

// stockGolden is the batch TestFitBatchStockGolden pins: every stock list
// of a fit (candidates' shelf and yard, shelf_by_vehicle, a fitment key's
// lists over two pages, a related vehicle's yard count), its vehicles
// named in fleet rather than by the data month's vPIC names, and no VIN to
// decode, so the answer is the synthetic files' alone.
var stockGolden = []string{
	`{"from":{"vocabulary":"catalog-models","entry":{"make":"Dodge","model":"Ram 1500"}},"model_year":2010,"part":{"vocabulary":"part-codes","entry":{"code":"ROT"}},"stock":["shelf","yard"]}`,
	`{"from":{"vocabulary":"catalog-models","entry":{"make":"Dodge","model":"Ram 1500"}},"model_year":2010,"part":{"spine":"part-types","id":"mirror"},"stock":["shelf"],"location":["north"]}`,
	`{"from":{"fitment":{"type":"brake-rotor-front","number":"BR-1999"}},"to":["fleet"],"stock":["shelf","yard"]}`,
	`{"from":{"fitment":{"type":"brake-rotor-front","number":"BR-2001"}},"to":["fleet"],"stock":["shelf","yard"],"limit":1}`,
	`{"from":{"fitment":{"type":"brake-rotor-front","number":"BR-2001"}},"to":["fleet"],"stock":["shelf","yard"],"limit":1,"after":"WyJ2ZWhpY2xlcyIsMV0"}`,
	`{"from":{"fitment":{"type":"mirror","number":"MR-1001","side":"left"}},"to":["fleet"],"stock":["shelf","yard"],"location":["north","south"]}`,
	`{"from":{"model":1850},"model_year":2010,"related":true,"to":["fleet"],"stock":["yard"]}`,
}

// TestFitBatchStockGolden: a batch that asks for neither stock_fields nor
// its stock rows once answers byte for byte as it did before either
// existed (#371). testdata/fit-batch-stock.json is that answer, written by
// the code before them, its data month left out; -update-stock-golden
// rewrites it, which only a change meant to change the answer may do.
func TestFitBatchStockGolden(t *testing.T) {
	tr := stockTranslator(t)
	reqs := make([]contract.FitRequest, len(stockGolden))
	for i, item := range stockGolden {
		var err error
		if reqs[i], err = contract.ParseFitRequest([]byte(item)); err != nil {
			t.Fatalf("%s: %v", item, err)
		}
		reqs[i].Now = fitNow
	}
	fits, err := tr.FitBatch(context.Background(), reqs)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(fits)
	if err != nil {
		t.Fatal(err)
	}
	got = append(bytes.ReplaceAll(got, []byte(`"data_month":"`+fits[0].DataMonth+`"`), []byte(`"data_month":"<month>"`)), '\n')
	const golden = "testdata/fit-batch-stock.json"
	if *updateStockGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("the batch's answer\n%s\nwant %s's\n%s", got, golden, want)
	}
	if !bytes.Contains(want, []byte(`"fields":{"stock_number"`)) {
		t.Errorf("%s holds no stock row's fields", golden)
	}
}
