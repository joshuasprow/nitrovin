package vocab

import (
	"cmp"
	"maps"
	"slices"
	"strconv"
)

// Link checks files against each other, as loaded together: one name per
// file across the kinds, every extension reference defined by a loaded
// extension set (and naming an extension make where a make is meant),
// every part vocabulary's spine loaded and every id it maps to in that
// spine, and at most one catalog, its spine and vehicle vocabulary loaded,
// its part types in that spine and its vehicles named by exactly that
// vocabulary's fields; and of each stock source, a shelf's catalog
// loaded, every part and vehicle its rows name resolved (linkStock). The
// error is Faults.
func Link(files []*File) error {
	var faults Faults
	byName := map[string]*File{}
	for _, f := range files {
		if g, dup := byName[f.Name]; dup {
			faults = append(faults, fault(f.Path, 1, "/name: %q is also the name of %s", f.Name, g.Path))
			continue
		}
		byName[f.Name] = f
	}
	var catalog *File
	ext := func(ref string) (*Extension, string) { return resolveExt(byName, ref) }
	for _, f := range files {
		switch f.Kind {
		case KindVocabulary:
			var spine *File
			if f.Subject == SubjectPart {
				switch spine = byName[f.Spine]; {
				case spine == nil:
					faults = append(faults, fault(f.Path, 1, "/spine: spine %q is not loaded", f.Spine))
					continue
				case spine.Kind != KindSpine:
					faults = append(faults, fault(f.Path, 1, "/spine: %q is %s, not a spine", f.Spine, spine.Kind.noun()))
					continue
				}
			}
			for i := range f.Entries {
				e := &f.Entries[i]
				for j := range e.Mappings {
					switch mp := &e.Mappings[j]; {
					case mp.Ext != "":
						if _, why := ext(mp.Ext); why != "" {
							faults = append(faults, ofEntry(fault(f.Path, e.Line, "/mappings/%d/ext: %s", j, why), f, e))
						}
					case mp.ID != "":
						if _, ok := spine.SpineEntry(mp.ID); !ok {
							faults = append(faults, ofEntry(fault(f.Path, e.Line, "/mappings/%d/id: spine %q has no id %q", j, f.Spine, mp.ID), f, e))
						}
					}
				}
			}
		case KindExtensionSet:
			for _, x := range f.Extensions {
				if x.MakeExt == "" {
					continue
				}
				switch mk, why := ext(x.MakeExt); {
				case why != "":
					faults = append(faults, ofID(fault(f.Path, x.Line, "/make: %s", why), x.ID))
				case !mk.IsMake():
					faults = append(faults, ofID(fault(f.Path, x.Line, "/make: %s is an extension model, not a make", x.MakeExt), x.ID))
				}
			}
		case KindCatalog:
			if catalog != nil {
				faults = append(faults, fault(f.Path, 1, "/kind: %s is a catalog too: a deployment loads one catalog", catalog.Path))
				continue
			}
			catalog = f
			faults = append(faults, linkCatalog(f, byName, ext)...)
		}
	}
	for _, f := range files {
		if f.Kind == KindStock {
			faults = append(faults, linkStock(f, byName, catalog)...)
		}
	}
	if len(faults) > 0 {
		return faults
	}
	return nil
}

// resolveExt resolves an extension reference among files byName, or says
// why it does not.
func resolveExt(byName map[string]*File, ref string) (*Extension, string) {
	set, id := splitExt(ref)
	s := byName[set]
	switch {
	case s == nil:
		return nil, "extension set " + strconv.Quote(set) + " is not loaded"
	case s.Kind != KindExtensionSet:
		return nil, strconv.Quote(set) + " is " + s.Kind.noun() + ", not an extension set"
	}
	x, ok := s.Extension(id)
	if !ok {
		return nil, "extension set " + strconv.Quote(set) + " has no id " + strconv.Quote(id)
	}
	return x, ""
}

