# Checks at scale: a streaming catalog check, drift beside a fault, the entry a finding names

`nitrovin vocab check` must check a catalog at the first user's scale in bounded
memory, keep reporting drift when one file of many has a fault, and say which entry
each finding is about ([Checks at scale](https://github.com/joshuasprow/nitrovin/issues/295),
on [the fitment and stock map](https://github.com/joshuasprow/nitrovin/issues/221)).
This amends three ADRs:

- **ADR 0011:** a fault in one file no longer hides every file's drift from `vocab
  check`. Loading still refuses every file on any fault.
- **ADR 0017:** `vocab check` reads a JSON Lines catalog by streaming, not by loading
  it whole.
- **ADR 0021:** compile's streaming check is also `vocab check`'s.

## Why

The first user's counts:

- **Memory.** `vocab check` on a catalog of about 792,000 fitments peaks at 945 MB,
  because it loads the catalog whole. `vocab compile` runs the same checks by
  streaming, at a peak of 172 MB. So the first user's hourly job uses compile as its
  check and throws away a 315 MB output. `-o /dev/null` does not work, because the
  temporary file is written beside `-o`.
- **Drift beside a fault.** A fault in any file prints only faults. In a fault drill,
  one malformed line in one vocabulary emptied a 634-row drift queue until the next
  run, because the queue is replaced with each run's findings.
- **Which entry.** A finding names a file and a line. The first user's export reads
  every exported line again to learn which entry each finding is about, and an
  extension's findings can only be matched by file and line.

## A streaming catalog check

`vocab check` checks a JSON Lines catalog with compile's own streaming pass and
writes the compiled bytes nowhere. It reports what compile reports, faults and
drift alike. It sorts in scratch under `TMPDIR`, as compile does: about twice the
compiled file on disk, deleted when it finishes. Its peak memory must stay within
512 MiB at about 792,000 fitments; compile measured 172 to 185 MB there. A compiled
catalog is checked as before.

One command checks. A check-only mode on `vocab compile` would leave two ways to
ask the same question.

## Drift beside a fault

`vocab check` measures drift for every file that reads and links cleanly, even
when another file has faults. A file is **unchecked** when it has faults, or when it
links, directly or through other files, to a file that has faults:

- a catalog to its spine, vehicle vocabulary and filed-under vocabulary;
- a part vocabulary to its spine;
- a vocabulary or extension set to the extension sets it references;
- a stock file to its catalog.

A file linked to an unchecked file is unchecked too: its links are incomplete, and
anything measured against them could be false drift.

Every unchecked file gets one line, the faulty ones included, with a third
severity:

```json
{"file":"…/catalog.jsonl","line":0,"severity":"unchecked","detail":"its drift was not measured: it links to …/models.jsonl, which has faults"}
{"file":"…/models.jsonl","line":0,"severity":"unchecked","detail":"its drift was not measured: it has faults"}
```

A tool that keeps findings keeps an unchecked file's last drift rather than
clearing it. Exit codes are unchanged: 1 on any finding, unchecked lines included.
Loading (`--vocab`, `serve`) still refuses every file when any has a fault: a
server answers from all its files or from none.

## The entry a finding names

A finding about a vocabulary's line carries `entry`: that entry's fields as written.
A finding about an extension set's or a spine's line carries `id`. This applies to
faults and drift alike, on every surface that writes findings (`vocab check`,
`vocab compile`, load's stderr). A line that did not read carries neither. Catalog
and stock findings keep their line and key, and carry neither.

```json
{"file":"…/models.jsonl","line":42,"severity":"drift","detail":"/mappings/0/model: vPIC model 88888 is not in data month 2026_11","kind":"model","entry":{"code":"0003"}}
```

The finding format is additive: a reader that ignores unknown fields keeps working.

## Considered options

- **A check-only mode on `vocab compile`.** It is the smallest change, but checking
  would then be two commands, and `vocab check` would stay unusable on a large
  catalog.
- **A separate streaming checker that sorts nothing.** It would use no scratch disk,
  but would need a second implementation of every whole-file fault (duplicate keys,
  retired numbers), and its memory would grow with the catalog's keys.
- **Reporting only files linked directly to a faulty file as unchecked.** Drift
  measured against a file whose own links are incomplete can be false.
- **One summary line for every unchecked file, or a drift kind `unchecked`.** One line
  per file groups by severity as faults and drift already do. A drift kind would say
  the opposite of what it means: nothing was measured.
- **Naming, on catalog findings, the vocabulary entry they are about.** A catalog's
  line is a fitment, not an entry. Its line and key identify it, and the vocabulary
  entry is already in the detail.
- **Measuring drift beside a fault on load too.** A server refuses faulty files
  whole; it has no use for their siblings' drift.

## Consequences

- `docs/api/README.md` (`vocab check`) and `docs/vocabularies.md` (Checking)
  describe the streaming check, the scratch it needs, `unchecked`, `entry` and `id`.
- `vocab check`'s peak memory on the catalog-scale fixture is held by a benchmark,
  as compile's is.
- A build ticket on [the map](https://github.com/joshuasprow/nitrovin/issues/221)
  carries the work, with ADR 0026.

## Amendment (2026-11-17): as built

Building it ([#300](https://github.com/joshuasprow/nitrovin/issues/300))
settled what this ADR left open:

- **A `vpic` catalog links to every extension set** checked beside it. Its
  vehicles may name any of them, and which ones is known only once the
  catalog has streamed by.
- **A stock file's links** are a shelf's catalog, the catalog when a yard's
  rows name a part or a catalog vehicle, and the part vocabularies and
  spines its rows name.
- **Stock beside a JSON Lines catalog.** A stock file's check reads its
  catalog by number and by vehicle. When a stock file to be checked links to
  the JSON Lines catalog, the catalog is compiled into the check's scratch,
  not to nowhere, and read there as a loaded compiled catalog; its findings
  are still the streaming pass's. Memory stays the compile's.
- **A second catalog** among the arguments is a fault, `/kind: … is a
  catalog too: a deployment loads one catalog`, as loading has it, so
  only one is checked.
- **The unchecked line's detail** is one of `its drift was not measured: it
  has faults`, `… it links to <file>, which has faults`, or `… it links
  through <file> to <file>, which has faults`, naming the faulty file and,
  when the link is not direct, the file it is reached through. It comes
  after the file's faults.
- **A file whose header does not read** has no name, so a file naming it
  gets the fault that it is not loaded, as before.
- **Findings come file by file**, in the order of the arguments, each
  file's faults in line order.
- **`entry` is on a fault about a line that read**: a duplicate entry, an
  extension reference or spine id no loaded file defines; `id` on a
  duplicate id and an extension's unresolved make. A line refused as it
  reads (malformed, a bad mapping) carries neither.

### Measured (2026-11-17)

`bench/vocab` on the catalog-scale fixture, one core (`GOMAXPROCS=1`),
Apple M5 Pro: `vocab check` 4.7 s and 191.0 MB at peak, against the
budget of 512 MiB; `vocab compile` 4.3 s and 188.2 MB.
