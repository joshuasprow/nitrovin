package ui

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/joshuasprow/nitrovin/pkg/contract"
)

// A VIN's decode on the page (page.go, ADR 0013): what the diagram's
// cards read, the VIN's error codes, and the same request as JSON, curl
// and the command line. The page asks one decode of a VIN, the plain one:
// GET /v1/decode/{vin} with no parameter, the vpic shape as of the clock.
// Its refusals are contract.ParseQuery's. copy.js adds copy buttons to
// the text the page already shows.
//
// The server decodes with its decoder. The browser's Service Worker,
// which draws the page itself when a lookup names a file it holds, has
// none: it asks the server's GET /v1/decode/{vin} for every item, once,
// lays the decode out as the server does (contract.NewDecode, over the
// elements the server gave it) and translates the same decode.

// decodePath was the decode form's page: old links to it move to the
// page (moved).
const decodePath = "/ui/decode"

// mainTarget is the layout's main, which the page's answer fills.
const mainTarget = "#ui-main"

const (
	repoURL     = "https://github.com/joshuasprow/nitrovin"
	dataRepoURL = "https://github.com/joshuasprow/nitrovin-data"
	// dataModule is the module whose version is the embedded data's
	// nitrovin-data release.
	dataModule = "github.com/joshuasprow/nitrovin-data"
)

// pageLinks are the page's links that depend on the running binary.
type pageLinks struct {
	// Tag is the release this build is, "" for any other build.
	Tag string
	// Release is Tag's release page, or the releases index without one.
	Release string
	// DataRelease is the data month's nitrovin-data release, or a search
	// of the releases for the month when the build cannot name it.
	DataRelease string
	// Parity is docs/parity.md at Tag, or on main; ParityReport finds the
	// data month's report, the body of its refresh pull request.
	Parity, ParityReport string
	// APIDocs is the API's documentation (apiDocs), which the Developers
	// dialog links.
	APIDocs string
}

// decodeView is what the page shows of a decode beside its cards.
type decodeView struct {
	// JSON is the same request on GET /v1/decode/{vin}; NHTSA is the VIN
	// on vPIC's own decoder.
	JSON, NHTSA string
	Codes       []errorCodeText
	Copy        decodeCopies
}

// vPIC's element ids the cards read (diagram.go has the rest).
const (
	elementCylinders    = 9
	elementDisplacement = 13
	elementMake         = 26
	elementModel        = 28
	elementPlantCity    = 31
	elementSeries       = 34
	elementTrim         = 38
	elementPlantState   = 77
)

// decodeCopies are the decode's copyable texts, rendered on the server.
type decodeCopies struct {
	// JSON is the decode as the contract layer marshals it; Curl and CLI
	// make the same request on /v1 and the command line.
	JSON, Curl, CLI string
}

// errorCodeText is one error code and its text from the data's error-code
// table, "" when the table lacks it.
type errorCodeText struct {
	Code int
	Text string
}

// attribute is element id's single value in dec, "" when it has none.
func attribute(dec contract.Decode, elements contract.Elements, id int32) string {
	e, ok := elements.Find(id)
	if !ok {
		return ""
	}
	switch attrs := dec.Attributes.(type) {
	case contract.VpicAttributes:
		return attrs[e.Code]
	case contract.NitrovinAttributes:
		return attrs[e.Variable].Value
	}
	return ""
}

// releaseTag matches a release's version, which is its tag; pseudoVersion
// matches the Go pseudo-version of any other commit.
var (
	releaseTag    = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)
	pseudoVersion = regexp.MustCompile(`\d{14}-[0-9a-f]{12}$`)
)

// embeddedDataTag is the nitrovin-data release of the embedded data: the
// version of the data module this build links, "" when it has none.
var embeddedDataTag = sync.OnceValue(func() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, m := range bi.Deps {
		if m.Path == dataModule && m.Replace == nil {
			return m.Version
		}
	}
	return ""
})

// releaseTagOf is version when it is a release's tag, "" for any other
// build: a Go pseudo-version or (devel).
func releaseTagOf(version string) string {
	if releaseTag.MatchString(version) && !pseudoVersion.MatchString(version) {
		return version
	}
	return ""
}