// linkCatalog checks catalog f against the files loaded with it, byName:
// its spine and vehicle vocabulary loaded, every part type of its
// fitments and redirects in that spine, and every vehicle named by
// exactly that vocabulary's fields, or by an extension a loaded set
// defines. Its faults come in line order. A compiled catalog, checked line
// by line when it was compiled, is checked by its dictionary: each part
// type and vehicle once, about the file as a whole.
func linkCatalog(f *File, byName map[string]*File, ext func(string) (*Extension, string)) Faults {
	l := newCatalogLink(f, byName, ext)
	if f.compiled != nil {
		return append(l.faults, f.compiled.link(l)...)
	}
	for _, v := range f.catalog.vehicleList {
		l.vehicle(v)
	}
	for i := range f.Fitments {
		l.fitment(&f.Fitments[i])
	}
	for i := range f.Redirects {
		l.redirect(&f.Redirects[i])
	}
	slices.SortStableFunc(l.faults, func(a, b Finding) int { return cmp.Compare(a.Line, b.Line) })
	return l.faults
}

// catalogLink checks a catalog's lines against its spine, vehicle
// vocabulary and filed-under vocabulary as loaded, a line at a time: a
// JSON Lines catalog's in memory, a compile's as its source streams by.
// Its faults are in the order of the lines checked, after its header's.
type catalogLink struct {
	f               *File // the catalog: its header
	spine, vehicles *File // nil when not loaded, or when its vehicles are vpic
	filed           *File // its filed-under vocabulary; nil when it names none, or it is not loaded
	ext             func(string) (*Extension, string)
	// why is, per vehicle that does not resolve, why not: once each,
	// however many lines name it.
	why map[*Vehicle]string
	// entries is, per entry fitments are filed under, what the filed-under
	// vocabulary says of it: once each, however many fitments name it.
	entries map[*Filed]filedLink
	faults  Faults
}

// filedLink is what a catalog's filed-under vocabulary says of an entry:
// why its fields are not the vocabulary's, or the part types the entry
// maps to, none when the vocabulary lacks it or maps it nowhere, which is
// drift, not a fault.
type filedLink struct {
	why   string
	types []string
}

// newCatalogLink checks catalog f's header: its spine, vehicle vocabulary
// and filed-under vocabulary loaded, among byName.
func newCatalogLink(f *File, byName map[string]*File, ext func(string) (*Extension, string)) *catalogLink {
	l := &catalogLink{f: f, ext: ext, why: map[*Vehicle]string{}, entries: map[*Filed]filedLink{}}
	header := 1 // the header's line
	if f.compiled != nil {
		header = 0
	}
	l.spine = byName[f.Spine]
	switch {
	case l.spine == nil:
		l.faults = append(l.faults, fault(f.Path, header, "/spine: spine %q is not loaded", f.Spine))
	case l.spine.Kind != KindSpine:
		l.faults = append(l.faults, fault(f.Path, header, "/spine: %q is %s, not a spine", f.Spine, l.spine.Kind.noun()))
		l.spine = nil
	}
	if f.Vehicles != Reserved {
		switch l.vehicles = byName[f.Vehicles]; {
		case l.vehicles == nil:
			l.faults = append(l.faults, fault(f.Path, header, "/vehicles: vocabulary %q is not loaded", f.Vehicles))
		case l.vehicles.Kind != KindVocabulary:
			l.faults = append(l.faults, fault(f.Path, header, "/vehicles: %q is %s, not a vehicle vocabulary", f.Vehicles, l.vehicles.Kind.noun()))
			l.vehicles = nil
		case l.vehicles.Subject != SubjectVehicle:
			l.faults = append(l.faults, fault(f.Path, header, "/vehicles: %q is a part vocabulary, not a vehicle vocabulary", f.Vehicles))
			l.vehicles = nil
		case l.vehicles.Case != f.VehiclesCase:
			l.faults = append(l.faults, caseFault(f, header, "/vehicles", l.vehicles, f.VehiclesCase))
		}
	}
	if f.FiledUnder != "" {
		switch l.filed = byName[f.FiledUnder]; {
		case l.filed == nil:
			l.faults = append(l.faults, fault(f.Path, header, "/filed_under: vocabulary %q is not loaded", f.FiledUnder))
		case l.filed.Kind != KindVocabulary:
			l.faults = append(l.faults, fault(f.Path, header, "/filed_under: %q is %s, not a part vocabulary", f.FiledUnder, l.filed.Kind.noun()))
			l.filed = nil
		case l.filed.Subject != SubjectPart:
			l.faults = append(l.faults, fault(f.Path, header, "/filed_under: %q is a vehicle vocabulary, not a part vocabulary", f.FiledUnder))
			l.filed = nil
		case l.filed.Spine != f.Spine:
			l.faults = append(l.faults, fault(f.Path, header, "/filed_under: vocabulary %q maps onto spine %q, not the catalog's spine %q", f.FiledUnder, l.filed.Spine, f.Spine))
			l.filed = nil
		case l.filed.Case != f.FiledUnderCase:
			l.faults = append(l.faults, caseFault(f, header, "/filed_under", l.filed, f.FiledUnderCase))
		}
	}
	return l
}

