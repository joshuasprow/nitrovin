# Vocabularies

A **vocabulary** is one system's names for vehicles or part types: a
yard's inventory, a parts catalog's codes, a marketplace's categories.
nitrovin loads vocabularies as data, beside the vPIC data month, and
**translates** between them through a **spine**: vPIC's own makes and
models for vehicles, a spine you declare for part types. A VIN translates
too, through its decode. A **catalog** is a fourth kind of file, loaded
the same way: which parts fit which vehicles. This page is how to write
the files, load and check them, and translate and list on every surface.
The rules are the
[API contract](api/README.md#vocabularies---vocab); the file format is
[`vocabulary.schema.json`](api/vocabulary.schema.json); the words are
[GLOSSARY.md](../GLOSSARY.md#translation)'s; the reasons are ADRs
[0009](adr/0009-what-a-vocabulary-mapping-says.md),
[0010](adr/0010-part-types-translate-through-a-declared-spine.md),
[0011](adr/0011-vocabularies-are-json-lines-loaded-once.md),
[0012](adr/0012-translation-and-listing-are-requests-beside-decode.md),
and for catalogs [0016](adr/0016-what-a-fitment-says.md) and
[0017](adr/0017-the-catalog-file.md).

Matching is exact. nitrovin compares names after one normalization,
trimming and collapsing white space and folding case (stated exactly under
**A vehicle vocabulary**), and nothing else: no fuzzy matching, no rules.
Deciding that your `Chevy Silverado` is vPIC's Chevrolet Silverado is the
vocabulary author's work, done before the file is written.

The examples run from a checkout of this repository, with `nitrovin` built
from it on `PATH`, against the synthetic vocabularies in
[`testdata/vocab/`](../testdata/vocab) (a fleet, a parts listing, its part
spine, an extension set of classics), the synthetic catalog in
[`testdata/vocab/fitment/`](../testdata/vocab/fitment), its synthetic
shelf and yard in [`testdata/vocab/stock/`](../testdata/vocab/stock) and
EPA's published vocabulary:

```sh
go build -o nitrovin ./cmd/nitrovin && export PATH="$PWD:$PATH"
```

## The five kinds

Every file is JSON Lines: a header line naming its `kind`, then one line
per entry. A file is one of five kinds:

| Kind | What it holds | Lines after the header | Fixture |
| --- | --- | --- | --- |
| `vocabulary` | A system's names (`entry`), each mapped to spine entries (`mappings`). `subject` `vehicle` maps onto vPIC; `subject` `part` onto the declared spine its header names | `{"entry": {…}, "mappings": […]}` | `fleet.jsonl` (vehicle), `listing.jsonl` (part) |
| `extension-set` | Vehicles vPIC lacks, minted so vocabularies can map to them: a make, or a model under a vPIC make or an extension make | `{"id", "make_name"}` or `{"id", "make", "name"}` | `classics.jsonl` |
| `spine` | A declared part-type spine: every id, flat, names optional | `{"id", "name"}` | `parts.jsonl` |
| `catalog` | Which parts fit which vehicles: fitments with their applications, redirects and retired numbers ([below](#a-catalog)) | `{"fitment", "applications"}`, `{"redirect"}` or `{"retired", "to"}` | `fitment/catalog.jsonl` |
| `stock` | One stock source: shelf parts or whole yard vehicles ([below](#stock)) | `{"id", "location", …}`, one row each | `stock/shelf-stock.jsonl`, `stock/yard-stock.jsonl` |

Names (`name` in the header) share one namespace across all five, and
`vpic` is reserved for the vehicle spine. Every object is closed: a field
the schema does not name refuses the file, so a typo never passes
silently.

## A vehicle vocabulary

[`testdata/vocab/fleet.jsonl`](../testdata/vocab/fleet.jsonl) begins:

```json
{"format":1,"kind":"vocabulary","name":"fleet","version":"2026.11","data_month":"2026_10","subject":"vehicle","fields":["make","model"]}
{"entry":{"make":"Dodge","model":"Ram 1500"},"mappings":[{"model":1938,"last":2012,"qualifiers":{"34":["1500","DS (1500)"]},"origin":"curated"},{"model":13620,"first":2013,"origin":"curated","note":"Ram is its own make from 2013"}]}
{"entry":{"make":"Dodge","model":"Other"},"mappings":[{"make":476}],"note":"any Dodge"}
{"entry":{"make":"Studebaker","model":"Lark"},"mappings":[],"note":"listed, not yet placed"}
```

- **The header.** `format` is 1. `version` is yours, free text; every
  answer carries it beside the name. `data_month` is the vPIC month you
  built against, used only to explain drift. `fields` are the fields that
  name an entry, in order: a make and a model here, one code in a catalog.
  A model year is never a field.
- **An entry** has exactly the header's fields, and is matched after one
  normalization (ADR 0009). Two entries equal after it refuse the file. A
  re-implementation, in SQL say, agrees with nitrovin only if it follows
  it exactly:
  - **White space** is trimmed from both ends, and every run of it inside
    becomes one space (U+0020). White space is Go's `unicode.IsSpace`:
    tab, line feed, vertical tab, form feed, carriage return and space
    (U+0009–U+000D, U+0020), U+0085 (NEL), U+00A0 (NBSP), and Unicode's
    category Z (U+1680, U+2000–U+200A, U+2028, U+2029, U+202F, U+205F,
    U+3000). Nothing else is: U+FEFF and the zero-width characters stay.
  - **Case** is folded by Go's `strings.ToLower`: each character's simple
    lower-case mapping, one character to one, with no locale and no
    special casing (`İ` is `i`, a final `Σ` is `σ`). A case-exact
    vocabulary skips it (below).
  - **Nothing else**: no Unicode normalization form, so `é` as one
    character and as `e` with a combining accent are two spellings.
  - **The Unicode version** is the Go release's that built nitrovin,
    `unicode.Version`: 17.0.0 for go1.27, the release `go.mod` names.
- **`"case": "exact"`**, in the header, keeps case
  ([ADR 0026](adr/0026-case-exact-vocabularies.md)), for a system whose
  codes differ only in case and name different things: entries still match
  after trimming and collapsing white space, but `old3` and `OLD3` are two
  entries, not a duplicate, and a spelling the vocabulary does not hold is
  unknown, never a guess between two that fold alike. It covers every
  field, and everything that names the vocabulary's entries follows it:
  translate and list requests, a catalog whose `vehicles` or `filed_under`
  vocabulary it is (its applications', redirects' and fitments' entries,
  which a catalog never declares itself), and stock rows. Any other value
  is a fault; a release before it refuses the field as unknown.
  [`testdata/vocab/case`](../testdata/vocab/case) has a case-exact vehicle
  vocabulary (`OLD3`, `old3`) and part vocabulary (`MIR`, `mir`) with a
  catalog and shelf naming both.
- **A mapping** names exactly one spine entry: `{"make": <vPIC make id>}`,
  `{"model": <vPIC model id>}` (it implies its make) or
  `{"ext": "ext:<set>/<id>"}`. An entry has as many mappings as it
  resolves to; `"mappings": []` lists an entry you have not placed yet,
  which translates as `unmapped` rather than `unknown`.
- **`first` and `last`** bound the model years a mapping holds for,
  inclusive; either may be absent.
- **`qualifiers`** narrow a mapping in vPIC's own terms: an element id and
  the values of it the mapping holds for, attribute ids for lookup
  elements (Drive Type `15`: `[2]`), exact text for text elements (Series
  `34`: `["DS (1500)"]`). Only a decode can satisfy one, so they narrow a
  translation from a VIN and never one from a name.
- **`origin`** (`derived`, `curated` or `seed`) and **`note`** are passed
  through to answers, never interpreted.

The ids and values come from the data. A decode in the `nitrovin` shape
carries every element id and attribute id:

```sh
nitrovin decode --shape nitrovin 1D7RV1GT5AS123456 | jq -c '.attributes | {Make, Model, Series, "Drive Type"}'
```

```json
{"Make":{"element_id":26,"value":"DODGE","attribute_id":476},"Model":{"element_id":28,"value":"Ram","attribute_id":1938},"Series":{"element_id":34,"value":"DS (1500)"},"Drive Type":{"element_id":15,"value":"4WD/4-Wheel Drive/4x4","attribute_id":2}}
```

and `list` pages through vPIC's makes, or one make's models:

```sh
nitrovin list vpic --limit 3
nitrovin list vpic --where make=476 --limit 3 | jq -c '.entries[]'
```

```json
{"model":14766,"name":"400","under":{"make":476}}
{"model":14416,"name":"600","under":{"make":476}}
{"model":14415,"name":"ARIES","under":{"make":476}}
```

A qualifier's values come from its model's own: `list` with a model and an
element gives every value a VIN of that model can decode the element to
([ADR 0015](adr/0015-model-values-for-an-element.md)), not every value of
the VIN schemas it shares with other models:

```sh
nitrovin list vpic --where model=1938 --where element=34 --limit 4 | jq -c '.entries[]'
```

```json
{"value":"1500"}
{"value":"2500"}
{"value":"3500"}
{"value":"3500W"}
```

### Extension sets

vPIC lacks vehicles from before 1981 and some marques. An extension set
mints the missing makes and models once, so that every vocabulary mapping
to them meets on the same spine entry.
[`testdata/vocab/classics.jsonl`](../testdata/vocab/classics.jsonl):

```json
{"format":1,"kind":"extension-set","name":"classics","version":"2026.11","data_month":"2026_10"}
{"id":"packard","make_name":"Packard"}
{"id":"packard-clipper","make":"ext:classics/packard","name":"Clipper"}
{"id":"corvair","make":467,"name":"Corvair"}
```

A vocabulary maps to one as `{"ext": "ext:classics/packard-clipper"}`.
No decode lands on an extension, so an extension mapping takes no
qualifiers and is reached from names only. When vPIC gains the vehicle,
map to its id instead; until then `vocab check` warns if an extension is
named like a vPIC make, or like a vPIC model under the same make.

An extension's `id` holds any characters but `/` and white space as the
normalization above has it (`unicode.IsSpace`): an extension line whose
id holds one is refused, its message listing them, and an extension
reference whose id does is malformed. The schema's pattern lists them
rather than writing `\s`, which Go's regular expressions (ASCII, without
the vertical tab) and JSON Schema's (ECMA-262: with NBSP and U+FEFF,
without NEL) read differently.

## A part vocabulary and its spine

vPIC names no parts, so part types route through a spine you declare:
every part type, flat, by id.
[`testdata/vocab/parts.jsonl`](../testdata/vocab/parts.jsonl) is one;
[`testdata/vocab/listing.jsonl`](../testdata/vocab/listing.jsonl) maps a
listing's categories onto it:

```json
{"format":1,"kind":"spine","name":"parts","version":"2026.11"}
{"id":"brake-pads","name":"Brake Pads"}
{"id":"brake-pads-motorcycle","name":"Brake Pads, Motorcycle"}
```

```json
{"format":1,"kind":"vocabulary","name":"listing","version":"2026.11","data_month":"2026_10","subject":"part","fields":["category"],"spine":"parts"}
{"entry":{"category":"Brake Pads"},"mappings":[{"id":"brake-pads","qualifiers":{"39":[2,3,7]},"origin":"curated"},{"id":"brake-pads-motorcycle","qualifiers":{"39":[1]},"origin":"curated"}]}
```

A part vocabulary names exactly one spine, which must be loaded beside
it, and a translation never crosses from one spine to another. Where a
part type depends on the vehicle, say so with qualifiers in vPIC's terms,
as above (Vehicle Type `39`); a translation given a VIN as context checks
them. A spine's named entries are also a vocabulary of their names, under
the spine's name. Add an id to the spine rather than minting an
extension.

## A catalog

A catalog is one author's statement of which parts fit which vehicles. It
names the declared spine its part types come from and the vehicle
vocabulary its vehicles are entries of, and both must be loaded beside
it. A deployment loads at most one catalog.
[`testdata/vocab/fitment/`](../testdata/vocab/fitment) holds one, with
its spine (`part-types`), its vehicle vocabulary (`catalog-models`) and
the part vocabulary its fitments are filed under (`part-codes`).
`catalog.jsonl`:

```json
{"format":1,"kind":"catalog","name":"parts-catalog","version":"2026.11","release":"7.02","data_month":"2026_10","spine":"part-types","vehicles":"catalog-models","filed_under":"part-codes"}
{"fitment":{"type":"mirror","number":"MR-1001","side":"left"},"note":"power, heated","filed_under":[{"code":"MIR"}],"applications":[{"vehicle":{"make":"Dodge","model":"Ram 1500"},"first":2009,"last":2012,"options":"w/o towing package"},{"vehicle":{"make":"Dodge","model":"Ram 1500"},"first":2009,"last":2012,"options":"4WD, w/ towing package","qualifiers":{"15":[2]}}]}
{"fitment":{"type":"mirror","number":"MR-1001","side":"right"},"note":"power, heated","filed_under":[{"code":"MIR"}],"applications":[{"vehicle":{"make":"Dodge","model":"Ram 1500"},"first":2009,"last":2012,"options":"w/o towing package"}]}
{"redirect":{"vehicle":{"make":"Chevrolet","model":"Silverado 1500"},"first":2007,"last":2013,"from":"brake-rotor-rear","to":"brake-rotor-front","options":"rear rotors are listed with the front"}}
{"retired":{"type":"brake-rotor-front","number":"BR-1999"},"to":{"type":"brake-rotor-front","number":"BR-2001"},"note":"7.01 → 7.02"}
```

- **The header.** `spine` names the declared spine. `vehicles` names a
  vehicle vocabulary, or is `vpic` when the catalog's vehicles are vPIC
  makes and models and extensions. `release` is the publisher's own
  release, separate from your `version`; it is optional, as is
  `data_month`.
- **A fitment** is one part: its key is the part type (an id of the
  spine), the catalog's `number` and an optional `side` (`left`, `right`
  or `both`). One number on two sides is two fitments, as `MR-1001` is
  above. A `note` is passed through.
- **Filed under**
  ([ADR 0024](adr/0024-the-codes-a-fitment-is-filed-under.md)). The
  header's optional `filed_under` names a part vocabulary on the catalog's
  spine, loaded beside it: a system's own part codes. A fitment line's
  `filed_under` lists the entries of it the fitment is filed under, each by
  exactly the vocabulary's fields, as an application's `vehicle` is: the
  union over its applications. They are an attribute, never part of the
  key, so a part filed under three codes is still one fitment. The entries
  are read in any order, matched after the same normalization as entries,
  and kept sorted by their fields as written, field by field in name
  order, each value in byte order (`"10"` before `"9"`, `"B"` before
  `"a"`); each is spelled as the file first writes it, as vehicles are.
  Redirects and retired numbers carry none. A catalog without
  a header `filed_under` stays valid, and no line of it may have one.
- **An application** is one vehicle and year range the fitment fits.
  `vehicle` is an entry of the vehicle vocabulary by exactly its fields,
  matched after the same normalization as entries, or for `vpic` one of
  `{"make"}`, `{"model"}` or `{"ext"}` as a mapping names one. `first` and
  `last` are optional and inclusive. `options` is the catalog's own option
  text, verbatim, and required (`""` for none). `qualifiers` are as on a
  mapping, and checked only from a VIN. One vehicle in several year
  ranges, or with other option text, is several applications; two that
  are the same in every field refuse the file.
- **A redirect** says that for a vehicle and years the `from` part type
  is listed under the `to` part type. A request follows it one hop.
- **A retired number** points at the fitment that replaced it. It must
  not be a live fitment, and its `to` must be a fitment in the file: the
  author resolves chains.

The catalog's vehicles are its own names, translated at request time, so
a model your vocabulary has not mapped yet stays listed by name, and
curating the vocabulary never forces a catalog rebuild. A catalog that
knows vPIC ids declares `vpic` and skips the translation.

Beyond the faults every file has, a catalog is refused for: a second
catalog; its spine or vehicle vocabulary not loaded; a part type the spine
lacks, in a fitment or a redirect; a vehicle whose fields are not exactly
the vocabulary's, or an extension no loaded set defines; qualifiers on an
application whose vehicle is an extension, which no decode lands on; a
side other than `left`, `right` or `both`; a fitment key, an application
within a fitment, or a retired number given twice; a retired number that
is live or points at no fitment. With `filed_under`, it is refused for: a
header vocabulary that is not loaded, not a part vocabulary or on another
spine; a line's `filed_under` when the header names none; an entry given
twice on one fitment, or whose fields are not exactly the vocabulary's;
an entry whose mapped part types do not include the fitment's part type.
Its drift, which is served: a vehicle the
vocabulary lacks or lists without a mapping (kind `vehicle`); a vPIC id
the month lacks (`make`, `model`); for a `vpic` catalog, an application
to a vPIC model in years outside the model's (`model-years`, ADR 0022); a
qualifier the month's decode cannot satisfy (`qualifier`); a redirect
whose `to` part type has no application for its vehicle in overlapping
years (`redirect`); an entry a fitment is filed under that the
`filed_under` vocabulary lacks or lists without a mapping
(`filed-under`); a fitment filed under no entry, in a catalog that names
a `filed_under` vocabulary (`unfiled`). A catalog names each vehicle on many lines, so its
drift is one finding per vehicle, vPIC id, qualifier value, model and
year range or filed-under entry, at the first application, redirect or
fitment naming it, with
`(and N more)` counting the others that do; one per redirect; and one at
the first unfiled fitment, counting the rest.

`/v1/info` lists a catalog with its `release`, its `filed_under`
vocabulary and its counts, `filings` its (entry, fitment) pairs:

```sh
nitrovin --vocab testdata/vocab/fitment info | jq -c '.vocabularies[] | select(.kind == "catalog") | del(.path, .blake3)'
```

```json
{"name":"parts-catalog","kind":"catalog","version":"2026.11","release":"7.02","filed_under":"part-codes","drift":[],"fitments":5,"applications":7,"redirects":1,"retired":1,"filings":6}
```

Translate reads only vocabularies, extension sets and spines: a catalog
is not in `list`'s catalog of what is loaded, and naming it as a `to` is
refused. **`fit`** reads the catalog
([the API's Fit section](api/README.md#fit-fit), ADR 0018): from a vehicle
its part types with counts, with a `part` its candidates, with `related`
the vehicles that share its fitments, and from a fitment key its vehicles.
Nothing to fit is a status, `no-catalog` included. `list` with `of` the
catalog's name pages its fitment lines, `where.type` keeping one part
type and the catalog's vehicle fields (`where.make`, `where.model`) one
vehicle's lines, each with only that vehicle's applications; a catalog
whose vehicles are vPIC's takes one of `where.make`, `where.model` and
`where.ext`, since its vehicle is exactly one of them. The synthetic catalog's `part-codes.jsonl` is a part vocabulary on
its spine, whose `ROT` code files both rotors under one code, so a fit
can name a part type in a parts system's words. As the catalog's
`filed_under` vocabulary, its entry as `part` also keeps only the direct
candidates filed under it, and a fitment key may name the code in place
of the part type, its absent `side` meaning any side; `MIR` with
`MR-1001` reaches both mirrors, so it answers `ambiguous`, listing them
in `fitments`, and the caller asks again with a side:

```sh
nitrovin --vocab testdata/vocab --vocab testdata/vocab/fitment fit --part part-types:mirror --side left 1D7RV1GT5AS123456
echo '{"from":{"model":1850},"model_year":2010,"part":{"vocabulary":"part-codes","entry":{"code":"ROT"}}}' |
  nitrovin --vocab testdata/vocab --vocab testdata/vocab/fitment fit -
echo '{"from":{"fitment":{"filed_under":{"code":"MIR"},"number":"MR-1001"}}}' |
  nitrovin --vocab testdata/vocab --vocab testdata/vocab/fitment fit -
```

`serve` never publishes a catalog, and a browser never reads the
server's: while the server has none, a browser may compile one of its own
(**In a browser**, below). In Go, a loaded
catalog is a `fitment.Reader` ([`pkg/fitment`](../pkg/fitment)), the one
interface every consumer reads it through (ADR 0017): `ByVehicle` and
`Matching` (a vehicle's fitments of every side), `ByNumber`, `ByFiled`
(by an entry, a number and a side, `nil` for any), `Redirect`,
`Fitments` (a page of fitments in key order) and `Info`. Which sides a
request keeps is the request's own rule, applied to what `ByVehicle`
answers ([the API's side filter](api/README.md#fit-fit)). Its vehicles
are the catalog's own; `fitment.Vehicles` turns a translation to the
catalog's vehicle vocabulary into them:

```go
t, _ := nitrovin.NewTranslator(d, files...) // files from vocab.Open("testdata/vocab/fitment")
cat := t.Catalog()                           // nil when no catalog is loaded
ram := vocab.Vehicle{Entry: map[string]string{"make": "Dodge", "model": "Ram 1500"}}
for _, c := range cat.ByVehicle("mirror", []vocab.Vehicle{ram}, new(2010)) {
	fmt.Println(c.Fitment.Key.Number, c.Fitment.Key.Side, len(c.Applications)) // MR-1001 left 2, MR-1001 right 1
}
fit, retired := cat.ByNumber("brake-rotor-front", "BR-1999", vocab.SideNone) // BR-2001, through the retired number
```

### Compiling a catalog

A catalog at a parts system's scale takes seconds and hundreds of MB to load
as JSON Lines. `nitrovin vocab compile` checks it as `vocab check` does and
writes it as a compiled catalog (ADR 0021): one indexed file that opens in
milliseconds with a few MB, read in place, answering exactly as its source.
It is compiled format 5 ([ADR 0026](adr/0026-case-exact-vocabularies.md)),
recording whether its vehicle and `filed_under` vocabularies were
case-exact, since that decides which spellings are one vehicle or entry,
carrying each fitment's entries and an index on (entry, number), and
clustered by vehicle: a vehicle's part types and candidates are two or
three reads of the file, and its related vehicles one plus one for about
each megabyte of the vehicles and years of the fitments of many
applications (universal parts) it shares, so it answers as quickly read by byte range from a browser's store
as from a disk. Compiling streams the catalog: it reads it once, sorts in a
temporary directory, and its memory grows with the catalog's vehicles and
part types, not its lines (about 200 MB at the scale of a parts system). Compile with the binary
that will serve it, at image build ([below](#at-image-build)):

```sh
nitrovin vocab compile -o catalog.compiled testdata/vocab/fitment   # the catalog, its spine, vehicles and codes
nitrovin --vocab testdata/vocab/fitment/part-types.jsonl --vocab testdata/vocab/fitment/catalog-models.jsonl \
  --vocab testdata/vocab/fitment/part-codes.jsonl --vocab catalog.compiled info | jq -c '.vocabularies[3].compiled'
```

```json
{"format":5,"source_blake3":"…","writer":"v…","drift_month":"2026_10","drift_month_differs":false,"links_differ":false,"skipped_drift":[]}
```

A fault refuses the compile, and nothing is written; drift is written to
stderr and stored in the file, `model-qualifier` included. `skipped_drift`
lists the drift kinds a compile could not measure: none from `vocab
compile`, `model-years` and `model-qualifier` from a compile in a browser,
whose spine bundle carries no model years or model values. `--vocab`
recognises the file by its magic (name it itself: a directory loads only
its `*.jsonl`) and refuses a compiled format the release cannot read
(format 1, from v0.3.5, and formats 2, 3 and 4 are refused with a message to
recompile), a spine, vehicle vocabulary or `filed_under` vocabulary
that is not loaded or lacks a part type, vehicle or entry it was compiled with,
and a vehicle or `filed_under` vocabulary that matches case otherwise than
the one it was compiled against, naming the vocabulary and both ways.
Loading it does not measure drift again in full: `/v1/info` shows the stored
drift, the month it was measured against, and `drift_month_differs` when
that is not the loaded month. Its `vehicle` drift alone is measured at load,
against the vehicle vocabulary loaded, which `links_differ` says is not the
one it was compiled with. `nitrovin vocab check` on the compiled file reads it
in full and checks and measures it again, as its source. JSON Lines stays
the published, diffable form; the compiled file is a deployment's, and the
same source, data month and nitrovin version always compile to the same
bytes.

Compiling sorts in a temporary directory, `$TMPDIR` or else `/tmp`, which
it removes when it ends, and writes the compiled file beside `-o` before
renaming it into place. The scratch is about twice the compiled file: a
parts system's catalog needed 537 MB of it for a 315 MB compiled file
(November 2026). When `/tmp` is small or in memory, point `TMPDIR` at a
disk with the room.

#### At image build

The release image (`ghcr.io/joshuasprow/nitrovin`) is the binary alone,
with no shell, run as uid 65532. From v0.3.11 it has an empty `/tmp`
owned by that user, mode 0700: the one directory the user can write. A
build stage from the image compiles there, in `RUN`'s exec form, and the
final stage, from the same image, serves the file. Saved as
`catalog.Dockerfile`, this compiles a directory holding a catalog with its
spine, vehicles and codes (the synthetic one's file names):

```dockerfile
ARG NITROVIN=ghcr.io/joshuasprow/nitrovin:latest

FROM ${NITROVIN} AS compile
COPY . /catalog/
RUN ["/usr/bin/nitrovin", "vocab", "compile", "-o", "/tmp/catalog.compiled", "/catalog"]

FROM ${NITROVIN}
COPY part-types.jsonl catalog-models.jsonl part-codes.jsonl /vocab/
COPY --from=compile /tmp/catalog.compiled /vocab/
ENV NITROVIN_VOCAB=/vocab:/vocab/catalog.compiled
```

```sh
docker build -f catalog.Dockerfile --build-arg NITROVIN=ghcr.io/joshuasprow/nitrovin:v0.3.11 -t parts testdata/vocab/fitment
docker run --rm -p 8080:8080 parts   # serve, as the image does; /v1/info lists the compiled catalog
```

Pin `NITROVIN` to the release you deploy: both stages read it, so the
binary that compiles is the one that serves. The compile stage's `/tmp`
holds the scratch and the compiled file at once, so the builder needs
room for about three times the compiled file; only the compiled file
reaches the final image, and `/vocab` loads the catalog's other files (a
directory loads only its `*.jsonl`). A pull request that touches the
release or this guide builds the recipe, as written here, from the
release's snapshot image against `testdata/vocab/fitment`, and asks the
result's `/v1/info` for the compiled catalog.

A writer that validates its lines in CI pins the schema by the same tag:
every GitHub release from v0.3.11 attaches `vocabulary.schema.json` and
`decode.schema.json`, each listed in the release's `checksums.txt` (which
its cosign bundle signs) and with a bundle of its own.

```sh
gh release download v0.3.11 -R joshuasprow/nitrovin -p vocabulary.schema.json -p checksums.txt
sha256sum --check --ignore-missing checksums.txt
```

## Stock

A **stock file** is one stock source for callers with no database
([ADR 0019](adr/0019-what-stock-says.md)): shelf parts, or whole vehicles
in a self-service yard, that `fit` answers beside its candidates. Any
number load with `--vocab`, each named in the one namespace; a new file
means a restart. Its header names the `source` (`shelf` or `yard`), its
`as_of` time (RFC 3339 or `YYYY-MM-DD`), and for a shelf the `catalog`
its numbers belong to, which must be loaded, and optionally the catalog
`release` they are under:

```json
{"format":1,"kind":"stock","name":"shelf-stock","version":"2026.11.01","source":"shelf","as_of":"2026-11-01T08:00:00Z","catalog":"parts-catalog","release":"7.01"}
```

Every row has the source's own `id` (unique in the file), an opaque
`location` a fit can filter on, and `fields`: an object passed through
verbatim and never interpreted. A shelf row's fields are flat (strings,
numbers and null); a yard row's are any JSON, so a list of photos passes
through. nitrovin interprets only how a row names what it is:

| Source | Row | Matched |
| --- | --- | --- |
| `shelf` | `type`, `number`, `side`: a fitment key. A retired number joins the fitment that replaced it | `fitment` |
| `shelf` | `part` (`{"spine", "id"}` on the catalog's spine, or `{"vocabulary", "entry"}` of a part vocabulary on it), `vehicle` (the catalog's own, as its lines write one) and `year`: a part with no number | `vehicle` |
| `yard` | `decoded`: `{"make", "model", "year"}`, a vPIC decode the source computed | `vin` |
| `yard` | `vin`: decoded by nitrovin, once, at load | `vin` |
| `yard` | `vehicle` and `year`: the catalog's own vehicle | `catalog-vehicle` |

A numbered row joins the fitment with exactly its key, side included: a
row with no `side` joins only the fitment with no side, never the left or
right fitment of its number. A system that leaves the side off a part the
catalog lists by side marks the side itself, in its view or its lines. A
row with no number is matched by `part`, `vehicle` and `year` alone: a
stock file refuses one that also gives `type` or `side`, and a connector,
whose relation has those columns on every row, does not read them where
`number` is null.

A yard row uses the first of its routes that gives a vPIC model and year.
One whose VIN decodes no model, or a model and no year, falls to its
`vehicle` and `year`; with no vehicle, a model without a year is in every
year of the model. Yard rows match by vehicle and years only: they carry
no decode for an application's qualifiers. `testdata/vocab/stock/` has a
row of each form. Its `shelf-stock` names release 7.01, so every answer
marks it `catalog_mismatch` against the catalog's 7.02, and its `S-3` is
filed under the retired number `BR-1999`; `shelf-annex` names no release,
so it is never marked:

```sh
echo '{"from":{"vocabulary":"catalog-models","entry":{"make":"Dodge","model":"Ram 1500"}},"model_year":2010,"part":{"vocabulary":"part-codes","entry":{"code":"ROT"}},"stock":["shelf","yard"]}' |
  nitrovin --vocab testdata/vocab --vocab testdata/vocab/fitment --vocab testdata/vocab/stock fit - |
  jq -c '.candidates[] | {number: .fitment.number, shelf: [.shelf.rows[] | {id, reached}], yard: [.yard.rows[] | {id, matched}]}'
nitrovin --vocab testdata/vocab --vocab testdata/vocab/fitment --vocab testdata/vocab/stock fit --part part-types:mirror --stock shelf --location north 1D7RV1GT5AS123456
```

```json
{"number":"BR-2001","shelf":[{"id":"A-1","reached":"direct"},{"id":"S-2","reached":"direct"},{"id":"S-3","reached":"retired"}],"yard":[{"id":"Y-1","matched":"vin"},{"id":"Y-2","matched":"vin"},{"id":"Y-3","matched":"catalog-vehicle"}]}
```

A shelf row whose number the catalog lacks, current or retired, is an
**orphan**: no candidate lists it, and `vocab check` and `/v1/info` count
it as drift, with examples. A count that should stay at 0 makes a good
alarm. A field named like personal data (its name holds `email`, `phone`,
`name`, `address` or `license` in any case, so `phonenumber` and
`part_name` alike; nested fields and those in arrays are checked too, as
`/fields/seller/email`) is drift too: a source supplies no names,
contact details or identity numbers, and nothing it would not show a
customer, such as cost. nitrovin cannot see what a field holds, so that
check is a nudge, not a guarantee. `testdata/vocab/stock/drift/` holds a
shelf with an orphan:

```sh
nitrovin vocab check testdata/vocab/fitment testdata/vocab/stock/drift   # one orphan, exit 1
nitrovin --vocab testdata/vocab/fitment --vocab testdata/vocab/stock info | jq -c '.vocabularies[] | select(.kind == "stock") | del(.path, .blake3)'
```

```json
{"file":".../testdata/vocab/stock/drift/shelf-orphans.jsonl","line":2,"severity":"drift","detail":"/number: catalog \"parts-catalog\" has no fitment {\"type\":\"mirror\",\"number\":\"MR-0000\",\"side\":\"right\"}, current or retired","kind":"orphan"}
{"name":"shelf-annex","kind":"stock","version":"2026.11.01","drift":[],"source":"shelf","catalog":"parts-catalog","as_of":"2026-10-30","rows":2}
{"name":"shelf-stock","kind":"stock","version":"2026.11.01","release":"7.01","drift":[],"source":"shelf","catalog":"parts-catalog","as_of":"2026-11-01T08:00:00Z","rows":5}
{"name":"yard-stock","kind":"stock","version":"2026.11.01","drift":[],"source":"yard","as_of":"2026-11-01T06:30:00Z","rows":4}
```

A shelf's catalog may be compiled (above): it answers stock, retired
numbers and orphans as its JSON Lines source, and `vocab check` on the
compiled file with the stock files finds the same drift.

**The order of a row's fields.** A row's `fields` are answered in the
order its source has them: a stock file's as each line writes them. A
header may declare the order instead
([ADR 0019](adr/0019-what-stock-says.md#amendment-2026-11-23-a-header-declares-its-rows-field-order)),
with `fields`, an array of distinct, non-empty field names:

```json
{"format":1,"kind":"stock","name":"shelf-order","version":"2026.11.01","source":"shelf","as_of":"2026-11-03","catalog":"parts-catalog","fields":["stock_number","condition","price","warranty"]}
```

Every row's fields then come with the names declared first, in the
header's order, for those the row has, then the row's other fields in the
line's order. It orders and never cuts: a name no row has (`warranty`,
here) is no fault and no drift, as two rows of one source need not share
their fields; `stock_fields` still cuts, and what it keeps is in this
order. A value is still verbatim: a member moves as the bytes the line
wrote, so `1850.00` stays `1850.00`, and only an object's own names are
ordered, never what a yard row nests under one. Without `fields`, or with
`[]`, every answer is what it was. `/v1/info` reports the declared order
beside the source's `source` and `catalog`. `testdata/vocab/stock/order`
is a shelf and a yard whose lines are written out of the order their
headers declare:

```sh
echo '{"from":{"fitment":{"type":"engine","number":"EN-57"}},"stock":["shelf","yard"]}' |
  nitrovin --vocab testdata/vocab/fitment --vocab testdata/vocab/stock/order fit - |
  jq -c '.shelf.rows[], .yard.rows[] | {id, fields: (.fields | keys_unsorted)}'
nitrovin --vocab testdata/vocab/fitment --vocab testdata/vocab/stock/order info |
  jq -c '.vocabularies[] | select(.kind == "stock") | {name, source, fields}'
```

```json
{"id":"R-1","fields":["stock_number","condition","price","tag","miles","status"]}
{"id":"R-2","fields":["stock_number","price","comments"]}
{"id":"R-3","fields":["tag","miles"]}
{"id":"R-4","fields":["stock_number","condition","price","tag"]}
{"id":"V-1","fields":["stock_number","row","photos","color","added"]}
{"id":"V-2","fields":["color","keys"]}
{"name":"shelf-order","source":"shelf","fields":["stock_number","condition","price","warranty"]}
{"name":"yard-order","source":"yard","fields":["stock_number","row","photos"]}
```

`R-1`'s line writes `tag` first and its stock number last; `R-3` has none
of the names and stays as written. In Go, the order is the stock file's
`vocab.File.Fields`, and a `stock.Source` returns each row's `Fields` in
it: `vocab.AppendStockFields` orders one row's.

A shelf whose catalog is not loaded, a row in neither form, a duplicate
id, a part the catalog's spine lacks, and a vehicle its vehicle
vocabulary has no entry for are faults. `serve` publishes no stock; a
browser loads stock files of its own beside the catalog it holds
(**In a browser**, below), and never a connector. In Go, each source is a `stock.Source`
([`pkg/stock`](../pkg/stock)): `stock.NewFile` over a loaded stock file,
or a connector's (below). Every
lookup takes a `context.Context` and returns the release and `as_of` it
read with its rows; a source that cannot answer returns
`*stock.Unavailable`, whose reason the answer shows. `Translator.Fit(ctx,
req)` passes its `ctx` on: a live source that has not answered when it
ends is `unavailable`, and the fit returns.

### Connectors

A **connector** reads a stock source from Postgres at request time
([ADR 0020](adr/0020-request-time-stock-the-postgres-connector.md)), so a
yard's rows and photos are as fresh as its database. It is a stock file
with a header and no lines, loaded with `--vocab` like any other, in the
one namespace of names. `testdata/vocab/stock/live` holds one for each
synthetic source:

```json
{"format":1,"kind":"stock","name":"shelf-stock","version":"2026.11.01","source":"shelf","catalog":"parts-catalog","connector":{"postgres":{"relation":"stock.shelf_stock","release_relation":"stock.shelf_stock_release","dsn_env":"NITROVIN_STOCK_DSN"}}}
```

| `connector.postgres` | |
| --- | --- |
| `relation` | the table or view holding the rows: `name` or `schema.name`, unquoted lower case |
| `release_relation` | a shelf's only, optional: a relation of one row whose `release` column is the catalog release its numbers are under (in place of the header's `release`) |
| `dsn_env` | the environment variable holding the DSN (`postgres://user:password@host/db` or `key=value` form, with libpq's `PG*` variables and `.pgpass` as defaults). The file carries no secret, and no answer, `/v1/info` included, shows a DSN, host or user |
| `statement_timeout_ms` | each query's bound, default 1000 |
| `connect_timeout_ms` | each connection's bound, default 2000 |
| `max_connections` | the most connections held open at once, default 4 |

A connector's header has no `as_of` and no `release`: its rows carry
their own. Any line after the header, and any other member (a DSN among
them), is a fault.

It may declare `fields`, the order of its rows' fields (**The order of a
row's fields**, above), beside `connector`, not inside it. A connector
needs it where a stock file does not: a file's author writes each line in
the order wanted, and a connector's `fields` column is read as `jsonb`,
which stores an object's keys by length and then by bytes. A row written
stock number first arrives as `tag`, `kind`, `miles`, `price`, `status`,
its stock number last, and no view can change that:

```json
{"format":1,"kind":"stock","name":"shelf-stock","version":"2026.11.01","source":"shelf","catalog":"parts-catalog","fields":["stock_number","condition","price"],"connector":{"postgres":{"relation":"stock.shelf_stock","dsn_env":"NITROVIN_STOCK_DSN"}}}
```

Each row is put in that order as it is read, every member as `jsonb`'s
text writes it, and the names the header leaves out follow in `jsonb`'s
order.

**The relation's columns.** nitrovin dictates column names, never SQL: a
view renames a system's own columns. Each is read in one of a few types:

| Source | Columns |
| --- | --- |
| `shelf` | `id`, `location` (text); `type`, `number`, `side` (text; `side` null for a fitment with none; `number` null on a row with no number, whose `type` and `side` are not read); `part_type` (text, a spine id, for a row with no number); the vehicle columns; `year` (integer); `fields` (jsonb or json); `as_of` (optional) |
| `yard` | `id`, `location` (text); `decoded_make`, `decoded_model`, `decoded_year` (integer, the source's own vPIC decode of the vehicle's VIN); the vehicle columns; `year` (integer); `fields` (jsonb or json); `as_of` (optional); `vin` (text, optional) |

A yard's `vin` is the vehicle's VIN (ADR 0028). With the column, a yard
row says its VIN, as a file's does, and the yard can name the vehicles it
holds most of ([`stocked`](api/README.md#stocked-stocked)). Without it
every lookup answers as before and the yard names none. nitrovin still
never decodes it: the decode is `decoded_*`'s.

The vehicle columns are `vehicle_<field>` (text) for each field of the
catalog's vehicle vocabulary, `vehicle_make` and `vehicle_model` for
`catalog-models`; under a catalog whose vehicles are vpic,
`vehicle_make` and `vehicle_model` (integer, vPIC ids) and `vehicle_ext`
(text), one of them set. A yard's catalog is the one loaded. Text is
`text`, `character varying` or `character`; an integer is `smallint`,
`integer` or `bigint`; `as_of` is `timestamp with time zone` (served in
UTC), `timestamp without time zone` (read as UTC) or `date`.

The rows match as a file's do (ADR 0019), with what a database cannot
carry left to the source: a yard row is matched by its decode when all
three `decoded_*` columns are set (`matched: "vin"`; nitrovin never
decodes a connector's VIN, so a source keeps its decodes fresh itself,
for example with `POST /v1/decode`), else by its vehicle and `year`; a
numbered shelf row matches a key's `side IS NOT DISTINCT FROM` its own,
so a null `side` joins only the fitment with no side, as a file's row
with no `side` does; a vehicle matches when its columns equal the
catalog's values exactly, as the catalog writes them; a row with a null
`id` or `location` is never
served; `fields` that are null, or not an object, are served as `{}`.
Each list's rows come in `id` order (bytewise), and its block's `as_of`
is the oldest of that list's own rows: two candidates' shelves can say
two times, and a list with none of the source's rows says none.

**Lookups and indexes.** Each lookup is one fixed, parameterised `SELECT`
for every list of its kind in an answer (for a batch's fitment keys, in
the batch), each list bounded by the page, in its own `READ ONLY`
transaction under the statement timeout. A yard lookup reads each vPIC
model and catalog vehicle once however many of its lists name it. Give
the relation's tables these indexes:

- shelf, by fitment keys: `(type, number, side)`;
- shelf, rows with no number: `(part_type, vehicle_<fields>, year)`;
- yard, by vPIC model: `(decoded_model, decoded_year)`;
- yard, by catalog vehicle: `(vehicle_<fields>, year)`.

Under a vpic catalog, `vehicle_<fields>` is `vehicle_make, vehicle_model,
vehicle_ext`: a lookup asks for the one column its vehicle sets, by
equality, and the other two `IS NULL`, which the same index serves.

**Through a barrier.** A lookup hands each of its keys to the relation as
conditions on the relation's own columns, the key's values as parameters:
a probe of the index per key, whether the relation is a table, a plain
view or a `security_barrier` view. Where no index can serve the key, a
lookup reads the whole relation once per key, not once per lookup.

A barrier lets a condition through to the view's tables only when every
function in it that is given one of the view's columns is leakproof,
which is Postgres's mark on a function that reveals nothing about its
arguments other than by its return value; it holds any other back until
the view has made its rows. A function of the request's values alone,
such as the one that asks whether the request names any location, needs
no mark. These are all the conditions a lookup puts on a column, and on
a `text`, `character varying` or integer column each passes (a
`character` column is below):

- `=` on a text or an integer column: `type`, `number`, `part_type`,
  `decoded_model`, a vehicle column, `year`;
- `IS NULL` and `IS NOT NULL`;
- `side IS NOT DISTINCT FROM` the key's side;
- `location = ANY(…)`, the request's locations;
- `>=` and `<=` on `year` or `decoded_year`, for a range of model years.

So the indexes serve a barrier view as they serve a plain one, as long as
the view leaves each indexed column something its index answers those
conditions with. The rule is on the lookup's conditions, not on the view,
whose own expressions need not be leakproof. Where a column below is not
served, no index serves the key, and a lookup reads the relation once
per key:

- a table's column under the relation's name, as in the worked view
  below, is served;
- an expression is served when the index is on the same expression: a
  view that trims, `btrim(catalog_no) AS number`, needs its index on
  `btrim(catalog_no)`, which one on `catalog_no` is not;
- a `character` equality key column (`type`, `number`, `part_type`, a
  vehicle column) is served by an index on the column. Postgres compares
  a `character` column with a key as text, through a cast that no index
  on the column answers and no barrier passes, so a lookup first holds
  an equality key column to the key as `character`, which both do; the
  answers are the text comparison's. A `character` `side` or `location`
  gets no second comparison, and its condition waits above a barrier:
  that loses no probe, since the key columns make the probe and these
  two only filter the rows it finds. Under a nondeterministic collation
  the two comparisons can disagree, so the lookup makes the text one
  alone and no index on the column serves it: cast such a key column in
  the view, `part_class::text AS type`, and index that expression;
- a view with `LIMIT` or `OFFSET` takes no condition at all, and so
  makes its every row for every key of every lookup; so does a view with
  a window function, unless the window partitions by the key's columns.
  A column that is an aggregate's result takes its condition only after
  the grouping.

**A worked view.** `testdata/vocab/stock/live/views.sql` is a synthetic
parts system's table and a yard system's, the views the connectors read,
and a role that reads only the views; CI loads the synthetic rows into it
and checks that the connectors answer every fit as the same rows loaded as
files do, and that each lookup's plan reads the tables' indexes through
the views. The shelf's view, and the role:

```sql
CREATE INDEX shelf_by_number ON parts.shelf (part_class, catalog_no, hand);
CREATE INDEX shelf_by_vehicle ON parts.shelf (spine_part, make, model, model_year) WHERE catalog_no IS NULL;

CREATE VIEW stock.shelf_stock WITH (security_barrier) AS
SELECT stock_id AS id, bin AS location,
       part_class AS type, catalog_no AS number, hand AS side, spine_part AS part_type,
       make AS vehicle_make, model AS vehicle_model, model_year AS year,
       attrs AS fields
FROM parts.shelf WHERE shelf = 'main';
CREATE VIEW stock.shelf_stock_release WITH (security_barrier) AS
SELECT release FROM parts.release WHERE shelf = 'main';

CREATE ROLE nitrovin_stock LOGIN PASSWORD 'stock-reader';
DO $$ BEGIN
  EXECUTE format('REVOKE TEMPORARY ON DATABASE %I FROM PUBLIC', current_database());
END $$;
GRANT USAGE ON SCHEMA stock TO nitrovin_stock;
GRANT SELECT ON ALL TABLES IN SCHEMA stock TO nitrovin_stock;
ALTER DEFAULT PRIVILEGES IN SCHEMA stock GRANT SELECT ON TABLES TO nitrovin_stock;
```

The role sees what the views show and no more, should its DSN reach
anyone but nitrovin:

- **`security_barrier`** runs a view's `WHERE` before any condition of
  the query reading it that gives a column to a function not marked
  leakproof. A plain view lets the planner run a cheap condition first,
  so a function the reader wrote, which can raise or log its arguments,
  would see the rows the view leaves out. It costs a lookup no index:
  every function its conditions give a column to is leakproof
  (**Through a barrier**, above), so they reach `shelf_by_number` and
  `shelf_by_vehicle` as they would through a plain view.
- **`TEMPORARY` revoked from `PUBLIC`** leaves the role nowhere to write
  such a function. Every role holds `TEMPORARY` through `PUBLIC`, and it
  lets a role create one in `pg_temp`; a database made by Postgres 15 or
  later already grants `PUBLIC` no `CREATE` on schema `public` (revoke it
  on an older one). A `NOINHERIT` role would not do: `PUBLIC`'s
  privileges reach every role whatever it inherits. Grant `TEMPORARY`
  back to the roles that need temporary tables.
- **`ALTER DEFAULT PRIVILEGES`** grants `SELECT` on the views the
  script's role makes in `stock` later: `GRANT … ON ALL TABLES` covers
  only those that exist when it runs.

A system without a jsonb column builds `fields` in the view, which is
also where it leaves out what it would not show a customer (ADR 0019):
`jsonb_strip_nulls(jsonb_build_object('stock_number', stock_no, 'price', list_price)) AS fields`.

**Status and failure.** Nothing connects at load: `nitrovin info` says
`status: "unchecked"` until a fit asks for the source's stock. Its first
lookup connects and checks the relation's columns against
`information_schema`; a missing or mistyped column makes the source
unavailable, naming the column (`column year of stock.shelf_stock is
text, not an integer (smallint, integer or bigint)`). A database that is
down does not stop nitrovin, at start or later: its lists answer
`unavailable` (`the database is unreachable`), at once and with no
connection for 5 s, then it tries again. A slow query fails only its own list. A relation that
changes under a running nitrovin is checked again. `/v1/info` shows each
connector's `connector`, `relation` and latest `status`:

```sh
NITROVIN_STOCK_DSN=postgres://… nitrovin --vocab testdata/vocab --vocab testdata/vocab/fitment --vocab testdata/vocab/stock/live serve
```

**Where it runs.** The `nitrovin` binary's `serve`, stdio, CLI and MCP
read a connector; its driver, `pgx` (MIT, [`pkg/stock/pg`](../pkg/stock/pg)),
adds 4.3 MB to the 75.4 MB release binary (`-trimpath -ldflags='-s -w'`,
darwin/arm64 and linux/amd64 alike), most of it Postgres's type codecs
and the Unicode tables its password hashing needs; ADR 0020's estimate
was 0.5 MB. The C library loads a connector's file as a source that is
always unavailable (`this build reads stock files only: a connector
needs the nitrovin binary`), and the browser refuses one (`connectors run
on a server`) and loads stock files alone; neither links a driver. In Go, `nitrovin.NewTranslatorWith(d,
nitrovin.TranslatorConfig{Connect: pg.Connect(os.Getenv)}, files...)`
connects them, and `nitrovin.NewTranslator` does not; the Translator's
`Close` releases their connections, which every `nitrovin` command does
when it ends (`serve` on SIGINT or SIGTERM, stdio and MCP at the end of
their input).

## Loading

`--vocab <file|dir>`, before or after any command and repeatable, loads
files and directories (every `*.jsonl` directly in one, in name order).
Without it, `NITROVIN_VOCAB` is a path list (`:`-separated, `;` on
Windows). Files load once, with the data, and never change while nitrovin
runs; restart `serve` to pick up a new file.

```sh
nitrovin --vocab testdata/vocab info | jq -c '.vocabularies[] | {name, kind, version, drift}'
NITROVIN_VOCAB=testdata/vocab nitrovin list | jq -c '.entries[] | {name, kind, spine}'
```

`/v1/info` (and `nitrovin info`) lists every loaded file with its version,
blake3 hash, absolute path and drift, a catalog with its release and
counts, and a stock source with what it holds, its time and its rows.

## Checking

nitrovin tells two kinds of problem apart:

- A **fault** is the file's own mistake, whatever the month: a malformed
  line, an unknown field, two entries that normalize alike, a first year
  after the last, a reference no loaded file defines. A fault stops every
  command before it answers, naming the file and line.
- **Drift** is where a file disagrees with the loaded data month: a vPIC
  id NHTSA has since dropped, a qualifier value the month never decodes, or
  one the month decodes but never for the mapping's own vPIC model (a
  Series only the van line has, on a pickup), an extension named like a
  vPIC entry. NHTSA's monthly changes are not your
  mistakes, so drift is a warning on stderr and in `/v1/info`, and the
  file is served as written. The per-model check (`model-qualifier`)
  costs about 0.14 s and 80–100 MB of memory for an EPA-sized vocabulary,
  so it is left out of load's stderr warnings: `nitrovin vocab check`
  runs it, and `/v1/info` counts it from its first request.

`nitrovin vocab check` reads only its arguments, prints every finding as
a JSON line on stdout and exits 1 on any, so CI can gate a
vocabulary on the month it will run against. A drift line names its
`kind`, so a tool that keeps findings can group them without reading
`detail`: `make`, `model`, `qualifier`, `extension`, `model-years`,
`vehicle`, `redirect`, `model-qualifier`, `orphan`, `personal-field`,
`filed-under` or `unfiled`,
the kinds `/v1/info` counts. A fault has no `kind`. A finding about a
vocabulary's line carries `entry`, its fields as written, and one about an
extension set's or spine's line `id`, faults and drift alike, so a tool
need not read the file again to learn which entry it is about
([ADR 0025](adr/0025-checks-at-scale.md)); a catalog's or stock file's
finding, or one about a line that did not read, carries neither.

A fault in one file does not hide the others' drift: every file that
reads and links cleanly is measured. A file that has faults, or links,
even through other files, to one that does (a catalog to its spine and
vocabularies, a part vocabulary to its spine, a file to the extension sets
it references, a stock file to its catalog), is **unchecked**: it gets one
line at line 0, after its faults, saying which file has them, and a tool
that keeps findings should keep its last drift rather than clear it.
Exit codes are unchanged, and loading still refuses every file when any
has a fault.

A JSON Lines catalog is checked as `vocab compile` reads it, streamed, its
compiled bytes written nowhere: like a compile it sorts in `$TMPDIR`, else
`/tmp`, about twice the compiled file, removed when it ends, and its
memory does not grow with the catalog's lines (191 MB at peak for 790,000
fitments on one core, against a budget of 512 MiB). When a stock file
being checked names the catalog, the catalog is compiled into that scratch
and read from there.

```sh
nitrovin vocab check testdata/vocab            # nothing printed, exit 0
nitrovin vocab check testdata/vocab/drift      # seven drift lines, exit 1
nitrovin vocab check testdata/vocab/fitment    # a catalog with its spine and vehicles: nothing, exit 0
nitrovin vocab check testdata/vocab/fitment/catalog.jsonl   # alone: three faults and an unchecked line, exit 1
nitrovin vocab check testdata/vocab/unchecked  # one fault, four unchecked lines (the faulty file and three linked to it), another file's drift, exit 1
```

```json
{"file":".../testdata/vocab/drift/fleet.jsonl","line":4,"severity":"drift","detail":"/mappings/0/make: vPIC make 99999 is not in data month 2026_10","kind":"make","entry":{"make":"Imperial","model":"Other"}}
{"file":".../testdata/vocab/unchecked/catalog.jsonl","line":0,"severity":"unchecked","detail":"its drift was not measured: it links through .../fleet.jsonl to .../classics.jsonl, which has faults"}
{"file":".../testdata/vocab/unchecked/classics.jsonl","line":3,"severity":"fault","detail":"malformed JSON: unexpected end of JSON input"}
{"file":".../testdata/vocab/unchecked/classics.jsonl","line":0,"severity":"unchecked","detail":"its drift was not measured: it has faults"}
{"file":".../testdata/vocab/unchecked/fleet.jsonl","line":0,"severity":"unchecked","detail":"its drift was not measured: it links to .../classics.jsonl, which has faults"}
```

A fault:

```sh
printf '%s\n' '{"format":1,"kind":"vocabulary","name":"bad","version":"1","subject":"vehicle","fields":["make"]}' \
  '{"entry":{"make":"Dodge"},"mappings":[{"make":476,"first":2014,"last":2012}]}' > bad.jsonl
nitrovin vocab check bad.jsonl
nitrovin --vocab bad.jsonl info    # refuses to start: exit 1
```

```json
{"file":".../bad.jsonl","line":2,"severity":"fault","detail":"/mappings/0: first year 2014 is after last year 2012"}
{"file":".../bad.jsonl","line":0,"severity":"unchecked","detail":"its drift was not measured: it has faults"}
```

Check against the data you will serve: `--data` picks the month for
`vocab check` as for any command.

## Translating

A translation starts `from` a VIN, a vocabulary entry or a spine entry,
reaches the spine, and answers the entries of the `to` vocabularies that
map there. Without `to` it answers the spine alone.

```sh
nitrovin --vocab testdata/vocab translate --to fleet 1D7RV1GT5AS123456 | jq -c '.to[0].entries[].entry'
```

```json
{"make":"Dodge","model":"Other"}
{"make":"Dodge","model":"Ram 1500"}
{"make":"Dodge","model":"Ram 1500 4x4"}
```

The VIN decodes to a 2010 Dodge Ram, Series `DS (1500)`, four-wheel drive,
so every mapping's year range and qualifiers hold. The 2015 Ram
(`1C6RR7LT5FS123456`) reaches the same two `Ram 1500` names through model
13620 instead, and `Ram / Other` through make 496.

- **`spine`** is every spine entry reached, once each, a model with the
  make it sits `under`; a vPIC id the month lacks comes back with
  `"drift": true`.
- **`via`** on each spine and target entry is the mappings that led there,
  with their `first`, `last`, `origin` and `note`.
- **`unconfirmed`** lists what could not be checked: qualifier element ids
  the decode left empty, and `model_year` when no year was known. Keep
  only entries without `unconfirmed` to keep only confirmed results.
- **`from.status`** is `mapped`, `unmapped` (listed without mappings) or
  `unknown` (not in the vocabulary, or a VIN that decodes to no make or
  model). Nothing to translate is an answer, never an error; an error is
  only for a request the loaded files cannot read, such as a `to` nothing
  loaded has.

From an entry, a translation checks year ranges against `model_year` and
ignores qualifiers, so it returns the union of the entry's mappings. Give
a VIN as context (`vin`) to check them against its decode:

```sh
nitrovin --vocab testdata/vocab translate - <<'EOF' | jq -c '{from: .from.status, spine: [.spine[] | .name]}'
{"from": {"vocabulary": "fleet", "entry": {"make": "Packard", "model": "Clipper"}}, "model_year": 1955}
{"from": {"vocabulary": "fleet", "entry": {"make": "Studebaker", "model": "Lark"}}}
{"from": {"vocabulary": "fleet", "entry": {"make": "Nash", "model": "Rambler"}}}
{"from": {"vocabulary": "listing", "entry": {"category": "Brake Parts"}}, "to": ["listing"], "vin": "1D7RV1GT5AS123456"}
EOF
```

```json
{"from":"mapped","spine":["Clipper","Packard"]}
{"from":"unmapped","spine":[]}
{"from":"unknown","spine":[]}
{"from":"mapped","spine":["Brake Pads","Brake Pads, Motorcycle","Brake Rotor"]}
```

In the last answer, the listing's `Brake Pads` entry comes back through
`brake-pads` but not through `brake-pads-motorcycle`: the VIN's Vehicle
Type rules that mapping out.

## Listing

`list` with nothing names what is loaded: vPIC, then every file in load
order. With a name it pages through that vocabulary's entries and their
mappings, a spine's or extension set's ids, or vPIC's makes; `where`
filters by the listing's fields after the same normalization, and `next`
is the cursor for the following page.

```sh
nitrovin --vocab testdata/vocab list
nitrovin --vocab testdata/vocab list fleet --where make=dodge --limit 2 | jq -c '[.entries[].entry], .next'
nitrovin --vocab testdata/vocab list fleet --where make=dodge --limit 2 --after WyJkb2RnZSIsInJhbSAxNTAwIl0 | jq -c '[.entries[].entry]'
```

## On every surface

Translate and list are the same two request objects everywhere
([Translation](api/README.md#translation-translate),
[Listing](api/README.md#listing-list)), answered from whatever `--vocab`
or `NITROVIN_VOCAB` loaded; with nothing loaded they answer from vPIC
alone.

**CLI**: `nitrovin translate <vin>…` or `-` for one request object per
line, `nitrovin list [of]`, both above.

**HTTP**: `POST /v1/translate` takes an array of up to 1,000 requests;
`GET /v1/translate` takes one as dotted query parameters; `GET
/v1/vocabularies` is the catalog and `GET /v1/vocabularies/{name}/entries`
a listing.

```sh
nitrovin --vocab testdata/vocab serve    # 127.0.0.1:8080; then, from another shell:
```

```sh
curl 'localhost:8080/v1/translate?from.vin=1D7RV1GT5AS123456&to=fleet'
curl 'localhost:8080/v1/translate?from.vocabulary=fleet&from.entry.make=Dodge&from.entry.model=Ram%201500&model_year=2015'
curl localhost:8080/v1/translate -d '[{"from": {"model": 1850}, "to": ["fleet"]}, {"from": {"spine": "parts", "id": "brake-rotor"}, "to": ["listing"]}]'
curl localhost:8080/v1/vocabularies
curl 'localhost:8080/v1/vocabularies/fleet/entries?where.make=dodge&limit=2'
```

`serve` also publishes each loaded file but a catalog, byte for byte, for a browser that
translates locally
([ADR 0013](adr/0013-browser-vocabularies-in-a-service-worker.md)), at
`/ui/r/<release>/vocab/<blake3>/<name>.jsonl`, where the release is
`/v1/info`'s `decoder_version`, `data_month` and `artifact_hash` joined by
`/` and the blake3 is the file's there; beside it the data month's spine
bundle at `/ui/r/<release>/spine.json`. Both are immutable for a year: a
new release or a changed file is a new URL. This exposes nothing new, since
`/v1/vocabularies/{name}/entries` already lists every line. `serve` reads a
file again when it is asked for, and refuses with a 500 a file that
changed on disk since it was loaded: restart `serve` to publish the new
one.

**In a browser**: `serve`'s page at `/` decodes a VIN, as a diagram of
its six groups, and shows the vehicle's names in every vocabulary loaded
on vPIC's spine, with JavaScript or without. Its form is a plate that
takes a VIN and nothing else, so the page's URL is `/?from.vin=…` and
names no vocabulary: there are no targets to pick, and what is loaded is
what is shown. The page reads no other parameter: a model year or a
clock in its URL is ignored, and the translation is the VIN's as of now.
A refusal is said below the plate. Older links
(`/?vin=…`, `/ui/decode`, `/ui/translate`, `/vocab`,
`/ui/vocabularies`) redirect to it (`301`) with the VIN they named and
nothing else. Below the plate the answer
is a card for each group and, under them, "Also known as": a tag for
each vocabulary, in one row that scrolls sideways, one with a name before
one without. A tag reads the vocabulary, then the make, the rest of the
entry and the years its mapping holds for. A vocabulary with several
names shows every one, with their count. A dashed underline is a doubt:
a mapping matched by rule, or
a qualifier or the model year that was not checked; each part's title
says which, and a small button on the name opens how it was reached.
A vocabulary with no name for the vehicle
keeps a tag that says so. The Developers button in the page's header
shows the answer's JSON, the decode's and the translation's. Back and
forward ask the server for each page again,
never a copy the browser kept. The header's Vocabularies button
opens a dialog that
lists what is loaded, each file marked by where it is kept, with its
drift counted by kind as `/v1/info` counts it (the first examples behind
a disclosure), each linked to its entries (`/ui/vocabularies/{name}`, a
page that carries the same dialog).
With JavaScript the dialog also loads your own files,
picked from disk or dropped anywhere on the page, into the browser: a
Service Worker (`/sw.js`), started when the dialog first opens or a file
is dropped, and at page load when one is already registered, keeps
each file's text and parsed index in
IndexedDB and runs the same pages, built from the same Go to WebAssembly,
over them and the server's files, which it links from the URLs above.
The dialog then lists one catalog: vPIC, the server's files and yours, each
marked as built in, on this server or in this browser, yours with their
file name, size and when they were stored, and how much the site keeps in
the browser. Files loaded together load as one batch, which must link
(with the server's files and those already stored), and a file named like
a stored one replaces it; a file named like one of the server's is
refused, and so is a file with a fault, naming the file and line. Removing
a file others link to is refused, naming them, with an offer to remove
them all. A stored file that this release can no longer read, or that no
longer links, is listed with its fault and can be removed. Once the
browser holds a file, the worker answers the page, over the server's
files and yours, and the entry lists of your files, and leaves
every other request to the server (for a VIN, the
decode is `/v1/decode`'s). The page's link names no file, so it opens in
every browser and with JavaScript off: it shows the server's
vocabularies everywhere, and yours beside them only in the browser that
holds them. A link to the entries of your file, opened in another
browser, says where it lives. Answers carry the `decoder_version` and `data_month`
of the decode they used. When the server moves to a new release, the
worker steps aside: every page comes from the server, your files are
paused, and Vocabularies says so. Once the new release's worker is ready,
**Update now** lights up in the header, beside the brand; it hands the
page to that worker and reloads, and your files stay. Without the click,
the new worker takes over once every tab of the site is closed. A tab
left open hears of a new release without a worker too, from the server's
next answer or when the tab is shown again, and its Update now reloads
it. A deploy's old container may answer once after the new one: the
files pause, the worker asks the server again within seconds, and they
come back by themselves. Every page's foot names the release that
drew it. A page loaded with
a hard reload is not the worker's, and says your files are paused for
that load. Release builds also offer [EPA's vocabulary](#epas-vocabulary)
of their data month as a one-click load: `serve` publishes it, byte for
byte as nitrovin-data's release has it, at
`/ui/r/<release>/vocab/<blake3>/epa-vehicles.jsonl`, and the browser keeps
it like a file of yours. Every build with the worker offers [the demo
set](#the-demo-set) the same way, publishing each of its files there. A catalog picked on its own is compiled in the
browser instead, while the server has none, and the dialog offers the
compiled file as a download
([ADR 0023](adr/0023-fitment-in-the-browser.md)). Stock files load in the
dialog too: a shelf beside that catalog, which the browser must hold, and
a yard, whose VINs (of the rows with no `decoded`) the worker sends to the
server's `POST /v1/decode` once as it loads, keeping the decodes with the
file. A row's fields never leave
the browser, and the dialog repeats that a stock file supplies no personal
data. A connector's file is refused: connectors run on a server.

**stdio**: a line with `"op": "translate"` or `"op": "list"` carries that
request, and its reply is keyed `translation` or `list`:

```sh
printf '%s\n' '{"id":1,"op":"translate","from":"1D7RV1GT5AS123456","to":["fleet"]}' \
  '{"id":2,"op":"list","of":"parts"}' | nitrovin --vocab testdata/vocab stdio
```

**MCP**: the `translate` and `list` tools take the same objects as
arguments, over stdio (`nitrovin --vocab <path> mcp`; put `--vocab` and an
absolute path in the client's `args`) or at `serve`'s `/mcp`:

```sh
npx @modelcontextprotocol/inspector --cli http://127.0.0.1:8080/mcp --transport http \
  --method tools/call --tool-name translate --tool-arg from=1D7RV1GT5AS123456 --tool-arg 'to=["fleet"]'
npx @modelcontextprotocol/inspector --cli http://127.0.0.1:8080/mcp --transport http \
  --method tools/call --tool-name list --tool-arg of=fleet --tool-arg limit=2
```

The tool descriptions never name what is loaded; an assistant calls
`list` with no arguments to find out.

**Go**: `vocab.Open` reads and links files, refusing faults;
`nitrovin.NewTranslator` measures their drift against the decoder's data
month and answers `Translate` and `List`.

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/joshuasprow/nitrovin/pkg/contract"
	"github.com/joshuasprow/nitrovin/pkg/nitrovin"
	"github.com/joshuasprow/nitrovin/pkg/vocab"
)

func main() {
	d, err := nitrovin.Embedded()
	if err != nil {
		log.Fatal(err)
	}
	files, err := vocab.Open("testdata/vocab") // files and directories, as --vocab takes them
	if err != nil {
		log.Fatal(err) // vocab.Faults: every fault, by file and line
	}
	t, err := nitrovin.NewTranslator(d, files...) // drift is in t.Drift(), served anyway
	if err != nil {
		log.Fatal(err)
	}
	req, err := contract.ParseTranslateRequest([]byte(`{"from": "1D7RV1GT5AS123456", "to": ["fleet"]}`))
	if err != nil {
		log.Fatal(err)
	}
	tr, err := t.Translate(req) // an error only for a request the loaded files refuse
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range tr.To[0].Entries {
		out, _ := json.Marshal(e.Entry)
		fmt.Println(string(out))
	}
}
```

**C**: `nitrovin_open_vocab` opens an artifact with vocabularies beside
it; `nitrovin_decode` then answers translate and list lines exactly as
`nitrovin stdio` does ([the C library](c-library.md#vocabularies)):

```c
#include <stdio.h>
#include "nitrovin.h"

int main(void) {
  const char *vocab[] = {"testdata/vocab"};
  char *err = NULL;
  nitrovin_t *h = nitrovin_open_vocab("nitrovin-2026_10.artifact", vocab, 1, &err);
  if (h == NULL) {
    fprintf(stderr, "%s\n", err); /* a fault: the file and line */
    nitrovin_free(err);
    return 1;
  }
  char *reply = nitrovin_decode(h,
      "{\"id\":1,\"op\":\"translate\",\"from\":\"1D7RV1GT5AS123456\",\"to\":[\"fleet\"]}");
  puts(reply); /* {"id":1,"translation":{...}} */
  nitrovin_free(reply);
  reply = nitrovin_decode(h, "{\"id\":2,\"op\":\"list\",\"of\":\"fleet\",\"limit\":2}");
  puts(reply); /* {"id":2,"list":{...}} */
  nitrovin_free(reply);
  nitrovin_close(h);
  return 0;
}
```

```sh
gh release download v1.202610.2 -R joshuasprow/nitrovin-data -p nitrovin-2026_10.artifact
cc -Wall -Wextra -o translate translate.c -L. -lnitrovin -Wl,-rpath,. && ./translate
```

A library has no stderr, so `nitrovin_info` is where a handle's drift
shows.

## EPA's vocabulary

Each [nitrovin-data release](https://github.com/joshuasprow/nitrovin-data/releases)
carries `epa-vehicles.jsonl`: the make and model names of EPA's
fuel-economy data
([vehicles.csv](https://www.fueleconomy.gov/feg/download.shtml), public
domain) mapped onto that month's vPIC makes and models, with Drive Type
and Series qualifiers where a name implies them. `cmd/nitrovin-epa`
builds it by rules plus hand-curated overrides (origins `derived` and
`curated`), and the monthly refresh publishes it only when it loads with
no fault. It loads like any vocabulary:

```sh
curl -fsSLO https://github.com/joshuasprow/nitrovin-data/releases/latest/download/epa-vehicles.jsonl
nitrovin --vocab epa-vehicles.jsonl translate --to epa-vehicles 1C6RR7LT5FS123456 | jq -c '.to[0].entries[].entry'
nitrovin --vocab testdata/vocab --vocab epa-vehicles.jsonl translate - <<'EOF' | jq -c '.to[0].entries[].entry'
{"from": {"vocabulary": "fleet", "entry": {"make": "chevy", "model": "silverado"}}, "to": ["epa-vehicles"], "model_year": 2004}
EOF
```

The VIN's four-wheel drive narrows the 2015 Ram to EPA's `1500 4WD`; the
fleet's `Chevy Silverado`, a name with no decode to check qualifiers
against, reaches every EPA name mapped to vPIC's Silverado for 2004.

## The demo set

The **demo set** is a small, invented catalog and the files around it, so
fitment, stock and translation can be seen without files of your own:
[`pkg/ui/demo/`](../pkg/ui/demo), every file named `demo-*` directly in it
(15 kB in all), and a second catalog of the same vehicles in `vpic/`.
It is built around popular vPIC models of the pinned data month (a Ford
F-150, a Chevrolet Silverado 1500, a Ram 1500, a Honda Civic and CR-V, a
Toyota Camry and RAV4) and two vehicles vPIC lacks. Every number
(`DEMO-…`), location (`demo-north`, `demo-south`), stock row and field
value in it is made up, and says so.

| File | Kind | What it shows |
| --- | --- | --- |
| `demo-part-types` | `spine` | the part types |
| `demo-part-codes` | `vocabulary` (part) | an invented parts system's codes on that spine (`DEMO-ROT` files every rotor) |
| `demo-vehicles` | `vocabulary` (vehicle) | the catalog's vehicles, mapped onto vPIC: `F150` is vPIC's `F-150`, `Silverado 1500` its `Silverado` of Series 1500, the Ram 1500 a Dodge to 2012 and a Ram from 2013 |
| `demo-classics` | `extension-set` | a Studebaker Champion and a Volkswagen Type 2, which vPIC lacks |
| `demo-catalog` | `catalog` | 46 fitments, release `DEMO-2`: qualified applications, sides, a redirect and a retired number |
| `demo-shelf` | `stock` (shelf) | numbered parts, one under a retired number and one with none, of release `DEMO-1`; its header declares `stock_number` and `condition`, the page's columns for a shelf row |
| `demo-yard-stock` | `stock` (yard) | whole vehicles, twenty by VIN and two by the catalog's vehicle; five Camrys and four 2017 RAV4s, so the page's "In stock" has an order; its header declares `stock_number` and `row` and leaves `color` out, so the page prints two columns and folds the colour under its row |
| `vpic/demo-vpic-catalog` | `catalog` (`"vehicles": "vpic"`) | the same vehicles and part types as vPIC's own, with no vehicle vocabulary: 46 fitments, release `DEMO-V1`, numbers `DEMO-V…`; applications to vPIC models (the F-150 is model 1801), to a make (every Toyota) and to `demo-classics`' extensions; qualified applications, sides, a redirect and a retired number. A server's alone, in place of `demo-catalog` (below) |

**In a browser**, the Vocabularies dialog of any `serve` with the browser
translator (release builds, and nitrovin.app) offers it as one **Load
demo**: the vocabularies load as one batch, the catalog compiles in the
browser as a catalog you pick does, and the shelf and the yard load
beside it, the yard's VINs alone going to the server to be decoded. It
loads in well under a second. One catalog per nitrovin holds: while the
server or the browser holds another catalog, the dialog says so instead.
**Remove demo** removes every file of it; after a new release whose
demo differs, the dialog offers **Refresh demo**. **On a server**, it
loads like any directory, for `/v1/fit`, stdio and MCP over the same data:

```sh
nitrovin --vocab pkg/ui/demo serve       # then ask localhost:8080/v1/fit
nitrovin --vocab pkg/ui/demo fit --part demo-part-types:alternator --stock shelf 1FTEW1EG7FFA12345
```

**The vPIC catalog.** `vpic/demo-vpic-catalog` names the demo's vehicles
as vPIC's own makes, models and extensions, as a catalog that knows vPIC
ids does. One catalog per nitrovin holds, so it has a directory of its
own and loads in place of `demo-catalog`: a directory loads only the
`*.jsonl` directly in it, so `--vocab pkg/ui/demo` never loads it, and the
spine and extension set it names, and the part vocabulary on that spine,
load by file beside it:

```sh
nitrovin --vocab pkg/ui/demo/vpic --vocab pkg/ui/demo/demo-part-types.jsonl \
  --vocab pkg/ui/demo/demo-classics.jsonl --vocab pkg/ui/demo/demo-part-codes.jsonl serve
```

Its pages name each vehicle in vPIC's words: the fit card's `F-150 /
Ford` for `/?from.vin=1FTEW1EG7FFA12345`; the catalog's lines, filtered
by one of make, model and ext
(`/ui/vocabularies/demo-vpic-catalog?where.model=1801`); and part-number
mode's vehicles, `vPIC model 1801 (F-150)` for
`/?from.fitment.type=alternator&from.fitment.number=DEMO-V1001`. The
shelf and the yard name `demo-catalog`'s numbers and vehicles, so it loads
without them. Load demo never offers it: `pkg/ui` embeds only the files
directly in `pkg/ui/demo/`, so no server publishes it
(`TestDemoVPICNotOffered`).

`nitrovin vocab check pkg/ui/demo` finds nothing in it on the pinned month,
nor does `vocab check` on the vPIC catalog beside the files it names, and
CI holds both there (`TestDemoCheck`), so a month that drops one of their
ids fails before a release ships it.

**A walkthrough.** On a server that loaded the set (the `serve` above),
these requests show one thing each.

From a VIN:

1. `/v1/fit?from.vin=1FTEW1EG7FFA12345`: a 2015 F-150 with four-wheel drive,
   from its VIN: its part types, each with its count of fitments.
2. `/v1/fit?from.vin=1FTEW1EG7FFA12345&part.spine=demo-part-types&part.id=alternator`:
   its three alternators, each with the catalog's own option text, verbatim
   (`3.5L V6 EcoBoost (VIN G, 8th digit)`): a person settles which.
3. `/v1/fit?from.vin=2HKRW2H53KH612345&part.spine=demo-part-types&part.id=transfer-case`:
   a 2019 CR-V, whose VIN carries no drive type, so the AWD fitment comes
   back with Drive Type not checked (`unconfirmed`).
4. `/v1/fit?from.vin=1FTEW1CG3FFA12345&part.spine=demo-part-types&part.id=brake-rotor-front`:
   a two-wheel-drive F-150: the 4WD rotor's qualifier rules it out.
5. `/v1/fit?from.vin=1GCVKREC6EZ123456&part.spine=demo-part-types&part.id=brake-rotor-rear`:
   a 2014 Silverado's rear rotors, listed under the front: fit follows
   the redirect and says so.
6. `/v1/fit?from.vin=2HKRW2H53KH612345&part.vocabulary=demo-part-codes&part.entry.code=DEMO-ROT`:
   a part by a parts system's code, translated onto both rotors.
7. `/v1/fit?from.vin=1FTEW1EG7FFA12345&part.spine=demo-part-types&part.id=alternator&stock=shelf`:
   shelf stock with its locations; `DEMO-S-02`, filed under the retired
   `DEMO-0901`, under its successor, marked; and the shelf's release
   `DEMO-1` marked as another than the catalog's (`catalog_mismatch`).
8. `/v1/fit?from.vin=4T1B11HK7JU512345&part.spine=demo-part-types&part.id=alternator&stock=yard`:
   the yard's vehicles the part would come off, matched by their VINs.
9. `/v1/fit?from.vin=1FTEW1EG7FFA12345&part.spine=demo-part-types&part.id=brake-rotor-front&stock=shelf&stock=yard&location=demo-north`:
   stock at one location.
10. `/v1/fit?from.vin=1C6RR7LT6FS512345`: a 2015 Ram 1500, which vPIC
    decodes to make Ram, model 1500, and the vehicle vocabulary maps to the
    catalog's `Dodge Ram 1500`.

From an entry of `demo-vehicles`:

1. `/v1/fit?from.vocabulary=demo-vehicles&from.entry.make=Toyota&from.entry.model=Camry&model_year=2018&part.spine=demo-part-types&part.id=fog-lamp&side=left`:
   the side filter keeps the left fog lamp and the one for both sides.
2. `/v1/fit?from.vocabulary=demo-vehicles&from.entry.make=Honda&from.entry.model=Civic&model_year=2018&related=true&stock=yard`:
   related vehicles, the CR-V sharing the Civic's alternator and starter,
   with the yard's count of each.
3. `/v1/translate?from.vocabulary=demo-vehicles&from.entry.make=Chevrolet&from.entry.model=Silverado+1500&model_year=2011&to=epa-vehicles`:
   the catalog's Silverado 1500 in [EPA's vocabulary](#epas-vocabulary),
   loaded beside it (`--vocab epa-vehicles.jsonl`): a catalog's vehicle
   in another vocabulary's names.

From an extension:

1. `/v1/fit?from.ext=ext:demo-classics/vw-type-2&model_year=1974`: a
   vehicle vPIC lacks, minted by the extension set, with its parts.

From a fitment's key:

1. `/v1/fit?from.fitment.type=alternator&from.fitment.number=DEMO-0901`: a
   retired number lands on its successor, `DEMO-1001`.
2. `/v1/fit?from.fitment.type=alternator&from.fitment.number=DEMO-5001`: a
   fitment's vehicles and years, a Camry and a RAV4.

## Keeping a vocabulary private

nitrovin reads vocabulary files from the paths you give it and sends them
nowhere (a file loaded in the page's Vocabularies dialog stays in that browser: only a VIN to
decode goes to the server), and an answer names a file only by its `name` and `version`
(`/v1/info` adds its hash and local path). nitrovin ships no third-party
vocabulary but EPA's, and the hosted [nitrovin.app](https://nitrovin.app)
loads none: there, the API's translate and list answer for vPIC alone, and
the page offers EPA's to load into your browser. A vocabulary
built from licensed names (a parts catalog, a marketplace's compatibility
data) is yours to build and load under your own licence, and stays on the
machines you load it on. A catalog stays on its server: `serve` neither
publishes it nor offers it to the browser.
