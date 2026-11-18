---
status: proposed
---

# The page is the VIN's diagram, and takes a VIN alone

`GET /`, nitrovin's one page for browsers, is the VIN decoder drawn as the
VIN's diagram: a plate of 17 cells under six groups, a card for each group, and
under them "Also known as", the vehicle's names in every vocabulary loaded on
vPIC's spine. It reads a VIN (`from.vin`) and nothing else. This was decided
and built on draft PR [#303](https://github.com/joshuasprow/nitrovin/pull/303),
whose description holds the spec (2026-11-17). The references behind it, and
what the page took of them, are in
[docs/research/vin-page.md](../research/vin-page.md) (section 9) and
[docs/research/textures.md](../research/textures.md). `/v1`, the CLI and MCP do
not change.

It amends four ADRs:

- **ADR 0013, the worker rule:** the worker answers only a request that
  "names a browser-held file (its `from.vocabulary`, its spine or a `to`)",
  so that a bare `/?from.vin=…` is the
  "server's answer and means the same in every browser".
  The page's URL names no file now. So the worker answers the page, with a VIN
  on the plate or without, whenever the browser holds a file, and leaves it to
  the server in a browser that holds none. A link that names only server files
  "means the same with JavaScript on or off, and in any browser"
  no longer holds for the page (see What it costs).
- **ADR 0013, targets:** the visitor's
  "selection is ticked into the forms and travels as an explicit `to=`",
  never implied. The page's target vocabularies are now implied: every
  vocabulary loaded on vPIC's spine. The selection and its checkboxes are gone;
  `to` stays the API's.
- **ADR 0013, the tabs and the map:**
  "Its controls (Translate from) are tabs" and "The answer is a map".
  The plate replaces the tabs and "Also known as" the map. The page has no
  form for a vocabulary's entry, a spine entry or a vPIC id, and draws no line
  for each mapping. A
  "refusal is said beside the field holding the value it names";
  it is now said below the plate, the page still, with its status.
- **ADR 0013, htmx:**
  "The UI is htmx over `/ui/*` routes outside the `/v1` contract".
  htmx no longer fetches the page's answers. `plate.js` does, sending
  `HX-Request` itself, so the route answers a fragment as before. htmx remains
  for the Vocabularies dialog's forms and the entry lists. Every page still
  works without JavaScript.
- **ADR 0013, old links:** `/ui/decode`, `/ui/translate`, `/vocab`,
  `/ui/vocabularies` and `/?vin=`
  "answer `301` to `/`, their query along".
  They still answer `301`, with the VIN they named and nothing else.
- **ADR 0013, the fit page:** its amendment of 2026-11-12 says ADR 0023
  "adds a fit page the worker"
  answers. The worker answers no fit page.
- **ADR 0018:** its amendment says ADR 0023
  "adds `/ui/fit`, whose URLs are `/v1/fit`'s parameters".
  `/ui/fit` is removed. `fit` is on every surface as before and has no page.
- **ADR 0023:** a visitor ran `fit`
  "on a `/ui/fit` page that the Service Worker answers",
  and of the pages it says
  "Every page is reachable from a navigation header."
  `/ui/fit` is no longer a route, no page in a browser draws a fit, and the
  header holds no navigation. The browser still loads a catalog and stock and
  compiles the catalog, and the fit code over them stays: "Parts that fit"
  returns as a card on this page in a later PR (vin-page.md, section 9). Until
  then the rule
  "The Service Worker answers the fit page only"
  when the browser holds the catalog has no page to apply to, and the map's
  "open question on the `/ui` pages for fitment is settled for pages"
  is open again, with that design.
- **ADR 0024:** a fitment key may name an entry wherever a key
  "is accepted (`fit`'s `from`, the batch, MCP and `/ui/fit`)".
  `/ui/fit` leaves that list.

## Why

The page before was a translation form: tabs, options, a map in three
columns and a second page for `fit` behind a navigation header. The spec asks
for a single page whose focus is the VIN's field, with the "What's in a VIN?"
diagram as the interface, working without JavaScript and live with it.

A VIN is 17 characters in fixed groups. That is what lets the page be drawn by
arithmetic, the same on the server and in the browser, at every width. Most of
the rules below follow from keeping that property, and from keeping the page's
URL to the VIN.

## The rules

- **Six groups, the standard ones:** maker (positions 1–3), vehicle (4–8),
  check digit (9), model year (10), plant (11), serial (12–17). The diagrams
  surveyed bracket four to eight. A group's key names its colour, its bracket
  and its card, so everything on the page hangs off this grouping.
- **The server draws, and the script only asks again.** The page answers
  whatever is typed so far, cleaned (letters and digits, upper case, 17 at
  most, never a letter swapped): the maker at three characters, the vehicle at
  8, the year and plant at 11, the check digit's verdict at 17. Fewer than
  three ask the decoder nothing. `plate.js` asks the page 150 ms after the last
  key, drops an answer still on its way, and swaps in what the server drew.
  There is one renderer, in Go, on the server and in the worker, so the page
  with JavaScript and without shows the same thing.
- **A fixed grid, and nothing measured to draw.** Labels, cells and brackets
  sit on one grid of 17 columns, the cards on six equal ones, and the leader
  lines are one SVG drawn to the grid's fractions. The CSP allows no inline
  style, so every position is a class. No script reads a position to draw
  anything, and nothing is redrawn on scroll or resize. The one exception:
  `plate.js` reads the cells' boxes to find the gap nearest a pointer, for the
  caret a click or a tap sets, the cells a mouse drags over, and the faint
  caret under the mouse.
- **One real field.** The plate is one field and one tab stop: an input below
  the cells without JavaScript, and with it a textarea laid over them, its
  characters spaced a cell apart. The keys, the selection and a paste are the
  browser's own, and the cells draw them. Where a pointer lands is the
  script's to say, because the browsers do not agree with the cells (Firefox
  halves a character at its glyph, a phone puts the caret at a word's end). A
  whole VIN takes no more.
- **The plate takes a VIN and nothing else.** The spec had the one field also
  take a vehicle ("2015 F-150") or a part number and guess which. It was
  dropped on 2026-11-17, never built: words of any length would give up the
  fixed cells, the two rows and the caret between cells. A lookup by name or
  part number, if it comes, is a control of its own.
- **On a phone the plate is two rows** (below 34rem), so a cell is wide enough
  for a finger. It splits after the check digit: nine cells above (maker,
  vehicle, check digit) and eight below (year, plant, serial). The grid becomes
  nine columns and the field's text wraps with the cells, which is why the
  field is a textarea. Below 46rem the leader lines go; the brackets and
  colours stay and the cards stack in the groups' order.
- **Colour means the group a thing came from.** A group has one colour, on its
  label, its cells' underline, its bracket, its line and its card, and on the
  words of a name that came from it: the make under the maker's, the rest of
  the entry under the vehicle's, the years a mapping holds for under the
  year's. A card's heading names its group and positions, so colour is never
  the only cue. Red is kept for what is wrong.
- **Doubt is drawn dashed, with the words behind a button.** A mapping matched
  by rule dashes the lines and marks into the make and the model; a qualifier
  or the model year that could not be checked dashes the model's or the year's.
  The words are a hover title and a button on the name that opens how it was
  reached. The survey found quality written in words at the value and never in
  a line's style (finding 2), and the spec said so too. The page departs from
  both knowingly.
- **The names are of what is loaded.** One stack for each vocabulary loaded on
  vPIC's spine, the server's and the browser's, always: a vocabulary with no
  name for the vehicle keeps its place and says so. Nothing is stored about
  targets. A browser that held files before keeps its `nitrovin-state`
  database, which is no longer opened: no migration.
- **The URL carries the VIN only.** `/?from.vin=…` follows what is typed, in
  place, and a whole VIN alone is a history entry. The decode's options
  (`model_year`, `shape`, `full`, `now`) are the API's and the CLI's; in the
  page's URL they are ignored, as any parameter it does not know is, where a
  bad one was refused. What the page asks of a VIN is always the plain decode.
  `/?surprise` answers `303` to a curated VIN.
- **Everything else is behind three header buttons:** VINs, Vocabularies and
  Developers. Each opens a popover, so it opens without a script.

## What it costs

- **A link no longer means the same in every browser.** `/?from.vin=…` shows
  the names of the server's vocabularies everywhere, and those of a browser's
  own files beside them only in that browser. ADR 0013 refused
  "With JavaScript on, browser files only: the same link means two things."
  That stays refused: the server's names are in every answer. But two people
  with one link may now see different lists. The alternative was a URL that
  names targets again.
- **Nothing a browser shows is a fit.** nitrovin.app ships no catalog, and of
  the fit page ADR 0023 says
  "so this is the only way it can"
  answer a `fit`. Until "Parts that fit", a visitor can load and compile a
  catalog and see none of its answers but through `/v1/fit` on a server that
  loads it. ADR 0023's browser timings were taken from BenchmarkFit's
  "64 fitments asked of the fit page through the Service Worker",
  and are not measured again until a page asks a fit.
- **A translation from anything but a VIN has no page.** From a vocabulary's
  entry, a spine entry or a vPIC id it is `/v1/translate`'s, the CLI's and
  MCP's. An old link to one lands on the empty plate.
- **Privacy is as it was.** A browser's files never leave it. What travels to
  the server is the VIN alone, to `/v1/decode`, as the visitor types.

## Considered options

- **A line for each target vocabulary,** as the map drew one for each mapping.
  None of the tools surveyed with more than about eight targets draws them
  (finding 2). Three lines, from the maker, the vehicle and the year, enter
  the top name of each stack, and the rest is a stack behind a count.
- **Seventeen one-character fields.** A tab stop and a screen-reader field
  each; the tools that replaced the one field lost paste (finding 3).
- **Overwrite typing,** a click picking a cell and a key replacing it. It was
  built first and replaced: it was an editing model of the script's own over a
  real field. Spacing the field's characters a cell apart gives the browser's
  caret, selection and keys instead.
- **Measuring the page to draw the lines.** The one reference that measures
  the DOM redraws on every scroll and resize and still loses its lines at 390
  pixels (finding 8). It would also be a drawing only JavaScript has.
- **Drawing partial VINs in the browser.** A second renderer to keep in step
  with the server's, as ADR 0013 refused a second translator.
- **A hidden clock parameter,** keeping `now` on the page so a test could pin
  it. The page would read something besides the VIN. The worker's tests blank
  the JSON's `now` before comparing the browser's page with the server's.
- **Ghost text or a suggestion list in the field.** The spec's typeahead is
  the maker named at three characters, and the recent VINs in the VINs menu.

## Consequences

- `CONTRIBUTING.md`, `README.md` and `docs/vocabularies.md` describe the page
  as it is, and `GLOSSARY.md`'s browser's catalog and browser's stock no
  longer name a fit page.
- The lookup tabs, the map, `/ui/fit` and their tests are deleted. The browser
  run in `cmd/nitrovin` types on the plate a key at a time, with JavaScript,
  and loads each step as a page without it.
- "Parts that fit" is a later PR on this page's patterns: counts that open
  rows in place, three levels at most, the open state in the URL.