// caseFault refuses catalog f, whose vehicles or entries match as keyed
// does, beside vocabulary g, named at ptr in its header on line header,
// which matches case otherwise: f was compiled beside another g, or, JSON
// Lines, read without g (Read), folding case (ADR 0026).
func caseFault(f *File, header int, ptr string, g *File, keyed Case) Finding {
	if f.compiled != nil {
		return fault(f.Path, header, "%s: vocabulary %q %s, and the catalog was compiled against it %s: compile it again beside the vocabulary", ptr, g.Name, g.Case.of(), keyed.as())
	}
	return fault(f.Path, header, "%s: vocabulary %q %s, and the catalog was read %s: open it beside the vocabulary", ptr, g.Name, g.Case.of(), keyed.as())
}

// vehicle checks vehicle v, before any line naming it.
func (l *catalogLink) vehicle(v *Vehicle) {
	if w := vehicleWhy(v, l.vehicles, l.ext); w != "" {
		l.why[v] = w
	}
}

// typeOK reports whether part type id is in the spine, or the spine is
// not loaded, which the header's fault says.
func (l *catalogLink) typeOK(id string) bool {
	if l.spine == nil {
		return true
	}
	_, ok := l.spine.SpineEntry(id)
	return ok
}

// fitment checks a fitment's part type, the entries it is filed under and
// its applications' vehicles.
func (l *catalogLink) fitment(fit *Fitment) {
	if !l.typeOK(fit.Key.Type) {
		l.faults = append(l.faults, fault(l.f.Path, fit.Line, "/fitment/type: spine %q has no id %q", l.f.Spine, fit.Key.Type))
	}
	if l.filed != nil {
		for _, e := range fit.FiledUnder {
			if w := l.filedWhy(e, fit.Key); w != "" {
				l.faults = append(l.faults, fault(l.f.Path, fit.Line, "/filed_under: %s", w))
			}
		}
	}
	if len(l.why) == 0 {
		return
	}
	for j := range fit.Applications {
		if w, bad := l.why[fit.Applications[j].Vehicle]; bad {
			l.faults = append(l.faults, fault(l.f.Path, fit.Line, "/applications/%d/vehicle%s", j, w))
		}
	}
}

// filedWhy is why a fitment of key k may not be filed under entry e of the
// catalog's filed-under vocabulary, "" when it may: e's fields are not the
// vocabulary's, or the vocabulary maps e to part types none of which is
// k's. An entry the vocabulary lacks, or maps nowhere, is drift.
func (l *catalogLink) filedWhy(e *Filed, k FitmentKey) string {
	el, seen := l.entries[e]
	if !seen {
		el = linkFiled(e, l.filed)
		l.entries[e] = el
	}
	if el.why != "" || len(el.types) == 0 || slices.Contains(el.types, k.Type) {
		return el.why
	}
	return "vocabulary " + strconv.Quote(l.filed.Name) + " maps " + entryText(e.Entry) + " to part types " + quoted(el.types) +
		", none of them the part type of " + k.String() + ", " + strconv.Quote(k.Type)
}

