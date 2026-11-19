# The design canvas, as last read

A snapshot of the Claude Design canvas where unbuilt UI is drawn. The canvas
is the source of truth and is private to the maintainer; this folder is its
record. [AGENTS.md](../../../AGENTS.md) ("Design Canvas") says how the two are
kept together.

## What is here

`project/` holds the canvas's own files, exactly as read down:

- `canvas.json`: the index. Each frame's place and size, the pages, the notes.
- One `.dc.html` per frame.

| Page | Frames |
| --- | --- |
| Page 1, what `DESIGN.md` keeps | `Main`, `Materials`, `Diagram`, `Controls` |
| Catalog and fitment | none yet; the texture pass for the fitment slices goes here |
| Wireframes | `Desk1Stack`, `Tablet1Stack`, `Detail1Stack` (the page at 1280, 768 and 390 wide), `First1Stack`, `Empty1Stack`, `Types1Dialog` |

The wireframes are the layout behind issues #320 to #325.

## This snapshot

Canvas version `1795112649-cf25`, read on 2026-11-19.

## Rebuilding a canvas from a snapshot

Publish this folder as the root of a new Design canvas. Four frames
(`Main`, `Materials`, `Diagram`, `Controls`) draw their textures from two
images uploaded to the original canvas, at addresses that resolve only there:

| Address in the frames | Original |
| --- | --- |
| `/_blob/9d26d515fa68a38e6638bb4a880cacf5` | `pkg/ui/static/brushed.svg` |
| `/_blob/8df67eb7edb3679ce1a61d297a1fcd76` | `pkg/ui/static/paper.svg` |

Upload both to the new canvas and replace each address with the one the
upload returns. Without that, those four frames draw with no texture.

## Data

Every part, number and stock row in the frames is the demo set's invented
data. Decoded values are the decoder's own for the demo set's VINs.
