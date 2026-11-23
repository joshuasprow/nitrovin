# The page's design

The page at `/` (`pkg/ui`) looks like a stamped VIN plate with tags hung from
it. This file states which parts of that look are the design, and which parts
are settled enough to leave alone. It describes; the code is the source:
[ui.css](pkg/ui/static/ui.css), [page.html](pkg/ui/page.html),
[plate.js](pkg/ui/static/plate.js), [brushed.svg](pkg/ui/static/brushed.svg) and
[paper.svg](pkg/ui/static/paper.svg).
The references behind the choices are in
[docs/research/vin-page.md](docs/research/vin-page.md) and
[docs/research/textures.md](docs/research/textures.md).

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
  motion that loops is the caret's blink on the focused plate, once a second.

## Composition rules

The ground rules say what the page is built with. These say how a change
is put together, and each came from a correction: a version that was built,
looked at, and sent back. A change is checked against every one before it
is shown. One that breaks a rule says which, and why, where it is proposed.

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
   candidate. Thermal paper is what a stock file says of a thing found: a
   receipt, a print and not the thing. Metal is a control. Pinstripe tape is a connection. Before
   content is placed, it is one of these, and it takes that material.
   *Sent back: names as strips of label tape. A name is found, so it is a
   tag.*
3. **Colour and texture before words.** A group's colour says which card a
   word comes from, so no label repeats it. A label stays only where
   nothing else says what a value is, as "Body" and "Engine" do. Nothing
   in one view says the same thing twice.
   *Sent back: "VIN 4–8" over five characters that each carry their
   position; "Make" and "Model" over words already underlined in the
   Maker's and the Vehicle's colours.*
4. **One thing leads; the rest sit on it.** A block has one primary thing.
   What serves it sits on the same piece of material, smaller, after it,
   and is dropped first when the window narrows. It never gets a block of
   its own beside the primary.
   *Sent back: recent part types as a strip of their own under the picker,
   then as a second row of the picker's tape. They are on the tape's one
   line, and a phone's tape is the picker alone.*
5. **One shape at every width.** A component is the same thing at 390,
   768 and 1280 px. It wraps or scrolls; it does not turn into another
   kind of control. A control has a largest width and keeps it: it does
   not stretch to fill a wide window.
   *Sent back: a strip of part types on a wide page and a list on a phone.
   Both became the one picker, no wider than 24 rem.*
6. **A row scrolls sideways before it stacks or hides.** Things of a kind
   are one row. Past the window, the row scrolls by itself, shows the next
   one at its edge, and fades at an end with more beyond it. Nothing is
   kept behind a "more" switch, and nothing is stacked to be fanned out.
   On a phone an item of a list is compact: several show at once.
   *Sent back: names in stacks behind two switches; a list of part types
   that showed four on a phone.*
7. **What belongs together is joined, and a line crosses nothing.** A line
   runs from a card to its word, and from what was asked to what answers
   it; a tag hangs from what it names; a control's tape touches the
   control. Nothing is over a line or under it: no tag, control or other
   line. A line runs straight from the word it leaves to the control it
   reaches, which sits under that word; it does not run off to a lane at
   the page's edge.
   *Sent back: a picker apart from its tape; lines to the picker running
   behind the name tags; then the same lines sent down the page's left
   edge to clear them.*
8. **What opens hangs from its control and overlaps the page.** A list or a
   note opened from a control is anchored to it, no wider than it where it
   is a list, and lies over what is beneath. It does not push the page
   down.
   *Sent back: a list of part types as wide as the page that moved the
   candidates.*
9. **Bracket ends are the plate's.** Only the VIN input's brackets end in
   a bracket. Every other line ends on the thing it reaches.

## Tokens

All tokens are custom properties on `body` in `ui.css`.

