package ui

import (
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/joshuasprow/nitrovin/pkg/contract"
)

// hondaInfo is testInfo with the elements the plate's cards read, and
// hondaDecode a decode of them: what backend answers every VIN with, so
// the page's tests say what the page draws of a decode (the decodes
// themselves are cmd/nitrovin's TestForm).
var (
	hondaInfo = func() contract.Info {
		info := testInfo
		info.Elements = contract.Elements{
			{ElementID: elementMake, Variable: "Make", Code: "Make", DataType: "lookup"},
			{ElementID: elementModel, Variable: "Model", Code: "Model", DataType: "lookup"},
			{ElementID: elementPlantCountry, Variable: "Plant Country", Code: "PlantCountry", DataType: "lookup"},
		}
		return info
	}()
	hondaDecode = contract.Decode{
		ErrorCodes: []int{0}, CheckDigitValid: true, ModelYear: contract.Null[int32]{V: 2003, Valid: true},
		Attributes: contract.VpicAttributes{"Make": "HONDA", "Model": "Accord", "PlantCountry": "UNITED STATES (USA)"},
	}
)

// TestPage walks the page on a stub backend (the decodes themselves are
// cmd/nitrovin's TestForm): / is the empty plate, and a VIN's answer below
// it, as a page and as a fragment for the page's main, where htmx is sent
// with a refusal too, while a fault is the error page or the error slot's
// fragment. The header's buttons and the vocabularies' dialog are the
// whole page's, never a fragment's.
func TestPage(t *testing.T) {
	h := New(backend{info: hondaInfo, decode: &hondaDecode}, log.New(io.Discard, "", 0))

	status, hdr, body := response(t, h, "GET", "/", nil)
	if status != 200 {
		t.Errorf("GET /: %d; want 200", status)
	}
	checkHTML(t, "GET /", hdr)
	checkPage(t, "GET /", body, "VIN decoder")
	for _, want := range []string{
		`<form id="vin-form" class="plate-form" method="get" action="/"`,
		`<input type="text" id="from-vin" name="from.vin" value=""`,
		`<button type="submit" class="decode">Decode</button>`,
		`<button type="button" class="open-vins" popovertarget="vins">VINs</button>`,
		`<div id="vins" class="menu" popover`,
		`<ul id="recent" `,
		`<a class="example" href="/?from.vin=` + exampleVINs[0].vin + `"`,
		`<a class="example surprise" href="/?surprise">`,
		`id="clear-recent"`,
		`<button type="button" class="open-vocabularies" popovertarget="vocabularies">Vocabularies</button>`,
		`<dialog id="vocabularies" popover aria-labelledby="browser-h">`,
		`<button type="button" class="open-developers" popovertarget="developers">Developers</button>`,
		`<div id="dev"><dialog id="developers" popover`,
		testAssetsAttr + `/vocab.js" defer></script>`,
		"Names turn up once the model is known.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GET /: lacks %s\n%s", want, body)
		}
	}
	if n := strings.Count(body, `<span class="cell g-`); n != vinLength || strings.Contains(body, " filled") || strings.Contains(body, `id="result-json"`) {
		t.Errorf("GET /: %d cells; want %d, all empty, and no answer\n%s", n, vinLength, body)
	}

	// Surprise me moves to one of its VINs, on the plate.
	status, hdr, _ = response(t, h, "GET", "/?surprise", nil)
	if vin, ok := strings.CutPrefix(hdr.Get("Location"), "/?from.vin="); status != http.StatusSeeOther || !ok || !slices.Contains(surpriseVINs, vin) {
		t.Errorf("GET /?surprise: %d, Location %q; want 303 to one of the VINs", status, hdr.Get("Location"))
	}

	// A query that asks no VIN is the page with no answer, and the
	// fragment is the main's alone.
	status, _, body = response(t, h, "GET", "/?from.model=1861", htmx)
	if status != 200 || !strings.Contains(body, `id="from-vin"`) || strings.Contains(body, `id="result-json"`) || strings.Contains(body, `id="vocabularies"`) || strings.Contains(body, "open-vocabularies") {
		t.Errorf("a query without a VIN, as a fragment: %d\n%s", status, body)
	}

	// Old links move to the page with the VIN they named, as the decode
	// form's vin or a translation's from.vin, and nothing else: the page
	// reads nothing else. One that named none moves to the empty page.
	for _, method := range []string{"GET", "HEAD"} {
		for old, want := range map[string]string{
			"/?vin=1HGCM82633A004352":                                "/?from.vin=1HGCM82633A004352",
			"/?vin=1HGCM82633A004352&model_year=2003&shape=nitrovin": "/?from.vin=1HGCM82633A004352",
			"/?vin=1HGCM82633A004352&from.model=1861&to=yard":        "/?from.vin=1HGCM82633A004352",
			"/?vin=": "/",
			"/ui/decode?vin=1HGCM82633A004352&model_year=2003&full=true&now=2026-11-01":     "/?from.vin=1HGCM82633A004352",
			"/ui/decode?vin=1hgcm-82633%20a004352":                                          "/?from.vin=1hgcm-82633+a004352",
			"/ui/decode":                                                                    "/",
			"/ui/translate?from.model=1861&model_year=2003&to=yard":                         "/",
			"/ui/translate?from.vocabulary=yard&vin=1HGCM82633A004352":                      "/?from.vin=1HGCM82633A004352",
			"/ui/translate?from.vin=1HGCM82633A004352&to=yard&to=shelf&tab=spine&full=true": "/?from.vin=1HGCM82633A004352",
			"/ui/translate?from.vin=1HGCM82633A004352&vin=5YJ3E1EA2JF000316&from.make=474":  "/?from.vin=1HGCM82633A004352",
			"/vocab":                                 "/",
			"/vocab?tab=entry&to=yard":               "/",
			"/ui/vocabularies":                       "/",
			"/ui/vocabularies?vin=1HGCM82633A004352": "/?from.vin=1HGCM82633A004352",
		} {
			status, hdr, _ := response(t, h, method, old, nil)
			if status != http.StatusMovedPermanently || hdr.Get("Location") != want {
				t.Errorf("%s %s: %d, Location %q; want 301 to %s", method, old, status, hdr.Get("Location"), want)
			}
		}
	}

	// A VIN's answer: the plate with it, the cards and the names below,
	// which plate.js swaps in by their ids, and in the Developers dialog
	// its own request to copy and its JSON.
	status, hdr, body = response(t, h, "GET", "/?from.vin=1HGCM82633A004352", htmx)
	if status != 200 || !strings.HasPrefix(body, "<title>2003 HONDA Accord · nitrovin</title><h1 ") || !strings.HasSuffix(body, `<div id="ui-error" role="alert" hx-swap-oob="true"></div>`) {
		t.Errorf("a VIN's fragment: %d\n%s", status, body)
	}
	for _, want := range []string{
		`<form id="vin-form" class="plate-form" method="get" action="/" aria-label="Decode a VIN" data-vin="1HGCM82633A004352">`,
		`<div id="answer" class="stage" data-vin="1HGCM82633A004352" data-label="2003 HONDA Accord" data-complete data-done="wmi vds check year plant serial ">`,
		`<section class="aka" id="aka" `,
		"No vocabulary loaded.",
		`<div id="dev"><dialog id="developers" popover`,
		`id="copy-url" data-copy="this page's URL">http://example.com/?from.vin=1HGCM82633A004352</pre>`,
		`id="copy-curl" data-copy="the curl command">curl http://example.com/v1/decode/1HGCM82633A004352</pre>`,
		`id="copy-cli" data-copy="the nitrovin decode command">nitrovin decode 1HGCM82633A004352</pre>`,
		`<button type="button" class="open-dialog" popovertarget="result-json">JSON</button>`,
		`<a href="https://vpic.nhtsa.dot.gov/decoder/VinDecoder?VIN=1HGCM82633A004352">NHTSA's decoder</a>`,
		`<dialog id="result-json" popover aria-labelledby="result-json-h">`,
		`<a class="hint" href="/v1/decode/1HGCM82633A004352" hx-boost="false">/v1</a>`,
		`id="decode-json" data-copy="the decode&#39;s JSON">{` + "\n" + `  &#34;vin&#34;: &#34;1HGCM82633A004352&#34;,`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("a VIN's fragment lacks %s\n%s", want, body)
		}
	}
	if strings.Contains(body, `id="vocabularies"`) {
		t.Errorf("a VIN's fragment holds the vocabularies' dialog\n%s", body)
	}
	checkHTML(t, "a VIN's fragment", hdr)
	_, _, body = response(t, h, "GET", "/?from.vin=1HGCM82633A004352", nil)
	checkPage(t, "a VIN's page", body, "2003 HONDA Accord")

	// A decode's options are not the page's: one the API would refuse is
	// ignored as any unknown parameter is, and the answer is the VIN's.
	for _, fragment := range []http.Header{htmx, nil} {
		status, _, body := response(t, h, "GET", "/?from.vin=1HGCM82633A004352&model_year=2003.5&shape=flat&full=1&now=yesterday", fragment)
		if status != 200 || !strings.Contains(body, `data-vin="1HGCM82633A004352" data-label="2003 HONDA Accord"`) ||
			!strings.Contains(body, `id="copy-url" data-copy="this page's URL">http://example.com/?from.vin=1HGCM82633A004352</pre>`) ||
			!strings.Contains(body, `<a class="hint" href="/v1/decode/1HGCM82633A004352" hx-boost="false">/v1</a>`) ||
			strings.Contains(body, `class="error"`) || strings.Contains(body, "2003.5") || strings.Contains(body, "yesterday") {
			t.Errorf("a VIN with a decode's options (fragment %t): %d\n%s", fragment != nil, status, body)
		}
	}

	// A refusal is the page with the refusal below the plate, for the
	// page's main, not the error slot.
	status, hdr, body = response(t, h, "GET", "/?%zz", htmx)
	if status != 400 || hdr.Get("HX-Retarget") != "#ui-main" || hdr.Get("HX-Reswap") != "innerHTML" || !strings.Contains(body, `<p class="error" role="alert">the query string is malformed: `) ||
		!strings.HasSuffix(body, `hx-swap-oob="true"></div>`) || strings.Contains(body, `id="result-json"`) {
		t.Errorf("refusal fragment: %d, HX-Retarget %q, HX-Reswap %q\n%s", status, hdr.Get("HX-Retarget"), hdr.Get("HX-Reswap"), body)
	}
	status, _, body = response(t, h, "GET", "/?%zz", nil)
	if status != 400 || !strings.Contains(body, `<p class="error" role="alert">the query string is malformed: `) || !strings.Contains(body, `name="from.vin"`) {
		t.Errorf("refusal page: %d\n%s", status, body)
	}

	// A fault in the decoder is nitrovin's: the error page, or the error
	// slot's fragment.
	faulty := New(backend{info: testInfo, err: contract.NewProblem(contract.Internal, "", "injected fault")}, log.New(io.Discard, "", 0))
	status, hdr, body = response(t, faulty, "GET", "/?from.vin=1HGCM82633A004352", htmx)
	if status != 500 || hdr.Get("HX-Retarget") != "" || !strings.HasPrefix(body, "<p><strong>Internal Server Error.</strong> nitrovin failed: ") || !strings.Contains(body, "injected fault") {
		t.Errorf("fault fragment: %d, HX-Retarget %q\n%s", status, hdr.Get("HX-Retarget"), body)
	}
	status, _, body = response(t, faulty, "GET", "/?from.vin=1HGCM82633A004352", nil)
	if status != 500 || !strings.Contains(body, "<h1>Internal Server Error</h1>") {
		t.Errorf("fault page: %d\n%s", status, body)
	}
	// Fewer than three characters ask the decoder nothing.
	if status, _, body := response(t, faulty, "GET", "/?from.vin=1H", nil); status != 200 || !strings.Contains(body, `name="from.vin" value="1H"`) {
		t.Errorf("two characters on a faulty decoder: %d\n%s", status, body)
	}
}

