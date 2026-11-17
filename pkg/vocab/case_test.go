package vocab

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A case-exact vocabulary (ADR 0026): a parts system's vehicle codes,
// where OLD3 and old3 name different vehicles, and its part codes, where
// PD and pd are different codes of one part type.
var (
	exactCodes = file{"e.jsonl", lines(`{"format":1,"kind":"vocabulary","name":"codes","version":"1","subject":"vehicle","fields":["code"],"case":"exact"}`,
		`{"entry":{"code":"OLD3"},"mappings":[{"model":1938}]}`,
		`{"entry":{"code":"old3"},"mappings":[{"make":476}]}`)}
	exactParts = file{"f.jsonl", lines(`{"format":1,"kind":"vocabulary","name":"system","version":"1","subject":"part","fields":["code"],"spine":"parts","case":"exact"}`,
		`{"entry":{"code":"PD"},"mappings":[{"id":"pads"}]}`,
		`{"entry":{"code":"pd"},"mappings":[{"id":"pads"}]}`)}
	// exactCatalog names both vehicles and both part codes.
	exactCatalog = file{"g.jsonl", lines(`{"format":1,"kind":"catalog","name":"cat","version":"1","spine":"parts","vehicles":"codes","filed_under":"system"}`,
		`{"fitment":{"type":"pads","number":"P1"},"filed_under":[{"code":"PD"},{"code":"pd"}],"applications":[{"vehicle":{"code":"OLD3"},"options":""},{"vehicle":{"code":"old3"},"options":""}]}`,
		`{"fitment":{"type":"pads","number":"P2"},"filed_under":[{"code":" pd "}],"applications":[{"vehicle":{"code":" old3"},"options":""}]}`)}
)

