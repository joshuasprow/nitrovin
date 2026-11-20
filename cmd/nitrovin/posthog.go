package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joshuasprow/nitrovin/pkg/nitrovin"
)

// postHog is serve's optional export to PostHog (#356): every fault serve
// answers or stops with as a $exception event, and every line it logs as a
// PostHog Logs record. NITROVIN_POSTHOG_TOKEN turns it on; a nil *postHog
// is the export turned off, and each of its methods does nothing.
//
// Sending never blocks a request and never fails one. Events and records
// wait in a queue that one goroutine sends from, in batches, every
// postHogInterval; when the queue is full they are dropped and counted, and
// a failed send is written to stderr.
type postHog struct {
	host       string
	token      string
	distinctID string
	// issuePrefix names an exception's issue, "<issuePrefix>: <type>":
	// PostHog keeps the first exception's name, so it is on every event.
	issuePrefix string
	// props are stamped on every event and on every log line;
	// decoder_version is always among them.
	props map[string]string
	// attrs are props as OTLP attributes, sorted by key; resource adds
	// service.name. Both are read-only once built.
	attrs    []otlpKV
	resource []otlpKV
	client   *http.Client
	// stderr is where the export's own failures go, never itself.
	stderr  io.Writer
	queue   chan postHogItem
	dropped atomic.Int64
	stop    chan struct{}
	done    chan struct{}
	closing sync.Once
}

// postHogItem is one queued $exception event or one log record.
type postHogItem struct {
	event  map[string]any
	record *otlpRecord
}

const (
	postHogDefaultHost = "https://us.i.posthog.com"
	postHogQueue       = 1024
	postHogBatch       = 100
	postHogInterval    = time.Second
	// postHogSendTimeout bounds one POST; postHogCloseTimeout bounds the
	// flush at shutdown.
	postHogSendTimeout  = 10 * time.Second
	postHogCloseTimeout = 5 * time.Second
)

// newPostHog is the export getenv configures, or nil when
// NITROVIN_POSTHOG_TOKEN is unset:
//   - NITROVIN_POSTHOG_TOKEN is a project's public ingest token;
//   - NITROVIN_POSTHOG_HOST is the ingest host, default postHogDefaultHost;
//   - NITROVIN_POSTHOG_DISTINCT_ID is the events' distinct id, default
//     "nitrovin";
//   - NITROVIN_POSTHOG_PROPERTIES is a JSON object of strings stamped on
//     every event and every log line. Its "app" names the issues ("nitrovin"
//     without one), and its "service" is the logs' service.name
//     ("nitrovin" without one).
//
// A host that is not an http(s) URL, or properties that are not an object
// of strings, are refused.
func newPostHog(getenv func(string) string, stderr io.Writer) (*postHog, error) {
	token := getenv("NITROVIN_POSTHOG_TOKEN")
	if token == "" {
		return nil, nil
	}
	host := strings.TrimRight(cmp.Or(getenv("NITROVIN_POSTHOG_HOST"), postHogDefaultHost), "/")
	if u, err := url.Parse(host); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return nil, fmt.Errorf("NITROVIN_POSTHOG_HOST: want an http(s) URL, not %q", host)
	}
	props := map[string]string{}
	if s := getenv("NITROVIN_POSTHOG_PROPERTIES"); s != "" {
		if err := json.Unmarshal([]byte(s), &props); err != nil {
			return nil, fmt.Errorf("NITROVIN_POSTHOG_PROPERTIES: want a JSON object of strings: %w", err)
		}
	}
	if _, ok := props["decoder_version"]; !ok {
		props["decoder_version"] = nitrovin.Version()
	}
	attrs := make([]otlpKV, 0, len(props))
	for _, k := range slices.Sorted(maps.Keys(props)) {
		attrs = append(attrs, otlpKV{k, otlpValue{props[k]}})
	}
	p := &postHog{
		host:        host,
		token:       token,
		distinctID:  cmp.Or(getenv("NITROVIN_POSTHOG_DISTINCT_ID"), "nitrovin"),
		issuePrefix: cmp.Or(props["app"], "nitrovin"),
		props:       props,
		attrs:       attrs,
		resource:    append([]otlpKV{{"service.name", otlpValue{cmp.Or(props["service"], "nitrovin")}}}, attrs...),
		client:      &http.Client{Timeout: postHogSendTimeout},
		stderr:      stderr,
		queue:       make(chan postHogItem, postHogQueue),
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
	}
	go p.run()
	return p, nil
}

