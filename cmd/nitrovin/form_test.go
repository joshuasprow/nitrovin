package main

import (
	"bytes"
	"encoding/json"
	"html"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/joshuasprow/nitrovin-data/pkg/embedded"
	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/nitrovin"
	"github.com/joshuasprow/nitrovin/pkg/ui"
)

// versioned is the embedded decoder reporting another decoder version, as
// a release build would.
type versioned struct {
	loaded
	version string
}

func (v versioned) Info() contract.Info {
	info := v.loaded.Info()
	info.DecoderVersion = v.version
	return info
}

func (v versioned) LoadInfo() contract.Info {
	info := v.loaded.LoadInfo()
	info.DecoderVersion = v.version
	return info
}

func (v versioned) Release() string { return ui.Release(v.LoadInfo()) }

// formServer serves the API on the embedded data, as serve does; a version
// other than "" is the one the decoder reports.
func formServer(t *testing.T, version string) (*httptest.Server, *nitrovin.Decoder) {
	t.Helper()
	d, err := nitrovin.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	texts, err := errorTexts(embedded.Artifact())
	if err != nil {
		t.Fatal(err)
	}
	tr, err := nitrovin.NewTranslator(d)
	if err != nil {
		t.Fatal(err)
	}
	var svc decodeService = loaded{d, tr}
	if version != "" {
		svc = versioned{loaded{d, tr}, version}
	}
	a, err := newAPI(svc, log.New(io.Discard, "", 0), io.Discard, texts, "")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a)
	t.Cleanup(srv.Close)
	return srv, d
}

// get fetches url, the page, and checks what it always carries: its
// headers, no CORS, and no script but the release's htmx, copy.js,
// plate.js and vocab.js, nothing inline.
func get(t *testing.T, url string, status int) string {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	if resp.StatusCode != status {
		t.Fatalf("GET %s: status %d; want %d\n%s", url, resp.StatusCode, status, body)
	}
	pageHeaders(t, resp.Header)
	for _, s := range inlineHandler.FindAllString(body, -1) {
		t.Errorf("inline handler %s", s)
	}
	if strings.Contains(body, "<style") || strings.Contains(body, " style=") {
		t.Errorf("inline style")
	}
	scripts := scriptTag.FindAllString(body, -1)
	if len(scripts) != 4 || slices.ContainsFunc(scripts, func(s string) bool { return !releaseScript.MatchString(s) }) ||
		!strings.Contains(scripts[0], "/htmx.min.js") || !strings.Contains(scripts[1], "/copy.js") || !strings.Contains(scripts[2], "/plate.js") || !strings.Contains(scripts[3], "/vocab.js") {
		t.Errorf("scripts %q; want the release's htmx, copy.js, plate.js and vocab.js alone", scripts)
	}
	return body
}

var (
	inlineHandler = regexp.MustCompile(`(?i)<[^>]*\son[a-z]+\s*=[^>]*>`)
	scriptTag     = regexp.MustCompile(`(?i)<script[^>]*>`)
	releaseScript = regexp.MustCompile(`^<script src="/ui/r/[^"]+/(htmx\.min|copy|plate|vocab)\.js" defer>$`)
)

func pageHeaders(t *testing.T, h http.Header) {
	t.Helper()
	for k, want := range map[string]string{
		"Content-Type":            "text/html; charset=utf-8",
		"Content-Security-Policy": ui.CSP,
		"X-Content-Type-Options":  "nosniff",
		"Vary":                    "HX-Request",
	} {
		if got := h.Get(k); got != want {
			t.Errorf("%s %q; want %q", k, got, want)
		}
	}
	for k := range h {
		if strings.HasPrefix(k, "Access-Control-") {
			t.Errorf("CORS header %s", k)
		}
	}
}

// textOf is the text of the <pre> with id in body, as a browser shows it.
func textOf(t *testing.T, body, id string) string {
	t.Helper()
	m := regexp.MustCompile(`(?s)<pre[^>]* id="` + regexp.QuoteMeta(id) + `"[^>]*>(.*?)</pre>`).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no <pre id=%q>\n%s", id, body)
	}
	return text(m[1])
}

var tag = regexp.MustCompile(`<[^>]*>`)

