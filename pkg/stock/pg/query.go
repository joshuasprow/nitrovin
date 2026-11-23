package pg

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joshuasprow/nitrovin/pkg/stock"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

// queries are a checked relation's three lookups, each one SELECT that
// answers every list of a call: shelf by fitment keys, shelf rows with no
// number by part type, vehicle and year, and yard by vPIC model or catalog
// vehicle and years.
//
// Each takes the same first four parameters: $1 how many lists, $2 the
// locations to keep (empty: every one), $3 the page's offset and $4 its
// limit; then the lookup's own, one array per column of its keys. A
// shelf's keys are each tagged with its list; a yard's are numbered, and
// a last table names each list's keys and years. Its result is a row per
// list's page row, in list order then id order, or one row with a null id
// for a list whose page is empty, each carrying its list's total and the
// release.
type queries struct {
	byFitment, byVehicle, yard string
	// stocked is a yard's best-stocked vehicles (Stocked), "" for a
	// shelf and for a yard whose relation has no vin column.
	stocked string
	// nv is how many vehicle columns there are: fields' (a vocabulary's
	// fields), or vpic's three.
	nv     int
	fields []string
	vpic   bool
}

// col is a relation column as a query reads it.
func col(name string) string { return "r." + pgx.Identifier{name}.Sanitize() }