// linkFiled is what filed-under vocabulary filed says of entry e.
func linkFiled(e *Filed, filed *File) filedLink {
	names := slices.Sorted(maps.Keys(e.Entry))
	if want := slices.Sorted(slices.Values(filed.Fields)); !slices.Equal(names, want) {
		return filedLink{why: entryText(e.Entry) + ": fields " + quoted(names) + " are not vocabulary " + strconv.Quote(filed.Name) + "'s: " + quoted(filed.Fields)}
	}
	var el filedLink
	if entry, ok := filed.Entry(e.Entry); ok {
		for _, mp := range entry.Mappings {
			if mp.ID != "" && !slices.Contains(el.types, mp.ID) {
				el.types = append(el.types, mp.ID)
			}
		}
	}
	return el
}

// redirect checks a redirect's part types and vehicle.
func (l *catalogLink) redirect(rd *Redirect) {
	for _, t := range []struct{ ptr, id string }{{"from", rd.From}, {"to", rd.To}} {
		if !l.typeOK(t.id) {
			l.faults = append(l.faults, fault(l.f.Path, rd.Line, "/redirect/%s: spine %q has no id %q", t.ptr, l.f.Spine, t.id))
		}
	}
	if w, bad := l.why[rd.Vehicle]; bad {
		l.faults = append(l.faults, fault(l.f.Path, rd.Line, "/redirect/vehicle%s", w))
	}
}

// vehicleWhy is why vehicle v of a catalog does not resolve against its
// vehicle vocabulary vehicles (nil when it is vpic or not loaded) and the
// extension sets ext reads, as a pointer's tail and a reason; "" when it
// does.
func vehicleWhy(v *Vehicle, vehicles *File, ext func(string) (*Extension, string)) string {
	switch {
	case v.Ext != "":
		if _, w := ext(v.Ext); w != "" {
			return "/ext: " + w
		}
	case vehicles != nil:
		names := slices.Sorted(maps.Keys(v.Entry))
		if want := slices.Sorted(slices.Values(vehicles.Fields)); !slices.Equal(names, want) {
			return ": fields " + quoted(names) + " are not vocabulary " + strconv.Quote(vehicles.Name) + "'s: " + quoted(vehicles.Fields)
		}
	}
	return ""
}

// link checks a compiled catalog against its spine, vehicle vocabulary and
// filed-under vocabulary as l has them loaded (nil when not, or when its
// vehicles are vpic): each part type a fitment or redirect names in the
// spine, each vehicle resolving, and each entry its fitments are filed
// under mapped to their part types. They did against the files it was
// compiled with; a fault here is a file loaded beside it that is not one
// of those.
func (c *compiled) link(l *catalogLink) Faults {
	spine, vehicles, ext := l.spine, l.vehicles, l.ext
	var faults Faults
	for _, t := range c.types {
		if spine == nil || !t.named {
			continue
		}
		if _, ok := spine.SpineEntry(t.name); !ok {
			faults = append(faults, fault(c.path, 0, "part type %q: spine %q has no id %q, and had when the catalog was compiled", t.name, spine.Name, t.name))
		}
	}
	for _, v := range c.vehicles {
		if w := vehicleWhy(v, vehicles, ext); w != "" {
			faults = append(faults, fault(c.path, 0, "vehicle %s%s", vehicleText(v), w))
		}
	}
	if l.filed != nil {
		for i, e := range c.filed {
			last := ""
			for _, k := range c.filedTypes[i] {
				if w := l.filedWhy(e, k); w != "" && w != last {
					faults = append(faults, fault(c.path, 0, "/filed_under: %s", w))
					last = w
				}
			}
		}
	}
	return faults
}

// vehicleText is a catalog's vehicle as a file writes it.
func vehicleText(v *Vehicle) string {
	switch {
	case v.Entry != nil:
		return entryText(v.Entry)
	case v.Make != 0:
		return `{"make":` + strconv.Itoa(int(v.Make)) + "}"
	case v.Model != 0:
		return `{"model":` + strconv.Itoa(int(v.Model)) + "}"
	}
	return `{"ext":` + strconv.Quote(v.Ext) + "}"
}
