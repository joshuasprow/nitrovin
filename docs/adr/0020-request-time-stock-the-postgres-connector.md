# Request-time stock: a Postgres connector declared as a stock file

A stock source (ADR 0019) can be read from Postgres at request time, so a yard's
rows and photos are as fresh as its database. This is decided on
[Request-time stock](https://github.com/joshuasprow/nitrovin/issues/227), on
[the fitment and stock map](https://github.com/joshuasprow/nitrovin/issues/221).
It amends ADR 0011 ("files load once… and never change while nitrovin runs") for
stock only, and the glossary's opening line. Fitment still loads from files
(ADR 0017), and its reader leaves room for a connector without one being built.

## Declaring a connector

A connector is a `kind: "stock"` file (ADR 0019) with a header and no lines:

```json
{"format":1,"kind":"stock","name":"shelf","version":"1","source":"shelf","catalog":"parts-catalog",
 "connector":{"postgres":{"relation":"stock.shelf_parts","release_relation":"stock.catalog_release",
   "dsn_env":"NITROVIN_STOCK_SHELF_DSN","statement_timeout_ms":1000,"max_connections":4}}}
```

- **Loading.** It loads through `--vocab` like every file, in the one namespace,
  with no new flag. Two sources with one name are a fault (ADR 0011).
- **Credentials.** The DSN comes only from the environment variable the header
  names, so the file carries no secret. Neither `/v1/info` nor any answer shows a
  DSN, host or user. Both show the source's name, `connector: "postgres"`, the
  relation and its status.
- **Freshness.** A connector's rows change while nitrovin runs. That is the
  amendment to ADR 0011, and only stock may do it.

## The relation's shape

nitrovin dictates column names, never SQL. It runs fixed, parameterised `SELECT`s
against the named relation, which is a table or view of the source's choosing:

| Source | Columns |
| --- | --- |
| `shelf` | `id`, `location`, `type`, `number`, `side` (all three null on a row with no number), `part_type` (a spine id, for rows with no number), `vehicle_<field>` for each field of the catalog's vehicle vocabulary, `year`, `fields` (jsonb), `as_of` (optional) |
| `yard` | `id`, `location`, `decoded_make`, `decoded_model`, `decoded_year`, `vehicle_<field>`, `year`, `fields` (jsonb), `as_of` (optional) |
| release | one row, column `release` |

The connector never decodes: a yard relation carries its decodes, which the
source refreshes itself, for example by calling nitrovin's batch decode.

**Lookups and indexes.** These are the lookups, and the indexes the documentation
says a source should have:

- shelf, by fitment keys: `(type, number, side)`;
- shelf, rows with no number: `(part_type, vehicle_<fields>, year)`;
- yard, by vPIC model: `(decoded_model, decoded_year)`;
- yard, by catalog vehicle: `(vehicle_<fields>, year)`.

All the stock lists of one answer are fetched together: at most one query per
source and list kind, each bounded by the page.

**Through a barrier (amended, [#306](https://github.com/joshuasprow/nitrovin/issues/306)).**
Each key reaches the relation as a restriction on the relation's own columns, with
the key's values as parameters: every condition on the relation sits in a subquery
`LATERAL` to the key, fenced with `OFFSET 0` so that the planner cannot pull it up.
A lookup is one index probe per key, and a `security_barrier` relation keeps the
indexes above, as a table or a plain view does. Where no index can serve a key, the
lookup reads the relation once per key, where a join read it once per lookup.

- **Why not a join.** As first built, a lookup joined the relation to its keys.
  Postgres never flattens a `security_barrier` view, and pushes no join condition
  into a subquery it did not flatten, so no key reached the view's tables and every
  lookup computed the whole view. The first user measured it on its shelf view of
  153,119 rows, for three fitment keys, first run then warm: 25 ms and 1.0 ms as a
  plain view, 55 s and 22 s behind a barrier, against the 1 s statement timeout. The
  documentation's own advice for the connection's role, a `security_barrier` view,
  would have made its shelf lists `unavailable`.
- **The fence is required.** Without it Postgres pulls a simple lateral subquery up,
  and its conditions are join conditions again.
- **A relation column goes only to leakproof functions.** A barrier passes a
  restriction to its tables only when every function in it that takes a relation
  column is leakproof. A function of the request's values alone needs no mark: the
  location filter's `cardinality` is not leakproof, and passes. The lookups give a
  column to text and integer equality, `IS NULL`, `IS NOT DISTINCT FROM`, `= ANY` on
  text, and integer `>=` and `<=`. A new condition has to keep to leakproof
  functions of a column too, or it stays above the barrier and the lookup scans. The
  casts that build a result's columns are not conditions.
- **A `character` equality key column is held to the key as `character` first.**
  Postgres compares a `character` column with a text key as text, through a cast
  that is not leakproof and that no index on the column serves, so on a plain
  relation too the fenced lookup scanned once per key. For the columns a lookup
  holds to a key by equality (`type`, `number`, `part_type`, the vehicle columns)
  it therefore adds the `character` comparison, which is leakproof and indexed,
  beside the text one. A `character` `side` or `location` keeps the text comparison
  alone, above a barrier: neither makes the probe. The added comparison changes no
  answer: the text comparison is true only of a key equal byte for byte to the
  value without its padding, so of a key ending in no space, and of those the
  `character` comparison, which ignores trailing spaces on both sides, is true too.
  That holds under a deterministic collation only (under one where a no-break space
  equals a space, a value ending in the first equals as text a key ending in the
  second, and not as `character`), so the column check reads each column's
  collation and a `character` column of a nondeterministic one keeps the text
  comparison alone.
- **The price is a scan per key where no index serves it.** The planner cannot turn
  a fenced subquery into one hash join, so a relation with no index on a key column
  is read once per key: an unindexed table, a key column the view computes with no
  matching expression index, a key column that is an aggregate's result, a view
  with `LIMIT`, `OFFSET` or a window function not partitioned by the key, a
  `character` key column of a nondeterministic collation. On a plain table of
  150,000 rows with no index (Postgres 17, shelf by fitment, warm): 8.0 ms, 8.1 ms
  and 8.2 ms as a join for 1, 10 and 40 keys; 8.9 ms, 80 ms and 375 ms as built. The
  documentation says which relations these are.
- **The plan is tested, not the time.** Fixture-sized tables never show the cost.
  `pkg/stock/pg` holds each of the four lookups to its index, with an index
  condition on the key's columns, through the worked view's barrier, the fitment
  lookup through a plain view as well, and both shelf lookups over `character`
  columns, on a table and behind a barrier.
- **Measured when built**, on a synthetic shelf of 150,000 rows behind the worked
  view's barrier (Postgres 17, three fitment keys of one number, first run then
  warm): 9.1 ms and 8.7 ms as a join, by a parallel scan of the table; 0.13 ms and
  0.06 ms as built, which is what the plain view takes either way. Behind a barrier
  view that works per row (two tables in a `UNION ALL`, two joins, an anti-join and
  an aggregate per row): 298 ms and 282 ms as a join, 0.29 ms and 0.14 ms as built.
  With `character` key columns, indexed, and 40 keys (warm): 14 ms through a plain
  view and 16 ms behind a barrier as a join, which no index served either; 677 ms
  and 960 ms fenced with the text comparison alone; 0.33 ms through both as built.

**At first connection** nitrovin checks the relation's columns against
`information_schema`. A missing or mistyped column makes the source `unavailable`,
with the reason. Nothing guesses.

## Reading and failure

- Every query runs in a `READ ONLY` transaction.
- Defaults, each settable in the header: `statement_timeout` 1 s,
  `max_connections` 4, connect timeout 2 s.
- **A database down at start does not stop nitrovin.** The source answers
  `unavailable` and retries at most every 5 s, so a dead database does not pile up
  connections.
- **A slow or failed query fails only its stock list.** The list answers
  `unavailable` with a reason (ADR 0019), and the rest of the answer stands.
- **Nothing is cached.** A request-time source is read at request time.

## Files and connectors together

Sources are named and independent, with no precedence. A list holds the rows of
every loaded source of its kind, each row tagged with its source, and the answer
carries each source's block.

## Surfaces, driver and licence

The connector works wherever a stock file does in the Go binary: `serve`, stdio,
the CLI and MCP. The C library and the browser (ADR 0013) read stock files only,
so neither carries a database driver.

The driver is `pgx/v5` (MIT). Through `pgxpool` it adds about 0.5 MB to a 75 MB
binary and 16 packages. It lives in its own package, `pkg/stock/pg`, which
`pkg/vocab` never imports, so the WebAssembly build is unchanged.

## Testing

CI runs the connector against a Postgres service, as the oracle tests already do:

- the synthetic shelf and yard rows (ADR 0019's fixtures) are loaded into views;
- the connector's answers must equal the same rows loaded as files, with the same
  candidates in the same order;
- an unreachable database must answer `unavailable`;
- a missing column must be reported.

## What "offline" still promises

The glossary's opening line becomes: "A decoder that turns a VIN into the vehicle
NHTSA's vPIC says it is, shipped as one binary with its data inside. It translates
names and answers fitment from files it loads, and reads stock from a database
only when a stock source says to."

Decode, translate and fitment never touch the network. Only a configured
connector does.

## Considered options

- **Flags per connector** (`--stock name=…`): a second way to name sources, beside
  files.
- **The source's own SQL per lookup:** nitrovin would run a caller's code, which
  the vocabulary map ruled out (plugins are data, not code).
- **Refusing to start without the database:** it turns a stock outage into a
  decode outage.
- **Caching answers:** stale stock, which is what a request-time source exists to
  avoid.
- **A driver in the C library:** the library's size and its dependency-free build
  for a feature only a server uses.
- **Precedence between a file and a connector:** names already keep sources apart.

## Consequences

- [#243](https://github.com/joshuasprow/nitrovin/issues/243) builds stock on one
  interface. A connector ticket adds `pkg/stock/pg` behind it.
- The documentation lists the columns and indexes, and gives a worked view over a
  synthetic schema.
- Measured when [#245](https://github.com/joshuasprow/nitrovin/issues/245) built
  it: `pgxpool` adds 4.3 MB to the 75.4 MB release binary (`-trimpath
  -ldflags='-s -w'`), not 0.5 MB, most of it pgx's type codecs and the
  `golang.org/x/text` tables its SCRAM password normalisation links. The C
  library and the WebAssembly build are unchanged.

## Amendment (2026-11-20): a batch's fitment keys share their lookups

[#354](https://github.com/joshuasprow/nitrovin/issues/354): "all the stock lists
of one answer are fetched together" left a batch of fitment keys one lookup per
key, in turn. A part's candidates are many fitments of the same vehicles, so a
caller paging each candidate's yard asks for those vehicles' rows once per key.
The first user's dev warehouse, a pickup's side mirror: 44 fitment keys, 8,467 yard
rows, through its `security_barrier` yard view of about 11,800 vehicles. Each key's
query took 0.1 to 0.3 s, mostly planning the view and reading its rows, and the
batch took 6 to 9 s on the server, past a 5 s caller's wait and, under load, past
the 1 s statement timeout of single keys.

- **A batch's fitment keys share one lookup per source, kind and page.** Fit
  answers each request of a batch, then looks up every fitment key's stock lists at
  once: those that page from their first row with the same `limit` and `location`
  go in one call, each list its own `Result`. A later page stays a call of its own:
  a call pages several lists only from their first row, as before. Answers are
  unchanged byte for byte. A lookup that fails now leaves every list of its call
  unavailable, a whole batch's where it left one key's.
- **The yard lookup reads each key once.** Each vPIC model and catalog vehicle the
  lists name is read once, in the union of their years for it, and each list keeps
  the rows in its own years, by a join on the rows read, outside the fence, where
  no barrier can hold it back.
- **Each row is sent once.** A lookup answers each list's total in a row of its
  own, then each row in any list's page once, with the lists whose pages hold it,
  in `id` order. The shelf lookups use the same shape.
- **Measured** on the same warehouse, under its other load, three rounds each,
  `EXPLAIN (ANALYZE, TIMING OFF)` as a superuser rather than the connector's role:
  the 44 lookups as built, 6.1, 7.2 and 7.6 s summed, 384,160 shared buffers; the
  one lookup, 0.52, 0.69 and 0.89 s, 38,505 buffers and 813 result rows for
  8,467. A `POST /v1/fit` of the 44 keys, from outside the warehouse's network:
  8.5 to 13.4 s before, some lists `unavailable`; 0.37 to 0.91 s after.
- **What is left is the view's rows.** The one lookup reads 769 vehicles, each
  through the view's joins; the view, not the lookup, sets that price.
