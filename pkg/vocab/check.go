package vocab

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
)

// Check checks the files at paths, as Open names them, against data month
// m, as `nitrovin vocab check` does (ADR 0011, ADR 0025): every fault of
// every file, and the drift (Drift, and ModelDrift when m is a ModelMonth)
// of every file that reads and links cleanly, even when another has
// faults. A file is unchecked when it has faults, or links, directly or
// through other files, to one that does: a catalog to its spine, vehicle
// vocabulary and filed-under vocabulary (a vpic catalog to every extension
// set); a part vocabulary to its spine; a vocabulary or extension set to
// the extension sets it references; a stock file to its catalog and the
// vocabularies and spines its rows name. Each unchecked file gets one
// SeverityUnchecked finding, at line 0, after its faults, saying which
// file has the faults.
//
// A JSON Lines catalog is checked by Compile's streaming pass, its
// compiled bytes written nowhere, sorting in scratch: its memory does not
// grow with its lines (ADR 0025). When a stock file to be checked links to
// it, it is compiled into scratch instead and read there, as a loaded
// catalog. A compiled catalog is read in full (Expand) and checked again.
// The findings come file by file, in the order of paths; the error is
// one that is not about the files, such as scratch storage failing.
func Check(m Month, scratch Scratch, paths ...string) ([]Finding, error) {
	c := &checker{m: m, scratch: scratch, byName: map[string]int{}, byPath: map[string]int{}}
	defer c.close()
	eachFile(paths, c.read, func(g Finding) { c.add(&checked{path: g.File, faults: Faults{g}}) })
	c.oneCatalog()
	for i, n := range c.files {
		if n.header != nil {
			if _, dup := c.byName[n.header.Name]; !dup {
				c.byName[n.header.Name] = i
			}
		}
	}
	for _, n := range c.files {
		n.links = c.linksOf(n)
	}
	c.mark()
	c.link(func(f *File) bool { return f.Kind != KindStock })
	c.mark()
	if err := c.stream(); err != nil {
		return nil, err
	}
	c.mark()
	c.link(func(*File) bool { return true })
	c.mark()
	c.drift()
	var out []Finding
	for _, n := range c.files {
		if n.bad == nil {
			out = append(out, n.drift...)
			continue
		}
		slices.SortStableFunc(n.faults, func(a, b Finding) int { return cmp.Compare(a.Line, b.Line) })
		out = append(out, n.faults...)
		out = append(out, Finding{File: n.path, Severity: SeverityUnchecked, Detail: c.why(n)})
	}
	return out, nil
}

// checker is one Check's state.
type checker struct {
	m       Month
	scratch Scratch
	files   []*checked
	byName  map[string]int // a header's name: the first file with it
	byPath  map[string]int
	catalog *checked // the catalog, the first file whose header says it is one
	opened  []io.Closer
}

// checked is one file Check reads.
type checked struct {
	path string
	// header is its header, when it reads: all a JSON Lines catalog's
	// (source) is read of it before Compile streams it. f is the file,
	// read and, a compiled catalog, expanded; nil when it has faults, and
	// for a source, unless it is compiled into scratch.
	header *File
	f      *File
	source bool
	faults Faults
	drift  []Finding
	links  []string // the names of the files it links to
	// bad is why it is unchecked, nil while it is not: the file with the
	// faults, by index, and the file it links to that one through, -1
	// when it links to it directly or is it.
	bad *struct{ faulty, via int }
}

func (c *checker) add(n *checked) {
	if _, dup := c.byPath[n.path]; !dup {
		c.byPath[n.path] = len(c.files)
	}
	c.files = append(c.files, n)
}

func (c *checker) close() {
	for _, o := range c.opened {
		o.Close()
	}
}

// read reads the file at path: a JSON Lines catalog's header alone, any
// other file whole.
func (c *checker) read(path string) {
	n := &checked{path: path}
	c.add(n)
	h, err := readHeader(path)
	if err != nil {
		n.faults = Faults{fault(path, 0, "%v", err)}
		return
	}
	n.header = h
	if h != nil && h.Kind == KindCatalog {
		n.source = true
		return
	}
	f, err := openFile(path, nil)
	if err == nil {
		f, err = f.Expand()
	}
	if err != nil {
		n.faults = err.(Faults)
		return
	}
	n.f, n.header = f, f
}

// oneCatalog refuses every catalog after the first, as Link does: a
// deployment loads one catalog.
func (c *checker) oneCatalog() {
	for _, n := range c.files {
		switch {
		case n.header == nil || n.header.Kind != KindCatalog:
		case c.catalog == nil:
			c.catalog = n
		default:
			n.faults = append(n.faults, fault(n.path, 1, "/kind: %s is a catalog too: a deployment loads one catalog", c.catalog.path))
		}
	}
}