// TestCaseExact: a case-exact vocabulary matches an entry after trimming
// and collapsing white space, never folding case: entries that differ only
// in case are two, a spelling it does not hold is unknown, and a duplicate
// is one equal after that normalization. Translate and list follow it.
func TestCaseExact(t *testing.T) {
	_, files, err := openFiles(t, []file{exactCodes})
	if err != nil {
		t.Fatal(err)
	}
	f := files[0]
	if f.Case != CaseExact {
		t.Errorf("case %q; want %q", f.Case, CaseExact)
	}
	for spelling, line := range map[string]int{"OLD3": 2, " OLD3 ": 2, "old3": 3, "old3\t": 3, "Old3": 0, "OLD 3": 0} {
		e, ok := f.Entry(map[string]string{"code": spelling})
		if got := 0; ok {
			got = e.Line
			if got != line {
				t.Errorf("entry %q: line %d; want %d", spelling, got, line)
			}
		} else if line != 0 {
			t.Errorf("entry %q: unknown; want line %d", spelling, line)
		}
	}

	set, err := NewSet(month{}, files)
	if err != nil {
		t.Fatal(err)
	}
	for spelling, want := range map[string][]string{"OLD3": {"model:1938", "make:476"}, "old3": {"make:476"}, "Old3": nil} {
		tr, err := set.Translate(Query{From: From{Vocabulary: "codes", Entry: map[string]string{"code": spelling}}})
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, r := range tr.Spine {
			got = append(got, refText(r.Ref))
		}
		if !slices.Equal(got, want) || (tr.Entry == nil) != (want == nil) {
			t.Errorf("translate from %q: entry %v, spine %q; want %q", spelling, tr.Entry, got, want)
		}
	}
	l, err := set.List(ListQuery{Of: "codes"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := names(l), []string{"OLD3", "old3"}; !slices.Equal(got, want) {
		t.Errorf("list of codes: %q; want %q", got, want)
	}
	for spelling, want := range map[string][]string{" old3": {"old3"}, "OLD3": {"OLD3"}, "Old3": nil} {
		l, err := set.List(ListQuery{Of: "codes", Where: map[string]string{"code": spelling}})
		if err != nil {
			t.Fatal(err)
		}
		if got := names(l); !slices.Equal(got, want) {
			t.Errorf("list of codes where %q: %q; want %q", spelling, got, want)
		}
	}

	// A duplicate is equal after trimming and collapsing white space; a
	// vocabulary that folds case finds OLD3 and old3 one entry.
	for _, tc := range []struct {
		name   string
		header string
		line   string
	}{
		{"exact", exactCodes.lines[0], `{"entry":{"code":" old3 "},"mappings":[]}`},
		{"folded", strings.Replace(exactCodes.lines[0], `,"case":"exact"`, ``, 1), ""},
	} {
		ls := slices.Clone(exactCodes.lines)
		ls[0] = tc.header
		if tc.line != "" {
			ls = append(ls, tc.line)
		}
		_, _, err := openFiles(t, []file{{"e.jsonl", ls}})
		faults, ok := errors.AsType[Faults](err)
		if !ok || len(faults) != 1 || !strings.Contains(faults[0].Detail, "duplicate entry: line ") {
			t.Errorf("%s: %v; want one duplicate entry", tc.name, err)
		}
	}
}

// TestCaseExactCatalog: a catalog whose vehicle and filed-under
// vocabularies are case-exact keeps vehicles and entries that differ only
// in case apart, as JSON Lines and compiled; one spelled otherwise is
// drift, an unknown spelling; and a stock row names them so too.
func TestCaseExactCatalog(t *testing.T) {
	_, files, err := openFiles(t, []file{partSpine, exactCodes, exactParts, exactCatalog})
	if err != nil {
		t.Fatal(err)
	}
	if d := Drift(month{}, files); len(d) > 0 {
		t.Errorf("drift %v; want none", d)
	}
	compiled := swapCatalog(t, files, compile(t, month{}, files))
	for name, files := range map[string][]*File{"JSON Lines": files, "compiled": compiled} {
		set, err := NewSet(month{}, files)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		c := set.Catalog()
		if info := c.Info(); info.VehiclesCase != CaseExact || info.FiledUnderCase != CaseExact {
			t.Errorf("%s: info %+v; want both case-exact", name, info)
		}
		if n := len(c.Vehicles()); n != 2 {
			t.Errorf("%s: %d vehicles; want OLD3 and old3", name, n)
		}
		for spelling, want := range map[string]string{"OLD3": "P1:a", "old3": "P1:a P2:a", " old3 ": "P1:a P2:a", "Old3": ""} {
			cs := c.ByVehicle("pads", []Vehicle{{Entry: map[string]string{"code": spelling}}}, nil, nil)
			if got := strings.Join(numbers(cs), " "); got != want {
				t.Errorf("%s: by vehicle %q: %q; want %q", name, spelling, got, want)
			}
		}
		for spelling, want := range map[string]int{"PD": 1, "pd": 1, "Pd": 0} {
			if got := c.ByFiled(map[string]string{"code": spelling}, "P1", nil); len(got) != want {
				t.Errorf("%s: by filed %q: %d fitments; want %d", name, spelling, len(got), want)
			}
		}
		p1, _ := c.ByNumber("pads", "P1", SideNone)
		p2, _ := c.ByNumber("pads", "P2", SideNone)
		if p1 == nil || p2 == nil || len(p1.FiledUnder) != 2 || len(p2.FiledUnder) != 1 || p2.FiledUnder[0] != p1.FiledUnder[1] {
			t.Errorf("%s: P1 and P2 filed under %v and %v; want PD and pd, and pd", name, p1, p2)
		}
	}

	// Another spelling is drift: the vocabulary lacks it.
	odd := file{exactCatalog.name, append(slices.Clone(exactCatalog.lines),
		`{"fitment":{"type":"pads","number":"P3"},"filed_under":[{"code":"Pd"}],"applications":[{"vehicle":{"code":"Old3"},"options":""}]}`)}
	_, files, err = openFiles(t, []file{partSpine, exactCodes, exactParts, odd})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, d := range Drift(month{}, files) {
		got = append(got, d.String()[len(filepath.Dir(d.File))+1:])
	}
	want := []string{
		`g.jsonl:4: /applications/0/vehicle: vocabulary "codes" has no entry {"code":"Old3"}`,
		`g.jsonl:4: /filed_under: vocabulary "system" has no entry {"code":"Pd"}`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("drift\n%q\nwant\n%q", got, want)
	}

	// A stock row names an entry as the vocabulary does.
	shelf := func(vehicle string) file {
		return file{"h.jsonl", lines(`{"format":1,"kind":"stock","name":"shelf","version":"1","source":"shelf","as_of":"2026-11-01","catalog":"cat"}`,
			`{"id":"S-1","location":"a","part":{"vocabulary":"system","entry":{"code":"pd"}},"vehicle":{"code":"`+vehicle+`"},"year":2010}`)}
	}
	if _, files, err := openFiles(t, []file{partSpine, exactCodes, exactParts, exactCatalog, shelf("old3")}); err != nil {
		t.Errorf("a shelf row of old3: %v", err)
	} else if r := files[4].Stock[0]; r.Vehicle == nil || r.Vehicle.Entry["code"] != "old3" || !slices.Equal(r.PartTypes, []string{"pads"}) {
		t.Errorf("a shelf row of old3: %+v", r)
	}
	_, _, err = openFiles(t, []file{partSpine, exactCodes, exactParts, exactCatalog, shelf("Old3")})
	if faults, ok := errors.AsType[Faults](err); !ok || len(faults) != 1 || !strings.Contains(faults[0].Detail, `/vehicle: vocabulary "codes" has no entry {"code":"Old3"}`) {
		t.Errorf("a shelf row of Old3: %v; want a fault, the vocabulary lacks it", err)
	}
}

// TestCaseFoldedCatalog: beside a vocabulary that folds case, a catalog
// finds OLD3 and old3 one vehicle, and an application to each of one
// fitment a duplicate.
func TestCaseFoldedCatalog(t *testing.T) {
	folded := file{"e.jsonl", lines(strings.Replace(exactCodes.lines[0], `,"case":"exact"`, ``, 1), exactCodes.lines[1])}
	cat := file{"g.jsonl", lines(`{"format":1,"kind":"catalog","name":"cat","version":"1","spine":"parts","vehicles":"codes"}`,
		`{"fitment":{"type":"pads","number":"P1"},"applications":[{"vehicle":{"code":"OLD3"},"options":""},{"vehicle":{"code":"old3"},"options":""}]}`)}
	_, _, err := openFiles(t, []file{partSpine, folded, cat})
	if faults, ok := errors.AsType[Faults](err); !ok || len(faults) != 1 || !strings.Contains(faults[0].Detail, "/applications/1: duplicate application") {
		t.Errorf("Open: %v; want the second application a duplicate", err)
	}
}

// TestCompiledCaseRecorded: a compiled catalog records the case modes of
// the vocabularies it was compiled against (format 5, ADR 0026), and is
// refused beside a vocabulary of the other mode, the fault naming it and
// both modes.
func TestCompiledCaseRecorded(t *testing.T) {
	_, files, err := openFiles(t, []file{partSpine, exactCodes, exactParts, exactCatalog})
	if err != nil {
		t.Fatal(err)
	}
	path := compile(t, month{}, files)
	folded := func(f file) file {
		return file{f.name, []string{strings.Replace(f.lines[0], `,"case":"exact"`, ``, 1), f.lines[1]}}
	}
	for _, tc := range []struct {
		name   string
		files  []file
		detail string
	}{
		{"vehicles folded", []file{partSpine, folded(exactCodes), exactParts},
			`/vehicles: vocabulary "codes" folds case, and the catalog was compiled against it matching case exactly: compile it again beside the vocabulary`},
		{"filed under folded", []file{partSpine, exactCodes, folded(exactParts)},
			`/filed_under: vocabulary "system" folds case, and the catalog was compiled against it matching case exactly: compile it again beside the vocabulary`},
	} {
		dir, _, err := openFiles(t, tc.files)
		if err != nil {
			t.Fatal(err)
		}
		var paths []string
		for _, f := range tc.files {
			paths = append(paths, filepath.Join(dir, f.name))
		}
		_, err = Open(append(paths, path)...)
		faults, ok := errors.AsType[Faults](err)
		if !ok || len(faults) != 1 || faults[0].File != path || faults[0].Line != 0 || faults[0].Detail != tc.detail {
			t.Errorf("%s: %v; want the fault %q", tc.name, err, tc.detail)
		}
	}
}
