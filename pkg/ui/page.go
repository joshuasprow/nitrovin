package ui

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"html/template"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/joshuasprow/nitrovin/pkg/contract"
)

// The page (ADR 0013). GET / is nitrovin's one page for browsers: the VIN
// decoder, drawn as the VIN's diagram (diagram.go). Its form, the plate,
// asks the page itself with a VIN's from (from.vin), so every answer is a
// shareable URL. The page reads no other parameter of a decode: what it
// asks of a VIN is always the same, the plain decode. It reads a few more,
// the part type pressed among the parts that fit (part.id, fit.go), that
// one's page of candidates (after), the candidate whose stock is open
// (fitment.number, fitment.side and fitment.type) and which stock panel
// shows every row (rows, fitstock.go). Below the plate is the answer for
// whatever is typed so far, a partial VIN included: a card for each of the
// VIN's groups, the vehicle's names in every vocabulary loaded, its
// translation on vPIC's spine, and the parts that fit it in the catalog
// loaded. A VIN the decoder refuses is the page
// still, with what is wrong said below the plate. A switch on the plate
// turns it to a part type and a number, part-number mode (part.go),
// whose answer is what a part fits and its stock. The vocabularies loaded
// are behind the header's Vocabularies button, in a dialog of their own
// (vocab.go), which the layout holds beside the page's main, so no answer
// replaces it; what developers need, the answer's JSON included, is
// behind the Developers one, in the page's fragment (decode.html).
//
// The page answers the same on the server and in the browser's Service
// Worker, which answers the page whenever the browser holds files, over
// the server's files and the browser's; a VIN's decode is the server's
// there too, asked for once (decoded). Without JavaScript every step is a
// page load, and the dialogs still open: they are popovers. Old links, to
// /ui/decode, /ui/translate, /vocab and the form at /?vin=…, move to the
// page, with the VIN they named and nothing else.

const (
	homePath = "/"
	// homeTitle is the page's title before it has an answer.
	homeTitle = "VIN decoder"
)

// homePage is what the page's template renders: the plate and everything
// hanging off it (diagram.go), or, in part-number mode, the plate with a
// part type and a number and the part below it (part.go); the chips to
// try, and what the Developers dialog shows.
type homePage struct {
	Diagram *diagram
	// Part is the page in part-number mode, nil in VIN mode, which then
	// draws no Diagram.
	Part     *partPage
	Examples []example
	// Result is the decode's own request and JSON, or the part's, nil
	// before either.
	Result                    *result
	DataMonth, DecoderVersion string
	Links                     pageLinks
	// PartMode is the switch's link to part-number mode.
	PartMode string
}

// example is a VIN to try: a link to the page with it on the plate.
type example struct {
	Label, Hint, Href string
}

// The examples a visitor with no recent VINs is offered: each has a name
// in the demo set's vehicle vocabulary and in EPA's, so "Also known as"
// has something to show once either is loaded.
var exampleVINs = []struct{ vin, hint string }{
	{"1FTEW1EG7FFA12345", "a 2015 Ford F-150"},
	{"2HKRW2H53KH612345", "a 2019 Honda CR-V"},
	{"4T1B11HK7JU512345", "a 2018 Toyota Camry"},
	{"1GCVKREC6EZ123456", "a 2014 Chevrolet Silverado"},
	{"1C6RR7LT9FS123456", "a 2015 Ram 1500"}, // check digit valid; the README's ...5FS... is not
}

// vinHref is the page with vin on the plate.
func vinHref(vin string) string {
	return homePath + "?" + url.Values{"from.vin": {vin}}.Encode()
}

// result is what the Developers dialog shows of the answer: the decode's
// own request, and the JSON of it, of its translation and of its fit.
type result struct {
	Decode *decodeView
	// JSON are the answer's objects, each in the dialog: the decode, the
	// translation, the part types and the candidates.
	JSON []jsonDoc
	// URL is the page's: the VIN, and the part type pressed with its page
	// of candidates.
	URL string
}

// jsonDoc is one JSON object the dialog shows: its element's id, what it
// is, in a heading and for its copy button, the request on /v1 that
// answers it ("" for a translation, which only this browser may be able
// to answer), and its text, indented and escaped for the page (jsonText).
type jsonDoc struct {
	ID, Title, What, Href string
	Text                  template.HTML
}