| Role | Light | Dark |
| --- | --- | --- |
| Ground `--bg` | `#dcecf7` | `#16222d` |
| Text `--fg` / `--muted` | `#14181d` / `#47515c` | `#eef2f5` / `#a9b6c2` |
| Panel ground `--card-bg` | `#ffffff` | `#1f2d3a` |
| Rule `--line`, not-yet `--faint` | `#9fb3c2`, `#a9bccb` | `#45566a`, `#3a4a5c` |
| Link and focus `--link` | `#0b4fa8` | `#8ec5ff` |
| Error `--bad` | `#b91c1c` | `#ff8a8a` |

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

Two typefaces carry the look, and both are system fonts.

- `--label-font` (Arial Black, then Helvetica Neue): everything stamped or
  printed as a heading. Weight 900, upper case, tracked from `.04em` to
  `.22em`, mostly 11–13 px.
- `--cell-font` (`ui-monospace`): everything that is data to read or copy: the
  VIN's characters, serials, code, and the words on the tape.

Body text is `system-ui` at 15 px.

## Signature: the VIN input

The form is the plate. It is 17 cells under six group labels, and one real
text field laid over them.

- **One native field.** `plate.js` lays a transparent `textarea` over the
  cells. The field keeps paste, selection and the phone's keyboard. The
  cells draw the text, the caret and the selection; the field shows none of
  its own. Without JavaScript the field is a plain input below the cells with
  a Decode button.
- **A fixed grid of 17 columns with no gaps.** Labels, cells and brackets
  share it. The field's letter-spacing is `100cqw / 17 - 1ch`, so each
  character sits on its cell and a click lands on the cell it shows.
- **A cell** is a struck character: monospace 700, `--on-metal`, lit from above
  by `--emboss`, over a 4 px underline in its group's mark. Its size follows
  the plate's width through container units.
- **Empty**, the plate reads "Type a VIN" in the label face across its middle.
- **Focused**, the plate takes the 3 px `--link` outline, 3 px off every side
  of what is seen, and the next empty cell blinks a 6 px bar of `--link` along
  its bottom. The brackets above and below stand 7 px off the plate, clear of
  the ring.
- **The caret among characters** is a 4 px bar in the gap before its cell.
  Under a mouse, the gap a click would pick shows the same bar at 35%.
- **A selected cell** is inked: `#16202b`, white character, ringed 3 px in its
  group's colour.
- **A character no VIN holds** fills its cell with `--bad`. A bad VIN is not
  an error: the plate still decodes, and once the VIN is whole its problems
  are listed under the answer.
- **A typed character stamps** into its cell (scale 1.6 to 1, 0.18 s).
- **Narrow windows keep the plate.** Under 34 rem the cells wrap to two rows of
  nine, breaking after the check digit, so a cell stays wide enough for a
  finger. Under a coarse pointer the selection handles are the cells' own,
  and a long press selects the whole VIN, drawn on the cells; the phone
  draws no handles of its own over them.
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
  the same history entry, and Look up (Enter) makes one of its own.
- **Under the plate** the number's bracket opens upward, flat and faint
  until a number is typed. One line of pinstripe tape in the Serial group's
  colour runs from its foot to the eyelet of the part's tag: an SVG path in
  the leader lines' 1020-unit box, from 680 (the number's third's centre)
  to the tag's centre, 170 on a wide page and 255 on a narrow one, and on
  down into the eyelet. Under 34 rem it runs straight down. No line is
  drawn without a tag. Each width draws its own path; nothing is measured.
- **The part** is a manila parts tag in the first third (the first half on
  a narrow page), as a candidate's:
  its number on black label tape under the eyelet with its part type's name
  at the tape's trailing end, then its side, how many vehicles it fits in
  the catalog, its note, and its stock strip along its bottom edge. A
  retired number answers with the part that replaced it: its tag carries a
  solid ink stamp, "Replaces" and the number asked for, and the catalog's
  note on the retirement as a row.