// text is markup's text: tags dropped, references decoded.
func text(markup string) string { return html.UnescapeString(tag.ReplaceAllString(markup, "")) }

// TestForm drives the page against the embedded data: the empty plate at
// /, a VIN's decode on its cards, with the decode's options, the check
// digit's verdict, the decoder's error texts and the VIN it suggests
// instead, refusals below the plate, the caller's text escaped, the
// headers on every page, and old links to the decode form moved to the
// page. Then a browser runs it (browser_test.go), without JavaScript and
// with it.
func TestForm(t *testing.T) {
	srv, d := formServer(t, "")
	info := d.Info()

	for _, c := range []struct {
		target string
		status int
		// has are substrings of the body; hasNot must be absent.
		has, hasNot []string
	}{
		{"/", 200,
			[]string{
				"data " + info.DataMonth, "decoder " + info.DecoderVersion,
				`href="https://vpic.nhtsa.dot.gov/"`, `href="https://vpic.nhtsa.dot.gov/api/"`,
				// A (devel) build is no release: the releases index, docs on main.
				`href="https://github.com/joshuasprow/nitrovin/releases"`, `/blob/main/docs/parity.md"`,
				// The embedded data's own release, and its month's report.
				`href="https://github.com/joshuasprow/nitrovin-data/releases/tag/v1.` + strings.ReplaceAll(info.DataMonth, "_", "") + ".",
				`href="https://github.com/joshuasprow/nitrovin/pulls?q=is%3Apr&#43;%22Refresh%3A&#43;data&#43;month&#43;` + info.DataMonth + `%22"`,
				"TAG=vX.Y.Z\n",
			},
			[]string{`id="copy-url"`, `class="error"`, "/releases/tag/v0"}},
		{"/?from.vin=1HGCM82633A004352", 200,
			[]string{
				"<title>2003 HONDA Accord · nitrovin</title>", `value="1HGCM82633A004352"`,
				"<dt>Make</dt><dd>HONDA</dd>", "<dt>Model</dt><dd>Accord</dd>", "<dt>Trim</dt><dd>EX-V6</dd>",
				"<dt>Engine</dt><dd>3.0 L · 6 cyl · Gasoline</dd>", `<p class="big">2003</p>`, "<dt>City</dt><dd>MARYSVILLE, OHIO</dd>",
				`<p class="big">Checks out</p>`,
				`href="/v1/decode/1HGCM82633A004352"`,
				`href="https://vpic.nhtsa.dot.gov/decoder/VinDecoder?VIN=1HGCM82633A004352"`,
			},
			[]string{`class="problems"`, `class="error"`}},
		// The page reads the VIN and nothing else: a decode's options in
		// its URL are ignored, as any parameter it does not know is. The
		// decode, its link to /v1 and NHTSA's decoder are the plain ones.
		{"/?from.vin=5YJ3E1EA2JF000316&model_year=2019&shape=nitrovin&full=true&now=2020-01-01&utm=x", 200,
			[]string{
				"<dt>Manufacturer</dt><dd>TESLA, INC.</dd>", "<dt>Model</dt><dd>Model 3</dd>", `<p class="big">2018</p>`,
				`href="/v1/decode/5YJ3E1EA2JF000316"`,
				`href="https://vpic.nhtsa.dot.gov/decoder/VinDecoder?VIN=5YJ3E1EA2JF000316"`,
				`&#34;shape&#34;: &#34;vpic&#34;`,
			},
			[]string{`class="error"`, "model_year=", "ModelYear=", "shape=", "full=", "now=", "utm=", `&#34;items&#34;`, `&#34;now&#34;: &#34;2020-`}},
		// A wrong check digit is error code 1, spelled out: a decode, not a
		// refusal.
		{"/?from.vin=1HGCM82633A004353", 200,
			[]string{
				`<p class="big">Should be 5</p>`, "<dt>Make</dt><dd>HONDA</dd>",
				"<p>1 - Check Digit (9th position) does not calculate properly</p>",
			},
			[]string{`class="error"`, "Checks out"}},
		// A clean decode's code 0 is no problem to list; code 14 is.
		{"/?from.vin=JH2PC40066M200001", 200,
			[]string{
				`<p class="big">Checks out</p>`,
				"<p>14 - Unable to provide information for some of the characters in the VIN, based on the manufacturer submission.</p>",
			},
			[]string{"<p>0 - "}},
		// A suggested VIN is a link to the page with it on the plate.
		{"/?from.vin=1FBNF0AA21A111111", 200,
			[]string{
				"<p>3 - VIN corrected, error in one position (assuming Check Digit is correct)</p>",
				`<a class="fix" href="/?from.vin=1FBNF01A21A111111">Use <span class="mono">1FBNF01A21A111111</span> instead</a>`,
			},
			nil},
		// '!' marks each position the procedure could not correct: no VIN
		// to offer.
		{"/?from.vin=1GTW7NFH5PA077131", 200,
			[]string{"<p>5 - VIN has errors in few positions</p>"},
			[]string{`class="fix"`}},
		// An option the API would refuse is ignored too: never validated.
		{"/?from.vin=1HGCM82633A004352&model_year=2003.5", 200,
			[]string{"<title>2003 HONDA Accord · nitrovin</title>", `value="1HGCM82633A004352"`, "<dt>Make</dt><dd>HONDA</dd>", `id="copy-url"`},
			[]string{`class="error"`, "2003.5"}},
		// The plate keeps the VIN's letters and digits alone, and a
		// parameter the page does not read is not echoed.
		{"/?from.vin=%3Cb%3E%C3%89%22x%3C%2Fb%3E&now=%3Cb%3E", 200,
			[]string{`value="BXB"`},
			[]string{"<b>", "<B>", "&lt;b&gt;", `class="error"`}},
		// The caller's text is escaped where the page echoes it: a
		// malformed query's refusal.
		{"/?from.vin=1HGCM82633A004352&x=%<b>", 400,
			[]string{`<p class="error" role="alert">the query string is malformed: invalid URL escape &#34;%&lt;b&#34;</p>`},
			[]string{"<b>", `id="copy-url"`}},
	} {
		t.Run(c.target, func(t *testing.T) {
			body := get(t, srv.URL+c.target, c.status)
			for _, s := range c.has {
				if !strings.Contains(body, s) {
					t.Errorf("body lacks %s\n%s", s, body)
				}
			}
			for _, s := range c.hasNot {
				if strings.Contains(body, s) {
					t.Errorf("body has %s", s)
				}
			}
		})
	}

	// HEAD is the GET's headers without the body.
	resp, err := client.Head(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	pageHeaders(t, resp.Header)
	if resp.StatusCode != 200 || resp.ContentLength <= 0 {
		t.Errorf("HEAD /: %d, Content-Length %d; want 200 and the GET's length", resp.StatusCode, resp.ContentLength)
	}

	// Every other method is the pages' 405.
	resp, err = client.Post(srv.URL+"/", "application/x-www-form-urlencoded", strings.NewReader("vin=1HGCM82633A004352"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 405 || resp.Header.Get("Allow") != "GET, HEAD" || resp.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("POST /: %d, Allow %q, %s; want 405, GET, HEAD, the error page", resp.StatusCode, resp.Header.Get("Allow"), resp.Header.Get("Content-Type"))
	}

	// An old link to the form, /?vin=… or /ui/decode?vin=…, moves to the
	// page, the VIN the page's from and the decode's options left behind,
	// and lands on the VIN's decode.
	const old = "/?vin=1HGCM82633A004352&shape=nitrovin"
	noFollow := &http.Client{Timeout: client.Timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for _, link := range []string{old, "/ui/decode?vin=1HGCM82633A004352&shape=nitrovin"} {
		resp, err = noFollow.Get(srv.URL + link)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMovedPermanently || resp.Header.Get("Location") != "/?from.vin=1HGCM82633A004352" {
			t.Errorf("GET %s: %d, Location %q; want 301 to /?from.vin=1HGCM82633A004352", link, resp.StatusCode, resp.Header.Get("Location"))
		}
	}
	if body := get(t, srv.URL+old, 200); !strings.Contains(body, "<dt>Make</dt><dd>HONDA</dd>") || !strings.Contains(body, `href="/v1/decode/1HGCM82633A004352"`) {
		t.Errorf("GET %s, redirected: not the decode\n%s", old, body)
	}

	t.Run("browser", func(t *testing.T) { browserForm(t, srv) })
}

// TestFormCopies: the decode's copyable texts are the decode's JSON as
// /v1 answers, indented, the VIN's page's URL, and a curl and a nitrovin
// decode that make the same request: the plain one, whatever else the
// page's URL holds. The JSON says when it was answered, which no later
// request repeats, so /v1 and the command line are asked as of then.
func TestFormCopies(t *testing.T) {
	srv, _ := formServer(t, "")
	const (
		vin    = "1HGCM82633A004352"
		target = "/?from.vin=" + vin + "&model_year=2010&shape=nitrovin&full=true&now=2020-01-01"
		v1     = "/v1/decode/" + vin
		cmd    = "nitrovin decode " + vin
	)
	body := get(t, srv.URL+target, 200)
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(textOf(t, body, "decode-json"))); err != nil {
		t.Fatal(err)
	}
	var shown struct {
		Now string `json:"now"`
	}
	if err := json.Unmarshal(compact.Bytes(), &shown); err != nil || shown.Now == "" || strings.HasPrefix(shown.Now, "2020-") {
		t.Fatalf("the decode's JSON: now %q, %v; want the clock's", shown.Now, err)
	}
	resp, err := client.Get(srv.URL + v1 + "?now=" + url.QueryEscape(shown.Now))
	if err != nil {
		t.Fatal(err)
	}
	api, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	for id, got := range map[string]string{
		"decode-json": compact.String(),
		"copy-url":    textOf(t, body, "copy-url"),
		"copy-curl":   textOf(t, body, "copy-curl"),
		"copy-cli":    textOf(t, body, "copy-cli"),
	} {
		want := map[string]string{
			"decode-json": strings.TrimSuffix(string(api), "\n"),
			"copy-url":    srv.URL + "/?from.vin=" + vin,
			"copy-curl":   "curl " + srv.URL + v1,
			"copy-cli":    cmd,
		}[id]
		if got != want {
			t.Errorf("%s:\n%s\nwant\n%s", id, got, want)
		}
	}
	// The command line gives the same decode.
	if r := cli(t, nil, "", "decode", "--now", shown.Now, vin); r.exit != 0 || r.stdout != string(api) {
		t.Errorf("%s, as of %s: exit %d\n%s%s\nwant\n%s", cmd, shown.Now, r.exit, r.stdout, r.stderr, api)
	}
}

// TestFormREADME: the empty page's install commands, MCP configurations
// and verify commands are the README's, and a release build links to its
// own release.
func TestFormREADME(t *testing.T) {
	b, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	readme := string(b)
	const version = "v9.8.7"
	srv, _ := formServer(t, version)
	body := get(t, srv.URL+"/", 200)
	for _, s := range []string{
		`href="https://github.com/joshuasprow/nitrovin/releases/tag/` + version + `"`,
		`href="https://github.com/joshuasprow/nitrovin/blob/` + version + `/docs/parity.md"`,
	} {
		if !strings.Contains(body, s) {
			t.Errorf("body lacks %s", s)
		}
	}

	// Install: each channel's commands are the README's code spans; the
	// last row links the releases, and has none.
	install := section(t, readme, "## Install")
	page := body[strings.Index(body, `id="install"`):strings.Index(body, `id="mcp"`)]
	var channels []string
	for _, row := range regexp.MustCompile(`(?s)<dt>([^<]+)</dt>\s*<dd>(.*?)</dd>`).FindAllStringSubmatch(page, -1) {
		channel := html.UnescapeString(row[1])
		var got []string
		for _, code := range regexp.MustCompile(`(?s)<pre[^>]*>(.*?)</pre>`).FindAllStringSubmatch(row[2], -1) {
			got = append(got, strings.Split(text(code[1]), "\n")...)
		}
		if got == nil {
			continue
		}
		channels = append(channels, channel)
		var want []string
		for line := range strings.SplitSeq(install, "\n") {
			if cells := strings.Split(line, " | "); len(cells) == 3 && cells[0] == "| "+channel {
				for _, span := range regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(cells[1], -1) {
					want = append(want, span[1])
				}
			}
		}
		if want == nil || !slices.Equal(got, want) {
			t.Errorf("%s: %q; README %q", channel, got, want)
		}
	}
	if want := []string{"Homebrew", "Scoop", "Docker", "Go 1.27+"}; !slices.Equal(channels, want) {
		t.Errorf("channels %q; want %q", channels, want)
	}

	// MCP: the README's snippet, and with Docker its command and args.
	mcp := section(t, readme, "### MCP")
	local := fence(t, mcp, "json")
	if got := textOf(t, body, "mcp-local"); got != local {
		t.Errorf("mcp-local:\n%s\nREADME:\n%s", got, local)
	}
	docker := regexp.MustCompile("With Docker instead, `([^`]+)` and\\s+`([^`]+)`").FindStringSubmatch(mcp)
	if docker == nil {
		t.Fatal("README: no Docker command and args")
	}
	lines := strings.Split(local, "\n")
	for i, line := range lines {
		indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
		switch {
		case strings.HasPrefix(line, indent+`"command": `):
			lines[i] = indent + docker[1] + ","
		case strings.HasPrefix(line, indent+`"args": `):
			lines[i] = indent + docker[2]
		}
	}
	if got, want := textOf(t, body, "mcp-docker"), strings.Join(lines, "\n"); got != want {
		t.Errorf("mcp-docker:\n%s\nwant:\n%s", got, want)
	}

	// Verify: the README's commands with this release's tag.
	verify := fence(t, section(t, readme, "## Verify a release"), "sh")
	first, rest, _ := strings.Cut(verify, "\n")
	if !strings.HasPrefix(first, "TAG=") {
		t.Fatalf("README's verify commands start %q, not TAG=", first)
	}
	if got, want := textOf(t, body, "verify-commands"), "TAG="+version+"\n"+rest; got != want {
		t.Errorf("verify-commands:\n%s\nwant:\n%s", got, want)
	}

	// Neither a pseudo-version nor a modified build is a release.
	for _, v := range []string{"v0.1.5-0.20261109120000-0123456789ab", "v0.1.4+dirty"} {
		srv, _ := formServer(t, v)
		if body := get(t, srv.URL+"/", 200); strings.Contains(body, "/releases/tag/v0") || !strings.Contains(body, "TAG=vX.Y.Z\n") {
			t.Errorf("%s: links a release or tags the verify commands", v)
		}
	}
}

// section is md from heading's line to the next heading of its level or
// higher.
func section(t *testing.T, md, heading string) string {
	t.Helper()
	i := strings.Index(md, "\n"+heading+"\n")
	if i < 0 {
		t.Fatalf("README: no %q", heading)
	}
	s := md[i+1+len(heading):]
	level := strings.Repeat("#", strings.Index(heading, " "))
	for j := 0; j < len(s); {
		k := strings.Index(s[j:], "\n#")
		if k < 0 {
			break
		}
		j += k + 1
		if h := s[j:]; strings.HasPrefix(h, level+" ") || !strings.HasPrefix(h, level) {
			return s[:j]
		}
	}
	return s
}

// fence is the first fenced block of lang in md, without its fences.
func fence(t *testing.T, md, lang string) string {
	t.Helper()
	_, s, ok := strings.Cut(md, "```"+lang+"\n")
	if !ok {
		t.Fatalf("README: no %s block", lang)
	}
	s, _, ok = strings.Cut(s, "\n```")
	if !ok {
		t.Fatalf("README: unclosed %s block", lang)
	}
	return s
}

// costlyInfo is loaded whose Info, which measures model-qualifier drift on
// its first call (ADR 0015), must not be reached.
type costlyInfo struct{ loaded }

func (costlyInfo) Info() contract.Info { panic("Info read: model-qualifier drift measured") }

// TestPagesReadLoadInfo: requests that show no drift (the Service Worker,
// assets, an unknown page) read the info load measured, never Info, so no
// page but the one listing drift builds the model values index.
func TestPagesReadLoadInfo(t *testing.T) {
	d, err := nitrovin.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	tr, err := nitrovin.NewTranslator(d)
	if err != nil {
		t.Fatal(err)
	}
	a, err := newAPI(costlyInfo{loaded{d, tr}}, log.New(io.Discard, "", 0), io.Discard, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a)
	t.Cleanup(srv.Close)
	for _, path := range []string{"/sw.js", "/ui/no-such-page"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusInternalServerError {
			t.Errorf("%s: 500, Info was read", path)
		}
	}
}
