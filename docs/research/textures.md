# Surface textures without raster images: references for the VIN plate page

Research date: 2026-11-17. Scope: textures a page can draw with CSS gradients, `mask-image`, blend modes, `filter`, and small same-origin SVG files, to sit beside the existing brushed-metal data plate.

Companion to [vin-page.md](vin-page.md), for draft PR
[#303](https://github.com/joshuasprow/nitrovin/pull/303).

**What has been taken.** One candidate: embossed label-maker tape (candidate
1), on the names under "Also known as" and that card's caption, as black tape
in both colour schemes (`--tape` in `pkg/ui/static/ui.css`). The names keep
their case, where real tape is capitals only, because a vocabulary's entry is
case-exact. Everything else here is kept as reference and nothing more of it
is built. A theme picker is planned, and these candidates are the material
for its themes; the ranked recommendation at the end is the starting order.

## How to read this

- Every technique statement links to a page that was opened and read. Code is quoted or paraphrased from that page.
- Lines marked **Adaptation** are my own changes or combinations. No source shows them.
- Lines marked **Render check** describe what I saw when I pasted the source's code, unchanged unless stated, into a local test page and took one screenshot in WebKit. Chromium and Firefox were not checked, and nothing was measured on a phone.
- The "physical" paragraphs are general knowledge about the material, not sourced.
- Sources vary in weight. MDN, web.dev, CSS-Tricks, Smashing, Codrops, Lea Verou's gallery and Temani Afif's sites are primary or well established. CodePen pens are primary for their own code only. Nodlume, CodeFronts, ArtboardLab and css3shapes are tool or tutorial sites; I used them for the code they show and say so where a claim rests on them alone.

## Mechanics shared by every candidate

### Three ways to draw a texture

1. **Layered CSS gradients on `background`.** A small tile repeated with `background-size`. The reference collection is [Lea Verou's CSS3 Patterns Gallery](https://projects.verou.me/css3patterns/) ([source](https://github.com/LeaVerou/css3patterns/blob/master/index.html)). Its submission rules state the cost test: "Is the number of gradients and color stops worth the effect?" [Temani Afif's css-pattern.com](https://css-pattern.com/) is the modern equivalent, built on `conic-gradient` and custom properties with one size variable.
2. **A static SVG file whose content is a filtered `<rect>`, used as `background-image`.** [CSS-Tricks, "Grainy Gradients"](https://css-tricks.com/grainy-gradients/) saves an SVG containing `<feTurbulence type='fractalNoise' baseFrequency='0.65' numOctaves='3' stitchTiles='stitch'/>` applied to a `<rect>` as a separate file and references it by URL in `background`, layered under a gradient. It notes that referencing the SVG by `id` from CSS does not work for a background. [CSS-Tricks, "Creating Patterns With SVG Filters"](https://css-tricks.com/creating-patterns-with-svg-filters/) calls this "Method 3: Using a standalone SVG" and explains `stitchTiles="stitch"`, which makes the noise repeat without a seam. This is the same shape as the page's current `brushed.svg`.
3. **A live SVG filter on an element, `filter: url(#id)`.** Needed only when the effect must follow the element's own shape, such as a raised letter or a die-cut outline. [MDN, `filter`](https://developer.mozilla.org/en-US/docs/Web/CSS/filter) shows `filter: url("file.svg#filter-element-id")`, and [MDN, "Applying SVG effects to HTML content"](https://developer.mozilla.org/en-US/docs/Web/SVG/Guides/Applying_SVG_effects_to_HTML_content) states that references to SVG in external files must be same origin.

### Cost

- A static SVG used as a background is rasterised once and cached. A live filter is not. [Craft, "Noise"](https://craft.gustavofior.com/noise): "`feTurbulence` is generated per pixel and re-rendered whenever the element repaints"; stretched across a full-screen area that scrolls or animates "it can drop a page to a handful of frames per second on a phone". Its fix is to render once and tile, with a 200px tile and `stitchTiles="stitch"`. [ArtboardLab](https://artboardlab.com/guides/grainy-gradients-and-noise-texture/) says the same: `filter: url(#grain)` on a live element "re-runs the filter on every repaint and applies to the element's children, text included", while the SVG as an image is "rasterised once, cached, free on repaint".
- [Taylor Hunt, "Improving SVG Runtime Performance"](https://codepen.io/tigt/post/improving-svg-rendering-performance) ranks SVG delivery from fast to slow as `background-image`, `<img>`, inline `<svg>`, `<object>`, and calls `<filter>` "the performance crimes kingpin", with hardware acceleration inconsistent between browsers and absent on some Android hardware.
- Lighting primitives are the expensive ones. [Smashing Magazine, "The Art Of SVG Filters"](https://www.smashingmagazine.com/2015/05/why-the-svg-filter-is-awesome/) says of its `feSpecularLighting` bevel: "This filter has a serious effect on performance, so use it wisely." [Codrops](https://tympanus.net/codrops/2019/02/19/svg-filter-effects-creating-texture-with-feturbulence/) gives the same warning for animated filters over large areas.
- Long filter chains cost on every display even as images. [CSS-Tricks](https://css-tricks.com/creating-patterns-with-svg-filters/) warns they "can pretty quickly create a computationally heavy 'monster' filter chain".
- `numOctaves` above 3 or 4 adds a full pass per octave for detail that is not visible ([ArtboardLab](https://artboardlab.com/guides/grainy-gradients-and-noise-texture/); [Creative Bloq, Michael Mullany](https://www.creativebloq.com/netmag/how-go-beyond-basics-svg-filters-71412280) puts the useful ceiling at five).
- CSS `drop-shadow()` chains are also costly. [Shubham Naik](https://shub.club/writings/2026/october/adventures-in-filters) measured a visible startup delay from eight chained `drop-shadow()` calls on ten or more elements under 6× CPU throttling.
- [CodeFronts, "Grain Paint Cost and Byte Size Measured"](https://codefronts.com/design-styles/css-grain-texture/grain-paint-cost-and-byte-size-measured/) is a page that measures frame time in the visitor's own browser. It quotes no fixed figures. Its notes say cost is close to linear in the number of stacked blended layers, and that a promoted full-viewport layer at 3× device pixel ratio costs several megabytes of GPU memory.

### Light and dark themes

- Line and dot patterns: mix the ink from the theme foreground at a few percent so one declaration serves both themes. [Nodlume](https://nodlume.dev/texture-playground) writes every pattern as `color-mix(in oklab, var(--foreground) N%, transparent)` and keeps line textures between 3% and 8%, halving the value where two layers cross.
- Noise: one opacity is wrong for one of the themes. [ArtboardLab](https://artboardlab.com/guides/grainy-gradients-and-noise-texture/) says to cut grain opacity by a third to a half in dark mode, because light specks on near-black are more conspicuous. [CodeFronts, "Dark Mode Noise Without Muddy Shadows"](https://codefronts.com/design-styles/css-grain-texture/dark-mode-noise-without-muddy-shadows/) ships two weights and uses `soft-light` instead of `overlay`.
- Noise output has colour and alpha. Desaturate it with `<feColorMatrix type="saturate" values="0"/>` ([Craft](https://craft.gustavofior.com/noise), [ArtboardLab](https://artboardlab.com/guides/grainy-gradients-and-noise-texture/)).
- `mix-blend-mode` differs slightly between Blink and WebKit ([CSS-Tricks](https://css-tricks.com/grainy-gradients/)). It also blends with whatever is behind the element unless the container has `isolation: isolate` ([Craft](https://craft.gustavofior.com/noise)). When the noise and the colour are two background layers of one element, `background-blend-mode` avoids the problem ([ArtboardLab](https://artboardlab.com/guides/grainy-gradients-and-noise-texture/)).
- Text contrast must be measured on the composited surface, not the flat colour ([ArtboardLab](https://artboardlab.com/guides/grainy-gradients-and-noise-texture/), [CodeFronts](https://codefronts.com/design-styles/css-grain-texture/dark-mode-noise-without-muddy-shadows/)).
- Remove textures for print and `forced-colors` ([Nodlume](https://nodlume.dev/textures/weaves-and-fabric)).
- `mask-image` is "Baseline Newly available"; older browsers need `-webkit-mask-image` ([web.dev, "CSS masking"](https://web.dev/articles/css-masking)).

### Note on the page's CSP

Most sources embed the SVG as a `data:` URI inside CSS. I did not verify how the page's policy treats `data:` images, so treat every `data:` example below as "put this SVG in a same-origin file", which is the form [CSS-Tricks](https://css-tricks.com/grainy-gradients/) uses and the form `brushed.svg` already has. Several sources also put per-element values in inline `style` attributes; those must move to classes here.

---

## Vinyl

### 1. Embossed label-maker tape

**Physical.** Glossy coloured vinyl strip with raised white capitals, one strip per label. Found on parts bins, shelf edges, fuse boxes and tool drawers.

**References.**
- [Colorion, "Embossed Label Text Effect"](https://text-effects.colorion.co/effects/embossed-label-text-effect), pure CSS with a static variant.
- [Stack Overflow, "Creating a dymo tape text effect"](https://stackoverflow.com/questions/24788516/creating-a-dymo-tape-text-effect-without-changing-the-markup), accepted answer by John Stimac. Its ragged left edge is a PNG; only the tape and lettering are relevant here.

**Technique.** Colorion: the tape is `background: repeating-linear-gradient(0deg, transparent 0 2px, #08070f22 2px 3px), color-mix(in srgb, var(--ink-2) 55%, #21142b)`, with a 2px lighter top border, a 3px dark bottom border and `box-shadow: 0 5px 7px #08070f66`. The letters are `font: 900 27px/1 'Courier New', monospace` with `letter-spacing: .1em` and `text-shadow: -1px -1px 0 #08070faa, 1px 1px 0 #ffffff55, 0 2px 2px #08070f88`. Serrated ends are two pseudo-elements with `clip-path: polygon(...)`. The static form is the same rule with `animation: none`. Stack Overflow: `background: linear-gradient(to bottom, #31221f, #030000)`, `border-top: 1px solid #6c5c5d`, uppercase bold Courier with `letter-spacing: 3px`, four hard 1px `text-shadow` offsets in the tape colour plus `0 0 4px #fff`.

**Render check.** Colorion's static rule with a red `--ink-2` reads as label tape at a glance.

**Cost and caveats.** One gradient and a text shadow. No filter, no SVG. Colorion's markup wraps each letter in `<b style="--i:n">`; that exists only for the animation stagger and is not needed for a static label. `color-mix()` is required as written.

**Fit.** Name plates and small captions: field names, segment labels under the 17 characters, a "copied" confirmation. It is the cheapest candidate that is clearly not metal.

### 2. Cut vinyl decal or sticker

**Physical.** Printed vinyl with a slight gloss, a visible cut edge (often a white border) and a thin shadow where it lifts from the surface. Found as window decals, service reminder stickers, tyre-pressure and warning labels, parts-brand stickers on a counter.

**References.**
- [Shubham Naik, "Adventures in CSS and SVG filters and browser performance"](https://shub.club/writings/2026/october/adventures-in-filters).
- [jh3y, "SVG Filter Sticker Effects" (CodePen)](https://codepen.io/jh3y/pen/OPJyVGb).
- Gloss overlay: [sfearl1, "Carbon Fiber Patterns (CSS)" (CodePen)](https://codepen.io/sfearl1/pen/Pogmorg).

**Technique.** Naik's outline: `feMorphology operator="dilate" radius="5"` on `SourceAlpha` for the white border, a second dilate of 1 for a grey keyline, `feFlood` plus `feComposite operator="in"` to colour each, then `feMerge` under `SourceGraphic`; applied with `.sticker { filter: url(#sticker-outline); }`. He replaced eight chained `drop-shadow()` calls with this because the chain was slow. jh3y adds gloss and lift: `feMorphology` dilate radius 2 flooded `#fafafa`; `feGaussianBlur stdDeviation="3"` of `SourceAlpha` as a height map into `feSpecularLighting surfaceScale="5" specularConstant="0.5" specularExponent="120"` with `fePointLight x="50" y="50" z="300"`; `feComposite operator="arithmetic" k2="1" k3="1"` to add the highlight; `feDropShadow dy="2" stdDeviation="2" flood-opacity="0.5"`. sfearl1's gloss is a plain overlay: `linear-gradient(160deg, transparent 10%, rgba(255,255,255,0), rgba(255,255,255,0.2) 30%, transparent 30%, rgba(255,255,255,0) 30%, rgba(255,255,255,0.2) 60%, transparent 60%, transparent 100%)`.

**Render check.** jh3y's filter, reduced to outline, specular highlight and drop shadow, on a red rounded label: white cut edge, soft gloss, small lift shadow. It reads as a sticker.

**Cost and caveats.** The filter version is a live SVG filter with a lighting primitive, which is the expensive class. The filter definitions must live in an inline `<svg>` in the document or in a same-origin file ([MDN](https://developer.mozilla.org/en-US/docs/Web/SVG/Guides/Applying_SVG_effects_to_HTML_content)). **Adaptation:** for a rectangular or rounded card no filter is needed. A white `border`, a small `box-shadow` and the sfearl1 gloss gradient as a top background layer give the same reading at no filter cost.

**Fit.** Status chips and badges (for example a "check digit valid" mark), or the coloured cards if they should read as decals rather than tags.

### 3. Textured vinyl or leatherette grain

**Physical.** Pebbled, slightly glossy PVC with a fine irregular grain, sometimes with a stitched seam. Found on dashboards, door cards, seats, counter stools and binder covers.

**References.**
- [ol-ivier, "Paper Textures — Pure SVG & CSS" (CodePen)](https://codepen.io/ol-ivier/pen/raWowqp), textures 12 "Leather Grain Paper" and 25 "Flint / Emery Paper". Each is one CSS rule with an SVG data URI background.
- [maevadevs, "Simple Stitched Leather With CSS" (CodePen)](https://codepen.io/maevadevs/pen/dOyMmw).
- [Lea Verou's gallery](https://github.com/LeaVerou/css3patterns/blob/master/index.html), "Upholstery" by Nate Eagle: a buttoned, quilted pattern from ten gradients on a 100px tile.

way

**Render check.** Leather Grain in the pen's dark red is high contrast and reads as pebbled leather, too heavy for a surface that carries text. Emery reads as a fine even pebble and is the closer match for clean dashboard vinyl.

**Cost and caveats.** Static SVG image, so the lighting runs once at rasterisation. Both filters use four octaves, the upper end of what [ArtboardLab](https://artboardlab.com/guides/grainy-gradients-and-noise-texture/) says adds visible detail. Emery has no `stitchTiles`, so a tiled copy can show seams unless it is added (**Adaptation**, per the CSS-Tricks `stitchTiles` explanation). The colour is baked into the SVG; for two themes either ship two files or make the SVG greyscale and blend it over a CSS colour (**Adaptation**). Excluded: [Tanner Nielsen's engraved leather article](https://blog.tannernielsen.com/2019/01/25/creating-an-engraved-leather-effect-with-css), whose textures are baked in Blender, and [Made Good Designs' leather tool](https://madegooddesigns.com/tools/leather-texture/), which exports PNG, JPG or WebP tiles.

**Fit.** A dialog or side panel surface in the dark theme, at low contrast. The stitched edge fits a dialog border. Not for cards that sit against the plate.

### 4. Carbon-fibre vinyl wrap

**Physical.** Printed or embossed vinyl imitating 2×2 twill carbon weave. Found on aftermarket trim, mirror caps and bonnet wraps.

**References.**
- [Lea Verou's gallery](https://github.com/LeaVerou/css3patterns/blob/master/index.html): "Carbon" (design Atle Mo, code Sébastien Grosjean) and "Carbon fibre" (design Atle Mo, code Lea Verou).
- [Stack Overflow, "How to do a 'true' carbon fiber CSS background"](https://stackoverflow.com/questions/34556024/how-to-do-a-true-carbon-fiber-css-background), answer by Harry.
- [sfearl1, "Carbon Fiber Patterns (CSS)" (CodePen)](https://codepen.io/sfearl1/pen/Pogmorg).
- [Nodlume, "Carbon"](https://nodlume.dev/texture-playground/carbon) and "Carbon twill" in [Fabric & Weave](https://nodlume.dev/textures/weaves-and-fabric).

**Technique.** Verou "Carbon": six `linear-gradient` layers at 27deg and 207deg in `#151515` to `#242424` on `#131313`, `background-size: 20px 20px`. Harry: two `linear-gradient(45deg, black 25%, transparent 25%, transparent 75%, black 75%, black)` layers at `10px 10px`, offset `5px 5px`, over `linear-gradient(to bottom, rgb(8,8,8), rgb(32,32,32))` at `10px 5px`. sfearl1: four `conic-gradient` wedges, each offset by one more `--size` step, over a four-band `repeating-linear-gradient`, tile `calc(4 * var(--size))`; a simpler form is left commented in the source as `conic-gradient(at 25% 50%, transparent 75%, #333 0%, #555 100%)` repeated at offsets 0, 1, 2, 3 × `--size`. Nodlume "Carbon": two `repeating-linear-gradient` layers at ±45deg, 1px line, 6px pitch, ink at 4% of the foreground on `#0a0a0a`. Nodlume "Carbon twill": two `radial-gradient` dot lattices offset half a cell, `background-size: 12px 12px`.

**Render check.** Verou "Carbon" reads as a subtle dark weave. sfearl1's simple form at `--size: 6px` reads most clearly as twill. Nodlume's crosshatch at 4% is almost invisible on its own. Verou "Carbon fibre" does not read as weave; see candidate 8.

**Cost and caveats.** Four to six gradient layers on a small tile; cheap to paint. All versions are designed dark. Nodlume notes that on a light theme its version becomes a grey crosshatch and calls carbon twill "the loudest weave", to be kept off body text.

**Fit.** Weak. It is a second fine directional pattern next to brushed metal, so the two compete. At most a narrow dark-theme header strip.

---

## Licence plates

### 5. Retroreflective plate sheeting

**Physical.** White or yellow sheeting with a faint honeycomb or diamond cell pattern under a clear top film, which brightens toward a light source near the viewer. Found on number plates, conspicuity tape and road signs.

**References.** I found no page that draws retroreflective sheeting by name. The cell pattern has references:
- [css-pattern.com, "Honeycomb"](https://css-pattern.com/honeycomb) by Temani Afif.
- [Lea Verou's gallery](https://github.com/LeaVerou/css3patterns/blob/master/index.html), "HoneyComb" by Paul Salentiny; the same code is discussed in [Stack Overflow, "Hexagonal CSS background"](https://stackoverflow.com/questions/50277743/hexagonal-css-background).

**Technique.** Temani: three `conic-gradient` definitions with the stop list `#0000, var(--c1) .5deg 119.5deg, #0000 120deg`, each used twice at offset positions, over `var(--c2)`; `background-size: calc(2*var(--s)) calc(3.46*var(--s))`; one size variable. Salentiny: two `radial-gradient` and four `linear-gradient` layers at 150deg, 30deg and 90deg, `background-size: 40px 60px`.

**Render check.** Temani's honeycomb at `--s: 10px` in two close yellows (**Adaptation:** my colours and size) reads as plate sheeting cells.

**Cost and caveats.** Six gradient images per tile. The pattern is only plausible on white or yellow, which fixes the element's colour in both themes. The brightening toward the viewer has no reference; a soft radial highlight would be my own addition. The licence-plate pens I opened ([mjh](https://codepen.io/mjh/pen/xEwxBd), [ix4](https://codepen.io/ix4/pen/OJJxWYN), [mhdhadjar](https://codepen.io/mhdhadjar/pen/YzXOaNL)) use flat colours, borders and one inset `box-shadow`; none draws sheeting.

**Fit.** Only for a single small element that is meant to be a plate, such as a registration field shown beside the VIN. Not a general surface.

### 6. Embossed plate characters

**Physical.** Characters pressed up from sheet metal and inked on top, with a soft shoulder that catches light on one side. The same press mark, without ink, is what a stamped VIN plate has.

**References.**
- [30 seconds of code, "Engraved and embossed text"](https://www.30secondsofcode.org/css/s/engraved-embossed-text).
- [Trumphouse, "CSS Metallic Plate" (CodePen)](https://codepen.io/Trumphouse/pen/DRGarB).
- [imohkay, "Embossed Effect with Pure CSS" (CodePen)](https://codepen.io/imohkay/pen/VwVBeLv). Its page background is a PNG; only the text rule is relevant.
- [Carmen Ansio, "SVG filters on type"](https://www.carmenansio.com/articles/svg-filters-on-type/), section "Metal under a lamp".
- [Smashing Magazine, "The Art Of SVG Filters"](https://www.smashingmagazine.com/2015/05/why-the-svg-filter-is-awesome/), section "Lighting Effect".
- [CodeFronts, "Paper Stock and Letterpress Press Effect"](https://codefronts.com/design-styles/css-grain-texture/paper-stock-and-letterpress-press-effect/).

**Technique.** CSS only: 30 seconds of code clips a background to the text and sets `color: #f0f0f0; text-shadow: 1px 4px 4px #555555` for embossed, `text-shadow: 3px 5px 1px rgba(245,245,245,0.5)` with transparent colour for engraved. Trumphouse on a metal gradient: `color: #999; text-shadow: 1px 0 rgba(200,200,200,.4), -1px 0 rgba(100,100,100,.4), 0 1px rgba(255,255,255,.8), 0 -1px rgba(0,0,0,.4), 0 0 .1em rgba(255,255,255,.5)`. imohkay: `text-shadow: 1px 1px 0 rgba(0,0,0,0.07), -1px -1px 0 rgba(255,255,255,0.7)` with a nearly transparent `color`. Filter: Ansio blurs `SourceAlpha` (`stdDeviation="3"`) as a height map, lights it with `feSpecularLighting surfaceScale="9" specularConstant="1.4" specularExponent="32"` and `fePointLight x="220" y="-50" z="160"`, clips with `feComposite operator="in"` and screens it over the source. Smashing's version uses `stdDeviation="5"`, `surfaceScale="6"`, `specularExponent="30"`, `fePointLight x="40" y="-30" z="200"`.

**Render check.** Trumphouse's shadows on a grey metal gradient give quiet raised characters in the metal's own colour, which is the stamped-plate look. Ansio's filter on dark blue text over yellow gives glossy rounded characters, which is the inked number-plate look.

**Cost and caveats.** The `text-shadow` versions cost nothing. The filter version is a live lighting filter; Smashing warns about its performance, and the same article reports that Safari at the time could fail to render filtered HTML at all. CodeFronts' notes say a deboss below about 13px makes type harder to read, and that the shadow direction must agree with the element's own drop shadow. My own note, not from a source: the `fePointLight` position is a fixed coordinate, so the highlight will fall differently on elements of different sizes that share one filter.

**Fit.** The 17 plate characters and the name plates, using the `text-shadow` form. The filter form is not justified for 17 characters that already read well.

---

## Metal

### 7. Diamond or tread plate

**Physical.** Rolled steel or aluminium with raised lozenges in alternating directions. Found on truck beds, running boards, toolboxes, trailer decks and workshop floors.

**References.**
- [matthewvisyak, "Diamond Plate CSS3 Pattern" (CodePen)](https://codepen.io/matthewvisyak/pen/NrXXmg). The description names it as chequer plate, tread plate or Durbar floor plate.
- [css-pattern.com, "Equal sign (rotated rectangles)"](https://css-pattern.com/equal-sign/) for a flat version.

**Technique.** matthewvisyak: twelve `linear-gradient` layers on `rgba(150,150,150,1)`, `background-size: 100px 100px`. Each lozenge is a pair of near-parallel hard-stop gradients (for example 132deg white and 128deg `rgba(200,200,200,0.7)`, stops at 26px/27px) for the lit and shaded halves, trimmed by "mask" gradients in the base colour, with twelve hand-tuned `background-position` offsets. Temani's equal sign: four `linear-gradient` layers at 45deg and 135deg from one stop list `#0000 8%, var(--c1) 0 17%, #0000 0 58%`, `background-size: var(--s) var(--s)`.

**Render check.** The pen reads as tread plate immediately, with bright white highlights. Temani's pattern in two greys reads as alternating perpendicular bars, a flat diagram of tread plate without relief.

**Cost and caveats.** Twelve layers, but on one small tile, so painting is cheap. Every stop and offset is in pixels, so changing the scale means recalculating all of them. The highlights are near white on mid grey, which is high contrast under text and wrong for a dark theme without recolouring every layer.

**Fit.** Poor for anything behind text. At most a narrow footer band or a separator strip.

### 8. Perforated or expanded metal mesh

**Physical.** Sheet with staggered round holes (speaker grilles, heat shields, pegboard-style display panels) or slit-and-stretched diamond mesh (grilles, shelving, security cages at a parts counter).

**References.** No page I opened names perforated or expanded metal. Patterns with the right geometry:
- [Lea Verou's gallery](https://github.com/LeaVerou/css3patterns/blob/master/index.html), "Carbon fibre".
- [Nodlume, "Dot grid"](https://nodlume.dev/texture-playground/dot-grid) and ["Diamond lattice"](https://nodlume.dev/texture-playground/diamond-lattice).
- [jdsteinbach, "Simple Dotted Background with Radial Gradient" (CodePen)](https://codepen.io/jdsteinbach/pen/aZerJB).
- Real holes through to the layer behind: [web.dev, "CSS masking"](https://web.dev/articles/css-masking) (gradient as mask) and [CSS-Tricks, "Tricks to Cut Corners Using CSS Mask and Clip-Path"](https://css-tricks.com/cut-corners-using-css-mask-and-clip-path-properties) (`radial-gradient(circle 30px at top left, #0000 98%, red)` as a mask layer).

**Technique.** Verou: `radial-gradient(black 15%, transparent 16%)` at `0 0` and `8px 8px`, plus `radial-gradient(rgba(255,255,255,.1) 15%, transparent 20%)` at `0 1px` and `8px 9px`, on `#282828`, `background-size: 16px 16px`. The second pair is the same dots shifted down 1px in faint white, which gives each hole a lit lower lip. Nodlume dot grid: `radial-gradient(color-mix(in oklab, var(--foreground) 16%, transparent) 1px, transparent 1px)` at `20px 20px`. Diamond lattice: two `repeating-linear-gradient` layers at ±45deg, 1px line, 16px pitch, 5% ink.

**Render check.** Verou's "Carbon fibre" reads as dark perforated sheet, not as weave.

**Cost and caveats.** Two to four gradient layers; cheap. Reading the diamond lattice as expanded metal is my interpretation (**Adaptation**); real expanded mesh has thick strands and elongated openings, which hairlines do not show. Using the dots as a `mask` to cut real holes is also an **Adaptation**.

**Fit.** A dark-theme page background at low contrast, or the empty area of a future parts list. It is regular and non-directional, so it competes less with brushed metal than weave or tread plate do.

### 9. Engine-turned (jewelled) metal

**Physical.** Rows of overlapping circular swirl marks ground into sheet metal. Found on vintage dashboards, firewalls and instrument bezels.

**References.** None found. No page I opened draws engine turning. The parts exist separately:
- One circular-brushed disc: [bennettfeely, "Brushed Metal with CSS Gradients" (CodePen)](https://codepen.io/bennettfeely/pen/dyGRrbp); [simurai, "Brushed Metal"](https://simurai.com/lab/2011/08/21/brushed-metal), which in 2011 faked the conical sweep with elliptical gradients and used three repeating gradients of different lengths to look random; [nikolab, "repeating-conic-gradient Metallic Background" (CodePen)](https://codepen.io/nikolab/pen/oNBbEWJ).
- The overlapping lattice: [css-pattern.com, "Overlapping circles"](https://css-pattern.com/overlapping-circles/), four `radial-gradient` layers with the fourth offset by half a tile.

**Technique in the sources.** Bennett: `repeating-radial-gradient(rgba(#8f8f8f,0) 0, rgba(#8f8f8f,0.7) 2px)` over `radial-gradient(#4a4a4a 0, rgba(#4a4a4a,0) 30%, ...)` over a twelve-stop `conic-gradient` of alternating light and dark greys. Temani's circles are filled with concentric rings, not sweeps.

**Render check.** Bennett's disc is a convincing spun-metal button. Temani's overlapping circles read as ring ornament, not jewelling. **Adaptation**, tested: a `conic-gradient` tile at 24px, masked to a circle with a `radial-gradient` mask, in two stacked layers with the second offset by half a tile, reads as engine turning. It needs two extra boxes, relies on `mask`, and is the busiest surface in this document. A single unmasked conic tile reads as faceted squares.

**Cost and caveats.** Not established by any source; treat the adaptation as an experiment.

**Fit.** None recommended. Bennett's single disc is useful on its own for round buttons (candidate 11).

### 10. Anodised aluminium

**Physical.** Satin aluminium with a dyed oxide layer: even colour, fine matte sheen, no visible grain at arm's length. Found on fittings, fuel rail covers, tool handles and aftermarket caps.

**References.** None found by name. The nearest are plain metallic gradients: [ibelick, "Creating a metallic effect with CSS"](https://ibelick.com/blog/creating-metallic-effect-with-css), a single `linear-gradient(45deg, #999 5%, #fff 10%, #ccc 30%, #ddd 50%, #ccc 70%, #fff 80%, #999 95%)` on a button.

**Technique.** **Adaptation:** anodising is a colour over a satin surface, so it is the existing brushed metal with a hue. Either tint the base colour under the existing `brushed.svg` mask, or lay a colour over the metal with `background-blend-mode` (the blend caveats above apply).

**Cost and caveats.** Nothing new to paint. It is a colour variant, not a new texture.

**Fit.** Buttons, as a way to give primary and secondary actions different colours while staying metal.

### 11. Circular-brushed (spun) metal

**Physical.** Concentric brush marks from finishing on a lathe. Found on knobs, caps, wheel centres and round push buttons.

**References.** [bennettfeely's pen](https://codepen.io/bennettfeely/pen/dyGRrbp) (technique in candidate 9); [simurai](https://simurai.com/lab/2011/08/21/brushed-metal); [CodeFronts, "Rotary Knob Toggle with Brushed-Metal Conic Gradient"](https://codefronts.com/components/tailwind-css-toggles-switches/tailwind-rotary-knob-toggle-brushed-metal-conic-gradient/), which alternates high, low and mid metal stops every 8% around a `conic-gradient` under a `radial-gradient(circle at 32% 26%, ...)` highlight.

**Cost and caveats.** Three gradient layers on one small element. The conic stops must be recoloured per theme.

**Fit.** Round icon buttons only.

---

## Coatings and rubber

### 12. Powder coat or wrinkle-finish paint

**Physical.** Powder coat is an even matte or satin skin with a faint orange-peel. Wrinkle finish is a tight, fine, irregular ripple, usually black or red. Found on valve covers, brackets, toolboxes, shelving and counter frames.

**References.** None found by name. The method is the lit-noise technique:
- [Michael Mullany, "How to go beyond the basics with SVG filters" (Creative Bloq)](https://www.creativebloq.com/netmag/how-go-beyond-basics-svg-filters-71412280).
- [Sara Soueidan, "Creating Texture with feTurbulence" (Codrops)](https://tympanus.net/codrops/2019/02/19/svg-filter-effects-creating-texture-with-feturbulence/).
- [SELFHTML-Wiki, "Muster in SVG/Texturen"](https://wiki.selfhtml.org/wiki/Muster_in_SVG/Texturen).
- [ol-ivier's pen](https://codepen.io/ol-ivier/pen/raWowqp), textures 25 "Flint / Emery Paper" and 35 "Slate / Chalkboard Paper".

**Technique.** Mullany: `feTurbulence type="fractalNoise" baseFrequency="0.04" numOctaves="5"` into `feDiffuseLighting lighting-color="white" surfaceScale="2"` with `feDistantLight azimuth="45" elevation="35"`. The lighting reads the noise's alpha as height. He states that "a surfaceScale of 20 will create a stucco texture and a surfaceScale of 50 will create crumpled plastic". Codrops walks the same filter with `elevation="60"` and notes that lowering it to 40 raises contrast. SELFHTML shows a finer variant at `baseFrequency="0.08" numOctaves="10"`. ol-ivier's Slate: `baseFrequency='0.4' numOctaves='3'`, `feDiffuseLighting lighting-color='#777' surfaceScale='0.5'`, `feDistantLight azimuth='90' elevation='70'`, drawn at `opacity='0.8'` over `#232529`. Emery is given in candidate 3.

**Render check.** Emery's values read as a fine texture coat. Slate's read as an almost flat dark satin, close to plain powder coat. Mullany's filter with only `surfaceScale` raised to 20 renders as hard black-and-white relief at that low frequency; it needs a higher frequency and lower contrast before it resembles paint (my observation, not from the source).

**Cost and caveats.** One lit-noise SVG as a static tiled background. Add `stitchTiles="stitch"` and pin the filter region to the tile for seamless repeat ([CSS-Tricks](https://css-tricks.com/creating-patterns-with-svg-filters/), [Craft](https://craft.gustavofior.com/noise)). Noise rendering differs slightly between engines ([ArtboardLab](https://artboardlab.com/guides/grainy-gradients-and-noise-texture/)). Two themes need two strengths.

**Fit.** Page background or dialog surface, especially in the dark theme. It is matte and has no direction, which is the opposite of brushed metal on both counts.

### 13. Rubber or tyre tread

**Physical.** Matte black rubber: ribbed or coin-dot floor mats, pedal pads, and the block and groove pattern of a tyre.

**References.** None found. [sudheer-ranga, "CSS3 Rotating Tyre Animation" (CodePen)](https://codepen.io/sudheer-ranga/pen/MyXoJJ) draws a tyre as a thick `border` ring with no tread. Chevron and zig-zag patterns that could stand in for tread blocks are listed on [css-pattern.com](https://css-pattern.com/) ("Chevron stripes", "Arrows", "Zig-Zag"); I read the index but not those pattern pages. Ribbed rubber is a one-direction stripe: [CSS-Tricks, "Stripes in CSS"](https://css-tricks.com/stripes-css/) and Nodlume's pinstripe in [Fabric & Weave](https://nodlume.dev/textures/weaves-and-fabric) (`repeating-linear-gradient(90deg, <ink> 0 1px, transparent 1px 6px)`, with the advice to keep the interval at 6px or more).

**Cost and caveats.** Any tread pattern here would be an **Adaptation** with no source showing it read as rubber.

**Fit.** None recommended.

---

## Paper stock at a parts counter

### 14. Manila parts tag with a reinforced eyelet

**Physical.** Buff card tag with clipped corners, a punched hole ringed by a card or metal reinforcement, tied on with string or wire. Tags also come in solid colours for status (repair, hold, scrap). Found tied to cores, returns and special orders.

**References.**
- [css3shapes, "How to Make a Price Tag with CSS"](https://css3shapes.com/shapes/how-to-make-a-price-tag-with-css/).
- [lrenhrda, "Ticket Shape" (CodePen)](https://codepen.io/lrenhrda/pen/powRPN), adapted from Lea Verou's inset border-radius technique according to its description.
- [CSS-Tricks, "Tricks to Cut Corners Using CSS Mask and Clip-Path"](https://css-tricks.com/cut-corners-using-css-mask-and-clip-path-properties).
- Shadow on a clipped shape: [Adnan Gobeljić, "How this desk was made"](https://www.adnangobeljic.com/guide).
- Stock: [ol-ivier's pen](https://codepen.io/ol-ivier/pen/raWowqp), texture 39 "Fawn Archival Board".

**Technique.** css3shapes: `background: radial-gradient(circle at 24px 24px, transparent 8px, #cd5454 8px); clip-path: polygon(16px 0, 100% 0, 100% 100%, 0 100%, 0 16px);`. The hole is truly transparent because it is part of the background image. lrenhrda: two `radial-gradient(circle at 0 50%, rgba($bg,0) $hole, $bg ($hole + .1em))` layers for side notches, with `filter: drop-shadow(0 2px 5px rgba(black,.5))` so the shadow follows the holes. Gobeljić: put the `drop-shadow` on a wrapper, "because `filter` is applied after `clip-path`". Stock: `feTurbulence type='fractalNoise' baseFrequency='0.15' numOctaves='3'`, `feColorMatrix` giving a brown at alpha 0.08, `feBlend mode='multiply'` on `#e0b589`.

**Render check.** The css3shapes rule draws a clean tag with a real hole. Fawn Archival Board renders as flat manila with a barely visible tooth.

**Cost and caveats.** One gradient and a `clip-path`. `box-shadow` is clipped away by `clip-path`, hence the wrapper `drop-shadow`, which is one filter per card and static. **Adaptation:** the reinforcement ring is an extra pair of stops in the same `radial-gradient` (transparent hole, then a ring in a darker tint, then the card colour). No source shows the ring.

**Fit.** The coloured cards that hang off the plate. The hole gives them something to hang from, and matte card against brushed metal is a contrast of both sheen and direction.

### 15. Kraft paper

**Physical.** Brown unbleached paper with visible fibre. Found as parts bags, wrapping, box liners and gasket paper.

**References.** [ol-ivier's pen](https://codepen.io/ol-ivier/pen/raWowqp), textures 2 "Raw Kraft Paper" and 7 "Brown Cardboard"; [Adnan Gobeljić's guide](https://www.adnangobeljic.com/guide), which lays `feTurbulence type='fractalNoise' baseFrequency='0.8'` with a low-alpha `feColorMatrix` over a kraft `#D9C7A7`.

**Technique.** Raw Kraft: `baseFrequency='0.02' numOctaves='4'`, a brown `feColorMatrix`, multiply, drawn at `opacity='0.4'` over `#bc9c74`. Cardboard: `baseFrequency='0.25' numOctaves='3'` with a threshold alpha row `1 1 1 0 -1.6` that leaves scattered flecks.

**Render check.** Raw Kraft renders as large soft blotches and looks stained. Cardboard renders as flecked board. Neither looks freshly made. The clean version is a flat kraft colour with fine grain at a few percent, which is Gobeljić's approach.

**Cost and caveats.** Static SVG tile. The brown is a fixed colour that needs a second value for the dark theme.

**Fit.** Weak. Once it is clean enough for this page it is a colour, not a texture.

### 16. Carbonless invoice and continuous-form paper

**Physical.** Multi-part invoice sets in white, canary and pink with ruled boxes; older counters used continuous green-bar paper with sprocket holes down both edges.

**References.** None found for carbonless sets. For the continuous form and ruling:
- [BigEd, "green bar paper in css" (gist)](https://gist.github.com/BigEd/56f6c0001c8670d1647d5448e91346d6).
- [Lea Verou's gallery](https://github.com/LeaVerou/css3patterns/blob/master/index.html), "Lined paper" by Sarah Backhouse.

**Technique.** BigEd: bands from `linear-gradient(to bottom, white 0px, white 59px, darkseagreen 60px, palegreen 61px, palegreen 119px, darkseagreen 120px)` at `background-size: 100% 120px`; sprocket holes from `radial-gradient` rings at `40px 40px` with `background-repeat: repeat-y` down each margin; dashed side borders for the tear-off perforation. Backhouse: `linear-gradient(90deg, transparent 79px, #abced4 79px, #abced4 81px, transparent 81px), linear-gradient(#eee .1em, transparent .1em)` at `background-size: 100% 1.2em`, which ties the rule spacing to the line height.

**Render check.** The band gradient renders as clean alternating white and green bars.

**Cost and caveats.** One or two gradients. The bands must be sized to the row height or text straddles them. The pastel copy colours of a carbonless set are flat fills; there is no texture to reference.

**Fit.** A future parts list, as zebra rows sized in `em` to the row. Not elsewhere.

### 17. Thermal label or receipt

**Physical.** Bright white coated paper, no visible fibre, crisp black print. Labels are die-cut with rounded corners; receipts end in a serrated tear. Found as bin labels, shipping labels and counter receipts.

**References.**
- [Stack Overflow, "Create a responsive receipt cutoff (zig-zag border)"](https://stackoverflow.com/questions/73561793/create-a-responsive-receipt-cutoff-zig-zag-border), answer by Temani Afif.
- [CodeFronts, "CSS Receipt Paper Total Count-Up Button"](https://codefronts.com/components/css-buttons/css-receipt-paper-total-count-up-button/).
- Stock: [ol-ivier's pen](https://codepen.io/ol-ivier/pen/raWowqp), texture 27 "Smooth Matte Bristol".

**Technique.** Temani: one mask, `conic-gradient(from -45deg at bottom, #0000, #000 1deg 89deg, #0000 90deg) 50%/30px 100%`, on the element; for a shadow, move the background and mask to a pseudo-element and put `filter: drop-shadow()` on the parent. CodeFronts: a two-layer mask, `conic-gradient(from 135deg at 50% 0, #000 90deg, transparent 0) 0 100%/.76rem .4rem repeat-x` for the teeth and `linear-gradient(#000,#000) 0 0/100% calc(100% - .38rem) no-repeat` for the body. Bristol's noise is drawn at alpha 0.015.

**Render check.** Temani's mask gives a clean serrated edge. Bristol renders as plain white.

**Cost and caveats.** One mask; needs the `-webkit-` prefix for older Safari ([web.dev](https://web.dev/articles/css-masking)). The stock itself has no texture worth drawing; the reading comes from the edge, the white and monospaced black type.

**Fit.** A future parts list or a printable summary. The serrated edge is the only part worth taking.

### 18. Blueprint or graph paper

**Physical.** White rules on blue, or fine coloured rules on white, with heavier lines every fifth or tenth division. Found as workshop drawings, wiring diagrams and parts-catalogue exploded views.

**References.**
- [Lea Verou's gallery](https://github.com/LeaVerou/css3patterns/blob/master/index.html), "Blueprint grid".
- [css-pattern.com, "Graph paper"](https://css-pattern.com/graph-paper/).
- [Nodlume, "Graph paper"](https://nodlume.dev/texture-playground/graph-paper) and ["Blueprint"](https://nodlume.dev/texture-playground/blueprint).
- [ol-ivier's pen](https://codepen.io/ol-ivier/pen/raWowqp), texture 38 "Millimeter Grid Paper", built from SVG `<pattern>` elements.
- [jh3y's sticker pen](https://codepen.io/jh3y/pen/OPJyVGb), whose page background is a grid that fades out under a mask.

**Technique.** Verou: on `#269`, `linear-gradient(white 2px, transparent 2px)` and its 90deg twin at `100px 100px`, plus `linear-gradient(rgba(255,255,255,.3) 1px, transparent 1px)` and its twin at `20px 20px`, positions `-2px -2px` and `-1px -1px`. Temani: `conic-gradient(from 90deg at 2px 2px, #0000 90deg, var(--c1) 0)` and the same at `1px 1px`, `background-size: var(--s) var(--s), calc(var(--s)/5) calc(var(--s)/5)`; two layers instead of four. Nodlume graph paper: two `repeating-linear-gradient` layers, 1px line, 24px pitch, ink at 7% of the foreground; blueprint: the same with `rgb(255 255 255 / 12%)` on `#1e3a8a`, kept blue in both themes. ol-ivier: a 10-unit `<pattern>` nested in a 50-unit `<pattern>`, `background-size: 50px 50px`. jh3y: grid lines from `color-mix(in hsl, canvasText, transparent 70%)` with `mask: linear-gradient(-20deg, transparent 50%, white)` so the grid fades toward one corner.

**Render check.** Verou's blueprint grid renders as expected: strong major lines, faint minor lines.

**Cost and caveats.** Two to four gradients; the cheapest surface here. Nodlume advises under 8% ink where text sits on the grid and warns that unequal pitch or alpha in the two directions "reads as a rendering bug". A saturated blueprint blue fights coloured cards; the theme-ink graph version does not.

**Fit.** Page background behind the diagram, as faint theme-ink rules, optionally faded with a mask so it is present near the plate and absent under body text.

---

## Others that fit

### 19. Hazard stripe tape

**Physical.** Yellow and black diagonal tape on dock edges, lifts, battery warnings and floor markings.

**References.** [hzlo, "Hazard tape effect" (CodePen)](https://codepen.io/hzlo/pen/VweYrLX): `repeating-linear-gradient(-55deg, #000, #000 20px, #ffb101 20px, #ffb101 40px)` as a background and, with a trailing `10`, as a `border-image`. [CSS-Tricks, "Stripes in CSS"](https://css-tricks.com/stripes-css/) covers the general form and notes the angle need not be 45 degrees.

**Cost and caveats.** One gradient. Very loud.

**Fit.** A thin edge on an error state (invalid check digit), nothing larger.

### 20. Pegboard or dot grid

**Physical.** Perforated hardboard wall panel with a square grid of holes, the standard parts-counter display wall.

**References.** [Nodlume, "Dot grid"](https://nodlume.dev/texture-playground/dot-grid): `radial-gradient(color-mix(in oklab, var(--foreground) 16%, transparent) 1px, transparent 1px)`, `background-size: 20px 20px`. [HTML-CSS Visual Playground, "CSS Textures"](https://drjulsgilliam.github.io/HTML-CSS-Visual-Playground/css-textures.html): `radial-gradient(#5e9ea0 2px, transparent 2px)` at `20px 20px`.

**Cost and caveats.** One gradient. Reading it as pegboard depends on a larger pitch and dot than the sources use (**Adaptation**).

**Fit.** An alternative page background to the graph rules. Use one or the other.

### 21. Stitched edge

Covered under candidate 3: `border: 2px dashed` inside a solid `box-shadow` ring ([maevadevs](https://codepen.io/maevadevs/pen/dOyMmw)). No texture cost. Fits a dialog frame if the dialog takes the vinyl or powder-coat surface.

---

## Ranked recommendation

Brushed metal is glossy, fine and strongly directional. The surfaces that sit well beside it are matte and either flat or non-directional, and each should be used on one kind of element only.

1. **Manila tag stock with a punched eyelet, on the coloured cards.** A flat card colour, a `radial-gradient` hole and clipped corners ([css3shapes](https://css3shapes.com/shapes/how-to-make-a-price-tag-with-css/)), with at most a faint static grain ([ol-ivier, Fawn Archival Board](https://codepen.io/ol-ivier/pen/raWowqp)). Reason: it is the real object that hangs off parts, the hole explains the "hanging" layout, coloured tag stock is authentic, and the cost is one gradient plus one static `drop-shadow` wrapper per card.
2. **Embossed label-maker tape, on name plates and small captions.** One gradient and a `text-shadow` ([Colorion](https://text-effects.colorion.co/effects/embossed-label-text-effect)). Reason: it is vinyl, it is what labels a parts bin, and it gives small labels a second material so that not every element is metal. Use one tape colour, or one per theme.
3. **Faint graph rules, on the page background around the diagram.** Two gradients in theme ink at 7% or less ([Nodlume](https://nodlume.dev/texture-playground/graph-paper), or [Temani's two-layer version](https://css-pattern.com/graph-paper/)), optionally faded with a mask ([jh3y](https://codepen.io/jh3y/pen/OPJyVGb)). Reason: it frames the plate as a labelled drawing, follows both themes from one declaration, and is the cheapest thing here to paint.
4. **Fine texture powder coat, on dialogs, and on the dark-theme page background in place of the grid if a grid looks too light there.** One static lit-noise SVG tile with `stitchTiles`, starting from the Emery or Slate values ([ol-ivier](https://codepen.io/ol-ivier/pen/raWowqp); method from [Mullany](https://www.creativebloq.com/netmag/how-go-beyond-basics-svg-filters-71412280) and [Codrops](https://tympanus.net/codrops/2019/02/19/svg-filter-effects-creating-texture-with-feturbulence/)), with a separate strength per theme. Reason: matte and directionless, so the metal plate reads as the only shiny object. It must be a static file, never a live filter.

Alongside these, with no new texture: the 17 characters and name-plate text take the `text-shadow` stamp ([Trumphouse](https://codepen.io/Trumphouse/pen/DRGarB)); buttons stay metal, with [Bennett's conic disc](https://codepen.io/bennettfeely/pen/dyGRrbp) for round ones and a tint for an anodised variant. A future parts list can take green-bar rows ([BigEd](https://gist.github.com/BigEd/56f6c0001c8670d1647d5448e91346d6)) and a serrated receipt edge ([Temani Afif](https://stackoverflow.com/questions/73561793/create-a-responsive-receipt-cutoff-zig-zag-border)).

## Candidates I would not use

- **Diamond plate.** Twelve pixel-fixed layers with white highlights. It reads as a floor, is loud under text, and is a second metal competing with the plate.
- **Carbon-fibre wrap.** A second fine directional pattern beside brushed metal, dark only, and it signals aftermarket styling, not a factory data plate.
- **Engine-turned metal.** No source draws it. My tested version needs masks and extra boxes and is the busiest surface considered.
- **Retroreflective honeycomb as a general surface.** Six gradient images, only plausible in plate white or yellow. Acceptable on one small plate-shaped field, not on cards or backgrounds.
- **Leather-grain vinyl at the source's contrast.** Too heavy under text. The fine Emery grain covers the same ground and is what recommendation 4 uses.
- **Kraft paper.** The referenced versions look stained. Cleaned up, it is only a brown fill.
- **Rubber or tyre tread.** No reference; anything drawn would be a guess.
- **Live SVG lighting filters on text or cards** (the specular emboss and the filter-based sticker). They are the most expensive class of effect on phones, and `text-shadow`, `border` and a gloss gradient reach the same reading.
- **Hazard stripes beyond a thin error edge.** Too loud for a page that should look freshly made and calm.
- **Blueprint blue as a full background.** It fixes a saturated colour in both themes and fights the coloured cards. The theme-ink graph rules do the same job quietly.

## No real reference found

- Engine-turned (jewelled) metal, as a pattern.
- Retroreflective sheeting by name; only the honeycomb cell pattern has references.
- Powder coat or wrinkle finish by name; only the general lit-noise method and paper textures that happen to resemble it.
- Anodised aluminium by name; only plain metallic gradients.
- Rubber and tyre tread.
- Carbonless (NCR) invoice sets; only green-bar continuous paper and lined paper.
- Expanded metal mesh; only hairline diamond lattices and dot patterns.
- A reinforced eyelet ring on a tag; only a plain punched hole.
- A licence plate drawn with embossing or sheeting; the three plate pens opened are flat.

Pages that could not be read and are not cited: `text-fx.app` sticker and brushed-metal effects (HTTP 404), the CodePen pens "Metal and Fire SVG FX" and CSS-Tricks' "Graph Paper Pattern" (HTTP 403), the Tearline receipt write-up (redirects to an unrelated site), and the "UK Car Reg Number Plate" pen (content did not load).
