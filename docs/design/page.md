# The page as built

The page at `/` (`pkg/ui`) looks like a stamped VIN plate with tags hung from
it. This file states which parts of that look are the design, and which parts
are settled enough to leave alone. It describes; the code is the source:
[ui.css](../../pkg/ui/static/ui.css), [page.html](../../pkg/ui/page.html),
[plate.js](../../pkg/ui/static/plate.js),
[brushed.svg](../../pkg/ui/static/brushed.svg) and
[paper.svg](../../pkg/ui/static/paper.svg).
The references behind the choices are in
[docs/research/vin-page.md](../research/vin-page.md) and
[docs/research/textures.md](../research/textures.md).
[DESIGN.md](../../DESIGN.md) is this file's map.

Each part has one of two standings.

| Standing | Parts | What it means for a change |
| --- | --- | --- |
| **Signature** | The VIN input, and the four materials: aluminum plates, vinyl tape, manila tags, thermal paper | These are the page's identity. New UI is built from them, and a change keeps every rule below. |
| **Good enough** | What a decoded card holds, where the brackets and leader lines run, the dialogs, and the forms and inputs | These work and stay as they are. A change fixes a bug or follows a signature part; it does not rearrange them. |

## Ground rules

- **No inline style or script.** The page's CSP refuses both. Every position is
  a class in the one stylesheet, on a fixed grid.
- **No raster images.** A texture is a gradient, a shadow, a mask, or
  `feTurbulence` noise in an SVG (`brushed.svg`, `paper.svg`).
- **The page works without JavaScript.** The server draws the whole answer for
  whatever VIN the URL carries. Scripts only make it live.
- **Popovers, not scripted overlays.** Menus, dialogs and notes are
  `popover` elements opened by `popovertarget`.
- **Two colour schemes, four fixed materials.** The ground, the text, the
  dialogs and the group colours change between light and dark. The metal, the
  tape, the manila and the thermal paper do not: each is the same object
  under either light.
- **Three heights: stuck, raised and pressed.** A static strip of tape is
  stuck on, with a small tight shadow, `--lift-stuck`. A control that is not
  pressed stands higher than anything static beside it, and every such
  control stands equally high, whether metal or tape: a 3 px ledge of
  `--steel-dark` under a wide soft shadow, `--lift-raised`. A pressed control
  is flush: 2 px down, no drop shadow, a visibly darker fill, and an inner
  shadow, `--pressed`. A drop shadow disappears on the dark ground, so there
  the ledge tells raised from static, and the fill and the inner shadow tell
  pressed from raised; the soft shadow is only deeper there. A control is
  pressed while it is held, while it is on (a switch, or
  `aria-pressed="true"`), and while the menu it opens hangs from it.
- **Motion marks a change: a stamp, a draw and a pop**, each under 0.4 s, and
  none of it under `prefers-reduced-motion`. A typed character stamps in; a
  finished group draws its bracket and line and pops its card. The only
  motion that loops is the caret's blink on the focused plate, once a
  second; under `prefers-reduced-motion` the caret stands still. What a
  scroll moves, a row's fade and the lines a narrow page carries, is tied
  to the scroll and not to a clock, so it stays.

## Composition rules

The ground rules say what the page is built with. These say how a change
is put together, and each came from a correction: a version that was built,
looked at, and sent back. A change is checked against every one before it
is shown. One that breaks a rule says which, and why, where it is proposed.
Where the maintainer later changed a rule knowingly, the rule says so after
*Ruled*, with the date.

1. **Nothing floats.** A fact about the vehicle, an answer and a control
   each sit on a material: the plate, a strip of tape or a tag. Bare text on
   the ground beside them reads as a note left on the page, not as part of
   it. The ground holds only what frames the page: the masthead, a
   bracket's label, and the sentence of an empty place on its blank strip.
   *Sent back: "This vehicle" and its facts as text under the picker. They
   went onto the vehicle's block.*
2. **A material says what a thing is.** The plate is what is typed. Label
   tape is the page's own word on something: a stamp, what was asked, a
   tag's heading. A manila tag is a thing found: a card, a name, a
   candidate, a part, a vehicle a part fits. Thermal paper is what a stock
   file says of a thing found: a print, and not the thing. Metal is a
   control. Pinstripe tape is a connection. Before content is placed, it
   is one of these, and it takes that material.
   *Sent back: names as strips of label tape. A name is found, so it is a
   tag.*
   *Ruled, 2026-11-23: in part-number mode a vehicle's names are
   underlined ink on that vehicle's tag. The table panel that set them on
   label tape is gone, which ends the contradiction between this rule and
   the second mode's section.*
