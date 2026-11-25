package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/nitrovin"
	"github.com/joshuasprow/nitrovin/pkg/stock"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// orderStockFixture is testdata/vocab/stock/order: a shelf and a yard
// whose headers declare their rows' field order (#390), over lines written
// out of it. The surface tests load it beside stockFixture.
var orderStockFixture = abs("../../testdata/vocab/stock/order")

// orderFit asks for the stock of the engine the order fixture's shelf and
// yard hold, and orderCut for four of its rows' fields, named out of the
// declared order; orderFitQuery and orderCutQuery are GET /v1/fit's of
// them.
const (
	orderFit      = `{"from":{"fitment":{"type":"engine","number":"EN-57"}},"stock":["shelf","yard"]}`
	orderCut      = `{"from":{"fitment":{"type":"engine","number":"EN-57"}},"stock":["shelf","yard"],"stock_fields":["tag","photos","price","stock_number"]}`
	orderFitQuery = "from.fitment.type=engine&from.fitment.number=EN-57&stock=shelf&stock=yard"
	orderCutQuery = orderFitQuery + "&stock_fields=tag&stock_fields=photos&stock_fields=price&stock_fields=stock_number"
)

// What a surface's bytes hold when the declared order reached them: the
// fields of the shelf's R-1 and the yard's V-1 in an answer to orderFit
// (orderedFields) and to orderCut (orderedCut), each value as its line
// writes it; and Info's report of the two sources (orderedInfo), the
// shelf's pin (#404) among it and the yard, which pins none, without one.
var (
	orderedFields = []string{
		`"fields":{"stock_number":"E500","condition":"A","price":1850.00,"tag":"T-9","miles":61250,"status":"on shelf"}`,
		`"fields":{"stock_number":"V900","row":"K","photos":[{"url":"v900-front.jpg","w":1600,"h":1200},{"url":"v900-rear.jpg","w":1600,"h":1200}],"color":"grey","added":"2026-10-30"}`,
	}
	orderedCut = []string{
		`"fields":{"stock_number":"E500","price":1850.00,"tag":"T-9"}`,
		`"fields":{"stock_number":"V900","photos":[{"url":"v900-front.jpg","w":1600,"h":1200},{"url":"v900-rear.jpg","w":1600,"h":1200}]}`,
	}
	orderedInfo = []string{
		`"source":"shelf","catalog":"parts-catalog","fields":["stock_number","condition","price","warranty"],"pin":["stock_number","price"]`,
		`"source":"yard","fields":["stock_number","row","photos"],"as_of"`,
	}
)

// holds fails unless answer, a surface's own bytes, holds each of want as
// written.
func holds(t *testing.T, what string, answer []byte, want []string) {
	t.Helper()
	for _, w := range want {
		if !bytes.Contains(answer, []byte(w)) {
			t.Errorf("%s holds no %s:\n%s", what, w, answer)
		}
	}
}

// TestStockFieldOrder: the CLI answers a stock row's fields in the order
// its source's header declares (#390), whole and cut by stock_fields, and
// `info` reports each source's declared order, and none for a source that
// declares none.
func TestStockFieldOrder(t *testing.T) {
	fixtureData(t)
	vocabArgs := []string{"--vocab", fixture, "--vocab", fitmentFixture, "--vocab", stockFixture, "--vocab", orderStockFixture}
	r := cli(t, nil, orderFit+"\n"+orderCut+"\n", append(vocabArgs, "fit", "-")...)
	answers := strings.Split(strings.TrimSuffix(r.stdout, "\n"), "\n")
	if r.exit != 0 || r.stderr != "" || len(answers) != 2 {
		t.Fatalf("fit -: exit %d, stderr %q, %d answers; want 2", r.exit, r.stderr, len(answers))
	}
	holds(t, "fit -", []byte(answers[0]), orderedFields)
	holds(t, "fit - with stock_fields", []byte(answers[1]), orderedCut)

	r = cli(t, nil, "", append(vocabArgs, "info")...)
	if r.exit != 0 {
		t.Fatalf("info: exit %d, %s", r.exit, r.stderr)
	}
	holds(t, "info", []byte(r.stdout), orderedInfo)
	if n := strings.Count(r.stdout, `"fields":`); n != len(orderedInfo) {
		t.Errorf("info reports fields of %d files; want of the %d sources that declare an order:\n%s", n, len(orderedInfo), r.stdout)
	}
	reportsPin(t, "info", []byte(r.stdout))
}

