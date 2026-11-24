# What stock says: shelf parts and yard vehicles, from named sources

nitrovin answers stock: the parts on a shelf and the whole vehicles in a
self-service yard that fit what a caller asked about. Decided on
[What stock says](https://github.com/joshuasprow/nitrovin/issues/226), on
[the fitment and stock map](https://github.com/joshuasprow/nitrovin/issues/221).
Stock attaches to `fit` (ADR 0018). A stock source is a file (below) or a
Postgres connector ([#227](https://github.com/joshuasprow/nitrovin/issues/227)).

## Stock sources

A **stock source** is named, in the one namespace of vocabularies, extension sets,
spines and catalogs. It has one of two kinds:

- **`shelf`**: parts. It names the one catalog its numbers belong to, and
  optionally the catalog `release` its numbers are under (for a connector, where to
  read it).
- **`yard`**: whole vehicles. It names no catalog, because a yard vehicle carries a
  VIN and no number.

Any number of sources may be loaded. A row belongs to exactly one of them.
Availability is the source's rule: a source supplies only what it would offer,
and nitrovin passes any status through without interpreting it.

## A shelf row

nitrovin interprets these fields:

- **`id`**: the source's own stock id.
- **`location`**: an opaque string that a request can filter on.
- One of two ways to name the part:
  - the **fitment key**: `type`, `number`, `side`;
  - for a row with no number, the **`part`** (a part-spine or part-vocabulary entry)
    plus the catalog **`vehicle`** and **`year`**.

  In one parts system, 12% of shelf rows have no number: part types the catalog does
  not cover.

Everything else goes in **`fields`**: a flat object of strings, numbers and null,
passed through verbatim and never interpreted. Stock number, condition, mileage,
price, status, comments and dates are all fields.

A row with a number joins its fitment, marked `matched: "fitment"`. A row whose
number is a retired number (ADR 0016) joins the successor, marked
`reached: "retired"`.

A row with no number is matched only by part type and the exact vehicle and year,
marked `matched: "vehicle"`.

A number the catalog lacks entirely makes an **orphan** row. It never appears under
a candidate. `/v1/info` and `vocab check` count orphans, with examples, as drift. One
user's count has been exactly 0 in every copy it kept, and it uses that count as an
alarm.

## A yard row

nitrovin interprets **`id`**, **`location`**, and the first of these routes to the
spine that is present:

1. **`decoded`**: a vPIC make, model and year that the source precomputed, which a
   connector can read from a column;
2. **`vin`**: decoded by nitrovin. This applies to file sources only, and the decode
   is cached for the life of the process;
3. **`vehicle`** plus **`year`**: an entry of the catalog's vehicle vocabulary, the
   fallback.

The answer says which route matched: `matched: "vin"` or `matched: "catalog-vehicle"`.
In one yard system, every live vehicle carries both a VIN and the catalog's own model,
and the two agree about 95% of the time.

Everything else goes in **`fields`**: any JSON value, so a list of photos passes
through. Stock number, colour, row, date added and photos are all fields.

## In answers

`fit` gains `stock` (`["shelf"]`, `["yard"]`, or both) and an optional `location`
filter (one or more):

- **Each candidate** carries two stock lists:
  - `shelf`: the rows of that fitment;
  - `yard`: the yard vehicles whose vehicle and year are among the fitment's
    applications.

  Each list carries its first 10 rows and a `total`. A `fit` from that fitment's key
  with the same `stock` pages the whole list.
- **The answer** also carries `shelf_by_vehicle`: the rows with no number for the
  part type and the exact vehicle and year.
- **Related vehicles** each carry their count of yard vehicles.

**Every stock list carries its source:**

- name and kind;
- **`kind`**: `file` or `live`;
- **`as_of`**: a file's own time, or a live source's load time when the source
  exposes one;
- for a shelf source, its catalog **`release`**, and **`catalog_mismatch: true`**
  when that differs from the loaded catalog's release.

A source that is down answers status `unavailable` with a reason and an empty list.
The rest of the answer stands.

## The stock file

A file of `kind: "stock"`, loaded by `--vocab`, is for callers with no database:

- **The header** carries `name`, `version`, `source` (`shelf` or `yard`), `as_of`,
  and for a shelf source `catalog` and `release`.
- **The lines** are one row each, as above.
- **A new file means a restart**, as ADR 0011 rules for every file.

## What stock never carries

`fields` are served verbatim to every caller of the deployment. So the contract
says a source supplies no personal data: no names, contact details or identity
numbers of owners, sellers, customers or staff. It also supplies nothing it would
not show a customer, such as cost.

nitrovin cannot tell what a field holds. The source chooses its columns: a view, or
the file's lines. `vocab check` reports field names that look personal (`email`,
`phone`, `name`, `address`, `license`) as drift. That is a nudge, not a guarantee.

## Considered options

- **One unnamed stock set.** A deployment with shelf and yard stock, or two parts
  systems, could not tell them apart.
- **A fixed, typed shelf schema.** nitrovin would interpret condition and price
  codes that differ between every system.
- **Always decoding a yard VIN in nitrovin.** A connector would decode thousands of
  rows on every request. A precomputed decode is a column read.
- **A separate `stock` request.** It costs a second round trip for the answer most
  callers want with their candidates.
- **Matching an orphan row by its vehicle.** It hides the row's real problem, a
  number the catalog lacks, which its count exists to raise.
- **No field-name check.** It gives up a cheap warning on the rule nitrovin cannot
  enforce.

## Consequences

- [The connector](https://github.com/joshuasprow/nitrovin/issues/227) supplies rows
  of exactly these shapes from a Postgres relation, and decides how columns map to
  them.
- [#241](https://github.com/joshuasprow/nitrovin/issues/241) builds `fit`. A stock
  ticket adds `stock`, `location` and the stock file to it.

## Amendment (2026-11-17): a shelf row's side

[#292](https://github.com/joshuasprow/nitrovin/issues/292) states what the shelf
row above left implicit:

- **A numbered row joins the fitment of exactly its key, side included.** A row
  with no side joins only the fitment with no side, never the left or right one of
  its number: the connector matches `side IS NOT DISTINCT FROM` the key's side
  (ADR 0020), and a stock file looks the row up by its whole key. `fit`'s side
  filter (ADR 0018) chooses among fitments, not among rows. A system that leaves
  the side off a part the catalog lists by side marks the side itself, in its view
  or its lines: the first user's shelf has 3,917 (code, number) pairs on two
  sides, and its view marks them.
- **A row with no number ignores `type` and `side`.** It is matched by part type,
  vehicle and year alone. A connector does not read those columns where `number`
  is null; a stock file, whose row names its part one way or the other, refuses a
  row that gives either without a number.

## Amendment (2026-11-20): a request names the row fields it reads

[#371](https://github.com/joshuasprow/nitrovin/issues/371): a caller that
lists the yard vehicles a part fits reads each row's identity and one or two of
its `fields`, such as the VIN, and gets every row whole: photos, row, colour,
stock number, dates. Measured on the first user's dev warehouse, a pickup's side
mirror (44 fitment keys in one batch): the answer was 3.86 MB, 3.44 MB of it
yard rows (8,467 rows for 769 vehicles); cut to their identity and `vin`, 1.03
MB.

- **`stock_fields`, a fit request's property.** An array of field names; every
  stock row of the answer keeps only those of its `fields`, in the source's
  order, values verbatim. Absent keeps every field, so an answer without it is
  byte for byte what it was. `[]` keeps none. In `GET /v1/fit` it repeats as
  `stock` does, and `stock_fields=` alone is `[]`. Only with `stock`, as
  `location`.
- **Only `fields`.** A row keeps its `source`, `id`, `location`, `matched`, what
  it names (a shelf row's `fitment` and `reached`, or its `part`, `vehicle` and
  `year`; a yard row's `vin`, `decoded`, or catalog `vehicle` and `year`) and
  `as_of`: what tells rows apart, pages them and says how each matched. Every
  list keeps its `total`, paging and `sources`.
- **A name no row has is no error.** The source decides its fields, and two
  sources of one kind may name theirs differently, so a name is a filter, not a
  claim about the source.
- **Every list of the answer.** A candidate's `shelf` and `yard`,
  `shelf_by_vehicle`, a fitment key's lists and a related vehicle's `yard`, so
  one property covers the answer.
- **Applied in the contract, once every list is whole**, where a batch's
  answers are assembled (`contract.Fitter`'s `Fit` and `FitBatch`), so HTTP,
  stdio, the C library, MCP and the browser's stock all answer it alike; no
  surface post-processes an answer. A source still reads whole rows: the cut is
  of what is served, not of what is read, and ADR 0020's lookups are unchanged.
- **Not a projection in the lookup.** A connector's `fields` column is the
  source's own choice of what it shows (above); cutting it in SQL would give each
  cut a query of its own and split ADR 0020's one lookup a batch.

## Amendment (2026-11-23): a header declares its rows' field order

[#390](https://github.com/joshuasprow/nitrovin/issues/390): a row's `fields`
are answered in the source's order, and only a stock file's author chooses
that order, by how each line is written. A connector's author cannot. Its
`fields` column is read as `jsonb` (ADR 0020), which stores an object's
keys by length and then by bytes, so a row's fields arrive as `tag`,
`kind`, `miles`, `status`, whatever their author meant to lead with.
Measured on a shelf whose rows carry 11 fields: the first five a caller
showed were the five shortest names, and the stock number, the condition
and the price came last.

- **`fields`, a stock header's property**, on a stock file's header and on
  a connector's: an array of distinct, non-empty field names. Anything
  else there is a fault at `/fields`, and `[]` declares no order.
- **A row's `fields` are answered in that order:** the names declared, in
  the header's order, where the row has them; then the row's other fields,
  in the source's order as before.
- **It orders and never cuts.** A name no row has is no fault and no
  drift, as two rows of one source need not share their fields.
  `stock_fields` still cuts, and what it keeps is in the declared order.
- **Absent, nothing changes:** the source's order, byte for byte.
- **A value is still verbatim.** A member moves as the bytes its source
  wrote: a number's digits, a string's escapes and nested JSON are never
  decoded and written again. Only an object's own keys are ordered: what a
  yard row nests under one stays as written. A key a row has twice keeps
  both members, the earlier first, as before.
- **In the source, once.** A stock file's rows are put in order when the
  file loads; a connector's as each is read. The order is the source's own,
  as its rows' is, so `contract.Fitter` and every surface read rows already
  in it: `stock_fields`' cut keeps a row's own order and is unchanged, and
  no surface post-processes an answer.
- **What it costs.** One pass over a row's members, and no allocation
  beyond the copy every row is already made with. A row of 11 fields as
  `jsonb`'s text takes 35 ns to copy and 295 ns to order
  (`BenchmarkAppendStockFields`, an M5 Pro), one allocation either way. One
  lookup of 20,000 such rows under 200 fitment keys, from a local Postgres
  17: 59.0 ms with no order declared and 60.2 ms with one, 282,554
  allocations either way. The same lookup took 59.4 ms and 322,555
  allocations before: a row's fields are now copied once from the
  driver's buffer, in order, where they were copied twice.
- **`/v1/info` reports it:** a source's declared `fields` beside its
  `source` and `catalog`, absent when it declares none; a browser's own
  `/v1/info` says the same of a stock file it holds. `list` names
  vocabularies, extension sets and spines and no stock source, so it has
  nothing to report it beside.
- **Not the column's type.** A `json` column keeps the order its text was
  written in, but the connector reads `fields` as `jsonb`'s text whatever
  the column is, and a view that builds its object, or reads a `jsonb`
  column a system already has, has no order to keep.
- **Not the request's.** What a row leads with is its source's to say, the
  same for every caller; a request already names the fields it reads.
- **Additive to `/v1`:** a header without `fields`, and an answer from
  one, are unchanged. No request or answer gains a property but
  `/v1/info`'s.