// close sends what is queued, waiting up to postHogCloseTimeout, and stops
// the sender. Anything queued after it is dropped.
func (p *postHog) close() {
	if p == nil {
		return
	}
	p.closing.Do(func() {
		close(p.stop)
		select {
		case <-p.done:
		case <-time.After(postHogCloseTimeout):
			fmt.Fprintf(p.stderr, "nitrovin: posthog: gave up flushing after %s\n", postHogCloseTimeout)
		}
	})
}

// enqueue queues it, or counts it dropped when the queue is full.
func (p *postHog) enqueue(it postHogItem) {
	select {
	case p.queue <- it:
	default:
		p.dropped.Add(1)
	}
}

// exception queues a $exception: typ is its type, which names its issue,
// value its text, handled whether serve went on after it, and frames its
// stack (callerStack's), nil for none.
func (p *postHog) exception(typ, value string, handled bool, frames []stackFrame) {
	if p == nil {
		return
	}
	item := map[string]any{
		"type":      typ,
		"value":     value,
		"mechanism": map[string]bool{"handled": handled, "synthetic": false},
	}
	if len(frames) > 0 {
		item["stacktrace"] = map[string]any{"type": "raw", "frames": frames}
	}
	props := make(map[string]any, len(p.props)+5)
	for k, v := range p.props {
		props[k] = v
	}
	props["$exception_list"] = []any{item}
	props["$issue_name"] = p.issuePrefix + ": " + typ
	props["$process_person_profile"] = false
	props["$lib"] = "nitrovin"
	props["$lib_version"] = p.props["decoder_version"]
	p.enqueue(postHogItem{event: map[string]any{
		"event":       "$exception",
		"distinct_id": p.distinctID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
		"properties":  props,
	}})
}

// Severities of the log lines, as OTLP numbers them.
const (
	severityInfo  = 9
	severityWarn  = 13
	severityError = 17
)

var severityText = map[int]string{severityInfo: "INFO", severityWarn: "WARN", severityError: "ERROR"}

// logTo is w, and with the export on, PostHog Logs too: each Write, one
// line as a log.Logger writes it, is a record of severity.
func (p *postHog) logTo(w io.Writer, severity int) io.Writer {
	if p == nil {
		return w
	}
	return io.MultiWriter(w, postHogLog{p, severity})
}

// postHogLog is logTo's PostHog half.
type postHogLog struct {
	p        *postHog
	severity int
}

func (l postHogLog) Write(b []byte) (int, error) {
	now := strconv.FormatInt(time.Now().UnixNano(), 10)
	l.p.enqueue(postHogItem{record: &otlpRecord{
		TimeUnixNano:         now,
		ObservedTimeUnixNano: now,
		SeverityNumber:       l.severity,
		SeverityText:         severityText[l.severity],
		Body:                 otlpValue{string(bytes.TrimSuffix(b, []byte("\n")))},
		// per line too: a PostHog log alert reads no resource attribute
		// but service.name
		Attributes: l.p.attrs,
	}})
	return len(b), nil
}

