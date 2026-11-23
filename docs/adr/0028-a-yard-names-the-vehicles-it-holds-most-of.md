# A yard names the vehicles it holds most of

`stocked` is a request beside decode, translate, list and fit: the vehicles the
loaded yard sources hold most of, each with a VIN of the yard's
([#384](https://github.com/joshuasprow/nitrovin/issues/384)). It amends ADR 0019,
whose sources answer three lookups, each from a vehicle or a fitment, and ADR 0020,
whose yard relation has no VIN.

## Why

Every stock lookup starts from something a caller already has. Nothing answers
"which vehicles are in the yard?", so a person, or an assistant over MCP, who
wants a vehicle with stock has to know a VIN that has some. The page's VINs to
try are fixed examples, most with no stock behind them.

## The rule

**The request** has two optional fields: `location`, the locations to keep yard
rows at, and `limit`, 1 to 100, 10 when absent. It has no `from`: it asks about
the yards, not about a vehicle.

**The answer** lists vehicles, the most yard rows first, then the lower model
id. Each has `vin`, a VIN of the yard's; `spine`, its vPIC make and model as the
data month names them; `model_year`; `yard`, how many of the source's rows are
of that model and year; and `source`. `sources` holds every loaded yard's block,
as a stock list's does.

**One vehicle a model.** A source answers each vPIC model once, in the model
year it has most rows of, the later of two alike. On a real yard of about 12,000
vehicles, seven of the eight best-stocked model-and-year pairs were one pickup in
neighbouring years: a list of them is one useful entry. A model two yards hold is
kept from the yard with more of it; the counts are not added, as the two may be
of different years.

**Yard rows only.** A shelf row is a part: it has no VIN to offer a vehicle by.
`yard` counts the rows of one model and year, so a fit's yard list, which spans
the years a part fits, can hold more.

**A source says whether it can.** `stock.Stocker` is an interface beside
`stock.Source`, not a method of it: `Source` is public API, and a method added
to it breaks whoever implements it. A yard file is one: its rows have VINs. The
Postgres connector is one when its relation has a `vin` column, a new optional
text column. With it, the yard's rows say their VIN, as a file's do, and the
yard names its vehicles. Without it nothing changes, and the yard names none. A
yard that cannot name vehicles is `available` in `sources` and gives none.

**Nothing to name is not a refusal.** No yard loaded, or none that can name a
vehicle, answers an empty `vehicles`. A yard that failed is `unavailable` with
its reason, and the others' vehicles stand.

**The connector's query** is one `SELECT`: the relation's decoded rows grouped
by model and year, each model's best year kept, ordered and cut. It is an
aggregate of the whole relation, so it reads all of it once; no index makes it
a probe. It runs under the statement timeout like every lookup.

## Considered options

- **A new form of `fit`.** Every fit starts from a vehicle or a fitment and
  answers part types, candidates or vehicles that fit. A fit from nothing, with
  an answer that holds none of those, is another request under fit's name.
- **A listing of `list`.** `list` reads the loaded files and takes no context;
  stock is read through the fitter, whose context bounds a live source's lookup.
- **A feature of the page alone.** Each surface answers through the same
  Translator; the Go method the page would call is public API already.
- **Model and year, not model.** It is what the yard holds, and what the count
  is of. It fills the answer with one vehicle's years.
- **Shelf counts in the score.** A shelf reaches a vehicle only through the
  catalog's fitments, a keyed lookup a part type. On a deployment's shelf view
  of about 155,000 rows an unkeyed count took 19 s. Yard rows alone say where
  the stock is.
- **A required `vin` column.** Every deployed yard relation would be
  unavailable until its view gained one.

## Consequences

- A fifth op on every surface: `GET /v1/stocked`, `nitrovin stocked`, the MCP
  tool `stocked`, `"op": "stocked"` on stdio and in the C library, and
  `Translator.Stocked` in Go.
- A yard relation that already has a column named `vin` must have it as text,
  or the source is unavailable, naming the column: an optional column is still
  checked when it is there.
- Measured through a `security_barrier` view: 50 ms on 200,000 synthetic rows
  (`TestStockedPostgres`), and about 200 ms on a real yard of about 12,000
  vehicles, whose view already had the column.
- The answer is read from the yard at request time, as every stock answer is.
  A caller that shows it on every page keeps it itself.
