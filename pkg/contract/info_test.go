package contract

import (
	"encoding/json"
	"flag"
	"os"
	"runtime/debug"
	"slices"
	"strings"
	"testing"

	"github.com/joshuasprow/nitrovin-data/pkg/artifact"
)

// elementsFixture is the pinned data month's public element table the
// layout tests run on, as PublicElements reads it from that month's
// artifact. Regenerate it with
//
//	NITROVIN_ARTIFACT=$(go mod download -json github.com/joshuasprow/nitrovin-data | jq -r .Dir)/pkg/embedded/nitrovin.artifact \
//	  go test ./pkg/contract -run 'TestInfo$' -update-elements
const elementsFixture = "testdata/elements.json"

var updateElements = flag.Bool("update-elements", false, "regenerate "+elementsFixture+" from $NITROVIN_ARTIFACT")

// pinnedMonth is the data month go.mod pins: nitrovin-data v1.YYYYMM.N is
// month YYYY_MM.
func pinnedMonth(t *testing.T) string {
	t.Helper()
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		t.Fatal("no build info")
	}
	for _, m := range bi.Deps {
		if m.Path != "github.com/joshuasprow/nitrovin-data" {
			continue
		}
		if v := strings.Split(m.Version, "."); len(v) == 3 && len(v[1]) == 6 {
			return v[1][:4] + "_" + v[1][4:]
		}
		t.Fatalf("nitrovin-data version %s names no month", m.Version)
	}
	t.Fatal("nitrovin-data is not in the build info")
	return ""
}

func loadElements(t *testing.T) Elements {
	t.Helper()
	b, err := os.ReadFile(elementsFixture)
	if err != nil {
		t.Fatal(err)
	}
	var es Elements
	if err := json.Unmarshal(b, &es); err != nil {
		t.Fatal(err)
	}
	return es
}

// TestInfo checks NewInfo on a real artifact: the Info validates against
// the contract's schema, its element table is the fixture the layout tests
// use, and it has the public and multi-valued elements docs/api/README.md
// names. All three hold for the pinned month; another month's artifact
// skips.
func TestInfo(t *testing.T) {
	path := os.Getenv("NITROVIN_ARTIFACT")
	if path == "" {
		t.Skip("NITROVIN_ARTIFACT not set: no artifact to read the element table from")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	a, err := artifact.Open(data)
	if err != nil {
		t.Fatal(err)
	}
	info, err := NewInfo(a, "/data/vpic.nitrovin", "(devel)")
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	if err := validate(schemaDef(t, "Info"), b); err != nil {
		t.Errorf("info does not validate: %v", err)
	}

	got, err := json.MarshalIndent(info.Elements, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	if month := pinnedMonth(t); a.Month() != month {
		t.Skipf("the fixture and README counts are the pinned month %s's; the artifact is %s", month, a.Month())
	}
	if *updateElements {
		if err := os.WriteFile(elementsFixture, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(elementsFixture)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("PublicElements differs from %s; regenerate it with -update-elements", elementsFixture)
	}

	var multi []int32
	for _, e := range info.Elements {
		if e.MultiValued {
			multi = append(multi, e.ElementID)
		}
	}
	if len(info.Elements) != 141 {
		t.Errorf("%d public elements, want 141", len(info.Elements))
	}
	if want := []int32{1, 114, 121, 129, 150, 154, 155, 169}; !slices.Equal(multi, want) {
		t.Errorf("multi-valued elements %v, want %v", multi, want)
	}
}

// TestInfoVocabularies: an Info listing loaded files, one with drift and
// one without, validates against the contract's schema, and so does one
// with none.
func TestInfoVocabularies(t *testing.T) {
	info := Info{
		DecoderVersion: "(devel)",
		DataMonth:      "2026_10",
		ArtifactHash:   "blake3:7518c5b46c4264fe94a8e5158cb1949b000e50803b36d388fec9ebe914a780cb",
		DataSource:     DataSourceEmbedded,
		Elements:       loadElements(t),
		Vocabularies: []VocabularyFile{
			{Name: "parts", Kind: "spine", Version: "2026.11", Blake3: "8f91f3ee24e7fc60ee1bb813ea5302bbb323175ce323146683801b5718e17afa", Path: "/vocab/parts.jsonl", Drift: []DriftCount{}},
			{Name: "fleet", Kind: "vocabulary", Version: "2026.11", Blake3: "579d688449a9bc496d26049222802c68af4ea6968257f0d97e54bcbad707d8e2", Path: "/vocab/fleet.jsonl", Drift: []DriftCount{
				{Kind: "model", Count: 4, Examples: []DriftExample{{Line: 5, Detail: "/mappings/0/model: vPIC model 99999 is not in data month 2026_10"}}},
			}},
		},
	}
	for _, vs := range [][]VocabularyFile{info.Vocabularies, {}} {
		info.Vocabularies = vs
		b, err := json.Marshal(info)
		if err != nil {
			t.Fatal(err)
		}
		if err := validate(schemaDef(t, "Info"), b); err != nil {
			t.Errorf("info with %d vocabularies does not validate: %v", len(vs), err)
		}
	}
}
