package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/joshuasprow/nitrovin-data/pkg/embedded"
	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/nitrovin"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// The synthetic fixture: testdata/vocab loads clean against the embedded
// month (its headers say 2026_10; its ids and values are long-lived), its
// drift/ variant with one drift of each kind, and fitment/ a catalog with
// its spine and vehicle vocabulary, and stock/ a shelf and a yard on it;
// unchecked/ has a faulty file, files linked to it and through it, and a
// clean file with drift (ADR 0025).
var (
	fixture          = abs("../../testdata/vocab")
	driftFixture     = abs("../../testdata/vocab/drift")
	fitmentFixture   = abs("../../testdata/vocab/fitment")
	stockFixture     = abs("../../testdata/vocab/stock")
	uncheckedFixture = abs("../../testdata/vocab/unchecked")
)

func abs(path string) string {
	p, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return p
}

// fixtureMonth is the data month testdata/vocab's headers say it was built
// against.
const fixtureMonth = "2026_10"

// fixtureData marks a test that runs the fixture against the embedded data:
// when it fails, it names the month, so a refresh that breaks the fixture
// says which month did.
func fixtureData(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("on the embedded data month %s; testdata/vocab was built against %s", embedded.Month, fixtureMonth)
		}
	})
}

// driftDetail is a drift finding's detail on testdata/vocab/drift as the
// embedded month words it: format's %[1]s is the month, and a month other
// than the files' own adds what they were built against.
func driftDetail(format string) string {
	s := fmt.Sprintf(format, embedded.Month)
	if embedded.Month != fixtureMonth {
		s += " (the file was built against " + fixtureMonth + ")"
	}
	return s
}

// epaSample is the EPA sample, cmd/nitrovin-epa/testdata/vehicles.jsonl,
// as built against the embedded month (that package's TestBuild holds a
// build to differ from it in the month alone, which its header names): a
// copy's path and its bytes. The server offers EPA's vocabulary only for
// its own month.
func epaSample(t *testing.T) (string, []byte) {
	t.Helper()
	b, err := os.ReadFile("../nitrovin-epa/testdata/vehicles.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	f, err := vocab.Read("vehicles.jsonl", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	b = bytes.ReplaceAll(b, []byte(f.DataMonth), []byte(embedded.Month))
	p := filepath.Join(t.TempDir(), "vehicles.jsonl")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p, b
}

// faultyFile writes a vocabulary whose line 2 is a fault.
func faultyFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "faulty.jsonl")
	data := `{"format":1,"kind":"vocabulary","name":"yard","version":"1","subject":"vehicle","fields":["make","model"]}` + "\n" +
		`{"entry":{"make":"Dodge","model":"Ram"},"mappings":[{"model":1938,"first":2014,"last":2012}]}` + "\n"
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// findings reads JSON lines of vocab.Finding.
func findings(t *testing.T, s string) []vocab.Finding {
	t.Helper()
	var out []vocab.Finding
	for line := range strings.Lines(s) {
		var f vocab.Finding
		dec := json.NewDecoder(strings.NewReader(line))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&f); err != nil {
			t.Fatalf("not a finding line: %q", line)
		}
		out = append(out, f)
	}
	return out
}

// TestVocabCheck: `vocab check` exits 0 on clean files and 1, with a line
// per finding, on faulty or drifting ones; it checks its arguments alone.
func TestVocabCheck(t *testing.T) {
	fixtureData(t)
	if r := cli(t, []string{"NITROVIN_VOCAB=" + faultyFile(t)}, "", "vocab", "check", fixture); r.exit != 0 || r.stdout != "" || r.stderr != "" {
		t.Errorf("vocab check on the fixture: exit %d, stdout %q, stderr %q; want exit 0 and nothing", r.exit, r.stdout, r.stderr)
	}

	faulty := faultyFile(t)
	r := cli(t, nil, "", "vocab", "check", faulty)
	if fs := findings(t, r.stdout); r.exit != 1 || r.stderr != "" || len(fs) != 2 ||
		fs[0].File != faulty || fs[0].Line != 2 || fs[0].Severity != vocab.SeverityFault || !strings.Contains(fs[0].Detail, "first year 2014 is after last year 2012") ||
		fs[1].File != faulty || fs[1].Line != 0 || fs[1].Severity != vocab.SeverityUnchecked {
		t.Errorf("vocab check on a faulty file: exit %d, stdout %q, stderr %q; want exit 1, one fault at line 2 and the file unchecked", r.exit, r.stdout, r.stderr)
	}

	r = cli(t, nil, "", "vocab", "check", driftFixture)
	var got []string
	for _, f := range findings(t, r.stdout) {
		got = append(got, string(f.Severity)+" "+filepath.Base(f.File)+":"+strconv.Itoa(f.Line))
	}
	if want := []string{"drift classics.jsonl:2", "drift yard.jsonl:2", "drift yard.jsonl:3", "drift yard.jsonl:4", "drift yard.jsonl:5", "drift yard.jsonl:7", "drift yard.jsonl:8"}; r.exit != 1 || r.stderr != "" || !slices.Equal(got, want) {
		t.Errorf("vocab check on the drift fixture: exit %d, findings %v, stderr %q; want exit 1 and %v", r.exit, got, r.stderr, want)
	}

	for _, args := range [][]string{{"vocab"}, {"vocab", "check"}, {"vocab", "lint", fixture}} {
		if r := cli(t, nil, "", args...); r.exit != 2 || r.stdout != "" || !strings.Contains(r.stderr, "usage: nitrovin vocab check") {
			t.Errorf("nitrovin %q: exit %d, stdout %q, stderr %q; want a usage error", args, r.exit, r.stdout, r.stderr)
		}
	}
}