// queries are the lookups of the source's relation, whose as_of column
// has data type asOf ("" when it has none), whose columns padded are
// character under a deterministic collation, and which, vin, has a yard's
// vin column.
func (s *Source) queries(asOf string, padded map[string]bool, vin bool) *queries {
	q := &queries{fields: s.cols.fields, vpic: s.cols.vpic}
	vehicle := s.cols.vehicle()
	q.nv = len(vehicle)
	rel := quoted(s.c.Relation)
	release := "NULL::text"
	if s.c.ReleaseRelation != "" {
		release = "(SELECT " + pgx.Identifier{"release"}.Sanitize() + "::text FROM " + quoted(s.c.ReleaseRelation) + ")"
	}
	asOfExpr := "NULL::text"
	switch asOf {
	case "timestamp with time zone":
		asOfExpr = `to_char(` + col("as_of") + ` AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`
	case "timestamp without time zone": // read as UTC
		asOfExpr = `to_char(` + col("as_of") + `, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`
	case "date":
		asOfExpr = `to_char(` + col("as_of") + `, 'YYYY-MM-DD')`
	}

	// selected is a matched row's columns, in result order: shelf names
	// its fitment key or its part type, and vehicles its vehicle and year.
	selected := func(shelfKey, partType, vehicles, decoded bool) string {
		var b []string
		b = append(b, col("id")+"::text AS c_id", col("location")+"::text AS c_location")
		for _, c := range []string{"type", "number", "side"} {
			if shelfKey {
				b = append(b, col(c)+"::text AS c_"+c)
			} else {
				b = append(b, "NULL::text AS c_"+c)
			}
		}
		if partType {
			b = append(b, col("part_type")+"::text AS c_part_type")
		} else {
			b = append(b, "NULL::text AS c_part_type")
		}
		for i, c := range vehicle {
			if vehicles {
				b = append(b, col(c)+"::text AS c_v"+strconv.Itoa(i))
			} else {
				b = append(b, "NULL::text AS c_v"+strconv.Itoa(i))
			}
		}
		if vehicles {
			b = append(b, col("year")+"::int4 AS c_year")
		} else {
			b = append(b, "NULL::int4 AS c_year")
		}
		for _, c := range []string{"decoded_make", "decoded_model", "decoded_year"} {
			if decoded {
				b = append(b, col(c)+"::int4 AS c_"+c)
			} else {
				b = append(b, "NULL::int4 AS c_"+c)
			}
		}
		// jsonb's text: a json column keeps its input's, spaces and all.
		b = append(b, col("fields")+"::jsonb::text AS c_fields", asOfExpr+" AS c_as_of")
		if vin && decoded {
			b = append(b, col("vin")+"::text AS c_vin")
		} else {
			b = append(b, "NULL::text AS c_vin")
		}
		return strings.Join(b, ", ")
	}
	where := col("id") + " IS NOT NULL AND " + col("location") + " IS NOT NULL AND (cardinality($2::text[]) = 0 OR " + col("location") + " = ANY($2::text[]))"
	// is holds relation column name to a key's value v. Postgres compares
	// a character column with a text key as text, through a cast that no
	// index on the column serves and no security_barrier view passes, so
	// a padded column is first held to the key as character, which both
	// do. That changes no answer: under a deterministic collation the
	// text comparison is true only of a key equal byte for byte to the
	// value without its padding, so one that ends in no space, and of
	// those the character comparison, which ignores each side's trailing
	// spaces, is true too.
	is := func(name, v string) string {
		if padded[name] {
			return col(name) + " = " + v + "::bpchar AND " + col(name) + " = " + v
		}
		return col(name) + " = " + v
	}
	// vehicleIs matches a relation row's vehicle to the vehicle columns
	// of key k, c_v0…: a vocabulary's every field, or vpic's make, model
	// and ext as a whole, as a file's vehicle key does, so a row matches
	// at most one of a list's keys. A vpic key sets one of the three: the
	// row equals it there and is null in the other two, one branch per
	// column, each an equality and IS NULLs an index can serve.
	vehicleIs := func(k string) string {
		if q.vpic {
			branches := make([]string, len(vehicle))
			for i, c := range vehicle {
				b := []string{is(c, k+".v"+strconv.Itoa(i))}
				for j, other := range vehicle {
					if j != i {
						b = append(b, col(other)+" IS NULL")
					}
				}
				branches[i] = "(" + strings.Join(b, " AND ") + ")"
			}
			return "(" + strings.Join(branches, " OR ") + ")"
		}
		var b []string
		for i, c := range vehicle {
			b = append(b, is(c, k+".v"+strconv.Itoa(i)))
		}
		return strings.Join(b, " AND ")
	}
	// keys is a lookup's keys, an unnest of one array per column from
	// parameter $5: names are its columns, types their array types.
	keys := func(name string, from int, names, types []string) string {
		params := make([]string, len(names))
		for i := range names {
			params[i] = "$" + strconv.Itoa(from+i) + "::" + types[i]
		}
		return name + " AS (SELECT * FROM unnest(" + strings.Join(params, ", ") + ") AS " + name + "(" + strings.Join(names, ", ") + "))"
	}
	vnames, vtypes := make([]string, len(vehicle)), make([]string, len(vehicle))
	for i := range vehicle {
		vnames[i], vtypes[i] = "v"+strconv.Itoa(i), "text[]"
		if q.vpic && i < 2 {
			vtypes[i] = "int4[]"
		}
	}
	// cols are a result row's columns after list, total, lists and
	// release, each with its type.
	cols := [][2]string{{"c_id", "text"}, {"c_location", "text"}, {"c_type", "text"}, {"c_number", "text"}, {"c_side", "text"}, {"c_part_type", "text"}}
	for i := range vehicle {
		cols = append(cols, [2]string{"c_v" + strconv.Itoa(i), "text"})
	}
	cols = append(cols, [2]string{"c_year", "int4"}, [2]string{"c_decoded_make", "int4"}, [2]string{"c_decoded_model", "int4"}, [2]string{"c_decoded_year", "int4"},
		[2]string{"c_by_model", "bool"}, [2]string{"c_fields", "text"}, [2]string{"c_as_of", "text"}, [2]string{"c_vin", "text"})
	rowCols, nullCols := make([]string, len(cols)), make([]string, len(cols))
	for i, c := range cols {
		rowCols[i], nullCols[i] = "f."+c[0], "NULL::"+c[1]+" AS "+c[0]
	}
	// page answers the rows read, each numbered n in f, and assign, the
	// lists' rows of them: (list, n, c_id) of f. Its result is a row for
	// each list, with its total and no row columns, then each row in any
	// list's page, once however many lists' pages hold it, with those
	// lists, in id order, which is each list's own: a list's rows read
	// once for many lists are sent once (#354).
	page := func(with, read, assign string) string {
		return "WITH " + with + ",\n" +
			"f AS (SELECT row_number() OVER () AS n, r.* FROM (" + read + ") r),\n" +
			"a AS (" + assign + "),\n" +
			"t AS (SELECT list, count(*) AS total FROM a GROUP BY list),\n" +
			`p AS (SELECT n, array_agg(list ORDER BY list) AS lists FROM (SELECT a.*, row_number() OVER (PARTITION BY a.list ORDER BY a.c_id COLLATE "C") AS rn FROM a) a` +
			" WHERE rn > $3::int8 AND rn <= $3::int8 + $4::int8 GROUP BY n)\n" +
			"SELECT * FROM (SELECT l.list, coalesce(t.total, 0) AS total, NULL::int4[] AS lists, " + release + " AS release, " + strings.Join(nullCols, ", ") + "\n" +
			"FROM generate_series(0, $1::int4 - 1) AS l(list) LEFT JOIN t ON t.list = l.list\n" +
			"UNION ALL\n" +
			"SELECT NULL, NULL, p.lists, NULL, " + strings.Join(rowCols, ", ") + " FROM p JOIN f ON f.n = p.n) x\n" +
			`ORDER BY x.c_id COLLATE "C" NULLS FIRST`
	}
	// matched are the relation's rows of each key of k, each tagged with
	// its key's column tag: a row's columns sel and, for a yard's,
	// whether its decode matched, byModel.
	//
	// Every condition on the relation, cond and where, is in a subquery
	// LATERAL to the key, fenced with OFFSET 0 so that the planner cannot
	// pull it up. There the key is a parameter and each condition a
	// restriction on the relation, which a security_barrier view passes
	// to its tables' indexes when every function it gives a relation
	// column to is leakproof (ADR 0020). Pulled up, or written as a join,
	// they are join conditions, which no barrier lets through: the lookup
	// computes the whole view. So, a character column's cast to text
	// aside (is, above), a relation column goes only to a text, character
	// or integer comparison, IS NULL, IS NOT DISTINCT FROM or = ANY, which
	// are leakproof. The fence has a price where no index serves a key:
	// the lookup reads the relation once per key, not once.
	matched := func(k, tag, sel, byModel, cond string) string {
		return "SELECT " + k + "." + tag + ", found.* FROM " + k + " CROSS JOIN LATERAL (SELECT " + sel + ", " + byModel + " AS c_by_model FROM " + rel + " r WHERE " +
			cond + " AND " + where + " OFFSET 0) found"
	}

	if s.info.Holds == stock.Shelf {
		// A shelf's keys are each its list's: a row read is in that list.
		const own = "SELECT list, n, c_id FROM f"
		q.byFitment = page(
			keys("k", 5, []string{"list", "type", "number", "side"}, []string{"int4[]", "text[]", "text[]", "text[]"}),
			matched("k", "list", selected(true, false, false, false), "NULL::bool",
				is("type", "k.type")+" AND "+is("number", "k.number")+" AND "+col("side")+" IS NOT DISTINCT FROM k.side"), own)
		q.byVehicle = page(
			keys("k", 5, append([]string{"list", "part_type", "year"}, vnames...), append([]string{"int4[]", "text[]", "int4[]"}, vtypes...)),
			matched("k", "list", selected(false, true, true, false), "NULL::bool",
				col("number")+" IS NULL AND "+is("part_type", "k.part_type")+" AND "+vehicleIs("k")+" AND "+
					col("year")+" IS NOT NULL AND (k.year IS NULL OR "+col("year")+" = k.year)"), own)
		return q
	}
	decoded := col("decoded_make") + " IS NOT NULL AND " + col("decoded_model") + " IS NOT NULL AND " + col("decoded_year") + " IS NOT NULL"
	within := func(k, year string) string {
		return "(" + k + ".first IS NULL OR " + year + " >= " + k + ".first) AND (" + k + ".last IS NULL OR " + year + " <= " + k + ".last)"
	}
	// A yard's keys are each model and vehicle any list names, read once
	// in every year any list names it in, km and kv, each numbered key;
	// lk is each list's keys and years. A row a key reads is in every
	// list naming that key in a year holding the row's: its decoded year
	// when its decode matched, else its year. One read of a key serves
	// every list naming it, so a batch's many fitments of the same
	// vehicles read each vehicle's rows once, not once a fitment (#354).
	// lk's conditions are on the rows read, outside the fence, so no
	// barrier ever sees them.
	read := matched("km", "key", selected(false, false, true, true), "true",
		col("decoded_model")+" = km.model AND "+decoded+" AND "+within("km", col("decoded_year")))
	with := keys("km", 5, []string{"key", "model", "first", "last"}, []string{"int4[]", "int4[]", "int4[]", "int4[]"})
	next := 9
	if len(vehicle) > 0 {
		with += ",\n" + keys("kv", 9, append([]string{"key", "first", "last"}, vnames...), append([]string{"int4[]", "int4[]", "int4[]"}, vtypes...))
		read += "\nUNION ALL\n" + matched("kv", "key", selected(false, false, true, true), "false",
			vehicleIs("kv")+" AND "+col("year")+" IS NOT NULL AND "+within("kv", col("year"))+" AND NOT ("+decoded+")")
		next = 12 + len(vehicle)
	}
	with += ",\n" + keys("lk", next, []string{"list", "key", "first", "last"}, []string{"int4[]", "int4[]", "int4[]", "int4[]"})
	q.yard = page(with, read, "SELECT lk.list, f.n, f.c_id FROM f JOIN lk ON lk.key = f.key AND "+
		within("lk", "CASE WHEN f.c_by_model THEN f.c_decoded_year ELSE f.c_year END"))
	if vin {
		// The yard's vehicles by their decode, each model once in the year
		// it has the most rows in (the later of two alike), the most rows
		// first: $1 how many, $2 the locations to keep. One read of the
		// relation, as an aggregate of all of it must be. The least VIN of
		// a vehicle's rows is bytewise, as a file's; an empty one is none.
		least := `min(NULLIF(` + col("vin") + `::text, '') COLLATE "C")`
		q.stocked = "SELECT m.* FROM (SELECT DISTINCT ON (g.model) g.* FROM (\n" +
			"SELECT " + col("decoded_model") + "::int4 AS model, " + col("decoded_year") + "::int4 AS year, count(*)::int8 AS n, " + least + " AS vin, min(" + asOfExpr + ") AS as_of\n" +
			"FROM " + rel + " r WHERE " + decoded + " AND " + where + "\n" +
			"GROUP BY 1, 2 HAVING " + least + " IS NOT NULL) g ORDER BY g.model, g.n DESC, g.year DESC) m\n" +
			"ORDER BY m.n DESC, m.model LIMIT $1::int8"
	}
	return q
}