// reportsPin fails unless info, a surface's /v1/info in its own bytes,
// reports a pin (#404) for the one source that declares one, shelf-order.
func reportsPin(t *testing.T, what string, info []byte) {
	t.Helper()
	if n := bytes.Count(info, []byte(`"pin":`)); n != 1 {
		t.Errorf("%s reports a pin of %d files; want of shelf-order alone:\n%s", what, n, info)
	}
}

// downSource is a live stock source that is down: every lookup fails with
// err.
type downSource struct {
	name  string
	holds stock.Holds
	err   error
}

func (s downSource) Info() stock.Info {
	return stock.Info{Name: s.name, Holds: s.holds, Kind: stock.KindLive}
}
func (s downSource) ShelfByFitment(context.Context, [][]vocab.FitmentKey, stock.Page) ([]stock.Result, stock.Meta, error) {
	return nil, stock.Meta{}, s.err
}
func (s downSource) ShelfByVehicle(context.Context, []stock.ShelfVehicle, stock.Page) ([]stock.Result, stock.Meta, error) {
	return nil, stock.Meta{}, s.err
}
func (s downSource) Yard(context.Context, []stock.YardQuery, stock.Page) ([]stock.Result, stock.Meta, error) {
	return nil, stock.Meta{}, s.err
}

// stockService answers fits from f, everything else from the embedded
// service.
type stockService struct {
	decodeService
	f contract.Fitter
}

func (s stockService) Fit(ctx context.Context, r contract.FitRequest) (contract.Fit, error) {
	return s.f.Fit(ctx, r)
}
func (s stockService) FitBatch(ctx context.Context, rs []contract.FitRequest) ([]contract.Fit, error) {
	return s.f.FitBatch(ctx, rs)
}
func (s stockService) FitBatchRows(ctx context.Context, rs []contract.FitRequest) (contract.FitBatchRows, error) {
	return s.f.FitBatchRows(ctx, rs)
}