// linksOf are the names of the files n links to, by what it has read.
func (c *checker) linksOf(n *checked) []string {
	h := n.header
	if h == nil || len(n.faults) > 0 {
		return nil
	}
	var links []string
	extSet := func(ref string) {
		set, _ := splitExt(ref)
		links = append(links, set)
	}
	switch h.Kind {
	case KindVocabulary:
		if h.Subject == SubjectPart {
			links = append(links, h.Spine)
		}
		for i := range h.Entries {
			for _, mp := range h.Entries[i].Mappings {
				if mp.Ext != "" {
					extSet(mp.Ext)
				}
			}
		}
	case KindExtensionSet:
		for _, x := range h.Extensions {
			if x.MakeExt != "" {
				extSet(x.MakeExt)
			}
		}
	case KindCatalog:
		links = append(links, h.Spine)
		switch {
		case h.Vehicles != Reserved:
			links = append(links, h.Vehicles)
		default: // its vehicles name extensions: any set may define one
			for _, g := range c.files {
				if g.header != nil && g.header.Kind == KindExtensionSet {
					links = append(links, g.header.Name)
				}
			}
		}
		if h.FiledUnder != "" {
			links = append(links, h.FiledUnder)
		}
	case KindStock:
		if h.Holds == StockShelf {
			links = append(links, h.StockCatalog)
		}
		for _, r := range h.Stock {
			if (r.Part != nil || r.vehicle != nil) && c.catalog != nil {
				links = append(links, c.catalog.header.Name) // a yard's rows name the catalog's
			}
			if r.Part != nil {
				links = append(links, r.Part.Spine, r.Part.Vocabulary)
			}
		}
	}
	return links
}

// mark marks unchecked every file that has faults or links to an
// unchecked file, until none is left to mark.
func (c *checker) mark() {
	for changed := true; changed; {
		changed = false
		for i, n := range c.files {
			if n.bad != nil {
				continue
			}
			if len(n.faults) > 0 {
				n.bad, changed = &struct{ faulty, via int }{i, -1}, true
				continue
			}
			for _, name := range n.links {
				j, ok := c.byName[name]
				if !ok || c.files[j].bad == nil {
					continue
				}
				if w := c.files[j].bad; w.faulty == j {
					n.bad = &struct{ faulty, via int }{j, -1}
				} else {
					n.bad = &struct{ faulty, via int }{w.faulty, j}
				}
				changed = true
				break
			}
		}
	}
}

// why is the detail of unchecked file n's finding.
func (c *checker) why(n *checked) string {
	switch faulty := c.files[n.bad.faulty]; {
	case faulty == n:
		return "its drift was not measured: it has faults"
	case n.bad.via < 0:
		return fmt.Sprintf("its drift was not measured: it links to %s, which has faults", faulty.path)
	default:
		return fmt.Sprintf("its drift was not measured: it links through %s to %s, which has faults", c.files[n.bad.via].path, faulty.path)
	}
}

// clean are the files read and not unchecked that keep accepts.
func (c *checker) clean(keep func(*File) bool) []*File {
	var out []*File
	for _, n := range c.files {
		if n.bad == nil && n.f != nil && keep(n.f) {
			out = append(out, n.f)
		}
	}
	return out
}

// link links the clean files keep accepts (Link), each fault to its file.
func (c *checker) link(keep func(*File) bool) {
	files := c.clean(keep)
	if len(files) == 0 {
		return
	}
	faults, _ := errors.AsType[Faults](Link(files))
	for _, g := range faults {
		n := c.files[c.byPath[g.File]]
		n.faults = append(n.faults, g)
	}
}

// stream checks the JSON Lines catalog, when there is one to check, with
// Compile: written nowhere, or into scratch, and read there, when a stock
// file to be checked links to it.
func (c *checker) stream() error {
	n := c.catalog
	if n == nil || !n.source || n.bad != nil {
		return nil
	}
	read := false
	for _, g := range c.files {
		if g.bad == nil && g.header != nil && g.header.Kind == KindStock && slices.Contains(g.links, n.header.Name) {
			read = true
		}
	}
	src, err := os.Open(n.path)
	if err != nil {
		n.faults = Faults{fault(n.path, 0, "%v", errorText(err))}
		return nil
	}
	defer src.Close()
	var w io.Writer = io.Discard
	var out *spill
	if read {
		if out, err = newSpill(c.scratch); err != nil {
			return err
		}
		c.opened = append(c.opened, out.f)
		w = out
	}
	beside := c.clean(func(f *File) bool { return f.Kind != KindStock && f.Kind != KindCatalog })
	drift, err := Compile(w, c.m, beside, src, n.path, c.scratch, CompileOptions{})
	if faults, ok := errors.AsType[Faults](err); ok {
		n.faults = faults
		return nil
	}
	if err != nil {
		return err
	}
	n.drift = drift
	if !read {
		return nil
	}
	if err := out.done(); err != nil {
		return err
	}
	f, err := openCompiled(n.path, out.f, out.size)
	if err != nil {
		return err
	}
	n.f = f
	return nil
}

// drift measures the clean files' drift, each finding to its file: a
// streamed catalog's is Compile's.
func (c *checker) drift() {
	files := c.clean(func(*File) bool { return true })
	if len(files) == 0 {
		return
	}
	all := Drift(c.m, files)
	if mm, ok := c.m.(ModelMonth); ok {
		if model := ModelDrift(mm, files); len(model) > 0 {
			order := make(map[string]int, len(files))
			for i, f := range files {
				order[f.Path] = i
			}
			all = slices.Concat(all, model)
			slices.SortStableFunc(all, func(a, b Finding) int {
				return cmp.Or(cmp.Compare(order[a.File], order[b.File]), cmp.Compare(a.Line, b.Line))
			})
		}
	}
	for _, g := range all {
		if n := c.files[c.byPath[g.File]]; !n.source {
			n.drift = append(n.drift, g)
		}
	}
}
