# The design canvas, as last read

A snapshot of the Claude Design canvas where unbuilt UI is drawn. The canvas
is the source of truth and is private to the maintainer; this folder is its
record. [AGENTS.md](../../../AGENTS.md) ("Design Canvas") says how the two are
kept together.

## What is here

`project/` holds the canvas's own files, exactly as read down:

- `canvas.json`: the index. Each frame's place and size, the pages, the notes.
- One `.dc.html` per frame.

`_blob/` is not the canvas's: it links each uploaded image's address to its
original in `pkg/ui/static/` (see "Rebuilding a canvas from a snapshot").

The canvas has two pages. The first, "nitrovin UI", is one row of six frames,
each named after the part of [DESIGN.md](../../../DESIGN.md) it illustrates.

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

The second page, "Proposals", holds what is not built yet. Its frames are
not `scripts/canvas.sh`'s: each is captured the same way from a prototype
running on a throwaway branch, which its row's title names by issue.

| Frames | Proposes | Prototype |
| --- | --- | --- |
| `Proposal-PartType-1280`, `-768`, `-390` | #366: the row of part types and its dialog become one picker under the "Parts that fit" strip | branch `prototype/part-types-366` |

The wireframes and texture frames that proposed the fitment slices (#320 to
#325) are removed from the canvas. This folder's history holds them.

## This snapshot

Canvas version `1795207513-3bbe`, published on 2026-11-20.

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
