package ui

import (
	"io"
	"log"
	"net/http"
	"slices"
	"strconv"
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
		// The header is the brand, Update now and Vocabularies; the foot is
		// Developers and the release; VINs is the plate's, after its mode
		// switch.
		`<a class="brand" href="/">nitrovin</a>` + "\n" + `<button type="button" id="update-now">Update now</button>` + "\n" + `<button type="button" class="open-vocabularies" popovertarget="vocabularies">Vocabularies</button>` + "\n" + `</header>`,
		`<footer>` + "\n" + `<button type="button" class="open-developers" popovertarget="developers">Developers</button>` + "\n" + `<small class="version">`,
		`</nav>` + "\n" + `<button type="button" class="open-vins" popovertarget="vins">VINs</button>` + "\n" + `</div>`,
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
			"/?vin=1HGCM82633A004352&from.model=1861&to=fleet":       "/?from.vin=1HGCM82633A004352",
			"/?vin=": "/",
			"/ui/decode?vin=1HGCM82633A004352&model_year=2003&full=true&now=2026-11-01":      "/?from.vin=1HGCM82633A004352",
			"/ui/decode?vin=1hgcm-82633%20a004352":                                           "/?from.vin=1hgcm-82633+a004352",
			"/ui/decode":                                                                     "/",
			"/ui/translate?from.model=1861&model_year=2003&to=fleet":                         "/",
			"/ui/translate?from.vocabulary=fleet&vin=1HGCM82633A004352":                      "/?from.vin=1HGCM82633A004352",
			"/ui/translate?from.vin=1HGCM82633A004352&to=fleet&to=shelf&tab=spine&full=true": "/?from.vin=1HGCM82633A004352",
			"/ui/translate?from.vin=1HGCM82633A004352&vin=5YJ3E1EA2JF000316&from.make=474":   "/?from.vin=1HGCM82633A004352",
			"/vocab":                                 "/",
			"/vocab?tab=entry&to=fleet":              "/",
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
// and the VIN it suggests instead. A cell the decode names is struck
// as a character no VIN holds is, the field says which positions in
// words, and a finished card with no value says why.
func TestPlate(t *testing.T) {
	h := New(backend{info: hondaInfo, decode: &hondaDecode}, log.New(io.Discard, "", 0))
	corrected := hondaDecode
	corrected.ErrorCodes, corrected.CorrectedVIN = []int{1}, contract.Null[string]{V: "1HGCM82633A004352", Valid: true}
	suggesting := New(backend{info: hondaInfo, decode: &corrected}, log.New(io.Discard, "", 0))
	suggesting.errorTexts = map[int]string{1: "1 - Check Digit (9th position) does not calculate properly"}
	texts := map[int]string{
		1:   "1 - Check Digit (9th position) does not calculate properly",
		8:   "8 - No detailed data available currently",
		11:  "11 - Incorrect Model Year - Position 10 does not match valid model year codes (I, O, Q, U, Z, 0). Decoded data may not be accurate.",
		400: "400 - Invalid Characters Present",
	}
	// A year code no year uses: the decode names position 10, reads no
	// model year, and fails the check digit, whose sum holds.
	yearless := hondaDecode
	yearless.ModelYear = contract.Null[int32]{}
	yearless.ErrorCodes, yearless.CorrectedVIN = []int{1, 11, 400}, contract.Null[string]{V: "1HGCM8261!A004352", Valid: true}
	naming := New(backend{info: hondaInfo, decode: &yearless}, log.New(io.Discard, "", 0))
	naming.errorTexts = texts
	// A problem that names no position, and no model year read.
	undetailed := hondaDecode
	undetailed.ModelYear, undetailed.ErrorCodes = contract.Null[int32]{}, []int{8}
	unnamed := New(backend{info: hondaInfo, decode: &undetailed}, log.New(io.Discard, "", 0))
	unnamed.errorTexts = texts
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
			[]string{`id="problems"`, `class="pstamp"`, " bad", "aria-invalid", "vin-wrong", "data-wrong", `class="why`}},
		{"a wrong check digit", h, "from.vin=1HGCM82603A004352",
			[]string{`<section class="card g-check done bad"`, `<p class="big">Should be 3</p>`, `so it should be <span class="mono">3</span>.`}, []string{"Checks out"}},
		{"a letter no VIN holds", h, "from.vin=1OG", []string{`<span class="cell g-wmi filled bad"><span class="ch">O</span></span>`}, nil},
		// The decoder's errors open in a dialog, each its code, its
		// sentence and the positions it names; with no vehicle's block
		// drawn, a stamp at the plate's foot opens it.
		{"the decoder's errors", suggesting, "from.vin=1HGCM82603A004352",
			[]string{`<li><b class="mono">1</b> <span>Check Digit (9th position) does not calculate properly</span> <span class="where">Position 9</span></li>`, `<a class="fix" href="/?from.vin=1HGCM82633A004352">Use <span class="mono">1HGCM82633A004352</span> instead</a>`,
				`<dialog id="problems" class="pdialog" popover aria-labelledby="problems-h">` + "\n" + `<div class="dialog-head">` + "\n" + `<h2 id="problems-h">VIN problems</h2>`,
				`<p class="pstamp"><button type="button" class="vbad" popovertarget="problems"><svg viewBox="0 0 14 14" aria-hidden="true" focusable="false"><path d="M7 1.5l6 11H1z M7 6v3 M7 10.6v.4"/></svg><b>1 problem</b> <span class="s">1 - Check Digit (9th position) does not calculate properly</span></button></p>`,
				`<span class="cell g-check filled bad"><span class="ch">0</span></span>`, `<p id="vin-wrong" class="visually-hidden" aria-hidden="true">Position 9 is wrong.</p>`}, []string{`class="problems"`}},
		{"a year code no year uses", naming, "from.vin=1HGCM8261UA004352",
			[]string{`<span class="cell g-year filled bad"><span class="ch">U</span></span>`, `data-wrong="10 "`,
				`aria-describedby="vin-wrong answer" aria-invalid="true">`, `<p id="vin-wrong" class="visually-hidden" aria-hidden="true">Position 10 is wrong.</p>`,
				`<p class="why">11 - Incorrect Model Year - Position 10 does not match valid model year codes (I, O, Q, U, Z, 0). Decoded data may not be accurate.</p>`,
				`<dialog id="problems" class="pdialog" popover`, `<p class="big">Checks out</p>`},
			// The check digit's card says its sum holds: the decoder fails
			// it for the year code, and its cell is not struck.
			[]string{`<span class="cell g-check filled bad">`, `<p class="big">2003</p>`}},
		{"a problem that names no position", unnamed, "from.vin=1HGCM82633A004352",
			[]string{`<p class="why none">Not in the decode</p>`, `<li><b class="mono">8</b> <span>No detailed data available currently</span></li>`},
			[]string{" bad", "aria-invalid", "vin-wrong", "data-wrong"}},
		{"none before it is whole", suggesting, "from.vin=1HGCM82603A", nil, []string{`id="problems"`, `class="pstamp"`, `class="fix"`}},
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
	// alpha sorts before fleet, and has no name for a Dodge.
	alpha := Picked{FileName: "alpha.jsonl", Data: []byte(`{"format":1,"kind":"vocabulary","name":"alpha","version":"1","data_month":"2026_10","subject":"vehicle","fields":["make","model"]}` + "\n" +
		`{"entry":{"make":"Chevy","model":"Silverado"},"mappings":[{"model":1850}]}` + "\n")}
	if _, err := l.Load(append(pick(t, "fleet"), alpha), quiet); err != nil {
		t.Fatal(err)
	}
	s.gets = nil
	h := NewInBrowser(l, BrowserConfig{ErrorTexts: map[int]string{0: "0 - VIN decoded clean"}}, log.New(io.Discard, "", 0))
	status, _, body := response(t, h, "GET", "/?from.vin="+vin, nil)
	for _, want := range []string{
		"<title>2010 DODGE Ram · nitrovin</title>",
		`<div><dt>Make</dt><dd>DODGE</dd></div>`,
		`<div><dt>Trim</dt><dd>DS (1500)</dd></div>`,
		`<div class="lrow"><dt>fleet</dt><dd><span class="nw g-wmi lmk">Dodge</span>`,
		`<span class="nw g-vds" title="From the vehicle, positions 4–8">Ram 1500</span>`,
		`<span class="nw g-vds unsure" title="From the vehicle, positions 4–8. Not checked: element 15">Ram 1500 4x4</span>`,
		`<div class="lrow"><dt>alpha</dt><dd><span class="none">No name for this one yet.</span></dd></div>`,
		// A name with something unchecked has the "i" key; a sure one
		// opens its note from the name itself.
		`<button type="button" class="info" popovertarget="aka-3" aria-label="How Dodge Ram 1500 4x4 was reached">i</button>`,
		`&#34;Make&#34;: &#34;DODGE&#34;`,
		`&#34;now&#34;: &#34;2026-11-10T00:00:00Z&#34;`,
		`id="translation-json" data-copy="the translation&#39;s JSON">{` + "\n" + `  &#34;from&#34;: {`,
		`<tr data-where="browser"><th scope="row"><a href="/ui/vocabularies/fleet">fleet</a></th>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the browser's page lacks %s\n%s", want, body)
		}
	}
	if status != 200 || strings.Contains(body, `id="problems"`) {
		t.Errorf("the browser's page: %d\n%s", status, body)
	}
	if want := []string{"/v1/decode/" + vin + "?full=true"}; !slices.Equal(s.gets, want) {
		t.Errorf("the browser asked the server %q; want %q: the VIN decoded once", s.gets, want)
	}
	if named, blank := strings.Index(body, `<dt>fleet</dt>`), strings.Index(body, `<dt>alpha</dt>`); named < 0 || named > blank {
		t.Errorf("alpha, which has no name, is listed before fleet, which has\n%s", body)
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

// TestNameTags: "Also known as" beside its names (#366), on the
// in-browser page over the fixture's fleet, whose month decodes no model
// year: a name that holds for some years is unsure of them, and its years
// say so; sure years do not. Every vocabulary loaded is a tag on the page,
// however many: the row scrolls, and nothing is kept behind a switch. A
// vocabulary with a name comes before those without.
func TestNameTags(t *testing.T) {
	const vin = "1D7RV1GT5AS123456"
	vocabulary := func(name, make, model string) Picked {
		return Picked{FileName: name + ".jsonl", Data: []byte(`{"format":1,"kind":"vocabulary","name":"` + name + `","version":"1","data_month":"2026_10","subject":"vehicle","fields":["make","model"]}` + "\n" +
			`{"entry":{"make":"` + make + `","model":"` + model + `"},"mappings":[{"model":` + map[string]string{"Dodge": "1938", "Chevy": "1850"}[make] + `}]}` + "\n")}
	}
	// page is the VIN's page over the files picked.
	page := func(t *testing.T, picked []Picked) string {
		t.Helper()
		s, d := server(t, "classics"), newDisk()
		general := contract.Value("General")
		s.info.Elements = contract.Elements{
			{ElementID: 26, Variable: "Make", Code: "Make", GroupName: general, DataType: "lookup"},
			{ElementID: 28, Variable: "Model", Code: "Model", GroupName: general, DataType: "lookup"},
			{ElementID: 143, Variable: "Error Code", Code: "ErrorCode", DataType: "lookup"},
		}
		s.decodes[vin] = `{"vin":"` + vin + `","wmi":"1D7","check_digit_valid":true,"now":"2026-11-10T00:00:00Z","data_month":"2026_10","decoder_version":"v9","items":[` +
			`{"element_id":26,"value":"DODGE","attribute_id":"476"},{"element_id":28,"value":"Ram","attribute_id":"1938"},{"element_id":143,"value":"0","attribute_id":"0"}]}`
		l := open(t, s, d)
		if _, err := l.Load(picked, quiet); err != nil {
			t.Fatal(err)
		}
		h := NewInBrowser(l, BrowserConfig{ErrorTexts: map[int]string{0: "0 - VIN decoded clean"}}, log.New(io.Discard, "", 0))
		status, _, body := response(t, h, "GET", "/?from.vin="+vin, nil)
		if status != 200 {
			t.Fatalf("the page: %d\n%s", status, body)
		}
		return body
	}

	// fleet's Ram 1500 holds to 2012, so its years are unsure; its name for
	// any Dodge, and plain's one name, hold for any year.
	t.Run("unsure years", func(t *testing.T) {
		body := page(t, append(pick(t, "fleet"), vocabulary("plain", "Dodge", "Ram")))
		if unsure := `<span class="nw g-year unsure" title="The years this name holds for. Model year not checked">…–2012</span>`; strings.Count(body, unsure) != 2 {
			t.Errorf("want fleet's two names that hold to 2012 unsure of their years, as %s\n%s", unsure, body)
		}
		if sure := `<span class="nw g-year" title="The years this name holds for">any year</span>`; strings.Count(body, sure) != 2 {
			t.Errorf("want fleet's name for any Dodge and plain's name sure of their years, as %s\n%s", sure, body)
		}
		if strings.Contains(body, "Check years") || strings.Contains(body, "more vocabular") {
			t.Errorf("the page links to years, or keeps vocabularies behind a switch\n%s", body)
		}
	})

	for _, extra := range []int{1, 2, 4} {
		t.Run(strconv.Itoa(1+extra)+" vocabularies", func(t *testing.T) {
			picked := pick(t, "fleet")
			for i := range extra {
				picked = append(picked, vocabulary("extra-"+strconv.Itoa(i+1), "Chevy", "Silverado"))
			}
			body := page(t, picked)
			if got := strings.Count(body, `<div class="lrow`); got != 1+extra {
				t.Errorf("%d lines; want %d: every vocabulary is on the page", got, 1+extra)
			}
			if strings.Contains(body, "more vocabular") || strings.Contains(body, `type="checkbox"`) {
				t.Errorf("the names keep vocabularies behind a switch\n%s", body)
			}
			// fleet, which has a name, is before those that have none.
			if fleet, none := strings.Index(body, `<dt>fleet</dt>`), strings.Index(body, `<span class="none">`); fleet < 0 || none < fleet {
				t.Errorf("fleet is not before the vocabularies with no name\n%s", body)
			}
		})
	}
}