// Stocked answers the yard's best-stocked vehicles (stock.Stocker): the
// vehicles its rows' decodes name, each with how many rows and the least
// of their VINs. A relation with no vin column has no VIN to offer one
// by, and answers none; so does a shelf.
func (s *Source) Stocked(ctx context.Context, sq stock.StockedQuery) ([]stock.StockedVehicle, stock.Meta, error) {
	out := []stock.StockedVehicle{}
	if s.info.Holds != stock.Yard || sq.Limit <= 0 {
		return out, stock.Meta{}, nil
	}
	q, pool, err := s.ready(ctx)
	if err != nil {
		return nil, stock.Meta{}, err
	}
	if q.stocked == "" {
		return out, stock.Meta{}, nil
	}
	bctx, cancel := context.WithTimeout(ctx, s.c.ConnectTimeout+s.c.StatementTimeout)
	defer cancel()
	conn, err := pool.Acquire(bctx)
	if err != nil {
		return nil, stock.Meta{}, s.acquireFailed(ctx, pool, err)
	}
	defer conn.Release()
	location := sq.Location
	if location == nil {
		location = []string{}
	}
	rows, err := conn.Query(bctx, q.stocked, int64(sq.Limit), location)
	if err != nil {
		return nil, stock.Meta{}, s.unavailable(ctx, err)
	}
	var (
		model, year int32
		n           int64
		vin         string
		asOf        *string
		meta        stock.Meta
	)
	if _, err := pgx.ForEachRow(rows, []any{&model, &year, &n, &vin, &asOf}, func() error {
		out = append(out, stock.StockedVehicle{Model: model, Year: int(year), Rows: int(n), VIN: vin})
		// The oldest of the rows counted, as a list's block says.
		if asOf != nil && (meta.AsOf == "" || *asOf < meta.AsOf) {
			meta.AsOf = *asOf
		}
		return nil
	}); err != nil {
		return nil, stock.Meta{}, s.unavailable(ctx, err)
	}
	return out, meta, nil
}

