# The codes a fitment is filed under

A catalog may say which of a system's own part codes each fitment is filed under,
and `fit` can narrow by those codes and look a fitment up by one
([Which of a system's part codes a fitment is filed under](https://github.com/joshuasprow/nitrovin/issues/289),
on [the fitment and stock map](https://github.com/joshuasprow/nitrovin/issues/221)).
This amends four ADRs:

- **ADR 0016** rejected "the system's own codes as a fitment attribute" and left
  the codes to a part vocabulary. This ADR takes them up as an attribute. They are
  never part of the identity, so a fitment is still one part.
- **ADR 0017** gains a header field, a line field, four faults and two kinds of drift.
- **ADR 0018** gains a narrowing, a lookup, a status and a field on answers.
- **ADR 0021**: the compiled form moves to format 4.

## Why a part vocabulary is not enough

A part vocabulary says which codes belong to a part type. It cannot say which of
those codes a given fitment is filed under. In the first user's catalog:

- 40 part types are split over 2 to 8 of the system's codes;
- 205,594 of 792,302 fitments are filed under fewer codes than their part type has;
- the vocabulary alone implies 1,127,093 (code, fitment) pairs, while the system has
  855,332.

In that system, the year, part code and number are the nearest thing to a
fitment's identity in daily use: stock, prices and sales are filed by them. A caller
holding (code, number) must reach the fitment, and an answer must say which codes
the fitment is filed under, or a reader cannot join it back to what is on the
shelf. A `note` is passed through and never interpreted, so nothing could check it
or look anything up by it.

## In the file

The catalog header names the part vocabulary, and each fitment line lists its
entries:

```json
{"format":1,"kind":"catalog","name":"parts-catalog","version":"…","spine":"part-types","vehicles":"catalog-models","filed_under":"system-parts"}
{"fitment":{"type":"300","number":"51014A","side":"left"},"filed_under":[{"code":"D1"},{"code":"D2"}],"applications":[…]}
```

- **One vocabulary, in the header,** as `spine` and `vehicles` are. It must be a
  part vocabulary on the catalog's spine and loaded beside it. A second system's
  codes on the same catalog would be a format change, made when someone needs it.
- **Entries are written by their fields,** as an application's `vehicle` is.
- **They are on the fitment, not on its applications.** The list is the union of
  the codes over the fitment's applications. In the first user's catalog,
  1,102 of the 27,726 fitments under two or more codes have an application not
  filed under all of them: 7,175 of 2,115,865 applications (0.34%) are listed under
  a code the system does not use for that vehicle and year. The part and its
  number are the same, so a stock join by (code, number) still finds the same part.
- **Redirects and retired numbers carry no codes.** They stay keyed by part type.

JSON Lines stays format 1, because both fields are optional additions (ADR 0016
allowed new fields in format 1). A release that predates them refuses a file that
uses them, as an unknown field.

### Faults

- a line's `filed_under` in a catalog whose header names no vocabulary;
- a header vocabulary that isn't loaded, or isn't a part vocabulary on the catalog's
  spine;
- the same entry twice on one fitment;
- an entry whose mapped part types don't include the fitment's part type.

The last check is a fault because the catalog's pairing of code and part type is
the author's own statement, and a vocabulary that contradicts it would send a
narrowed request to the wrong fitments. In the first user's system, every code sits
under exactly one part type.

### Drift

- **An entry the vocabulary lacks.** As with an unmapped vehicle entry, curation
  lags the catalog.
- **A fitment filed under no entry** in a catalog that names a vocabulary. The
  first user's count is 0, and it is kept as an alarm.

`/v1/info` names the vocabulary and counts (entry, fitment) pairs beside the
catalog's other counts.

## In requests

**Narrowing.** When a request's `part` is an entry of the catalog's `filed_under`
vocabulary, direct candidates are only the fitments filed under that entry.
Candidates reached through a redirect are not narrowed, because they are filed
under another part type's codes. A `part` from any other part vocabulary expands to
its part types as before.

**Lookup.** A fitment key may name an entry instead of a part type, wherever a key
is accepted (`fit`'s `from`, the batch, MCP and `/ui/fit`):

```json
{"from":{"fitment":{"filed_under":{"code":"D1"},"number":"51014A"}}}
```

- **`side` is optional and, when absent, means any side,** unlike the type key,
  where absent means none. Callers holding a code and a number usually have no
  side.
- **One fitment reached** is answered as its key would be.
- **Two fitments reached** answer status `ambiguous`, listing both keys with their
  sides, and the caller asks again with a side. In the first user's catalog,
  851,415 (code, number) pairs exist, 3,917 reach two fitments, and all of those
  differ only by side.
- **No live fitment:** the entry maps to its part types through the vocabulary,
  and each (type, number, side) is looked up among retired numbers, answered as
  `reached: retired`.
- **An entry the vocabulary lacks, or no fitment** is a status with empty lists,
  never an error, as ADR 0018 rules for nothing to fit.
- **An entry key on a catalog whose header names no vocabulary** is refused as
  `#invalid-request`, with a pointer.
- **The budget** is reverse's: 1 ms per key at p99 on the server, 3 ms in a browser
  (ADR 0023).

**Answers.** Every candidate, every fitment answered by key, and every row of
`list` with `of: "<catalog>"` carries the fitment's `filed_under` entries, when the
catalog names a vocabulary. Without them a reader could not join an answer back to
stock filed by code.

**Stock** (ADR 0019) is unchanged. A shelf source maps its own code to the fitment
key, because each code has one part type and side comes from the catalog. No
shelf row of the first user's reaches two fitments by (code, number).

## In the compiled form

`fitment.Reader` gains a lookup by (entry, number, side). Compiled format 4
carries each fitment's entries and an index on (entry, number). `--vocab` refuses
a format 3 file with "recompile", as format 1 and 2 files are refused.

## Considered options

- **Keep ADR 0016's rule and leave the pairing outside nitrovin.** Every caller
  would keep its own (code, number) → fitment table beside the catalog, and only
  the catalog's author knows the pairing.
- **Codes as part of the identity.** One part becomes several fitments, which ADR
  0016 rejected, and it still is.
- **Several vocabularies, listed in the header or named on each line.** No second
  system needs it yet; naming the vocabulary on each line repeats it 792,302 times.
- **Codes on each application, or on the fitment with a per-application subset.**
  It is exact for the 0.34% above, at the cost of a second place to check, compile
  and answer from.
- **An absent side in a code lookup meaning none,** as the type key has it. Callers
  holding a code rarely hold a side, so most lookups of a sided part would miss.
- **Answering both fitments of an ambiguous lookup together,** each vehicle naming
  its fitment. One answer would then mix two parts' vehicles.
- **Codes on redirects,** so a code request follows only its own. All 103,086 of the
  first user's redirects carry a code, and 48% are on a split part type, but a
  redirect only leads to another type's candidates, which are not narrowed anyway.
  This can be revisited if overstated redirects turn up.
- **A (code, number) route for shelf rows.** A source can name the key itself, and
  none of the first user's shelf rows is ambiguous.
- **Answering codes only when asked.** It saves a few bytes per candidate and breaks
  the join for every caller that forgets to ask.
- **A type mismatch as drift.** The file would load, and narrowed requests would
  quietly answer from a vocabulary that disagrees with the catalog.
- **A new JSON Lines format.** Both fields are optional additions, which format 1
  allows.

## Consequences

- `docs/api/vocabulary.schema.json`, `openapi.yaml`, the API README and
  `mcp-tools.json` gain the fields, the key form and `ambiguous` in the same PR as the
  code that serves them.
- The synthetic catalog in `testdata/vocab/` gains a filed-under vocabulary, a part
  type split over codes, and a number on two sides under one code.
- A build ticket on [the map](https://github.com/joshuasprow/nitrovin/issues/221)
  carries the work.

## Amendment (2026-11-16): as built

Building it ([#291](https://github.com/joshuasprow/nitrovin/issues/291)) settled
what this ADR left open:

- **A fifth fault:** an entry named by fields other than the vocabulary's, as a
  vehicle named by other fields than its vocabulary's is.
- **An entry the vocabulary lists without a mapping** is drift, like one it
  lacks: the vocabulary does not say where it goes, which is curation still to
  come, not a contradiction.
- **Order:** a fitment's entries are kept and answered sorted by their fields
  as written, field by field in name order, each value in byte order, and each
  is spelled as the file first writes it, as vehicles are. Two files that list
  them in other orders answer the same.
- **The lookup tries the catalog first.** A key whose entry the vocabulary
  lacks still reaches the fitments the catalog files under it; it is a status
  only when nothing is filed under it. A code new to the parts system reaches
  the catalog before the vocabulary's next build, and only drifts meanwhile.
- **A retired number by code** is looked up on each side of each of the entry's
  part types: in a compiled catalog, a read of filed and up to five more, about
  1.5 ms where a read costs 0.25 ms, within the browser's 3 ms for a key.

## Measured (2026-11-16)

`bench/vocab` on the catalog-scale fixture, now filed under 257 codes, 40 part
types split over two or three of them, 856,069 (code, fitment) pairs; one core
(`GOMAXPROCS=1`), Apple M5 Pro:

| | Format 4 | Budget |
| --- | ---: | ---: |
| Compiled file | 360.9 MB | 500 MB |
| Open | 5.4 ms | 1 s |
| Retained heap | 3.8 MB | 256 MiB |
| Peak RSS, loading | 20 MB | 512 MiB |
| Compile | 4.3 s, 185.5 MB peak | |
| A code and number, p99, compiled | 50 µs, 2 reads | 1 ms |
| A code and number, p99, JSON Lines | 31 µs | 1 ms |

The JSON Lines form of the same fixture loads in 3.2 s with 1,267 MB at peak,
inside ADR 0017's 15 s and 1.5 GiB.