// TestVocabFindingsNameTheirEntry: a finding about a vocabulary's line
// carries "entry", its fields as written, and one about an extension
// set's line "id" (ADR 0025): drift and faults alike, from `vocab check`,
// `vocab compile` and load's stderr. A catalog's finding carries neither.
func TestVocabFindingsNameTheirEntry(t *testing.T) {
	fixtureData(t)
	about := func(fs []vocab.Finding) []string {
		var out []string
		for _, f := range fs {
			b, err := json.Marshal(map[string]any{"entry": f.Entry, "id": f.ID})
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, filepath.Base(f.File)+":"+strconv.Itoa(f.Line)+" "+string(b))
		}
		return out
	}
	ext, ram := `classics.jsonl:2 {"entry":null,"id":"silverado"}`, `yard.jsonl:2 {"entry":{"make":"Dodge","model":"Ram 1500"},"id":""}`

	r := cli(t, nil, "", "vocab", "check", driftFixture)
	if got := about(findings(t, r.stdout)); r.exit != 1 || len(got) < 2 || got[0] != ext || got[1] != ram {
		t.Errorf("vocab check on the drift fixture: exit %d, %q; want %q, %q first", r.exit, got, ext, ram)
	}
	r = cli(t, []string{"NITROVIN_VOCAB=" + driftFixture}, "", "info")
	if got := about(findings(t, r.stderr)); r.exit != 0 || len(got) < 2 || got[0] != ext || got[1] != ram {
		t.Errorf("load's stderr with the drift fixture: exit %d, %q; want %q, %q first", r.exit, got, ext, ram)
	}

	dup := filepath.Join(t.TempDir(), "dup.jsonl")
	data := `{"format":1,"kind":"vocabulary","name":"dup","version":"1","subject":"vehicle","fields":["make","model"]}` + "\n" +
		`{"entry":{"make":"Dodge","model":"Ram"},"mappings":[]}` + "\n" +
		`{"entry":{"make":"DODGE","model":" ram"},"mappings":[]}` + "\n"
	if err := os.WriteFile(dup, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	fault := `dup.jsonl:3 {"entry":{"make":"DODGE","model":" ram"},"id":""}`
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"vocab", "check", dup}, []string{fault, `dup.jsonl:0 {"entry":null,"id":""}`}},
		{[]string{"vocab", "compile", "-o", filepath.Join(t.TempDir(), "c"), fitmentFixture, dup}, []string{fault}},
	} {
		r := cli(t, nil, "", tc.args...)
		if got := about(findings(t, r.stdout)); r.exit != 1 || !slices.Equal(got, tc.want) {
			t.Errorf("nitrovin %q: exit %d, %q; want %q", tc.args, r.exit, got, tc.want)
		}
	}

	dir := t.TempDir()
	for _, name := range []string{"part-types.jsonl", "catalog-models.jsonl", "part-codes.jsonl", "catalog.jsonl"} {
		b, err := os.ReadFile(filepath.Join(fitmentFixture, name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "catalog.jsonl" {
			b = append(b, `{"redirect":{"vehicle":{"make":"Dodge","model":"Ram 1500"},"from":"engine","to":"mirror","first":2015}}`+"\n"...)
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r = cli(t, nil, "", "vocab", "check", dir)
	if got, want := about(findings(t, r.stdout)), []string{`catalog.jsonl:9 {"entry":null,"id":""}`}; r.exit != 1 || !slices.Equal(got, want) {
		t.Errorf("vocab check on a drifting catalog: exit %d, %q; want %q", r.exit, got, want)
	}
}

// TestVocabCheckBesideAFault: `vocab check` measures the drift of every
// file that reads and links cleanly though another has faults, and gives
// each file it did not measure one "unchecked" line at line 0: the faulty
// one, one linked to it, and those linked through it (ADR 0025). Exit 1,
// as on any finding. Loading the same files still refuses them all.
func TestVocabCheckBesideAFault(t *testing.T) {
	fixtureData(t)
	at := func(name string) string { return filepath.Join(uncheckedFixture, name) }
	r := cli(t, nil, "", "vocab", "check", uncheckedFixture)
	var got []string
	for _, f := range findings(t, r.stdout) {
		got = append(got, string(f.Severity)+" "+filepath.Base(f.File)+":"+strconv.Itoa(f.Line)+": "+f.Detail)
	}
	want := []string{
		"unchecked catalog.jsonl:0: its drift was not measured: it links through " + at("yard.jsonl") + " to " + at("classics.jsonl") + ", which has faults",
		"fault classics.jsonl:3: malformed JSON: unexpected end of JSON input",
		"unchecked classics.jsonl:0: its drift was not measured: it has faults",
		"unchecked shelf.jsonl:0: its drift was not measured: it links through " + at("catalog.jsonl") + " to " + at("classics.jsonl") + ", which has faults",
		"drift trucks.jsonl:3: " + fmt.Sprintf("/mappings/0/make: vPIC make 99999 is not in data month %s", embedded.Month),
		"unchecked yard.jsonl:0: its drift was not measured: it links to " + at("classics.jsonl") + ", which has faults",
	}
	if r.exit != 1 || r.stderr != "" || !slices.Equal(got, want) {
		t.Errorf("vocab check: exit %d, stderr %q, findings\n%s\nwant exit 1 and\n%s", r.exit, r.stderr, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	r = cli(t, nil, "", "--vocab", uncheckedFixture, "info")
	if p := problem(t, r); r.exit != 1 || r.stdout != "" || !strings.HasPrefix(p.Detail, "--vocab: "+at("classics.jsonl")+":3: ") {
		t.Errorf("info with the files: exit %d, stdout %q, problem %+v; want every file refused", r.exit, r.stdout, p)
	}
}

// TestVocabCheckStreamsCatalog: `vocab check` streams a JSON Lines catalog
// as compile does (ADR 0025), and finds what loading it whole finds:
// on the fixture's catalog clean, drifting, faulty, beside stock and
// alone, its findings, but for its unchecked lines, are those of the
// files opened, expanded and loaded beside the data. Its scratch, under
// TMPDIR, is gone when it ends, and it writes no other file.
func TestVocabCheckStreamsCatalog(t *testing.T) {
	fixtureData(t)
	d, err := nitrovin.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	// whole is what checking the files loaded whole finds.
	whole := func(paths []string) []vocab.Finding {
		files, err := vocab.Open(paths...)
		for i := 0; err == nil && i < len(files); i++ {
			files[i], err = files[i].Expand()
		}
		var tr *nitrovin.Translator
		if err == nil {
			tr, err = nitrovin.NewTranslator(d, files...)
		}
		if faults, ok := errors.AsType[vocab.Faults](err); ok {
			return faults
		}
		if err != nil {
			t.Fatal(err)
		}
		return tr.AllDrift()
	}
	catalogWith := func(lines ...string) string {
		dir := t.TempDir()
		data, err := os.ReadFile(filepath.Join(fitmentFixture, "catalog.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range lines {
			data = append(data, l+"\n"...)
		}
		if err := os.WriteFile(filepath.Join(dir, "catalog.jsonl"), data, 0o644); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	beside := []string{filepath.Join(fitmentFixture, "part-types.jsonl"), filepath.Join(fitmentFixture, "catalog-models.jsonl"), filepath.Join(fitmentFixture, "part-codes.jsonl")}
	drifting := catalogWith(
		`{"redirect":{"vehicle":{"make":"Dodge","model":"Ram 1500"},"from":"engine","to":"mirror","first":2015}}`,
		`{"fitment":{"type":"mirror","number":"MR-9001","side":"left"},"filed_under":[{"code":"MIRX"}],"applications":[{"vehicle":{"make":"Dodge","model":"Dakota"},"first":2009,"last":2012,"options":""}]}`,
		`{"fitment":{"type":"mirror","number":"MR-9002","side":"left"},"applications":[{"vehicle":{"make":"Dodge","model":"Ram 1500"},"first":2009,"last":2012,"options":""}]}`)
	faulty := catalogWith(`{"fitment":{"type":"mirror","number":"MR-1001","side":"left"},"applications":[]}`, `{"fitment":`)
	for _, tc := range []struct {
		name  string
		paths []string
		n     int // findings, unchecked lines aside
	}{
		{"clean", []string{fitmentFixture}, 0},
		{"drifting", append(slices.Clone(beside), drifting), 4},
		{"faulty", append(slices.Clone(beside), faulty), 2},
		{"beside stock", []string{fitmentFixture, stockFixture, filepath.Join(stockFixture, "drift")}, 1},
		{"drifting beside stock", append(slices.Clone(beside), drifting, stockFixture, filepath.Join(stockFixture, "drift")), 5},
		{"alone", []string{filepath.Join(fitmentFixture, "catalog.jsonl")}, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			r := cli(t, []string{"TMPDIR=" + tmp}, "", append([]string{"vocab", "check"}, tc.paths...)...)
			got := slices.DeleteFunc(findings(t, r.stdout), func(f vocab.Finding) bool { return f.Severity == vocab.SeverityUnchecked })
			want := whole(tc.paths)
			if len(want) != tc.n || !reflect.DeepEqual(got, want) || r.stderr != "" {
				t.Errorf("vocab check: exit %d, stderr %q, findings\n%#v\nwant the %d of the files loaded whole:\n%#v", r.exit, r.stderr, got, tc.n, want)
			}
			if left, err := os.ReadDir(tmp); err != nil || len(left) > 0 {
				t.Errorf("TMPDIR after vocab check: %v, %v; want it empty", left, err)
			}
		})
	}
	for _, dir := range []string{fitmentFixture, drifting} {
		if entries, err := os.ReadDir(dir); err != nil || slices.ContainsFunc(entries, func(e os.DirEntry) bool { return !strings.HasSuffix(e.Name(), ".jsonl") }) {
			t.Errorf("%s after vocab check: %v, %v; want its files alone", dir, entries, err)
		}
	}
}

// TestVocabCheckDriftKinds: each drift line of `vocab check` names its
// kind, by the strings /v1/info counts, and a fault line has no kind
// (#282). The drift fixture has one of each vocabulary kind; the catalog
// and stock kinds come from the fitment and stock fixtures, each given one
// more line that drifts.
func TestVocabCheckDriftKinds(t *testing.T) {
	fixtureData(t)
	dir := t.TempDir()
	copyWith := func(src string, extra ...string) {
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range extra {
			data = append(data, l+"\n"...)
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(src)), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	copyWith(filepath.Join(fitmentFixture, "part-types.jsonl"))
	copyWith(filepath.Join(fitmentFixture, "catalog-models.jsonl"))
	copyWith(filepath.Join(fitmentFixture, "part-codes.jsonl"))
	copyWith(filepath.Join(fitmentFixture, "catalog.jsonl"),
		`{"redirect":{"vehicle":{"make":"Dodge","model":"Ram 1500"},"from":"engine","to":"mirror","first":2015}}`,
		`{"fitment":{"type":"mirror","number":"MR-9001","side":"left"},"filed_under":[{"code":"MIRX"}],"applications":[{"vehicle":{"make":"Dodge","model":"Dakota"},"first":2009,"last":2012,"options":""}]}`,
		`{"fitment":{"type":"mirror","number":"MR-9002","side":"left"},"applications":[{"vehicle":{"make":"Dodge","model":"Ram 1500"},"first":2009,"last":2012,"options":""}]}`)
	copyWith(filepath.Join(stockFixture, "drift", "shelf-orphans.jsonl"),
		`{"id":"O-3","location":"south","type":"mirror","number":"MR-1001","side":"left","fields":{"seller_email":"x"}}`)

	r := cli(t, nil, "", "vocab", "check", driftFixture, dir)
	kinds := map[string]bool{}
	for line := range strings.Lines(r.stdout) {
		var f map[string]any
		if err := json.Unmarshal([]byte(line), &f); err != nil {
			t.Fatalf("not a JSON line: %q", line)
		}
		kind, ok := f["kind"].(string)
		if f["severity"] != "drift" || !ok {
			t.Errorf("want a drift line with a kind: %q", line)
		}
		kinds[kind] = true
	}
	want := []string{"extension", "filed-under", "make", "model", "model-qualifier", "model-years", "orphan", "personal-field", "qualifier", "redirect", "unfiled", "vehicle"}
	if got := slices.Sorted(maps.Keys(kinds)); r.exit != 1 || !slices.Equal(got, want) {
		t.Errorf("vocab check's drift kinds: exit %d, %q; want %q\nstdout:\n%s", r.exit, got, want, r.stdout)
	}

	r = cli(t, nil, "", "vocab", "check", faultyFile(t))
	if r.exit != 1 || !strings.Contains(r.stdout, `"severity":"fault"`) || strings.Contains(r.stdout, `"kind"`) {
		t.Errorf("vocab check on a faulty file: exit %d, stdout %q; want a fault line without a kind", r.exit, r.stdout)
	}
}

// vocabularies reads the names in `nitrovin info`'s vocabularies.
func vocabularies(t *testing.T, stdout string) []string {
	t.Helper()
	var info contract.Info
	if err := json.Unmarshal([]byte(stdout), &info); err != nil {
		t.Fatalf("info: %v: %q", err, stdout)
	}
	names := []string{}
	for _, v := range info.Vocabularies {
		names = append(names, v.Name)
	}
	return names
}

// TestVocabLoad: --vocab (repeatable, on the root and the command), else
// NITROVIN_VOCAB, loads files beside the data on every command; a fault
// refuses with an internal problem naming the flag or variable, the file
// and the line; drift is written to stderr and served.
func TestVocabLoad(t *testing.T) {
	fixtureData(t)
	parts, listing := filepath.Join(fixture, "parts.jsonl"), filepath.Join(fixture, "listing.jsonl")
	missing := filepath.Join(t.TempDir(), "missing.jsonl")
	for _, tc := range []struct {
		env   []string
		args  []string
		names []string
	}{
		{nil, []string{"info"}, []string{}},
		{nil, []string{"--vocab", fixture, "info"}, []string{"classics", "listing", "parts", "yard"}},
		{nil, []string{"--vocab", parts, "info", "--vocab", listing}, []string{"parts", "listing"}},
		{[]string{"NITROVIN_VOCAB=" + parts + string(filepath.ListSeparator) + listing}, []string{"info"}, []string{"parts", "listing"}},
		{[]string{"NITROVIN_VOCAB=" + missing}, []string{"info", "--vocab", parts}, []string{"parts"}},
	} {
		r := cli(t, tc.env, "", tc.args...)
		if r.exit != 0 || r.stderr != "" {
			t.Errorf("%q nitrovin %q: exit %d, stderr %q", tc.env, tc.args, r.exit, r.stderr)
			continue
		}
		if got := vocabularies(t, r.stdout); !slices.Equal(got, tc.names) {
			t.Errorf("%q nitrovin %q: vocabularies %v; want %v", tc.env, tc.args, got, tc.names)
		}
	}

	// Drift: on stderr, one finding per line, and in info.
	// Load leaves model-qualifier drift out of stderr (ADR 0015's cost);
	// info counts it, all five of yard's kinds.
	r := cli(t, []string{"NITROVIN_VOCAB=" + driftFixture}, "", "info")
	if fs := findings(t, r.stderr); r.exit != 0 || len(fs) != 6 || fs[0].Severity != vocab.SeverityDrift {
		t.Errorf("info with the drift fixture: exit %d, stderr %q; want exit 0 and six drift lines", r.exit, r.stderr)
	}
	var info contract.Info
	if err := json.Unmarshal([]byte(r.stdout), &info); err != nil || len(info.Vocabularies) != 2 || len(info.Vocabularies[1].Drift) != 5 || info.Vocabularies[1].Drift[4].Kind != "model-qualifier" {
		t.Errorf("info with the drift fixture: %v, %+v", err, info.Vocabularies)
	}
	clock, _ := contract.ParseNow(now)
	d, err := nitrovin.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	req := contract.Request{VIN: "1HGCM82633A004352", Now: clock}
	if r, w := cli(t, nil, "", "--vocab", driftFixture, "decode", "--now", now, req.VIN), want(t, d, []contract.Request{req}); r.exit != 0 || r.stdout != w || len(findings(t, r.stderr)) != 6 {
		t.Errorf("decode with the drift fixture: exit %d, stdout %q, stderr %q; want the decode, and the drift on stderr", r.exit, r.stdout, r.stderr)
	}

	// Faults refuse every command.
	faulty := faultyFile(t)
	for _, tc := range []struct {
		env    []string
		args   []string
		detail string
	}{
		{nil, []string{"--vocab", faulty, "decode", req.VIN}, "--vocab: " + faulty + ":2: /mappings/0: first year 2014"},
		{nil, []string{"info", "--vocab", fixture, "--vocab", faulty}, "--vocab: " + faulty + ":2: "},
		{nil, []string{"--vocab", faulty, "--version"}, "--vocab: " + faulty + ":2: "},
		{[]string{"NITROVIN_VOCAB=" + faulty}, []string{"stdio"}, "NITROVIN_VOCAB: " + faulty + ":2: "},
		{nil, []string{"--vocab", missing, "info"}, "--vocab: " + missing + ": stat: "},
	} {
		r := cli(t, tc.env, "", tc.args...)
		p := problem(t, r)
		if r.exit != 1 || r.stdout != "" || p.Type != contract.Internal || !strings.HasPrefix(p.Detail, tc.detail) {
			t.Errorf("%q nitrovin %q: exit %d, stdout %q, problem %+v; want exit 1 and an internal problem starting %q", tc.env, tc.args, r.exit, r.stdout, p, tc.detail)
		}
	}
}

// TestServeVocabularies: /v1/info lists the files serve loaded, as
// `nitrovin info` does.
func TestServeVocabularies(t *testing.T) {
	fixtureData(t)
	d, err := nitrovin.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	files, err := vocab.Open(driftFixture)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := nitrovin.NewTranslator(d, files...)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newAPI(loaded{tr.Decoder(), tr}, log.New(io.Discard, "", 0), io.Discard, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a)
	t.Cleanup(srv.Close)
	resp, err := client.Get(srv.URL + "/v1/info")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var info contract.Info
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || !reflect.DeepEqual(info.Vocabularies, tr.Info().Vocabularies) {
		t.Errorf("GET /v1/info: status %d, vocabularies %+v; want %+v", resp.StatusCode, info.Vocabularies, tr.Info().Vocabularies)
	}
}

// TestVocabCheckCatalog: `vocab check` checks a catalog with the files it
// names: clean, refused without its spine and vehicle vocabulary, and
// drifting; `info` and GET /v1/info list it with its release and counts.
func TestVocabCheckCatalog(t *testing.T) {
	fixtureData(t)
	if r := cli(t, nil, "", "vocab", "check", fixture, fitmentFixture); r.exit != 0 || r.stdout != "" || r.stderr != "" {
		t.Errorf("vocab check on the catalog fixture: exit %d, stdout %q, stderr %q; want exit 0 and nothing", r.exit, r.stdout, r.stderr)
	}

	catalog := filepath.Join(fitmentFixture, "catalog.jsonl")
	r := cli(t, nil, "", "vocab", "check", catalog)
	var got []string
	for _, f := range findings(t, r.stdout) {
		got = append(got, string(f.Severity)+" "+strconv.Itoa(f.Line)+": "+f.Detail)
	}
	if want := []string{`fault 1: /spine: spine "part-types" is not loaded`, `fault 1: /vehicles: vocabulary "catalog-models" is not loaded`, `fault 1: /filed_under: vocabulary "part-codes" is not loaded`, `unchecked 0: its drift was not measured: it has faults`}; r.exit != 1 || !slices.Equal(got, want) {
		t.Errorf("vocab check on the catalog alone: exit %d, findings %v; want exit 1 and %v", r.exit, got, want)
	}

	dir := t.TempDir()
	data, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, `{"redirect":{"vehicle":{"make":"Dodge","model":"Ram 1500"},"from":"engine","to":"mirror","first":2015}}`+"\n"...)
	if err := os.WriteFile(filepath.Join(dir, "catalog.jsonl"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	r = cli(t, nil, "", "vocab", "check", filepath.Join(fitmentFixture, "part-types.jsonl"), filepath.Join(fitmentFixture, "catalog-models.jsonl"), filepath.Join(fitmentFixture, "part-codes.jsonl"), dir)
	fs := findings(t, r.stdout)
	if r.exit != 1 || len(fs) != 1 || fs[0].Severity != vocab.SeverityDrift || fs[0].Line != 9 || fs[0].Detail != `/redirect/to: part type "mirror" has no application for this vehicle in 2015 on` {
		t.Errorf("vocab check on a drifting catalog: exit %d, findings %+v", r.exit, fs)
	}

	r = cli(t, nil, "", "--vocab", fitmentFixture, "info")
	var info contract.Info
	if err := json.Unmarshal([]byte(r.stdout), &info); err != nil || r.exit != 0 {
		t.Fatalf("info with the catalog: exit %d, %v, %q", r.exit, err, r.stdout)
	}
	i := slices.IndexFunc(info.Vocabularies, func(v contract.VocabularyFile) bool { return v.Kind == "catalog" })
	if i < 0 || info.Vocabularies[i].Release != "7.02" || info.Vocabularies[i].FiledUnder != "part-codes" || info.Vocabularies[i].CatalogCounts == nil || info.Vocabularies[i].Fitments != 5 || info.Vocabularies[i].Filings != 6 {
		t.Errorf("info's catalog: %+v", info.Vocabularies)
	}

	d, err := nitrovin.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	files, err := vocab.Open(fitmentFixture)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := nitrovin.NewTranslator(d, files...)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newAPI(loaded{tr.Decoder(), tr}, log.New(io.Discard, "", 0), io.Discard, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a)
	t.Cleanup(srv.Close)
	resp, err := client.Get(srv.URL + "/v1/info")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var served contract.Info
	if err := json.NewDecoder(resp.Body).Decode(&served); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || !reflect.DeepEqual(served.Vocabularies, tr.Info().Vocabularies) {
		t.Errorf("GET /v1/info: status %d, vocabularies %+v; want %+v", resp.StatusCode, served.Vocabularies, tr.Info().Vocabularies)
	}
}

// TestVocabCompile: `vocab compile` checks its arguments as `vocab check`
// does, refuses on a fault and writes nothing, and otherwise writes the
// catalog compiled (ADR 0021), the same bytes each time; --vocab loads the
// compiled file, refusing it without its spine; `info` lists it with the
// drift its compiling stored; and `vocab check` reads it in full and
// measures it again, as its source.
func TestVocabCompile(t *testing.T) {
	fixtureData(t)
	out := filepath.Join(t.TempDir(), "catalog.compiled")
	catalog := filepath.Join(fitmentFixture, "catalog.jsonl")
	spine, models, codes := filepath.Join(fitmentFixture, "part-types.jsonl"), filepath.Join(fitmentFixture, "catalog-models.jsonl"), filepath.Join(fitmentFixture, "part-codes.jsonl")

	r := cli(t, nil, "", "vocab", "compile", "-o", out, catalog)
	var got []string
	for _, f := range findings(t, r.stdout) {
		got = append(got, string(f.Severity)+" "+strconv.Itoa(f.Line)+": "+f.Detail)
	}
	if want := []string{`fault 1: /spine: spine "part-types" is not loaded`, `fault 1: /vehicles: vocabulary "catalog-models" is not loaded`, `fault 1: /filed_under: vocabulary "part-codes" is not loaded`}; r.exit != 1 || !slices.Equal(got, want) {
		t.Errorf("vocab compile of the catalog alone: exit %d, findings %v; want exit 1 and %v", r.exit, got, want)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("a refused compile wrote %s: %v", out, err)
	}

	var compiled [2][]byte
	for i := range compiled {
		if r := cli(t, nil, "", "vocab", "compile", fitmentFixture, "-o", out); r.exit != 0 || r.stdout != "" || r.stderr != "" {
			t.Fatalf("vocab compile: exit %d, stdout %q, stderr %q", r.exit, r.stdout, r.stderr)
		}
		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		compiled[i] = b
	}
	if !bytes.Equal(compiled[0], compiled[1]) || !bytes.HasPrefix(compiled[0], []byte("nvcatalg")) {
		t.Errorf("two compiles differ, or are not compiled catalogs: %d and %d bytes", len(compiled[0]), len(compiled[1]))
	}

	r = cli(t, nil, "", "--vocab", spine, "--vocab", models, "--vocab", codes, "--vocab", out, "info")
	var info contract.Info
	if err := json.Unmarshal([]byte(r.stdout), &info); err != nil || r.exit != 0 || r.stderr != "" || len(info.Vocabularies) != 4 {
		t.Fatalf("info with the compiled catalog: exit %d, %v, stdout %q, stderr %q", r.exit, err, r.stdout, r.stderr)
	}
	source, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	src, err := vocab.Read(catalog, bytes.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	c := info.Vocabularies[3]
	if c.Kind != "catalog" || c.Path != out || c.Fitments != 5 || c.Filings != 6 || c.FiledUnder != "part-codes" || c.Compiled == nil || c.Compiled.SourceBlake3 != fmt.Sprintf("%x", src.Hash) ||
		c.Compiled.Writer != nitrovin.Version() || c.Compiled.DriftMonth != embedded.Month || c.Compiled.DriftMonthDiffers || len(c.Drift) != 0 {
		t.Errorf("info's compiled catalog: %+v, compiled %+v", c, c.Compiled)
	}

	r = cli(t, nil, "", "--vocab", models, "--vocab", out, "info")
	if p := problem(t, r); r.exit != 1 || !strings.Contains(p.Detail, out+`: /spine: spine "part-types" is not loaded`) {
		t.Errorf("info with the compiled catalog and no spine: exit %d, problem %+v", r.exit, p)
	}

	// fit answers from the compiled catalog as from its source.
	fitArgs := []string{"fit", "--part", "part-types:mirror", "--now", now, "1D7RV1GT5AS123456"}
	fromJSONL := cli(t, nil, "", append([]string{"--vocab", fitmentFixture}, fitArgs...)...)
	fromCompiled := cli(t, nil, "", append([]string{"--vocab", spine, "--vocab", models, "--vocab", codes, "--vocab", out}, fitArgs...)...)
	if fromCompiled.exit != 0 || fromJSONL.exit != 0 || fromCompiled.stdout != fromJSONL.stdout || !strings.Contains(fromCompiled.stdout, `"MR-1001"`) {
		t.Errorf("fit with the compiled catalog: exit %d, stdout %q, stderr %q; want exit 0 and %q", fromCompiled.exit, fromCompiled.stdout, fromCompiled.stderr, fromJSONL.stdout)
	}
	if r := cli(t, nil, "", "vocab", "check", spine, models, codes, out); r.exit != 0 || r.stdout != "" || r.stderr != "" {
		t.Errorf("vocab check on the compiled catalog: exit %d, stdout %q, stderr %q; want exit 0 and nothing", r.exit, r.stdout, r.stderr)
	}
	// Stock on the compiled catalog: check finds the orphan as on its
	// source, and a fit answers the shelf row under a retired number alike.
	orphans := filepath.Join(stockFixture, "drift")
	fromJSONL = cli(t, nil, "", "vocab", "check", fitmentFixture, stockFixture, orphans)
	r = cli(t, nil, "", "vocab", "check", spine, models, codes, out, stockFixture, orphans)
	if fs := findings(t, r.stdout); r.exit != 1 || fromJSONL.exit != 1 || r.stdout != fromJSONL.stdout || len(fs) != 1 || fs[0].Line != 2 || !strings.Contains(r.stdout, `MR-0000`) {
		t.Errorf("vocab check on the compiled catalog and stock: exit %d, stdout %q; want exit 1 and %q", r.exit, r.stdout, fromJSONL.stdout)
	}
	stockArgs := []string{"fit", "--part", "part-types:brake-rotor-front", "--stock", "shelf,yard", "--now", now, "1D7RV1GT5AS123456"}
	fromJSONL = cli(t, nil, "", append([]string{"--vocab", fitmentFixture, "--vocab", stockFixture}, stockArgs...)...)
	fromCompiled = cli(t, nil, "", append([]string{"--vocab", spine, "--vocab", models, "--vocab", codes, "--vocab", out, "--vocab", stockFixture}, stockArgs...)...)
	if fromCompiled.exit != 0 || fromCompiled.stdout != fromJSONL.stdout || !strings.Contains(fromCompiled.stdout, `"id":"S-3","location":"north","matched":"fitment","fitment":{"type":"brake-rotor-front","number":"BR-1999"},"reached":"retired"`) {
		t.Errorf("fit with stock on the compiled catalog: exit %d, stdout %q, stderr %q; want %q", fromCompiled.exit, fromCompiled.stdout, fromCompiled.stderr, fromJSONL.stdout)
	}
	// A drifting catalog: check finds the same drift in either form, and
	// compile writes it to stderr.
	dir := t.TempDir()
	data := append(slices.Clone(source), `{"redirect":{"vehicle":{"make":"Dodge","model":"Ram 1500"},"from":"engine","to":"mirror","first":2015}}`+"\n"...)
	if err := os.WriteFile(filepath.Join(dir, "catalog.jsonl"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	r = cli(t, nil, "", "vocab", "compile", "-o", out, spine, models, codes, dir)
	if fs := findings(t, r.stderr); r.exit != 0 || r.stdout != "" || len(fs) != 1 || fs[0].Line != 9 || fs[0].Severity != vocab.SeverityDrift {
		t.Errorf("vocab compile of a drifting catalog: exit %d, stdout %q, stderr %q", r.exit, r.stdout, r.stderr)
	}
	jsonl := cli(t, nil, "", "vocab", "check", spine, models, codes, dir)
	r = cli(t, nil, "", "vocab", "check", spine, models, codes, out)
	if want := strings.ReplaceAll(jsonl.stdout, strconv.Quote(filepath.Join(dir, "catalog.jsonl")), strconv.Quote(out)); r.exit != 1 || jsonl.exit != 1 || r.stdout != want || len(findings(t, r.stdout)) != 1 {
		t.Errorf("vocab check on the compiled drifting catalog: exit %d, stdout %q; want exit 1 and %q", r.exit, r.stdout, want)
	}

	for _, args := range [][]string{{"vocab", "compile", fitmentFixture}, {"vocab", "check", "-o", out, fitmentFixture}, {"vocab", "compile", "-o", out}} {
		if r := cli(t, nil, "", args...); r.exit != 2 || r.stdout != "" || !strings.Contains(r.stderr, "usage: nitrovin vocab check") {
			t.Errorf("nitrovin %q: exit %d, stdout %q, stderr %q; want a usage error", args, r.exit, r.stdout, r.stderr)
		}
	}
}
