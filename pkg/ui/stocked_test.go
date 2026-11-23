package ui

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/joshuasprow/nitrovin/pkg/contract"
)

// yards is a backend whose stocked answers are scripted: answer n is
// answers[n], the last one from then on, each after wait unless its
// context ends first.
type yards struct {
	backend
	mu      sync.Mutex
	asked   int
	wait    time.Duration
	answers []contract.Stocked
}

func (y *yards) Stocked(ctx context.Context, _ contract.StockedRequest) (contract.Stocked, error) {
	y.mu.Lock()
	n, wait := y.asked, y.wait
	y.asked++
	y.mu.Unlock()
	select {
	case <-time.After(wait):
	case <-ctx.Done():
		return contract.Stocked{}, ctx.Err()
	}
	return y.answers[min(n, len(y.answers)-1)], nil
}

func (y *yards) calls() int {
	y.mu.Lock()
	defer y.mu.Unlock()
	return y.asked
}

func stockedOf(vin string, yard int, status string) contract.Stocked {
	s := contract.Stocked{Sources: []contract.StockSource{{Name: "yard", Kind: "live", Status: status}}}
	if vin != "" {
		s.Vehicles = []contract.StockedVehicle{{VIN: vin, ModelYear: 2015, Yard: yard, Source: "yard",
			Spine: []contract.SpineEntry{{SpineRef: contract.SpineRef{Make: 460}, Name: "Ford"}, {SpineRef: contract.SpineRef{Model: 1801}, Name: "F-150"}}}}
	}
	return s
}

// TestStockedKept: the VINs menu's stocked vehicles are kept, so a page
// never waits on a yard past the first. The first page waits for the
// answer; later pages show it without asking; once it is no longer fresh
// a page still shows it and has it asked again behind it. An answer a
// yard did not give leaves the last whole one showing, and is asked again
// after the retry time. A first answer slower than its bound shows
// nothing and is asked again behind the page.
func TestStockedKept(t *testing.T) {
	label := func(list []stockedVIN) string {
		if len(list) == 0 {
			return ""
		}
		return list[0].Label + " " + list[0].Counts() + " " + list[0].Href
	}
	until := func(what string, ok func() bool) {
		t.Helper()
		for start := time.Now(); !ok(); time.Sleep(time.Millisecond) {
			if time.Since(start) > 5*time.Second {
				t.Fatalf("%s: never", what)
			}
		}
	}
	y := &yards{answers: []contract.Stocked{
		stockedOf("VIN1", 6, contract.StockAvailable),
		stockedOf("VIN2", 7, contract.StockAvailable),
		stockedOf("", 0, contract.StockUnavailable),
		stockedOf("VIN3", 8, contract.StockAvailable),
	}}
	h := New(y, log.New(io.Discard, "", 0))
	h.stock.fresh, h.stock.retry = time.Hour, time.Hour
	const first = "2015 Ford F-150 6 in the yard /?from.vin=VIN1"
	if got := label(h.stocked()); got != first || y.calls() != 1 {
		t.Fatalf("the first page: %q after %d asks; want %q after 1", got, y.calls(), first)
	}
	if got := label(h.stocked()); got != first || y.calls() != 1 {
		t.Errorf("a page while it is fresh: %q after %d asks; want the same after 1", got, y.calls())
	}

	// Stale: the page shows what it has, and the next shows the new one.
	h.stock.mu.Lock()
	h.stock.fresh, h.stock.retry = time.Nanosecond, time.Nanosecond
	h.stock.mu.Unlock()
	if got := label(h.stocked()); got != first {
		t.Errorf("a page once it is stale: %q; want the last answer, %q", got, first)
	}
	until("the second answer", func() bool { return label(h.stocked()) == "2015 Ford F-150 7 in the yard /?from.vin=VIN2" })

	// A yard that did not answer: the last whole answer stays, and it is
	// asked again.
	until("the answer after a yard failed", func() bool { return label(h.stocked()) == "2015 Ford F-150 8 in the yard /?from.vin=VIN3" })
	if y.calls() < 4 {
		t.Errorf("%d asks; want the failed one asked again", y.calls())
	}

	// A first answer slower than its bound.
	slow := &yards{wait: 50 * time.Millisecond, answers: []contract.Stocked{stockedOf("VIN9", 2, contract.StockAvailable)}}
	h = New(slow, log.New(io.Discard, "", 0))
	h.stock.first, h.stock.retry = time.Millisecond, time.Nanosecond
	start := time.Now()
	if got := h.stocked(); len(got) != 0 || time.Since(start) > 40*time.Millisecond {
		t.Errorf("a slow first answer: %v after %v; want nothing, at once", got, time.Since(start))
	}
	until("the slow yard's answer", func() bool { return label(h.stocked()) == "2015 Ford F-150 2 in the yard /?from.vin=VIN9" })
}
