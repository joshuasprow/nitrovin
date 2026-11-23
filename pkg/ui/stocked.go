package ui

import (
	"cmp"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joshuasprow/nitrovin/pkg/contract"
)

// The VINs menu's "In stock": the vehicles the yards hold most of (ADR
// 0028), so a page with stock on it is always two clicks away. A yard may
// be a database, so the page keeps the answer and never waits on one.

// stockedVIN is one row of "In stock": a vehicle by its year, make and
// model, how many the yard holds, and the page with its VIN on the plate.
type stockedVIN struct {
	Label, Href string
	Yard        int
}

// Counts is the stock under the vehicle's name.
func (s stockedVIN) Counts() string { return strconv.Itoa(s.Yard) + " in the yard" }

const (
	// stockedKept is how many vehicles the menu offers.
	stockedKept = 5
	// stockedFresh is how long an answer is shown before it is asked
	// again, and stockedRetry how long after an answer that was not whole.
	stockedFresh = 5 * time.Minute
	stockedRetry = 15 * time.Second
	// stockedFirst bounds the first answer, the one a page waits for: a
	// yard file answers from memory well inside it. stockedBound bounds
	// every later one, which no page waits for.
	stockedFirst = 250 * time.Millisecond
	stockedBound = 10 * time.Second
)

// stockedCache is the last answer shown and when it was asked for.
type stockedCache struct {
	mu         sync.Mutex
	list       []stockedVIN
	asked      bool      // the first answer was waited for
	whole      time.Time // the last answer every yard gave
	tried      time.Time // the last answer asked for
	refreshing bool
	// first, bound, fresh and retry are stockedFirst, stockedBound,
	// stockedFresh and stockedRetry unless a test sets them.
	first, bound, fresh, retry time.Duration
	// files is what the answer was of: the browser's files as they were,
	// nil on the server, whose files never change.
	files *shelf
}

// stocked are the vehicles "In stock" shows now. The first page waits
// stockedFirst for them; every later page gets the last answer and, once
// it is stockedFresh old, has it asked again behind it. An answer some
// yard did not give is asked again after stockedRetry, and the last whole
// one stays until then. In the browser, loading or removing a file starts
// over.
func (h *Handler) stocked() []stockedVIN {
	c := &h.stock
	var files *shelf
	if h.lib != nil {
		files = h.lib.state.Load()
	}
	c.mu.Lock()
	if c.files != files {
		// The browser's files changed: the answer kept is of others.
		c.files, c.list, c.asked, c.refreshing, c.whole, c.tried = files, nil, false, false, time.Time{}, time.Time{}
	}
	if !c.asked {
		c.asked, c.refreshing = true, true
		c.mu.Unlock()
		h.refreshStocked(cmp.Or(c.first, stockedFirst), files)
		c.mu.Lock()
	}
	now := time.Now()
	if !c.refreshing && now.Sub(c.whole) > cmp.Or(c.fresh, stockedFresh) && now.Sub(c.tried) > cmp.Or(c.retry, stockedRetry) {
		c.refreshing = true
		go h.refreshStocked(cmp.Or(c.bound, stockedBound), files)
	}
	defer c.mu.Unlock()
	return c.list
}

// refreshStocked asks for the stocked vehicles, waiting at most bound, and
// keeps the answer: always when every yard gave it, and otherwise only
// when it names a vehicle and nothing is kept yet. files is what it is
// asked of: an answer of files that have changed since is dropped.
func (h *Handler) refreshStocked(bound time.Duration, files *shelf) {
	ctx, cancel := context.WithTimeout(context.Background(), bound)
	defer cancel()
	s, err := h.p.Stocked(ctx, contract.StockedRequest{Limit: stockedKept})
	whole := err == nil
	var list []stockedVIN
	for _, src := range s.Sources {
		whole = whole && src.Status == contract.StockAvailable
	}
	for _, v := range s.Vehicles {
		label := []string{strconv.Itoa(int(v.ModelYear))}
		for _, e := range v.Spine {
			if e.Name != "" {
				label = append(label, e.Name)
			}
		}
		list = append(list, stockedVIN{Label: strings.Join(label, " "), Yard: v.Yard, Href: vinHref(v.VIN)})
	}
	c := &h.stock
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.files != files {
		return
	}
	c.refreshing, c.tried = false, time.Now()
	if whole {
		c.whole = c.tried
	}
	if whole || len(c.list) == 0 {
		c.list = list
	}
}

// Stocked answers as /v1/stocked does: from the yards the browser holds,
// or, while the server holds the catalog and its stock (ADR 0023), the
// server's own answer.
func (l *Library) Stocked(ctx context.Context, req contract.StockedRequest) (contract.Stocked, error) {
	if l.serverCatalog.Name == "" {
		return l.fitter(l.state.Load()).Stocked(ctx, req)
	}
	q := url.Values{}
	if req.Limit != 0 {
		q.Set("limit", strconv.Itoa(req.Limit))
	}
	if len(req.Location) > 0 {
		q["location"] = req.Location
	}
	status, body, err := l.remote.Get("/v1/stocked?" + q.Encode())
	if err != nil {
		return contract.Stocked{}, err
	}
	var s contract.Stocked
	if status != http.StatusOK || json.Unmarshal(body, &s) != nil {
		return contract.Stocked{}, &pageError{Status: http.StatusBadGateway, Message: "The server gave no stocked vehicles: HTTP " + strconv.Itoa(status)}
	}
	return s, nil
}