// ShelfByFitment answers the shelf rows of each list's keys: one query.
func (s *Source) ShelfByFitment(ctx context.Context, lists [][]vocab.FitmentKey, p stock.Page) ([]stock.Result, stock.Meta, error) {
	var li []int32
	var types, numbers []string
	var sides []*string
	for i, keys := range lists {
		seen := map[vocab.FitmentKey]bool{}
		for _, k := range keys {
			if seen[k] {
				continue
			}
			seen[k] = true
			li, types, numbers = append(li, int32(i)), append(types, k.Type), append(numbers, k.Number)
			var side *string
			if k.Side != vocab.SideNone {
				side = new(k.Side.String())
			}
			sides = append(sides, side)
		}
	}
	return s.lookup(ctx, stock.Shelf, len(lists), p, func(q *queries) string { return q.byFitment }, stock.MatchedFitment, li, types, numbers, sides)
}

// ShelfByVehicle answers the shelf rows with no number of each list's part
// types, vehicles and year: one query.
func (s *Source) ShelfByVehicle(ctx context.Context, lists []stock.ShelfVehicle, p stock.Page) ([]stock.Result, stock.Meta, error) {
	var li []int32
	var types []string
	var years []*int32
	vs := s.vehicleArgs()
	for i, l := range lists {
		seen := map[string]bool{}
		for _, t := range l.PartTypes {
			for j := range l.Vehicles {
				v := &l.Vehicles[j]
				k := t + "\x00" + vocab.VehicleKey(v)
				if seen[k] || !vs.add(v) {
					continue
				}
				seen[k] = true
				li, types, years = append(li, int32(i)), append(types, t), append(years, year32(l.Year))
			}
		}
	}
	return s.lookup(ctx, stock.Shelf, len(lists), p, func(q *queries) string { return q.byVehicle }, stock.MatchedVehicle, append([]any{li, types, years}, vs.args()...)...)
}

