# The design canvas, as last read

A snapshot of the Claude Design canvas where unbuilt UI is drawn. The canvas
is the source of truth and is private to the maintainer; this folder is its
record. [AGENTS.md](../../../AGENTS.md) ("Design Canvas") says how the two are
kept together.

## What is here

`project/` holds the canvas's own files, as read down, for the pages
"nitrovin UI" and "Swatches" only. A proposal's page and a page drawn by
hand stay on the canvas. `scripts/canvas.sh --import` is how a read-down
gets here, and `scripts/canvas.sh --check` fails on anything else:

- `canvas.json`: the index, cut to those two pages. Each frame's place and
  size, the pages, the notes. It is not what is published:
  `scripts/canvas.sh --publish` writes that, from the canvas's whole index.
- One `.dc.html` per frame.

`_blob/` is not the canvas's: it links each uploaded image's address to its
original in `pkg/ui/static/` (see "Rebuilding a canvas from a snapshot").

The canvas's page, "nitrovin UI", is one row of six frames, each named after the part of [DESIGN.md](../../../DESIGN.md) it illustrates.

| Frame | Shows |
| --- | --- |
| `Main` | The VIN input, both modes |
| `Materials` | Materials and elevation |
| `Diagram` | Cards, brackets and lines |
| `Fitment` | The fitment card |
| `Controls` | Dialogs, forms and inputs |
| `States` | The empty places and other states |

Each frame is drawn twice: its twin, in the dark scheme, sits under it in a
file named with `-Dark`. A frame is fixed to its scheme; none has a switch.

The frames are redrawn from the page as built (#334). Each holds the page's
own markup, captured from a server with the demo set, and the rules of
`pkg/ui/static/ui.css` that the markup uses. A specimen sits in a box classed
`vp-wide`, `vp-tab` or `vp-phone`, and the stylesheet's width and
colour-scheme queries are resolved for that box: the page at 1280, 768 and
390 wide.

`scripts/canvas.sh` builds the frames, and its header says how. No frame is
edited by hand: a change to the page is a rerun, and the diff of this folder
is what to publish.

A page per proposal appears when something is drawn before it is
built, named for what it proposes. Its frames are not `scripts/canvas.sh`'s:
each is captured the same way from a prototype running on a throwaway
branch, and that branch, not this folder, is their record. When the
proposal is built its frames are folded into the six and the page goes.
The part type picker of #366 was the first.

A page, "Swatches", holds five boards in two rows. The first row is what
was made for the page: `Swatches-Page`, the materials in use and the marks
and fittings made of them, each chip the stylesheet's own value; and
`Swatches-Tried`, what was drawn and is not on the page, set
aside, each naming where its drawing is kept. The second row is what was
found for the page's look, none of it settled: `Swatches-Thermal`,
`Swatches-Yard` and `Swatches-Vehicle`, a web search's findings of
2026-11-20. Each source of a found swatch is linked by its full address and was opened
and checked against its card on 2026-11-21: a card marks a page that backs
it only in part, one read as a search result's summary, and one that could
not be read, and lists what it says that no source read does. Its colours
are estimates, since no source publishes one. `scripts/canvas/swatches.mjs` builds all five from the files
of `scripts/canvas/swatches/`. `scripts/canvas.sh` does not build these
boards and leaves them alone.

The wireframes and texture frames that proposed the fitment slices (#320 to
#325) are removed from the canvas. This folder's history holds them.

## This snapshot

Canvas version `1795401466-a169`, published on 2026-11-22.

## Rebuilding a canvas from a snapshot

Publish this folder as the root of a new Design canvas. Every frame and its dark twin
draws their textures from two
images uploaded to the original canvas, at addresses that resolve only there:

| Address in the frames | Original |
| --- | --- |
| `/_blob/9d26d515fa68a38e6638bb4a880cacf5` | `pkg/ui/static/brushed.svg` |
| `/_blob/8df67eb7edb3679ce1a61d297a1fcd76` | `pkg/ui/static/paper.svg` |

Upload both to the new canvas and replace each address with the one the
upload returns. Without that, those frames draw with no texture.

`_blob/` holds one link per upload, named for its address and pointing at
its original. A reader that draws the frames off the canvas, from this
folder, can take each `/_blob/<id>` from `_blob/<id>`.

## Data

Every part, number and stock row in the frames is the demo set's invented
data. Decoded values are the decoder's own for the demo set's VINs.