3. **Colour and texture before words.** A group's colour says which card a
   word comes from, so no label repeats it. A label stays only where
   nothing else says what a value is, as "Body" and "Engine" do. Nothing
   in one view says the same thing twice.
   *Sent back: "VIN 4–8" over five characters that each carry their
   position; "Make" and "Model" over words already underlined in the
   Maker's and the Vehicle's colours.*
   *Ruled, 2026-11-24: a VIN's problems are said where the reader works,
   in more than one place at once: the vehicle's block's stamp, the
   struck cell, the stamp at the candidates' head, an empty card, and the
   strip under the answer (#398). A part pulled for a mistyped VIN is the
   trade's expensive error. It is a knowing exception to "nothing says
   the same thing twice".*
4. **One thing leads; the rest sit on it.** A block has one primary thing.
   What serves it sits on the same piece of material, smaller, after it,
   and is dropped first when the window narrows. It never gets a block of
   its own beside the primary.
   *Sent back: recent part types as a strip of their own under the picker;
   then as a second row on a wide tape, most of it bare. A wide tape
   became one line, and a narrow one kept a second line for them, the key
   as wide as it. A phone's tape with no recents on it was sent back too.*
   *Ruled, 2026-11-23: recent part types left the picker's tape and stay
   in its list. The tape holds the picker and the key that clears it. The
   vehicle's names are secondary: compact tags hung from the block, in
   one row that scrolls sideways beside the picker.*
   On a phone nothing fits beside the picker, and the names' row is
   drawn last, under the candidates, the last thing on the trunk (the
   parts that fit, below). That is as built, and not ruled.
5. **One shape at every width.** A component is the same thing at 390,
   768 and 1280 px. It wraps or scrolls; it does not turn into another
   kind of control. A control has a largest width and keeps it: it does
   not stretch to fill a wide window.
   *Sent back: a strip of part types on a wide page and a list on a phone.
   Both became the one picker, then no wider than 24 rem.*
   The picker is now 19.5 rem, two of a wide page's columns, at every
   width from 34 rem up; under 34 rem it is the row's width, where the
   row is narrower than that or nothing is beside it. A stock print is
   columns at every width, and scrolls sideways where its paper is too
   narrow (thermal paper, below).
   *Ruled, 2026-11-24: under 64 rem the cards, the block's words and the
   picker are set on one module, 9.75 rem, half the picker's 19.5 rem.
   The rows under them keep their own widths there: a row's number tape
   and stock strip are sixths of the row with their floors, so its
   number tape no longer ends where the picker does, and its ink keeps
   room for its option text; the tape and the strip are a row high at
   the row's top and do not stretch with the ink. A first version put
   the rows on the module too, and was sent back: between 34 and 64 rem
   it squeezed the option text into a narrow column beside tall fields
   of tape and metal. Under 34 rem
   a print holds the row's id and the field its header pins first at
   its leading edge, and the field it pins second at its trailing edge,
   while the rest scrolls between. The pass the widths under 64 rem had
   of their own, before this ruling, set the cards, the block's words
   and the picker on three lengths.*
6. **A row scrolls sideways before it stacks or hides.** Things of a kind
   are one row. Past the window, the row scrolls by itself, shows the next
   one at its edge, and fades at an end with more beyond it. Nothing is
   kept behind a "more" switch, and nothing is stacked to be fanned out.
   On a phone an item of a list is compact: several show at once.
   *Sent back: names in stacks behind two switches; a list of part types
   that showed four on a phone.*
   *Ruled, 2026-11-23: candidates, and in part-number mode the vehicles a
   part fits, are rows stacked down the page, scanned down. It is a
   knowing change to this rule, which still governs the names and the
   cards. On a stock print, a folded value past 160 characters is cut to
   two lines and opens in place, without a script: a knowing, narrow
   exception to "nothing is kept behind a 'more' switch".*
7. **What belongs together is joined, and a line crosses nothing.** A line
   runs from a card to its word, and from what was asked to what answers
   it; a tag hangs from what it names; a control's tape touches the
   control. A line from a card ends on the vehicle's block, over its
   word. From the block one trunk runs straight down, and the picker and
   every candidate are tied to it by a stub. No line detours, and none is
   drawn over anything but the eyelet it lands in.
   *Sent back: a picker apart from its tape; the lines to the picker sent
   down the page's left edge to clear the name tags; the name tags hung
   from the picker's tape, which they do not name; lines that ran over a
   stamp.*
   *Ruled, 2026-11-23: under the vehicle's block the page is a counter.
   The lines from the cards end on the block. One trunk of pinstripe tape
   replaces the lines from the block to the picker and the candidates'
   wires. The history above stands: the trunk differs from the lane that
   was sent back because it never detours. It leaves the block and goes
   straight down, in the gutter between the plate's edge and the cards'
   edge.*
8. **What opens hangs from its control and overlaps the page.** A list or a
   note opened from a control is anchored to it, no wider than it where it
   is a list, and lies over what is beneath. It does not push the page
   down.
   *Sent back: a list of part types as wide as the page that moved the
   candidates.*
   *Ruled, 2026-11-23: the picker's list always opens downward. On a page
   too short for it the list lengthens the page, and it never opens
   upward over the plate. It is a bend of this rule: the list still lies
   over the candidates and moves none of them.*
9. **Bracket ends are the plate's.** Only the VIN input's brackets end in
   a bracket. Every other line ends on the thing it reaches.

## Tokens

All tokens are custom properties in `ui.css`, on `body` unless the table
says where.

| Role | Light | Dark |
| --- | --- | --- |
| Ground `--bg` | `#dcecf7` | `#16222d` |
| Text `--fg` / `--muted` | `#14181d` / `#47515c` | `#eef2f5` / `#a9b6c2` |
| Panel ground `--card-bg` | `#ffffff` | `#1f2d3a` |
| Rule `--line`, not-yet `--faint` | `#9fb3c2`, `#a9bccb` | `#45566a`, `#3a4a5c` |
| Link `--link`, and the focus ring on the ground and a panel, `--ring` | `#0b4fa8` | `#8ec5ff` |
| Error `--bad` | `#b91c1c` | `#ff8a8a` |

A focus ring is drawn on something, and takes the colour that shows on
it. On the ground and on a dialog's panel, which change with the scheme,
it is `--ring`, the link's blue. A material does not change, so a ring
on one is fixed, as a group's mark is: `--ring-pale`, the light scheme's
blue (`#0b4fa8`), on metal, manila and thermal paper, and `--ring-tape`,
the dark scheme's (`#8ec5ff`), on label tape. Each is 3:1 or more
against what it is on, in both schemes. It was `--link` everywhere, which
in the dark scheme was pale blue on manila, on thermal paper and on the
plate, and in the light scheme dark blue on black tape.

Each of the six groups of a VIN has one colour, `--c`. Its label, bracket and
leader line all use it, and its card prints it as a band.

| Group | Positions | Light | Dark |
| --- | --- | --- | --- |
| Maker `wmi` | 1–3 | `#c8102e` | `#ff6b81` |
| Vehicle `vds` | 4–8 | `#0b4fa8` | `#7db4ff` |
| Check `check` | 9 | `#5b2d8e` | `#c9a2ff` |
| Year `year` | 10 | `#a34c09` | `#ffb067` |
| Plant `plant` | 11 | `#0f6b46` | `#6fdcab` |
| Serial `serial` | 12–17 | `#3d4650` | `#b8c4cf` |

A group's mark on a material is `--cm`. On metal and on manila it is the
light-scheme colour in both schemes, except the year's, which is a shade
brighter there (`#b3540a`). On label tape it is the dark-scheme colour in both
schemes.

Three typefaces carry the page, and all are the system's: no font file is
sent.

- `--label-font` (Arial Black, then Helvetica Neue): what is stamped or
  struck: a key's word, a label over the plate, a card's band, a mark's
  stamp. Weight 900, upper case, tracked.
- `--cell-font`, the monospace: data, which is what a visitor reads or
  copies: the VIN's characters, a make, a model, a year, a part number,
  option text, a stock field, a location, a count. Under the cards a datum
  is this face on every material, so it is one voice on the vehicle's
  block, a name tag, a row and a print.
- `system-ui`: prose. Body text is 15 px (`.9375rem`) on a line of 1.5: a
  blank strip's sentence, a dialog, a card's rows.

Few browsers know `ui-monospace`, so `--cell-font` names a designed face
for each system: `ui-monospace` (Safari's, SF Mono), Menlo (macOS),
Cascadia Mono and Consolas (Windows), Roboto Mono and Droid Sans Mono
(Android), DejaVu Sans Mono, Liberation Mono and Noto Sans Mono (Linux),
then `monospace`. No character of them is wider than .62 em, and the
plate's field spaces its characters by `1ch`, so the 17 cells hold in
each. Only SF Mono has a weight above bold. In every other face 900 draws
as 700, so size carries the hierarchy and weight only helps it.

### Type roles

Under the cards, type is six roles. A role fixes face, size, weight,
line height, tracking and case together. Each is a token on `body`, a
`font` value, beside a token for its tracking, and every rule under the
cards takes one: none writes a size, a weight or a tracking of its own.
Five carry the hierarchy. The sixth, the fine print, is one size under
the row for one place, the fold of a stock print, where a row's size
would turn a file's undeclared fields from one line into two.

| Role | Token | What takes it | Size | Weight | Line | Tracking | Case |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Micro label | `--type-micro` | The page's own word for a value, a column or a count: the block's stamp and its labels, "Part type", a vocabulary's name, "Note", "shelf", a kind of stock's bar, a count in a pill, a mark in plain words | 11 px | 700 | 1.2 | `--track-micro`, .04 em | Upper |
| Row value | `--type-value`, `--type-value-b` | What a row says: a print's cells and its columns' names, a note, a name's years. Bold (`-b`) where it leads its row or sits on tape: option text, a name, the block's details, a row's years | 13 px | 400, and 700 | 18 px | None | As written |
| Fine print | `--type-fine` | What a stock row carries beyond its declared fields, folded under it | 12 px | 400 | 18 px | None | As written |
| Head | `--type-head` | What a strip of tape is headed with: the block's make, model and year, a vehicle's name on its row, the pressed part type, the tape over the rows with no number. On a print, the location, which is walked to | 16 px | 900 | 18 px | `--track-tape`, .04 em, on tape; none in ink | As written |
| Display | `--type-display` | What was asked for and how many there are: a part's number and its stock counts. On a card, its one large value: the serial, the model year | 20 px | 900 | 24 px | `--track-tape` on tape; none in ink | As written |
| Key | `--type-key` | What is struck or stamped: a key's word, a table's header, a mark's stamp | 11 px | 900 | The face's own; 12 px in a stamp | `--track-key`, .08 em | Upper |

All but the key are the monospace, and the key is the label face.

- **The hierarchy is the sizes**, largest first: a part's number and its
  stock counts; then a print's location and the block's words; then a
  row's values; then the labels. Each step is about a fifth larger than
  the one under it: 11, 13, 16 and 20 px. Nothing under the plate is
  larger than a part's number: a card's large value, the serial or the
  model year, is its size. The fine print is no step of it: colour
  and its indent set it under its row.
  *Ruled, 2026-11-24: a card's large value dropped from 24 px to the
  display role, so that nothing under the plate is larger than a part's
  number.*
- **Tracking and weight follow the ground.** A head and a display are
  raised white on label tape or struck in the second tape, where letters
  fill in, so they are tracked. The location is a head in ink and a
  card's large value a display in ink, and neither is. A value is never
  tracked, and on tape it is always bold.
- **A stamp holds data in the data's face.** "Replaces" is stamped in the
  label face; the number after it is a bold row value, in its own case.
- **Sizes are in rem**, so a visitor's font setting scales them, and
  `--row` is in rem with them. A micro label and a key's word are never
  under 11 px, whatever that setting.
- **No text on the page is under 11 px.**
- **Under 64 rem one role is redefined:** the display is 18 px, in every
  row and on a card. The head is 16 px on the block at every width, and
  a row's years are the bold row value, 13 px, as on a wide page. The
  smaller sizes the page was first built with there, 14.4 and 12.8 px on
  the block and 12 px on a row's tape, are gone: the block's words have
  columns wide enough for the head, and a row's years are under its
  number.
- **Off the scale, and in px:** the plate, whose characters follow its
  width through container units; the six cards, whose rows are the body
  face and whose check digit's verdict is the label face at 16 px; the
  header, the foot and the dialogs. They take a role's token where they
  already had its values: a key (`button`), a table's header, the
  foot's release and a count in the picker's list. Two places on a card
  take one by decision: its large value takes the display (the ruling
  above), and its "Show the sum" takes the micro label, which raised it
  from 10 px.

Everything under the six cards is set on one module and a few lengths, so
that its parts line up down the page.

| Token | Value | What it sets |
| --- | --- | --- |
| Module `--m` | a sixth of the row under the block: 160 px on a wide page; 9.75 rem, 156 px, under 64 rem | A column. A card is a module less the 8 px between two, and what hangs under the vehicle's block is a whole number of modules wide, less those 8 px. Under 64 rem the module is fixed, half the picker's 19.5 rem, and sets the cards, the block's words and the picker: the cards scroll, six modules (`--col`), the first on the rows' leading edge at rest, the block's words are a module each, and the picker two |
| `--rail`, `--end` | 20 px each; `--end` is 0 under 64 rem | How far the first column's leading edge and the last column's trailing edge are in from the block's. The gutter that `--rail` leaves beside the first column is the trunk's |
| `--num`, `--strip` | two modules, and one, each less the 8 px, the first less the lead too. Under 64 rem the same of the row's own sixth, not the module, and no narrower than 9.5 rem and 7.5 rem | A row's two fixed columns: its number's tape and its stock strip. The floors are what they hold: a number of eleven characters, and two counts of three digits |
| Spacing `--s1` to `--s5` | 4, 8, 12, 16 and 24 px | The one scale of spaces |
| Inset `--inset` | 12 px | Text's distance from the edge of its material, on every material |
| Row `--row` | 3 rem, 48 px; 3.5 rem, 56 px, under 64 rem | The height of a tag laid long, the picker and a blank strip. Under 64 rem it is a finger's: the picker's key in it is 45 px high, a stock strip is a row high, and a tag's two names, 24 px each, are 8 px apart |
| Lead `--lead` | 28 px | The leading end of a tag laid long, which holds its eyelet. Its content begins after it |
| `--trunk` | the Vehicle group's `--c`; the Serial group's in part-number mode | The trunk and its stubs |
| `--gutter` | 1 rem; 0.5 rem under 46 rem | The page's side margin |

`--m`, `--rail`, `--end`, `--num`, `--strip`, `--lead` and `--trunk` are
set on the parts that fit (`.fit`) and on a part's answer (`.part`). The
others are on `body`.

### Widths

The stylesheet asks the window's width at three places, and each is where
something stops fitting. Between them a part keeps one shape, and its
lengths are the module's or a floor of their own.

| Width | What stops fitting | What changes under it |
| --- | --- | --- |
| 64 rem | Six cards in one row, each wide enough for its rows | The cards, the block's words and the picker are on one module, 9.75 rem; a row's columns are its own sixths. The cards scroll sideways. The leader lines and the stepped lines are not drawn, and the lines to the block are carried. The block's words are packed at its leading end. `--end` is 0, the row is 3.5 rem and the display 18 px. A row's years are under its number, and its tape and strip stand at its top. The keys over the plate are 44 px high |
| 46 rem | The label of a one-cell group, 47 px, over a cell of 40 | Those three labels are not drawn. The cells stand 2 px apart, not 4, and the page's margin is 0.5 rem |
| 34 rem | A cell wide enough for a finger, in one row of 17; the names beside the picker; a row's ink beside its number; a print's columns on its paper | The plate is two rows. The picker is the row's width, and the names are under the candidates. A row is two lines. The print scrolls sideways, holding the row's id and what its header pins. Part-number mode's fields stack, and its switch is the plate's width. The check digit's sum is turned on its side |

The Developers dialog asks once more, at 40 rem, where a snippet's label
no longer fits beside it. Every query is the window's, not a container's:
under the cards a part's width is the window's less the page's margins.
At 1023 and 1024 px the page differs only in what
the cards' row forces: the picker is the same width on either side, and a
row's columns are within 6 px.

## Signature: the VIN input

The form is the plate. It is 17 cells under six group labels, and one real
text field laid over them.

- **One native field.** `plate.js` lays a transparent `textarea` over the
  cells. The field keeps paste, selection and the phone's keyboard. The
  cells draw the text, the caret and the selection; the field shows none of
  its own. Without JavaScript the field is a plain input below the cells with
  a Decode button. With it the plate answers as it is typed and Enter
  sends it, so the button is not drawn, and the Tab key stops on nothing
  a visitor cannot see.
- **A fixed grid of 17 columns with no gaps.** Labels, cells and brackets
  share it. The field's letter-spacing is `100cqw / 17 - 1ch`, so each
  character sits on its cell and a click lands on the cell it shows.
- **A cell** is a struck character: monospace 700, `--on-metal`, lit from above
  by `--emboss`, over a 4 px underline in its group's mark. Its size follows
  the plate's width through container units.
- **Empty**, the plate reads "Type a VIN" in the label face across its middle.
- **Focused**, the plate takes the 3 px `--ring` outline, 3 px off every side
  of what is seen, and the next empty cell blinks a 6 px bar along its
  bottom, in `--ring-pale`, the same blue on the metal in both schemes.
  The brackets above and below stand 7 px off the plate, clear of the
  ring.
  *Ruled, 2026-11-24: in the dark scheme the bar marking the next cell
  and the touch selection handles take the light scheme's blue, for
  contrast on the metal.*
- **The caret among characters** is a 4 px bar in the gap before its cell.
  Under a mouse, the gap a click would pick shows the same bar at 35%.
- **A selected cell** is inked: `#16202b`, white character, ringed 3 px in its
  group's colour.
- **A cell that is wrong** fills with `--bad`: a character no VIN holds,
  and a cell the decode names, as a year code no year uses (position 10,
  error 11) or a position its suggested VIN marks with "!". It has one
  meaning, "this cell is wrong", and the vehicle's block says why (the
  parts that fit, below). The check digit's cell is struck only where
  its card's own sum fails: the decoder also fails the check digit of a
  VIN with a character invalid for its position, whose sum the card
  shows holds. A bad VIN is not an error: the plate still decodes, and
  once the VIN is whole its problems are on a strip of label tape under
  the answer (vinyl tape, below). While a cell is struck the field is
  `aria-invalid`, and a line out of sight says in words which positions
  are wrong ("Position 10 is wrong."), its first description; the answer
  is the rest of it, as on any VIN. `plate.js` strikes a cell as it is
  typed when it holds a letter no VIN holds, and the cells the answer
  names (`data-wrong`) when it comes.
- **A typed character stamps** into its cell (scale 1.6 to 1, 0.18 s).
- **Narrow windows keep the plate.** Under 34 rem the cells wrap to two rows of
  nine, breaking after the check digit, so a cell stays wide enough for a
  finger. Under a coarse pointer the selection handles are the cells' own,
  in `--ring-pale`, and a long press selects the whole VIN, drawn on the
  cells; the phone draws no handles of its own over them.
- **A one-cell group's label is wider than its cell** under 64 rem, and
  the three are neighbours. Check ends on its bracket's trailing edge and
  Plant begins on its bracket's leading edge, with Year centred between
  them, so none runs into the next. Under 46 rem the three are not
  drawn: their cards name them. Under 34 rem Check is over the upper
  row's last cell, and the lower row's brackets hold their names under
  them: the Year's ends on its bracket's trailing edge and the Plant's
  begins on its bracket's leading edge, 14 px apart at any width.
- **The second mode changes none of this.** In VIN mode the plate, its
  cells, its field and its answer are as above. The one addition is the
  switch over the plate's leading edge (the plate's second mode, below).