// TestServeStockUnavailable: over HTTP, a stock source that is down
// answers unavailable in its block, with its own reason or, for any other
// error, a fixed one that never shows the error's text; the file sources'
// rows and the rest of the answer stand.
func TestServeStockUnavailable(t *testing.T) {
	fixtureData(t)
	d, err := nitrovin.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	files, err := vocab.Open(fixture, fitmentFixture, stockFixture)
	if err != nil {
		t.Fatal(err)
	}
	set, err := vocab.NewSet(d.Month(), files)
	if err != nil {
		t.Fatal(err)
	}
	info := d.Info()
	f := contract.Fitter{Translator: contract.Translator{Set: set, DataMonth: info.DataMonth, DecoderVersion: info.DecoderVersion}, Catalog: set.Catalog()}
	for _, s := range set.Stock() {
		f.Stock = append(f.Stock, stock.NewFile(s, nil))
	}
	f.Stock = append(f.Stock,
		downSource{"live-shelf", stock.Shelf, &stock.Unavailable{Reason: "the database is not answering"}},
		downSource{"live-yard", stock.Yard, errors.New("dial tcp 10.0.0.5:5432: user=stock")})
	a, err := newAPI(stockService{translating(t, d, files...), f}, log.New(io.Discard, "", 0), io.Discard, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a)
	defer srv.Close()

	resp, err := client.Get(srv.URL + "/v1/fit?from.fitment.type=brake-rotor-front&from.fitment.number=BR-2001&stock=shelf&stock=yard&now=" + now)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	// The parts of the answer this test reads.
	type list struct {
		Rows    []json.RawMessage
		Total   int
		Sources []contract.StockSource
	}
	var got struct {
		Status   contract.FitStatus
		Vehicles []json.RawMessage
		Shelf    *list
		Yard     *list
	}
	if resp.StatusCode != http.StatusOK || json.Unmarshal(body, &got) != nil || got.Shelf == nil || got.Yard == nil {
		t.Fatalf("GET /v1/fit: %d %s", resp.StatusCode, body)
	}
	last := func(l *list) contract.StockSource { return l.Sources[len(l.Sources)-1] }
	if s := last(got.Shelf); s.Name != "live-shelf" || s.Kind != "live" || s.Status != "unavailable" || s.Reason != "the database is not answering" {
		t.Errorf("the live shelf's block %+v", s)
	}
	if s := last(got.Yard); s.Name != "live-yard" || s.Status != "unavailable" || s.Reason != contract.StockFailed {
		t.Errorf("the live yard's block %+v", s)
	}
	if strings.Contains(string(body), "10.0.0.5") || got.Shelf.Total == 0 || got.Status != contract.FitFound || len(got.Vehicles) == 0 {
		t.Errorf("the answer: %s", body)
	}
}

// faultySource is a live stock source whose every lookup panics.
type faultySource struct{}

func (faultySource) Info() stock.Info {
	return stock.Info{Name: "faulty", Holds: stock.Shelf, Kind: stock.KindLive}
}
func (faultySource) ShelfByFitment(context.Context, [][]vocab.FitmentKey, stock.Page) ([]stock.Result, stock.Meta, error) {
	panic("a fault in a stock source")
}
func (faultySource) ShelfByVehicle(context.Context, []stock.ShelfVehicle, stock.Page) ([]stock.Result, stock.Meta, error) {
	panic("a fault in a stock source")
}
func (faultySource) Yard(context.Context, []stock.YardQuery, stock.Page) ([]stock.Result, stock.Meta, error) {
	panic("a fault in a stock source")
}

// TestServeStockPanic: a stock source that panics, on a lookup's goroutine
// of its own, is an #internal problem naming the panic, logged with the
// stack it happened on, and serve outlives it.
func TestServeStockPanic(t *testing.T) {
	fixtureData(t)
	d, err := nitrovin.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	files, err := vocab.Open(fixture, fitmentFixture, stockFixture)
	if err != nil {
		t.Fatal(err)
	}
	set, err := vocab.NewSet(d.Month(), files)
	if err != nil {
		t.Fatal(err)
	}
	info := d.Info()
	f := contract.Fitter{Translator: contract.Translator{Set: set, DataMonth: info.DataMonth, DecoderVersion: info.DecoderVersion}, Catalog: set.Catalog(), Stock: []stock.Source{faultySource{}}}
	var logged strings.Builder
	a, err := newAPI(stockService{translating(t, d, files...), f}, log.New(&logged, "", 0), io.Discard, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	for range 2 {
		resp, err := client.Get(srv.URL + "/v1/fit?from.fitment.type=brake-rotor-front&from.fitment.number=BR-2001&stock=shelf&now=" + now)
		if err != nil {
			t.Fatal(err)
		}
		var p contract.Problem
		err = json.NewDecoder(resp.Body).Decode(&p)
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusInternalServerError || p.Type != contract.Internal || p.Detail != "panic: a fault in a stock source" {
			t.Fatalf("GET /v1/fit: %d %+v (%v)", resp.StatusCode, p, err)
		}
	}
	if !strings.Contains(logged.String(), "faultySource.ShelfByFitment") {
		t.Errorf("the log has not the stack the panic happened on:\n%s", logged.String())
	}
}
