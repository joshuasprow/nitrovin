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
| **Signature** | The VIN input, and the three materials: aluminum plates, vinyl tape, manila tags | These are the page's identity. New UI is built from them, and a change keeps every rule below. |
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
- **Two colour schemes, three fixed materials.** The ground, the text, the
  dialogs and the group colours change between light and dark. The metal, the
  tape and the manila do not: each is the same object under either light.
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
  finger. Under a coarse pointer the selection handles are the cells' own.

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

Vinyl is three tapes. A name from a vocabulary is a strip of black
label-maker tape. A name that can be pressed is the second tape, embossing
tape in the plate's metal. A bracket or a leader line is pinstripe tape in
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
- **Case is kept.** A vocabulary's entry is case-exact, so the tape never
  upper-cases a name. The vocabulary's own stamp and the "Also known as"
  heading are upper case.
- **Four columns at fixed fractions** (130, 170, 465 and 255 of 1020): the
  vocabulary's stamp, then make, model and years. Each word sits over a 3 px
  underline in its group's mark, under a bracket in its group's colour, on the
  line that runs down from its card. On a narrow page the strip lies under the
  first four of the cards' six columns, and the stamp is a label above it.
- **A dashed line, bracket or underline is a doubt.** The strip's note, behind
  its round "i" button, says the doubt in words.
- **Several names stack.** The first strip is on top, with the edges of up to
  three more under it. The stamp is the switch and carries the count. Opened,
  the stack is one strip with a row to a name.
- **A switch of tape stands raised.** "N more vocabularies", and on a narrow
  page a stack's stamp on its line of its own, keep a 1 px bottom edge on the
  controls' ledge and shadow. On, each is flush, and its fill is `--tape-in`,
  the tape's body darkened to `#0c0e10`–`#1d2125`. On a wide page a stack's
  stamp is a column of its strip, and lies level with it.
- **An empty place stays.** A vocabulary with no name, or none loaded, keeps a
  blank strip: a 3 px dashed `--faint` outline with a sentence in it (empty
  places, below).
- **On a tag the tape takes the tag's colour.** A control on a manila tag is
  the same ridged tape with the group's mark as its body, edges mixed lighter
  and darker from that colour, and a faint white halo on its letters, as
  vinyl whitens where it is struck. "Check years" is the same tape in the
  year's colour. Each stands raised, its own bottom edge 1 px, and pressed it
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

## Good enough: what a card holds

Each group has a card under its bracket, six in one row. The page has one
width where the row changes, 64 rem: at 64 rem and up it is wide, and below
that it is narrow.

- The band reads the group's title and its positions ("Vehicle 4–8").
- The body is either label and value rows, or one large value in the monospace
  face. A row is its label and its value on one line; a long value wraps
  under that line. The check digit's card opens its arithmetic in a dialog.
  The year's card shows the year, then the codes around its own as one run of
  small text with its own in bold.
- The six cards share one grid row, as tall as its tallest card. Four or five
  lines is usual and is not a limit. No card cuts text off or keeps it behind
  a control.
- A newly finished group pops its card in (0.3 s).
- **Wide**, the six cards span the plate's width, each under its leader line.
- **Narrow**, the six cards are one row that scrolls sideways, in the VIN's
  order, six columns of one width (12.5 rem, and 9.5 rem under 34 rem). The
  names of "Also known as" are in the same region and the same columns, so a
  swipe moves them with the cards. The region reaches both edges of the
  window, is named, and takes focus, so a keyboard scrolls it.

## Good enough: the parts that fit

Under "Also known as", in the page's single column, the part types the loaded
catalog has for the vehicle, and the candidates of the one pressed.

- **The strip** is label tape, stuck on: its stamp reads "Parts that fit ·"
  and the catalog's name, over the names' stamp and make columns (300 of
  1020); the catalog's model the decode reached and the year it asked with
  ("any year" without one) sit under the names' model and years, each over
  its group's underline and under its group's bracket, on the lines from
  Vehicle and Model year that run on through the names. No line runs from a
  blank strip, or past the names' "more" switch. Narrow, the stamp is a line
  of its own above the two words and no line runs to it.
- **The part types** are the second tape, a wrapping row, by name A to Z,
  each with its count. Each is a link to the page with it pressed (the URL's
  `part.id`); the pressed one's link presses nothing. Nothing is pressed
  without the parameter, and a blank strip asks for a press.
- **The candidates** are manila tags, three to a row, one to a row under
  34 rem, in the answer's order. A tag's number is a strip of label tape
  across it under the eyelet, where a card prints its band, with the years at
  its trailing end; a candidate has no group and takes no colour. Its side,
  option text (verbatim) and note are rows in the tag's ink under muted
  labels.
- **The rails** are two lines of pinstripe tape, Vehicle's and Model year's,
  down from the pressed part type (by anchor positioning) and along to two
  drops into each eyelet. No piece of tape ends but at an eyelet or a joint
  with its rail's next piece. One row's rails run only between the riser and
  the eyelets, from whichever is nearer the leading edge to whichever is
  further. Several rows' run along to the tags' leading edge, down it to the
  last row's bar, and along each row to its last eyelet. Where a browser has
  no anchor positioning, the rails start at the tags' top: at the leading
  edge for several rows, above the first eyelet for one. Where they cross,
  Model year's lies over Vehicle's.
- **Marks are ink on the tag**, in its ordinary ink: a solid stamp, a little
  askew, with a tick, "Checked" and the decoded elements that met the
  catalog's conditions; a dashed stamp with a warning triangle, "Not
  checked" and what could not be; "Catalog text only" and "No conditions",
  plain muted words, when nothing was either; and a solid stamp with an
  arrow, "By redirect" and the part type the catalog lists it under. A model
  year confirmed alone earns no stamp: every candidate's years held.
