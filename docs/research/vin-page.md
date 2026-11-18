# The VIN page: references for the bracket diagram, the input, mapping branches and a fitment drill-down

Design research behind the VIN page prototype, draft PR
[#303](https://github.com/joshuasprow/nitrovin/pull/303). Date: 2026-11-17.
Terms from GLOSSARY.md: vocabulary, entry, spine, mapping, translation.

The question: what should nitrovin's page be if it is a single page, the VIN
input is its focus, and the "What's in a VIN?" bracket diagram is the interface
itself, with the vehicle's names in other vocabularies branching from the
brackets? How can it be playful and still rest on real, working examples? And
can parts fitment later be reached on the same patterns, without leaving the
page? Four surveys of live pages answered it: annotated inputs and anatomy
diagrams, displays of one-to-many mappings, playful single-input tools and
typeahead, and drill-downs into parts fitment. This document merges the four.
The decisions the research led to are in the spec in the PR's description; §9
summarises them against what the prototype does today.

**Answer in brief.**

- **The diagram exists on the web, and it breaks on phones.** explainshell is
  the closest live match to "bracket, leader line, labelled box". It and the
  one live VIN infographic (FreeVINGenerator) are the two that keep their lines
  at every width, and the two that fail at 390 pixels. Every other interactive
  example gives each group one colour, repeats it on the explanation, and drops
  the lines first (§7, finding 1).
- **Draw on a fixed grid, over one real input.** A VIN is 17 fixed-width
  characters, so brackets and lines can be placed by arithmetic and never
  measured. The input stays one native field with drawn cells over it; tools
  that replaced the field lost paste (findings 3, 4 and 8).
- **Mapping quality is written, not drawn.** No tool with more than about eight
  targets draws a line per target, and none carries match quality in a line's
  style. Quality is a word, tag or glyph at the value; an empty target keeps
  its place and says so; one name leads and the rest sit behind a count
  (findings 2, 5 and 6).
- **Fitment drills down in place.** A count is the handle that opens the next
  level under its own row; alternates differ by one plain line of option text;
  caveats are sentences where the doubt applies; the open state is in the URL.
  The people it is for value density and speed over looks, by the project
  owner's account of RockAuto (findings 5 and 10 to 13).

## How to read the entries

Every link below was opened on 2026-11-17, in a real browser at about 1280
pixels wide or, where an entry says so, fetched as text. "At 390" means the
viewport was set to 390 pixels wide and the page looked at again. "Seen" means
observed in the rendered page or in its DOM and CSS; CSS values quoted are the
page's own. Anything deduced and not observed is marked [INFERENCE]; anything
not tried says "not tested". §8 lists what could not be opened. Where two
surveys disagreed about a page, it was opened again the same day, at 1280×900
and at 390×844, and the entry carries what that second look showed.

Each entry gives the link, the specific element worth looking at, how it
behaves, how it is drawn where that was read, any measurements, what it suggests
for a VIN, and what does not transfer. The sections group references by what
they are good for. Where a section keeps an order, it is the order of fit the
survey gave, best first. Each VIN example's entry states that example's own
grouping of the 17 characters. They run from four groups to eight; the page
settled on six (§9).

## 1. Annotated inputs and bracket diagrams

Pages that take one string and label its parts: brackets, underlines, colours,
leader lines. In order of fit.

### explainshell: brackets under tokens, leader lines to boxes

- **Links:** <https://explainshell.com/explain?cmd=tar+-xzvf+archive.tar.gz>,
  and a six-token command at
  <https://explainshell.com/explain?cmd=tar+-xzvf+archive.tar.gz+-C+%2Ftmp>.
- **Element:** The command is re-rendered in Courier New 18px under a small
  text field in the header. Each token gets a 5px-deep square bracket beneath
  it and a 1px leader line that drops from the bracket, runs out to a side
  gutter and ends in a dot beside a bordered help box. The boxes are one
  stacked column. Bracket, line, dot and box border share one colour per token,
  taken from a list of 20 in `es.js` (blues, oranges, greens, purples and
  greys, among them `#31a354`, `#bcbddc` and `#dadaeb`) that is shuffled on
  every load. State is in `?cmd=`. The wordmark is script lettering with a
  drawn seashell.
- **Behaviour:** Hovering a token or its box hides every other token's lines
  (`visibility: hidden`), dims the other tokens and boxes to 0.4 opacity and
  bolds the hovered token. Seen from a token and from a box. An unknown token
  gets no line.
- **How it is drawn:** One `<svg id="canvas">` overlay built with d3 3.1.6,
  absolutely positioned, as wide as the viewport, with `overflow: hidden`. Two
  paths per token (10 for the five tokens of the first link, 12 for the six of
  the second):
  - bracket: `M500,0V5H500V5H563V0H563`
  - leader: `M532,5V19H75V125H155`, ending in `<circle r="4">`, a dot 8px
    across
  - Leaders are staggered: each drops 20px lower (y = 19, 39, 59, 79) and sits
    20px further in (x = 75, 95, 115, 135), so they nest and never cross.
    Left-half tokens route down a left gutter, right-half tokens down a right
    one. Each token is a span whose `helpref` attribute names its box.
  - Positions come from `getBoundingClientRect`. `es.js` has no resize handler
    of its own, but the page's inline script binds a throttled `draw()` to the
    window's scroll and resize, so the paths are recomputed: narrowing the
    window from 1280 to 700 without a reload turned the first leader into
    `M362,5V19H-95V125H-15`.
- **Scale:** Six lines are readable only because they nest (outermost token to
  lowest box) and hover isolates one. Box height is unbounded, so the boxes for
  the last tokens, and the ends of their lines, were below the fold at 900
  pixels high. One-to-many is not shown at all, and the lines carry no quality.
- **At 390:** No reflow, and redrawing does not save it. The layout is a fixed
  940px container, so the document is 940px wide in a 390px viewport and
  scrolls sideways. The help boxes fill that width (x = 1 to 939) and the
  command's tokens sit at x = 330 to 631: the first token is cut by the right
  edge of the screen, and the rest of the command, most of each box and the
  wordmark run off the right until scrolled to. The lines are lost on both
  sides. With the boxes at the left edge, the left-going leaders are computed
  to negative x (−95, −75, −55, −35) and their end dots sit at x = −16 to −8,
  off the left edge where no scroll reaches. The overlay is only 390px wide, so
  the brackets and leaders beyond x = 390 fall outside it.
- **Suggests:** A bracket under each VIN group, a staggered right-angled leader
  and a box per group, built from DOM spans and one SVG with no canvas library.
  Mapping rows can sit inside or under the box the leader lands on, so there
  are no more lines than groups.
- **Does not transfer:** The drawing depends on measuring the DOM and redrawing
  on every scroll and resize, and a fixed-width layout still defeats it on a
  phone. The token spans have no `tabindex` or ARIA, so it is hover-only. It
  pulls in jQuery and d3 for a handful of `M/V/H` paths. The input is a small
  field, not the hero.

### Crew Chief "Explore a VIN": VIN tiles, CSS brackets, one detail panel

- **Link:** <https://crewchief.cc/blog/decoding-your-vin> (the widget "Explore
  a VIN", then "Try Your Own VIN" below it).
- **Element:** Three stacked rows. 17 monospace tiles (44×56px) coloured by
  section: WMI blue, VDS amber, check digit red, VIS green. A row of position
  numbers 1–17 at the same widths. A bracket row: four `div.bracket` elements
  with `border-top: 2px solid` and a label, coloured to match. Its grouping is
  four: WMI over positions 1–3, VDS over 4–8, CHECK over 9 and VIS over 10–17.
- **Behaviour:** Hover, focus or tap lifts the tile (`translateY(-4px)`,
  shadow, border) and dims the other tiles to 0.5. One panel under the row
  fades in: "Position 11 of 17", a VIS badge, "Assembly Plant", a paragraph,
  and a left border in the section colour. The input below is monospace with
  letter-spacing, `maxlength=17` and an "11 / 17" counter inside the field. Its
  status line read "11 of 17 characters entered." and then "Valid VIN — check
  digit verified." Typing a full VIN replaced the tiles above with that VIN.
- **How it is drawn:** CSS only. Bracket widths are hard-coded from the tile
  size: 140px, 236px, 44px and 380px (n × 44 plus 4px gaps). The rows are flex
  rows, and 17 tiles need 812px: in a 718px row at 1280 the tiles shrank to
  38px and the brackets to 123, 207, 43 and 333px, within 3px of the tiles
  above them.
- **Narrow:** At 768px and below, tiles are 36×48 and the bracket widths are
  recalculated. At 540px and below, tiles are 30×40 and
  `.vin-brackets { display: none }`. At 380px and below, `overflow-x: auto`.
  At 390 only tiles and numbers showed; colour and the legend carry the
  grouping.
- **Accessibility:** Each tile is `role="button" tabindex="0"` with
  `aria-label="Position 10: M — Model Year"`. The panel is
  `role="tooltip" aria-live="polite"`. Decorative rows are `aria-hidden`.
- **Suggests:** The most directly reusable layout: tiles, a position row and
  border-top brackets, with the decoded value and mapping rows in one panel.
- **Does not transfer:** The panel text is generic per position, not the
  decoded value. The brackets show only the three coarse sections and the check
  digit, with no leader lines. Hard-coded widths need redoing at every
  breakpoint.

### FreeVINGenerator "VIN Breakdown Visualizer": the print infographic as live SVG

- **Link:** <https://freevingenerator.com/vin-breakdown-visualizer> (enter
  `5YJSA1DG9DFP14705`, press Visualize).
- **Element:** The same layout as the owner's reference image. Dark character
  tiles in a row. Its grouping is eight, each group with a 4px coloured bar on
  one edge, a 4px vertical line and a rounded coloured box of its own: country
  (position 1), manufacturer (2), type or division (3), the descriptor (4–8),
  check digit (9), model year (10), plant (11) and serial (12–17). Boxes
  alternate above and below the row: manufacturer, descriptor, year and serial
  above; country, type, check digit and plant below.
- **Behaviour:** Rendered only after pressing the button, not while typing.
  Hover adds a `.highlight` class (`brightness(1.2)` and a white drop-shadow)
  to the box and its tiles; the effect is faint.
- **How it is drawn:** `<svg id="vinSvg" width="1402" height="800">` with eight
  `<g class="group-…">` (country, wmi, type, vds, check, year, plant, serial).
  Each holds one `line.connecting-line`, one `g.info-box` (a `rect rx=10` and
  text), one `line.underline` and one `g.vin-char-group` per character
  (Courier New, 36px).
- **Measured:** Fixed pixel coordinates in a 1402×800 SVG inside an
  `overflow: auto` wrapper. The drawing is the same eight groups and eight
  boxes at every width; only the wrapper scrolls. At 1280 the wrapper is 1240px
  wide: 15 of the 17 characters are fully in view and the serial box (x = 1092
  to 1292) is cut at its right edge. At 390 the wrapper is 350px wide: three of
  the 17 characters and three of the eight boxes are in view, and the page
  itself does not overflow.
- **Suggests:** The alternating above-and-below layout fits all eight groups
  without crossing lines. The lower boxes are where mapping branches could
  hang.
- **Does not transfer:** The fixed coordinates. Boxes show the raw characters
  ("SA1DG"), not decoded values, and the serial box prints seven ("FP14705")
  for a group of six. All labels are SVG text at 11px.

### GetVIN "Interactive Explainer": sticky VIN, one animated arch, scroll-spy

- **Link:** <https://get.vin/decoder> (scroll to "Understand Your Vehicle's
  Identity").
- **Element:** The example VIN `1GN ERGKW 5 J J 148668` sits in a sticky bar as
  six rounded segment chips in bold monospace: a grouping of 3, 5, 1, 1, 1 and
  6 characters. One indigo arch (rounded top, 2px) rises from the active chip
  and ends in a dot above the explanation column.
- **Behaviour:** Hovering a chip, or scrolling its section into view, fills
  the chip indigo and slides the arch to it. In the model-year section the
  lookup table highlights the cell matching the VIN's year character
  (J / 2018).
- **How it is drawn:** One absolutely positioned div, `#vin_part_line`, with
  `border-2 border-b-0 rounded-2xl rounded-b-none`. Script sets inline `left`
  and `width` (`left:36px; width:528px` for the WMI,
  `left:232.5px; width:331.5px` for the year). `transition: all` animates the
  move. It has no pseudo-elements.
- **Narrow:** The arch is `hidden lg:block`, so it disappears. The VIN stays
  sticky and wraps by whole segment because each chip is `inline-block`.
- **Suggests:** A single moving connector is cheap and lively. Highlighting
  the matching row of a reference table fits mapping tables too.
- **Does not transfer:** The explainer uses a fixed example VIN; whether
  decoding another VIN swaps it in was not tested. Only one group is connected
  at a time, so it is not an at-a-glance diagram.

### crontab.guru: one big input, the caret picks the part explained

- **Links:** <https://crontab.guru/#5_4_*_*_sun>; the error state at
  <https://crontab.guru/#5_4_*_*_banana>.
- **Element:** One centred text input is the page. JetBrains Mono throughout;
  the input is 30px, centred, `border: 4px solid rgb(255,255,128)`, radius 6px,
  glow `box-shadow: rgba(255,255,128,.15) 0 0 20px 5px`, on `rgb(23,23,23)`.
  One accent colour, pale yellow, for the title, the border and the buttons.
  Above the input is a large italic quoted sentence ("At 04:05 on Sunday.") and
  a small "next at 2026-11-22 04:05:00" line. Below are five field labels
  (minute, hour, day, month, weekday) and a small table of allowed values.
- **Behaviour:**
  - The sentence and the "next at" line update per keystroke.
  - Moving the caret into a field turns that field's label yellow (`.active`),
    turns the matching words of the sentence yellow ("05"), and swaps the table
    to that field's allowed values ("0-59 allowed values"; with the caret in
    "sun": 0-6, SUN-SAT and 7).
  - Error: border and glow turn salmon (`rgb(255,149,128)`; the glow doubles to
    `0 0 40px 5px` at .3 alpha, class `input.invalid`), the sentence
    disappears, and the offending field's label becomes a filled salmon chip.
    There is no error text at all.
  - A "random" tab is stuck to the input's top-right edge; a "Copy" button sits
    inside the input.
  - The URL hash rewrites on every keystroke, so every state is a link.
- **How it is drawn:** No connectors. The sentence is spans; the table is
  several `<tbody>` elements toggled with `display: none`.
- **Suggests:** Keep the VIN field as the only large thing on the page. Let
  the caret position, not only hover, select the active group. Echo the decode
  as one sentence above the field. A "random" control loads a sample.