// run sends the queue in batches until close, then sends what is left.
func (p *postHog) run() {
	defer close(p.done)
	tick := time.NewTicker(postHogInterval)
	defer tick.Stop()
	var events []map[string]any
	var records []*otlpRecord
	add := func(it postHogItem) {
		if it.event != nil {
			events = append(events, it.event)
		} else {
			records = append(records, it.record)
		}
	}
	flush := func() {
		if n := p.dropped.Swap(0); n > 0 {
			fmt.Fprintf(p.stderr, "nitrovin: posthog: dropped %d events and log lines: the queue was full\n", n)
		}
		if len(events) > 0 {
			p.post("/batch/", map[string]any{"api_key": p.token, "batch": events})
			events = events[:0]
		}
		if len(records) > 0 {
			p.post("/i/v1/logs", otlpLogs{[]otlpResourceLogs{{
				Resource:  otlpResource{p.resource},
				ScopeLogs: []otlpScopeLogs{{Scope: otlpScope{"nitrovin", p.props["decoder_version"]}, LogRecords: records}},
			}}})
			records = records[:0]
		}
	}
	for {
		select {
		case it := <-p.queue:
			add(it)
			if len(events)+len(records) >= postHogBatch {
				flush()
			}
		case <-tick.C:
			flush()
		case <-p.stop:
			for {
				select {
				case it := <-p.queue:
					add(it)
				default:
					flush()
					return
				}
			}
		}
	}
}

// post sends v as JSON to path on the host; a failure goes to stderr.
func (p *postHog) post(path string, v any) {
	body, err := json.Marshal(v)
	if err != nil { // strings, numbers and bools: unreachable
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), postHogSendTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.host+path, bytes.NewReader(body))
	if err != nil { // newPostHog checked the host: unreachable
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.token)
	res, err := p.client.Do(req)
	if err != nil {
		fmt.Fprintf(p.stderr, "nitrovin: posthog: POST %s: %v\n", path, err)
		return
	}
	io.Copy(io.Discard, io.LimitReader(res.Body, 1<<16)) // drained so the connection is reused
	res.Body.Close()
	if res.StatusCode/100 != 2 {
		fmt.Fprintf(p.stderr, "nitrovin: posthog: POST %s: %s\n", path, res.Status)
	}
}

// stackFrame is one frame of a $exception's stack, as PostHog reads a Go
// one.
type stackFrame struct {
	Filename string `json:"filename"`
	LineNo   int    `json:"lineno"`
	Function string `json:"function"`
	InApp    bool   `json:"in_app"`
	Platform string `json:"platform"`
}

// callerStack is the stack of callerStack's caller, skip frames above it,
// outermost first as PostHog orders frames. During a panic (a recover's
// deferred call) it starts at the panic instead, and panicking is true.
func callerStack(skip int) (frames []stackFrame, panicking bool) {
	pcs := make([]uintptr, 64)
	pcs = pcs[:runtime.Callers(skip+2, pcs)]
	it := runtime.CallersFrames(pcs)
	for {
		f, more := it.Next()
		if f.Function == "runtime.gopanic" {
			frames, panicking = frames[:0], true
		} else {
			frames = append(frames, stackFrame{
				Filename: f.File,
				LineNo:   f.Line,
				Function: f.Function,
				InApp:    strings.HasPrefix(f.Function, "main.") || strings.HasPrefix(f.Function, "github.com/joshuasprow/nitrovin/"),
				Platform: "go",
			})
		}
		if !more {
			break
		}
	}
	slices.Reverse(frames)
	return frames, panicking
}

// The OTLP/HTTP JSON logs request, as much of it as nitrovin sends.
type (
	otlpLogs struct {
		ResourceLogs []otlpResourceLogs `json:"resourceLogs"`
	}
	otlpResourceLogs struct {
		Resource  otlpResource    `json:"resource"`
		ScopeLogs []otlpScopeLogs `json:"scopeLogs"`
	}
	otlpResource struct {
		Attributes []otlpKV `json:"attributes"`
	}
	otlpScopeLogs struct {
		Scope      otlpScope     `json:"scope"`
		LogRecords []*otlpRecord `json:"logRecords"`
	}
	otlpScope struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	otlpRecord struct {
		TimeUnixNano         string    `json:"timeUnixNano"`
		ObservedTimeUnixNano string    `json:"observedTimeUnixNano"`
		SeverityNumber       int       `json:"severityNumber"`
		SeverityText         string    `json:"severityText"`
		Body                 otlpValue `json:"body"`
		Attributes           []otlpKV  `json:"attributes"`
	}
	otlpKV struct {
		Key   string    `json:"key"`
		Value otlpValue `json:"value"`
	}
	otlpValue struct {
		StringValue string `json:"stringValue"`
	}
)
