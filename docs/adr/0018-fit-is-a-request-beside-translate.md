# `fit` is a request beside translate and list

A loaded catalog (ADR 0016, ADR 0017) is reached through one new request,
`fit`, on every surface. This decision comes from
[Fitment requests](https://github.com/joshuasprow/nitrovin/issues/225), on
[the fitment and stock map](https://github.com/joshuasprow/nitrovin/issues/221).
It is additive under `/v1` (ADR 0004, ADR 0007) and amends ADR 0012, which named
only `translate` and `list`.

## The request

Like `translate`, `fit` is shaped by its inputs rather than split by direction:

| Field | Meaning |
| --- | --- |
| `from` | A vehicle (a VIN; a vocabulary entry, `{"vocabulary","entry":{…}}`; or a spine target, `{"make"}`, `{"model"}`, `{"ext"}`), or a fitment key `{"fitment":{"type","number","side"?}}` |
| `model_year` | The vehicle's year when no VIN gives one; a decoded year wins, as in translate |
| `part` | A part type: an entry of the catalog's part spine, `{"spine","id"}`, or of a part vocabulary on that spine, `{"vocabulary","entry":{…}}` |
| `side` | `left`, `right` or `both`, to filter on |
| `to` | Vehicle vocabularies to name vehicles in, as in translate; without it, vehicles come back as spine entries |
| `related` | `true` for vehicles that share fitments with `from` |
| `limit`, `after` | Paging, as in `list`: 100 by default, at most 1,000, with a cursor |

The inputs decide the answer:

| `from` | Also given | Answer |
| --- | --- | --- |
| a vehicle | nothing | its **part types**, each with a count of fitments |
| a vehicle | `part` | **candidates** |
| a fitment | — | its **vehicles** |
| a vehicle | `related` | **related vehicles** |

A vehicle reaches the catalog the way `translate` would reach the catalog's
vehicle vocabulary: from a VIN or a spine target through the spine, and from an
entry of another vocabulary through the spine and back. A vocabulary entry of the
catalog's own vehicle vocabulary is used as it is.

## Candidates

The answer has, at the top level:

- `vehicle`: the decode's make, model and year, when a VIN was given;
- `models`: the catalog entries the vehicle reached, so a person can be asked which
  one when there are several;
- the catalog's name, version and `release`;
- `candidates`, `total` and `next`.

Each candidate carries:

- `fitment`: type, number, side and `note`;
- `applications`: those that matched, each with
  - the catalog's vehicle entry;
  - `via`, the spine entries that led to it;
  - `first` and `last`;
  - `options`, verbatim;
  - `unconfirmed`: the qualifier element ids, and `"model_year"`, that applied but
    could not be checked, absent when everything was checked (as in translate);
- `reached`: `direct`, `redirect` (with the type asked for), or `retired` (with the
  number asked for).

An application whose qualifier fails against a decode is dropped. An unknown value
keeps it (ADR 0009). Candidates are ordered by part type, side and number, which is
stable. The catalog's own order is not kept, because not every catalog has one.

A request with no part never returns candidates. A whole vehicle measured 1,090
fitments and 148 KB; with a part type and a year it is about 5. Without a part, the
answer is the part-type summary.

A request with no year (and no VIN) is allowed. Every year's applications come
back, paged, with `"model_year"` in `unconfirmed`. Nothing is ever cut short
silently: every list pages, and `total` says how many there are.

## Vehicles of a fitment, and related vehicles

From a fitment key, the answer is the fitment and its applications. Each vehicle
is the catalog's entry translated into every `to` vocabulary, or into the spine
without `to`. A retired number answers its successor, marked `reached: retired`.

Related vehicles are the vehicles (a catalog entry and a year range) that share at
least one fitment with `from`. Each comes with its count of shared fitments per
part type. They are ordered by shared count, then name, and paged.

## Bulk

`list` with `of: "<catalog>"` pages through the catalog's fitments, 1,000 at most at
a time, as ADR 0012 pages files. It takes `where.type`. Each row is a fitment line
as the file has it.

`POST /v1/fit` takes an array of up to 1,000 requests and answers them in order, as
translate's batch does. Existence checks and lookups of the newest model, one
number at a time, go there.

## Nothing to fit is never an error

- An unknown vehicle, a vehicle the catalog has no entry for, or a part type with
  no fitment for that vehicle answers with a status and empty lists.
- With no catalog loaded, which is nitrovin.app's case, `fit` answers status
  `no-catalog`. `list` returns vPIC alone the same way.

Only a malformed request is refused, as `#invalid-request` with a pointer:

- a part vocabulary that isn't on the catalog's spine;
- `part` or `related` given with a fitment key;
- a side outside `left`, `right` or `both`;
- an unknown vocabulary;
- entry fields other than the vocabulary's;
- a fitment key without `type` and `number`.

## Surfaces

| Surface | `fit` |
| --- | --- |
| HTTP | `POST /v1/fit` (an array of up to 1,000, at most 1 MiB, answered in order); `GET /v1/fit` with dotted parameters (`from.vocabulary`, `from.entry.<field>`, `part.spine`, `part.id`, `side`, `to` repeated, `model_year`, `related`, `limit`, `after`) |
| stdio and C | `op: "fit"`, reply keyed `fit`; C answers it through `nitrovin_decode` |
| CLI | `nitrovin fit [--part spine:id] [--side s] [--to a,b] [--model-year N] [--related] <vin \| ->` |
| MCP | the tool `fit`, always listed, with a static description |
| Go | `Fitter.Fit`, beside `Translator`, over `fitment.Reader` (ADR 0017) |

The contract documents change in the same PR as the code that serves them.

## Budget

These are measured at the server on the catalog-scale fixture (ADR 0017), on the
reference machine, by the same CI benchmark as the load. All are p99:

| Request | Budget |
| --- | --- |
| Candidates for a vehicle, a year and a part type | at most 5 ms |
| The part-type summary | at most 10 ms |
| Related vehicles | at most 50 ms |
| Reverse | at most 1 ms per key |

A whole vehicle measured 2.7 ms p99 on today's engine, and one fitment to its
vehicles 1 µs. The first user asked only to beat today's 0.1–0.9 s p50. These
budgets leave a wide margin and still catch regressions.

## Considered options

- **Four requests, one per answer.** Each shape is plainer, but there are four shapes
  on six surfaces: the per-direction split ADR 0012 rejected for translate.
- **A kind of `translate`.** Translate's answer has no room for candidates, option
  text or paging.
- **Flat rows, one per application.** The same fitment would repeat across rows.
- **Whole-vehicle candidates.** About a thousand entries per vehicle. The part-type
  summary answers the question a caller actually has.
- **Refusing a request with no year.** Callers who know a model but not its year
  could then not page through its fitments.
- **Reverse only, leaving related vehicles to callers.** They would rebuild it from
  thousands of reverse answers.
- **`serve` only.** Every other request is on every surface, and a static MCP tool
  list depends on that.

## Consequences

- Stock answers ([#226](https://github.com/joshuasprow/nitrovin/issues/226)) attach to
  candidates and to related vehicles. That ticket decides how.
- [#239](https://github.com/joshuasprow/nitrovin/issues/239) builds the reader, and a
  request ticket builds `fit` on it.

## Amendment (2026-11-12): the fit page

[ADR 0023](0023-fitment-in-the-browser.md) adds `/ui/fit`, whose URLs are `/v1/fit`'s parameters, answered by the
server and, for a catalog the browser holds, by its Service Worker.

## Amendment (2026-11-12): related vehicles' ties

Related vehicles of equal names, years and shared count, as two vPIC models of one
name can be, come in the order the catalog first lists them among the shared
fitments: by part type, fitment and application
([#267](https://github.com/joshuasprow/nitrovin/issues/267)). The order among them
was unstated, and was whatever an unstable sort left.

## Amendment (2026-11-17): the side filter

The request table's `side` is "to filter on", and never said how. The rule, as the
API README states it: `left` keeps `left`, `both` and fitments with no side;
`right` likewise; `both` keeps `both` and fitments with no side; no `side` keeps
every fitment. A part for both sides goes on either, and a fitment that states no
side is never ruled out by one, as an unknown value keeps an application
(ADR 0009). `fitment.Reader` answers every side and `fit` applies the rule
([ADR 0017](0017-the-catalog-file.md)'s amendment,
[#292](https://github.com/joshuasprow/nitrovin/issues/292)).

## Amendment (2026-11-19): what an application's conditions confirmed

An application said what could not be checked (`unconfirmed`) and not what was. One
whose qualifier the decode met and one with no qualifier at all read the same:
neither had `unconfirmed`. A reader showing candidates to a person needs to tell
"the decode backs this" from "nothing was checked"
([#321](https://github.com/joshuasprow/nitrovin/issues/321)).

Each matched application gains `confirmed`, in `unconfirmed`'s type and wire form:
the element ids of the qualifiers a decoded value met, then `"model_year"` when the
years were checked and held: a known year is inside them, or with none known every
year the vehicle's entry spans is. It covers what `unconfirmed` covers: the
application's own qualifiers and years, its model's and its part type's. A
condition that applied is in exactly one of the two, and `confirmed` is absent when
empty. So an application with `confirmed` had conditions that were met, one with
`unconfirmed` had conditions that could not be checked, one with both had some of
each, and one with neither had no condition: only its option text describes the
fit.

- **Both fields, not one status.** A status per application ("checked", "possible")
  would decide for the reader what counts as backing. Most applications have years,
  so `"model_year"` is confirmed on most of them, and whether a year alone is enough
  is the reader's call. The lists let it look for an element id.
- **Several mappings.** A model or a part type reached by several mappings has
  `unconfirmed` empty when one of them checked everything (ADR 0012). `confirmed`
  follows the same mappings: those that checked everything, else all of them. A
  condition on a mapping that `unconfirmed` does not read is in neither field.
- **Unchecked wins.** A condition left unchecked anywhere on the route is
  `unconfirmed` and never also `confirmed`, so the two never contradict.
- **Additive.** Which applications an answer holds, their order, paging and
  `unconfirmed` are as they were. A failed condition still drops the application.
- **Fit's applications only.** A translation's entries and a mapping's route keep
  `unconfirmed` alone.

## Amendment (2026-11-19): a catalog's fitments for a vehicle

`list` of the catalog took only `where.type`, so a reader showing a person a
vehicle's lines in the catalog had to fit the vehicle and could not page the file
as it is ([#322](https://github.com/joshuasprow/nitrovin/issues/322)). Beside
`where.type`, `list` of the catalog takes the catalog's vehicle fields: its vehicle
vocabulary's fields, matched after that vocabulary's normalization, every one given
holding; or, when its vehicles are vPIC's, one of `make` and `model` ids and `ext`,
matched exactly. The lines are those `fitment.Reader.Matching` gives for the
matching vehicles, in key order with the same cursor, each holding only those
vehicles' applications in file order; no matching vehicle is an empty page.
`/v1/info` names the catalog's `spine` and `vehicles`, so a reader learns which
fields it takes without a fit.

- **Additive.** Without a vehicle field `list` of the catalog is as it was; any other
  field is still refused.
- **No years, no translation.** The fields name the catalog's own vehicles, as its
  lines do; reaching them from a VIN or another vocabulary stays `fit`'s work.
- **One vPIC field.** A vPIC vehicle is exactly one make, one model or one extension,
  so two of `make`, `model` and `ext` together could match nothing; the second is
  refused at its pointer rather than answered with an empty page. `make` keeps the
  make's own lines, not its models'.
- **`Matching`, not `ByVehicle`.** A line needs a fitment's key, note, filed-under
  entries and the matching applications, which `Matching` reads from the vehicle's
  block alone in a compiled catalog (ADR 0023); `ByVehicle` would read every far
  fitment whole on every page.

## Amendment (2026-11-20): a batch's stock rows once, compressed, with its times

[#371](https://github.com/joshuasprow/nitrovin/issues/371): once a batch's
fitment keys share one stock lookup (ADR 0020's amendment of the same day), the
answer's size is what is left. The first user's side-mirror batch, 44 fitment
keys: 3.86 MB, 3.44 MB of it yard rows, 8,467 rows for 769 vehicles, each row
whole in every list holding it. Cut to identity and `vin` (ADR 0019's
amendment of the same day) it is 1.03 MB; with each row sent once besides,
about 0.67 MB. Compressed, the whole answer is 380 KB as gzip level 6 and 47 KB
as brotli quality 4, in 0.01 to 0.02 s of CPU each.

- **Each stock row once, asked for by the batch.** `POST /v1/fit?stock_rows=once`
  answers a `FitBatchRows`: `fits`, the fits in request order, every stock list's
  `rows` indexes into `stock_rows`, the batch's distinct rows in the order the
  fits first list them. Expanded, it is the array of fits. It is the batch's
  option, not a request's, because only a batch has rows to share, and one item
  could not change the shape of every other's answer. A query parameter keeps the
  body the array of requests it was; `GET /v1/fit` refuses it, a single fit
  having no batch, and so does every other surface, which answers one fit a call.
- **A distinct answer, opt-in.** Without the option the answer is the bare array,
  byte for byte as before (`TestFitBatchStockGolden` holds a batch's bytes from
  before #371), so no caller's integration changes when nitrovin upgrades; with
  it, the answer is an object, so a caller decoding into types meets one shape
  per request, never an array that might hold indexes.
- **A row is its kind, source and id, and all of it.** Two lists holding a row
  alike share its index; a shelf row and a yard row never do; a row two lists
  hold differently, cut by different `stock_fields` or reached through different
  fitments, is two rows, so no fit receives a field it did not ask for. Totals,
  paging and every `sources` block are as without the option.
- **In Go, types of its own.** `FitRefs`, `FitCandidateRefs` and
  `RelatedVehicleRefs` embed `Fit`, `FitCandidate` and `RelatedVehicle` and
  replace their stock lists with `StockRefs`. A `MarshalJSON` on `StockList`
  writing either rows or indexes was the smaller change, and was measured
  doubling the marshalling of every stock answer, with the option or without
  (3.1 to 6.3 ms for 3.6 MB of rows): `encoding/json` compacts again whatever a
  marshaller returns. The price is the order of members: a `FitRefs` writes its
  lists after the `Fit`'s other members.
- **Compressed answers, in `serve` alone.** `GET` and `POST /v1/fit` answer
  brotli when `Accept-Encoding` allows `br`, else gzip when it allows `gzip`,
  else nothing; a body under 1 KiB goes as it is, as a frame and its CPU cost
  more than they save there. Refusals follow the same rule, every `/v1/fit`
  response varies by `Accept-Encoding`, and `Content-Length` is what is sent.
  The JSON is unchanged: compression is HTTP's, so the other surfaces, stdio and
  the C library among them, answer exactly as before. Brotli is
  `github.com/andybalholm/brotli`, a pure-Go port, so the binary stays
  `CGO_ENABLED=0`; it was a test dependency already. Other routes are left as
  they were: their answers are small, or (the spine bundle) compressed already.
- **`Server-Timing` on every fit answer:** `fit`, the contract's whole answer,
  stock lookups included; `stock`, the lookups summed over every source, timed
  where every lookup passes (`contract.StockTime`, given in the fit's context);
  `encode`, marshalling and compressing. Nothing else, so a caller learns no
  source, connection or relation name, and none of it enters an answer.
- **Measured** on the reference machine, on a synthetic yard shaped as the
  mirror: 769 vehicles of about 500 bytes of `fields` each (hashes for photos,
  VINs and stock numbers, so they do not compress as a template would), against
  the synthetic catalog's four fitment keys repeated to 44, `stock: ["yard"]`,
  `limit` 192: 8,448 rows. The contract's batch took 2.4 ms from the files; 3.3 ms
  cutting `stock_fields`, 5.2 ms with the rows once, 6.2 ms with both.

  | Answer | JSON | Marshal | brotli q4 | gzip 6 |
  | --- | --- | --- | --- | --- |
  | As before | 4.49 MB | 8.1 ms | 19.9 KB, 9.2 ms | 661 KB, 9.8 ms |
  | `stock_fields: ["vin"]` | 1.55 MB | 5.8 ms | 4.4 KB, 3.4 ms | 128 KB, 2.6 ms |
  | `stock_rows=once` | 216 KB | 0.5 ms | 21.0 KB, 1.6 ms | 22.7 KB, 0.5 ms |
  | Both | 123 KB | 0.4 ms | 4.7 KB, 0.9 ms | 5.7 KB, 0.2 ms |

  The levels, on the answer as before: brotli quality 1 is 216 KB in 2.9 ms, 4 is
  19.9 KB in 9.2 ms, 5 18.6 KB in 9.9 ms, 6 18.5 KB in 9.6 ms, 11 16.4 KB in 216
  ms; gzip level 1 is 772 KB in 6.3 ms, 6 661 KB in 9.8 ms, 9 640 KB in 43.9 ms.
  Quality 4 is the first to find the repeats across lists, ten times smaller
  than quality 1, and later qualities buy a few percent for more CPU; level 6 is
  gzip's default, and its 32 KB window cannot reach those repeats at any level.
  Brotli at quality 4 and gzip at 6 are `serve`'s constants.
- **What is left** is the database's read (0.5 to 0.9 s on the mirror, ADR 0020)
  and, at about 0.4 MB of the mirror's answer, each fitment key's catalog
  `vehicles`, which a caller paging only stock still receives. A request option
  leaving them out is a separate change, when one is wanted.