- **Its stock** behaves as a candidate's: the strip opens the shelf rows and
  the yard vehicles among the vehicles listed, named in the URL as a
  candidate's are (`fitment.number`). They are a receipt
  under the tag and its width, as a candidate's are, shelf over yard at
  every width (thermal paper, below). The rows are the page's, paged with
  the vehicles; "Show all" is every one.
- **The vehicles** are the page's own table panel beside the tag, with no
  line to it: the catalog's name for each vehicle (a vPIC
  catalog's as VIN mode names one, "vPIC model 1801 (F-150)", where the
  answer carries the name), its years, its option text verbatim, and its
  names in the vehicle vocabularies loaded, each a strip of label tape
  after its vocabulary's name. While the server holds the catalog, a
  browser names them in the server's vocabularies alone, which the
  server's fit knows. A vehicle is no link. More than a page (30) are a
  "Next page" link away. On a narrow page each vehicle is a row of its
  own under the table's caption (name and years, then options, then
  names), and under 34 rem the table follows the tag.
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
  with `--pressed` inside its top. A rivet or the Menu button's bars stay
  on it.

The fastener says what the piece is.

| Fastener | On | Drawn as |
| --- | --- | --- |
| Rivet | Small tags: header buttons, menu rows; one at each end of the brand | A 2.5 px `--steel-dark` dot in a light ring |
| Slotted screw | The VIN plate and every dialog's head, one at each end | A 5 px light disc, a `--steel` ring, a 9 × 2 px slot |
| Eyelet | A manila tag | A hole punched by a radial mask through a darker patch that reinforces it |

Only a dialog's head is metal. The panel under it is panel ground. Text to
copy sits in a recess: an inset shadow on a slightly darkened ground.

## Signature: vinyl tape

Vinyl is three tapes. The page's own word on a thing, a stamp or a tag's
heading, is a strip of black label-maker tape. A name that can be pressed
is the second tape, embossing tape in the plate's metal. A bracket or a leader line is pinstripe tape in
its group's colour.

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
  upper-cased. A vocabulary's own name, on the strip across its tag, and a
  stamp are upper case.
- **A word on the vehicle's block** sits over a 3 px underline in its
  group's mark, on the line that runs straight down into the tape from its
  card. No line ends in a bracket but the plate's own.
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
- **Lettering:** `--on-metal` in the monospace face at 14 px, weight 900,
  struck as the plate's characters are (`--emboss`), in the name's own case.
- **A count** sits after the name in a small pill outlined in `--on-metal`
  at 55%.
- **Raised and pressed** as every control: `--raised` unpressed; pressed
  (`aria-current`, the part type the page shows), flush on `--alu-in` under
  `--pressed`.
- **Along a tag's bottom edge** (a candidate's stock strip) it runs edge to
  edge inside the tag and casts no shadow off it, so it stands raised by its
  edges alone: a white top and 3 px of `--steel-dark` below. Pressed (open)
  it is `--alu-in` under `--pressed`, its bottom edge 1 px, and its chevron
  turns up. Its words are no vocabulary's name but the page's own count of
  stock, so they are 12 px and upper case, as drawn, to sit as a footer
  under the tag's rows, not as large as a part type's name.

### Pinstripe tape

- **Body:** the group's colour, `--c`. A bracket is 4 px with its outer
  corners rounded 2 px. A leader line is 5 px with round joins.
- **Lift:** `--stripe-lift`, two drop shadows: a 1 px light edge up and left,
  and a soft shadow down and right. The stripe reads as stuck on the ground.
- **Not yet typed**, a bracket or a line is flat: 3 px of `--faint`, no lift.
- **A bracket that holds its name** (the lower row on a phone) lies flat, so
  the lift does not edge the letters.

## Signature: manila tags

A decoded card is a manila parts tag, hung by its eyelet from its leader line.

- **Stock:** `--manila`, `paper.svg`'s fine noise at `200px 200px` over
  `linear-gradient(#efdcae, #e2c88f)`.