## Signature: the plate's second mode

The plate also asks for a part number (#325). The page opens on a VIN; the
mode is in the URL, and nothing is remembered between visits.

- **The switch** sits over the plate's leading edge: two keys of metal
  joined, "VIN" and "Part number", a rivet each. Each is a link to the page
  in its mode with nothing on the plate, so it works without a script; the
  mode the page is in is pressed, flush. With a script the key's plate,
  empty, and its answer are swapped in place, the pressed key's too, a
  history entry of their own.
  Under 34 rem the switch is the plate's width, the keys halves of it.
  Under 64 rem its keys, and VINs beside them, are 44 px high.
- **The URL** names the mode by the fit request's own fitment key:
  `from.fitment.type`, `from.fitment.number` and, for a sided part,
  `from.fitment.side`, as the VIN's is `from.vin`, and the request is the
  fit request's own query, refused in its words. A URL is one mode: one
  naming a VIN beside a fitment key is the VIN's plate, empty, over its
  empty cards, and under them, in the names' place, a blank strip saying
  both ("This link names a VIN, …, and a part number, …") with a link to
  each mode with its own. Nothing there waits for a model: no names or
  parts that fit are drawn, nor the lines to them.
- **The plate** is the VIN plate unchanged: its metal, its screws, its
  ledge, its focus ring. It holds two fields on a grid of three, the part
  type in the first and the number in the other two, under labels and
  brackets in the Serial group's colour: a part number is the nearest thing
  to a serial, and a seventh colour would need a reason. Under 34 rem the
  fields stack, the brackets above give way, and each field's label is
  struck into the plate over it.
- **The part type** is a strip of the second tape stuck to the plate, raised
  as a control, its name in its own case and a chevron at its trailing end.
  It is a button that opens the part types dialog (a popover, so without a
  script too), which lists every part type of the catalog: its spine's
  entries, under the headings of a part vocabulary, with no count. Choosing
  one closes the dialog, fills the field, and keeps the number; with a
  script the focus goes back to the number. While the dialog is open the
  strip is pressed. With no catalog it reads "No catalog loaded" and opens
  the Vocabularies dialog.
- **The number** is the form's own text field, struck into the plate in the
  VIN's lettering (monospace 700, `--on-metal`, `--emboss`) over a 4 px
  underline in the Serial group's mark, in the case it was typed. Empty, it
  reads "Type its number" in the label face. Without a script Look up sends
  it; with one the page asks as it is typed, 150 ms after the last key, in
  the same history entry, Enter makes one of its own, and the Look up key
  is not drawn.
- **Under the plate** the number's bracket opens upward, flat and faint
  until a number is typed. One line of pinstripe tape in the Serial group's
  colour runs from its foot into the eyelet of the part's tag: down from
  the bracket's middle, two thirds of the way across the plate, along to
  the tag's leading end, and down into the eyelet. Under 34 rem the
  bracket's middle is the plate's. The line is two boxes on the plate's
  grid, so nothing is measured. No line is drawn without a tag.
- **The part** is a manila tag laid long, as a candidate's is (the parts
  that fit, below), and as wide as the plate, as the vehicle's block is in
  VIN mode. After its eyelet, its number is on black label tape with its
  part type's name and its side at the tape's trailing end, and under the
  number under 64 rem; then, in the tag's ink, how many vehicles it fits
  in the catalog, over its note; and
  its stock strip is its trailing end. Its lead is the gutter's width
  longer than a vehicle's, so its tape and its stock strip stand over the
  vehicles' columns. A retired number answers with the part that replaced
  it: its tag carries a solid ink stamp, "Replaces" and the number asked
  for, and the catalog's note on the retirement beside its own.
- **Its stock** behaves as a candidate's: the strip opens the shelf rows and
  the yard vehicles among the vehicles listed, named in the URL as a
  candidate's are (`fitment.number`). They are a print under the tag, from
  its number's tape to its trailing edge, shelf over yard at every width
  (thermal paper, below). The rows are the page's, paged with the
  vehicles; "Show all" is every one.
