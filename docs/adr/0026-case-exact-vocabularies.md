# Case-exact vocabularies

A vocabulary may declare that its entries match without folding case, for a system
whose codes differ only in case and name different things
([Checks at scale](https://github.com/joshuasprow/nitrovin/issues/295), on
[the fitment and stock map](https://github.com/joshuasprow/nitrovin/issues/221)).
This amends two ADRs:

- **ADR 0009:** an entry matches "exactly after one fixed normalization of case and
  spaces". A case-exact vocabulary's entries match after the same normalization of
  spaces, and none of case.
- **ADR 0021:** compiled format 5 records the case mode it was compiled with.

## Why

In one of the first user's code vocabularies, 23 keys are shared by 50 codes that
differ only in case and name different vehicles; one pair names a truck and the
other a motorcycle. Under ADR 0009's normalization they are one entry. So the first user
leaves them unmapped and cannot curate them. Another of its code vocabularies, of
257 codes, has no such collision and stays as it is.

## The rule

A vocabulary's header may declare `"case": "exact"`. It covers all the vocabulary's
fields. The header field is optional, so JSON Lines stays format 1; a release that
predates it refuses a file that declares it, as an unknown field.

In a case-exact vocabulary:

- **An entry matches** after trimming and collapsing white space, as ADR 0009 has
  it, and without folding case. `old3` and `OLD3` are two entries.
- **A duplicate entry** is one equal after that normalization. Two entries that
  differ only in case are not duplicates.
- **A spelling the vocabulary does not hold is unknown**, never a guess between two
  entries that fold alike.

Everything that names its entries follows the vocabulary's mode, declared once:

- a translate or list request that names an entry of it;
- a catalog whose vehicle vocabulary, or filed-under vocabulary (ADR 0024), it is:
  its applications', redirects' and fitments' entries;
- a stock row that names its entries.

A catalog declares nothing of its own.

## The compiled form

Compiling decides which spellings are one vehicle or one entry: folded, `old3` and
`OLD3` are one; exact, two. So **compiled format 5** records the case mode of the
vehicle and filed-under vocabularies it was compiled against. Loading it beside a
vocabulary of the other mode is a fault naming the vocabulary and both modes.
Format 4 files are refused with "recompile", as earlier formats are.

## Considered options

- **Re-keying such codes in the author's export.** It needs no change here, but the
  codes are the parts system's own: every caller would have to learn the re-keyed
  spelling, and a lookup by the system's code would miss.
- **Per-field case modes** (`"case_exact": ["code"]`). More general, and no known
  vocabulary needs one field exact and another folded.
- **The catalog declaring the mode too.** It says again what its vocabulary says,
  and gives the two a way to disagree.
- **No new compiled format:** keeping exact spellings apart at compile and folding at
  load when the vocabulary folds. Every catalog would hold more vehicles, and a
  compiled file's answers would depend on which mode was loaded beside it.
- **Case-exact by default.** It would change every existing vocabulary's matching.

## Consequences

- `docs/api/vocabulary.schema.json` gains the header field. `docs/vocabularies.md`
  documents it with the normalization it keeps.
- The synthetic fixtures gain a case-exact vocabulary with two entries that differ
  only in case.
- A build ticket on [the map](https://github.com/joshuasprow/nitrovin/issues/221)
  carries the work, with ADR 0025.

## Amendment (2026-11-17): as built

Building it ([#300](https://github.com/joshuasprow/nitrovin/issues/300))
settled what this ADR left open:

- **A JSON Lines catalog is read after the other files** a load opens, so
  its vehicles and entries are keyed as their vocabularies match from the
  first line; compile takes the mode from the files beside it. A catalog
  read alone (Go's `vocab.Read`) is read folding case, and is refused
  beside a case-exact vocabulary, as a compiled one is.
- **The mismatch fault** names the vocabulary and both ways, at the
  header's field: `/vehicles: vocabulary "codes" folds case, and the
  catalog was compiled against it matching case exactly: compile it again
  beside the vocabulary` (and `/filed_under: …` alike). Format 4 and older
  files are refused with the message to recompile.
- **Any `case` other than `exact`** is a header fault: `/case: unknown case
  "Exact": exact, or no case to fold it`.
- **A fit follows the mode end to end**: the request's vehicles carry their
  vocabulary's identity, so a candidate's applications, related vehicles,
  narrowing by a `filed_under` entry, a `filed_under` key and stock rows
  (a file's and a connector's lists) all keep `OLD3` and `old3` apart.
- **Extension and vPIC names** still match folding case: the mode is a
  vocabulary's alone.

### Measured (2026-11-17)

Format 5 adds two strings to the meta section; on the catalog-scale
fixture (folding case) compile took 4.3 s and 188.2 MB at peak and
`vocab check` 4.7 s and 191.0 MB, one core (`GOMAXPROCS=1`), Apple M5 Pro.