- **Does not transfer:** The labels sit in a row under a centred proportional
  string, not locked to character columns, because cron fields vary in width.
  A VIN is fixed-width, so alignment is possible, and 17 characters need fixed
  cells. Narrow screens were not checked.

### float.exposed: hero input with a coloured marker under every character

- **Link:** <https://float.exposed/0x40490fdb>
- **Element:** The value field is a borderless, transparent, centred 50px
  monospace input spanning the page. Each bit below it has a 3px coloured bar
  beneath it: red sign, green exponent, blue significand. The fields and the
  formula further down reuse the same three colours.
- **Behaviour:** Clicking a bit toggles it and everything updates. The URL
  carries the value.
- **How it is drawn:** Each bit is `span.bit_container[title="Bit 31"]` holding
  `div.bit_label.monospace` and `div.bit_marker.float_sign`, a 3px-tall
  coloured div. No SVG.
- **At 390:** The hero font shrinks, the bit row wraps at byte boundaries, and
  there is no horizontal scroll. The underline colours survive the wrap.
- **Suggests:** A per-character coloured underline is the lightest bracket
  there is and it survives wrapping. Reusing the group colour on the decoded
  value and the mapping row removes the need for a line.
- **Does not transfer:** No labels are attached to the groups; the colours are
  learned from the fields below.

### Node.js URL docs: one string, two vocabularies, above and below

- **Link:** <https://nodejs.org/api/url.html#url-strings-and-url-objects>
- **Element:** The URL sits on the middle line of a monospace block. Nested
  brackets above it label the string in the legacy `url.parse()` vocabulary
  (protocol, auth, host, path, hash). Nested brackets below label the same
  characters in the WHATWG vocabulary (origin, username, password, host,
  pathname, search, hash). Where the two disagree the brackets visibly differ:
  `auth` above splits into `username` and `password` below.
- **How it is drawn:** Box-drawing characters (`┌ ─ ┬ ┤ │`) in a `<pre>` with
  `white-space: pre` and `overflow-x: auto`. Alignment comes free from the
  monospace grid.
- **Suggests:** vPIC's reading of the VIN above the characters and another
  vocabulary's below, both bracketing the same positions. Nesting handles a
  group that contains finer fields.
- **Does not transfer:** Static. It needs about 98 columns, so it scrolls
  sideways on a phone. Screen readers read the box characters. It labels spans
  but shows no values.

### rvcodec.js: two vocabularies linked by colour and hover, including split fields

- **Link:** <https://luplab.gitlab.io/rvcodecjs/#q=addi+x1,+x2,+42&abi=false&isa=AUTO>
- **Element:** One input in a terminal-style frame. Below it are two rows:
  "Assembly = addi x1, x2, 42" and "Binary = 0000 0010 1010 0001 …". Each
  assembly token and the bits that encode it share a colour. The opcode's red
  appears in two separate runs of bits.
- **Behaviour:** Hovering `x1` gave it a yellow background and put a grey
  background on its five bits, which straddle two nibble groups. State is in
  the URL hash.
- **How it is drawn:** Plain text spans sharing a class (`fragId-6`); no
  lines. No CSS hover rule was seen for it, so the highlight is set by script.
- **Suggests:** A one-to-many or non-adjacent mapping (one vPIC model name to
  several catalog names) reads fine as shared colour and hover, where a leader
  line would have to fork.
- **Does not transfer:** Colour alone carries the link, with no label or
  shape; it needs a second cue for colour-blind readers.

### cidr.xyz: the input is the diagram

- **Links:** <https://cidr.xyz/>; a /28 at <https://cidr.xyz/#10.88.135.144/28>.
- **Element:** Five 80×80px inputs (30px text, centred, radius 6px) with "."
  and "/" between them, each a flat Tailwind colour: purple-400, red-400,
  green-400, yellow-400, slate-300. They carry `inputmode="numeric"` and
  `aria-label="Octet 1"` and so on. Under each octet is a strip of eight
  bordered bit cells in the same colour. Bits outside the prefix turn grey: for
  /28 the last four cells of the fourth octet were grey. Results are monospace
  values over grey bold labels. "Copy CIDR" and "Copy Share Link" buttons.
- **Behaviour:** Pasting `192.168.1.0/24` into the first box filled all five
  boxes.
- **At 390:** The boxes wrap to two rows and the bit strips stack. No
  horizontal scroll; the colours keep each strip tied to its box with no lines
  at all.
- **Suggests:** One flat colour per VIN group, repeated on the group's label
  box and its mapping branches. Big blocky colour is what makes this read as
  playful. The greyed "not applicable" state suits VIN positions a
  manufacturer leaves undefined.
- **Does not transfer:** Five separate inputs are five tab stops and five
  fields for a screen reader, and splitting a VIN that way would break typing
  flow. It works here because the separators are real characters.

### CheckerVIN `vin-anatomy.svg`: a minimal two-tier bracket diagram

- **Links:** <https://checkervin.com/vin-anatomy.svg>, embedded in
  <https://checkervin.com/how-to-read-a-vin>.
- **Element:** 17 rounded tiles holding position numbers. Above them are three
  solid bars (WMI · 1–3, VDS · 4–8, VIS · 9–17). Below are seven thin grey
  brackets, each with a label: Country (1), Manufacturer & division (2–3),
  Model, body, engine, restraints (4–8), Check digit (9), Model year (10),
  Assembly plant (11) and Sequential production number (12–17). The check
  digit is the only orange item.
- **How it is drawn:** A 6.9 KB hand-written SVG, `viewBox="0 0 1200 452"`.
  Each bracket is one path such as `<path d="M60 232 v8 h58 v-8"/>`. It has
  `role="img"` with `<title>` and `<desc>`.
- **Narrow:** The page embeds it as an `<img>` with `min-width: 800px` inside
  an `overflow-x-auto` box, so it scrolls sideways.
- **Suggests:** Two tiers answer "coarse sections against fine groups" without
  clutter: bars above, brackets below. The three-segment path is the whole
  bracket recipe.
- **Does not transfer:** A static image that shows position numbers, not a
  VIN.

### RegExr and regex101: two-way hover between a string and its explanation

- **Links:** <https://regexr.com/> (Tools → Explain) and
  <https://regex101.com/?regex=%5E%5BA-HJ-NPR-Z0-9%5D%7B8%7D%5B0-9X%5D%5BA-HJ-NPR-Z0-9%5D%7B2%7D%5Cd%7B6%7D%24&testString=5YJSA1DG9DFP14705&flags=gm>
- **Element (RegExr):** The expression's tokens are coloured. The Explain
  panel lists one card per token; cards nest and take the token's colour
  (group, then set, then range).
- **Behaviour (RegExr):** Hovering a card outlined the matching characters in
  the expression (class `exp-selected`, with left and right caps). Hovering a
  token opened a dark tooltip under it ("Word. Matches any word character…")
  and outlined its card.
- **Element (regex101):** A right-hand "Explanation" tree repeats each token
  as a coloured chip beside its description.
- **Suggests:** Nested cards suit "group, decoded value, mappings" as
  indentation in place of lines. Two-way highlighting should be the baseline.
- **Does not transfer:** Both are dense three-pane tools, the opposite of a
  single-focus page. regex101 showed "Engine Error" in the headless browser;
  only its explanation tree rendered.

### Rough Notation: hand-drawn brackets that draw themselves

- **Link:** <https://roughnotation.com/> (section "Brackets", press Annotate).
- **Element:** Wobbly red brackets draw themselves around a heading (top) and
  a paragraph (left and right). The page also shows underline, box, circle and
  highlighter styles.
- **How it is drawn:** An absolutely positioned `svg.rough-annotation` overlay
  with `overflow: visible; pointer-events: none`. Paths are cubic curves,
  animated by `stroke-dasharray` and `stroke-dashoffset` with a
  `rough-notation-dash` keyframe and staggered delays. The page states 3.8 kb
  gzipped.
- **Suggests:** The stroke-dash draw-on is what makes a bracket feel playful,
  and it works on any SVG path, a straight bracket included. Brackets could
  draw in left to right as each VIN group becomes decodable.
- **Does not transfer:** It is a library, so a new dependency, and the effect
  can be done without it. Its brackets wrap an element's box and have no leader
  lines. A reduced-motion switch was not checked.

### The static infographic family: the look the owner asked for

All four are images: not interactive, not selectable, fixed in size. They scale
down on a narrow screen and their labels become tiny.

- **UAW Union-Built Vehicle Guide, "What's in a VIN?"**:
  <https://uaw.org/wp-content/uploads/2026/04/2026-UAW-union-built-vehicle-guide-1BFINAL-PB.pdf>.
  The source of the owner's example image (§8). In that image, red-outlined
  white boxes hold small lookup tables, and thick white-and-red brackets
  alternate above and below an embossed VIN plate.
- **The Conversation, "Decoding the VIN"**:
  <https://theconversation.com/whats-in-a-vin-how-to-decode-the-vehicle-identification-number-your-cars-unique-fingerprint-228328>.
  A PNG, CC BY-SA. The VIN sits inside a plate-shaped pill outline. Red arrows
  alternate up and down. Multi-character groups get a thin red underline.
  Labels alternate red and black.
- **DRiV Parts, "Decoding your VIN"**:
  <https://www.drivparts.com/parts-matter/learning-center/by-the-numbers/about-vehicle-identification-numbers.html>.
  A JPG. The VIN is spaced into six groups with a numbered lollipop pin above
  each. A numbered list 1–6 below carries the text: the first three characters
  (the first the country, the next two the manufacturer), the vehicle's
  features, the ninth character as a check, the model year, the plant and a
  sequential production number.
- **MDN, "Anatomy of a URL"**:
  <https://developer.mozilla.org/en-US/docs/Learn_web_development/Howto/Web_mechanics/What_is_a_URL#basics_anatomy_of_a_url>.
  A PNG. Each segment has a highlighter fill and a small elbow arrow down to an
  italic label in the same colour. One dimension-line bracket above spans two
  segments ("Authority").
- **Suggests:** The UAW boxes hold lookup tables, not just labels; that is the
  natural slot for mapping rows. The plate or pill outline around the VIN is a
  cheap way to make the input look like the physical object. DRiV's numbered
  pins are a narrow-screen fallback for leader lines: the numbers survive when
  the boxes have to stack.

### Polypane CSS Specificity Calculator: a mirror row of chips, sorted into buckets

- **Link:** <https://polypane.app/css-specificity-calculator/?selector=header%20h1%23sitetitle%20%3E%20.logo%3Ahover>
- **Element:** A plain 24px monospace field. As you type, a row of coloured
  chips directly under the score re-renders each token (`header`, `h1`,
  `#sitetitle`, `>`, `.logo`, `:hover`). Three coloured cards below (ids,
  classes, elements) each repeat the chips that belong to them and show a
  count.
- **Suggests:** Leave the input undecorated and put a decorated mirror of it
  right underneath. Repeating a chip inside each destination card is a
  line-free way to show "this group feeds these mappings".
- **Does not transfer:** The mirror row is centred, not aligned under the
  typed characters. No hover linking was seen.

### Jisho: the query re-rendered as tokens, a reading above and an underline per token

- **Link:** <https://jisho.org/search/%E6%98%A8%E6%97%A5%E3%81%99%E3%81%8D%E7%84%BC%E3%81%8D%E3%82%92%E9%A3%9F%E3%81%B9%E3%81%BE%E3%81%97%E3%81%9F>
- **Element:** Directly under the search box the sentence is repeated large,
  split into words. Each word has its own underline, small reading text above
  it, and a colour by part of speech (two nouns orange, a particle red, a verb
  grey). Each word is a link and carries its part of speech as a tooltip
  (`data-pos="Noun"`, shown below on hover).
- **Behaviour:** The first word is marked current and its dictionary entries
  fill the page below. The strip is one line whatever the length; detail is one
  token at a time.
- **Suggests:** The decoded value above each VIN group, an underline per
  group, and a click on a group to load its mappings below.
- **Does not transfer:** One-at-a-time detail hides the other segments'
  mappings. It shows no edge quality and no empty state.

### vindecode.app "VIN Anatomy" widget: a caution about wrapping

- **Link:** <https://www.vindecode.app/blog/vin-number-anatomy>
- **Element:** Coloured tile buttons, a "Checksum Valid" pill in the card
  header, and six legend cards (range, characters, name, sentence) in a
  two-column grid: 1–3 WMI, 4–8 VDS, 9 Check Digit, 10 Model Year, 11 Assembly
  Plant and 12–17 Sequence Number. Hovering a tile ringed its card.
- **Measured:** The tile row is `flex-wrap`. In its 624px column the 17th tile
  dropped to a second line and the separate position-number row no longer
  lined up, at desktop width.

### How the connectors are drawn

| Technique | Seen at | Alignment source | Narrow-screen result |
| --- | --- | --- | --- |
| SVG overlay, right-angled paths computed from DOM rects | explainshell | `getBoundingClientRect`, redrawn on scroll and resize | breaks: fixed 940px layout, lines fall outside a 390px overlay |
| One fixed-coordinate SVG holding characters, lines and boxes | FreeVINGenerator, CheckerVIN | hard-coded x per tile | scrolls sideways (1402px and 800px minimum) |
| CSS `border-top` divs in a flex row | Crew Chief | widths = n × tile + gaps, per breakpoint | brackets hidden at 540px and below |
| One absolutely positioned bordered div, moved by script | GetVIN | inline `left` / `width` | hidden below the `lg` breakpoint |
| 3px coloured div under each character | float.exposed | each character is its own inline box | wraps cleanly |
| Box-drawing glyphs in `<pre>` | Node.js docs | monospace grid | scrolls sideways |
| Transparent `<textarea>` over a coloured `<pre>` | jwt.io (§2) | identical font metrics in both layers | wraps (`pre-wrap`) |
| SVG overlay with stroke-dash draw-on animation | Rough Notation | element box at annotate time | not tested |
| Raster image | UAW, The Conversation, DRiV, MDN | baked in | scales down; labels become tiny |

No pseudo-element (`::before` / `::after`) connectors were seen in any of
these.

## 2. One real input under a drawn one, and typeahead

How to decorate what is typed without giving up the native field, and how a
prefix can be answered early.

### input-otp and its shadcn/ui docs: one real input under drawn character cells

- **Links:** <https://input-otp.rodz.dev/> and
  <https://ui.shadcn.com/docs/components/base/input-otp> (sections
  `#separator`, `#pattern`, `#alphanumeric`, `#invalid`).
- **Element:** The cells are plain divs. A single
  `<input autocomplete="one-time-code" maxlength="6" pattern="…" inputmode="…" spellcheck="false">`
  is absolutely positioned over them with
  `color: transparent; caret-color: transparent; letter-spacing: -0.5em; font-family: monospace`.
  The divs render each character, a fake blinking caret and the focus ring,
  which slides between cells. Groups are separated by a "·".