- **The trunk** leaves the tag's bottom edge and runs straight down, in the
  Serial group's colour, beside the open print and through the vehicles,
  each tied to it by a stub into its eyelet, as a candidate is in VIN
  mode.
- **The vehicles** the part fits are rows down the page under the tag, each
  a manila tag laid long with no stock strip. On its label tape is the
  catalog's name for the vehicle (a vPIC catalog's as VIN mode names one,
  "vPIC model 1801 (F-150)", where the answer carries the name), with its
  years at the tape's trailing end, and under the name under 64 rem. In
  the tag's ink is its option text
  verbatim, over its names in the vehicle vocabularies loaded: a
  vocabulary's name once, a micro label, muted, before every name it has for
  the vehicle, each underlined in the Vehicle group's colour, as a name
  is on its own tag. A vehicle no vocabulary names says "No name". While
  the server holds the catalog, a browser names them in the server's
  vocabularies alone, which the server's fit knows. A vehicle is no link.
  More than a page (30) are a "Next page" link away. Under 34 rem a row
  is two lines: its tape, then its ink.
- **A sided part** asked for with no side is asked on each side: one found
  is the answer, and its side stays on every link from it; several are
  links, one to each side.
- **Nothing of a VIN's answer** is drawn in this mode: no cards, no "Also
  known as", no lines to them, no parts that fit.

## Signature: aluminum plates

Everything a visitor presses or reads a heading from is the same brushed
metal: the VIN plate, the buttons, the brand, the VINs menu's rows and the
dialogs' heads. The metal is four layers.

1. **Body:** `linear-gradient(#eef1f4, #b4bec7 55%, #cdd5dc)`.
2. **Light streaks:** `brushed.svg` over the body at `600px 120px`. Together
   these are `--metal`.
3. **Dark streaks** (the VIN plate only): a `#1c2630` layer at 22% with
   `brushed.svg` as its mask, offset so the dark grain falls between the light.
4. **A band of light** (the VIN plate only): a 105° white gradient at 35%,
   faded out downward by a mask.

Around the body:

- **Edge:** 1 px `--steel` (`#6f7b86`), 2 px on the VIN plate and dialogs.
- **Raised:** `--raised`, a white line inside the top over `--lift-raised`,
  on every key. The VIN plate and dialogs sit on 3 px of `--steel-dark`
  (`#58626c`) and a soft drop shadow. On the VIN plate that ledge is its own
  bottom border, not a shadow, so the focus ring clears it. The brand is
  static: a white line inside the top and 2 px of `--steel` below, with no
  shadow, so the header's keys stand higher than it.
- **Lettering:** `--on-metal` (`#232930`) with `--emboss`, a white edge below
  and a dark edge above, so the letter reads as struck in.
- **Pressed:** a key moves down 2 px, loses its ledge and its shadow, and
  takes `--alu-in`, `linear-gradient(#8e99a3, #aab4bd)` under faint ridges,
  with `--pressed` inside its top. A rivet stays on it.

The fastener says what the piece is.

| Fastener | On | Drawn as |
| --- | --- | --- |
| Rivet | Small tags: the page's keys (Vocabularies, VINs, Developers), menu rows; one at each end of the brand | A 2.5 px `--steel-dark` dot in a light ring |
| Slotted screw | The VIN plate and every dialog's head, one at each end | A 5 px light disc, a `--steel` ring, a 9 × 2 px slot |
| Eyelet | A manila tag | A hole punched by a radial mask through a darker patch that reinforces it |

Only a dialog's head is metal. The panel under it is panel ground. Text to
copy sits in a recess: an inset shadow on a slightly darkened ground.

## Signature: vinyl tape

Vinyl is three tapes. The page's own word on a thing, a stamp or a tag's
heading, is a strip of black label-maker tape. A name that can be pressed
is the second tape, embossing tape in the plate's metal. A bracket, a
leader line or the trunk is pinstripe tape in a group's colour.

### Label tape

- **Body:** `--tape`, faint ridges every 3 px over
  `linear-gradient(#30363d, #171a1e)`.
- **Edges:** a lit top (`--tape-lit`, `#69727c`), a heavy bottom
  (`--tape-edge`, `#050607`, 3 px), and `--lift-stuck` under it: a strip of
  tape is stuck on, lower than any control.
- **Lettering:** `--on-tape` (`#f6f8fa`) in the monospace face with
  `--tape-emboss`, a dark edge up and left and a light edge down and right, so
  the letter reads as raised.
- **Case is kept.** A vocabulary's entry is case-exact, so its name is never
  upper-cased. A vocabulary's own name, on its tag's strip, and a stamp are
  upper case.
- **A word on the vehicle's block** sits over a 3 px underline in its
  group's mark, under the line that comes down to the block from its card.
  No line ends in a bracket but the plate's own.
- **A tag's heading** under the cards is label tape after the tag's eyelet,
  with the tape's lit top and heavy bottom and no shadow of its own: a
  vocabulary's name on a name's tag, and a candidate's or a part's number,
  or a vehicle's name, on a row's.
- **What the decoder says is wrong** with a whole VIN is a strip of label
  tape, stuck on, as wide as the plate: a warning triangle in the error
  colour as tape takes it (`#ff8a8a`), then the decoder's sentences, each
  a bold row value raised white, and, where the decoder reads another
  VIN in its place, a link to it, underlined in the tape's own white. It
  was text in `--bad` on the ground, which rule 1 does not allow. It is
  under the answer at every width, where the text was: between the
  brackets and the cards it would push the picker under a phone's fold
  on every VIN with a problem. The reader is told where they work
  instead, at no cost in height: the vehicle's block's stamp carries the
  count and the first sentence, and a stamp at the candidates' head says
  to check the VIN; each links to this strip (the parts that fit, below).
- **An empty place stays.** With no vocabulary loaded, or before the model
  is known, the names' place is a blank strip: a 3 px dashed `--faint`
  outline with a sentence in it (empty places, below).
- **On a tag the tape takes the tag's colour.** A control on a manila tag is
  the same ridged tape with the group's mark as its body, edges mixed lighter
  and darker from that colour, and a faint white halo on its letters, as
  vinyl whitens where it is struck. A name's "i", on a name tag, is plain
  black tape. Each stands raised, its own bottom edge 1 px, and pressed it
  is flush, its colour mixed darker with black.

### The second tape

A part type is both a name and a control, and so is a candidate's stock
strip. A name that can be pressed is the second tape: embossing tape in the
plate's own metal. Black label tape stays for a name that does nothing, and
a key of metal for a control that names nothing.

- **Body:** `--alu`, the tape's faint ridges over `--metal`, a 1 px `--steel`
  edge with a white top and a `--steel-dark` bottom.
- **Lettering:** `--on-metal` in the monospace face, weight 900, struck as
  the plate's characters are (`--emboss`), in the name's own case: the
  head role on the picker, 16 px, and 14 px in its list.
- **A count** sits after the name in a small pill outlined in `--on-metal`
  at 55%.
- **Raised and pressed** as every control: `--raised` unpressed; pressed
  (`aria-current`, the part type the page shows), flush on `--alu-in` under
  `--pressed`.
- **At a tag's trailing end** (a candidate's stock strip) it is the tag's
  last column, edge to edge, and casts no shadow off the tag, so it stands
  raised by its edges alone: a white top and 3 px of `--steel-dark` below.
  A kind of stock is a cell of it: its count in the display role, as large
  as the part's number, over the kind's name, a micro label. A chevron ends
  it. Pressed (open) it is `--alu-in` under `--pressed`, its lettering
  darker, its bottom edge 1 px, and its chevron turns up.
- **It carries** the picker's key, each part type in the picker's list and
  in the part types dialog, part-number mode's part type on the plate, and
  a stock strip.

### Pinstripe tape

- **Body:** the group's colour, `--c`. A bracket is 4 px with its outer
  corners rounded 2 px. A leader line is 5 px with round joins.
- **Lift:** `--stripe-lift`, two drop shadows: a 1 px light edge up and left,
  and a soft shadow down and right. The stripe reads as stuck on the ground.
- **Not yet typed**, a bracket or a line is flat: 3 px of `--faint`, no lift.
- **A bracket that holds its name** (the lower row on a phone) lies flat, so
  the lift does not edge the letters.
- **The trunk** is 5 px of `--trunk`: the Vehicle group's colour, and the
  Serial group's in part-number mode. The stylesheet draws it a piece an
  item. A piece is lit and shaded along its sides alone, so no seam shows
  where two meet; a stub, along its top and bottom.

## Signature: manila tags

A decoded card is a manila parts tag, hung by its eyelet from its leader line.

- **Stock:** `--manila`, `paper.svg`'s fine noise at `200px 200px` over
  `linear-gradient(#efdcae, #e2c88f)`.
- **Shape:** the top corners are clipped 16 px by a `clip-path`. The stock is
  the card's `::before`, so the card's own `drop-shadow` follows the shape; a
  `box-shadow` would be clipped away.
- **Laid long**, a tag has its leading corners clipped 12 px and its eyelet
  at its leading end, in the tag's lead and half a row down, where the
  stub or the wire it hangs by lands. Every tag under the cards is one: a
  name's, a candidate's, a part's and a vehicle's. It prints no band: its
  heading is a strip of label tape (vinyl tape, above).
