# Swatches: materials for the page's look

Research dates: 2026-11-20 to 2026-11-21. Scope: the materials the page is
built from, those drawn for it and set aside, and materials found for its
look in the places its users work. Companion to [textures.md](../textures.md),
which covers how a page draws a texture; these files cover what to draw.

One file per set, a card per material:

|File|Set|
|---|---|
|[page.json](page.json)|In use: the page's materials, then the marks and fittings made of them, as `pkg/ui/static/ui.css` defined them on 2026-11-24, after #395. The stylesheet, not this file, is the record of what is in use.|
|[tried.json](tried.json)|Tried: drawn for the page and set aside when another was chosen. `from` names where each drawing is kept.|
|[thermal.json](thermal.json)|Found: direct-thermal paper and its print, as a counter's receipts and a yard's part tags come off the printer.|
|[yard.json](yard.json)|Found: what parts and vehicles are tagged, labelled and marked with in a yard, a warehouse and a shop.|
|[vehicle.json](vehicle.json)|Found: surfaces of the vehicle itself, its labels and instruments, and of the garage around it.|

## A card's fields

- `name`, `family`: the material, and the group it is sorted under.
- `where_seen`: where it is seen.
- `texture`: what its surface looks like.
- `base`, `ink`, `accent`: its colours. In a found file every colour is an
  estimate, since no source publishes one.
- `css`: a recipe that draws it, a `background` value. No recipe in a
  found file is settled.
- `from` (tried only): where the drawing is kept, by branch or commit.
- `in` (in use only): the stylesheet's custom properties that hold it.
- `sources`: the pages it was found on.

The three found files carry four more:

- `checks`: a verdict for each link kept: `supports`, `partial` (the page
  is about the material and states none of the card's facts) or `blocked`
  (the page refused), with how it was read (`fetched`, `search summary` or
  `not read`) and the evidence quoted from it.
- `dropped`: the dead and unrelated links taken out.
- `unsupported`: what the card says that no source read does.
- `corrected`: marks a card whose wording was changed to what a source
  read says.

## Method

The found files are a web search's findings from 2026-11-20. Every source
was opened and checked against its card on 2026-11-21. Where a page read
says otherwise than a card did, the card now says what the page says.