- **Behaviour:** One tab stop, one field for a screen reader, native selection
  and OS paste, because it is one input.
- **Measured (paste, on the shadcn page):**
  - Six-cell digits-only field (`^\d+$`): pasting `987654321` kept `987654`,
    truncated to `maxlength`.
  - Same field: pasting `48a1` was rejected whole; the field stayed empty.
  - Field with no pattern: pasting `48 21-77` kept `48 21-`, spaces and dash
    included.
  - The project README lists a `pasteTransformer` prop (seen in the README
    text, not exercised).
- **Suggests:** 17 drawn cells in groups over one native input. The technique
  is CSS; the React library is not required.
- **Does not transfer:** A VIN pasted with a space, a dash or a lowercase
  letter would be rejected whole by a strict pattern, so the paste has to be
  cleaned first. A scripted paste test on the rodz.dev hero was inconclusive
  because the hero types its own demo.

### jwt.io: colouring segments inside a real, editable field

- **Link:** <https://www.jwt.io/>
- **Element:** The token field shows header, payload and signature in orange,
  purple and green while staying a normal editable text area. Under it are
  status lines (a tick and "Valid JWT", "Signature Verified"). Decoded panels
  sit to the right, with a "Claims Breakdown" tab: a table of claim, value and
  one-line meaning. Claim keys in the decoded JSON carry a dotted underline
  (hover not exercised). A "Generate example" control fills the field.
- **How it is drawn:** `react-simple-code-editor`. A `<textarea>` is
  absolutely positioned at 0,0 with `-webkit-text-fill-color: transparent` and
  a transparent background. Under it is a `<pre>` with `pointer-events: none`
  holding `span.ace_header`, `span.ace_dot`, `span.ace_payload` and
  `span.ace_signature`. Both use Roboto Mono 16px / 22.4px and `pre-wrap`, so
  the glyphs line up.
- **Suggests:** Keep one native VIN input (caret, paste, autofill, screen
  reader) and still colour or underline its groups, by drawing the decoration
  in a layer behind transparent text. A one-line verdict under the field, and
  a control that fills in an example.
- **Does not transfer:** No connectors. The layers must match font metrics
  exactly or they drift. The decoded panels do not repeat the segment colours.
  The two-column debugger layout pushes the field to the side and below the
  fold at 1280×800.

### react-otp-input: the counter-example, one field per character

- **Link:** <https://devfolioco.github.io/react-otp-input/>
- **Element:** One `<input>` per character, each labelled "Please enter OTP
  character N".
- **Measured:** Set to 17 cells and given `1HGCM82633A004352` by paste, the
  paste spread across all cells, but the row wrapped to 11 + 6 in a card
  roughly 1070px wide. Seventeen 52px cells with gaps do not fit one row.
- **Does not transfer:** 17 labelled fields is poor for a screen reader.

### Wordle: per-character state tiles with spoken labels

- **Link:** <https://www.nytimes.com/games/wordle/index.html>
- **Element:** 52px square tiles, 32px bold uppercase (nyt-franklin), a thin
  grey border when empty and a brighter one once a letter is in. Each tile
  carries `data-state` (`empty`, `tbd`, `absent`, `present`, `correct`) and
  `data-animation`. Three fills only: dark grey, ochre, green. The on-screen
  keyboard repeats the same three fills.
- **Accessibility:** Every tile is `role="img"` with a label such as "2nd
  letter, R, present in another position".