- **Eyelet:** a 3.5 px hole at the top centre, in a 10 px patch of
  `--manila-patch` (`#c39a5c`). The ground shows through the hole.
- **Band:** the group's mark, `--cm`, printed across the tag under the eyelet,
  with the title and positions in white in the label face.
- **Ink:** `--on-manila` (`#2b2418`) for values and `--manila-muted`
  (`#5c4c2e`) for row labels. Both hold in the dark scheme, because the stock
  does.
- **Wrong:** a card whose group is wrong prints its band and its large value
  in `--manila-bad` (`#a01414`).
- **A control on a tag is label tape, not metal.** The check digit's "Show
  the sum" is a short strip of ridged tape in the tag's own colour, `--cm`,
  with raised white letters. Aluminum clashed with the paper, and nothing
  bolts metal to a tag. A candidate's stock strip is the one metal on a
  tag, and it is tape, not bolted: it names what it opens, so it is the
  second tape, and it is the tag's trailing end, edge to edge, rather
  than a control sitting on the paper.
- **Not yet typed**, a card keeps its place as a 2 px dashed `--faint` outline
  with its muted title, and no stock.

## Signature: thermal paper

What a stock file says of a part is a print: a strip of thermal paper, fed
out from under the tag it is of, and torn at its foot (#376). A tag is a
thing found; the print is about it, so it is neither hung nor stuck. It is
a print in columns (#390), drawn with only what a receipt printer does.
"Strip" alone, here and in the stylesheet, is the tag's stock strip, the
control; the paper is the print (`.receipt` in the stylesheet).

- **Stock:** `--thermal` (`#f6f4ee`), the same in both colour schemes,
  shaded for 15 px where it leaves the tag and faintly down each edge. Its
  `drop-shadow` is on the paper itself, so it follows the torn foot.
- **Print:** `--on-thermal` (`#232229`), one near-black in the page's
  monospace face, the row value: 13 px on an 18 px line, the system's,
  since the page ships no font file. `--thermal-muted` (`#62606b`) is the
  fainter print: a column's name, what is folded, a note. No other colour
  is printed, and a group's colour never is.
- **Width:** the paper runs from its tag's content, after the tag's lead,
  to the tag's trailing end.
- **A kind of stock:** its name and count reversed, a micro label,
  `--thermal` on a small bar of `--on-thermal`, and on the bar's line, at
  the right, its sources' as-of dates, or why one did not answer, each
  named when there are several. A kind none of whose sources answered
  states no count: a dash stands where the count would, read out as
  "unavailable". Shelf is over yard, and one kind is printed when one is
  loaded.
- **Columns:** a row is a line. Its id is first, then its location in the
  head role, larger and heavier than the row, as it is what is walked to,
  then a column for each field its
  source's header declares (`fields`, ADR 0019's amendment of 2026-11-23),
  in the header's order. A header may pin one or two of its fields
  (`pin`, #404): the first is printed beside the id, before the location,
  and the second last, at every width; a print whose rows come from
  several sources takes the pins of the first that pins. A declared
  column stands under its field's name,
  in the fainter print over a rule of dashes; the id and the location have
  no name printed. The id, the location and the first pinned field are
  each as wide as the longest of them and never wrap, short of 24
  characters; with nothing after it, the location takes the line's rest.
  A declared column is
  never narrower than its field's name. The declared columns
  share the rest of the paper by what they hold: none asks for more than
  24 characters, and a longer value wraps in its column, between words.
  Where rows of two
  sources share a print, it has a column for every field either declares,
  each once, and a row leaves empty those its own source does not declare.
- **Exactly the declared fields are columns**, but one: a declared field
  whose value is the row's id on every row of the print says what the id
  says, so it is no column and is not folded (#397). A source that
  declares none
  prints an id and a location a row, and folds the rest. A null prints
  nothing. A number or a boolean prints as its JSON. A list or an object
  is never a column's. A field is printed as its file has it.
- **The fold:** what a row has beyond its declared fields is under it,
  indented 24 px, in the fine role: the fainter print, a size under the
  row, at regular weight, so the declared columns and the location are the
  darkest things in a row.
  Each is a name, a colon and its value, one after another, wrapping at 80
  characters where the paper is wider. A list or an object is its JSON,
  compact, cut to one line.
- **The clamp:** a folded value past 160 characters, two such lines, is cut
  to two lines behind a plus sign. Pressing it prints it whole, in place,
  behind a minus sign. It is a `details` element whose summary is the
  value, so it opens without a script, and every field's name shows while
  it is shut.
- **A note** is a starred line under its row, after what is folded, in the
  fainter print and as the page writes it, not in capitals: a shelf row
  tagged with a retired number quotes the number as typed, and the yard's
  vehicle that is the VIN looked up says "This VIN", reversed.
- **Nothing in a kind** that answered is "None." between asterisks, under
  its bar and date. A kind that did not answer prints its bar and why,
  and no "None.": what it holds is not known.
- **Past ten rows** a kind says how many it prints, under a rule of dashes,
  with "Show all" underlined, in bold.
- **Foot:** a torn edge of 9 px teeth.
- **Case:** nothing typed into a stock file changes case on a print: an
  id, a location, a field's name and its value are as written. Of the
  page's own words, a kind's name and "None." are capitals.
- **Where the paper is too narrow for its columns** the print scrolls
  sideways under its bar, by itself, and the page stays. It is the same
  columns at every width. From 34 rem up the location stays at the
  print's leading edge
  while the id and the columns slide under it, so the eye still runs
  down the locations, and what is folded keeps the paper's width and
  does not move. An end with more beyond it is shaded for 12 px, as the
  paper is where it leaves its tag, and the shade goes at that end:
  paper does not fade, and a fade would take the last characters of
  what is folded with it. It needs no script and no scroll timeline.
- **Under 34 rem** a phone's paper is narrower than most prints, so the
  columns keep the widths their values ask, and the print scrolls. It
  holds what the yard needs in view: the row's id and the first pinned
  field at its leading edge, and the second pinned field at its trailing
  edge, shaded for 8 px along its leading edge on each of its rows, where
  the rest slides under it; that print draws no shade at the paper's end.
  The location scrolls with the rest. With no pin only the id is held.
  The first pinned field stands beside the id by the id's width, which
  the print names as a class (`.idw1` to `.idw24`, its widest id in
  characters), since a script would be needed to measure it. With the
  demo set the yard's print at 390 px holds `DEMO-Y-01` and `D-1` in
  view, where its paper, 302 px, showed the id, the location and the
  stock number that repeated the id. What is folded is indented 12 px,
  not 24, and wraps as one row
  whose space between two fields is not counted at a line's end, so a
  field as long as the paper keeps one line. On a deployer's shelf rows
  of eleven fields a row is 76 px high there. The form it replaced, a
  row's fields one after another, each after its name, took 146.
- **A screen reader** is told what the print is of by the region's name,
  "Stock of" and the number. A kind's bar is a heading one level under the
  part's number, and the columns are a table's, the id's and the
  location's named out of sight. The table is in a group named by its
  kind's bar, which is what scrolls, and takes focus, so a keyboard
  scrolls it: a kind of stock is one stop of the Tab key, whether or not
  its paper is wide enough for its columns.
- **It has no head, no barcode and no dot leaders.** It names no part: its
  tag does, and a press keeps the tag in sight (the parts that fit,
  below). So the print no longer breaks rule 3.
- **It breaks one composition rule, knowingly, in three places.**
  - *Rule 6, stacked.* Shelf and yard are one print, one over the other,
    at every width, where a row would scroll sideways: a print is one
    strip of paper, and two side by side are two prints.
  - *Rule 6, folded.* A folded value past 160 characters is behind its
    plus sign, where the rule keeps nothing behind a "more" switch. The
    exception is the maintainer's ruling, and narrow: that value alone,
    opened in place, with no script.
  - *Rule 6, shaded.* Where a print scrolls, an end with more beyond it
    is shaded, where the rule fades it. Not ruled.
- **The rows with no number** are the same print, under the strip of label
  tape that names them (the parts that fit, below).
- **Still open:** photos on a yard's rows are #391, and a yard's very long
  lists are not addressed.

## Good enough: what a card holds

Each group has a card under its bracket, six in one row. The page has one
width where the row changes, 64 rem: at 64 rem and up it is wide, and below
that it is narrow.

- The band reads the group's title and its positions ("Vehicle 4–8").
- The body is either label and value rows, or one large value in the monospace
  face, the display role (Tokens, above). A row is its label, small, with its value under it, both at the
  card's leading edge, so every row reads alike whatever its value's length. The check digit's card opens its arithmetic in a dialog.
  The year's card shows the year, then the codes around its own as one run of
  small text with its own in bold.
- **A finished card with no value**, on a VIN with problems, says why in
  the body face: the decode's own sentence for the first problem at the
  card's positions, in its ink, or "Not in the decode", muted, where no
  problem names them. A year code no year uses leaves the Model year's
  card with no year, and it says error 11's sentence. On a VIN with no
  problem an empty card stays as it was.
- The six cards share one grid row, as tall as its tallest card. Up to four
  rows, eight or nine lines, is usual and is not a limit. No card cuts text off or keeps it behind
  a control.
- A newly finished group pops its card in (0.3 s).
- **Wide**, the six cards span the plate's width, each under its leader line.
- **Narrow**, the six cards are one row that scrolls sideways, in the VIN's
  order, six columns of the module under the cards (9.75 rem, so a card
  is 148 px at every narrow width), the first on the rows' leading edge
  at rest. The region reaches both edges of the
  window, is named, and takes focus, so a keyboard scrolls it. An end of
  it with more beyond fades over 48 px, as the names' row does, but for
  the row's last 14 px, where the lines to the block leave it whole. The
  fade needs scroll-driven animation, and without it the row is cut
  square.

## Good enough: the parts that fit

Under the cards is the vehicle's block, and under the block the page is a
counter: rows, read down. They are the picker of the part types the loaded
catalog has for the vehicle, with the vehicle's names beside it, and the
candidates of the part type pressed, each with its stock. Everything is set
on the cards' columns (Tokens, above). The block is the plate's width. What
is under it starts on the first card's leading edge, ends on the last
card's trailing edge, and is a whole number of columns wide.

- **The vehicle's block** is label tape, stuck on: a strip across its top
  with its stamp, "This vehicle ·" and the catalog's name, as a name tag
  has its vocabulary, then one row. From the left: the make the catalog
  files the vehicle under, in the catalog's word, where it has one; the
  catalog's model the decode reached; and the year it asked with ("any
  year" without one), each over its group's underline, the Maker's red,
  the Vehicle's blue and the Model year's orange, lightened for the tape.
  Then comes what the decode says of the vehicle that a catalog's option
  text names: the Vehicle card's rows but the model, a small label before
  each value, raised white on the tape, in the face and size of a
  candidate's option text, which they are read against; and at the
  block's trailing end the Vehicle group's five characters, each over the
  group's mark, as on the plate, with its position under it and no label:
  the positions say what they are. On a wide page the block's first three
  columns are the cards': the make is under the Maker's card and the
  model under the Vehicle's, whose lines come straight down, and the year
  is in the third, where the Model year's line steps left to it. On a
  narrow page, where the cards scroll and the lines are carried, the
  words are packed at the block's left, a module each, centred on the
  cards' columns as they rest, or a third of the block where three
  modules are wider than it (a phone's), so a word of eleven characters
  keeps the head's size on one line, and
  the details are a row under them. Each line ends on the block's top
  edge, over the stamp's strip and never across it. The page puts the
  vehicle's details over the candidates' options and judges nothing: it
  marks no candidate as the match.
- **On a VIN with problems the block's stamp says so** (#398), after the
  catalog's name on the same strip: a warning triangle and the count, in
  the error colour as tape takes it (`#ff8a8a`), then one of the decode's
  sentences, a bold row value underlined in the tape's white, the whole a
  link to the problems under the answer. The sentence is the first
  problem's that names a cell the plate strikes (error 11's cell 10,
  error 7's cells 1–3, a "!" in the suggested VIN), so the stamp says why
  the red cell is red; where none does, it is the decode's first. On a
  year code no year uses it is error 11's, not error 1's, which comes
  first in the decode. It keeps the strip's one line,
  so the block is no taller: the catalog's name gives way to 40% of the
  strip, and the sentence is cut with "…", the link's name still whole.
  On a wide page whose catalog files the vehicle under no make, where
  the stamp shares the block's row, the link takes the strip across the
  top that a block with a make has, 24 px; the stamp stays beside the
  catalog's word. That block grows, 52 to 77 px on the demo set's vPIC
  catalog, the one place a VIN's problems cost height. *Ruled,
  2026-11-24: accepted.* With the vehicle found and a stock print open, the
  first statement of the problem is in the first window at 1440, 768
  and 390 px, where the strip under the answer was at 1,130, 1,247 and
  1,613 px on a VIN whose year code no year uses.
- **A catalog that files the vehicle under no make**, as one on vPIC's own
  vehicles does, has a block with no make, and the Maker's line is not
  brought down. On a wide page the stamp then takes the first column,
  beside the catalog's word for the vehicle, and the block is a row
  shorter. On a narrow page the stamp stays a strip across the top.
- **Narrow, the cards move and the block does not**, so the lines are
  carried between them by the row's own scroll (a scroll timeline; no
  script reads a position). In the row each line runs on from its card to
  the row's bottom edge. In the 30 px above the block it comes down where
  the row left it, turns (Vehicle's 4 px down, Model year's 13 px, Maker's
  22 px), and runs along to drop into the block over its word. The
  Maker's, whose card and word are both the first, turns deepest, so it
  crosses neither of the others. A line whose column is off the window
  runs to the window's edge. A browser without scroll timelines draws no
  line between the cards and the block.
- **The trunk** is the one line under the block: pinstripe tape in the
  Vehicle group's colour, from the block's bottom edge straight down the
  gutter beside the first column, to the last thing tied to it. That is
  the picker with no candidates listed, and the last candidate with some.
  Each is tied to it by a stub half a row down: the picker's ends on its
  tape, and a candidate's runs into its eyelet. On a VIN with problems a
  stamp of label tape stands at the candidates' head, its stub ending on
  its tape as the picker's does: a warning triangle in the error colour
  as tape takes it, and "Check the VIN before pulling", a micro label, a
  link to the problems under the answer. It is VIN mode's alone, as
  part-number mode asks no VIN, and it stops nothing. The trunk is there
  whenever the picker is, with a part type pressed or none. It is drawn a
  piece an item, so nothing is measured and it needs no anchor
  positioning. It ends before the shelf rows with no number. Under
  34 rem the last thing tied to it is the names' row (the names, below).
- **The band** under the block holds the picker and, beside it, the
  names' row, which reaches the block's trailing edge. The picker is two
  columns wide on a wide page, 312 px, and two modules under 64 rem,
  19.5 rem, the same 312 px. Under 34 rem nothing fits beside it: the picker is
  as wide as a candidate's row, and the names are drawn last, under the
  candidates (the names, below).
- **The picker** is one key of the second tape on a short piece of label
  tape, a row high, tied to the trunk by a stub. Over its name the key is
  labelled "Part type". With nothing pressed (no `part.id` in the URL) it
  reads "Choose a part type" with the count of the vehicle's part types
  in the part types' pill, and a blank strip under the band asks for a
  choice. With one pressed it reads the part type's name, cut with "…"
  where it does not fit, and counts its candidates. While its list is
  open the key is pressed. Under 64 rem the key is 45 px high.
- **Clearing is its own key.** Beside the picker, on its tape, with one
  pressed, a key of the plate's metal reads "×" and links to the page
  with none pressed. Nothing else un-presses a part type. Under 64 rem it
  is 44 px wide and 8 px from the picker's key.
- **The names** ("Also known as") are secondary: compact manila tags, a
  tag a vocabulary, in one row beside the picker that scrolls sideways by
  itself. Each hangs from the block's bottom edge by a wire 2 px wide to
  its eyelet. A tag is laid long, a row high, and a whole number of
  columns wide, from two to nine, which the page counts from the tag's
  words; one whose words turn out wider grows past its columns and cuts
  nothing off. A tag's column is no narrower than 10 rem. Its first
  column is its eyelet and the vocabulary's name on a strip of label
  tape, with how many names it holds where it has several. The rest is
  its words, with no labels: each stands over its group's colour, which
  says what it is. The make is over the Maker's, once where every name of
  the tag shares it; then a line a name, the model over the Vehicle's and
  the years its mapping holds for, lighter, over the Model year's. A
  tag's names fill two lines, then a second column, so no tag is taller
  than a row. A dashed underline is a doubt, and only a name with one has
  the "i", a key of label tape. Every name opens a note that says in
  words how it was reached: a doubtful one from its "i", a sure one from
  the name itself, which darkens under a pointer and more while it is
  held, without the travel of a key: it is ink. The note hangs from
  what opened it (dialogs, below). Where tags are out of
  sight past an end of the row, that end fades over 48 px, and the fade
  goes when the row is scrolled to that end; a row that fits has none.
  It needs scroll-driven animation, and without it the row is cut square.
  The row takes focus, so a keyboard scrolls it. A vocabulary with no
  name keeps a tag that says so, after those with names. Every vocabulary
  is on the page: nothing is stacked or kept behind a switch. Under
  64 rem a tag's two names, each 24 px high with its "i", are 8 px apart.
- **Under 34 rem the names are drawn last**, under the candidates and
  over the shelf rows with no number, as rule 4 drops what is secondary.
  The row cannot hang from the block there, so it is the last thing on
  the trunk: its first tag takes a stub of the trunk in its eyelet, as a
  candidate does, and each tag after it is wired to the one before, the
  wire the tags hang by on a wide page turned along the row. The stub's
  first piece, from the trunk to the row's edge, is the row's and stays;
  the rest is the tag's, and scrolls with it under the row's fade. With
  nothing pressed the row is under the blank strip that asks for a
  choice, and the trunk runs past the strip to it. The page writes the
  names after the picker, where a wide page draws them. Where scripts
  run, `plate.js` moves them in the document to where a phone draws
  them, before the rows with no number and "Next page", and back when
  the window widens, so the Tab key and a screen reader go down the page
  as it reads in every browser. Without scripts a browser that knows
  `reading-flow` still takes the Tab key that way; another takes it to
  the names before the candidates, and skips nothing.
- **The picker's list** is the part types dialog, hung from the picker at
  its width, 4 px under it, over the candidates: it moves none of them.
  It has no head, no line saying what it lists and no backdrop, and
  pressing outside it or Escape shuts it, and the focus goes back to the
  picker. It opens with the focus in its filter, a search field, whose
  Escape Safari keeps for the field: `plate.js` shuts the list on it, as
  the other browsers do. Under a coarse pointer, a finger, it opens with
  the focus on the list itself, as part-number mode's dialog does, so a
  phone raises no keyboard over it; a tap on the filter focuses it, and
  clearing the recent part types leaves the focus on the list. It
  always opens downward, and
  it is laid on the page, not on the window. It moves with the page; a
  page too short for it grows longer; and where the window ends before
  the list does, the page scrolls to it where scripts run, the picker
  still in sight above it. It never opens upward, over the plate. It
  holds the filter (with JavaScript), then the part types opened
  recently, then every part type under its headings in one column, each
  a row of the second tape with its count at the far end. It is never
  taller than 30 rem, nor than the window less the picker and 3 rem, and
  scrolls inside itself. It hangs by anchor positioning; a browser
  without it shows the same list as the page's dialog, with its head and
  Close.
- **Order never changes.** The headings are the entries of a part
  vocabulary on the catalog's spine (the one the catalog files its
  fitments under, else the first loaded), each over the part types it maps
  to by name, so a part type two entries map to is under both, and "Other"
  last over the ones none maps to; an entry mapping to none of the
  vehicle's has no heading. They are the dialogs' section style in the
  entry's own case. With no part vocabulary the list is one, by name.
  Pressing a part type does not move it or any other.
- **The pressed part type** is pressed in the list, flush on `--alu-in`.
  Its link is the page as it is: pressing it again shuts the list and
  changes nothing.
- **The filter** is the page's recessed field at the head of the list.
  Typing narrows the list by name and hides a heading left with none.
  None matching says so under it.
- **Recent part types**, with JavaScript, are kept in the browser as
  recent VINs are, the last eight, newest first, and nothing is kept when
  storage fails. They are the list's alone: it leads with every one the
  vehicle has, under "Opened recently" with its "Clear recent", in the
  order they were opened. None sits on the picker's tape. The VINs menu's
  "Clear recent" forgets them and the VINs.
- **A choice keeps the focus on the picker**, and so does clearing.
- **The candidates** are rows down the page, in the answer's order, each a
  manila tag laid long, tied to the trunk by a stub into its eyelet. A
  row is one grid on the columns, so its parts line up down the list and
  are scanned down. After the tag's lead comes its number on label tape,
  with its side and its years at the tape's trailing end, two columns;
  then, in the tag's ink, its option text (verbatim, each application's
  on a line of its own), its marks and its note, one run of lines; then
  its stock
  strip, the last column. A candidate has no group and takes no colour. A
  row is a row high, and grows only where that run needs another line,
  or a kind of stock is unavailable (the stock
  strip, below). Under 64 rem a row is 3.5 rem and a finger's. Its
  side and its years are under its number there, so a number never
  breaks for want of room. The number's tape is two sixths of the row
  less 8 px and the lead, and no narrower than 9.5 rem; the strip one
  sixth less 8 px, and no narrower than 7.5 rem. The ink has what is
  left: 363 px of a row at 768 px, and 261 px at 600, room for 28
  characters of option text a line. The tape and the strip are a row
  high at the row's top: a row the ink makes taller does not stretch
  them. A mark's stamp wraps before it leaves
  the ink. Under
  34 rem a row is two lines: its number and its stock strip, a row high,
  then its ink.
- **Marks are ink on the tag**, in its ordinary ink: a solid stamp, a little
  askew, with a tick, "Checked" and the decoded elements that met the
  catalog's conditions; a dashed stamp with a warning triangle, "Not
  checked" and what could not be; "Match the options yourself" and "No
  conditions", plain words in capitals with no box, when nothing was either:
  in the tag's ink, as a stamp is, so that they stand apart from the muted
  label of the note after them; the first where
  the catalog gives option text, which the page cannot read and the reader
  matches against the vehicle; and a solid stamp with an
  arrow, "By redirect" and the part type the catalog lists it under. A model
  year confirmed alone earns no stamp: every candidate's years held. The
  marks follow the option text on its last line, and wrap to the next
  only where that one is full; the note follows them. No width asks for
  that: a candidate whose ink fits one line is a row high wherever its
  row has the room. They began the line under the option text, which
  made every candidate with marks two lines high.
- **Weakest wins.** Each of a candidate's applications earns its own marks,
  and the tag shows their union, each kind once with its elements merged, so
  it never claims more than the weakest way the candidate fits: an
  application checked by Drive Type beside one with no condition reads "No
  conditions" and "Checked · Drive Type", never Checked alone. The weaker
  come first: Not checked, Match the options yourself, No conditions, then
  Checked,
  then By redirect.
- **More candidates** than a page holds (30) are a "Next page" link away.
- **The stock strip** is the trailing end of every tag, and the only part
  of it that presses: the second tape, a cell for each kind of stock
  loaded, its count over its name, and a chevron. A screen reader is told
  it once, in words: "Stock · 2 shelf · 2 yard" and the number. Every
  candidate arrives with its totals from the candidates' one request,
  which asks for the stock loaded beside the catalog. A kind none of
  whose sources answered reads a dash over "yard", with "unavailable" on
  a line under it, never 0: a 0 is a loaded source with none. That cell
  is wider and a line taller than a count's, so the strip is as wide as
  it asks, 167 px for one kind and 163 under 64 rem, the one state in
  which a strip is wider than its column, and the cells stand on their
  counts' line. It was one line that ran off the tag, and on a narrow
  page took the window sideways. Under 34 rem the wider strip takes its
  width from the number's tape: a number of eleven characters keeps one
  line down to 378 px, and breaks under that. With stock loaded and
  none for the
  candidate it is "Stock · none", muted ink beside a 2 px dashed rule,
  and presses nothing; with none loaded it is "Stock · not loaded" with
  "Add stock" underlined, and opens the Vocabularies dialog.
- **A press shows what it opened.** The picker's list, a part type's
  candidates and a strip's rows open under the control pressed, often
  below the window. Where scripts run the page scrolls down just far
  enough to bring in the list, a part type's first two candidates or the
  sentence in their place, or the strip's rows. It never scrolls so far
  that the picker, or the open candidate's tag, leaves the top, and never
  up. It scrolls smoothly unless the visitor asks for less motion.
- **The open candidate's rows.** A strip is a link to the page with its
  candidate open (the URL's `fitment.number`, with `fitment.side` for a
  sided fitment and `fitment.type` for one whose type is not the pressed
  part type's); the open one's links to the page with none, so one is
  open at a time. The rows are a print (thermal paper, above), fed out
  from under the candidate's tag, in its own item of the list: the
  candidates after it are under the print. Shelf is over yard on the one
  print at every width, one kind when one is loaded, each under its count
  on a reversed bar and its sources' as-of dates, or why one did not
  answer. A row is its id, its location and the fields its source
  declares, in columns, with what else it has folded under it. A shelf
  row tagged with a retired number says which in a starred line under
  it; the yard's vehicle that is the VIN looked up says "This VIN",
  reversed.
- **Past ten rows** a kind of stock says how many it shows, and "Show all" is a
  link to the page with every row (`rows=all`), swapped in place where
  scripts run.
- **Shelf rows with no number** belong to no candidate: after the first
  page's candidates, a strip of label tape, stuck on, names the part type
  and the vehicle in their own case, over the same print, under their
  count and as-of dates. They hang from no tag. Past the candidates' page
  of them, "Show all" is the page with every one (`rows=unnumbered`).
  They never keep the candidates' "Next page". "Show all" leaves the
  focus on the strip's heading. The section itself takes none: where
  the page's order is drawn (`reading-flow`), a part of it that only a
  script can focus keeps the Tab key out of everything in it.

## Good enough: empty places

Where the page has no answer, the place stays, as a blank strip: a 3 px dashed
`--faint` outline with one sentence in `--muted`. A control in it is a key of
metal; a way on is a link. Under the cards a blank strip is set on the
columns and is at least a row high.

- **The names' place**, in the band: no vocabulary loaded, with a key that
  opens the Vocabularies dialog, or a model not known yet. A vocabulary
  with no name for the vehicle is not a blank strip: its tag says so.
- **The parts that fit:** no catalog loaded, with a key that opens the
  Vocabularies dialog; the model not known yet; the catalog lacks the
  vehicle, naming both, with a link to the catalog's lines; the catalog has
  the vehicle and nothing for its year, with a link to its lines for the
  vehicle; nothing pressed; and a pressed part type with nothing for the
  vehicle.
- **A part number's place**, the tag's and its vehicles': nothing typed yet,
  asking for a part type and a number; a number with no part type, asking
  for its part type; no catalog loaded, with the same key; no such part,
  naming the catalog, the part type and the number; a part listed with no
  vehicle, saying so; and a part on more than one side, with a link to
  each.
- **A card** not typed yet is a 2 px dashed outline (manila tags, above).
- **A candidate's stock** with none for it is muted words beside a dashed
  rule at its tag's trailing end, and a kind of stock that answered with
  none says "None." between asterisks on the print (the parts that fit,
  above).

## Good enough: where the brackets and lines run

- A label sits over its cells with a bracket opening downward. A second
  bracket under the plate opens upward. Both are borders, not images.
- **Wide**, a leader line runs from each lower bracket to its card, with
  square turns and one elbow. A second set of lines joins the cards to the
  words of the vehicle's block, where a block is drawn: the Vehicle's
  straight down, the Maker's too where the catalog files the vehicle under
  a make, and the Model year's stepped left.
- Those lines are SVG paths in a 1020-unit box (60 a cell, 170 a card),
  stretched to the stage with `vector-effect="non-scaling-stroke"`. Positions
  come from arithmetic on the grid, and nothing is measured at run time.
- **Narrow**, the stepped lines are not drawn, and neither are the leader
  lines: the cards they would end on move. The lines to the block are
  carried by the row's scroll (the parts that fit, above). Under 34 rem
  the trunk's last stub runs into the eyelet of the names' first tag,
  under the candidates, and a wire joins each tag to the next.
- A newly finished group draws its bracket and line in.
- **The lines from the cards end on the block.** None runs on from it.
  Under the block the one line is the trunk, from the block's bottom edge
  straight down the gutter, with a stub to the picker and to each
  candidate. In part-number mode the number's line ends in the eyelet of
  the part's tag, and the trunk runs from that tag through its vehicles.
- No line ends on a blank strip: with no block, none is drawn from the
  cards, and with no picker there is no trunk.

## Good enough: dialogs

Everything behind a button is a `popover`, opened by `popovertarget` with no
script. There are three kinds.

| Kind | Used for | Width | Backdrop |
| --- | --- | --- | --- |
| Dialog (`dialog[popover]`) | Vocabularies, Developers, JSON, the check digit's sum, part-number mode's part types | `min(56rem, 100vw - 2rem)` | `rgb(10 18 26 / .55)` |
| Note (`.note[popover]`) | How a name on a tag was reached | up to `30rem` | `rgb(10 18 26 / .35)` |
| Menu (`.menu[popover]`) | The VINs to try | `19rem` | none |

- **Panel:** panel ground and `--fg`, a 2 px `--steel` edge, on 3 px of
  `--steel-dark` and a deep soft shadow. Corners are 6 px on a dialog and 4 px
  on a note or a menu.
- **Head** (dialogs only): a bar of metal across the top with a slotted screw
  at each end, the title struck in the label face at 13 px, and a Close button
  at its far end. It is sticky, so Close stays in reach while the panel
  scrolls.
- **Height:** a dialog keeps inside the window (`100dvh - 2rem` where the
  browser knows it) and scrolls inside itself.
- **The menu hangs from its button** by CSS anchor positioning, and from the
  page's end where the browser has none. Its rows are small metal tags,
  raised as keys are. While it is open, its button is pressed.
- **A note hangs from the name that opened it** (rule 8): 6 px under it,
  centred on it, moved along only as far as keeps it 1 rem inside the
  window, and over the name where the window ends first. It is anchored
  to its button, with no script. A browser without anchor positioning
  opens it in the window's middle, as every browser did.
- **The part types dialog** is one dialog in two places. Part-number mode
  opens it as a dialog, with its head, a line saying what it lists, and
  its part types under their headings in two columns. On a VIN's page it
  is the picker's list, hung from the picker (the parts that fit, above).
- **Each key is where it belongs, the same at every width** (rules 5, 6
  and 7). No menu holds the page's keys and no row of them scrolls.
  Vocabularies is the header's, at its far end, as it is about the whole
  page. VINs is the plate's: it ends the row over the plate, whose start is
  the mode switch, and on a phone the switch gives it room. Developers and
  the release are the page's foot, under a rule: they are about nitrovin,
  not about the vehicle. *Sent back: a Menu key that held all three and
  the release on a phone; one row of keys that scrolled sideways.*
- **The VINs menu is two lists and a key**, each list under a small label,
  each row a small metal tag of two lines: a vehicle, and under it what it
  has. "Recent" is this browser's last eight vehicles, each with the part
  type that was open on it, so a row is back at its stock; "In stock" is
  the vehicles the yards hold most of, a model once, with the yard's count
  (ADR 0028), and is there whether or not anything is recent. The examples
  show only until there is a recent vehicle. Surprise me opens a vehicle
  of the yard's when a yard is loaded. *Sent back: the recents as a row of
  their own over the plate.*
- **Update now is the header's one lit key.** When a new release is
  there, a key shows beside the brand, at every width: a key like the
  others, its rivet an amber lamp (`--lamp`)
  and a ring of the lamp's light around it, which breathes where motion is
  allowed. It is a control, so it is metal (rule 2), and it sits in the
  masthead and not over the page (rule 1): no toast. The lamp says "new";
  the key's words say only what a press does (rule 3). No group's colour
  is amber. It is hidden the rest of the time, and the Vocabularies dialog
  holds no update button.
- **Sections** inside a dialog are `h3`: the label face at 12 px over a 3 px
  rule in the vehicle group's blue.
- **A dialog inside a card** (the check digit's sum) takes the panel's colours,
  not the manila's.

## Good enough: forms and inputs

The page has one form that matters, the plate, and it is described above.
Every other control is plain and small.

- **Button:** a key of the plate's metal. The label face at 11 px, upper case,
  `--on-metal`, a 1 px `--steel` edge, `--raised`. Pressed, it is flush
  (ground rules: three heights). A button in the header or a row in the menu
  adds a rivet. The exceptions are on a manila tag: a button there is label
  tape in the tag's colour, and a candidate's "Stock · not loaded" is its
  stock strip, the second tape.
- **Link-like button** (`.linkish`, "Clear recent"): no metal. `--link`,
  underlined, in the body face.
- **Text and search fields:** a recess. Panel ground, a 1 px `--steel` edge,
  2 px corners, and an inset shadow at the top. The text is the body face.
- **Label:** weight 600, above its field. A hint beside or under it is
  `--muted` at weight 400.
- **Text to copy** (`pre`): the same recess on a slightly darkened ground, in
  the monospace face at 13 px. `copy.js` adds a Copy button after each; without
  the script the text stays to select.
- **File loader** (the Vocabularies dialog): a 2 px dashed `--steel` box
  holding a native file input, its label and a hint. While files are dragged
  over the dialog the dashes turn `--link`.
- **Offer** (Load demo, Load EPA): a 2 px `--line` box holding one button and
  a hint.
- **Filter** (a file's entries, a catalog's lines): each field a label over a
  search field, then a Filter button and a "Show all" link, in a wrapping
  row. A catalog's lines filter on the part type and the vehicle's fields.
- **Inline form** (Remove, in a table row): a form that is only its button.
- **Disclosure:** `details` with a weight-600 `summary`, in a 1 px `--line`
  box. In a table row it has no box. On a stock print it has neither box
  nor weight: its summary is a folded value, cut to two lines (thermal
  paper, above).
- **Table:** rows ruled 1 px in `--line`. The header row is the label face at
  11 px over a 3 px `--steel` rule. A page's own table (a vocabulary's
  entries) sits in a panel like a note's: panel ground, a 2 px `--steel` edge,
  4 px corners, on 3 px of `--steel-dark` and a soft shadow.
- **Focus:** every link, button, summary, field and `pre` takes a 3 px
  outline, 2 px off, in the ring's colour for what it is drawn on
  (Tokens, above). The plate takes it 3 px off. A stock strip is its
  tag's edge, so its ring is inside it, 6 px in, on its metal and clear
  of its ledge, as the cards' row's is inside its region under 64 rem.
  The mode switch's focused key lies over its neighbour, so its ring is
  whole.
- **Errors:** `--bad` text. A refusal and the page's error slot add a 4 px
  `--bad` rule down the left. Under the plate a refusal is the last line
  of the parts that fit, or of a part's answer, on the columns. What the
  decoder says is wrong with a whole VIN is not an error of the page's:
  it is on label tape (vinyl tape, above), and the cells it names are
  struck (the VIN input, above).
- **Targets:** a link or a button in a table row is at least 24 px high,
  and so is a key on a card, and a link in a sentence under the cards
  ("Show all", "Next page", a blank strip's way on), whose line is no
  taller for it. Under 64 rem, where a finger presses, the picker's key
  and a stock strip are 44 px high or more, as are the keys over the
  plate, and two targets that are not joined are 8 px apart. The mode
  switch's two keys are joined, by design. The header's and the foot's
  keys are 26 to 28 px high at every width.

## Building something new

- A new control is a piece of metal. Reuse `--metal`, `--raised` and
  `--emboss`, and pick its fastener from the table above. On a manila tag it
  is a strip of label tape in the tag's colour, unless it names what it
  opens and is the tag's trailing end, as the stock strip is. Either way
  it stands at the controls' one height, and pressed it is flush.
- A new static strip of tape is stuck on, with `--lift-stuck`, never raised.
- A new name from a vocabulary is ink on a manila tag, after its
  vocabulary's name; one that can be pressed is the second tape.
- A new fact about the VIN itself is a row on a tag, under its group's band.
- A new connection between two things is a line of pinstripe tape. Under
  the vehicle's block it is a stub to the trunk: no second line runs down
  the page.
- A new list under the vehicle's block is rows down the page, set on the
  cards' columns with the page's inset, spaces and row height (Tokens,
  above).
- A new part under the cards keeps one shape at every width. Where it
  must change, it changes at one of the page's three widths, for what
  stops fitting there (Tokens, "Widths"), and what a finger presses is
  44 px high under 64 rem.
- New text under the cards takes a type role (Tokens, above). It writes no
  size, weight or tracking of its own, and data is the monospace.
- A fifth material needs a reason, as manila and thermal paper had:
  something the page hangs, sticks, bolts on or prints that the four cannot
  be.
- A seventh colour needs a reason. The six group colours are the only hues
  besides `--link` and `--bad`.
- Check it against each composition rule above.
- Check both colour schemes, a wide window (1280 px), a narrow one (768 px), a
  390 px one and a 320 px one, and the page with JavaScript off. Check it
  with as much on
  the page as a deployer has: several vocabularies, a vehicle with many
  part types, a long body or engine, a stock file of a dozen fields. The
  demo set alone hides what crowds.