// Yard answers the yard rows of each list's vPIC models and catalog
// vehicles in their years: one query. Each model and vehicle the lists
// name is one key, read once in every year any list names it in, and each
// list keeps the rows of its own years. A model or vehicle named twice in
// a list is kept in the union of its years, so no row matches twice.
func (s *Source) Yard(ctx context.Context, lists []stock.YardQuery, p stock.Page) ([]stock.Result, stock.Meta, error) {
	// key is a model, or a vehicle v, its number, and the years the lists
	// name it in; read is whether it is read at all: a vehicle with none
	// of the source's columns is not.
	type key struct {
		n     int32
		model int32
		v     *vocab.Vehicle
		years []span
		read  bool
	}
	type listKey struct {
		list int32
		k    *key
		sp   span
	}
	var mkeys, vkeys []*key
	var named []listKey
	byModel, byVehicle := map[int32]*key{}, map[string]*key{}
	for i, l := range lists {
		spans := map[int32][]span{}
		var order []int32
		for _, m := range l.Models {
			if _, ok := spans[m.Model]; !ok {
				order = append(order, m.Model)
			}
			spans[m.Model] = append(spans[m.Model], spanOf(m.First, m.Last))
		}
		for _, m := range order {
			k := byModel[m]
			if k == nil {
				k = &key{n: int32(len(mkeys) + len(vkeys)), model: m}
				byModel[m] = k
				mkeys = append(mkeys, k)
			}
			for _, sp := range merge(spans[m]) {
				k.years = append(k.years, sp)
				named = append(named, listKey{int32(i), k, sp})
			}
		}
		if len(s.cols.vehicle()) == 0 {
			continue // no catalog: no vehicle columns, and no vehicle to ask for
		}
		vspans := map[string][]span{}
		var vorder []*vocab.Vehicle
		for j := range l.Vehicles {
			v := &l.Vehicles[j]
			vk := vocab.VehicleKey(&v.Vehicle)
			if _, ok := vspans[vk]; !ok {
				vorder = append(vorder, &v.Vehicle)
			}
			vspans[vk] = append(vspans[vk], spanOf(v.First, v.Last))
		}
		for _, v := range vorder {
			vk := vocab.VehicleKey(v)
			k := byVehicle[vk]
			if k == nil {
				k = &key{n: int32(len(mkeys) + len(vkeys)), v: v}
				byVehicle[vk] = k
				vkeys = append(vkeys, k)
			}
			for _, sp := range merge(vspans[vk]) {
				k.years = append(k.years, sp)
				named = append(named, listKey{int32(i), k, sp})
			}
		}
	}
	var mn, models, vn []int32
	var mfirst, mlast, vfirst, vlast []*int32
	for _, k := range mkeys {
		k.read = true
		for _, sp := range merge(k.years) {
			mn, models = append(mn, k.n), append(models, k.model)
			mfirst, mlast = append(mfirst, sp.first()), append(mlast, sp.last())
		}
	}
	vs := s.vehicleArgs()
	for _, k := range vkeys {
		for _, sp := range merge(k.years) {
			if k.read = vs.add(k.v); !k.read {
				break
			}
			vn, vfirst, vlast = append(vn, k.n), append(vfirst, sp.first()), append(vlast, sp.last())
		}
	}
	var li, ln []int32
	var lfirst, llast []*int32
	for _, nk := range named {
		if nk.k.read {
			li, ln = append(li, nk.list), append(ln, nk.k.n)
			lfirst, llast = append(lfirst, nk.sp.first()), append(llast, nk.sp.last())
		}
	}
	args := []any{mn, models, mfirst, mlast}
	if len(s.cols.vehicle()) > 0 {
		args = append(append(args, vn, vfirst, vlast), vs.args()...)
	}
	args = append(args, li, ln, lfirst, llast)
	return s.lookup(ctx, stock.Yard, len(lists), p, func(q *queries) string { return q.yard }, "", args...)
}

