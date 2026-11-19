# The page's design

The page at `/` (`pkg/ui`) looks like a stamped VIN plate with tags hung from
it. This file states which parts of that look are the design, and which parts
are settled enough to leave alone. It describes; the code is the source:
[ui.css](pkg/ui/static/ui.css), [page.html](pkg/ui/page.html),
[plate.js](pkg/ui/static/plate.js) and [brushed.svg](pkg/ui/static/brushed.svg).
The references behind the choices are in
[docs/research/vin-page.md](docs/research/vin-page.md) and
[docs/research/textures.md](docs/research/textures.md).

Each part has one of two standings.

| Standing | Parts | What it means for a change |
| --- | --- | --- |
| **Signature** | The VIN input, the vinyl tape, the aluminum plates | These are the page's identity. New UI is built from them, and a change keeps every rule below. |
| **Good enough** | The decoded cards, the brackets and leader lines | These work and stay as they are. A change fixes a bug or follows a signature part; it does not restyle them. |

## Ground rules

- **No inline style or script.** The page's CSP refuses both. Every position is
  a class in the one stylesheet, on a fixed grid.
- **No raster images.** A texture is a gradient, a shadow, a mask, or
  `brushed.svg`'s `feTurbulence` noise.
- **The page works without JavaScript.** The server draws the whole answer for
  whatever VIN the URL carries. Scripts only make it live.
- **Popovers, not scripted overlays.** Menus, dialogs and notes are
  `popover` elements opened by `popovertarget`.
- **Two colour schemes, two fixed materials.** The ground, the text, the cards
  and the group colours change between light and dark. The metal and the tape
  do not: both are the same object under either light.
- **Motion is a stamp, a draw and a pop**, each under 0.4 s, and none of it
  under `prefers-reduced-motion`.

## Tokens

All tokens are custom properties on `body` in `ui.css`.

| Role | Light | Dark |
| --- | --- | --- |
| Ground `--bg` | `#dcecf7` | `#16222d` |
| Text `--fg` / `--muted` | `#14181d` / `#47515c` | `#eef2f5` / `#a9b6c2` |
| Card ground `--card-bg` | `#ffffff` | `#1f2d3a` |
| Rule `--line`, not-yet `--faint` | `#9fb3c2`, `#a9bccb` | `#45566a`, `#3a4a5c` |
| Link and focus `--link` | `#0b4fa8` | `#8ec5ff` |
| Error `--bad` | `#b91c1c` | `#ff8a8a` |

Each of the six groups of a VIN has one colour, `--c`. Its label, bracket,
leader line and card all use it.

| Group | Positions | Light | Dark |
| --- | --- | --- | --- |
| Maker `wmi` | 1–3 | `#c8102e` | `#ff6b81` |
| Vehicle `vds` | 4–8 | `#0b4fa8` | `#7db4ff` |
| Check `check` | 9 | `#5b2d8e` | `#c9a2ff` |
| Year `year` | 10 | `#a34c09` | `#ffb067` |
| Plant `plant` | 11 | `#0f6b46` | `#6fdcab` |
| Serial `serial` | 12–17 | `#3d4650` | `#b8c4cf` |

A group's mark on a material is `--cm`. On metal it is the light-scheme
colour in both schemes. On tape it is the dark-scheme colour in both schemes.

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
- **Focused**, the plate takes the 3 px `--link` outline, and the next empty
  cell blinks a 6 px bar of `--link` along its bottom.
- **The caret among characters** is a 4 px bar in the gap before its cell.
  Under a mouse, the gap a click would pick shows the same bar at 35%.
- **A selected cell** is inked: `#16202b`, white character, ringed 3 px in its
  group's colour.
- **A character no VIN holds** fills its cell with `--bad`. A bad VIN is not
  an error: the plate still decodes, and the problems are listed under the
  answer.
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
- **Raised:** `--raised`, a white line inside the top and 2 px of `--steel`
  below. The VIN plate and dialogs sit on 3 px of `--steel-dark` (`#58626c`)
  and a soft drop shadow.
- **Lettering:** `--on-metal` (`#232930`) with `--emboss`, a white edge below
  and a dark edge above, so the letter reads as struck in.