- **Weakest wins.** Each of a candidate's applications earns its own marks,
  and the tag shows their union, each kind once with its elements merged, so
  it never claims more than the weakest way the candidate fits: an
  application checked by Drive Type beside one with no condition reads "No
  conditions" and "Checked · Drive Type", never Checked alone. The weaker
  come first: Not checked, Catalog text only, No conditions, then Checked,
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
- **The open candidate's rows.** A strip is a link to the page with its
  candidate open (the URL's `fitment.number`, with `fitment.side` for a
  sided fitment and `fitment.type` for one whose type is not the pressed
  part type's); the open one's links to the page with none, so one is
  open at a time. The rows are the page's table panel (panel ground, a
  2 px `--steel` edge on 3 px of `--steel-dark` and a soft shadow), notched
  up at its tag: on a wide page the row under the tag's row, the tags after
  it filling theirs first; on a narrow one right under the tag. The rails
  run on down its leading edge to the next row of tags. Shelf and yard are
  two equal halves divided by a rule, one when one kind is loaded, each
  under its count in the label face and, muted, its sources' as-of dates,
  or why one did not answer. A row is its id in the monospace face, its
  location, and its source's fields verbatim, each name muted before its
  value. A shelf row tagged with a retired number says which in a muted
  line under it; the yard's vehicle that is the VIN looked up says "This
  VIN" in plain bold. Under 34 rem the halves are one row that scrolls
  sideways, the next showing at the edge, a row's fields under its id and
  location.
- **Past ten rows** a column says how many it shows, and "Show all" is a
  link to the page with every row (`rows=all`), swapped in place where
  scripts run.
- **Shelf rows with no number** belong to no candidate: after the first
  page's candidates, a strip of label tape, stuck on, names the part type
  and the vehicle in their own case, over the same panel, headed by their
  count and as-of dates. Past the candidates' page of them, "Show all" is
  the page with every one (`rows=unnumbered`). They never keep the
  candidates' "Next page".

## Good enough: empty places

Where the page has no answer, the place stays, as a blank strip: a 3 px dashed
`--faint` outline with one sentence in `--muted`. A control in it is a key of
metal; a way on is a link.

- **Under the names:** a vocabulary with no name, none loaded, or a model
  not known yet.
- **The parts that fit:** no catalog loaded, with a key that opens the
  Vocabularies dialog; the model not known yet; the catalog lacks the
  vehicle, naming both, with a link to the catalog's lines; the catalog has
  the vehicle and nothing for its year, with a link to its lines for the
  vehicle; nothing pressed; and a pressed part type with nothing for the
  vehicle.
- **A card** not typed yet is a 2 px dashed outline (manila tags, above).
- **A candidate's stock** with none for it is muted words over a dashed
  rule on its tag, and a column of its rows with none says "None." (the
  parts that fit, above).

## Good enough: where the brackets and lines run

- A label sits over its cells with a bracket opening downward. A second
  bracket under the plate opens upward. Both are borders, not images.
- **Wide**, a leader line runs from each lower bracket to its card, with
  square turns and one elbow. A second set of lines joins the maker's, the
  vehicle's and the year's cards to the label tape's columns.
- Those lines are SVG paths in a 1020-unit box (60 a cell, 170 a card),
  stretched to the stage with `vector-effect="non-scaling-stroke"`. Positions
  come from arithmetic on the grid, and nothing is measured at run time.
- **Narrow**, a name's make is under Maker, its model under Vehicle and its
  years under Model year; nothing of a name is under Check digit, Plant or
  Serial. A straight line runs down from each of those three cards to the
  part under it, and on from name to name, behind each vocabulary's label.
  The stepped lines are not drawn. Neither are the leader lines: the cards
  they would end on move.
- A vocabulary's label is the plate's first column on a wide page. On a narrow
  one it is a line of its own above the strip, a piece of label tape, and it
  stays at the region's leading edge while the row scrolls. The first label
  carries the heading's words, "Also known as".
- **"Check years"** is on a vocabulary's label exactly when one of its names'
  years is unsure. It is a strip of tape in the year's colour and an in-page
  link to the first of those years. Following it scrolls them into view, rings
  them, and fans their stack out if they are under its top strip. The scroll
  is smooth only where motion is welcome.
- **The names block shows three vocabularies**, two under 34 rem, in loaded
  order. A piece of tape after them reads "N more vocabularies" and shows the
  rest in place. It is a switch without a script, as a stack's is.
- A newly finished group draws its bracket and line in.
- No line runs to a blank strip.

## Good enough: dialogs

Everything behind a button is a `popover`, opened by `popovertarget` with no
script. There are three sizes.

| Kind | Used for | Width | Backdrop |
| --- | --- | --- | --- |
| Dialog (`dialog[popover]`) | Vocabularies, Developers, JSON, the check digit's sum | `min(56rem, 100vw - 2rem)` | `rgb(10 18 26 / .55)` |
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
- A new name from a vocabulary is a strip of label tape; one that can be
  pressed is the second tape.
- A new fact about the VIN itself is a row on a tag, under its group's band.
- A new connection between two things is a line of pinstripe tape.
- A fourth material needs a reason, as manila had: something the page hangs,
  sticks or bolts on that the three cannot be.
- A seventh colour needs a reason. The six group colours are the only hues
  besides `--link` and `--bad`.
- Check both colour schemes, a wide window (1280 px), a narrow one (768 px), a
  390 px one, and the page with JavaScript off.