// vehicleArgs are the vehicle columns of a lookup's keys, one array per
// column, for the source's vehicle columns.
type vehicleArgs struct {
	fields []string
	vpic   bool
	text   [][]*string // a vocabulary's fields, or vpic's ext as text[2]
	ids    [2][]*int32 // vpic's make and model
}

func (s *Source) vehicleArgs() *vehicleArgs {
	return &vehicleArgs{fields: s.cols.fields, vpic: s.cols.vpic, text: make([][]*string, max(len(s.cols.fields), 3))}
}

// add adds v's columns, false when it has none of the source's: a vehicle
// that names fields its vocabulary lacks, or of the other kind.
func (a *vehicleArgs) add(v *vocab.Vehicle) bool {
	if a.vpic {
		if v.Entry != nil {
			return false
		}
		a.ids[0], a.ids[1] = append(a.ids[0], nonZero(v.Make)), append(a.ids[1], nonZero(v.Model))
		var ext *string
		if v.Ext != "" {
			ext = new(v.Ext)
		}
		a.text[2] = append(a.text[2], ext)
		return true
	}
	vals := make([]*string, len(a.fields))
	for i, f := range a.fields {
		val, ok := v.Entry[f]
		if !ok {
			return false
		}
		vals[i] = new(val)
	}
	for i := range vals {
		a.text[i] = append(a.text[i], vals[i])
	}
	return true
}

