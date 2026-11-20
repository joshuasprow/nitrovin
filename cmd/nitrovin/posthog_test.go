package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// postHogSink is a PostHog ingest host in-process: it keeps what it is
// sent, by path.
type postHogSink struct {
	mu     sync.Mutex
	events []map[string]any // from /batch/
	logs   []map[string]any // each /i/v1/logs body
	auth   []string
}

func (s *postHogSink) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auth = append(s.auth, r.Header.Get("Authorization"))
	switch r.URL.Path {
	case "/batch/":
		for _, e := range body["batch"].([]any) {
			s.events = append(s.events, e.(map[string]any))
		}
	case "/i/v1/logs":
		s.logs = append(s.logs, body)
	default:
		http.NotFound(w, r)
	}
}

func postHogEnv(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

// TestPostHogOff: without a token there is no export, and serve's wiring
// takes a nil one as no export at all.
func TestPostHogOff(t *testing.T) {
	ph, err := newPostHog(postHogEnv(map[string]string{"NITROVIN_POSTHOG_PROPERTIES": `{"app":"x"}`}), io.Discard)
	if err != nil || ph != nil {
		t.Fatalf("newPostHog without a token = %v, %v; want nil, nil", ph, err)
	}
	var stderr strings.Builder
	if w := ph.logTo(&stderr, severityError); w != &stderr {
		t.Errorf("logTo with the export off is %T, want stderr itself", w)
	}
}

func TestPostHogRefusesConfig(t *testing.T) {
	for name, vars := range map[string]map[string]string{
		"host not a URL":        {"NITROVIN_POSTHOG_HOST": "us.i.posthog.com"},
		"properties not object": {"NITROVIN_POSTHOG_PROPERTIES": `["app"]`},
		"properties not string": {"NITROVIN_POSTHOG_PROPERTIES": `{"replicas":2}`},
	} {
		t.Run(name, func(t *testing.T) {
			vars["NITROVIN_POSTHOG_TOKEN"] = "phc_test"
			if ph, err := newPostHog(postHogEnv(vars), io.Discard); err == nil {
				ph.close()
				t.Fatal("newPostHog accepted it")
			}
		})
	}
}

// TestPostHogExport: a panic and an internal problem served on faulty
// arrive as $exceptions with their stacks and the deployer's properties,
// and the lines serve logs about them as log records carrying the same.
func TestPostHogExport(t *testing.T) {
	sink := &postHogSink{}
	host := httptest.NewServer(sink)
	defer host.Close()
	var stderr strings.Builder
	ph, err := newPostHog(postHogEnv(map[string]string{
		"NITROVIN_POSTHOG_TOKEN":       "phc_test",
		"NITROVIN_POSTHOG_HOST":        host.URL + "/",
		"NITROVIN_POSTHOG_DISTINCT_ID": "deployer:nitrovin",
		"NITROVIN_POSTHOG_PROPERTIES":  `{"app":"deployer","service":"nitrovin","environment":"test"}`,
	}), &stderr)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newAPI(faulty{}, log.New(ph.logTo(io.Discard, severityError), "nitrovin: ", 0), io.Discard, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	a.ph = ph
	for _, path := range []string{"/v1/info", "/v1/decode/1HGCM82633A004352", "/healthz"} {
		a.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}
	ph.close()
	if stderr.Len() > 0 {
		t.Errorf("stderr: %s", stderr.String())
	}

	for i, want := range []struct{ typ, value, top string }{
		{"panic", "panic: injected fault", ".faulty.Info"},
		{"internal", "injected fault", ".(*api).problem"},
	} {
		e := sink.events[i]
		props := e["properties"].(map[string]any)
		if e["event"] != "$exception" || e["distinct_id"] != "deployer:nitrovin" {
			t.Errorf("event %d is %v for %v", i, e["event"], e["distinct_id"])
		}
		for k, v := range map[string]any{
			"app": "deployer", "service": "nitrovin", "environment": "test",
			"$issue_name": "deployer: " + want.typ, "$process_person_profile": false,
		} {
			if props[k] != v {
				t.Errorf("event %d: %s = %v, want %v", i, k, props[k], v)
			}
		}
		if props["decoder_version"] == "" {
			t.Errorf("event %d has no decoder_version", i)
		}
		ex := props["$exception_list"].([]any)[0].(map[string]any)
		if ex["type"] != want.typ || ex["value"] != want.value {
			t.Errorf("event %d: exception %v: %v, want %s: %s", i, ex["type"], ex["value"], want.typ, want.value)
		}
		frames := ex["stacktrace"].(map[string]any)["frames"].([]any)
		top := frames[len(frames)-1].(map[string]any)
		if fn, _ := top["function"].(string); !strings.HasSuffix(fn, want.top) || top["in_app"] != true {
			t.Errorf("event %d: innermost frame %v (in app %v), want …%s", i, top["function"], top["in_app"], want.top)
		}
	}

	var lines []string
	for _, body := range sink.logs {
		rl := body["resourceLogs"].([]any)[0].(map[string]any)
		res := attrs(rl["resource"].(map[string]any)["attributes"])
		if res["service.name"] != "nitrovin" || res["app"] != "deployer" {
			t.Errorf("resource %v", res)
		}
		for _, r := range rl["scopeLogs"].([]any)[0].(map[string]any)["logRecords"].([]any) {
			r := r.(map[string]any)
			if a := attrs(r["attributes"]); a["app"] != "deployer" || a["environment"] != "test" || r["severityText"] != "ERROR" {
				t.Errorf("record %v: attributes %v", r["severityText"], a)
			}
			lines = append(lines, r["body"].(map[string]any)["stringValue"].(string))
		}
	}
	if len(lines) != 3 ||
		!strings.HasPrefix(lines[0], "nitrovin: panic serving GET /v1/info: injected fault\n") ||
		!strings.Contains(lines[2], "injected fault") {
		t.Errorf("log lines %q, want the panic with its stack and the two problems", lines)
	}
	for _, a := range sink.auth {
		if a != "Bearer phc_test" {
			t.Errorf("Authorization %q", a)
		}
	}
}

// attrs is OTLP attributes as a map.
func attrs(v any) map[string]string {
	m := map[string]string{}
	for _, kv := range v.([]any) {
		kv := kv.(map[string]any)
		m[kv["key"].(string)] = kv["value"].(map[string]any)["stringValue"].(string)
	}
	return m
}