// newFormLinks derives the page's links from the running binary: its
// version, and its data month and source.
func newFormLinks(info contract.Info) pageLinks {
	l := pageLinks{Tag: releaseTagOf(info.DecoderVersion)}
	l.Release, l.Parity = repoURL+"/releases", repoURL+"/blob/main/docs/parity.md"
	if l.Tag != "" {
		l.Release, l.Parity = repoURL+"/releases/tag/"+l.Tag, repoURL+"/blob/"+l.Tag+"/docs/parity.md"
	}
	// A data release is v<format>.<YYYYMM>.<rebuild>.
	month := strings.ReplaceAll(info.DataMonth, "_", "")
	l.DataRelease = dataRepoURL + "/releases?q=" + url.QueryEscape(month)
	if tag := embeddedDataTag(); info.DataSource == contract.DataSourceEmbedded && strings.Contains(tag, "."+month+".") {
		l.DataRelease = dataRepoURL + "/releases/tag/" + tag
	}
	l.ParityReport = repoURL + "/pulls?q=" + url.QueryEscape(`is:pr "Refresh: data month `+info.DataMonth+`"`)
	return l
}

// links are the page's links: the running binary's, and in the browser
// the data release the server's build named (WithDataRelease).
func (h *Handler) links(info contract.Info) pageLinks {
	l := newFormLinks(info)
	l.APIDocs = apiDocs(&info)
	if h.dataRelease != "" {
		l.DataRelease = h.dataRelease
	}
	return l
}

// decoded is the plain decode of vin (GET /v1/decode/{vin}'s, with no
// parameter), as the page shows it, and in the browser the same decode as
// a translation reads it (nil on the server, whose translator decodes
// itself). The error is a refusal (a *contract.Problem, or in the browser
// a *pageError when the server cannot be asked) or a fault.
func (h *Handler) decoded(r *http.Request, vin string) (*decodeView, contract.Decode, *contract.VINDecode, error) {
	req, err := contract.ParseQuery(vin, nil)
	if err != nil {
		return nil, contract.Decode{}, nil, err
	}
	var dec contract.Decode
	var vd *contract.VINDecode
	if h.b != nil {
		dec, err = h.b.Decode(req)
	} else {
		dec, vd, err = h.lib.decodeVIN(vin, req)
	}
	if err != nil {
		return nil, contract.Decode{}, nil, err
	}
	body, err := json.Marshal(dec) // as GET /v1/decode/{vin} writes it
	if err != nil {
		return nil, contract.Decode{}, nil, err
	}
	jsonURL := "/v1/decode/" + url.PathEscape(vin)
	d := &decodeView{
		JSON: jsonURL, NHTSA: "https://vpic.nhtsa.dot.gov/decoder/VinDecoder?VIN=" + url.QueryEscape(dec.VIN),
		Copy: decodeCopies{
			JSON: string(body),
			Curl: "curl " + shellQuote(requestOrigin(r)+jsonURL),
			CLI:  cliCommand(vin),
		},
	}
	for _, c := range dec.ErrorCodes {
		d.Codes = append(d.Codes, errorCodeText{c, h.errorTexts[c]})
	}
	return d, dec, vd, nil
}

// requestOrigin is the scheme and host r was sent to, as the client named
// them: https behind a proxy that says so, and in the browser's worker,
// whose requests carry their whole URL, that URL's.
func requestOrigin(r *http.Request) string {
	if r.Host == "" {
		return ""
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" || r.URL.Scheme == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// cliCommand is `nitrovin decode` of vin as sent, with no flag: the plain
// request.
func cliCommand(vin string) string {
	cmd := "nitrovin decode"
	switch {
	case vin == "-": // an argument "-" reads stdin
		return "echo - | " + cmd + " -"
	case strings.HasPrefix(vin, "-"):
		return cmd + " -- " + shellQuote(vin)
	}
	return cmd + " " + shellQuote(vin)
}

// shellQuote is s as one word of a POSIX shell (zsh included): itself when
// it holds only characters no shell treats specially, else single-quoted.
func shellQuote(s string) string {
	plain := s != ""
	for _, c := range s {
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || strings.ContainsRune("-_./:,+@%", c)) {
			plain = false
			break
		}
	}
	if plain {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