// TestPlate: the plate takes what is typed, cleaned, and the page draws
// the answer for as much of a VIN as it holds: a card's rows once its
// group's characters are typed, the check digit's verdict by its own
// arithmetic once the VIN is whole, and then the decoder's error texts
// and the VIN it suggests instead.
func TestPlate(t *testing.T) {
	h := New(backend{info: hondaInfo, decode: &hondaDecode}, log.New(io.Discard, "", 0))
	corrected := hondaDecode
	corrected.ErrorCodes, corrected.CorrectedVIN = []int{1}, contract.Null[string]{V: "1HGCM82633A004352", Valid: true}
	suggesting := New(backend{info: hondaInfo, decode: &corrected}, log.New(io.Discard, "", 0))
	suggesting.errorTexts = map[int]string{1: "1 - Check Digit (9th position) does not calculate properly"}
	for _, c := range []struct {
		name       string
		h          *Handler
		query      string
		want, lack []string
	}{
		{"cleaned", h, "from.vin=+1hg-cm8+2633a004352xyz", []string{`name="from.vin" value="1HGCM82633A004352"`, `data-vin="1HGCM82633A004352"`}, nil},
		{"two characters", h, "from.vin=1H",
			[]string{`<span class="cell g-wmi filled"><span class="ch">H</span></span>`, `data-done=""`, "Names turn up once the model is known."},
			[]string{"HONDA", "data-complete"}},
		{"the maker", h, "from.vin=1HG",
			[]string{`data-done="wmi "`, "<title>2003 HONDA Accord · nitrovin</title>", `<div><dt>Make</dt><dd>HONDA</dd></div>`},
			[]string{"<dt>Model</dt>", "<dt>Country</dt>", `<p class="big">`, "data-complete", `id="check-sum"`}},
		{"the plant", h, "from.vin=1HGCM82633A",
			[]string{`data-done="wmi vds check year plant "`, `<div><dt>Model</dt><dd>Accord</dd></div>`, `<p class="big">2003</p>`, `<li class="current" aria-current="true"><span class="mono">3</span> 2003</li>`,
				`<div><dt>Country</dt><dd><span class="flag" aria-hidden="true">🇺🇸</span> UNITED STATES (USA)</dd></div>`},
			[]string{"data-complete", `id="check-sum"`, "Checks out"}},
		{"whole", h, "from.vin=1HGCM82633A004352",
			[]string{"data-complete", `<p class="big">004352</p>`, `<p class="big">Checks out</p>`, `<button type="button" class="how" popovertarget="check-sum">`, `<dialog id="check-sum" popover`, "it checks out."},
			[]string{`class="problems"`, " bad"}},
		{"a wrong check digit", h, "from.vin=1HGCM82603A004352",
			[]string{`<section class="card g-check done bad"`, `<p class="big">Should be 3</p>`, `so it should be <span class="mono">3</span>.`}, []string{"Checks out"}},
		{"a letter no VIN holds", h, "from.vin=1OG", []string{`<span class="cell g-wmi filled bad"><span class="ch">O</span></span>`}, nil},
		{"the decoder's errors", suggesting, "from.vin=1HGCM82603A004352",
			[]string{`<p>1 - Check Digit (9th position) does not calculate properly</p>`, `<a class="fix" href="/?from.vin=1HGCM82633A004352">Use <span class="mono">1HGCM82633A004352</span> instead</a>`}, nil},
		{"none before it is whole", suggesting, "from.vin=1HGCM82603A", nil, []string{`class="problems"`, `class="fix"`}},
	} {
		status, _, body := response(t, c.h, "GET", "/?"+c.query, nil)
		if status != 200 {
			t.Errorf("%s: %d; want 200", c.name, status)
		}
		for _, want := range c.want {
			if !strings.Contains(body, want) {
				t.Errorf("%s: lacks %s\n%s", c.name, want, body)
			}
		}
		for _, lack := range c.lack {
			if strings.Contains(body, lack) {
				t.Errorf("%s: holds %s\n%s", c.name, lack, body)
			}
		}
	}
}

