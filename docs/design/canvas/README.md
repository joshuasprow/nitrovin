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

The canvas is one page of four rows. A frame's title starts with its row and
its place in it.

| Row | Frames |
| --- | --- |
| 1. The page as built, what `DESIGN.md` describes | `Main` (the VIN input, both modes), `Materials` (materials and elevation), `Diagram` (cards, brackets and lines), `Fitment` (the fitment card), `Controls` (dialogs, forms and inputs), `States` |
| 2. Wireframes, VIN mode (#320 to #324) | `Wire-Vin-Desktop`, `Wire-Vin-Tablet`, `Wire-Vin-Phone` (the page at 1280, 768 and 390 wide), `Wire-Vin-FirstView`, `Wire-Vin-Empty`, `Wire-Vin-TypesDialog` |
| 3. Wireframes, part-number mode (#325) | `Wire-Part-Desktop`, `Wire-Part-Phone`, `Wire-Part-States` |
| 4. Textures (#322 to #325) | `Texture-SecondTape` (the second tape and the elevation rule), then one frame a slice: `Texture-Slice1` to `Texture-Slice4` |

Rows 1 and 4 are drawn twice: each frame has a twin beside it in the dark
scheme, in a file named with `-Dark`. A frame is fixed to its scheme; none has
a switch. The wireframes in rows 2 and 3 have no colour and no twin.

Row 1 is redrawn from the page as built (#334). Each of its frames holds
the page's own markup, captured from a server with the demo set, and the
rules of `pkg/ui/static/ui.css` that the markup uses. A specimen sits in a
box classed `vp-wide`, `vp-tab` or `vp-phone`, and the stylesheet's width and
colour-scheme queries are resolved for that box: the page at 1280, 768 and
390 wide. Rows 2 to 4 are the proposals row 1 replaces; they stay until
their removal is confirmed.

## This snapshot

Canvas version `1795194153-2ac4`, published on 2026-11-20.

## Rebuilding a canvas from a snapshot

Publish this folder as the root of a new Design canvas. Eleven frames and their dark twins
(the six of row 1 and the five `Texture-` frames)
draw their textures from two
images uploaded to the original canvas, at addresses that resolve only there:

| Address in the frames | Original |
| --- | --- |
| `/_blob/9d26d515fa68a38e6638bb4a880cacf5` | `pkg/ui/static/brushed.svg` |
| `/_blob/8df67eb7edb3679ce1a61d297a1fcd76` | `pkg/ui/static/paper.svg` |

Upload both to the new canvas and replace each address with the one the
upload returns. Without that, those twenty-two frames draw with no texture.

`_blob/` holds one link per upload, named for its address and pointing at
its original. A reader that draws the frames off the canvas, from this
folder, can take each `/_blob/<id>` from `_blob/<id>`.

## Data

Every part, number and stock row in the frames is the demo set's invented
data. Decoded values are the decoder's own for the demo set's VINs.