- **Shape:** the top corners are clipped 16 px by a `clip-path`. The stock is
  the card's `::before`, so the card's own `drop-shadow` follows the shape; a
  `box-shadow` would be clipped away.
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
  second tape, and it runs along the tag's bottom edge, its footer, rather
  than sitting on the paper as a control.
- **Not yet typed**, a card keeps its place as a 2 px dashed `--faint` outline
  with its muted title, and no stock.

## Signature: thermal paper

What a stock file says of a part is a receipt: a strip of thermal paper, fed
out from under the stock strip of the tag it is of, and torn at its foot
(#376). A tag is a thing found; the receipt is a print about it, so it is
neither hung nor stuck. It is drawn with only what a receipt printer does.
"Strip" alone, here and in the stylesheet, is the tag's stock strip, the
control; the paper is the receipt.

- **Stock:** `--thermal` (`#f6f4ee`), the same in both colour schemes,
  shaded for 15 px where it leaves the tag and faintly down each edge. Its
  `drop-shadow` is on the receipt itself, so it follows the torn foot.
- **Print:** `--on-thermal` (`#232229`), one near-black in the page's
  monospace face at 13 px: the system's, since the page ships no font file.
  `--thermal-muted` (`#62606b`) is a note's fainter print. No other colour
  is printed, and a group's colour never is.
- **Width:** a receipt is its tag's width and no wider than 23 rem: about 35
  columns under a candidate's tag on a wide page, 42 at its widest. Where
  three tags share a row narrower than 64 rem it is no narrower than 19 rem,
  and so wider than its tag, under whichever tag is pressed.
- **Head:** "Stock of" and the number, centred, in double height
  (`scaleY(1.7)`).
- **A kind of stock:** its name and count reversed, `--thermal` on a bar of
  `--on-thermal`; its sources' dates on the line under it, at the right;
  and a rule of dashes under both, dated or not.
- **A row:** the id in bold, led by dots to its location; then each field,
  its name led by dots to its value at the right. A location too long for
  the id's line takes the next. A rule of dashes parts two rows. A field is
  printed as its file has it, so a file that repeats a row's id as a field
  prints it twice.
- **A note** is a starred line under its row, in the fainter print and as
  the page writes it, not in capitals: it quotes a number as typed. "This
  VIN" is reversed.
- **Nothing in a kind** is "None." between asterisks, under its bar and date.
- **Past ten rows** a kind says how many it prints, under a rule of dashes,
  with "Show all" underlined.
- **Foot:** a barcode over the number, then a torn edge of 9 px teeth. The
  bars are a fixed pattern and encode nothing.
- **Case:** nothing typed into a stock file changes case on a receipt: an
  id, a location, a field's name and its value are as written. Of the
  page's own words, a kind's name, "Stock of" and "None." are capitals.
- **The head and the foot are print.** They are hidden from a screen reader,
  which is told the same by the region's name.
- **It breaks two composition rules, knowingly.**
  - *Rule 6.* Shelf and yard are one receipt, one over the other, at every
    width, where a row would scroll sideways: a receipt is one column, and
    two side by side are two prints. Rule 5 is kept by it: the receipt is
    the same thing at 390, 768 and 1280 px.
  - *Rule 3.* The part's number is on its tag, at the receipt's head and
    under its barcode. A print says what it is of, and the head is what
    names the part once a long receipt has scrolled its tag away: a press
    keeps the stock strip in sight, not the tag (the parts that fit,
    below).
- **The rows with no number** are a receipt with no head and no foot, under
  the label tape that names them.

## Good enough: what a card holds

Each group has a card under its bracket, six in one row. The page has one
width where the row changes, 64 rem: at 64 rem and up it is wide, and below
that it is narrow.

- The band reads the group's title and its positions ("Vehicle 4–8").
- The body is either label and value rows, or one large value in the monospace
  face. A row is its label, small, with its value under it, both at the
  card's leading edge, so every row reads alike whatever its value's length. The check digit's card opens its arithmetic in a dialog.
  The year's card shows the year, then the codes around its own as one run of
  small text with its own in bold.
- The six cards share one grid row, as tall as its tallest card. Up to four
  rows, eight or nine lines, is usual and is not a limit. No card cuts text off or keeps it behind
  a control.
- A newly finished group pops its card in (0.3 s).
- **Wide**, the six cards span the plate's width, each under its leader line.
- **Narrow**, the six cards are one row that scrolls sideways, in the VIN's
  order, six columns of one width (12.5 rem, and 9.5 rem under 34 rem). The region reaches both edges of the
  window, is named, and takes focus, so a keyboard scrolls it.

## Good enough: the parts that fit

Under the cards, in the page's single column: the vehicle's block, its names
in the vocabularies loaded, a picker of the part types the loaded catalog has
for the vehicle, and the candidates of the one pressed.

- **The vehicle's block** is label tape, stuck on: a strip across its top
  with its stamp, "This vehicle ·" and the catalog's name, as a name tag
  has its vocabulary, then one row. From the left: the make the catalog
  files the vehicle under, in the catalog's word, where it has one; the
  catalog's model the decode reached; and the year it asked with ("any
  year" without one), each over its group's underline, the Maker's red,
  the Vehicle's blue and the Model year's orange, lightened for the tape;
  then what the decode says of the
  vehicle that a catalog's option text names, so a candidate's options
  are matched at a glance: the Vehicle card's rows but the model, a small
  label over each value, raised white on the tape, then the Vehicle
  group's five characters, each over the group's mark, as on the plate,
  with its position under it and no label: the positions say what they
  are. The words are packed at the block's left, a column 6.6 rem wide
  each, at every width, so the picker's key under the block is under them;
  the details take the rest of the row on a wide page and a row of their
  own on a narrow one. The Maker, Vehicle and Model year cards' lines each
  step left to their word and come down onto the block's top edge, over
  the stamp's strip and never across it. A catalog on vPIC's own vehicles
  has no make of its own: the block has none, and the Maker's line is not
  brought down. The page puts the vehicle and the candidates side by side
  and judges nothing: it marks no candidate as the match.
- **The names** ("Also known as") are manila tags, a tag a vocabulary, in
  one row under the picker that scrolls sideways at every width; narrow, it
  reaches the window's right edge. No line passes it. Where tagsWhere tags are out of sight past an end of the
  row, that end fades over 48 px, and the fade goes when the row is
  scrolled to that end; a row that fits has none. It needs scroll-driven
  animation, and without it the row is cut square. A
  tag is a card's stock, hung from the bottom edge of the picker's tape by a wire
  2 px wide from its eyelet, which moves along the edge with it. It is no wider than 72% of the window, so the next
  one shows. Across it, the vocabulary's name on a strip of
  label tape, with how many names it holds where it has several. Under
  that its words, with no labels: each stands over its group's colour,
  which says what it is. The make is over the Maker's, once where every
  name of the tag shares it; then a line a name, the model over the
  Vehicle's and the years its mapping holds for, lighter, over the Model
  year's. A tag's names fill two lines, then a second column, so no tag
  is taller than its strip, a make and two names. A dashed underline is a doubt, and only a name with
  one has the "i", a key of label tape. Every name opens a note that says
  in words how it was reached: a doubtful one from its "i", a sure one
  from the name itself, which darkens under a pointer. A vocabulary with no
  name keeps a tag that says so, after those with names. Every vocabulary
  is on the page: nothing is stacked or kept behind a switch.
- **From the block to the picker**, at every width, the Vehicle's and the
  Model year's lines leave the block's bottom edge under their words and
  come straight down the 30 px into the picker's key, and end there. No
  line runs on from the picker: the candidates under it are what it
  names. With nothing pressed no line leaves the block. They need anchor
  positioning.
- **Narrow, the cards move and the block does not**, so the lines are
  carried between them by the row's own scroll (a scroll timeline; no
  script reads a position). In the row each line runs on from its card to
  the row's bottom edge. In the 30 px above the block it comes down where
  the row left it, turns (Vehicle's 4 px down, Model year's 13 px, Maker's 22 px), and
  runs along to drop into the block over its word. The Maker's, whose card
  and word are both the first, turns deepest, so it crosses neither of
  the others. A line whose column is
  off the window runs to the window's edge. A browser without scroll
  timelines draws no line between the cards and the block.
- **The picker** is one key of the second tape on its own piece of label
  tape, under the block and as wide as it, the key at its left end, at most
  24 rem wide, under the block's words. Everything on the tape is on one
  line: the key, the key that clears it, then recent part types, and one
  that does not fit is not shown. Over its name it is labelled "Part type". With nothing pressed
  (no `part.id` in the URL) it reads "Choose a part type" with the count
  of the vehicle's part types in the part types' pill, and a blank strip
  under it asks for a choice. With one pressed it reads the part type's
  name, cut with "…" where it does not fit, and counts its candidates.
- **Clearing is its own key.** Beside the picker, with one pressed, a key
  of the plate's metal reads "×" and links to the page with none pressed.
  Nothing else un-presses a part type.
- **The picker's list** is the part types dialog, hung from the picker at
  its width, over whatever is under it: it pushes nothing down. It has no
  head and no backdrop, and pressing outside it or Escape shuts it. It
  holds the filter (with JavaScript), then those opened recently, then
  every part type under its
  headings in one column, each a row of the second tape with its count at
  the far end. It is never taller than half the window less the picker,
  so it fits under the picker, or over it where there is no room under,
  and scrolls. It hangs by anchor positioning; a browser without it shows
  the same list as the page's dialog, with its head and Close.
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
  storage fails. The picker's list leads with every one the vehicle has,
  under "Opened recently" with its "Clear recent", at every width. Above
  46 rem, up to two, never the pressed one, also sit on the picker's own
  tape, beside it after a hairline and the word "Recent", each a smaller
  strip of the second tape with its count: pressing one presses it. A
  phone's tape is the picker alone. The VINs menu's "Clear recent" forgets
  them and the VINs.
- **A choice keeps the focus on the picker**, from the list or from under
  it, and so does clearing.
- **The candidates** are manila tags, three to a row, in the answer's
  order. Under 34 rem they are one row that scrolls sideways, the next tag
  showing at the edge; the row reaches both edges of the window, takes
  focus, and starts at the open candidate where the browser can. A tag's number is a strip of label tape
  across it under the eyelet, where a card prints its band, with the years at
  its trailing end; a candidate has no group and takes no colour. Its side,
  option text (verbatim) and note are rows in the tag's ink under muted
  labels.
- **Marks are ink on the tag**, in its ordinary ink: a solid stamp, a little
  askew, with a tick, "Checked" and the decoded elements that met the
  catalog's conditions; a dashed stamp with a warning triangle, "Not
  checked" and what could not be; "Match the options yourself" and "No
  conditions", plain muted words, when nothing was either: the first where
  the catalog gives option text, which the page cannot read and the reader
  matches against the vehicle; and a solid stamp with an
  arrow, "By redirect" and the part type the catalog lists it under. A model
  year confirmed alone earns no stamp: every candidate's years held.
- **Weakest wins.** Each of a candidate's applications earns its own marks,
  and the tag shows their union, each kind once with its elements merged, so
  it never claims more than the weakest way the candidate fits: an
  application checked by Drive Type beside one with no condition reads "No
  conditions" and "Checked · Drive Type", never Checked alone. The weaker
  come first: Not checked, Match the options yourself, No conditions, then
  Checked,
  then By redirect.
- **More candidates** than a page holds (30) are a "Next page" link away.
- **The stock strip** is the bottom edge of every tag, and the only part of
  it that presses: the second tape, "Stock · 2 shelf · 2 yard", naming only
  the kinds of stock loaded, with a chevron at its trailing end. Every
  candidate arrives with its totals from the candidates' one request, which
  asks for the stock loaded beside the catalog. A kind none of whose
  sources answered reads "yard unavailable", never 0: a 0 is a loaded
  source with none. The number above it stays black label tape. With stock
  loaded and none for the candidate it is "Stock · none", muted ink over a
  2 px dashed rule, and presses nothing; with none loaded it is "Stock ·
  not loaded" with "Add stock" underlined, and opens the Vocabularies
  dialog.
- **A press shows what it opened.** A part type's candidates and a strip's
  rows open under the control pressed, often below the window. Where
  scripts run the page scrolls down just far enough to bring them in, never
  so far that the part types or the pressed stock strip leave the top, and
  never up. It scrolls smoothly unless the visitor asks for less motion.
- **The open candidate's rows.** A strip is a link to the page with its
  candidate open (the URL's `fitment.number`, with `fitment.side` for a
  sided fitment and `fitment.type` for one whose type is not the pressed
  part type's); the open one's links to the page with none, so one is
  open at a time. The rows are a receipt (thermal paper, above), fed out
  from under the tag's stock strip: on a wide page in the
  row under the tag's row, under the tag that is pressed, the tags after
  it filling theirs first; under 34 rem right under the tag, in its column,
  moving with it. Shelf is over yard on the one receipt at every width, one
  kind when one is loaded, each under its count on a reversed bar and its
  sources' as-of dates, or why one did not answer. A row is its id, led by
  dots to its location, and its source's fields verbatim, each name led to
  its value. A shelf row tagged with a retired number says which in a
  starred line under it; the yard's vehicle that is the VIN looked up says
  "This VIN", reversed.
- **Past ten rows** a kind of stock says how many it shows, and "Show all" is a
  link to the page with every row (`rows=all`), swapped in place where
  scripts run.
- **Shelf rows with no number** belong to no candidate: after the first
  page's candidates, a strip of label tape, stuck on, names the part type
  and the vehicle in their own case, over a receipt with no head and no
  foot, under their count and as-of dates. Past the candidates' page of them, "Show all" is
  the page with every one (`rows=unnumbered`). They never keep the
  candidates' "Next page".

## Good enough: empty places

Where the page has no answer, the place stays, as a blank strip: a 3 px dashed
`--faint` outline with one sentence in `--muted`. A control in it is a key of
metal; a way on is a link.

- **The names' place:** no vocabulary loaded, with a key that opens the
  Vocabularies dialog, or a model not known yet. A vocabulary with no name
  for the vehicle is not a blank strip: its tag says so.
- **The parts that fit:** no catalog loaded, with a key that opens the
  Vocabularies dialog; the model not known yet; the catalog lacks the
  vehicle, naming both, with a link to the catalog's lines; the catalog has
  the vehicle and nothing for its year, with a link to its lines for the
  vehicle; nothing pressed; and a pressed part type with nothing for the
  vehicle.
- **A part number's place**, the tag's and the table's: nothing typed yet,
  asking for a part type and a number; a number with no part type, asking
  for its part type; no catalog loaded, with the same key; no such part,
  naming the catalog, the part type and the number; a part listed with no
  vehicle, saying so; and a part on more than one side, with a link to
  each.
- **A card** not typed yet is a 2 px dashed outline (manila tags, above).
- **A candidate's stock** with none for it is muted words over a dashed
  rule on its tag, and a kind of stock with none says "None." between
  asterisks on the receipt (the parts that fit, above).

## Good enough: where the brackets and lines run

- A label sits over its cells with a bracket opening downward. A second
  bracket under the plate opens upward. Both are borders, not images.
- **Wide**, a leader line runs from each lower bracket to its card, with
  square turns and one elbow. A second set of lines joins the vehicle's and
  the year's cards to the block's two words, where a block is drawn.
- Those lines are SVG paths in a 1020-unit box (60 a cell, 170 a card),
  stretched to the stage with `vector-effect="non-scaling-stroke"`. Positions
  come from arithmetic on the grid, and nothing is measured at run time.
- **Narrow**, the stepped lines are not drawn, and neither are the leader
  lines: the cards they would end on move.
- A newly finished group draws its bracket and line in.
- **The lines never stop short.** Vehicle's and Model year's run from the
  card into the vehicle's block, and from it to the picker and its
  candidates' eyelets, at every width. Maker's ends at its card: a name's
  make carries the Maker's colour instead of a line.
- No line ends on a blank strip: with no block, none is drawn from the cards.

## Good enough: dialogs

Everything behind a button is a `popover`, opened by `popovertarget` with no
script. There are three sizes.

| Kind | Used for | Width | Backdrop |
| --- | --- | --- | --- |
| Dialog (`dialog[popover]`) | Vocabularies, Developers, JSON, the check digit's sum, the part types | `min(56rem, 100vw - 2rem)` | `rgb(10 18 26 / .55)` |
| Note (`.note[popover]`) | How a name on the tape was reached | up to `30rem` | `rgb(10 18 26 / .35)` |
| Menu (`.menu[popover]`) | The VINs to try | `19rem` | none |
| Header menu (`nav.top[popover]`) | The header's buttons and the release, at 46 rem and under | `16rem` | none |

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
  header's end where the browser has none. Its rows are small metal tags,
  raised as keys are. While it is open, its button is pressed.
- **The header collapses where its buttons would wrap** (46 rem, its own
  width: the cards change at 64 rem). The
  header's nav is itself a popover. In a wide window it has no box: the
  release sits beside the brand and the buttons at the far end. In a narrow
  one the Menu button, a key of metal with three bars, opens it as a panel
  hung from the button: the buttons in a column, then the release under a
  rule. The VINs menu and the dialogs open from inside it.
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
- **Switch without a script:** a visually hidden checkbox whose label is the
  control, as on a stack of label tape. Off, it stands raised; on, it is
  pressed. A stack's stamp on a wide page is the one switch that lies level
  with its strip (vinyl tape, above).
- **Disclosure:** `details` with a weight-600 `summary`, in a 1 px `--line`
  box. In a table row it has no box.
- **Table:** rows ruled 1 px in `--line`. The header row is the label face at
  11 px over a 3 px `--steel` rule. A page's own table (a vocabulary's
  entries) sits in a panel like a note's: panel ground, a 2 px `--steel` edge,
  4 px corners, on 3 px of `--steel-dark` and a soft shadow.
- **Focus:** every link, button, summary, field and `pre` takes a 3 px
  `--link` outline, 2 px off. The plate takes it 3 px off.
- **Errors:** `--bad` text. A refusal and the page's error slot add a 4 px
  `--bad` rule down the left.
- **Targets:** a link or a button in a table row is at least 24 px high.

## Building something new

- A new control is a piece of metal. Reuse `--metal`, `--raised` and
  `--emboss`, and pick its fastener from the table above. On a manila tag it
  is a strip of label tape in the tag's colour, unless it names what it
  opens and runs along the tag's edge, as the stock strip does. Either way
  it stands at the controls' one height, and pressed it is flush.
- A new static strip of tape is stuck on, with `--lift-stuck`, never raised.
- A new name from a vocabulary is on a manila tag, under a heading of
  label tape; one that can be pressed is the second tape.
- A new fact about the VIN itself is a row on a tag, under its group's band.
- A new connection between two things is a line of pinstripe tape.
- A fifth material needs a reason, as manila and thermal paper had:
  something the page hangs, sticks, bolts on or prints that the four cannot
  be.
- A seventh colour needs a reason. The six group colours are the only hues
  besides `--link` and `--bad`.
- Check it against each composition rule above.
- Check both colour schemes, a wide window (1280 px), a narrow one (768 px), a
  390 px one, and the page with JavaScript off. Check it with as much on
  the page as a deployer has: several vocabularies, a vehicle with many
  part types, a long body or engine. The demo set alone hides what
  crowds.