// TestCuratedVINs: every VIN the page offers, as an example and behind
// Surprise me, is a whole VIN the plate keeps as it is, whose check digit
// checks out.
func TestCuratedVINs(t *testing.T) {
	vins := slices.Clone(surpriseVINs)
	for _, e := range exampleVINs {
		vins = append(vins, e.vin)
	}
	for _, vin := range vins {
		if len(vin) != vinLength || cleanVIN(vin) != vin {
			t.Errorf("%s is no whole VIN as the plate takes it", vin)
			continue
		}
		if sum := newCheckSum(vin); sum == nil || !sum.Valid {
			t.Errorf("%s: its check digit does not check out: %+v", vin, sum)
		}
	}
}

// TestBrowserDecode: the browser's page, which has no decoder, decodes a
// VIN once, asking the server's GET /v1/decode/{vin} for every item with
// the decode's query alone (the names of the browser's files stay in the
// browser), lays it out by the elements the server gave it, and
// translates the same decode to every vocabulary loaded on vPIC's spine:
// "Also known as", a stack to a vocabulary, one with a name before one
// without. No route under /ui answers JSON.
func TestBrowserDecode(t *testing.T) {
	const vin = "1D7RV1GT5AS123456"
	srv := New(backend{info: testInfo}, log.New(io.Discard, "", 0))
	if status, _, _ := response(t, srv, "GET", "/ui/decoded?vin="+vin, nil); status != 404 {
		t.Errorf("/ui/decoded on the server: %d; want 404", status)
	}

	s, d := server(t, "classics"), newDisk()
	general := contract.Value("General")
	s.info.Elements = contract.Elements{
		{ElementID: 26, Variable: "Make", Code: "Make", GroupName: general, DataType: "lookup"},
		{ElementID: 28, Variable: "Model", Code: "Model", GroupName: general, DataType: "lookup"},
		{ElementID: 29, Variable: "Model Year", Code: "ModelYear", GroupName: general, DataType: "int"},
		{ElementID: 34, Variable: "Series", Code: "Series", GroupName: general, DataType: "string"},
		{ElementID: 143, Variable: "Error Code", Code: "ErrorCode", DataType: "lookup"},
	}
	s.decodes[vin] = `{"vin":"` + vin + `","wmi":"1D7","check_digit_valid":true,"now":"2026-11-10T00:00:00Z","data_month":"2026_10","decoder_version":"v9","items":[` +
		`{"element_id":26,"value":"DODGE","attribute_id":"476"},{"element_id":28,"value":"Ram","attribute_id":"1938"},{"element_id":29,"value":"2010","attribute_id":"2010"},` +
		`{"element_id":34,"value":"DS (1500)","attribute_id":"DS (1500)"},{"element_id":143,"value":"0","attribute_id":"0"}]}`
	l := open(t, s, d)
	// alpha sorts before yard, and has no name for a Dodge.
	alpha := Picked{FileName: "alpha.jsonl", Data: []byte(`{"format":1,"kind":"vocabulary","name":"alpha","version":"1","data_month":"2026_10","subject":"vehicle","fields":["make","model"]}` + "\n" +
		`{"entry":{"make":"Chevy","model":"Silverado"},"mappings":[{"model":1850}]}` + "\n")}
	if _, err := l.Load(append(pick(t, "yard"), alpha), quiet); err != nil {
		t.Fatal(err)
	}
	s.gets = nil
	h := NewInBrowser(l, BrowserConfig{ErrorTexts: map[int]string{0: "0 - VIN decoded clean"}}, log.New(io.Discard, "", 0))
	status, _, body := response(t, h, "GET", "/?from.vin="+vin, nil)
	for _, want := range []string{
		"<title>2010 DODGE Ram · nitrovin</title>",
		`<div><dt>Make</dt><dd>DODGE</dd></div>`,
		`<div><dt>Trim</dt><dd>DS (1500)</dd></div>`,
		`<input type="checkbox" class="fan visually-hidden" id="fan-yard" aria-label="Show every name in yard">`,
		`<label class="stamp switch" for="fan-yard" title="yard: 3 names">`,
		`<span class="seg g-wmi" title="From the maker, positions 1–3"><span class="w">Dodge</span></span>` + "\n" +
			`<span class="seg g-vds" title="From the vehicle, positions 4–8"><span class="w">Ram 1500</span></span>`,
		`<span class="seg g-vds unsure" title="From the vehicle, positions 4–8. Not checked: element 15"><span class="w">Ram 1500 4x4</span></span>`,
		`<li class="tagplate blank"><span class="stamp">alpha</span><span class="none">No name for this one yet.</span></li>`,
		`&#34;Make&#34;: &#34;DODGE&#34;`,
		`&#34;now&#34;: &#34;2026-11-10T00:00:00Z&#34;`,
		`id="translation-json" data-copy="the translation&#39;s JSON">{` + "\n" + `  &#34;from&#34;: {`,
		`<tr data-where="browser"><th scope="row"><a href="/ui/vocabularies/yard">yard</a></th>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the browser's page lacks %s\n%s", want, body)
		}
	}
	if status != 200 || strings.Contains(body, `class="problems"`) {
		t.Errorf("the browser's page: %d\n%s", status, body)
	}
	if want := []string{"/v1/decode/" + vin + "?full=true"}; !slices.Equal(s.gets, want) {
		t.Errorf("the browser asked the server %q; want %q: the VIN decoded once", s.gets, want)
	}
	if named, blank := strings.Index(body, `id="fan-yard"`), strings.Index(body, `<span class="stamp">alpha</span>`); named > blank {
		t.Errorf("alpha, which has no name, is listed before yard, which has\n%s", body)
	}

	// A decode's options in the page's URL are ignored: the answer is the
	// same, and nothing of them reaches the server.
	s.gets = nil
	status, _, body = response(t, h, "GET", "/?from.vin="+vin+"&model_year=x&now=2020-01-01", nil)
	if status != 200 || !strings.Contains(body, "<title>2010 DODGE Ram · nitrovin</title>") || strings.Contains(body, `class="error"`) ||
		slices.ContainsFunc(s.gets, func(g string) bool { return g != "/v1/decode/"+vin+"?full=true" }) {
		t.Errorf("a decode's options in the browser's page: %d, asked %q\n%s", status, s.gets, body)
	}
}