// args are the arrays, in column order.
func (a *vehicleArgs) args() []any {
	if a.vpic {
		return []any{a.ids[0], a.ids[1], a.text[2]}
	}
	out := make([]any, len(a.fields))
	for i := range a.fields {
		out[i] = a.text[i]
	}
	return out
}

func nonZero(id int32) *int32 {
	if id == 0 {
		return nil
	}
	return new(id)
}

func year32(y *int) *int32 {
	if y == nil {
		return nil
	}
	return new(int32(*y))
}

// span is a range of model years, each end inclusive, unbounded at
// math.MinInt32 or math.MaxInt32.
type span struct{ lo, hi int64 }

func spanOf(first, last *int) span {
	sp := span{math.MinInt32, math.MaxInt32}
	if first != nil {
		sp.lo = int64(*first)
	}
	if last != nil {
		sp.hi = int64(*last)
	}
	return sp
}

func (sp span) first() *int32 {
	if sp.lo == math.MinInt32 {
		return nil
	}
	return new(int32(sp.lo))
}

func (sp span) last() *int32 {
	if sp.hi == math.MaxInt32 {
		return nil
	}
	return new(int32(sp.hi))
}

// merge is spans as the fewest disjoint spans holding the same years, in
// order: overlapping and adjacent spans join.
func merge(spans []span) []span {
	slices.SortFunc(spans, func(a, b span) int { return cmp.Compare(a.lo, b.lo) })
	var out []span
	for _, sp := range spans {
		if n := len(out); n > 0 && sp.lo <= out[n-1].hi+1 {
			out[n-1].hi = max(out[n-1].hi, sp.hi)
			continue
		}
		out = append(out, sp)
	}
	return out
}

// lookup runs one of the relation's queries for n lists of what holds,
// with p's page and the lookup's own args: every list's page and total,
// each row with its own as_of, and the release it read; Meta.AsOf is "",
// so each list's block says the oldest of its own rows' times.
// matched is how a shelf row matched; a yard row says its own.
func (s *Source) lookup(ctx context.Context, holds stock.Holds, n int, p stock.Page, sql func(*queries) string, matched stock.Matched, args ...any) ([]stock.Result, stock.Meta, error) {
	out := make([]stock.Result, n)
	for i := range out {
		out[i].Rows = []stock.Row{}
	}
	if s.info.Holds != holds || n == 0 {
		return out, stock.Meta{}, nil
	}
	q, pool, err := s.ready(ctx)
	if err != nil {
		return nil, stock.Meta{}, err
	}
	bctx, cancel := context.WithTimeout(ctx, s.c.ConnectTimeout+s.c.StatementTimeout)
	defer cancel()
	conn, err := pool.Acquire(bctx)
	if err != nil {
		return nil, stock.Meta{}, s.acquireFailed(ctx, pool, err)
	}
	defer conn.Release()
	location := p.Location
	if location == nil {
		location = []string{}
	}
	rows, err := conn.Query(bctx, sql(q), append([]any{int32(n), location, int64(p.Offset), int64(max(p.Limit, 0))}, args...)...)
	if err != nil {
		return nil, stock.Meta{}, s.unavailable(ctx, err)
	}
	meta, err := scan(rows, q, holds, matched, out)
	if err != nil {
		return nil, stock.Meta{}, s.unavailable(ctx, err)
	}
	return out, meta, nil
}

// acquireFailed is why a lookup found no connection: every one busy until
// its bound, or none to be made.
func (s *Source) acquireFailed(ctx context.Context, pool *pgxpool.Pool, err error) *stock.Unavailable {
	if ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
		if st := pool.Stat(); st.AcquiredConns() >= st.MaxConns() {
			return &stock.Unavailable{Reason: fmt.Sprintf("all %d of its connections were busy until the lookup's deadline", st.MaxConns())}
		}
	}
	return s.unavailable(ctx, err)
}