// jsonText is v as json.MarshalIndent writes it, indented by two spaces,
// escaped as html/template escapes an element's text, in one pass: a
// related page's answer is hundreds of KB, and marshalling it, formatting
// it again and escaping a copy of it cost a browser's module MBs a request
// (ADR 0023).
func jsonText(v any) (template.HTML, error) {
	var b strings.Builder
	if err := jsonv2.MarshalWrite(htmlText{&b}, v, json.DefaultOptionsV1(), jsontext.WithIndent("  ")); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

// indentText is raw, JSON, indented and escaped as jsonText's.
func indentText(raw []byte) (template.HTML, error) {
	var indented bytes.Buffer
	if err := json.Indent(&indented, raw, "", "  "); err != nil {
		return "", err
	}
	var b strings.Builder
	_, _ = htmlText{&b}.Write(indented.Bytes())
	return template.HTML(b.String()), nil
}

// htmlText writes text to b escaped as html/template escapes an element's
// text: the same bytes as {{.}} of a string.
type htmlText struct{ b *strings.Builder }

func (w htmlText) Write(p []byte) (int, error) {
	last := 0
	for i, c := range p {
		var esc string
		switch c {
		case 0:
			esc = "\uFFFD"
		case '"':
			esc = "&#34;"
		case '&':
			esc = "&amp;"
		case '\'':
			esc = "&#39;"
		case '+':
			esc = "&#43;"
		case '<':
			esc = "&lt;"
		case '>':
			esc = "&gt;"
		default:
			continue
		}
		w.b.Write(p[last:i])
		w.b.WriteString(esc)
		last = i + 1
	}
	w.b.Write(p[last:])
	return len(p), nil
}

// oldVIN is the VIN an old link named, as the decode form's vin or a
// translation's from.vin, the from first; "" when it named none.
func oldVIN(q url.Values) string {
	if vin := q.Get("from.vin"); vin != "" {
		return vin
	}
	return q.Get("vin")
}

// moved answers a page that the one page holds now by sending the
// visitor there with the VIN the link named, if it named one, and
// nothing else of its query: the page reads nothing else. worker.js
// moves /ui/translate by the same rule.
func (h *Handler) moved(r *http.Request) (view, error) {
	loc := homePath
	if vin := oldVIN(r.URL.Query()); vin != "" {
		loc = vinHref(vin)
	}
	return view{location: loc, status: http.StatusMovedPermanently}, nil
}

// home is GET /: the plate, empty or with what the query's VIN holds so
// far, cleaned, and below it the cards that VIN fills. A partial VIN is
// decoded as far as it goes (three characters name the maker), so the page
// answers every keystroke plate.js sends as it answers the form. A refusal
// is the page still, with its status. ?surprise moves to one of the
// curated VINs. A fitment key in the query is part-number mode instead
// (partHome).
func (h *Handler) home(r *http.Request) (view, error) {
	info := h.info()
	raw, malformed := url.ParseQuery(r.URL.RawQuery)
	if malformed != nil {
		raw = url.Values{}
	}
	// A vin is the old decode form's: no page asks with one now.
	if raw.Has("vin") {
		return h.moved(r)
	}
	if raw.Has("surprise") {
		return view{location: vinHref(surpriseVINs[rand.IntN(len(surpriseVINs))]), status: http.StatusSeeOther}, nil
	}
	page := &homePage{DataMonth: info.DataMonth, DecoderVersion: info.DecoderVersion, Links: h.links(info), PartMode: partModeHref}
	for _, e := range exampleVINs {
		page.Examples = append(page.Examples, example{Label: e.vin, Hint: e.hint, Href: vinHref(e.vin)})
	}
	if partMode(raw) {
		// A URL is one mode: one naming both offers each.
		if raw.Has("from.vin") {
			return h.mixedHome(raw, info, page)
		}
		return h.partHome(r, raw, info, page)
	}
	d := newDiagram(cleanVIN(raw.Get("from.vin")))
	d.Fit = h.idleFit(info)
	ask := fitAsk{Part: raw.Get(partParam), After: raw.Get(afterParam), Type: raw.Get(typeParam), Number: raw.Get(numberParam), Side: raw.Get(sideParam), Rows: raw.Get(rowsParam)}
	d.Fit.Part, d.Fit.After, d.Fit.Type, d.Fit.Number, d.Fit.Side, d.Fit.Rows = ask.Part, ask.After, ask.Type, ask.Number, ask.Side, ask.Rows
	page.Diagram = d
	v := view{title: homeTitle, template: "home", data: page, target: mainTarget, vocabDialog: true}
	refuse := func(err error) (view, error) {
		status, words, fault := refusal(err)
		if fault != nil {
			return view{}, fault
		}
		v.status, d.Error = status, words
		return v, nil
	}
	if malformed != nil {
		return refuse(contract.NewProblem(contract.InvalidRequest, "", "the query string is malformed: "+malformed.Error()))
	}
	// Three characters are the least the decoder reads anything from.
	if len(d.VIN) < 3 {
		return v, nil
	}
	cat, err := h.catalog()
	if err != nil {
		return view{}, err
	}
	res := &result{URL: requestOrigin(r) + vinHref(d.VIN)}
	dv, dec, vd, err := h.decoded(r, d.VIN)
	if err != nil {
		return refuse(err)
	}
	res.Decode = dv
	d.fill(dec, info.Elements, dv)
	if d.FixV != "" {
		d.Fix = vinHref(d.FixV)
	}
	text, err := indentText([]byte(dv.Copy.JSON))
	if err != nil {
		return view{}, err
	}
	res.JSON = []jsonDoc{{ID: "decode-json", Title: "The decode", What: "the decode's JSON", Href: dv.JSON, Text: text}}
	page.Result = res
	if d.Label != "" {
		v.title = d.Label
	}

	// Also known as: the VIN in every vocabulary loaded on vPIC's spine,
	// once its model is known.
	to := vehicleVocabularies(cat)
	d.Names.None = len(to) == 0
	if !d.Names.Waiting && !d.Names.None {
		req, err := contract.ParseTranslateQuery(url.Values{"from.vin": {d.VIN}, "to": to})
		if err != nil {
			return refuse(err)
		}
		tr, err := h.translate(req, vd)
		if err != nil {
			return refuse(err)
		}
		d.names(tr, cat, info.Elements)
		tr.Vocabularies = slices.Clone(tr.Vocabularies)
		slices.SortFunc(tr.Vocabularies, func(a, b contract.FileVersion) int { return strings.Compare(a.Name, b.Name) })
		body, err := jsonText(tr)
		if err != nil {
			return view{}, err
		}
		res.JSON = append(res.JSON, jsonDoc{ID: "translation-json", Title: "The translation", What: "the translation's JSON", Text: body})
	}

	// The parts that fit, once the model is known.
	if d.Names.Waiting {
		return v, nil
	}
	ask.Stock = h.stockKinds(info)
	if err := h.fitVIN(r.Context(), d, vd, cat, info.Elements, ask, res); err != nil {
		return refuse(err)
	}
	if d.Fit.Page != "" {
		res.URL = requestOrigin(r) + d.Fit.Page
	}
	return v, nil
}

// refusal is err as the page says it below the plate: its status and its
// words; or, when err is a fault in nitrovin, fault, which is the error
// page, logged.
func refusal(err error) (status int, words string, fault error) {
	if e, ok := errors.AsType[*pageError](err); ok {
		return e.Status, e.Message, nil
	}
	p := contract.AsProblem(err)
	if p.Type == contract.Internal {
		return 0, "", err
	}
	return p.Status, p.Detail, nil
}

// translate answers req, in the browser over dec, the VIN's decode the
// page has already, when it is not nil.
func (h *Handler) translate(req contract.TranslateRequest, dec *contract.VINDecode) (contract.Translation, error) {
	if h.lib != nil && dec != nil {
		return h.lib.translate(req, dec)
	}
	return h.p.Translate(req)
}

// vocabDialog is the Vocabularies dialog, behind the header's button: the
// catalog, and whether the visitor's own files can be loaded, which needs
// the worker's module on the server. Release is the release that drew
// it, which vocab.js holds the worker in control to.
type vocabDialog struct {
	Catalog catalogTable
	Worker  bool
	// Demo says the demo set can load here (demoLoadable), which needs
	// JavaScript too.
	Demo    bool
	Release string
}

// newVocabDialog is the Vocabularies dialog of a page that shows it,
// drawn beside its main when the page is whole, by info's release.
func (h *Handler) newVocabDialog(info contract.Info) (vocabDialog, error) {
	cat, err := h.catalog()
	if err != nil {
		return vocabDialog{}, err
	}
	return vocabDialog{Catalog: catalogTable{Rows: h.rows(cat)}, Worker: h.worker != nil || h.lib != nil, Demo: h.demoLoadable(), Release: Release(info)}, nil
}