- **Pressed:** a button moves down 2 px and loses its lower edge.

The fastener says what the piece is.

| Fastener | On | Drawn as |
| --- | --- | --- |
| Rivet | Small tags: header buttons, menu rows; one at each end of the brand | A 2.5 px `--steel-dark` dot in a light ring |
| Slotted screw | The VIN plate and every dialog's head, one at each end | A 5 px light disc, a `--steel` ring, a 9 × 2 px slot |
| Eyelet | A decoded card | A hole punched by a radial mask, ringed in the card's heading |

Only a dialog's head is metal. The panel under it is card ground. Text to
copy sits in a recess: an inset shadow on a slightly darkened ground.

## Signature: vinyl tape

A name from a vocabulary is a strip of black label-maker tape. The tape is
the one material on the page that is not metal, as a parts bin's label is not.

- **Body:** `--tape`, faint ridges every 3 px over
  `linear-gradient(#30363d, #171a1e)`.
- **Edges:** a lit top (`--tape-lit`, `#69727c`), a heavy bottom
  (`--tape-edge`, `#050607`, 3 px), and `--tape-lift` under it.
- **Lettering:** `--on-tape` (`#f6f8fa`) in the monospace face with
  `--tape-emboss`, a dark edge up and left and a light edge down and right, so
  the letter reads as raised.
- **Case is kept.** A vocabulary's entry is case-exact, so the tape never
  upper-cases a name. The vocabulary's own stamp and the "Also known as"
  heading are upper case.
- **Four columns at fixed fractions** (130, 170, 465 and 255 of 1020): the
  vocabulary's stamp, then make, model and years. Each word sits over a 3 px
  underline in its group's mark, under a bracket in its group's colour, on the
  line that runs down from its card.
- **A dashed line, bracket or underline is a doubt.** The strip's note, behind
  its round "i" button, says the doubt in words.
- **Several names stack.** The first strip is on top, with the edges of up to
  three more under it. The stamp is the switch and carries the count. Opened,
  the stack is one strip with a row to a name.
- **An empty place stays.** A vocabulary with no name, or none loaded, keeps a
  blank strip: a 3 px dashed `--faint` outline with a sentence in it.

## Good enough: decoded cards

Each group has a card under its bracket, six in a row on the same width as the
plate.

- A card is a tag: card ground, a 3 px border in its group's colour, a hard
  `3px 3px 0` shadow, and an eyelet at the top of its heading band.
- The heading band is the group's colour with the title in the label face and
  the positions beside it ("Vehicle 4–8").
- The body is either label and value rows, or one large value in the monospace
  face. The check digit's card opens its arithmetic in a dialog. The year's
  card lists the codes around its own.
- A card for a group not yet typed keeps its place: dashed `--faint` border,
  no shadow, a muted heading.
- A card whose group is wrong turns its border and band to `--bad`.
- A newly finished group pops its card in (0.3 s).
- Under 46 rem the cards stack in the brackets' order.

## Good enough: brackets and leader lines

- A label sits over its cells with a 3 px bracket opening downward. A second
  bracket under the plate opens upward. Both are borders, not images.
- A leader line runs from each lower bracket to its card: 3 px, square
  corners, one elbow. A second set of lines joins the maker's, the vehicle's
  and the year's cards to the tape's columns.
- The lines are SVG paths in a 1020-unit box (60 a cell, 170 a card),
  stretched to the stage with `vector-effect="non-scaling-stroke"`. Positions
  come from arithmetic on the grid, and nothing is measured at run time.
- Brackets and lines are `--faint` until their group is typed, then take its
  colour. A newly finished group draws its bracket and line in.
- No line runs to a blank strip.
- Under 46 rem the lines go, and the brackets and colours stay.

## Building something new

- A new control is a piece of metal. Reuse `--metal`, `--raised` and
  `--emboss`, and pick its fastener from the table above.
- A new name from a vocabulary is a strip of tape.
- A new fact about the VIN itself is a row in a card, in its group's colour.
- A seventh colour needs a reason. The six group colours are the only hues
  besides `--link` and `--bad`.
- Check both colour schemes, a 390 px window, and the page with JavaScript
  off.