// scan reads a lookup's result into out, one Result per list: each list's
// total, then each row, appended to every list it names.
func scan(rows pgx.Rows, q *queries, holds stock.Holds, matched stock.Matched, out []stock.Result) (stock.Meta, error) {
	var (
		list                        *int32
		total                       *int64
		lists                       []int32
		release, id, location       *string
		typ, number, side, partType *string
		year, dmake, dmodel, dyear  *int32
		byModel                     *bool
		fields, asOf, vin           *string
		vehicle                     = make([]*string, q.nv)
		meta                        stock.Meta
	)
	dest := []any{&list, &total, &lists, &release, &id, &location, &typ, &number, &side, &partType}
	for i := range vehicle {
		dest = append(dest, &vehicle[i])
	}
	dest = append(dest, &year, &dmake, &dmodel, &dyear, &byModel, &fields, &asOf, &vin)
	_, err := pgx.ForEachRow(rows, dest, func() error {
		if release != nil {
			meta.Release = *release
		}
		if list != nil {
			if *list < 0 || int(*list) >= len(out) || total == nil {
				return fmt.Errorf("pg: a result row of list %d of %d", *list, len(out))
			}
			out[*list].Total = int(*total)
			return nil
		}
		if id == nil || location == nil {
			return errors.New("pg: a result row of no list with no id")
		}
		row := stock.Row{ID: *id, Location: *location, Matched: matched}
		if typ != nil && number != nil {
			k := vocab.FitmentKey{Type: *typ, Number: *number}
			if side != nil {
				k.Side, _ = vocab.ParseSide(*side) // the key asked for it, so it parses
			}
			row.Fitment = &k
		}
		if partType != nil {
			row.PartType = *partType
		}
		row.Vehicle = vehicleOf(q, vehicle)
		if year != nil {
			row.Year = new(int(*year))
		}
		if holds == stock.Yard {
			row.Matched = stock.MatchedCatalogVehicle
			if byModel != nil && *byModel {
				row.Matched = stock.MatchedVIN
			}
			if dmake != nil && dmodel != nil && dyear != nil {
				row.Decoded = &vocab.StockDecoded{Make: *dmake, Model: *dmodel, Year: int(*dyear)}
			}
		}
		// fields are a JSON object, as a file's are; null, or any other
		// value, which the column type allows, is served as {}. The text
		// is jsonb's, so an object starts with '{'.
		row.Fields = json.RawMessage("{}")
		if fields != nil && strings.HasPrefix(*fields, "{") {
			row.Fields = json.RawMessage(*fields)
		}
		if asOf != nil {
			row.AsOf = *asOf
		}
		if vin != nil {
			row.VIN = *vin
		}
		for _, l := range lists {
			if l < 0 || int(l) >= len(out) {
				return fmt.Errorf("pg: a result row of list %d of %d", l, len(out))
			}
			out[l].Rows = append(out[l].Rows, row)
		}
		return nil
	})
	return meta, err
}

// vehicleOf is a row's catalog vehicle from its vehicle columns, nil when
// it names none: a vocabulary's every field, or one of vpic's make, model
// or ext.
func vehicleOf(q *queries, cols []*string) *vocab.Vehicle {
	if q.nv == 0 {
		return nil
	}
	if q.vpic {
		var v vocab.Vehicle
		switch {
		case cols[0] != nil:
			id, err := strconv.ParseInt(*cols[0], 10, 32)
			if err != nil {
				return nil
			}
			v.Make = int32(id)
		case cols[1] != nil:
			id, err := strconv.ParseInt(*cols[1], 10, 32)
			if err != nil {
				return nil
			}
			v.Model = int32(id)
		case cols[2] != nil:
			v.Ext = *cols[2]
		default:
			return nil
		}
		return &v
	}
	entry := make(map[string]string, len(q.fields))
	for i, f := range q.fields {
		if cols[i] == nil {
			return nil
		}
		entry[f] = *cols[i]
	}
	return &vocab.Vehicle{Entry: entry}
}
