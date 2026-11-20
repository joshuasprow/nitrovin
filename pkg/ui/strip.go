package ui

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The strip of part types (#324) is a grid: 5 columns from 64rem, 4 above
// 46rem, 3 above 34rem and 2 on a phone (ui.css). Two rows of it hold the
// part types in strip order, then "N more", a key to the part types
// dialog. A tape takes one cell, or two where its name does not fit one,
// and a two-cell tape that does not fit at a row's end starts the next
// row, as the grid places it. The server packs the rows at each width
// from stripFit and writes, by class, which tapes take two cells and
// which are past the rows, and N; ui.css shows each width its own. With a
// script, plate.js packs them again from the same numbers (page.html's
// data-fit) beside its filter and Recent stamps, so neither reads a
// position (ADR 0027).

// stripMeasure is how wide a tape is, in CSS px, a little wider than any
// browser draws it, so a tape the server gives one cell never cuts its
// name: Tape is a tape's padding, edges and gap, and its count's pill's
// padding and edges; Digit a digit of the count; Stamp the Recent stamp
// a script puts at a tape's leading edge; Slack what is left for
// rounding. The tape's face is monospace, so its name is Char a
// character at each grid.
type stripMeasure struct {
	Tape  float64     `json:"tape"`
	Digit float64     `json:"digit"`
	Stamp float64     `json:"stamp"`
	Slack float64     `json:"slack"`
	Grids []stripGrid `json:"grids"`
}

// stripGrid is the strip at one width: its columns, its narrowest cell
// there, a character of a name there, and the cells the script's filter
// takes in the first row (none on a phone, where it is a row of its own).
type stripGrid struct {
	Cols int     `json:"cols"`
	Cell float64 `json:"cell"`
	Char float64 `json:"char"`
	Lead int     `json:"lead"`
}

// stripFit is the strip's measure, from ui.css: a tape's 10px padding, 1px
// edges and 8px gap, its count's 7px padding and 1px edges; a character
// 0.62em of the 900 monospace face and 0.06em of tracking (14px, 13px on
// a phone), a count's digit of 11px; the Recent stamp's six characters of
// 10px, tracked .08em, its padding and edges. A cell is the narrowest the
// width has: from a 1024px window with a 17px scroll bar, the main
// column's 62rem less the strip's 16px margins (898px); from a 737px one,
// 688px; from a 545px one, 512px; and on a phone from a 360px one, 344px;
// each less its gaps, and shared among its columns.
var stripFit = stripMeasure{
	Tape: 46, Digit: 7.7, Stamp: 56, Slack: 2,
	Grids: []stripGrid{
		{Cols: 5, Cell: 171, Char: 9.52, Lead: 1},
		{Cols: 4, Cell: 164, Char: 9.52, Lead: 1},
		{Cols: 3, Cell: 164, Char: 9.52, Lead: 1},
		{Cols: 2, Cell: 167, Char: 8.84, Lead: 0},
	},
}

// stripFitJSON is stripFit as page.html gives it to plate.js.
var stripFitJSON = func() string {
	b, err := json.Marshal(stripFit)
	if err != nil { // a struct of numbers: unreachable
		panic(err)
	}
	return string(b)
}()

// stripLaid is the strip as two rows hold it at each width: Fit the
// measure the script packs by; Keys "N more" at each width; KeyClass the
// key's classes, k<cols> at each width it is drawn.
type stripLaid struct {
	Fit      string
	Keys     []stripKey
	KeyClass string
}

// stripKey is "N more" at one width, of Cols columns.
type stripKey struct {
	Cols, N int
}

// tapeSpan is the cells a tape takes at g: one when its name, its count
// and the Recent stamp when stamp fit one cell, else two. A name too long
// for two still takes two, and ends in "…".
func tapeSpan(name string, count int, stamp bool, g stripGrid) int {
	w := stripFit.Tape + stripFit.Slack + g.Char*float64(utf8.RuneCountInString(name)) + stripFit.Digit*float64(len(strconv.Itoa(count)))
	if stamp {
		w += stripFit.Stamp
	}
	if w <= g.Cell {
		return 1
	}
	return 2
}

// packRows is how many tapes of spans, in order, two rows of cols cells
// hold after lead cells: all of them, or as many as hold with the key's
// cell after them. A tape that does not fit at a row's end starts the
// next row.
func packRows(spans []int, cols, lead int) int {
	fits := func(k int, key bool) bool {
		row, col := 0, lead
		place := func(s int) bool {
			if col+s > cols {
				row, col = row+1, 0
			}
			col += s
			return row < 2
		}
		for _, s := range spans[:k] {
			if !place(s) {
				return false
			}
		}
		return !key || place(1)
	}
	if fits(len(spans), false) {
		return len(spans)
	}
	k := len(spans) - 1
	for k > 0 && !fits(k, true) {
		k--
	}
	return k
}

// layStrip packs types, in strip order, into two rows at each width, as
// the page draws them without a script: it sets each type's Class, s<cols>
// where its tape takes two cells and x<cols> where it is past the rows,
// and is the key at each width.
func layStrip(types []partTypeLink) stripLaid {
	laid := stripLaid{Fit: stripFitJSON}
	classes := make([][]string, len(types))
	keyClass := []string{"ptkey"}
	for _, g := range stripFit.Grids {
		spans := make([]int, len(types))
		for i, t := range types {
			spans[i] = tapeSpan(t.Name, t.Count, false, g)
			if spans[i] == 2 {
				classes[i] = append(classes[i], "s"+strconv.Itoa(g.Cols))
			}
		}
		shown := packRows(spans, g.Cols, 0)
		for i := shown; i < len(types); i++ {
			classes[i] = append(classes[i], "x"+strconv.Itoa(g.Cols))
		}
		if shown < len(types) {
			keyClass = append(keyClass, "k"+strconv.Itoa(g.Cols))
		}
		laid.Keys = append(laid.Keys, stripKey{Cols: g.Cols, N: len(types) - shown})
	}
	for i := range types {
		types[i].Class = strings.Join(classes[i], " ")
	}
	laid.KeyClass = strings.Join(keyClass, " ")
	return laid
}