- **Suggests:** Three or four character states (plain, illegal character,
  check digit, recognised) and the label formula ("9th character, X, check
  digit").
- **Does not transfer:** There is no input element; the page listens for key
  presses. So no paste and no native mobile keyboard. Keep the tile look, not
  the input model.

### Toptal Keycode: typing anywhere goes to the tool

- **Link:** <https://www.toptal.com/developers/keycode>
- **Element:** No field. Any key press replaces the page's giant numeral
  ("74", roughly a quarter of the viewport height; not measured) and fills the
  cards below it.
- **Suggests:** The oversized single answer.
- **Does not transfer:** With no input there is nothing to paste into.

### Cleave.js demo: a brand recognised from the first one or two characters

- **Link:** <https://nosir.github.io/cleave.js/>
- **Element:** A `type="tel"` input over a legend of six card brands. Each
  legend row shows the logo, the prefix rule in words ("American Express:
  starts with 34/37") and a mask (`34•• •••••• •••••`).
- **Behaviour:** Typing `3782822` turned the Amex logo dark while the other
  five stayed grey, and regrouped the value to that brand's spacing
  (`3782 822`). Recognition is purely prefix-driven.
- **Suggests:** The direct analogue of recognising a manufacturer from the
  first characters of a VIN: a badge that lights after character 2 or 3, and a
  legend of sample prefixes with masks (`1HG•• • • • ••••••`) that doubles as
  the empty state.
- **Does not transfer:** The library writes the spaces into the input's value;
  a VIN field must copy out clean.

### typeahead.js examples: grey ghost completion inside the field

- **Link:** <https://twitter.github.io/typeahead.js/examples/#the-basics>
- **Element:** Typing `Mi` shows `Mi` in black followed by `chigan` in grey
  inside the field, and a menu below with the match bolded in each row.
- **How it is built:** A second, read-only input
  (`class="tt-hint" readonly tabindex="-1"`) sits under the real one and holds
  the completed string; a hidden `<pre aria-hidden="true">` mirrors the typed
  text.
- **Behaviour:** The ghost is prefix-driven; the menu is not (it listed
  "Wyoming" for `Mi`). The markup inspected had no `role="combobox"` or
  listbox attributes; Bundlephobia, below, has them.
- **Suggests:** Ghost the rest of a known manufacturer prefix after one or two
  characters, then a mask for the remaining positions.
- **Does not transfer:** Completing a whole VIN is impossible; ghost text only
  helps for positions 1-3 and for fixed-vocabulary positions such as the model
  year [INFERENCE].

### Bundlephobia: a mascot, a big mono field, a proper combobox

- **Link:** <https://bundlephobia.com/>
- **Element:** A line-drawn screaming face above a letter-spaced wordmark
  (BUNDLE bold, PHOBIA light), one tagline, then a 32px Source Code Pro field,
  centred text, placeholder "find package", on dark navy.
- **Behaviour:** A prefix-driven list; the typed prefix is bold in each row
  and a grey description line sits under each name. The input is
  `role="combobox"` with `aria-autocomplete="list"`, `aria-controls`,
  `aria-activedescendant`, `aria-expanded` and `aria-label="Package name"`
  (Downshift).
- **Suggests:** Suggestion rows of "1HG" and "Honda, United States" with the
  typed prefix bold, and those ARIA attributes. One drawn mascot carries the
  personality.
- **Does not transfer:** The suggestions come from a network API.

### what3words: three suggestions, each with a second line that tells them apart

- **Link:** <https://what3words.com/filled.count.soap>
- **Element:** A fixed-format code with a brand glyph: every code is prefixed
  by a red `///`. The code is the URL path.
- **Behaviour:** Typing `filled.count.so` listed exactly three candidates
  (`filled.count.also`, `filled.count.stop`, `filled.count.soon`), each with
  its nearest place underneath. The candidates are near-misses, not pure prefix
  matches.
- **Suggests:** Cap suggestions at three; give each a second line that tells
  them apart (make, country); a small prefix glyph; the VIN as the URL.
- **Does not transfer:** A consent wall covers the page on first load; the
  suggestions are server-side.

### neal.fun, The Password Game: validation as a stack of rule cards

- **Link:** <https://neal.fun/password-game/>
- **Element:** A cream page (`rgb(255,250,233)`), a serif title with an
  asterisk mark, a plain label ("Please choose a password"), one 28px monospace
  field with a live character count to its right.
- **Behaviour:** Rules appear one at a time as you type. Each is a rounded
  card with a tinted header (a cross and "Rule 4") and body text. The failing
  rule is red and sits on top; satisfied rules turn green with a tick and sink
  below it. After `vin95Xk`: Rule 4 red, Rules 3, 2 and 1 green.
- **Suggests:** VIN checks as cards ("17 characters", "no I, O or Q", "check
  digit matches"), the failing one on top, and a count beside the field.
  Mapping states could reuse the same card vocabulary.
- **Does not transfer:** The field is a rich-text editor (ProseMirror
  `contenteditable`), far more than a VIN needs. The page carries ads.

Typeahead also appears in §3: Google Translate's suggestions carry the mapped
name on a second line, and DenCode puts a character-count pill on its input.

## 3. Showing one-to-many mappings

How pages show one item's names or ids in many other systems. Judged against
nitrovin's shape: one source (a VIN, or one group of it), a spine entry (vPIC's
make and model), and from none to many entries in each of several target
vocabularies, with qualities on the edges (direct, matched by rule,
unconfirmed, no name yet, year ranges). In order of fit; explainshell (§1) led
the survey's list as the one that extends an annotated input with lines.

### HGNC gene symbol report: one item's ids in about 30 databases

- **Link:** <https://www.genenames.org/data/gene-symbol-report/#!/hgnc_id/HGNC:1100>
- **Element:** Below the canonical block ("HGNC data for BRCA1": approved
  symbol, previous names, alias symbols) sit grouped panels ("Gene resources",
  "Nucleotide resources", "Protein resources", "Orthologs", "Clinical
  resources"). Each panel is a two-column grid: a bold right-aligned database
  name, then the linked id.
- **Quality:** A small yellow tag reading "Curated" directly after some ids, a
  green "VGNC" tag after others, no tag on the rest: three visible levels of
  provenance. What "Curated" means is stated only by the tag text [INFERENCE:
  manually curated against imported].
- **Empty:** The row stays and the value becomes a link, "Search via BRCA1" or
  "Search via NCBI Gene ID 672", saying which other identifier it will search
  with.
- **One and ten:** Several values stack in the value cell (CCDS shows five,
  each with its own tag); rows with several values get a −/+ toggle before the
  label; each panel collapses.
- **Suggests:** The order, canonical block then target panels, is spine then
  targets. Inside a make-and-model card, one row per target vocabulary: the
  vocabulary, the name, a tag ("direct", "by rule", "unconfirmed"); for "no
  name yet", a search link in place of a blank.
- **Does not transfer:** It needs about 1100px for the two-column grid; a
  cookie dialog covers the page on first load.

### Wiktionary: a tree and a crosswalk in plain text, quality in glyphs and small words

- **Links:**
  - Descendants tree:
    <https://en.wiktionary.org/wiki/Reconstruction:Proto-Germanic/wagnaz#Descendants>
  - Etymology tree box (Polish section, collapsed, click "show"):
    <https://en.wiktionary.org/wiki/wagon#Polish>
  - Translations box: <https://en.wiktionary.org/wiki/vehicle#Translations>
- **Element (descendants):** A nested bullet list, `Language: form, form,
  form`. One source fans out to dozens of targets with no drawn lines; depth is
  indentation. A leading `→` on a bullet marks a borrowed, not inherited, edge;
  a superscript `?` after a form marks an uncertain one; a red link marks a
  form with no entry yet; a leading `*` marks a reconstructed form.
- **Element (etymology tree):** A vertical chain of small rounded boxes joined
  by a short vertical rule. The edge type is an italic abbreviation sitting on
  the rule ("der.", "bor."); an unlabelled rule is the default edge. HTML and
  CSS only (classes `etytree-block`, `etytree-connector-vertical`,
  `etytree-label`).
- **Element (translations):** `Language: term (code) gender, term …` in a
  pale-yellow box per sense. Unverified items live in a separate box titled
  "Translations to be checked" and each carries "(please verify)" on a pale
  green highlight before the term. A language with nothing reads "please add
  this translation if you can" (seen for Shan in the page text).
- **One and ten:** One target is one line; ten values for one target are a
  comma list on that line (Middle English shows 12 forms on two lines).
- **Suggests:** The etymology chain is small enough to hang under a bracket:
  characters, then the vPIC name, then a target name, with "rule" or "?"
  printed on the connector. Target lists as one line per vocabulary;
  unconfirmed names grouped under "to be checked".
- **Does not transfer:** Nothing technical. The glyphs need a legend, and
  Wiktionary gives none on the page. The bullet tree reads as a result list.

### ZOOMA: annotated text chips above a mapping table tinted by confidence

- **Link:** <https://www.ebi.ac.uk/spot/zooma/> (the state is not linkable. On
  the "Annotate Text" tab, type or paste a sentence and press "Annotate Text".
  On the "Map Strings" tab, enter one string a line; "Load examples" fills in
  ten).
- **Element:** After "Annotate Text" the text is echoed in a strip with each
  recognised span turned into a pale green chip (with a ×); text it does not
  recognise stays plain. Directly below, a heading "Results (N out of M
  mapped)" and a table: Text to Map · Term Label · Confidence · Term ID ·
  Source · Prediction Type. "Map Strings" gives the same table with a Term
  Type column.
- **Quality:** Three signals at once. Row tint (pale green, pale yellow, grey;
  grey rows also get a warning mark after the label and grey text); a
  "Confidence" percentage; and a "Prediction Type" link reading Lexical,
  Embedding or Curated, which is the "how was this matched" column. Observed: a
  lone 100% row was green; rows tied at 100%, and rows from 41% to 95%, were
  yellow; rows from 8.75% to 17.5% were grey [INFERENCE: thresholds].
- **Empty:** It depends on the tab. On "Annotate Text" a span with no mapping
  gets no chip and no row, which hides it. On "Map Strings" an unmapped string
  keeps its row: the string, blank cells, a dash for Confidence and for
  Prediction Type, and no tint.
- **The count:** The heading is not a count of terms. Four strings, two of
  them nonsense, gave "2 out of 4 mapped". The ten built-in examples gave "11
  out of 10 mapped" over 19 rows: the first number counted rows, not strings
  [INFERENCE: rows at about 95% confidence or more]. On "Annotate Text" the
  second number was the size of the last "Map Strings" run, not of the
  sentence: the same sentence read "6 out of 10 mapped" and then "6 out of 4
  mapped". The first look at the page recorded "6 out of 7" and "4 out of 7"
  for a typed sample that was not kept (§8).
- **One and ten:** One row per candidate, the source text repeated on every
  row; 14 rows fit a screen.
- **Suggests:** Group chips in the VIN, then rows of group, target name,
  vocabulary and how matched, tinted for direct, by rule and unconfirmed.
- **Does not transfer:** Thumbs-up, thumbs-down and delete buttons are
  curation tooling; percentages imply a score nitrovin does not have.

### roadmap.sh: a spine with dotted branches and a tick-badge legend

- **Link:** <https://roadmap.sh/frontend>
- **Element:** A solid blue vertical spine through bright yellow boxes with
  thick black outlines; dotted blue lines branch left and right to paler boxes,
  alone or in clusters. One large inline SVG, with a hand-drawn look.
- **Quality:** On the target box, not the line. A purple tick badge on the box
  corner means "Personal Recommendation", a green tick "Alternative Option",
  and the legend is printed on the page ("Personal Recommendation /
  Alternative Option / Order not strict on roadmap"). Line style separates
  spine (solid) from branch (dotted).
- **One and ten:** One branch is a single dotted line; four targets fan from
  one point ("Internet" to four boxes); alternatives are packed as a 2×2 block
  with a single line to the block (npm, yarn, pnpm, Bun).
- **Suggests:** Spine boxes in one colour, target-vocabulary boxes paler on
  dotted branches, a corner badge per target for its quality, the legend
  printed once.
- **Does not transfer:** The spine is vertical and a page long: it shows how a
  playful spine and branches look, not how to attach one to a text field. A
  fixed-layout SVG that only scales as a whole; boxes are fixed width, so long
  names would not fit.

### Finto (Skosmos): match quality as the row heading

- **Link:** <https://finto.fi/yso/en/page/p7308>
- **Element:** At the foot of the concept page, a block with rows "Closely
  matching concepts" and "Exactly matching concepts". Each row: a bold heading
  on the left, then one line per mapping, the linked target name (with its
  language) and the target vocabulary's name in plain text.
- **Quality:** The heading itself, closely against exactly. No icons, colour
  or lines.
- **Empty:** A heading with no mappings is not rendered.
- **One and ten:** Values stack under their heading (three under "Exactly
  matching"). The whole block was 230px tall for four mappings in four
  vocabularies: the most compact layout seen.
- **Suggests:** As the content of a card hanging off a bracket: group by
  quality ("Direct", "Matched by rule", "Unconfirmed"), each line a target name
  and its vocabulary.
- **Does not transfer:** Grouping by quality splits one vocabulary across
  groups; for one row per vocabulary, HGNC's layout is the one to use.

### Wikidata statements and Wikipedia's "Authority control" strip: one crosswalk at two densities

- **Links:** <https://www.wikidata.org/wiki/Q29570#P2002> and
  <https://en.wikipedia.org/wiki/General_Motors#External_links> (the box
  "Authority control databases" at the foot, collapsed, click "show").
- **Element (Wikidata):** Each target system is a bordered group with the
  system's name once in a grey left rail and its values stacked in a white
  right cell. Under a value, indented key and value qualifiers ("start time 17
  August 2009", "point in time", "subject named as"). A small three-state rank
  icon sits left of each value and "1 reference" or "0 references" below it.
- **Element (Wikipedia):** One table, with row headings International,
  National, Academics, Artists and Other, each cell a dot-separated run of
  links named by the target (ISNI · VIAF · GND). A second value is shown as
  "(2)" after the name ("ISNI (2)", "Czech Republic (2)"). 21 targets fit in
  five short rows.
- **Quality:** Wikidata has rank, qualifiers and a reference count (no
  deprecated value was seen on this item). Wikipedia has none.
- **Empty:** Both omit absent targets.
- **One and ten:** Wikidata grows very tall (one value with qualifiers is
  about 250px); Wikipedia stays one line per group.
- **Suggests:** Wikidata's qualifier lines are the model for year ranges under
  a mapped name. The Wikipedia strip is the model for a collapsed one-line
  summary that expands to the full branch.
- **Does not transfer:** Wikidata's edit chrome ("edit", "add reference", "add
  value") takes half the width.

### RxNav: related names in typed boxes, counts in the headers, letter badges with a legend

- **Link:** <https://mor.nlm.nih.gov/RxNav/search?searchBy=String&searchTerm=atorvastatin>
  (a banner says the URL moves on 10 Feb 2027).
- **Element:** The "Graph" tab, "Classic" view: a 3×4 grid of boxes around a
  centre logo. Each box header has a code tab, a name and a count ("IN/MIN ·
  Ingredient (3)", "BN · Brand Name (3)", "SBDC · Branded Drug Component
  (13)"). Rows are names; the searched concept's row is tinted peach.
- **Quality:** One to three tiny coloured letter badges before each name (H,
  Rx, S, M, …) with a "Legend" block, and filter switches for the same letters
  in the left rail.
- **Empty:** A type with no entries keeps its box, empty.
- **One and ten:** Boxes are fixed height and scroll internally past about six
  rows; the count in the header says how many.
- **Suggests:** One small box per target vocabulary with `(n)` in the header
  and a one-letter quality badge per row, the legend printed once.
- **Does not transfer:** No lines at all; relation is by position, and it
  reads as a dashboard. Internally scrolling boxes; the grid needs desktop
  width; dated styling.

### Translation tools: a primary result, demoted alternatives, a popover anchored to one word

- **Links:**
  - Linguee: <https://www.linguee.com/english-german/search?source=auto&query=vehicle>
  - Google Translate: <https://translate.google.com/?sl=en&tl=de&text=vehicle&op=translate>
  - DeepL (then click a word in the right pane):
    <https://www.deepl.com/en/translator#en/de/The%20vehicle%20was%20built%20at%20the%20Bowling%20Green%20plant.>
- **Element (Linguee):** The source word, then indented translations each with
  an example pair; then an italic grey "less common:" line followed by seven
  more translations inline, dot-separated. Further down, a section headed
  "External sources (not reviewed)" (seen in the page text).
- **Element (Google):** The input shows a typeahead list where each suggestion
  has its translation on a second grey line ("vehicles / Fahrzeuge"). The
  result card shows one large translation, then "More translations" rows: the
  article and word, the part of speech, back-translation chips, a chevron, with
  "Expand all".
- **Element (DeepL):** Clicking a word in the translation tints that word blue
  and opens a popover directly under it with tabs "Alternatives | Dictionary"
  and four alternatives, each shown with the following word for context ("Auto
  wurde ...").
- **Quality:** By tier and by words ("less common:", "not reviewed"), never by
  line or icon. Empty states were not seen.
- **One and ten:** One big primary; the rest shrink to an inline list (Linguee)
  or one-line rows (Google).
- **Suggests:** DeepL's popover is a branch that opens from the clicked piece
  of text: clicking a bracket could list other vocabularies' names under those
  characters. Typeahead rows could show a VIN prefix with the make and model on
  a second line. Rule-matched names could go after a quiet "by rule:" line.
- **Does not transfer:** Google truncates long targets with an ellipsis ("die
  Trägersubst…"); DeepL's popover covers the text below it.

### Mix'n'match: named match states with fixed colours, provenance per row

- **Links:** <https://mix-n-match.toolforge.org/#/catalog/1> and
  <https://mix-n-match.toolforge.org/#/list/1/auto>
- **Element (catalog):** Four states, each a labelled bar with a count and a
  percentage: "Fully matched" green, "Preliminarily matched" blue, "Unmatched"
  orange, "Not applicable to Wikidata" grey. The same four colours form one
  stacked bar above.
- **Element (list):** Each entry is two lines. Line 1: the source name in
  green and its description. Line 2: the candidate target in blue with its id
  in brackets, and its description. At the right of line 1, the text
  "Automatic, preliminary matcher".
- **Quality:** The state's name and who or what made the match, in words. "Not
  applicable" is a state distinct from "unmatched".
- **Suggests:** Four states map onto direct, matched by rule, unconfirmed and
  no name yet, each with one colour used everywhere; a row is the vPIC name
  over the target name, with "matched by rule" at the right.
- **Does not transfer:** It is a work queue, built for long paged lists (50 a
  page), with Confirm and Remove buttons. The page hung headless WebKit and
  loaded only in Chromium.

### EMBL-EBI OxO: weaker edge kinds off by default, "(+N)" showing what is hidden

- **Link:** <https://www.ebi.ac.uk/spot/oxo/search/DOID%3A162>
- **Element:** Under the single term input, an "Also include:" row of
  checkboxes: "Subclass (rdfs:subClassOf) (+1)", "Cross-reference
  (oboInOwl:hasDbXref) (+5)", "Obsolete terms (+0)". Help text reads "A loose
  link between two identifiers, with no stated meaning." and "Counts show
  additional results for your current search."
- **Results table:** Subject · Predicate · Object · Mapping justification ·
  Mapping confidence · Distance · Type · Mapping set. Missing confidence is an
  em dash.
- **Quality:** The predicate's name (SKOS:exactMatch), the justification
  ("MappingChaining"), confidence and distance (hops), each its own column.
- **One and ten:** Poor. The subject is repeated on every row and each row is
  about 100px, so seven mappings fill the screen.
- **Suggests:** Show direct mappings by default with a toggle such as "also
  show unconfirmed (+3)".
- **Does not transfer:** The table itself: too many columns for a branch.

### FHIR ConceptMap narrative: the relationship in its own column, "(not mapped)" rows

- **Links:** <https://build.fhir.org/conceptmap-example-specimen-type.html> and
  <https://hl7.org/fhir/R4/cm-address-use-v3.html>
- **Element:** A table: source code and name, relationship, target code,
  dependent columns, comment. The relationship is a link in the middle column
  ("is equivalent to"; the R4 page uses "equal" and "narrower").
- **Empty:** The middle cell reads "(not mapped)" and it and the target cells
  are filled grey.
- **One-to-many:** The R4 page leaves the source cell blank on the second row
  ("old" to OLD, then blank to BAD); the CI-build page repeats the source (CNJT
  three times).
- **Suggests:** The reading order source, relation word, target: each branch
  line reads `Corvette — matched by rule → Corvette Stingray`, and a target
  with nothing gets a grey "(no name yet)" line.
- **Does not transfer:** It is a reference table, not an extension of an
  input.

### One input fanned out to many labelled outputs, with explicit empty states

- **Links:** <https://dencode.com/en/string?v=1G1YY22G965109876>,
  <https://www.nslookup.io/domains/github.com/dns-records/> and
  <https://www.whatsmydns.net/#AAAA/github.com>
- **DenCode:** A full-width input on top with a character-count pill ("17") at
  its corner; below, rows of bold label and value separated by dashed rules,
  updating as you type. A format that yields nothing keeps its row with the
  label greyed and no value (Binary String, Base32 and Base45 for this input).
  The URL carries the input.
- **nslookup.io:** One heading per record type; an empty type keeps its
  heading with a sentence ("No AAAA records found.", "No CNAME record found.").
  Recognised TXT values become logo cards (Adobe, Apple, Atlassian …) under
  "Site ownership verification".
- **whatsmydns:** 22 target rows of flag, place and operator with a status
  mark at the right; for this query every row showed a red cross and the same
  marks were plotted on a map.
- **One and ten:** All three are fixed lists of targets, so the page has the
  same shape whether one target answers or all of them.
- **Suggests:** Every target vocabulary keeps a row; empty ones are greyed or
  say "no EPA name yet"; a "17" counter pill on the VIN input.
- **Does not transfer:** nslookup's logo cards are large (five to a row);
  whatsmydns' map.

### GitHub Actions run graph: cards joined by thin elbow connectors, "N jobs" grouping

- **Link:** <https://github.com/withastro/astro/actions/runs/44107565155>
- **Element:** Small dark cards left to right joined by thin grey rounded
  elbows with a dot at each end. A card that stands for many items has a tab
  label ("Matrix: build") and reads "2 jobs completed · Show all jobs".
- **Quality:** On the node (a green tick, a grey "!" for cancelled), not the
  line.
- **One and ten:** Three sources fan to five targets through one shared
  vertical trunk; many-item nodes collapse to a count.
- **Suggests:** The connector style (a trunk and elbows with end dots) would
  suit bracket-to-card links. A vocabulary with many names collapses to "ACES ·
  6 names · show all".
- **Does not transfer:** The horizontal left-to-right layout needs width; the
  graph is pannable.

## 4. Multi-level drill-down and parts fitment

For extending the page from the vehicle's names down into parts: a catalog
(part type and count), candidate fitments, stock (shelf rows, yard vehicles)
and related vehicles, without leaving the page or burying the VIN plate. In
order of fit. Images were blocked by the browser on some of these pages; the
structure is what was read.

### Tesla Parts Catalog: one callout, all the candidates for that position

- **Links:**
  - Diagram state, reached by clicking the tile:
    <https://parts.tesla.com/en-US/catalogs/9678bc03-3011-4c95-9620-bb32d391cac0/systemGroups/f5f8b1f8-12ca-4bf3-af94-38351c7dd1ba?categoryExternalReference=4140f488-b94b-4369-9c61-5f8ca6373c08>
  - Category state, loaded cold and restored correctly:
    <https://parts.tesla.com/en-US/catalogs?catalogExternalReference=9678bc03-3011-4c95-9620-bb32d391cac0&categoryExternalReference=4140f488-b94b-4369-9c61-5f8ca6373c08>
- **Element:** An inline SVG exploded view on the left with small green
  numbered callout dots on leader lines; a parts list on the right whose rows
  start with the same number.
- **Behaviour:** Clicking callout 5 opened a popover anchored to the diagram
  listing all four parts that can sit at position 5 (left and right, base and
  sport), each with part number, name, price and one qualifier line, and the
  side list scrolled to row 5 and shaded it.
- **Qualifiers:** A plain second line per row: "Service option code BC3B",
  "Option codes: BC3R", and a caveat written as a sentence: "Ok to use 00-D in
  all countries except S.Korea". Side is in the name ("LEFT HAND"). Alternates
  repeat the callout number ("1." twice, for base and sport rotor);
  un-numbered hardware gets "*.".
- **Also:** The landing page says, above the VIN tab: "VIN Search is
  recommended for catalog selection and fitment checking." The breadcrumb
  "Home › Model 3 Jun 2017 - Dec 2023" keeps the vehicle above everything.
- **At 390:** Works. The diagram stacks above the list, no horizontal overflow
  (document width 390), zoom buttons overlay the drawing. A callout tap at 390
  was not tested.
- **Suggests:** This is the candidate list. One part type is one position; its
  candidates are rows that share the position and differ by a short option
  line; the caveat lives in that same line, as words. The popover on a callout
  continues bracket, leader line, value card.
- **Does not transfer:** It needs a real drawing per assembly, and nitrovin
  has none. Callout numbers are SVG paths, not text (0 `<text>` nodes), so
  there is nothing for a screen reader. In WebKit the category tile grid
  rendered overlapped and unclickable (it worked in Chromium). Hover and
  keyboard behaviour were not tested.

### PatternFly "compound expandable" table: the count is the button

- **Link:** <https://www.patternfly.org/components/table/react/compound-expandable/>
- **Element:** Each row has several count cells (an icon and "10", "4", "4").
  Each count is a `<button aria-expanded>`; the open one gets a top border and
  a panel opens directly under the row, spanning its width. Other rows stay in
  place.
- **At 390:** Works. The row becomes a label and value stack ("Branches 10",
  "Pull requests 4"), counts stay tappable, and the panel opens below with its
  column name as a heading. The document was 512px wide at 390 in this bare
  demo frame (a floating theme switcher); the table itself fit.
- **Suggests:** "Alternator 3 · Brake rotor front 2" is this exactly. The
  count is the handle, the panel opens under its own row, and the catalog above
  and below does not move away. The same row could carry two counts (fitments,
  in stock) that open different panels.
- **Does not transfer:** The demo panel is one level deep; nesting a second
  compound table inside is not shown. [INFERENCE] Opening a second cell in the
  same row swaps the panel (one `aria-expanded=true` was seen; another was not
  clicked).

### RockAuto: an inline tree, a pinned breadcrumb, qualifier chips that say what they hid

- **Links:**
  - Candidates for one part type:
    <https://www.rockauto.com/en/catalog/ford,2004,mustang,3.9l+v6,1432947,electrical,alternator+/+generator,2412>
  - A category open, siblings visible:
    <https://www.rockauto.com/en/catalog/ford,2004,mustang,3.9l+v6,1432947,brake+&+wheel+hub>
- **Element:** One page from make to part. Each level expands in place under
  its parent with dotted indent rails; a yellow breadcrumb bar (FORD > 2004 >
  MUSTANG > 3.9L V6 > Electrical) stays pinned above the scrolling tree; the
  URL path grows one comma-segment per level. The candidate listing opens
  inside the tree, with sibling part types continuing below it.
- **Qualifiers:** Each candidate is two lines: brand, part number, grade and
  an Info link, then a plain option line ("105 Amps", "w/ 105 Amps
  Alternator", "Clock: 11:00; 105 Amps (Only 1 Remaining)", "Clockwise
  Rotation; 105A; 1.1KW"). Above the list the distinct qualifier values appear
  as chips (105 Amps / 110 Amps / 130 Amps).
- **Measured:** Clicking "130 Amps" turned 17 listings into 1, and a red
  button appeared reading "Reset (16 listings hidden)".
- **At 390:** Partly. The document is 492px wide in a 390px viewport: the row
  of section tabs is a 487px table whose last control ("Bigger", x = 437 to
  492) sits off the right edge. On the category page the tree holds: its panel
  is 385px wide, the labels fit, and the breadcrumb wraps to two lines and
  stays pinned. On the candidates page the panel widens to 425px and each
  listing row to 417px, so the last price column (x = 365 to 402) is cut at
  the right edge until scrolled sideways; the qualifier chips and "Add to
  Cart" fit. The tree scrolls inside a window 406px tall of the 844, between a
  header and tab row of 285px and a footer of 151px; the document itself does
  not scroll. RockAuto links a separate "Mobile Site".
- **The owner's view:** Not something the survey observed. The project owner,
  who buys parts, calls RockAuto "strangely the gold standard of auto parts
  website UX. It's ugly, but so dense, intuitive, and quick, that every parts
  buyer bookmarks it."
- **Suggests:** Inline expansion under the parent keeps everything above it on
  screen. The growing breadcrumb is the cheapest "where am I" for three or four
  levels. Chips derived from the candidates themselves narrow "4WD / w/ towing
  package" without a form. Saying how many were hidden is the honest version
  of a filter.
- **Does not transfer:** The survey listed the visual density and table layout
  here; the owner's view disagrees about the density. No left or right and no
  replaced part numbers are shown on this listing. The chips filter but do not
  say which value your vehicle has.

### car-part.com: the option chooser, then stock rows that name the donor vehicle

- **Link:** <https://www.car-part.com/> (the start; the two result steps are
  form POSTs to `/cgi-bin/search.cgi` with no deep link).
- **Steps run:** 2004 · Chevy Truck-Silverado 1500 (1999 Up) · Alternator ·
  ZIP 49024, Search, pick the first option, Search.
- **Element (option chooser):** A boxed card headed with the vehicle and part,
  then four radio lines in catalog shorthand: "w/o hybrid option; 145 amp (opt
  KG3)", "w/o hybrid option; 105 amp (opt K68)", "hybrid option (opt HP2),
  (inside transmission bell housing), generator rotor (inner)", "… generator
  stator (outer)".
- **Element (stock):** A table of physical parts. Columns: Year, Part and
  Model of the donor vehicle; Description (the option text again, "(145 amp,
  opt KG3),5.3L"); Part Grade (A, C, X); Stock#; Price; Dealer (name, city,
  phone); Distance. The first rows for a 2004 Silverado query are a 2005 GMC
  Envoy and a 2005 Buick Rainier: the interchange is shown by simply listing
  other vehicles as sources. Header links: "see All Interchange Choices" and
  "see Non-Interchange Choices". Page-top caveats are sentences ("Some parts
  may not be available for immediate pickup. Confirm store pickup time with
  dealer.").
- **Also:** The front page has a "SmartVin: enter your VIN to simplify your
  vehicle option questions" input.
- **At 390:** Fails. The small choice card is already 524px wide; the results
  table is a fixed wide table.
- **Suggests:** Candidates and stock in the wild. The option text is shown
  verbatim, as the thing chosen between; stock rows carry stock number,
  location and grade; the yard vehicles a part could come off are a column,
  not a separate screen. SmartVin's claim is one nitrovin could make good on,
  by pre-selecting or striking the options a VIN rules out.
- **Does not transfer:** Two full page loads, no URL state, no way back except
  the browser; the table needs about 1000px.

### iFixit teardown step: colour is the leader line

- **Link:** <https://www.ifixit.com/Teardown/iPhone+13+Pro+Teardown/144928#s294270>
  (Step 7; click the second thumbnail).
- **Element:** A photo of a logic board on the left with seven coloured
  outline boxes (red, orange, yellow, green, light blue, blue, violet) around
  chips; on the right a bullet list where each bullet dot is the matching
  colour and the text names the chip. No lines are drawn; the colour does the
  linking.
- **At 390:** Works. Image, thumbnails, then bullets, with no overflow; the
  pairing survives because nothing depended on position.
- **Suggests:** The rule for phones, proven on a dense object with seven
  parts. For parts: one flat colour per candidate inside an open part type,
  reused on the candidate row, its stock rows and its yard rows.
- **Does not transfer:** The boxes are baked into the image, so there is no
  hover or focus link between bullet and box. Seven colours is the ceiling of
  the scheme. Colour alone fails for colour-blind readers unless the bullet
  also carries a number or shape.

### BrickLink part page: "Item Appears In" as counts, alternates and variants as plain lines

- **Link:** <https://www.bricklink.com/v2/catalog/catalogitem.page?P=3001#T=S&O={%22iconly%22:0}>
- **Element:** Under the part name, "Item No: 3001  Alternate Item No: 3001f1,
  3556, 15589, 54534, 72841". A three-column block: Item Info, "Item Consists
  Of", and "Item Appears In" with linked counts (4181 Sets, 19 Minifigures, 26
  Parts, 33 Books, 18 Gear). Collapsible "Additional Notes" ("This part has a
  dizzying array of variants, only a few of which have separate catalog
  entries.") and "Related Items: This Item is a mold variant of the following
  Item(s)", listing variants whose names end in the distinguishing detail ("…
  Slotted (with 2 slots, near corner left)", "… near corner right"). The hash
  carries the open tab.
- **Suggests:** "Related vehicles that share fitments" is "Item Appears In": a
  count per kind, each a link, sitting beside the item and not on another page.
  Alternate numbers on the identity line are the calm way to show that one
  number replaced another: part of the number line, not a warning banner.
- **Does not transfer:** An old fixed-width layout; the counts navigate away
  and do not expand in place. Not checked at 390.

### Pick-n-Pull inventory: a yard header, then row number and set date

- **Link:** <https://www.picknpull.com/check-inventory/vehicle-search?make=234&model=4352&distance=50&zip=95123&year=>
- **Element:** Results grouped by yard. Group header: the yard's name,
  "(Approx. 8.9 miles)", address, phone, "Store Layout Map", "Part Pricing".
  Rows: photo, Year, Make, Model, Row, Set Date ("2017 Toyota Camry · 35 ·
  10/26/2026"). A sentence with a warning triangle above: "Part availability,
  vehicle condition and position in the yard may vary as our inventory changes
  daily." The empty state (seen with a model that had no stock) says to leave
  the year blank or "ask for an interchangeable list that will show other
  vehicles that may work for the part".
- **At 390:** Works. The table becomes stacked cards ("2017 Toyota Camry / Row
  35 / Set: 10/26/2026") with no overflow.
- **Suggests:** The shape of a yard row: group by location, then vehicle, row
  and date, with the freshness caveat once per group and not per row.
- **Does not transfer:** It lists whole vehicles, not parts, so it has no
  option text or part number.

### eBay Motors listing: a compatibility table with a Notes column

- **Link:** <https://www.ebay.com/itm/157124355412> (scroll to "Compatibility"
  under "About this item").
- **Element:** Near the price, a line "Compatibility: See compatible vehicles"
  jumps down to a table headed "This part is compatible with 32 vehicle(s)".
  Columns: Year, Make, Model, Trim, Engine, Notes. The Notes cell holds the
  qualifier ("135 A/12 V, Clockwise Rotation, 6-Groove Clutch Pulley")
  truncated with a "Read more" link.
- **At 390:** Fails in this layout. The table is 639px wide inside a 321px
  section, so Notes is off-screen behind a horizontal scroll. [INFERENCE] eBay
  serves a different page to real phones; the window was only resized.
- **Suggests:** The reverse lookup, part to vehicles, as a flat table with the
  count in its heading: a model for a "related vehicles" list once opened.
- **Does not transfer:** The deciding text is in the last column and
  truncated; 32 near-identical rows that differ only by trim are noise.

### PartSelect model diagram: the number on the card is the number on the drawing

- **Links:**
  - Diagram:
    <https://www.partselect.com/Models/WDT750SAHZ0/Sections/Pump-Washarm-And-Motor-Parts/?ModelID=7178085&ModelNum=WDT750SAHZ0&mfgModelNum=&ManufactureID=3&Selected=E8871A2C7CFE71AB3B9FA8DD77A53E64A8720168&Position=8&mfg=Whirlpool&Type=Dishwasher&Mark=8>
  - Part page, with the line naming the numbers it replaces:
    <https://www.partselect.com/PS3406971-Whirlpool-W10195416-Lower-Dishrack-Wheel.htm?SourceCode=19&SearchTerm=WDT750SAHZ0&ModelNum=WDT750SAHZ0>
- **Element:** Three panes under one model header: a horizontal strip of
  diagram thumbnails (the assemblies); a scrolling list of part cards, each
  with an orange circular number badge, price, "In Stock" and part number; and
  the exploded drawing with the same numbers on arrows. The drawing is a plain
  GIF that opens larger in a modal on click; nothing links a number to its card
  except the matching digit. On the part page a replaced number is one
  sentence: "Part# W10195416 replaces these: W10195416V, W10195416VP".
- **At 390:** Works structurally (no overflow; strip, then drawing, then list)
  but the page is 6312px tall and the list ends up far below the drawing.
- **Suggests:** The badge. A small round number or letter on each candidate
  card is enough to tie it to a mark elsewhere, with no hit-testing. The
  thumbnail strip is a horizontal level switcher that costs one row of height.
- **Does not transfer:** It needs drawings; a marketing modal on arrival; the
  list and drawing scroll independently, so the pairing is by eye.

### RealOEM: the classic diagram and table, with Supp., From, Up To and Notes

- **Links:** front brake,
  <https://www.realoem.com/bmw/enUS/showparts?id=DT63-USA-09-2002-E39-BMW-530i&diagId=34_1123>;
  alternator,
  <https://www.realoem.com/bmw/enUS/showparts?id=DT63-USA-09-2002-E39-BMW-530i&diagId=12_2313>
- **Element:** The breadcrumb "Select another car > BMW 5 Series E39 530i >
  Engine electrical system > Alternator"; a line drawing with numbered leader
  lines and a tiny car silhouette with a dot showing where the assembly sits; a
  table with columns No., Description, Supp., Qty, From, Up To, Part Number,
  Price, Notes. Left and right share a number ("03 Protection plate left", "03
  Protection plate right"). Alternates share a number and differ in Supp. ("01
  RP REMAN compact alternator · 120A VALEO" against "· 120A BOSCH"). Notes
  carried "ENDED" and "only in conjunction with".
- **At 390:** Fails. The document is 640px wide; drawing and table both scroll
  sideways.
- **Suggests:** The column set is a checklist for a candidate row: a qualifier
  (Supp.), a validity range (From, Up To), a status word (ENDED). The locator
  silhouette is a small playful idea: a tiny car outline with a dot for "front
  left".
- **Does not transfer:** The table needs width; ads and donation overlays. The
  site states it is under heavy bot load, so it should not be scripted
  against.

### GitHub Projects: a side pane with its own URL

- **Link:** <https://github.com/orgs/github/projects/4247?pane=issue&itemId=88046209&issue=github%7Croadmap%7C1025>
- **Element:** Clicking a card title on the board slides a pane in from the
  right covering about 90% of the width; the left edge of the board stays
  visible and the page behind keeps its scroll. The URL gains
  `?pane=issue&itemId=…`, so the open item can be linked and Back closes it. A
  close button sits top right.
- **At 390:** A full-screen takeover with a close button; the board is gone
  until it is closed. Acceptable for a leaf, not for a level drilled from.
- **Suggests:** A way to show one deep thing (a candidate's stock and yard
  rows) without navigating, with a shareable address. The sliver of the page
  left visible is what says "you have not left".
- **Does not transfer:** It covers the VIN plate. A pane on nitrovin would have
  to start below the plate or be narrow enough to leave it readable.

### GitHub code view: a tree on desktop, a breadcrumb on phones

- **Link:** <https://github.com/torvalds/linux/blob/master/drivers/gpu/drm/amd/display/Kconfig>
- **Element:** A left sidebar tree with ancestors expanded and the current
  node highlighted with a blue bar; a slash-separated breadcrumb above the
  content (linux / drivers / gpu / drm / amd / display / Kconfig) where every
  segment is a link.
- **At 390:** Works. The tree collapses to a "← Files" button and the
  breadcrumb wraps to two lines; six levels remain legible.
- **Suggests:** The same ancestry shown twice on desktop, once on phones:
  indent on wide screens, a breadcrumb of chips on narrow ones (vehicle ›
  Alternator › 8266R › stock).
- **Does not transfer:** A persistent sidebar competes with the VIN plate for
  the top left.

### JSON Hero: Miller columns with a path bar and "Related values"

- **Link:** <https://jsonhero.io/j/PjHo1o5MVeH4?path=0.owner.login>
- **Element:** Columns left to right (root, 0, owner), each row showing a type
  icon, a key, and either a value preview or a child count with a chevron ("77
  fields ›", "18 fields ›"). A path bar above ("root › 0 › owner › login") with
  back and forward arrows. A right-hand inspector for the selected leaf with a
  "Related values" list (the same key across siblings). The footer shows the
  keys: arrows Navigate, History, Reset path, Copy selected node. The path is
  in the URL.
- **At 390:** Does not work. The site shows "JSON Hero only works on desktop
  (For now!)".
- **Suggests:** "N fields ›" is the same count-as-handle as PatternFly's;
  "Related values" is the sibling comparison candidates need; the path bar is
  the breadcrumb again.
- **Does not transfer:** At three levels the first column is already scrolled
  half off the left edge at 1280px, so the top context is not actually kept;
  the columns need the full page width.

### Andy Matuschak's notes: stacked panes with spines

- **Link:** <https://notes.andymatuschak.org/About_these_notes?stackedNotes=z5E5QawiXCMbtNtupvxeoEX&stackedNotes=z2hQEhqWkdRLL9JUwfawZZx>
- **Element:** Clicking a link opens the target as a new pane to the right;
  earlier panes slide left and collapse to a vertical title spine ("About
  these notes" rotated 90°) that stays clickable. The link that opened the
  current pane stays highlighted in its parent. The stack is in the URL
  (`stackedNotes=` repeated).
- **At 390:** The stack is gone. Only the newest note shows under the site
  title; ancestors are reachable only with Back.
- **Suggests:** The highlighted "this is what you opened" row in the parent,
  and a collapsed ancestor that shrinks to its title and does not disappear.
- **Does not transfer:** Horizontal growth fights a page whose focus is a wide
  VIN plate at the top; three panes fill 1280px.

### 7zap: the vehicle pinned as chips, a caveat banner, the tree in the hash

- **Link:** <https://7zap.com/en/catalog/cars/toyota/4runner-n210-facelift-parts-catalog/#region=usa&section=brake-system>
- **Element:** A header card that persists at every level: a car picture,
  chips "Toyota 4Runner (N210) Facelift (2006–2009)" and "USA", small print
  "Catalog filtered by vehicle parameters. Use VIN for exact fitment.", and a
  yellow banner "For an exact parts match, please provide your vehicle's
  parameters! Change parameters". Below, a left tree (Chassis Systems › Brake
  System › nine assemblies) that expands in place next to a grid of
  line-drawing tiles; the open branch is written into the URL hash.
- **Suggests:** The wording and placement of "we could not confirm X": one
  sentence attached to the vehicle header, with the action that would fix it.
  nitrovin's version is per candidate ("unconfirmed: Drive Type") but the tone
  is right.
- **Does not transfer:** The deepest level, the drawing, is behind "Sign up or
  sign in to view the diagram", and part numbers are masked ("04*****9*"). Not
  checked at 390.

### Narrower references

- **Park Tool "Where's the problem?"**:
  <https://www.parktool.com/en-us/blog/repair-help> (click the bike button). A
  full-screen overlay: part names in a left menu and a blue line drawing of a
  bike; hovering "DERAILLEUR" tints the derailleurs yellow on the drawing
  (stacked transparent PNGs, one per part). Charming and cheap. The page itself
  says "(Tablet or Desktop)"; at 390 the menu fills the screen and the bike is
  below the fold with no hover.
- **W3C APG treegrid example**:
  <https://www.w3.org/WAI/ARIA/apg/patterns/treegrid/examples/treegrid-1/>.
  Rows carry `aria-level` 1–4 and `aria-expanded`; documented keys: Right Arrow
  expands a collapsed row, Left Arrow collapses or moves to the parent. The
  reference for keyboard behaviour if catalog, candidates and stock are built
  as nested rows.
- **PatternFly primary-detail**:
  <https://www.patternfly.org/patterns/primary-detail/react-demos/primary-detail-full-page/>.
  Clicking a list row opens a drawer over the right third; the list stays, with
  a breadcrumb above both. At 390 the drawer replaces the list under the
  breadcrumb.
- **FCP Euro product page**:
  <https://www.fcpeuro.com/products/bmw-cable-alternator-starter-genuine-bmw-12421436547>.
  The unconfirmed-fit state: "Make sure this fits your car", a yellow "Validate
  fitment" button with a warning mark, and the sentence "To ensure fitment and
  browse only products that fit, please select a vehicle first." OE Numbers and
  MFG Numbers as labelled columns.
- **Amazon search**:
  <https://www.amazon.com/s?k=alternator+2004+ford+mustang+3.9>. Above the
  results: "Filter results by your vehicle: 2004 Ford Mustang  [Filter results]
  [Change vehicle]". The vehicle was read out of the query text and offered
  back as a chip to confirm.
- **finderjs demo**: <https://markmatyas.dev/finderjs/> (Example #3). Plain
  Miller columns with a preview pane. At 390 the widget is 560px wide and
  overflows.

## 5. Style and automotive motifs

What gives a single-input page its character, and the physical objects a VIN
page can borrow from. The static infographics (§1), crontab.guru and cidr.xyz
(§1), and Bundlephobia and The Password Game (§2) are style references too.

### Tactile UI Odometer: drums in a screwed metal plate

- **Links:** <https://tactileui.dev/objects/odometer>; the author's notes,
  <https://dev.to/cipriantitire/i-made-a-web-odometer-that-carries-like-the-real-one-1i6i>
- **Element:** Six digit drums (`001,099`) in a dark window set into a brushed
  plate with a screw head at each end. Controls under it: "+1 CARRY", "+37
  ROLL", "×10 SPIN", "? RANDOM". The toolbar has "SOUND" and "REDUCED MOTION"
  toggles.
- **From the author's notes:** CSS 3D ten-sided drums; the faces are real
  text; screen readers hear the number once when it lands; reduced motion swaps
  the number without rolling; MIT; about 8 KB of script, 4.2 KB of CSS and a
  20.7 KB numerals font.
- **Suggests:** The serial (positions 12-17) rolling in on decode; the screwed
  plate as a ready frame for the whole VIN.
- **Does not transfer:** The drums are digits only (the article says the
  bundled numerals are Latin only), so VIN letters need a different face set.
  Sound must stay opt-in.

### Carwow "Enter reg": the input styled as the physical plate

- **Links:** <https://www.carwow.co.uk/sell-my-car>; a font to look at in
  place of Carwow's, K-Type Mandatory, <https://www.k-type.com/fonts/mandatory/>
  (its poster:
  <https://www.k-type.com/wp-content/uploads/2013/04/mandatory_poster-1.png>)
- **Element:** A 313×48px white plate-shaped field labelled "Your vehicle
  registration". Computed style: `font-family: wowplate, monospace`, 40px,
  weight 700, `text-transform: uppercase`, centred, placeholder "ENTER REG" in
  grey. Around it: a very heavy all-caps headline, a pastel peach-to-aqua
  gradient, two cars cropped in from the left and right edges, a black button
  with cyan text.
- **Suggests:** The VIN field set in a plate or VIN-tag face, uppercase,
  centred, with illustration peeking in from the edges and not competing with
  the field.
- **Does not transfer:** `text-transform` only changes the look; the value
  stays as typed. Carwow's font is proprietary. K-Type Mandatory is a full font
  drawn from the UK plate face, but its page says "free for personal use" (§8).

### Overpass: road-sign type that is open source and has a mono cut

- **Link:** <https://overpassfont.org/>
- **Element:** "An open source font family inspired by Highway Gothic", eight
  weights and Overpass Mono ("monospace for that analog feel"). The page uses
  a highway-sign green top bar with white caps tabs ("OVERPASS / TRY IT / TRY
  MONO"), a very heavy all-caps wordmark, and letters that fly in and assemble
  ("Smashed Apart").
- **Suggests:** A road-sign voice for group labels, Overpass Mono for the VIN
  cells, green-and-white sign panels as label boxes.
- **Does not transfer:** Nothing technical. For the licence, see §8.

### Embossed label effect: label-maker tape in CSS

- **Link:** <https://text-effects.colorion.co/effects/embossed-label-text-effect>
- **Element:** CSS-only label-maker tape: `font: 900 27px 'Courier New'`,
  `letter-spacing: .1em`, three stacked text-shadows for the raised glyphs,
  repeating gradients for the tape's ridges, a per-letter stagger of .13s with
  `steps()`, `prefers-reduced-motion` honoured, and a wrapper with `role="img"`
  and an `aria-label`.
- **Suggests:** A route to a stamped VIN tag with no image.

### 7 Letter License Plate Generator: a plate drawn in the DOM

- **Link:** <https://7letterlicenseplategenerator.app/>
- **Element:** A DOM-drawn plate with bolt holes that updates as you type, a
  "7/7" counter in the field, inline rule errors, and a "Try: 1H9CM82"
  suggestion chip.
- **Does not transfer:** Generic styling; useful only for those four parts.

### ACME License Maker: copy voice and a rendered plate

- **Link:** <https://www.acme.com/licensemaker/>
- **Element:** The copy voice ("Pretend you're in prison and make a
  license!"), 505 plate blanks, a rendered embossed plate with your text, and a
  "recent licenses made by other folks" gallery.
- **Does not transfer:** A form submit and a server-rendered JPEG; not live.

### Car Dashboard study: amber gauges and dotted leaders

- **Link:** <https://shchilkin.dev/design-lab/studies/car-dashboard/>
- **Element:** Amber on black, ring gauges with a glowing arc, a trip panel
  whose rows use dotted leaders ("DISTANCE ……… 0.0 km"), a "000000 km" odometer
  line, a P R N D S selector and a round "ENGINE START STOP" button.
- **Does not transfer:** The simulation and the audio. The dotted-leader rows
  and the start button do.

### Lando Norris: a racing palette and a pass card

- **Link:** <https://landonorris.com/>
- **Element:** Fluorescent yellow-green `rgb(210,255,0)` on off-white
  `rgb(244,244,237)` and dark olive `rgb(40,44,32)`; Mona Sans Variable with a
  display face (Brier); hairline contour loops behind the content; a small
  "NEXT RACE" pass card with a circuit outline.
- **Does not transfer:** The WebGL portrait. The palette and the pass card do.

### HTTP Cats: one joke, held consistently, and a hidden command line

- **Link:** <https://http.cat/status/418>
- **Element:** Each code is a URL and a photo. Pressing `:` opens a vim-style
  command box (": type a command") listing `status {code}`, `blog`,
  `lang {eng|cat}`, `theme {dark|light}`, `vim {on|off}`, the first row
  highlighted red.
- **Suggests:** One running conceit in place of many jokes; a keyboard-only
  extra for people who look for it; one URL per code.
- **Does not transfer:** The charm rests on photographs.

### Regexper: a wordmark, a tagline and a railroad diagram

- **Links:** a VIN pattern at
  <https://regexper.com/#%5E%5BA-HJ-NPR-Z0-9%5D%7B8%7D%5B0-9X%5D%5BA-HJ-NPR-Z0-9%5D%7B2%7D%5Cd%7B6%7D%24>
  and another at
  <https://regexper.com/#%5E%5BA-HJ-NPR-Z0-9%5D%7B3%7D%5BA-HJ-NPR-Z0-9%5D%7B5%7D%5B0-9X%5D%5BA-HJ-NPR-Z0-9%5D%7B8%7D%24>
- **Element:** A full-width field and a railroad diagram as inline SVG with
  rounded-corner paths (`q-10,0 -10,10`), captioned "One of:" boxes and
  loop-backs labelled with a count ("7 times", "2 times"). A brush-lettered
  wordmark, the tagline "You thought you only had two problems…", and a
  permalink in the hash.
- **Does not transfer:** Not live: it needs the "Display" button. The diagram
  suits alternatives and repeats, and is not aligned to the input string.

### tixy.land: one field, one colour

- **Link:** <https://tixy.land/?code=sin%28y%2F8%2Bt%29>
- **Element:** A black page, one red, 16×16 dots, and a single short code
  field shown as the body of `(t,i,x,y) =>`. Onboarding is two comment lines
  above the field ("// click the dots for more info"; not exercised). State is
  in `?code=`.
- **Does not transfer:** The animation is canvas.

## 6. Looked at and passed over

Opened and seen, but weaker than the entries above or not suited to the
question.

- **LeaderLine**, <https://anseki.github.io/leader-line/>. Draws a line
  between any two DOM elements, each its own absolutely positioned
  `svg.leader-line` (the demo page had 116), with a `position()` call needed
  after layout changes. Badges read v1.0.8, "No dependency", MIT. A new
  dependency for what explainshell does with a few `M/V/H` paths.
- **learn me a bitcoin**, <https://learnmeabitcoin.com/technical/transaction/#examples>.
  A long hex string as nested coloured spans with tippy.js tooltips naming each
  field, and Raw and Split tabs. Useful, but outranked by the entries in §1.
- **ASN.1 JavaScript decoder**, <https://lapo.it/asn1js/#MAoGCCqGSIb3DQIF>.
  Hovering a tree node highlights its bytes in the hex dump and shows an offset
  and length tooltip. Useful, but outranked by the entries in §1.
- **Wise IBAN checker**, <https://wise.com/gb/iban/checker>. After checking a
  real IBAN the breakdown still showed placeholder segments ("GB · 12 · ABCD ·
  123*** · 123*****") with labels, not the typed value.
- **CarShots VIN decoder**, <https://carshots.ai/vin-decoder>. Its "position
  by position" section is a three-column table.
- **chmod-calculator.com**. A checkbox grid with heavy ads and no annotated
  string.
- **strfti.me**. A reference table with no per-token annotation of the typed
  format.
- **OHDSI Athena concept page**, <https://athena.ohdsi.org/search-terms/terms/201826>.
  "Term connections (435)": a flat four-column table with at least 14
  vocabularies mixed in one list; the relationship name is printed once and
  left blank for the following rows of the same kind. No confidence, no empty
  state. A licence dialog covers the page and did not close; Accept was not
  pressed.
- **deps.dev dependencies**, <https://deps.dev/npm/express/5.1.0/dependencies>.
  A Table and Graph toggle; the table is the default and the graph is opt-in.
- **npmgraph**, <https://npmgraph.js.org/?q=express>. A full node graph, 68
  nodes, crossing curves, unreadable without pan and zoom. The side panel's
  counted sections are the readable part.
- **Mergely**, <https://editor.mergely.com/>. Two columns with thin bracket
  connectors in the gutter (canvas). Pairs ranges one to one only.
- **Parallel Sets**, <https://www.jasondavies.com/parallel-sets/>. Ribbon
  width encodes counts; a single item's mappings have no quantities, so the
  ribbons would carry nothing.
- **Every Time Zone**, <https://everytimezone.com/>. One bright draggable
  needle crosses 13 or more labelled rows, each showing its value at the
  crossing. Playful, but the fan-out is a shared axis, not a mapping.
- **r12a Unicode converter**, <https://r12a.github.io/app-conversion/?q=VIN%20%E2%86%92%20%E8%BB%8A>.
  A green input box, then about 15 labelled grey output boxes in two columns,
  each editable with its own Convert button.
- **MusicBrainz external links**, <https://musicbrainz.org/artist/b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d>.
  A sidebar list of 40 or more targets as favicon and name. No quality marks,
  no values.
- **CLDR Zone → Tzid chart**, <https://www.unicode.org/cldr/charts/latest/supplemental/zone_tzid.html>.
  A plain three-column table; one-to-many by repeating the key on every row,
  region "001" first as the default. Read as text only.
- **AfterShip tracking**, <https://www.aftership.com/track>. No carrier
  detection while typing a UPS number; detection happens after submit.
- **GOV.UK vehicle services**, <https://www.check-mot.service.gov.uk/> and
  <https://vehicleenquiry.service.gov.uk/>. Plain 19px GOV.UK text fields, no
  plate styling.
- **webuyanycar**, <https://www.webuyanycar.com/>. Redirected to the US site,
  which has no plate field.
- **cmdk**, <https://cmdk.paco.me/>. Now redirects to the GitHub repository;
  no live demo.
- **Algolia autocomplete guide**, <https://www.algolia.com/doc/guides/building-search-ui/ui-and-ux-patterns/autocomplete/js>.
  Documentation only; the live demo is behind CodeSandbox.
- **Sketchfab animated disassembly**, <https://sketchfab.com/3d-models/disassembly-of-a-2-stroke-engine-animated-d772d848f2714a72bfa69f77ba59225f>.
  A WebGL engine that comes apart along a scrub bar. This model has no
  annotations and an annotated one was not found. Needs a canvas and a 3D
  asset.
- **LEGO building instructions** (`lego.com/en-us/service/building-instructions/10497`):
  PDF downloads only, no diagram on the page. **IKEA spare parts**: an
  article-number form with no exploded view. **Fairphone spare parts**: device
  chips leading to a product grid, no exploded view.
- **LKQ Pick Your Part** (`pyp.com/inventory/`): loads, but needs a location
  chosen before any inventory; Pick-n-Pull covered the shape.

## 7. Findings the surveys agree on

Each finding names the references that carry it. The first six shaped the page
most.

1. **Lines do not survive phones; colour, or a number, carries the link.**
   Every interactive example gives each group one colour and repeats it on the
   explanation. Lines are the first thing removed on small screens (Crew Chief
   hides its brackets at 540px, GetVIN its arch below `lg`). The two that keep
   lines at any width, explainshell and FreeVINGenerator, are the two that
   break at 390: explainshell redraws its lines on resize and still loses them,
   because its layout is a fixed 940px and its overlay only as wide as the
   screen, and FreeVINGenerator's fixed 1402px drawing shows three of its 17
   characters. cidr.xyz and float.exposed wrap cleanly because colour is the
   only link; iFixit pairs seven chips with seven bullets by colour alone;
   Tesla, PartSelect and RealOEM pair by number. Where the mark and the row
   cannot sit side by side, the shared key still works and a drawn line does
   not. Colour on its own needs a second cue for colour-blind readers
   (rvcodec.js, iFixit).
2. **At scale, mapping quality is written, not drawn.** Quality sits at the
   value as a word, tag or glyph: "Curated" (HGNC); "(please verify)",
   "Translations to be checked", `→`, `?`, "bor.", "der." (Wiktionary); "less
   common:" and "not reviewed" (Linguee); "Automatic, preliminary matcher"
   (Mix'n'match); Lexical, Embedding or Curated with a row tint (ZOOMA);
   closely against exactly as a heading (Finto); tick badges with a legend
   (roadmap.sh). Line style carried quality nowhere: on roadmap.sh solid
   against dotted means spine against branch. The target system is named once
   on one side and its values stack on the other (Wikidata, HGNC, Finto,
   DenCode, nslookup.io, RxNav), and none of the tools with more than about
   eight targets draws a line per target. Drawn connectors appear only for a
   short source and few targets, where hover or grouping keeps them legible:
   explainshell (six lines), the Wiktionary etymology chain, GitHub Actions (a
   shared trunk), roadmap.sh (a cluster shares one line). Full graphs and
   ribbons (npmgraph, Parallel Sets, Mergely) needed pan and zoom or a canvas.
   Fitment catalogs agree: a caveat is a sentence placed where the doubt
   applies, on the vehicle header (7zap, FCP Euro), once per yard
   (Pick-n-Pull), or on the candidate row (Tesla, RealOEM's "ENDED").
3. **Keep one real input and draw around it.** input-otp lays a transparent
   input over drawn cells; jwt.io lays a transparent text area over a coloured
   copy; typeahead.js puts a read-only input underneath for the ghost text;
   Carwow only changes the font. Most of the annotated-string tools draw on a
   copy of the string beside a plain field (explainshell, Crew Chief, Polypane,
   rvcodec.js, FreeVINGenerator); only crontab.guru and float.exposed keep one
   oversized field as the hero and annotate around it. The two that replaced
   the input (Wordle, Keycode) lost paste; one field per character
   (react-otp-input, cidr.xyz) costs a tab stop and a screen-reader field each.
4. **Clean pasted input before testing it.** A strict pattern rejects a paste
   whole (`48a1` into input-otp's digits-only field), and no pattern keeps the
   spaces and dashes (`48 21-77` became `48 21-`). A VIN arrives with spaces,
   dashes and lowercase letters. The cleaning must not leak into the value:
   Cleave.js writes its spaces into the field, and `text-transform` on Carwow
   changes the look and not the value.
5. **The count is the drill handle, and one answer leads.** Counts open
   things: PatternFly's "10" branches, JSON Hero's "77 fields ›", BrickLink's
   "4181 Sets", eBay's "32 vehicle(s)", RxNav's "Ingredient (3)", GitHub
   Actions' "2 jobs completed · Show all jobs", Wikipedia's "ISNI (2)", OxO's
   "(+5)". Filters say what they removed: RockAuto's "Reset (16 listings
   hidden)". One primary result with the rest behind a quiet word or a count
   ("less common:", "More translations", −/+ toggles, collapsed "show" boxes)
   is how every mapping page stayed the same shape for one value and for ten.
6. **An empty target keeps its place and says so.** A greyed label (DenCode),
   "No AAAA records found." (nslookup.io), a grey "(not mapped)" row (FHIR),
   "please add this translation if you can" (Wiktionary), a "Search via …"
   link (HGNC), an em dash (OxO), a red cross (whatsmydns), an empty box
   (RxNav). Finto and Wikidata drop empty targets. ZOOMA does both: its string
   mapper keeps an unmapped string's row with dashes in it, while its text
   annotator drops what it cannot map and leaves a heading count that does not
   count terms.
7. **Select one part, dim the rest, show one detail area.** explainshell dims
   to 0.4 and Crew Chief to 0.5, and each fills a single panel or box. The
   better ones work in both directions (RegExr, rvcodec.js, vindecode.app) and
   from the caret or keyboard (crontab.guru, Crew Chief). Jisho and DeepL open
   detail from the clicked piece of text.
8. **Brackets are three-segment shapes, leaders are right-angled, and fixed
   cells make alignment arithmetic.** Brackets are `v8 h58 v-8` (CheckerVIN),
   `V5 H563 V0` (explainshell) or a `border-top` (Crew Chief). Leaders go
   straight up or down with labels alternating above and below
   (FreeVINGenerator, UAW, The Conversation), or step into a side gutter at
   staggered depths (explainshell). Nobody uses curves except GetVIN's single
   arch. The tools that position from tile size or the monospace grid stay
   aligned; the one that measures the DOM (explainshell) has to redraw on every
   scroll and resize, and still loses its lines at 390. A row of 17 cells has
   to be sized to fit: vindecode.app's 17th tile wrapped in a 624px column,
   react-otp-input's 17 fields wrapped 11 + 6 in about 1070px, and Crew
   Chief's 44px tiles shrank to 38px in a 718px row.
9. **A second vocabulary goes on the opposite side or in a parallel row.**
   Node.js puts one naming scheme above the string and the other below.
   rvcodec.js and Polypane use parallel rows or cards that share the token's
   colour, which also handles one-to-many and split fields without forking a
   line.
10. **Inline expansion and breadcrumbs survive 390; side-by-side does not.**
    PatternFly's panel and GitHub's breadcrumb held up. RockAuto's tree and
    pinned breadcrumb held up too, inside a page that is 492px wide at 390 and
    whose candidate rows lose their last price column. JSON Hero refuses
    phones, finderjs overflows, Matuschak's stack drops to one pane, and the
    GitHub pane and the PatternFly drawer become full-screen.
11. **Alternates share a position and differ by one plain line.** Tesla
    repeats "1." and "5.", RealOEM repeats "01" and "03", RockAuto and
    car-part.com list rows that differ only in the option line. Nobody hides
    the option text behind a tooltip: it is the second line of the row, in
    catalog shorthand. A replaced part number is likewise a line on the
    identity ("Alternate Item No" on BrickLink, "replaces these" on
    PartSelect). Density and speed are what the people who buy parts value
    over looks: that is the project owner's view of RockAuto, recorded in its
    entry (§4), not a survey observation.
12. **State lives in the URL and follows what is typed or opened.**
    crontab.guru's hash, explainshell's `?cmd=`, Regexper's hash, tixy's
    `?code=`, the paths of what3words and http.cat, float.exposed, rvcodec.js,
    DenCode; for open levels, RockAuto's path segments, 7zap's hash, GitHub's
    `?pane=`, Matuschak's `stackedNotes=`, JSON Hero's `?path=`, Tesla's
    catalog ids. car-part.com is the counter-example and is the one that feels
    like leaving.
13. **A prefix earns an early reward.** Cleave.js lights a brand after two
    digits, typeahead.js ghosts the rest, Bundlephobia and what3words give
    short lists with a second line that tells candidates apart, Google
    Translate puts the mapped name on each suggestion's second line.
14. **One big monospace field, one accent, one small consistent device.** 30px
    JetBrains Mono with pale yellow (crontab.guru), 32px Source Code Pro
    (Bundlephobia), 28px monospace on cream (The Password Game), one red on
    black (tixy). Colour carries state, not decoration: three fills per
    character (Wordle), the accent turning salmon for an error with no message
    (crontab.guru), red and green rule cards (The Password Game). Personality
    comes from one device held consistently: a drawn mascot (Bundlephobia), a
    tagline (Regexper), a "random" tab (crontab.guru), a hidden `:` command box
    (http.cat), a copy voice (ACME), or a physical object rendered faithfully
    (odometer drums, a plate font, embossed tape).

## 8. Limits

### How the pages were examined

- Browsers and sizes differed by survey: Chromium at 1280×900 and 390×844 for
  the annotated inputs; headless Chromium or WebKit at 1280×900, or a plain
  text fetch, for the mapping pages; headless WebKit at 1280×800 for the
  playful inputs; headed WebKit (Chromium for Tesla) with 390×844 for the
  fitment pages. The second looks were in WebKit at 1280×900 and 390×844.
- Of the mapping pages, only explainshell was looked at at 390. crontab.guru,
  BrickLink and 7zap were not checked at a narrow width at all. eBay's 390
  result is a resized desktop window, not a phone.
- Typed input was used on ZOOMA and Mergely; neither state can be linked.
  car-part.com's results have no deep link, and Tesla's diagram link was
  reached by clicking a tile.
- Not tested: whether GetVIN swaps in another VIN; a reduced-motion switch in
  Rough Notation; hover and keyboard on Tesla's diagram and a callout tap at
  390; the second count cell on PatternFly; the hover on jwt.io's claim keys;
  the dots on tixy.land; input-otp's `pasteTransformer`.
- Where the surveys disagreed, the page was opened again the same day. All of
  them opened, and the entries, the findings and §9 carry what the second
  look showed:
  - **explainshell at 390.** One survey had the lines cut at the left, two had
    the content running off the right. Both happen (§1). One had found no
    resize handler in `es.js`: there is none there, but the page's inline
    script has one, and the lines are redrawn. The end dot is
    `<circle r="4">`, which one note gave as "4px". The dimming on hover,
    exercised by one survey and not by another, was seen again. One note gave
    the colours as a small purple, green and blue set; they are drawn from a
    shuffled list of 20.
  - **ZOOMA's count.** The first notes give "6 out of 7 mapped" in one place
    and "4 out of 7 mapped" in another, for a typed sample that was not kept,
    so neither could be reproduced. The heading turned out not to count terms
    (§3), and neither figure is used.
  - **RockAuto at 390.** Recorded both as "partly" and as having held up.
    Partly is right (§4).
  - **FreeVINGenerator.** Eight groups and eight boxes at every width; one
    note said its layout fits seven.
  - **VIN groupings.** Crew Chief, GetVIN, CheckerVIN, vindecode.app and DRiV
    were read again for their own groupings (§1, §9). The UAW and The
    Conversation diagrams are pictures and their groupings were not read.
  - **Crew Chief's sizes.** Its style sheet has the 44px tiles and bracket
    widths the survey recorded; in the second look they rendered smaller (§1).
- regex101 showed "Engine Error" in the headless browser. Mix'n'match hung
  headless WebKit and loaded only in Chromium. Tesla's tile grid was
  unclickable in WebKit.
- RealOEM states it is under heavy bot load: it should not be scripted
  against. RxNav's banner says its URL moves on 10 Feb 2027.

### Could not be opened, or opened without the element

- **Timed out, hung or did not resolve:** sizemytires.com/explanation;
  Partsouq (`partsouq.com/en/catalog/genuine/pick?c=Toyota`, 60 seconds,
  twice); an Amazon product page (`/dp/B0GH1XWQPF`), so the "fits your vehicle"
  widget was not seen; <https://www.carreg.co.uk/number-plate-viewer>;
  <https://anyparcel.net/>.
- **Bot checks:** <https://monkeytype.com/>, BioPortal mappings and etymonline
  (Cloudflare checks that never cleared); Rebrickable (403); eReplacementParts
  (a model page once, then a challenge; no diagram reached); id.loc.gov (403
  from a text fetch, not retried in a browser).
- **Sign-in:** the NLM UMLS Metathesaurus browser; 7zap's diagram layer (its
  tree level is public).
- **Gone:** <https://jessepollak.github.io/card/> (404); the two CMSNL fiche
  URLs tried (404); a guessed Partzilla diagram URL (404); the GOV.UK Miller
  columns demo (`alphagov.github.io/miller-columns-element/` 404, its component
  guide page 500).
- **Not opened:** etytree, Schema.org, the ISO browsing platform, transit-style
  line maps; GitHub code search, Linear and Raycast (login or native app);
  PartsTech, Linear's and Stripe's side peeks (login); HondaPartsNow and other
  dealer OEM catalogs; McMaster-Carr; Shimano's exploded-view PDFs.
- **Not found:** a small inline Sankey widget; an annotated 3D disassembly.

### Fonts

- **Overpass** is published under the SIL Open Font License 1.1. The survey
  itself saw only "open source" on the font's page; the licence was read
  afterwards.
- **K-Type Mandatory** is "free for personal use" by its own page, which does
  not cover an open-source tool.
- **Carwow's `wowplate`** is proprietary.
- The spec's rule follows from this: fonts only from licences that have been
  read. The prototype as it stands ships no font (§9).

### The owner's example image

The "What's in a VIN?" image the work began from comes from the UAW
Union-Built Vehicle Guide (§1). It was matched by text, not by sight: the PDF's
extracted text was searched and the page was not rendered. The text contains
"What's in a VIN?", "WHERE VEHICLE WAS BUILT", "VEHICLE SECURITY CODE", "PLANT
THAT ASSEMBLED THE VEHICLE" and the year and manufacturer tables, matching the
image.

## 9. What the page took

The spec in the PR's description records the decisions; this section sets them
and the page's code (`pkg/ui/page.html`, `pkg/ui/diagram.go`,
`pkg/ui/static/ui.css`, `pkg/ui/static/plate.js`) against the findings in §7.
It describes the page on 2026-11-17, in draft PR
[#303](https://github.com/joshuasprow/nitrovin/pull/303). Its tests are
repaired: `pkg/ui`'s tests and the browser runs in `cmd/nitrovin` describe this
page, the typing on the plate included.

### The plate and its six groups

- **One real field under 17 drawn cells** (finding 3; input-otp). The form is
  the plate. Without JavaScript the field is an input below the cells and a
  Decode button sends it. With JavaScript it is a textarea in the input's
  place, laid over the cells, its own text, caret and selection not shown: the
  cells show what is typed. Either way it is one field and one tab stop.
- **The plate edits as a text field, because it is one.** The field's
  characters are spaced a cell apart and its line is a cell high, so the
  browser's own caret and selection, by click, drag, touch or key, fall on the
  cells, and the cells draw them: a bar in the gap between two cells for the
  caret, the selected cells inked, the next empty cell marked. Under a mouse a
  faint caret shows the gap a click would pick. A whole VIN takes no more: a
  character typed into it is refused and nothing is pushed off its end. Where
  a pointer lands is the script's to say, from the cell under it, because the
  browsers do not agree with the cells: Firefox halves a character at its
  glyph and not at its cell, and a phone places the caret of a tap at an end
  of the word, a VIN being one word. So a click or a tap's gap, and the cells
  a mouse drags over, are worked out from the pointer and held there; double
  clicks and taps, Shift and a click, and a phone's drags and handles stay
  the browser's own.
- **Pasted input is cleaned first** (finding 4). Only letters and digits are
  kept, upper-cased, and cut at 17, by the same rule on the server and in the
  browser. Letters are never swapped: an O stays an O, and a character no VIN
  holds (I, O, Q) turns its cell red at once. The value itself stays clean:
  the grouping is drawn, not typed into the field (against Cleave.js).
- **Six groups, the standard ones:** maker (positions 1–3), vehicle (4–8),
  check digit (9), model year (10), plant (11), serial (12–17). The examples
  do not agree on a grouping. Crew Chief brackets four (3, 5, 1 and 8
  characters), CheckerVIN seven (1, 2, 5, 1, 1, 1, 6) and FreeVINGenerator
  eight (1, 1, 1, 5, 1, 1, 1, 6). GetVIN, vindecode.app and DRiV have the
  page's six (3, 5, 1, 1, 1, 6). Each group's label sits on a short bracket
  above the plate and a bracket closes under it.
- **A fixed grid, nothing measured to draw** (finding 8). One CSS grid of 17
  columns holds labels, cells and brackets; the cards are six equal columns.
  The cells scale with the plate's width. No script reads a position to draw
  anything, so the page with and without JavaScript is the same drawing, and
  there is nothing to redraw on scroll or resize (explainshell redraws on
  both). The script reads the cells' places for one thing: the gap nearest a
  pointer.
- **On a phone the plate is two rows** (below 34rem), so a cell is wide enough
  for a finger. It splits where a VIN does, after the check digit: nine cells
  above (maker, vehicle, check digit) under their labels, and eight below
  (year, plant, serial) over theirs. The grid becomes nine columns and the
  field's text wraps with the cells, nine characters to a line, which is why
  the field is a textarea.
- **Shown where it is wrong.** At 17 characters the check-digit card says
  "Checks out" or "Should be N" and opens the arithmetic (each character's
  value, weight and product, the sum, the remainder). The decoder's suggested
  correction is one link. This follows the verdict lines of jwt.io and Crew
  Chief; The Password Game's rule cards were not taken.
- **Not taken:** a decoded sentence above the field (crontab.guru: the spec
  rules out a one-line summary); a group picked out by the caret or by hover
  with the rest dimmed (finding 7: the page shows all six cards at once, and
  its one hover mark is the faint caret); ghost text or a suggestion list
  (typeahead.js, Bundlephobia, what3words: the spec defines typeahead as
  naming the manufacturer as soon as three characters match, and recent
  VINs); a character counter in the field; odometer drums for the serial; any
  sound.
- **In the spec, not yet built:** the model-year hint on the year bracket, the
  maker's country on the maker card, and announcing each completed group to
  screen readers. Today the answer is tied to the input with
  `aria-describedby` and the decorative rows are hidden from assistive
  technology.

### Cards on leader lines

- **Six cards in a row under the plate, each on a right-angled leader line**
  (finding 8; explainshell's `M/V/H` paths, the UAW boxes). The lines are six
  paths in one SVG drawn to the grid's fractions (a 1020-unit box: 60 a cell,
  170 a card), stretched with the page and kept at a constant stroke.
- **One colour per group** on its label, its cells' underline, its bracket,
  its line and its card (finding 1), with the group's name and position range
  as the card's heading, so colour is never the only cue.
- **Below 46rem the lines go** and the brackets and colours stay; the cards
  stack under the plate in bracket order (finding 1; Crew Chief, cidr.xyz,
  iFixit). The labels of the three one-cell groups are hidden there and their
  cards name them, until the plate is two rows and has room for them again.
- **A card is present but empty until its characters are typed**, faint and
  dashed (finding 6). The server decodes whatever is typed so far: the make at
  3 characters, model, trim, body and engine at 8, year and plant at 11 (as the
  spec records). The early manufacturer is Cleave.js's prefix reward (finding
  13).
- **Contents are fixed:** maker (make, manufacturer, vehicle type); vehicle
  (model, trim, body, engine); check digit; model year, with a strip of the
  three year codes either side and this one marked (GetVIN's highlighted
  table cell); plant (city, country with its flag beside the name); serial.
- **Motion** (Rough Notation's draw-on, without the library): a group just
  completed draws its bracket and line in and pops its card, and a character
  stamps into its cell. None of it under reduced motion.
- **Live, and in the URL** (finding 12). With JavaScript the page asks the
  server 150 ms after the last key, drops an answer still on its way, and
  swaps in the same server-drawn diagram. The URL follows what is typed in
  place; only a whole VIN adds a history entry, so Back steps through VINs and
  not keystrokes.
- **VINs to try are a menu in the header,** not chips on the page. The VINs
  button opens it, a popover, so it opens without a script: the examples, five
  VINs that each have a name in the demo set's vehicle vocabulary and in
  EPA's; in their place, once there are any, the whole VINs that decoded in
  this browser (the last eight, kept in the browser), with "Clear recent"; and
  "Surprise me", which draws from 24 curated VINs (crontab.guru's "random",
  jwt.io's "Generate example").
- **Vocabularies and Developers are header buttons too,** beside VINs, each
  opening a popover dialog. Nothing floats over the page. Developers holds the
  answer's own request (its URL, `curl`, the CLI) and its JSON, then install,
  MCP, verify and links.

### The stamped-metal look

- **Variant A, stamped metal, was chosen.** The spec asked for three
  switchable treatments of the plate, picked in the browser. A road-sign
  variant and a label-maker variant were built and dropped (compare Overpass
  and the embossed label effect in §5), and the code now carries one look.
  The label-maker tape came back for the names alone (below).
- **Where it comes from:** the UAW diagram (a pale blue ground, heavy black
  capitals, boxes on bracket lines) and Tactile UI's odometer (a brushed plate
  with a screw at each end). Everything pressed or read as a heading is the
  same metal: the plate, the buttons, the VINs in the menu, the brand and the
  dialogs' heads. The cards are tags hung from their lines by an eyelet.
- **How it is made:** gradients and shadows for the metal's body, one small
  SVG of stretched noise for the brushed grain (as light streaks, and as a
  mask that cuts dark ones), characters struck with a light and a dark text
  shadow. No image of a plate and no web font: the labels use a system stack
  led by Arial Black and the cells the system monospace, so neither Overpass
  nor K-Type Mandatory ships.
- **A wear texture was tried on the metal and removed.**
- **Voice:** short and a little wry, never jokey ("Type a VIN.", "No name for
  this one yet."), one device held consistently (finding 14).

### Names as stacked strips of tape, doubt drawn dashed

- **"Also known as" is the first derived card**, joined from the maker, the
  vehicle and the year. It has no box: those three groups' lines run on down
  through it. The old three-column map with drawn lines is deleted.
- **Every vocabulary loaded, with no choosing.** The card shows the vehicle's
  names in every vocabulary loaded on vPIC's spine. The old page's "Translate
  to" checkboxes and the visitor's kept selection of targets are gone: what is
  loaded is what is shown.
- **A name is a strip of black label-maker tape** ([textures.md](textures.md),
  candidate 1), the one thing on the page that is not metal, as a parts bin's
  label is not: the vocabulary stamped at the left, then the make under the
  maker's line, the rest of the entry under the vehicle's, and the years the
  mapping holds for under the year's, each word raised white over its group's
  colour (finding 9: a parallel row sharing the group's colour, as in
  rvcodec.js and Polypane, not Node.js's above-and-below). The tape is black
  in both colour schemes, and the card's caption is a strip of it too. The
  words keep their case, where real tape is capitals only, because a
  vocabulary's entry is case-exact.
- **One stack per loaded vocabulary, always** (finding 6), a vocabulary with a
  name listed before one without. A vocabulary with no name keeps a blank
  dashed strip, "No name for this one yet."; before the model is known the
  card reads "Names turn up once the model is known."; with no vocabulary
  loaded it offers "Add a vocabulary".
- **No line is drawn to a blank name.** When the first stack is blank (no
  vocabulary, no model yet, or no vocabulary with a name), the three lines to
  the card are not drawn.
- **Several names are a stack, and the count is its switch** (finding 5). The
  first name lies on top with the edges of up to three more showing under it.
  The vocabulary's stamp on the top strip carries the count and opens the
  stack into one strip, a row to a name, with no script. Which name leads is a
  stated first judgment: the decoded make before another make, nothing
  unchecked before a doubt, a listed mapping before one matched by rule, the
  shorter model name before the longer. The spec's wording was five names
  visible and then "+N more"; the page shows one and a count.
- **Doubt is drawn as dashes, against finding 2.** The mapping survey found
  quality written in a word, tag or glyph at the value, and line style
  carrying quality nowhere; the spec accordingly said "quality written as tags
  in words". The page instead dashes the lines into the make and the
  model, and the marks under them, when the mapping was matched by rule, and
  dashes the bracket and mark of the model or the year when a qualifier or the
  model year could not be checked. The words exist but are one step away: each
  segment's hover title says "Matched by rule" or "Model year not checked",
  and a small button on the strip opens how the name was reached. On a narrow
  screen the lines and brackets of the strips go, and only the dashed
  underline under the word, and those words behind it, still carry the doubt.
- **The lines stay few.** Three lines enter the top strip of each stack, from
  three groups, which is within what finding 2 saw drawn connectors survive: a
  short source and few targets.

### Designed, not built

- **"Parts that fit"**, a second derived card, is in the spec only. The
  browser's catalog, stock and fit code stays for it. Its design follows the
  fitment findings:
  - one row per part type with its count ("Alternator 3"); the count is a
    button that opens the candidates in place, under its own row (finding 5;
    PatternFly);
  - a candidate is one row: the part number, then one plain line that tells it
    apart ("4WD, with towing package"), side as a tag, and any caveat as a
    sentence on that row ("Drive type not confirmed", "Replaced by DEMO-1001")
    (findings 2 and 11; Tesla, RockAuto, car-part.com, RealOEM);
  - with more than five candidates, chips built from their differences narrow
    the list and say how many are hidden (RockAuto);
  - stock opens in place under a candidate from a count on its row ("2 on the
    shelf", "1 in the yard"): shelf rows with location, yard rows with the
    donor vehicle (car-part.com, Pick-n-Pull); three levels at most, all in the
    one card (finding 10);
  - going elsewhere (a related vehicle, a part number's other vehicles) loads
    it into the plate with a new URL and history entry, and a short trail of
    chips above the plate shows the way there (GitHub's breadcrumb); the open
    part type and candidate are in the URL, without JavaScript too (finding
    12);
  - the plate shrinks and stays pinned at the top while scrolling (RockAuto's
    pinned breadcrumb, 7zap's persistent vehicle header);
  - no exploded parts diagram, because the catalogs carry no drawings (what
    Tesla, PartSelect and RealOEM cannot transfer);
  - the card is to be judged against RockAuto's density: by the owner's view
    (§4), a parts buyer values dense, quick and intuitive over good-looking.

### Dropped

- **The input accepting other kinds.** The spec had the one input also taking
  a vehicle ("2015 F-150") or a part number, guessing the kind and showing it
  as a tag on the plate, with brackets over the words. It was never built and
  was dropped on 2026-11-17: the plate takes a VIN and nothing else. A VIN is
  17 fixed characters, which is what lets the cells, the two rows on a phone
  and the caret between cells be drawn by arithmetic; words of any length
  would give that up. Looking a vehicle up by name or a part by number, if it
  comes, is a control of its own and not the plate.
